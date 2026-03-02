package glyph

import (
	"bufio"
	"io"
	"sync"
)

type LogC struct {
	reader     io.Reader
	maxLines   int
	autoScroll bool
	onUpdate   func() // called when new lines arrive (for RequestRender)

	// layout
	grow   float32
	margin [4]int16

	// key bindings
	declaredBindings []binding

	// internal state
	layer        *Layer
	lines        []string
	mu           sync.Mutex
	started      sync.Once
	following    bool // true = auto-scroll active, false = user scrolled away
	newLineCount int  // lines arrived while not following (for "X new lines" indicator)
}

// Log creates a scrollable log viewer that reads lines from an io.Reader.
// The reader is consumed in a background goroutine that exits on EOF/error.
// Lines are buffered with a configurable max (ring buffer); scrolling follows
// new content automatically until the user scrolls away.
func Log(r io.Reader) *LogC {
	return &LogC{
		reader:     r,
		maxLines:   10000, // large default buffer
		autoScroll: true,
		layer:      NewLayer(),
		following:  true, // start following new content
	}
}

// MaxLines sets the maximum number of lines to keep in the buffer.
// Oldest lines are dropped when the limit is exceeded. Default is 1000.
func (lv *LogC) MaxLines(n int) *LogC {
	lv.maxLines = n
	return lv
}

// AutoScroll controls whether the view automatically scrolls to show new lines.
// Default is true. When the user scrolls up, auto-scroll pauses until they
// return to the bottom.
func (lv *LogC) AutoScroll(enabled bool) *LogC {
	lv.autoScroll = enabled
	return lv
}

// Grow sets the flex grow factor.
func (lv *LogC) Grow(g float32) *LogC {
	lv.grow = g
	return lv
}

// Margin sets equal margin on all sides.
func (lv *LogC) Margin(all int16) *LogC {
	lv.margin = [4]int16{all, all, all, all}
	return lv
}

// MarginVH sets vertical and horizontal margins.
func (lv *LogC) MarginVH(v, h int16) *LogC {
	lv.margin = [4]int16{v, h, v, h}
	return lv
}

// MarginTRBL sets top, right, bottom, left margins individually.
func (lv *LogC) MarginTRBL(t, r, b, l int16) *LogC {
	lv.margin = [4]int16{t, r, b, l}
	return lv
}

// Layer returns the underlying layer for manual scroll control.
// Use this to bind key handlers for scrolling (j/k, Page Up/Down, etc.).
func (lv *LogC) Layer() *Layer {
	return lv.layer
}

// Ref calls f with this LogC and returns it for chaining.
func (lv *LogC) Ref(f func(*LogC)) *LogC {
	f(lv)
	return lv
}

// NewLines returns the number of new lines that have arrived while not following.
// Use this to display an indicator like "42 new lines ↓".
func (lv *LogC) NewLines() int {
	lv.mu.Lock()
	defer lv.mu.Unlock()
	return lv.newLineCount
}

// resume syncs the display to current buffer and resets new line count.
func (lv *LogC) resume() {
	lv.mu.Lock()
	defer lv.mu.Unlock()
	lv.following = true
	lv.newLineCount = 0
	lv.syncToLayer()
	lv.layer.ScrollToEnd()
}

// OnUpdate sets a callback to be called when new lines arrive.
// Use this with app.RequestRender to trigger redraws:
//
//	Log(reader).OnUpdate(app.RequestRender)
func (lv *LogC) OnUpdate(f func()) *LogC {
	lv.onUpdate = f
	return lv
}

// BindNav registers key bindings for scrolling down/up by one line.
func (lv *LogC) BindNav(down, up string) *LogC {
	lv.declaredBindings = append(lv.declaredBindings,
		binding{down, func() { lv.layer.ScrollDown(1) }},
		binding{up, func() { lv.following = false; lv.layer.ScrollUp(1) }},
	)
	return lv
}

