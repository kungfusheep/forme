package forme

// Layer is a pre-rendered buffer with scroll management.
// Content is rendered once (expensive), then blitted to screen each frame (cheap).
//
// If Render is set, the framework automatically calls it before blitting when
// the viewport dimensions change. This ensures content is always rendered at
// the correct size without manual timing coordination.
type Layer struct {
	buffer    *Buffer
	scrollY   int
	maxScroll int

	// Viewport dimensions (set during layout)
	viewWidth  int
	viewHeight int

	// Track dimensions at last render to detect when re-render needed
	lastRenderWidth  int
	lastRenderHeight int

	// Cursor state (buffer-relative coordinates)
	cursor Cursor

	// Screen offset (set by framework during blit for cursor translation)
	screenX, screenY int

	// Render populates the layer buffer. Called automatically by the framework
	// before blitting when viewport dimensions change. The layer ensures its
	// buffer exists and is sized appropriately before calling this.
	//
	// Width changes always trigger a re-render (text wrapping changes).
	// Height changes trigger a re-render if content height depends on viewport.
	Render func()
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
// Called internally by the framework during layout.
func (l *Layer) SetViewport(width, height int) {
	l.viewWidth = width
	l.viewHeight = height
	l.updateMaxScroll()
}

// NeedsRender returns true if the layer needs to re-render before blitting.
// Width changes always require re-render (text wrapping). Height changes
// require re-render if this is the first render or content is height-dependent.
func (l *Layer) NeedsRender() bool {
	if l.Render == nil {
		return false
	}
	// first render, or width changed (text wrapping)
	return l.lastRenderWidth == 0 || l.lastRenderWidth != l.viewWidth
}

// prepare ensures the layer is ready to blit. Called by the framework before
// blitting. If Render is set and dimensions changed, calls Render automatically.
func (l *Layer) prepare() {
	if !l.NeedsRender() {
		return
	}
	l.lastRenderWidth = l.viewWidth
	l.lastRenderHeight = l.viewHeight
	l.Render()
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

// ViewportWidth returns the visible viewport width.
func (l *Layer) ViewportWidth() int {
	return l.viewWidth
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

// =============================================================================
// Cursor API
// =============================================================================

// SetCursor sets the cursor position in buffer coordinates.
// The framework translates this to screen coordinates when rendering.
func (l *Layer) SetCursor(x, y int) {
	l.cursor.X = x
	l.cursor.Y = y
}

// SetCursorStyle sets the cursor visual style.
func (l *Layer) SetCursorStyle(style CursorShape) {
	l.cursor.Style = style
}

// ShowCursor makes the cursor visible.
func (l *Layer) ShowCursor() {
	l.cursor.Visible = true
}

// HideCursor hides the cursor.
func (l *Layer) HideCursor() {
	l.cursor.Visible = false
}

// Cursor returns the full cursor state.
func (l *Layer) Cursor() Cursor {
	return l.cursor
}

// ScreenCursor returns the cursor position in screen coordinates.
// This accounts for the layer's position on screen and scroll offset.
// Returns the cursor and whether it's visible and within the viewport.
func (l *Layer) ScreenCursor() (x, y int, visible bool) {
	if !l.cursor.Visible {
		return 0, 0, false
	}

	// cursor Y relative to viewport (account for scroll)
	viewY := l.cursor.Y - l.scrollY

	// check if cursor is within visible viewport
	if viewY < 0 || viewY >= l.viewHeight {
		return 0, 0, false
	}

	// translate to screen coordinates
	x = l.screenX + l.cursor.X
	y = l.screenY + viewY
	return x, y, true
}
