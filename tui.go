// Package forme provides a high-performance terminal UI framework.
package forme

import "unsafe"

// Attribute represents text styling attributes that can be combined.
type Attribute uint8

const (
	AttrNone Attribute = 0
	AttrBold Attribute = 1 << iota
	AttrDim
	AttrItalic
	AttrUnderline
	AttrBlink
	AttrInverse
	AttrStrikethrough
)

// TextTransform represents text case transformations.
type TextTransform uint8

const (
	TransformNone TextTransform = iota
	TransformUppercase
	TransformLowercase
	TransformCapitalize // first letter of each word
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
	Color16                       // Basic 16 colours (0-15)
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

// BasicColor returns one of the 16 basic terminal colours.
func BasicColor(index uint8) Color {
	return Color{Mode: Color16, Index: index}
}

// PaletteColor returns one of the 256 palette colours.
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

// LerpColor blends between two colours. t=0 returns a, t=1 returns b.
func LerpColor(a, b Color, t float64) Color {
	// Clamp t to 0-1
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return RGB(
		uint8(float64(a.R)+t*(float64(b.R)-float64(a.R))),
		uint8(float64(a.G)+t*(float64(b.G)-float64(a.G))),
		uint8(float64(a.B)+t*(float64(b.B)-float64(a.B))),
	)
}

// Standard basic colours for convenience.
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

// Equal returns true if two colours are equal.
func (c Color) Equal(other Color) bool {
	return c == other
}

// Style combines foreground, background colours and attributes.
type Style struct {
	FG        Color
	BG        Color // text background (behind characters)
	Fill      Color // container fill (entire area)
	Attr      Attribute
	Transform TextTransform // text case transformation (uppercase, lowercase, etc.)
}

// DefaultStyle returns a style with default colours and no attributes.
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

// FillColor returns a new style with the given fill color (for containers).
func (s Style) FillColor(c Color) Style {
	s.Fill = c
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

// Uppercase returns a new style with uppercase text transform.
func (s Style) Uppercase() Style {
	s.Transform = TransformUppercase
	return s
}

// Lowercase returns a new style with lowercase text transform.
func (s Style) Lowercase() Style {
	s.Transform = TransformLowercase
	return s
}

// Capitalize returns a new style with capitalize transform (first letter of each word).
func (s Style) Capitalize() Style {
	s.Transform = TransformCapitalize
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

// TextNode displays text content.
type TextNode struct {
	Flex
	Content any   // string or *string
	Style   Style // styling (use Attr for bold, dim, etc.)
}

// LeaderNode displays "Label.....Value" with dots filling the space.
// Supports pointer bindings for dynamic updates.
type LeaderNode struct {
	Label any   // string or *string
	Value any   // string or *string
	Width int16 // total width (0 = fill available from parent)
	Fill  rune  // fill character (0 = '.')
	Style Style // styling (use Attr for bold, dim, etc.)
}

// Align specifies text alignment within a cell.
type Align uint8

const (
	AlignLeft Align = iota
	AlignRight
	AlignCenter
)

// TableColumn defines a column in a Table.
type TableColumn struct {
	Header string // column header text
	Width  int    // column width (0 = auto-size)
	Align  Align  // text alignment
}

// Table displays tabular data with columns and optional headers.
// Uses pointer bindings for dynamic data updates.
type Table struct {
	Columns     []TableColumn // column definitions
	Rows        any           // *[][]string - pointer to row data
	ShowHeader  bool          // show header row
	HeaderStyle Style         // style for header row
	RowStyle    Style         // style for data rows
	AltRowStyle Style         // style for alternating rows (if non-zero)
}

// SparklineNode displays a mini chart using Unicode block characters.
// Values are normalized to fit within the available height (1 character).
// Uses: ▁▂▃▄▅▆▇█
type SparklineNode struct {
	Values any     // []float64 or *[]float64
	Width  int16   // width (0 = auto from data length)
	Min    float64 // minimum value (0 = auto-detect)
	Max    float64 // maximum value (0 = auto-detect)
	Style  Style   // styling
}

// HRuleNode draws a horizontal line that fills available width.
// Default character is '─' (box drawing light horizontal).
type HRuleNode struct {
	Char  rune  // line character (0 = '─')
	Style Style // styling
}

// VRuleNode draws a vertical line that fills available height.
// Default character is '│' (box drawing light vertical).
type VRuleNode struct {
	Char  rune  // line character (0 = '│')
	Style Style // styling
}

// SpacerNode creates empty space with specified dimensions.
// If no dimensions are set, Spacer grows to fill available space (implicit Grow(1)).
// With explicit Width/Height, it becomes a fixed-size spacer.
//
// Examples:
//   - Spacer{}              → fills available space (grows)
//   - Spacer{Height: 1}     → fixed 1-line vertical gap
//   - Spacer{Width: 10}     → fixed 10-char horizontal gap
//   - Spacer{}.Grow(2)      → grows with weight 2
//   - Spacer{Char: '.'}     → dotted leader (fills with dots)
type SpacerNode struct {
	flex
	Width  int16 // fixed width (0 = grow to fill)
	Height int16 // fixed height (0 = grow to fill, but defaults to 1 in VBox if not growing)
	Char   rune  // fill character (0 = empty space)
	Style  Style // style for fill character
}

// Grow sets flex grow factor for Spacer.
func (s SpacerNode) Grow(g float32) SpacerNode { s.flexGrow = g; return s }

// FG sets the foreground color for the fill character.
func (s SpacerNode) FG(c Color) SpacerNode { s.Style.FG = c; return s }

// SpinnerNode displays an animated loading indicator.
// The Frame pointer controls which animation frame to show.
// Increment Frame and re-render to animate.
type SpinnerNode struct {
	Frame  *int     // pointer to current frame index
	Frames []string // custom frames (nil = default braille spinner)
	Style  Style    // styling
}

// SpinnerBraille is the default spinner animation (braille dots).
var SpinnerBraille = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// SpinnerDots is a simple dot spinner.
var SpinnerDots = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

// SpinnerLine is a line spinner.
var SpinnerLine = []string{"-", "\\", "|", "/"}

// SpinnerCircle is a circle spinner.
var SpinnerCircle = []string{"◐", "◓", "◑", "◒"}

// ScrollbarNode displays a visual scroll indicator.
// Vertical by default; set Horizontal to true for horizontal scrollbar.
type ScrollbarNode struct {
	ContentSize int   // total content size
	ViewSize    int   // visible viewport size
	Position    *int  // pointer to current scroll position
	Length      int16 // scrollbar length (0 = fill available)
	Horizontal  bool  // true for horizontal scrollbar
	TrackChar   rune  // track character (default: '│' or '─')
	ThumbChar   rune  // thumb character (default: '█')
	TrackStyle  Style // track styling
	ThumbStyle  Style // thumb styling
}

// TabsStyle defines the visual style for tab headers.
type TabsStyle uint8

const (
	TabsStyleUnderline TabsStyle = iota // active tab has underline
	TabsStyleBox                        // tabs in boxes
	TabsStyleBracket                    // tabs with [ ] brackets
)

// TabsNode displays a row of tab headers with active tab indicator.
type TabsNode struct {
	Labels        []string  // tab labels
	Selected      *int      // pointer to selected tab index
	Style         TabsStyle // visual style
	Gap           int       // gap between tabs (default: 2)
	ActiveStyle   Style     // style for active tab
	InactiveStyle Style     // style for inactive tabs
}

// TreeNode represents a node in a tree structure.
type TreeNode struct {
	Label    string      // display label
	Children []*TreeNode // child nodes
	Expanded bool        // whether children are visible
	Data     any         // optional user data
}

// TreeView displays a hierarchical tree structure.
type TreeView struct {
	Root          *TreeNode // root node (can be hidden)
	ShowRoot      bool      // whether to display the root node
	Indent        int       // indentation per level (default: 2)
	ShowLines     bool      // show connecting lines (├ └ │)
	ExpandedChar  rune      // character for expanded nodes (default: '▼')
	CollapsedChar rune      // character for collapsed nodes (default: '▶')
	LeafChar      rune      // character for leaf nodes (default: ' ')
	Style         Style     // styling for labels
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

// JumpNode wraps a component to make it a jump target.
// When jump mode is active, a label is displayed at this component's position.
// When the user types the label, OnSelect is called.
type JumpNode struct {
	Child    any    // The wrapped component
	OnSelect func() // Called when this target is selected
	Style    Style  // Optional: per-target label style override
}

// ProgressNode displays a progress bar.
type ProgressNode struct {
	Flex
	Value    any   // int or *int (0-100)
	BarWidth int16 // width of the bar in characters (distinct from Flex.Width layout width)
}

// LayerViewNode displays a scrollable layer.
// The Layer is pre-rendered content that gets blitted to screen.
type LayerViewNode struct {
	Flex
	Layer      *Layer // the pre-rendered layer
	ViewHeight int16  // viewport height (0 = fill available, distinct from Flex.Height)
	ViewWidth  int16  // viewport width (0 = fill available, distinct from Flex.Width)
}

// Grow sets the flex grow factor for this layer.
func (l LayerViewNode) Grow(factor float32) LayerViewNode { l.FlexGrow = factor; return l }

// HBoxNode arranges children horizontally.
type HBoxNode struct {
	flex
	Children     []any
	Title        string // title for bordered containers
	Gap          int8
	CascadeStyle *Style // style inherited by children (pointer for dynamic themes)

	// Set via chainable methods
	border   BorderStyle
	borderFG *Color
	borderBG *Color
	margin   [4]int16 // top, right, bottom, left
}

// VBoxNode arranges children vertically.
type VBoxNode struct {
	flex
	Children     []any
	Title        string // title for bordered containers
	Gap          int8
	CascadeStyle *Style // style inherited by children (pointer for dynamic themes)

	// Set via chainable methods
	border   BorderStyle
	borderFG *Color
	borderBG *Color
	margin   [4]int16 // top, right, bottom, left
}

// flex contains internal layout properties (use chainable methods to set).
type flex struct {
	percentWidth float32
	width        int16
	height       int16
	flexGrow     float32
	fitContent   bool
}

// Chainable layout methods for HBox

// WidthPct sets width as percentage of parent (0.5 = 50%).
func (r HBoxNode) WidthPct(pct float32) HBoxNode { r.percentWidth = pct; return r }

// Width sets explicit width in characters.
func (r HBoxNode) Width(w int16) HBoxNode { r.width = w; return r }

// Height sets explicit height in lines.
func (r HBoxNode) Height(h int16) HBoxNode { r.height = h; return r }

// Grow sets flex grow factor.
func (r HBoxNode) Grow(g float32) HBoxNode { r.flexGrow = g; return r }

// Border sets the border style.
func (r HBoxNode) Border(b BorderStyle) HBoxNode { r.border = b; return r }

// BorderFG sets the border foreground color.
func (r HBoxNode) BorderFG(c Color) HBoxNode { r.borderFG = &c; return r }

// BorderBG sets the border background color.
func (r HBoxNode) BorderBG(c Color) HBoxNode { r.borderBG = &c; return r }

// Margin sets uniform margin on all sides.
func (r HBoxNode) Margin(all int16) HBoxNode {
	r.margin = [4]int16{all, all, all, all}
	return r
}

// MarginXY sets vertical and horizontal margin.
func (r HBoxNode) MarginXY(vertical, horizontal int16) HBoxNode {
	r.margin = [4]int16{vertical, horizontal, vertical, horizontal}
	return r
}

// MarginTRBL sets individual margins for each side.
func (r HBoxNode) MarginTRBL(top, right, bottom, left int16) HBoxNode {
	r.margin = [4]int16{top, right, bottom, left}
	return r
}

// Chainable layout methods for VBox

// WidthPct sets width as percentage of parent (0.5 = 50%).
func (c VBoxNode) WidthPct(pct float32) VBoxNode { c.percentWidth = pct; return c }

// Width sets explicit width in characters.
func (c VBoxNode) Width(w int16) VBoxNode { c.width = w; return c }

// Height sets explicit height in lines.
func (c VBoxNode) Height(h int16) VBoxNode { c.height = h; return c }

// Grow sets flex grow factor.
func (c VBoxNode) Grow(g float32) VBoxNode { c.flexGrow = g; return c }

// Border sets the border style.
func (c VBoxNode) Border(b BorderStyle) VBoxNode { c.border = b; return c }

// BorderFG sets the border foreground color.
func (c VBoxNode) BorderFG(fg Color) VBoxNode { c.borderFG = &fg; return c }

// BorderBG sets the border background color.
func (c VBoxNode) BorderBG(bg Color) VBoxNode { c.borderBG = &bg; return c }

// Margin sets uniform margin on all sides.
func (c VBoxNode) Margin(all int16) VBoxNode {
	c.margin = [4]int16{all, all, all, all}
	return c
}

// MarginXY sets vertical and horizontal margin.
func (c VBoxNode) MarginXY(vertical, horizontal int16) VBoxNode {
	c.margin = [4]int16{vertical, horizontal, vertical, horizontal}
	return c
}

// MarginTRBL sets individual margins for each side.
func (c VBoxNode) MarginTRBL(top, right, bottom, left int16) VBoxNode {
	c.margin = [4]int16{top, right, bottom, left}
	return c
}

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

// SelectionList displays a list of items with selection marker.
// Items must be a pointer to a slice (*[]T).
// Selected must be a pointer to an int (*int) tracking the selected index.
// Render is optional - if nil, items are rendered using fmt.Sprintf("%v", item).
// Marker defaults to "> " if not specified.
type SelectionList struct {
	Items         any    // *[]T - pointer to slice of items
	Selected      *int   // pointer to selected index
	Marker        string // selection marker (default "> ", use " " for no visible marker)
	MarkerStyle   Style  // style for marker text (merged with SelectedStyle.BG for selected rows)
	Render        any    // func(*T) any - optional, renders each item
	MaxVisible    int    // max items to show (0 = all)
	Style         Style  // default style for non-selected rows (e.g., background)
	SelectedStyle Style  // style for selected row (e.g., background color)
	len           int    // cached length for bounds checking
	offset        int    // scroll offset for windowing
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

// RichTextNode displays text with mixed inline styles.
// Spans can be []Span (static) or *[]Span (dynamic binding).
type RichTextNode struct {
	Flex
	Spans any // []Span or *[]Span
}

// Rich creates a RichText from a mix of strings and Spans.
// Plain strings get default styling, Spans keep their styling.
//
// Example:
//
//	Rich("Hello ", Bold("world"), "!")
func Rich(parts ...any) RichTextNode {
	spans := make([]Span, 0, len(parts))
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			spans = append(spans, Span{Text: v})
		case Span:
			spans = append(spans, v)
		}
	}
	return RichTextNode{Spans: spans}
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

// Field bundles the state for a text input field.
// Use with TextInput.Field for cleaner multi-field forms.
type Field struct {
	Value  string
	Cursor int
}

// Clear resets the field value and cursor.
func (f *Field) Clear() {
	f.Value = ""
	f.Cursor = 0
}

// FocusGroup tracks which field in a group is focused.
// Share a single FocusGroup across multiple inputs.
type FocusGroup struct {
	Current int
}

// TextInput is a single-line text input field.
// Wire up input handling via riffkey.NewTextHandler or riffkey.NewFieldHandler.
//
// Example with Field + FocusGroup (recommended for forms):
//
//	name := tui.Field{}
//	focus := tui.FocusGroup{}
//	tui.TextInput{Field: &name, FocusGroup: &focus, FocusIndex: 0}
//
// Example with separate pointers (for single fields):
//
//	tui.TextInput{Value: &query, Cursor: &cursor, Placeholder: "Search..."}
type TextInput struct {
	// Field-based API (recommended for forms)
	Field      *Field      // Bundles Value + Cursor in one struct
	FocusGroup *FocusGroup // Shared focus tracker - cursor shows when FocusGroup.Current == FocusIndex
	FocusIndex int         // This field's index in the focus group

	// Pointer-based API (for single fields)
	Value   *string // Bound text value (ignored if Field is set)
	Cursor  *int    // Cursor position (ignored if Field is set)
	Focused *bool   // Show cursor only when true (ignored if FocusGroup is set)

	// Common options
	Placeholder      string // Shown when value is empty
	Width            int    // Field width (0 = fill available)
	Mask             rune   // Password mask character (0 = none)
	Style            Style  // Text style
	PlaceholderStyle Style  // Placeholder style (zero = dim text)
	CursorStyle      Style  // Cursor style (zero = reverse video)
}

// OverlayNode displays content floating above the main view.
// Use for modals, dialogs, and floating windows.
// Control visibility with forme.If:
//
//	forme.If(&showModal).Eq(true).Then(forme.Overlay{Child: ...})
type OverlayNode struct {
	Centered   bool  // true = center on screen (default behavior if X/Y not set)
	X, Y       int   // explicit position (used if Centered is false)
	Width      int   // explicit width (0 = auto from content)
	Height     int   // explicit height (0 = auto from content)
	Backdrop   bool  // draw dimmed backdrop behind overlay
	BackdropFG Color // backdrop dim color (default: BrightBlack)
	BG         Color // background color for overlay content area (fills before rendering child)
	Child      any   // overlay content
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
