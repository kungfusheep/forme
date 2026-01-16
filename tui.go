// Package tui provides a high-performance terminal UI framework.
package tui

import "unsafe"

// Attribute represents text styling attributes that can be combined.
type Attribute uint8

const (
	AttrNone          Attribute = 0
	AttrBold          Attribute = 1 << iota
	AttrDim
	AttrItalic
	AttrUnderline
	AttrBlink
	AttrInverse
	AttrStrikethrough
)

// Has returns true if the attribute set contains the given attribute.
func (a Attribute) Has(attr Attribute) bool {
	return a&attr != 0
}

// With returns a new attribute set with the given attribute added.
func (a Attribute) With(attr Attribute) Attribute {
	return a | attr
}

// Without returns a new attribute set with the given attribute removed.
func (a Attribute) Without(attr Attribute) Attribute {
	return a &^ attr
}

// ColorMode represents the color mode for a color value.
type ColorMode uint8

const (
	ColorDefault ColorMode = iota // Terminal default
	Color16                       // Basic 16 colors (0-15)
	Color256                      // 256 color palette (0-255)
	ColorRGB                      // 24-bit true color
)

// Color represents a terminal color.
type Color struct {
	Mode    ColorMode
	R, G, B uint8 // For RGB mode
	Index   uint8 // For 16/256 mode
}

// DefaultColor returns the terminal's default color.
func DefaultColor() Color {
	return Color{Mode: ColorDefault}
}

// BasicColor returns one of the 16 basic terminal colors.
func BasicColor(index uint8) Color {
	return Color{Mode: Color16, Index: index}
}

// PaletteColor returns one of the 256 palette colors.
func PaletteColor(index uint8) Color {
	return Color{Mode: Color256, Index: index}
}

// RGB returns a 24-bit true color.
func RGB(r, g, b uint8) Color {
	return Color{Mode: ColorRGB, R: r, G: g, B: b}
}

// Hex returns a 24-bit true color from a hex value (e.g., 0xFF5500).
func Hex(hex uint32) Color {
	return Color{
		Mode: ColorRGB,
		R:    uint8((hex >> 16) & 0xFF),
		G:    uint8((hex >> 8) & 0xFF),
		B:    uint8(hex & 0xFF),
	}
}

// Standard basic colors for convenience.
var (
	Black   = BasicColor(0)
	Red     = BasicColor(1)
	Green   = BasicColor(2)
	Yellow  = BasicColor(3)
	Blue    = BasicColor(4)
	Magenta = BasicColor(5)
	Cyan    = BasicColor(6)
	White   = BasicColor(7)

	// Bright variants
	BrightBlack   = BasicColor(8)
	BrightRed     = BasicColor(9)
	BrightGreen   = BasicColor(10)
	BrightYellow  = BasicColor(11)
	BrightBlue    = BasicColor(12)
	BrightMagenta = BasicColor(13)
	BrightCyan    = BasicColor(14)
	BrightWhite   = BasicColor(15)
)

// Equal returns true if two colors are equal.
func (c Color) Equal(other Color) bool {
	return c == other
}

// Style combines foreground, background colors and attributes.
type Style struct {
	FG   Color
	BG   Color
	Attr Attribute
}

// DefaultStyle returns a style with default colors and no attributes.
func DefaultStyle() Style {
	return Style{
		FG: DefaultColor(),
		BG: DefaultColor(),
	}
}

// Foreground returns a new style with the given foreground color.
func (s Style) Foreground(c Color) Style {
	s.FG = c
	return s
}

// Background returns a new style with the given background color.
func (s Style) Background(c Color) Style {
	s.BG = c
	return s
}

// Bold returns a new style with bold enabled.
func (s Style) Bold() Style {
	s.Attr = s.Attr.With(AttrBold)
	return s
}

// Dim returns a new style with dim enabled.
func (s Style) Dim() Style {
	s.Attr = s.Attr.With(AttrDim)
	return s
}

// Italic returns a new style with italic enabled.
func (s Style) Italic() Style {
	s.Attr = s.Attr.With(AttrItalic)
	return s
}

// Underline returns a new style with underline enabled.
func (s Style) Underline() Style {
	s.Attr = s.Attr.With(AttrUnderline)
	return s
}

// Inverse returns a new style with inverse enabled.
func (s Style) Inverse() Style {
	s.Attr = s.Attr.With(AttrInverse)
	return s
}

// Strikethrough returns a new style with strikethrough enabled.
func (s Style) Strikethrough() Style {
	s.Attr = s.Attr.With(AttrStrikethrough)
	return s
}

// Equal returns true if two styles are equal.
func (s Style) Equal(other Style) bool {
	return s == other
}

// Cell represents a single character cell on the terminal.
type Cell struct {
	Rune  rune
	Style Style
}

// EmptyCell returns a cell with a space and default style.
func EmptyCell() Cell {
	return Cell{Rune: ' ', Style: DefaultStyle()}
}

