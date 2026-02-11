package forme

import (
	"bufio"
	"io"
	"sync"
)

// FilterLogC is a filterable log viewer. It composes an input and a log
// into a single component with fzf-style filtering.
//
// usage:
//
//	FilterLog(reader).
//	    Placeholder("filter...").
//	    MaxLines(10000).
//	    BindVimNav()
type FilterLogC struct {
	input *InputC
	log   *LogC

	placeholder string
	query       FzfQuery
	lastQuery   string

	// filtered view
	filteredLines []string
	filterMu      sync.Mutex

	// layout
	grow float32

	// focus management
	focused bool
	manager *FocusManager
}

// FilterLog creates a filterable log viewer.
func FilterLog(r io.Reader) *FilterLogC {
	fl := &FilterLogC{
		input: Input(),
		log: &LogC{
			reader:     r,
			maxLines:   10000,
			autoScroll: true,
			layer:      NewLayer(),
			following:  true,
		},
	}

	// wire input changes to filter
	fl.input.declaredTIB = &textInputBinding{
		value:  &fl.input.field.Value,
		cursor: &fl.input.field.Cursor,
		onChange: func(string) {
			fl.updateFilter()
		},
	}

	// default nav keys that don't conflict with text input
	fl.log.BindNav("<C-n>", "<C-p>").
		BindPageNav("<C-d>", "<C-u>").
		BindFirstLast("<C-Home>", "<C-End>")

	return fl
}

// toTemplate returns the template tree for compilation.
func (fl *FilterLogC) toTemplate() any {
	fl.input.placeholder = fl.placeholder
	// propagate focus state for cursor visibility
	if fl.manager != nil {
		fl.input.focused = fl.focused
		fl.input.manager = fl.manager
	}

	children := []any{
		HBox(
			Text("> ").Bold(),
			fl.input,
		),
		fl.log,
	}

	if fl.grow > 0 {
		return VBox.Grow(fl.grow)(children...)
	}
	return VBox(children...)
}

// bindings returns declared bindings from the log component.
func (fl *FilterLogC) bindings() []binding {
	return fl.log.bindings()
}

// textBinding returns the text input binding.
func (fl *FilterLogC) textBinding() *textInputBinding {
	return fl.input.textBinding()
}

// Placeholder sets the input placeholder text.
func (fl *FilterLogC) Placeholder(p string) *FilterLogC {
	fl.placeholder = p
	return fl
}

// MaxLines sets the maximum number of lines to keep in the buffer.
func (fl *FilterLogC) MaxLines(n int) *FilterLogC {
	fl.log.maxLines = n
	return fl
}

// Grow sets the flex grow factor.
func (fl *FilterLogC) Grow(g float32) *FilterLogC {
	fl.grow = g
	fl.log.grow = g
	return fl
}

// BindNav registers key bindings for scrolling.
func (fl *FilterLogC) BindNav(down, up string) *FilterLogC {
	fl.log.BindNav(down, up)
	return fl
}

// BindPageNav registers key bindings for half-page scrolling.
func (fl *FilterLogC) BindPageNav(down, up string) *FilterLogC {
	fl.log.BindPageNav(down, up)
	return fl
}

// BindFirstLast registers key bindings for jumping to top/bottom.
func (fl *FilterLogC) BindFirstLast(first, last string) *FilterLogC {
	fl.log.BindFirstLast(first, last)
	return fl
}

// BindVimNav wires vim-style scroll keys (overrides defaults).
func (fl *FilterLogC) BindVimNav() *FilterLogC {
	// clear default bindings and use vim style
	fl.log.declaredBindings = nil
	fl.log.BindVimNav()
	return fl
}

// Ref provides access to the FilterLogC for external references.
func (fl *FilterLogC) Ref(f func(*FilterLogC)) *FilterLogC {
	f(fl)
	return fl
}

// ManagedBy registers this FilterLog with a FocusManager.
func (fl *FilterLogC) ManagedBy(fm *FocusManager) *FilterLogC {
	fl.manager = fm
	fl.focused = false
	fm.Register(fl)
	return fl
}

// focusBinding implements Focusable (returns the input's binding).
func (fl *FilterLogC) focusBinding() *textInputBinding {
	return fl.input.declaredTIB
}

// setFocused implements Focusable.
func (fl *FilterLogC) setFocused(focused bool) {
	fl.focused = focused
	fl.input.focused = focused
}

