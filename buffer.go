package tui

// Buffer is a 2D grid of cells representing a drawable surface.
type Buffer struct {
	cells  []Cell
	width  int
	height int
}

// NewBuffer creates a new buffer with the given dimensions.
func NewBuffer(width, height int) *Buffer {
	cells := make([]Cell, width*height)
	empty := EmptyCell()
	for i := range cells {
		cells[i] = empty
	}
	return &Buffer{
		cells:  cells,
		width:  width,
		height: height,
	}
}

// Width returns the buffer width.
func (b *Buffer) Width() int {
	return b.width
}

// Height returns the buffer height.
func (b *Buffer) Height() int {
	return b.height
}

// Size returns the buffer dimensions.
func (b *Buffer) Size() (width, height int) {
	return b.width, b.height
}

// InBounds returns true if the given coordinates are within the buffer.
func (b *Buffer) InBounds(x, y int) bool {
	return x >= 0 && x < b.width && y >= 0 && y < b.height
}

// index converts x,y coordinates to a slice index.
func (b *Buffer) index(x, y int) int {
	return y*b.width + x
}

// Get returns the cell at the given coordinates.
// Returns an empty cell if out of bounds.
func (b *Buffer) Get(x, y int) Cell {
	if !b.InBounds(x, y) {
		return EmptyCell()
	}
	return b.cells[b.index(x, y)]
}

// Set sets the cell at the given coordinates.
// Does nothing if out of bounds.
func (b *Buffer) Set(x, y int, c Cell) {
	if !b.InBounds(x, y) {
		return
	}
	b.cells[b.index(x, y)] = c
}

// SetRune sets just the rune at the given coordinates, preserving style.
func (b *Buffer) SetRune(x, y int, r rune) {
	if !b.InBounds(x, y) {
		return
	}
	idx := b.index(x, y)
	b.cells[idx].Rune = r
}

// SetStyle sets just the style at the given coordinates, preserving rune.
func (b *Buffer) SetStyle(x, y int, s Style) {
	if !b.InBounds(x, y) {
		return
	}
	idx := b.index(x, y)
	b.cells[idx].Style = s
}

// Fill fills the entire buffer with the given cell.
func (b *Buffer) Fill(c Cell) {
	for i := range b.cells {
		b.cells[i] = c
	}
}

// Clear clears the buffer to empty cells with default style.
func (b *Buffer) Clear() {
	b.Fill(EmptyCell())
}

// FillRect fills a rectangular region with the given cell.
func (b *Buffer) FillRect(x, y, width, height int, c Cell) {
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			b.Set(x+dx, y+dy, c)
		}
	}
}

// WriteString writes a string at the given coordinates with the given style.
// Returns the number of cells written.
func (b *Buffer) WriteString(x, y int, s string, style Style) int {
	written := 0
	for _, r := range s {
		if !b.InBounds(x, y) {
			break
		}
		b.Set(x, y, NewCell(r, style))
		x++
		written++
	}
	return written
}

// WriteStringClipped writes a string, stopping at maxWidth.
// Returns the number of cells written.
func (b *Buffer) WriteStringClipped(x, y int, s string, style Style, maxWidth int) int {
	written := 0
	for _, r := range s {
		if written >= maxWidth || !b.InBounds(x, y) {
			break
		}
		b.Set(x, y, NewCell(r, style))
		x++
		written++
	}
	return written
}

// HLine draws a horizontal line of the given rune.
func (b *Buffer) HLine(x, y, length int, r rune, style Style) {
	for i := 0; i < length; i++ {
		b.Set(x+i, y, NewCell(r, style))
	}
}

// VLine draws a vertical line of the given rune.
func (b *Buffer) VLine(x, y, length int, r rune, style Style) {
	for i := 0; i < length; i++ {
		b.Set(x, y+i, NewCell(r, style))
	}
}

// Box drawing characters for borders.
const (
	BoxHorizontal        = '─'
	BoxVertical          = '│'
	BoxTopLeft           = '┌'
	BoxTopRight          = '┐'
	BoxBottomLeft        = '└'
	BoxBottomRight       = '┘'
	BoxRoundedTopLeft    = '╭'
	BoxRoundedTopRight   = '╮'
	BoxRoundedBottomLeft = '╰'
	BoxRoundedBottomRight = '╯'
	BoxDoubleHorizontal  = '═'
	BoxDoubleVertical    = '║'
	BoxDoubleTopLeft     = '╔'
	BoxDoubleTopRight    = '╗'
	BoxDoubleBottomLeft  = '╚'
	BoxDoubleBottomRight = '╝'
)

// BorderStyle defines the characters used for drawing borders.
type BorderStyle struct {
	Horizontal  rune
	Vertical    rune
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
}