// NewCell creates a cell with the given rune and style.
func NewCell(r rune, style Style) Cell {
	return Cell{Rune: r, Style: style}
}

// Equal returns true if two cells are equal.
func (c Cell) Equal(other Cell) bool {
	return c == other
}


// Flex contains layout properties for display components.
// Embedded in Row, Col, Text, etc. for consistent layout behavior.
// Layout only - no visual styling here.
type Flex struct {
	PercentWidth float32 // fraction of parent width (0.5 = 50%)
	Width        int16   // explicit width in characters
	Height       int16   // explicit height in lines
	FlexGrow     float32 // share of remaining space (0 = none, 1 = equal share)
}

// Text displays text content.
type Text struct {
	Flex
	Content any   // string or *string
	Style   Style // styling (use Attr for bold, dim, etc.)
}

// Leader displays "Label.....Value" with dots filling the space.
// Supports pointer bindings for dynamic updates.
type Leader struct {
	Label any   // string or *string
	Value any   // string or *string
	Width int16 // total width (0 = fill available from parent)
	Fill  rune  // fill character (0 = '.')
	Style Style // styling (use Attr for bold, dim, etc.)
}

// Custom allows user-defined components without modifying the framework.
// Use this for specialized widgets that aren't covered by built-in primitives.
// Note: Custom components use function calls (not inlined like built-ins),
// but with viewport culling this overhead is negligible.
type Custom struct {
	// Measure returns natural (width, height) given available width.
	// Called during the measure phase of rendering.
	Measure func(availW int16) (w, h int16)

	// Render draws the component to the buffer at the given position.
	// Called during the draw phase with computed geometry.
	Render func(buf *Buffer, x, y, w, h int16)
}

// Progress displays a progress bar.
type Progress struct {
	Flex
	Value    any   // int or *int (0-100)
	BarWidth int16 // width of the bar in characters (distinct from Flex.Width layout width)
}

// LayerView displays a scrollable layer.
// The Layer is pre-rendered content that gets blitted to screen.
type LayerView struct {
	Flex
	Layer      *Layer // the pre-rendered layer
	ViewHeight int16  // viewport height (0 = fill available, distinct from Flex.Height)
	ViewWidth  int16  // viewport width (0 = fill available, distinct from Flex.Width)
}

// Row arranges children horizontally.
type Row struct {
	flex
	Children []any
	Title    string // title for bordered containers
	Gap      int8

	// Set via chainable methods
	border   BorderStyle
	borderFG *Color
}

// Col arranges children vertically.
type Col struct {
	flex
	Children []any
	Title    string // title for bordered containers
	Gap      int8

	// Set via chainable methods
	border   BorderStyle
	borderFG *Color
}

// flex contains internal layout properties (use chainable methods to set).
type flex struct {
	percentWidth float32
	width        int16
	height       int16
	flexGrow     float32
}

// Chainable layout methods for Row

// WidthPct sets width as percentage of parent (0.5 = 50%).
func (r Row) WidthPct(pct float32) Row { r.percentWidth = pct; return r }

// Width sets explicit width in characters.
func (r Row) Width(w int16) Row { r.width = w; return r }

// Height sets explicit height in lines.
func (r Row) Height(h int16) Row { r.height = h; return r }

// Grow sets flex grow factor.
func (r Row) Grow(g float32) Row { r.flexGrow = g; return r }

// Border sets the border style.
func (r Row) Border(b BorderStyle) Row { r.border = b; return r }

// BorderFG sets the border foreground color.
func (r Row) BorderFG(c Color) Row { r.borderFG = &c; return r }

// Chainable layout methods for Col

// WidthPct sets width as percentage of parent (0.5 = 50%).
func (c Col) WidthPct(pct float32) Col { c.percentWidth = pct; return c }

// Width sets explicit width in characters.
func (c Col) Width(w int16) Col { c.width = w; return c }

// Height sets explicit height in lines.
func (c Col) Height(h int16) Col { c.height = h; return c }

// Grow sets flex grow factor.
func (c Col) Grow(g float32) Col { c.flexGrow = g; return c }

// Border sets the border style.
func (c Col) Border(b BorderStyle) Col { c.border = b; return c }

// BorderFG sets the border foreground color.
func (c Col) BorderFG(fg Color) Col { c.borderFG = &fg; return c }

// IfNode conditionally renders content.
type IfNode struct {
	Cond any // *bool
	Then any
}

// ElseNode renders when preceding If was false.
type ElseNode struct {
	Then any
}

// Else creates an else branch.
func Else(then any) ElseNode {
	return ElseNode{Then: then}
}

// ForEachNode iterates over a slice.
type ForEachNode struct {
	Items  any // *[]T
	Render any // func(*T) any
}

// ForEach creates an iteration over a slice.
func ForEach(items any, render any) ForEachNode {
	return ForEachNode{Items: items, Render: render}
}

