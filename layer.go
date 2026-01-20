package tui

// Layer is a pre-rendered buffer with scroll management.
// Content is rendered once (expensive), then blitted to screen each frame (cheap).
type Layer struct {
	buffer   *Buffer
	scrollY  int
	maxScroll int

	// Viewport dimensions (set during layout)
	viewWidth  int
	viewHeight int
}

// NewLayer creates a new empty layer.
func NewLayer() *Layer {
	return &Layer{}
}

// SetContent renders a template to the layer's internal buffer.
// Call this when content changes (e.g., page navigation).
func (l *Layer) SetContent(tmpl *Template, width, height int) {
	l.buffer = NewBuffer(width, height)
	tmpl.Execute(l.buffer, int16(width), int16(height))
	l.scrollY = 0
	l.updateMaxScroll()
}

// SetBuffer directly sets the layer's buffer.
// Use this if you're managing the buffer yourself.
func (l *Layer) SetBuffer(buf *Buffer) {
	l.buffer = buf
	l.scrollY = 0
	l.updateMaxScroll()
}

// Buffer returns the underlying buffer (for direct manipulation if needed).
func (l *Layer) Buffer() *Buffer {
	return l.buffer
}

// updateMaxScroll recalculates the maximum scroll position.
func (l *Layer) updateMaxScroll() {
	if l.buffer == nil || l.viewHeight <= 0 {
		l.maxScroll = 0
		return
	}
	l.maxScroll = l.buffer.Height() - l.viewHeight
	if l.maxScroll < 0 {
		l.maxScroll = 0
	}
	// Clamp current scroll to new bounds
	if l.scrollY > l.maxScroll {
		l.scrollY = l.maxScroll
	}
}

// SetViewport sets the viewport dimensions for the layer.
// This should be called before ScrollTo if the viewport height affects maxScroll.
// Also called internally by the framework during layout.
func (l *Layer) SetViewport(width, height int) {
	l.viewWidth = width
	l.viewHeight = height
	l.updateMaxScroll()
}

// ScrollY returns the current scroll position.
func (l *Layer) ScrollY() int {
	return l.scrollY
}

// MaxScroll returns the maximum scroll position.
func (l *Layer) MaxScroll() int {
	return l.maxScroll
}

// ContentHeight returns the total content height.
func (l *Layer) ContentHeight() int {
	if l.buffer == nil {
		return 0
	}
	return l.buffer.Height()
}

// ViewportHeight returns the visible viewport height.
func (l *Layer) ViewportHeight() int {
	return l.viewHeight
}

// ScrollTo sets the scroll position, clamping to valid range.
func (l *Layer) ScrollTo(y int) {
	if y < 0 {
		y = 0
	}
	if y > l.maxScroll {
		y = l.maxScroll
	}
	l.scrollY = y
}

// ScrollDown scrolls down by n lines.
func (l *Layer) ScrollDown(n int) {
	l.ScrollTo(l.scrollY + n)
}

// ScrollUp scrolls up by n lines.
func (l *Layer) ScrollUp(n int) {
	l.ScrollTo(l.scrollY - n)
}

// ScrollToTop scrolls to the top.
func (l *Layer) ScrollToTop() {
	l.scrollY = 0
}

// ScrollToEnd scrolls to the bottom.
func (l *Layer) ScrollToEnd() {
	l.scrollY = l.maxScroll
}

// PageDown scrolls down by one viewport height.
func (l *Layer) PageDown() {
	l.ScrollDown(l.viewHeight)
}

// PageUp scrolls up by one viewport height.
func (l *Layer) PageUp() {
	l.ScrollUp(l.viewHeight)
}

// HalfPageDown scrolls down by half a viewport.
func (l *Layer) HalfPageDown() {
	l.ScrollDown(l.viewHeight / 2)
}

// HalfPageUp scrolls up by half a viewport.
func (l *Layer) HalfPageUp() {
	l.ScrollUp(l.viewHeight / 2)
}

// blit copies the visible portion of the layer to the destination buffer.
func (l *Layer) blit(dst *Buffer, dstX, dstY, width, height int) {
	if l.buffer == nil {
		return
	}
	dst.Blit(l.buffer, 0, l.scrollY, dstX, dstY, width, height)
}

// SetLine updates a single line in the layer buffer with styled spans.
// This is the efficient path for partial updates (e.g., cursor moved).
// Clears the line first to prevent ghost content from shorter lines.
func (l *Layer) SetLine(y int, spans []Span) {
	if l.buffer == nil || y < 0 || y >= l.buffer.Height() {
		return
	}
	l.buffer.ClearLine(y)
	l.buffer.WriteSpans(0, y, spans, l.buffer.Width())
}

// SetLineString updates a single line with a plain string and style.
// Clears the line first to prevent ghost content from shorter lines.
func (l *Layer) SetLineString(y int, s string, style Style) {
	if l.buffer == nil || y < 0 || y >= l.buffer.Height() {
		return
	}
	l.buffer.ClearLine(y)
	l.buffer.WriteStringFast(0, y, s, style, l.buffer.Width())
}

// SetLineAt updates a line with spans at a given x offset.
// Clears the entire line with clearStyle first, then writes spans at offset x.
// Use this to avoid creating padding spans for margins.
func (l *Layer) SetLineAt(y, x int, spans []Span, clearStyle Style) {
	if l.buffer == nil || y < 0 || y >= l.buffer.Height() {
		return
	}
	l.buffer.ClearLineWithStyle(y, clearStyle)
	l.buffer.WriteSpans(x, y, spans, l.buffer.Width()-x)
}

// EnsureSize ensures the buffer is at least the given size.
// If the buffer needs to grow, existing content is preserved.
func (l *Layer) EnsureSize(width, height int) {
	if l.buffer == nil {
		l.buffer = NewBuffer(width, height)
		return
	}
	if l.buffer.Width() >= width && l.buffer.Height() >= height {
		return
	}
	// Need to grow - create new buffer and copy
	newWidth := max(l.buffer.Width(), width)
	newHeight := max(l.buffer.Height(), height)
	newBuf := NewBuffer(newWidth, newHeight)
	newBuf.Blit(l.buffer, 0, 0, 0, 0, l.buffer.Width(), l.buffer.Height())
	l.buffer = newBuf
	l.updateMaxScroll()
}

// Clear clears the entire layer buffer.
func (l *Layer) Clear() {
	if l.buffer != nil {
		l.buffer.Clear()
	}
}