// Focused returns whether this FilterLog currently has focus.
func (fl *FilterLogC) Focused() bool {
	return fl.focused
}

// NewLines returns the number of new lines while not following.
func (fl *FilterLogC) NewLines() int {
	return fl.log.NewLines()
}

// Layer returns the underlying layer for scroll info.
func (fl *FilterLogC) Layer() *Layer {
	return fl.log.layer
}

// Clear resets the filter input.
func (fl *FilterLogC) Clear() {
	fl.input.Clear()
	fl.updateFilter()
}

// Active reports whether a filter is currently applied.
func (fl *FilterLogC) Active() bool {
	return !fl.query.Empty()
}

// updateFilter re-applies the filter when query changes.
func (fl *FilterLogC) updateFilter() {
	query := fl.input.Value()
	if query == fl.lastQuery {
		return
	}
	fl.lastQuery = query
	fl.query = ParseFzfQuery(query)

	// trigger re-sync on next render
	fl.log.mu.Lock()
	fl.log.syncToLayerFiltered(&fl.query)
	fl.log.mu.Unlock()
}

// Override the log's syncToLayer to support filtering
func (lc *LogC) syncToLayerFiltered(query *FzfQuery) {
	if len(lc.lines) == 0 {
		return
	}

	const bufferWidth = 500

	if query == nil || query.Empty() {
		// no filter, show all lines
		buf := NewBuffer(bufferWidth, len(lc.lines))
		for i, line := range lc.lines {
			buf.WriteStringFast(0, i, line, Style{}, bufferWidth)
		}
		lc.layer.SetBuffer(buf)
		return
	}

	// filter lines
	var filtered []string
	for _, line := range lc.lines {
		if _, ok := query.Score(line); ok {
			filtered = append(filtered, line)
		}
	}

	if len(filtered) == 0 {
		// no matches, show empty
		buf := NewBuffer(bufferWidth, 1)
		buf.WriteStringFast(0, 0, "(no matches)", Style{FG: PaletteColor(8)}, bufferWidth)
		lc.layer.SetBuffer(buf)
		return
	}

	buf := NewBuffer(bufferWidth, len(filtered))
	for i, line := range filtered {
		buf.WriteStringFast(0, i, line, Style{}, bufferWidth)
	}
	lc.layer.SetBuffer(buf)
}

// startFiltered begins reading with filter support.
func (fl *FilterLogC) startFiltered() {
	go fl.readLoopFiltered()
}

// readLoopFiltered reads lines and applies filter on each update.
func (fl *FilterLogC) readLoopFiltered() {
	scanner := bufio.NewScanner(fl.log.reader)

	const maxLineSize = 1024 * 1024
	scanner.Buffer(make([]byte, 64*1024), maxLineSize)

	for scanner.Scan() {
		line := scanner.Text()

		fl.log.mu.Lock()

		fl.log.lines = append(fl.log.lines, line)

		// ring buffer: drop oldest if over limit
		if fl.log.maxLines > 0 && len(fl.log.lines) > fl.log.maxLines {
			dropped := len(fl.log.lines) - fl.log.maxLines
			fl.log.lines = fl.log.lines[dropped:]
			if !fl.log.following {
				newScrollY := fl.log.layer.ScrollY() - dropped
				if newScrollY >= 0 {
					fl.log.layer.ScrollTo(newScrollY)
				}
			}
		}

		// sync with current filter
		fl.log.syncToLayerFiltered(&fl.query)

		if fl.log.following {
			if fl.log.autoScroll {
				fl.log.layer.ScrollToEnd()
			}
		} else {
			fl.log.newLineCount++
		}

		fl.log.mu.Unlock()

		if fl.log.onUpdate != nil {
			fl.log.onUpdate()
		}
	}
}

// compileFilterLogC compiles the FilterLog component.
func (t *Template) compileFilterLogC(fl *FilterLogC, parent int16, depth int) int16 {
	// collect bindings
	t.collectBindings(fl)
	t.collectTextInputBinding(fl)

	// wire app for invalidation
	if fl.log.onUpdate == nil {
		t.pendingLogs = append(t.pendingLogs, fl.log)
	}

	// start reader goroutine (once)
	fl.log.started.Do(fl.startFiltered)

	// compile the template tree
	return t.compile(fl.toTemplate(), parent, depth, nil, 0)
}