// Standard border styles.
var (
	BorderSingle = BorderStyle{
		Horizontal:  BoxHorizontal,
		Vertical:    BoxVertical,
		TopLeft:     BoxTopLeft,
		TopRight:    BoxTopRight,
		BottomLeft:  BoxBottomLeft,
		BottomRight: BoxBottomRight,
	}
	BorderRounded = BorderStyle{
		Horizontal:  BoxHorizontal,
		Vertical:    BoxVertical,
		TopLeft:     BoxRoundedTopLeft,
		TopRight:    BoxRoundedTopRight,
		BottomLeft:  BoxRoundedBottomLeft,
		BottomRight: BoxRoundedBottomRight,
	}
	BorderDouble = BorderStyle{
		Horizontal:  BoxDoubleHorizontal,
		Vertical:    BoxDoubleVertical,
		TopLeft:     BoxDoubleTopLeft,
		TopRight:    BoxDoubleTopRight,
		BottomLeft:  BoxDoubleBottomLeft,
		BottomRight: BoxDoubleBottomRight,
	}
)

// DrawBorder draws a border around the given rectangle.
func (b *Buffer) DrawBorder(x, y, width, height int, border BorderStyle, style Style) {
	if width < 2 || height < 2 {
		return
	}

	// Corners
	b.Set(x, y, NewCell(border.TopLeft, style))
	b.Set(x+width-1, y, NewCell(border.TopRight, style))
	b.Set(x, y+height-1, NewCell(border.BottomLeft, style))
	b.Set(x+width-1, y+height-1, NewCell(border.BottomRight, style))

	// Horizontal lines
	for i := 1; i < width-1; i++ {
		b.Set(x+i, y, NewCell(border.Horizontal, style))
		b.Set(x+i, y+height-1, NewCell(border.Horizontal, style))
	}

	// Vertical lines
	for i := 1; i < height-1; i++ {
		b.Set(x, y+i, NewCell(border.Vertical, style))
		b.Set(x+width-1, y+i, NewCell(border.Vertical, style))
	}
}

// Region returns a view into a rectangular region of the buffer.
// The returned Region shares the underlying cells with the parent buffer.
type Region struct {
	buf    *Buffer
	x, y   int
	width  int
	height int
}

// Region creates a view into a rectangular region of the buffer.
func (b *Buffer) Region(x, y, width, height int) *Region {
	return &Region{
		buf:    b,
		x:      x,
		y:      y,
		width:  width,
		height: height,
	}
}

// Width returns the region width.
func (r *Region) Width() int {
	return r.width
}

// Height returns the region height.
func (r *Region) Height() int {
	return r.height
}

// Size returns the region dimensions.
func (r *Region) Size() (width, height int) {
	return r.width, r.height
}

// InBounds returns true if the given coordinates are within the region.
func (r *Region) InBounds(x, y int) bool {
	return x >= 0 && x < r.width && y >= 0 && y < r.height
}

// Get returns the cell at the given region-relative coordinates.
func (r *Region) Get(x, y int) Cell {
	if !r.InBounds(x, y) {
		return EmptyCell()
	}
	return r.buf.Get(r.x+x, r.y+y)
}

// Set sets the cell at the given region-relative coordinates.
func (r *Region) Set(x, y int, c Cell) {
	if !r.InBounds(x, y) {
		return
	}
	r.buf.Set(r.x+x, r.y+y, c)
}

// Fill fills the region with the given cell.
func (r *Region) Fill(c Cell) {
	for y := 0; y < r.height; y++ {
		for x := 0; x < r.width; x++ {
			r.Set(x, y, c)
		}
	}
}

// Clear clears the region to empty cells.
func (r *Region) Clear() {
	r.Fill(EmptyCell())
}

// WriteString writes a string at the given region-relative coordinates.
func (r *Region) WriteString(x, y int, s string, style Style) int {
	written := 0
	for _, ch := range s {
		if !r.InBounds(x, y) {
			break
		}
		r.Set(x, y, NewCell(ch, style))
		x++
		written++
	}
	return written
}

// DrawBorder draws a border around the entire region.
func (r *Region) DrawBorder(border BorderStyle, style Style) {
	if r.width < 2 || r.height < 2 {
		return
	}

	// Corners
	r.Set(0, 0, NewCell(border.TopLeft, style))
	r.Set(r.width-1, 0, NewCell(border.TopRight, style))
	r.Set(0, r.height-1, NewCell(border.BottomLeft, style))
	r.Set(r.width-1, r.height-1, NewCell(border.BottomRight, style))

	// Horizontal lines
	for i := 1; i < r.width-1; i++ {
		r.Set(i, 0, NewCell(border.Horizontal, style))
		r.Set(i, r.height-1, NewCell(border.Horizontal, style))
	}

	// Vertical lines
	for i := 1; i < r.height-1; i++ {
		r.Set(0, i, NewCell(border.Vertical, style))
		r.Set(r.width-1, i, NewCell(border.Vertical, style))
	}
}

// Resize resizes the buffer to new dimensions.
// Existing content is preserved where it fits.
func (b *Buffer) Resize(width, height int) {
	if width == b.width && height == b.height {
		return
	}

	newCells := make([]Cell, width*height)
	empty := EmptyCell()
	for i := range newCells {
		newCells[i] = empty
	}

	// Copy existing content
	minWidth := b.width
	if width < minWidth {
		minWidth = width
	}
	minHeight := b.height
	if height < minHeight {
		minHeight = height
	}

	for y := 0; y < minHeight; y++ {
		for x := 0; x < minWidth; x++ {
			newCells[y*width+x] = b.cells[y*b.width+x]
		}
	}

	b.cells = newCells
	b.width = width
	b.height = height
}