// SelectionList displays a list of items with selection marker.
// Items must be a pointer to a slice (*[]T).
// Selected must be a pointer to an int (*int) tracking the selected index.
// Render is optional - if nil, items are rendered using fmt.Sprintf("%v", item).
// Marker defaults to "> " if not specified.
type SelectionList struct {
	Items      any    // *[]T - pointer to slice of items
	Selected   *int   // pointer to selected index
	Marker     string // selection marker (default "> ")
	Render     any    // func(*T) any - optional, renders each item
	MaxVisible int    // max items to show (0 = all)
	len        int    // cached length for bounds checking
	offset     int    // scroll offset for windowing
}

// ensureVisible adjusts scroll offset so selected item is visible.
func (s *SelectionList) ensureVisible() {
	if s.Selected == nil || s.MaxVisible <= 0 {
		return
	}
	sel := *s.Selected
	// Scroll up if selection is above visible window
	if sel < s.offset {
		s.offset = sel
	}
	// Scroll down if selection is below visible window
	if sel >= s.offset+s.MaxVisible {
		s.offset = sel - s.MaxVisible + 1
	}
}

// Up moves selection up by one. Safe to use directly with app.Handle.
func (s *SelectionList) Up(m any) {
	if s.Selected != nil && *s.Selected > 0 {
		*s.Selected--
		s.ensureVisible()
	}
}

// Down moves selection down by one. Safe to use directly with app.Handle.
func (s *SelectionList) Down(m any) {
	if s.Selected != nil && s.len > 0 && *s.Selected < s.len-1 {
		*s.Selected++
		s.ensureVisible()
	}
}

// PageUp moves selection up by page size (MaxVisible or 10).
func (s *SelectionList) PageUp(m any) {
	if s.Selected != nil {
		pageSize := 10
		if s.MaxVisible > 0 {
			pageSize = s.MaxVisible
		}
		*s.Selected -= pageSize
		if *s.Selected < 0 {
			*s.Selected = 0
		}
		s.ensureVisible()
	}
}

// PageDown moves selection down by page size (MaxVisible or 10).
func (s *SelectionList) PageDown(m any) {
	if s.Selected != nil && s.len > 0 {
		pageSize := 10
		if s.MaxVisible > 0 {
			pageSize = s.MaxVisible
		}
		*s.Selected += pageSize
		if *s.Selected >= s.len {
			*s.Selected = s.len - 1
		}
		s.ensureVisible()
	}
}

// First moves selection to the first item.
func (s *SelectionList) First(m any) {
	if s.Selected != nil {
		*s.Selected = 0
		s.ensureVisible()
	}
}

// Last moves selection to the last item.
func (s *SelectionList) Last(m any) {
	if s.Selected != nil && s.len > 0 {
		*s.Selected = s.len - 1
		s.ensureVisible()
	}
}

// Span represents a styled segment of text within RichText.
type Span struct {
	Text  string
	Style Style
}

// RichText displays text with mixed inline styles.
// Spans can be []Span (static) or *[]Span (dynamic binding).
type RichText struct {
	Flex
	Spans any // []Span or *[]Span
}

// Rich creates a RichText from a mix of strings and Spans.
// Plain strings get default styling, Spans keep their styling.
//
// Example:
//
//	Rich("Hello ", Bold("world"), "!")
func Rich(parts ...any) RichText {
	spans := make([]Span, 0, len(parts))
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			spans = append(spans, Span{Text: v})
		case Span:
			spans = append(spans, v)
		}
	}
	return RichText{Spans: spans}
}

// Styled creates a span with the given style.
func Styled(text string, style Style) Span {
	return Span{Text: text, Style: style}
}

// Bold creates a bold text span.
func Bold(text string) Span {
	return Span{Text: text, Style: Style{Attr: AttrBold}}
}

// Dim creates a dim text span.
func Dim(text string) Span {
	return Span{Text: text, Style: Style{Attr: AttrDim}}
}

// Italic creates an italic text span.
func Italic(text string) Span {
	return Span{Text: text, Style: Style{Attr: AttrItalic}}
}

// Underline creates an underlined text span.
func Underline(text string) Span {
	return Span{Text: text, Style: Style{Attr: AttrUnderline}}
}

// Inverse creates an inverse text span.
func Inverse(text string) Span {
	return Span{Text: text, Style: Style{Attr: AttrInverse}}
}

// FG creates a span with foreground color.
func FG(text string, color Color) Span {
	return Span{Text: text, Style: Style{FG: color}}
}

// BG creates a span with background color.
func BG(text string, color Color) Span {
	return Span{Text: text, Style: Style{BG: color}}
}

// sliceHeader is the runtime representation of a slice.
// Used for zero-allocation slice iteration.
type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

// isWithinRange checks if a pointer falls within a memory range.
// Used to determine if a pointer is inside a struct for offset calculation.
func isWithinRange(ptr, base unsafe.Pointer, size uintptr) bool {
	p := uintptr(ptr)
	b := uintptr(base)
	return p >= b && p < b+size
}