// BindPageNav registers key bindings for half-page scrolling.
func (lv *LogC) BindPageNav(down, up string) *LogC {
	lv.declaredBindings = append(lv.declaredBindings,
		binding{down, func() { lv.layer.HalfPageDown() }},
		binding{up, func() { lv.following = false; lv.layer.HalfPageUp() }},
	)
	return lv
}

// BindFirstLast registers key bindings for jumping to top/bottom.
func (lv *LogC) BindFirstLast(first, last string) *LogC {
	lv.declaredBindings = append(lv.declaredBindings,
		binding{first, func() { lv.following = false; lv.layer.ScrollToTop() }},
		binding{last, func() { lv.resume() }},
	)
	return lv
}

// BindVimNav wires standard vim-style scroll keys:
// j/k: line, Ctrl-d/u: half-page, g/G: top/bottom
func (lv *LogC) BindVimNav() *LogC {
	return lv.BindNav("j", "k").BindPageNav("<C-d>", "<C-u>").BindFirstLast("g", "G")
}

// bindings implements the bindable interface.
func (lv *LogC) bindings() []binding {
	return lv.declaredBindings
}

// start begins reading from the reader in a background goroutine.
// Called once via sync.Once when the component is first compiled.
func (lv *LogC) start() {
	go lv.readLoop()
}

// readLoop reads lines from the reader and appends them to the buffer.
// Exits when the reader returns EOF or an error.
func (lv *LogC) readLoop() {
	scanner := bufio.NewScanner(lv.reader)

	// increase scanner buffer for long lines
	const maxLineSize = 1024 * 1024 // 1MB
	scanner.Buffer(make([]byte, 64*1024), maxLineSize)

	for scanner.Scan() {
		line := scanner.Text()

		lv.mu.Lock()

		lv.lines = append(lv.lines, line)

		// ring buffer: drop oldest if over limit
		if lv.maxLines > 0 && len(lv.lines) > lv.maxLines {
			dropped := len(lv.lines) - lv.maxLines
			lv.lines = lv.lines[dropped:]
			// adjust scroll position to keep viewing same content
			if !lv.following {
				newScrollY := lv.layer.ScrollY() - dropped
				if newScrollY >= 0 {
					lv.layer.ScrollTo(newScrollY)
				}
				// if newScrollY < 0, content is gone but we keep them at 0
				// they can still scroll down through remaining buffered content
			}
		}

		// always sync so layer has valid content at any viewport size
		lv.syncToLayer()

		if lv.following {
			// following: scroll to end
			if lv.autoScroll {
				lv.layer.ScrollToEnd()
			}
		} else {
			// not following: count new lines for indicator
			lv.newLineCount++
		}

		lv.mu.Unlock()

		// request redraw
		if lv.onUpdate != nil {
			lv.onUpdate()
		}
	}
}

// syncToLayer writes all buffered lines to the layer's buffer.
func (lv *LogC) syncToLayer() {
	if len(lv.lines) == 0 {
		return
	}

	// create exact-sized buffer (EnsureSize only grows, which breaks maxScroll after ring buffer truncates)
	const bufferWidth = 500
	buf := NewBuffer(bufferWidth, len(lv.lines))
	for i, line := range lv.lines {
		buf.WriteStringFast(0, i, line, Style{}, bufferWidth)
	}
	lv.layer.SetBuffer(buf)
}

// compileLogC compiles the Log component into the template.
// Starts the reader goroutine on first compile and returns a LayerView.
func (t *Template) compileLogC(lv *LogC, parent int16, depth int) int16 {
	// collect for later wiring (app not available yet during compile)
	if lv.onUpdate == nil {
		t.pendingLogs = append(t.pendingLogs, lv)
	}

	// start reader goroutine (once)
	lv.started.Do(lv.start)

	// compile as LayerView with the internal layer
	layerView := LayerView(lv.layer).Grow(lv.grow)
	if lv.margin != [4]int16{} {
		layerView = layerView.MarginTRBL(lv.margin[0], lv.margin[1], lv.margin[2], lv.margin[3])
	}

	return t.compileLayerViewC(layerView, parent, depth)
}
