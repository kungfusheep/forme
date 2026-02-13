package forme

import (
	"fmt"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// Buffer is a 2D grid of cells representing a drawable surface.
type Buffer struct {
	cells     []Cell
	width     int
	height    int
	dirtyMaxY int // highest row written to (for partial clear)

	// Row-level dirty tracking for efficient flush
	dirtyRows []bool
	allDirty  bool // true after Clear() - all rows need checking
}

// emptyBufferCache is a pre-filled buffer of empty cells for fast clearing via copy()
var emptyBufferCache []Cell

// NewBuffer creates a new buffer with the given dimensions.
func NewBuffer(width, height int) *Buffer {
	cells := make([]Cell, width*height)
	empty := EmptyCell()
	for i := range cells {
		cells[i] = empty
	}
	return &Buffer{
		cells:     cells,
		width:     width,
		height:    height,
		dirtyRows: make([]bool, height),
		allDirty:  true, // new buffer needs full flush
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
// When drawing border characters, automatically merges with existing borders.
func (b *Buffer) Set(x, y int, c Cell) {
	if !b.InBounds(x, y) {
		return
	}
	idx := b.index(x, y)
	existing := b.cells[idx]

	// Merge border characters
	if merged, ok := mergeBorders(existing.Rune, c.Rune); ok {
		c.Rune = merged
	}

	b.cells[idx] = c

	// Track dirty region
	if y > b.dirtyMaxY {
		b.dirtyMaxY = y
	}
	b.dirtyRows[y] = true
}

// SetFast sets a cell without border merging. Use for text/progress where
// you know the content isn't a border character.
func (b *Buffer) SetFast(x, y int, c Cell) {
	if y < 0 || y >= b.height || x < 0 || x >= b.width {
		return
	}
	b.cells[y*b.width+x] = c
	if y > b.dirtyMaxY {
		b.dirtyMaxY = y
	}
	b.dirtyRows[y] = true
}

// Partial block characters for sub-character progress bar precision (1/8 to 8/8)
var partialBlocks = [9]rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

// WriteProgressBar writes a progress bar directly to the buffer.
// Uses partial block characters for smooth sub-character precision.
// Background color fills the empty space for seamless appearance.
// Batch write - much faster than per-cell Set calls.
func (b *Buffer) WriteProgressBar(x, y, width int, ratio float32, style Style) {
	if y < 0 || y >= b.height {
		return
	}
	if y > b.dirtyMaxY {
		b.dirtyMaxY = y
	}
	b.dirtyRows[y] = true

	// Calculate fill in eighths for sub-character precision
	totalEighths := int(ratio * float32(width) * 8)
	if totalEighths < 0 {
		totalEighths = 0
	}
	maxEighths := width * 8
	if totalEighths > maxEighths {
		totalEighths = maxEighths
	}

	fullBlocks := totalEighths / 8
	partialEighths := totalEighths % 8

	// Direct slice access - no bounds check per cell, no border merge
	base := y * b.width

	// Dark gray background for empty areas - partial blocks blend into this
	emptyBG := Color{Mode: ColorRGB, R: 60, G: 60, B: 60}
	filledCell := Cell{Rune: '█', Style: style}
	emptyCell := Cell{Rune: ' ', Style: Style{BG: emptyBG}}
	partialStyle := Style{FG: style.FG, BG: emptyBG}

	end := x + width
	if end > b.width {
		end = b.width
	}
	if x < 0 {
		x = 0
	}

	for i := x; i < end; i++ {
		pos := i - x
		if pos < fullBlocks {
			b.cells[base+i] = filledCell
		} else if pos == fullBlocks && partialEighths > 0 {
			b.cells[base+i] = Cell{Rune: partialBlocks[partialEighths], Style: partialStyle}
		} else {
			b.cells[base+i] = emptyCell
		}
	}
}

// WriteStringFast writes a string without border merging.
// Direct slice access for maximum speed.
func (b *Buffer) WriteStringFast(x, y int, s string, style Style, maxWidth int) {
	if y < 0 || y >= b.height {
		return
	}
	if y > b.dirtyMaxY {
		b.dirtyMaxY = y
	}
	b.dirtyRows[y] = true

	base := y * b.width
	written := 0
	for _, r := range s {
		if written >= maxWidth || x >= b.width {
			break
		}
		if x >= 0 {
			b.cells[base+x] = Cell{Rune: r, Style: style}
		}
		x++
		written++
	}
}

// WriteSpans writes multiple styled text spans sequentially.
// Each span has its own style. Spans are written left to right.
// Handles double-width CJK characters correctly.
func (b *Buffer) WriteSpans(x, y int, spans []Span, maxWidth int) {
	if y < 0 || y >= b.height {
		return
	}
	if y > b.dirtyMaxY {
		b.dirtyMaxY = y
	}
	b.dirtyRows[y] = true

	base := y * b.width
	written := 0
	for _, span := range spans {
		for _, r := range span.Text {
			rw := runewidth.RuneWidth(r)
			if rw == 0 {
				rw = 1 // treat zero-width as 1 for positioning
			}
			if written+rw > maxWidth || x+rw > b.width {
				return
			}
			if x >= 0 {
				b.cells[base+x] = Cell{Rune: r, Style: span.Style}
				// for double-width chars, fill second cell with placeholder
				if rw == 2 && x+1 < b.width {
					b.cells[base+x+1] = Cell{Rune: 0, Style: span.Style}
				}
			}
			x += rw
			written += rw
		}
	}
}

// WriteLeader writes "Label.....Value" format with fill characters.
// The label is left-aligned, value is right-aligned, fill chars in between.
func (b *Buffer) WriteLeader(x, y int, label, value string, width int, fill rune, style Style) {
	if y < 0 || y >= b.height {
		return
	}
	if y > b.dirtyMaxY {
		b.dirtyMaxY = y
	}
	b.dirtyRows[y] = true

	if fill == 0 {
		fill = '.'
	}

	base := y * b.width
	labelLen := utf8.RuneCountInString(label)
	valueLen := utf8.RuneCountInString(value)

	// Calculate fill length
	fillLen := width - labelLen - valueLen
	if fillLen < 1 {
		fillLen = 1 // at least one fill char
	}

	pos := x
	// Write label
	for _, r := range label {
		if pos >= b.width || pos-x >= width {
			return
		}
		if pos >= 0 {
			b.cells[base+pos] = Cell{Rune: r, Style: style}
		}
		pos++
	}

	// Write fill
	for i := 0; i < fillLen && pos < b.width && pos-x < width; i++ {
		if pos >= 0 {
			b.cells[base+pos] = Cell{Rune: fill, Style: style}
		}
		pos++
	}

	// Write value
	for _, r := range value {
		if pos >= b.width || pos-x >= width {
			return
		}
		if pos >= 0 {
			b.cells[base+pos] = Cell{Rune: r, Style: style}
		}
		pos++
	}
}

// sparklineChars maps values 0-7 to Unicode block characters.
var sparklineChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// WriteSparkline writes a sparkline chart using Unicode block characters.
func (b *Buffer) WriteSparkline(x, y int, values []float64, width int, min, max float64, style Style) {
	if y < 0 || y >= b.height || len(values) == 0 {
		return
	}
	if y > b.dirtyMaxY {
		b.dirtyMaxY = y
	}
	b.dirtyRows[y] = true

	// Auto-detect min/max if not specified
	if min == 0 && max == 0 {
		min, max = values[0], values[0]
		for _, v := range values {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
	}

	// Handle case where all values are the same
	valRange := max - min
	if valRange == 0 {
		valRange = 1
	}

	base := y * b.width
	dataLen := len(values)

	for i := 0; i < width && x+i < b.width; i++ {
		// Map position to data index (handles width != len(values))
		dataIdx := i * dataLen / width
		if dataIdx >= dataLen {
			dataIdx = dataLen - 1
		}

		// Normalize value to 0-7 range
		normalized := (values[dataIdx] - min) / valRange
		if normalized < 0 {
			normalized = 0
		} else if normalized > 1 {
			normalized = 1
		}
		charIdx := int(normalized * 7.99) // 0-7
		if charIdx > 7 {
			charIdx = 7
		}

		if x+i >= 0 {
			b.cells[base+x+i] = Cell{Rune: sparklineChars[charIdx], Style: style}
		}
	}
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
// Uses copy() from a cached empty buffer for speed (memmove vs scalar loop).
func (b *Buffer) Clear() {
	size := len(b.cells)

	// Grow cache if needed (one-time cost)
	if len(emptyBufferCache) < size {
		emptyBufferCache = make([]Cell, size)
		empty := EmptyCell()
		for i := range emptyBufferCache {
			emptyBufferCache[i] = empty
		}
	}

	// Fast path: copy uses optimized memmove
	copy(b.cells, emptyBufferCache[:size])
	b.dirtyMaxY = 0
	b.allDirty = true
	// Clear individual row flags (allDirty takes precedence)
	for i := range b.dirtyRows {
		b.dirtyRows[i] = false
	}
}

// RowDirty returns true if the given row has been modified since last ClearDirtyFlags.
// If allDirty is set (after Clear/Resize), all rows are considered dirty.
func (b *Buffer) RowDirty(y int) bool {
	if b.allDirty {
		return true
	}
	if y < 0 || y >= len(b.dirtyRows) {
		return false
	}
	return b.dirtyRows[y]
}

// ClearDirtyFlags resets all row dirty flags after a flush.
// Call this after Screen.Flush() to start tracking changes for next frame.
func (b *Buffer) ClearDirtyFlags() {
	b.allDirty = false
	for i := range b.dirtyRows {
		b.dirtyRows[i] = false
	}
}

// MarkAllDirty forces all rows to be considered dirty.
// Useful after external modifications or for testing.
func (b *Buffer) MarkAllDirty() {
	b.allDirty = true
}

// ResetDirtyMax resets the dirty tracking without clearing content.
// Use when you know the template will overwrite all cells.
func (b *Buffer) ResetDirtyMax() {
	b.dirtyMaxY = -1
}

// ClearDirty clears only the rows that were written to since last clear.
// Much faster than Clear() when content doesn't fill the buffer.
func (b *Buffer) ClearDirty() {
	if b.dirtyMaxY < 0 {
		return
	}

	// Only clear rows 0..dirtyMaxY
	size := (b.dirtyMaxY + 1) * b.width
	if size > len(b.cells) {
		size = len(b.cells)
	}

	// Ensure cache is big enough
	if len(emptyBufferCache) < size {
		emptyBufferCache = make([]Cell, len(b.cells))
		empty := EmptyCell()
		for i := range emptyBufferCache {
			emptyBufferCache[i] = empty
		}
	}

	copy(b.cells[:size], emptyBufferCache[:size])

	// Mark cleared rows as dirty (content changed) and reset tracking
	for y := 0; y <= b.dirtyMaxY && y < b.height; y++ {
		b.dirtyRows[y] = true
	}
	b.dirtyMaxY = 0
}

// ClearLine clears a single line to empty cells.
func (b *Buffer) ClearLine(y int) {
	if y < 0 || y >= b.height {
		return
	}
	base := y * b.width
	empty := EmptyCell()
	for x := 0; x < b.width; x++ {
		b.cells[base+x] = empty
	}
	b.dirtyRows[y] = true
}

// ClearLineWithStyle clears a single line with a styled space cell.
func (b *Buffer) ClearLineWithStyle(y int, style Style) {
	if y < 0 || y >= b.height {
		return
	}
	base := y * b.width
	cell := Cell{Rune: ' ', Style: style}
	for x := 0; x < b.width; x++ {
		b.cells[base+x] = cell
	}
	b.dirtyRows[y] = true
}

// FillRect fills a rectangular region with the given cell.
// Uses direct slice writes (no border merge) for non-border cells,
// falls back to Set() only when the cell is a border character.
func (b *Buffer) FillRect(x, y, width, height int, c Cell) {
	// fast path: non-border fills bypass Set() entirely
	if c.Rune < boxDrawingMin || c.Rune > boxDrawingMax {
		for dy := 0; dy < height; dy++ {
			row := y + dy
			if row < 0 || row >= b.height {
				continue
			}
			if row > b.dirtyMaxY {
				b.dirtyMaxY = row
			}
			b.dirtyRows[row] = true
			base := row * b.width
			for dx := 0; dx < width; dx++ {
				col := x + dx
				if col >= 0 && col < b.width {
					b.cells[base+col] = c
				}
			}
		}
		return
	}
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

// WriteStringPadded writes a string and pads with spaces to fill width.
// This allows skipping Clear() when UI structure is stable.
func (b *Buffer) WriteStringPadded(x, y int, s string, style Style, width int) {
	written := 0
	for _, r := range s {
		if written >= width || !b.InBounds(x, y) {
			break
		}
		b.Set(x, y, NewCell(r, style))
		x++
		written++
	}
	// Pad with spaces
	space := NewCell(' ', style)
	for written < width && b.InBounds(x, y) {
		b.Set(x, y, space)
		x++
		written++
	}
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

// Box junction characters for merged borders
const (
	BoxTeeDown  = '┬' // ─ meets │ from below
	BoxTeeUp    = '┴' // ─ meets │ from above
	BoxTeeRight = '├' // │ meets ─ from right
	BoxTeeLeft  = '┤' // │ meets ─ from left
	BoxCross    = '┼' // all four directions
)

// Box drawing range constants for fast rejection
const (
	boxDrawingMin = 0x2500
	boxDrawingMax = 0x257F
)

// borderEdgesArray provides O(1) lookup for border edge bits
// Index = rune - boxDrawingMin, value = edge bits (0 = not a border char)
// Using bits: 1=top, 2=right, 4=bottom, 8=left
var borderEdgesArray = [128]uint8{
	0x00: 0b1010, // ─ BoxHorizontal (0x2500)
	0x02: 0b0101, // │ BoxVertical (0x2502)
	0x0C: 0b0110, // ┌ BoxTopLeft (0x250C)
	0x10: 0b1100, // ┐ BoxTopRight (0x2510)
	0x14: 0b0011, // └ BoxBottomLeft (0x2514)
	0x18: 0b1001, // ┘ BoxBottomRight (0x2518)
	0x1C: 0b0111, // ├ BoxTeeRight (0x251C)
	0x24: 0b1101, // ┤ BoxTeeLeft (0x2524)
	0x2C: 0b1110, // ┬ BoxTeeDown (0x252C)
	0x34: 0b1011, // ┴ BoxTeeUp (0x2534)
	0x3C: 0b1111, // ┼ BoxCross (0x253C)
	0x6D: 0b0110, // ╭ BoxRoundedTopLeft (0x256D)
	0x6E: 0b1100, // ╮ BoxRoundedTopRight (0x256E)
	0x6F: 0b1001, // ╯ BoxRoundedBottomRight (0x256F)
	0x70: 0b0011, // ╰ BoxRoundedBottomLeft (0x2570)
}

// edgesToBorderArray provides O(1) lookup from edge bits to border rune
// Index = edge bits (0-15), value = border rune (0 = invalid)
var edgesToBorderArray = [16]rune{
	0b0011: BoxBottomLeft,
	0b0101: BoxVertical,
	0b0110: BoxTopLeft,
	0b0111: BoxTeeRight,
	0b1001: BoxBottomRight,
	0b1010: BoxHorizontal,
	0b1011: BoxTeeUp,
	0b1100: BoxTopRight,
	0b1101: BoxTeeLeft,
	0b1110: BoxTeeDown,
	0b1111: BoxCross,
}

// mergeBorders combines two border characters into one.
// Returns the merged rune and true if both were border chars, otherwise false.
func mergeBorders(existing, new rune) (rune, bool) {
	// Fast path: reject non-border characters immediately (99% of calls)
	if existing < boxDrawingMin || existing > boxDrawingMax {
		return new, false
	}
	if new < boxDrawingMin || new > boxDrawingMax {
		return new, false
	}

	// Array lookup for edge bits
	existingEdges := borderEdgesArray[existing-boxDrawingMin]
	newEdges := borderEdgesArray[new-boxDrawingMin]
	if existingEdges == 0 || newEdges == 0 {
		return new, false
	}

	// Merge and lookup result
	merged := existingEdges | newEdges
	if result := edgesToBorderArray[merged]; result != 0 {
		return result, true
	}
	return new, false
}

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

// GetLine returns the content of a single line as a string (trimmed).
func (b *Buffer) GetLine(y int) string {
	if y < 0 || y >= b.height {
		return ""
	}
	var line []byte
	lastNonSpace := -1
	for x := 0; x < b.width; x++ {
		c := b.Get(x, y)
		r := c.Rune
		if r == 0 {
			r = ' '
		}
		line = append(line, string(r)...)
		if r != ' ' {
			lastNonSpace = len(line)
		}
	}
	if lastNonSpace >= 0 {
		return string(line[:lastNonSpace])
	}
	return ""
}

// GetLineStyled returns a line with embedded ANSI escape codes for styles.
func (b *Buffer) GetLineStyled(y int) string {
	if y < 0 || y >= b.height {
		return ""
	}
	var line []byte
	var lastStyle Style
	defaultStyle := DefaultStyle()

	for x := 0; x < b.width; x++ {
		c := b.Get(x, y)
		r := c.Rune
		if r == 0 {
			r = ' '
		}

		// Emit style change if needed
		if !c.Style.Equal(lastStyle) {
			line = append(line, b.styleToANSI(c.Style)...)
			lastStyle = c.Style
		}
		line = append(line, string(r)...)
	}

	// Reset style at end
	if !lastStyle.Equal(defaultStyle) {
		line = append(line, "\x1b[0m"...)
	}

	return string(line)
}

// styleToANSI converts a Style to ANSI escape codes.
func (b *Buffer) styleToANSI(style Style) string {
	var codes []byte
	codes = append(codes, "\x1b[0"...)

	// Attributes
	if style.Attr.Has(AttrBold) {
		codes = append(codes, ";1"...)
	}
	if style.Attr.Has(AttrDim) {
		codes = append(codes, ";2"...)
	}
	if style.Attr.Has(AttrItalic) {
		codes = append(codes, ";3"...)
	}
	if style.Attr.Has(AttrUnderline) {
		codes = append(codes, ";4"...)
	}
	if style.Attr.Has(AttrInverse) {
		codes = append(codes, ";7"...)
	}

	// Foreground
	codes = append(codes, b.colorToANSI(style.FG, true)...)

	// Background
	codes = append(codes, b.colorToANSI(style.BG, false)...)

	codes = append(codes, 'm')
	return string(codes)
}

// colorToANSI converts a Color to ANSI escape code fragment.
func (b *Buffer) colorToANSI(c Color, fg bool) string {
	switch c.Mode {
	case ColorDefault:
		if fg {
			return ";39"
		}
		return ";49"
	case Color16:
		base := 30
		if !fg {
			base = 40
		}
		if c.Index >= 8 {
			return fmt.Sprintf(";%d", base+60+int(c.Index-8))
		}
		return fmt.Sprintf(";%d", base+int(c.Index))
	case Color256:
		if fg {
			return fmt.Sprintf(";38;5;%d", c.Index)
		}
		return fmt.Sprintf(";48;5;%d", c.Index)
	case ColorRGB:
		if fg {
			return fmt.Sprintf(";38;2;%d;%d;%d", c.R, c.G, c.B)
		}
		return fmt.Sprintf(";48;2;%d;%d;%d", c.R, c.G, c.B)
	}
	return ""
}

// String returns the buffer contents as a string (for testing/debugging).
// Each row is separated by a newline. Trailing spaces are preserved.
func (b *Buffer) String() string {
	var result []byte
	for y := 0; y < b.height; y++ {
		for x := 0; x < b.width; x++ {
			c := b.Get(x, y)
			if c.Rune == 0 {
				result = append(result, ' ')
			} else {
				result = append(result, string(c.Rune)...)
			}
		}
		if y < b.height-1 {
			result = append(result, '\n')
		}
	}
	return string(result)
}

// StringTrimmed returns the buffer contents with trailing spaces removed per line.
func (b *Buffer) StringTrimmed() string {
	var lines []string
	for y := 0; y < b.height; y++ {
		var line []byte
		lastNonSpace := -1
		for x := 0; x < b.width; x++ {
			c := b.Get(x, y)
			r := c.Rune
			if r == 0 {
				r = ' '
			}
			line = append(line, string(r)...)
			if r != ' ' {
				lastNonSpace = len(line)
			}
		}
		if lastNonSpace >= 0 {
			lines = append(lines, string(line[:lastNonSpace]))
		} else {
			lines = append(lines, "")
		}
	}
	// Trim trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	result := ""
	for i, line := range lines {
		result += line
		if i < len(lines)-1 {
			result += "\n"
		}
	}
	return result
}

// Blit copies a rectangular region from src buffer to this buffer.
// srcX, srcY: top-left corner in source buffer (for scrolling)
// dstX, dstY: top-left corner in destination buffer
// width, height: size of region to copy
// Uses optimized row-by-row copy() for speed.
func (b *Buffer) Blit(src *Buffer, srcX, srcY, dstX, dstY, width, height int) {
	// Clip to source bounds
	if srcX < 0 {
		width += srcX
		dstX -= srcX
		srcX = 0
	}
	if srcY < 0 {
		height += srcY
		dstY -= srcY
		srcY = 0
	}
	if srcX+width > src.width {
		width = src.width - srcX
	}
	if srcY+height > src.height {
		height = src.height - srcY
	}

	// Clip to destination bounds
	if dstX < 0 {
		width += dstX
		srcX -= dstX
		dstX = 0
	}
	if dstY < 0 {
		height += dstY
		srcY -= dstY
		dstY = 0
	}
	if dstX+width > b.width {
		width = b.width - dstX
	}
	if dstY+height > b.height {
		height = b.height - dstY
	}

	// Nothing to copy
	if width <= 0 || height <= 0 {
		return
	}

	// Row-by-row copy using copy() for speed
	for y := 0; y < height; y++ {
		srcStart := (srcY+y)*src.width + srcX
		dstStart := (dstY+y)*b.width + dstX
		copy(b.cells[dstStart:dstStart+width], src.cells[srcStart:srcStart+width])
		b.dirtyRows[dstY+y] = true
	}

	// Update dirty tracking
	if dstY+height-1 > b.dirtyMaxY {
		b.dirtyMaxY = dstY + height - 1
	}
}

// CopyFrom copies all cells from src to b using a single bulk copy.
// Requires both buffers to have identical dimensions.
// This is much faster than cell-by-cell copy for full-buffer transfers.
func (b *Buffer) CopyFrom(src *Buffer) {
	if b.width == src.width && b.height == src.height {
		copy(b.cells, src.cells)
		b.dirtyMaxY = src.dirtyMaxY
		// Mark all rows dirty since we did a full copy
		b.allDirty = true
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

	// Resize dirty tracking - mark all dirty after resize
	b.dirtyRows = make([]bool, height)
	b.allDirty = true
}
