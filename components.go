package forme

import "reflect"

// binding represents a declared key binding on a component.
// stored as data during construction, wired to a router during setup.
type binding struct {
	pattern string
	handler any
}

// textInputBinding represents an InputC that wants unmatched keys routed to it.
type textInputBinding struct {
	value    *string
	cursor   *int
	onChange func(string) // optional callback when value changes
}

// ============================================================================
// Functional Component API
// ============================================================================
//
// Container components (VBox, HBox) use a function-type-with-methods pattern:
//   VBox(children...)                    - simple usage
//   VBox.Fill(c).Gap(2)(children...)     - with fill color
//   VBox.CascadeStyle(&s)(children...)   - with style inheritance
//
// Leaf components (Text, Spacer, etc.) use simple functions with method chaining:
//   Text("hello")                        - simple usage
//   Text("hello").Bold().FG(Red)         - with styling
//
// ============================================================================

// Define creates a scoped block for local component helpers and styles.
// The function runs once at compile time (when SetView is called).
// Pointers inside still provide dynamic values at render time.
//
//	app.SetView(
//	    Define(func() any {
//	        dot := func(ok *bool) any {
//	            return If(ok).Then(Text("●")).Else(Text("○"))
//	        }
//	        return VBox(dot(&a), dot(&b), dot(&c))
//	    }),
//	)
func Define(fn func() any) any {
	return fn()
}

// ============================================================================
// VBox - Vertical container
// ============================================================================

type VBoxC struct {
	fill         Color
	inheritStyle *Style
	gap          int8
	border       BorderStyle
	borderFG     *Color
	borderBG     *Color
	title        string
	width        int16
	height       int16
	percentWidth float32
	flexGrow     float32
	fitContent   bool
	margin       [4]int16 // top, right, bottom, left
	children     []any
}

type VBoxFn func(children ...any) VBoxC

func (f VBoxFn) Fill(c Color) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.fill = c
		return v
	}
}

func (f VBoxFn) CascadeStyle(s *Style) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.inheritStyle = s
		return v
	}
}

func (f VBoxFn) Gap(g int8) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.gap = g
		return v
	}
}

func (f VBoxFn) Border(b BorderStyle) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.border = b
		return v
	}
}

func (f VBoxFn) BorderFG(c Color) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.borderFG = &c
		return v
	}
}

func (f VBoxFn) BorderBG(c Color) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.borderBG = &c
		return v
	}
}

func (f VBoxFn) Title(t string) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.title = t
		return v
	}
}

func (f VBoxFn) Width(w int16) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.width = w
		return v
	}
}

func (f VBoxFn) Height(h int16) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.height = h
		return v
	}
}

func (f VBoxFn) Size(w, h int16) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.width = w
		v.height = h
		return v
	}
}

func (f VBoxFn) WidthPct(pct float32) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.percentWidth = pct
		return v
	}
}

func (f VBoxFn) Grow(g float32) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.flexGrow = g
		return v
	}
}

func (f VBoxFn) FitContent() VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.fitContent = true
		return v
	}
}

func (f VBoxFn) Margin(all int16) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.margin = [4]int16{all, all, all, all}
		return v
	}
}

func (f VBoxFn) MarginXY(vertical, horizontal int16) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.margin = [4]int16{vertical, horizontal, vertical, horizontal}
		return v
	}
}

func (f VBoxFn) MarginTRBL(top, right, bottom, left int16) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.margin = [4]int16{top, right, bottom, left}
		return v
	}
}

// VBox is the vertical container constructor
var VBox VBoxFn = func(children ...any) VBoxC {
	return VBoxC{children: children}
}

// ============================================================================
// HBox - Horizontal container
// ============================================================================

type HBoxC struct {
	fill         Color
	inheritStyle *Style
	gap          int8
	border       BorderStyle
	borderFG     *Color
	borderBG     *Color
	title        string
	width        int16
	height       int16
	percentWidth float32
	flexGrow     float32
	fitContent   bool
	margin       [4]int16 // top, right, bottom, left
	children     []any
}

type HBoxFn func(children ...any) HBoxC

func (f HBoxFn) Fill(c Color) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.fill = c
		return h
	}
}

func (f HBoxFn) CascadeStyle(s *Style) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.inheritStyle = s
		return h
	}
}

func (f HBoxFn) Gap(g int8) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.gap = g
		return h
	}
}

func (f HBoxFn) Border(b BorderStyle) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.border = b
		return h
	}
}

func (f HBoxFn) BorderFG(c Color) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.borderFG = &c
		return h
	}
}

func (f HBoxFn) BorderBG(c Color) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.borderBG = &c
		return h
	}
}

func (f HBoxFn) Title(t string) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.title = t
		return h
	}
}

func (f HBoxFn) Width(w int16) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.width = w
		return h
	}
}

func (f HBoxFn) Height(h int16) HBoxFn {
	return func(children ...any) HBoxC {
		c := f(children...)
		c.height = h
		return c
	}
}

func (f HBoxFn) Size(w, h int16) HBoxFn {
	return func(children ...any) HBoxC {
		c := f(children...)
		c.width = w
		c.height = h
		return c
	}
}

func (f HBoxFn) WidthPct(pct float32) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.percentWidth = pct
		return h
	}
}

func (f HBoxFn) Grow(g float32) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.flexGrow = g
		return h
	}
}

func (f HBoxFn) FitContent() HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.fitContent = true
		return h
	}
}

func (f HBoxFn) Margin(all int16) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.margin = [4]int16{all, all, all, all}
		return h
	}
}

func (f HBoxFn) MarginXY(vertical, horizontal int16) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.margin = [4]int16{vertical, horizontal, vertical, horizontal}
		return h
	}
}

func (f HBoxFn) MarginTRBL(top, right, bottom, left int16) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.margin = [4]int16{top, right, bottom, left}
		return h
	}
}

// HBox is the horizontal container constructor
var HBox HBoxFn = func(children ...any) HBoxC {
	return HBoxC{children: children}
}

// ============================================================================
// Arrange - Container with custom layout function
// ============================================================================

// Arrange creates a container with a custom layout function.
// The layout function receives child sizes and available space, returns positions.
//
//	Arrange(Grid(3, 20, 5))(
//	    Text("A"), Text("B"), Text("C"),
//	    Text("D"), Text("E"), Text("F"),
//	)
func Arrange(layout LayoutFunc) func(children ...any) Box {
	return func(children ...any) Box {
		return Box{Layout: layout, Children: children}
	}
}

// ============================================================================
// Widget - Fully custom component
// ============================================================================

// Widget creates a fully custom component with explicit measure and render functions.
// Use this when you need complete control over sizing and drawing.
//
//	Widget(
//	    func(availW int16) (w, h int16) { return 20, 3 },
//	    func(buf *Buffer, x, y, w, h int16) {
//	        buf.WriteString(int(x), int(y), "Custom!", Style{})
//	    },
//	)
func Widget(
	measure func(availW int16) (w, h int16),
	render func(buf *Buffer, x, y, w, h int16),
) Custom {
	return Custom{Measure: measure, Render: render}
}

// ============================================================================
// Text - Text display
// ============================================================================

type TextC struct {
	content any // string or *string
	style   Style
	width   int16    // explicit width (0 = content-sized)
	margin  [4]int16 // top, right, bottom, left
}

func Text(content any) TextC {
	return TextC{content: content}
}

func (t TextC) Style(s Style) TextC {
	t.style = s
	return t
}

func (t TextC) FG(c Color) TextC {
	t.style.FG = c
	return t
}

func (t TextC) BG(c Color) TextC {
	t.style.BG = c
	return t
}

func (t TextC) Bold() TextC {
	t.style.Attr |= AttrBold
	return t
}

func (t TextC) Dim() TextC {
	t.style.Attr |= AttrDim
	return t
}

func (t TextC) Italic() TextC {
	t.style.Attr |= AttrItalic
	return t
}

func (t TextC) Underline() TextC {
	t.style.Attr |= AttrUnderline
	return t
}

func (t TextC) Inverse() TextC {
	t.style.Attr |= AttrInverse
	return t
}

func (t TextC) Strikethrough() TextC {
	t.style.Attr |= AttrStrikethrough
	return t
}

func (t TextC) Width(w int16) TextC {
	t.width = w
	return t
}

func (t TextC) Margin(all int16) TextC            { t.margin = [4]int16{all, all, all, all}; return t }
func (t TextC) MarginXY(v, h int16) TextC         { t.margin = [4]int16{v, h, v, h}; return t }
func (t TextC) MarginTRBL(a, b, c, d int16) TextC { t.margin = [4]int16{a, b, c, d}; return t }

// ============================================================================
// Spacer - Empty space
// ============================================================================

type SpacerC struct {
	width    int16
	height   int16
	char     rune
	style    Style
	flexGrow float32
	margin   [4]int16
}

func Space() SpacerC {
	return SpacerC{}
}

func SpaceH(h int16) SpacerC {
	return SpacerC{height: h}
}

func SpaceW(w int16) SpacerC {
	return SpacerC{width: w}
}

func (s SpacerC) Width(w int16) SpacerC {
	s.width = w
	return s
}

func (s SpacerC) Height(h int16) SpacerC {
	s.height = h
	return s
}

func (s SpacerC) Char(c rune) SpacerC {
	s.char = c
	return s
}

func (s SpacerC) Style(st Style) SpacerC {
	s.style = st
	return s
}

func (s SpacerC) Grow(g float32) SpacerC {
	s.flexGrow = g
	return s
}

func (s SpacerC) Margin(all int16) SpacerC            { s.margin = [4]int16{all, all, all, all}; return s }
func (s SpacerC) MarginXY(v, h int16) SpacerC         { s.margin = [4]int16{v, h, v, h}; return s }
func (s SpacerC) MarginTRBL(a, b, c, d int16) SpacerC { s.margin = [4]int16{a, b, c, d}; return s }

// ============================================================================
// HRule - Horizontal line
// ============================================================================

type HRuleC struct {
	char   rune
	style  Style
	margin [4]int16
}

func HRule() HRuleC {
	return HRuleC{char: '─'}
}

func (h HRuleC) Char(c rune) HRuleC {
	h.char = c
	return h
}

func (h HRuleC) Style(s Style) HRuleC {
	h.style = s
	return h
}

func (h HRuleC) Margin(all int16) HRuleC            { h.margin = [4]int16{all, all, all, all}; return h }
func (h HRuleC) MarginXY(v, hz int16) HRuleC        { h.margin = [4]int16{v, hz, v, hz}; return h }
func (h HRuleC) MarginTRBL(a, b, c, d int16) HRuleC { h.margin = [4]int16{a, b, c, d}; return h }

// ============================================================================
// VRule - Vertical line
// ============================================================================

type VRuleC struct {
	char   rune
	style  Style
	height int16
	margin [4]int16
}

func VRule() VRuleC {
	return VRuleC{char: '│'}
}

func (v VRuleC) Char(c rune) VRuleC {
	v.char = c
	return v
}

func (v VRuleC) Style(s Style) VRuleC {
	v.style = s
	return v
}

func (v VRuleC) Height(h int16) VRuleC {
	v.height = h
	return v
}

func (v VRuleC) Margin(all int16) VRuleC            { v.margin = [4]int16{all, all, all, all}; return v }
func (v VRuleC) MarginXY(vt, hz int16) VRuleC       { v.margin = [4]int16{vt, hz, vt, hz}; return v }
func (v VRuleC) MarginTRBL(a, b, c, d int16) VRuleC { v.margin = [4]int16{a, b, c, d}; return v }

// ============================================================================
// Progress - Progress bar
// ============================================================================

type ProgressC struct {
	value  any // int (0-100) or *int
	width  int16
	style  Style
	margin [4]int16
}

func Progress(value any) ProgressC {
	return ProgressC{value: value}
}

func (p ProgressC) Width(w int16) ProgressC {
	p.width = w
	return p
}

func (p ProgressC) Style(s Style) ProgressC {
	p.style = s
	return p
}

func (p ProgressC) Margin(all int16) ProgressC            { p.margin = [4]int16{all, all, all, all}; return p }
func (p ProgressC) MarginXY(v, h int16) ProgressC         { p.margin = [4]int16{v, h, v, h}; return p }
func (p ProgressC) MarginTRBL(a, b, c, d int16) ProgressC { p.margin = [4]int16{a, b, c, d}; return p }

// ============================================================================
// Spinner - Animated spinner
// ============================================================================

type SpinnerC struct {
	frame  *int
	frames []string
	style  Style
	margin [4]int16
}

func Spinner(frame *int) SpinnerC {
	return SpinnerC{frame: frame, frames: SpinnerBraille}
}

func (s SpinnerC) Frames(f []string) SpinnerC {
	s.frames = f
	return s
}

func (s SpinnerC) Style(st Style) SpinnerC {
	s.style = st
	return s
}

func (s SpinnerC) Margin(all int16) SpinnerC            { s.margin = [4]int16{all, all, all, all}; return s }
func (s SpinnerC) MarginXY(v, h int16) SpinnerC         { s.margin = [4]int16{v, h, v, h}; return s }
func (s SpinnerC) MarginTRBL(a, b, c, d int16) SpinnerC { s.margin = [4]int16{a, b, c, d}; return s }

// ============================================================================
// Leader - Label.....Value display
// ============================================================================

type LeaderC struct {
	label  any // string or *string
	value  any // string or *string
	width  int16
	fill   rune
	style  Style
	margin [4]int16
}

func Leader(label, value any) LeaderC {
	return LeaderC{label: label, value: value, fill: '.'}
}

func (l LeaderC) Width(w int16) LeaderC {
	l.width = w
	return l
}

func (l LeaderC) Fill(r rune) LeaderC {
	l.fill = r
	return l
}

func (l LeaderC) Style(s Style) LeaderC {
	l.style = s
	return l
}

func (l LeaderC) Margin(all int16) LeaderC            { l.margin = [4]int16{all, all, all, all}; return l }
func (l LeaderC) MarginXY(v, h int16) LeaderC         { l.margin = [4]int16{v, h, v, h}; return l }
func (l LeaderC) MarginTRBL(a, b, c, d int16) LeaderC { l.margin = [4]int16{a, b, c, d}; return l }

// ============================================================================
// Sparkline - Mini chart
// ============================================================================

type SparklineC struct {
	values any // []float64 or *[]float64
	width  int16
	min    float64
	max    float64
	style  Style
	margin [4]int16
}

func Sparkline(values any) SparklineC {
	return SparklineC{values: values}
}

func (s SparklineC) Width(w int16) SparklineC {
	s.width = w
	return s
}

func (s SparklineC) Range(min, max float64) SparklineC {
	s.min = min
	s.max = max
	return s
}

func (s SparklineC) Style(st Style) SparklineC {
	s.style = st
	return s
}

func (s SparklineC) Margin(all int16) SparklineC    { s.margin = [4]int16{all, all, all, all}; return s }
func (s SparklineC) MarginXY(v, h int16) SparklineC { s.margin = [4]int16{v, h, v, h}; return s }
func (s SparklineC) MarginTRBL(a, b, c, d int16) SparklineC {
	s.margin = [4]int16{a, b, c, d}
	return s
}

// ============================================================================
// Jump - Jumpable target wrapper
// ============================================================================

type JumpC struct {
	child    any
	onSelect func()
	style    Style
	margin   [4]int16
}

func Jump(child any, onSelect func()) JumpC {
	return JumpC{child: child, onSelect: onSelect}
}

func (j JumpC) Style(s Style) JumpC {
	j.style = s
	return j
}

func (j JumpC) Margin(all int16) JumpC            { j.margin = [4]int16{all, all, all, all}; return j }
func (j JumpC) MarginXY(v, h int16) JumpC         { j.margin = [4]int16{v, h, v, h}; return j }
func (j JumpC) MarginTRBL(a, b, c, d int16) JumpC { j.margin = [4]int16{a, b, c, d}; return j }

// ============================================================================
// LayerView - Display a pre-rendered layer
// ============================================================================

type LayerViewC struct {
	layer      *Layer
	viewHeight int16
	viewWidth  int16
	flexGrow   float32
	margin     [4]int16
}

func LayerView(layer *Layer) LayerViewC {
	return LayerViewC{layer: layer}
}

func (l LayerViewC) ViewHeight(h int16) LayerViewC {
	l.viewHeight = h
	return l
}

func (l LayerViewC) ViewWidth(w int16) LayerViewC {
	l.viewWidth = w
	return l
}

func (l LayerViewC) Grow(g float32) LayerViewC {
	l.flexGrow = g
	return l
}

func (l LayerViewC) Margin(all int16) LayerViewC    { l.margin = [4]int16{all, all, all, all}; return l }
func (l LayerViewC) MarginXY(v, h int16) LayerViewC { l.margin = [4]int16{v, h, v, h}; return l }
func (l LayerViewC) MarginTRBL(a, b, c, d int16) LayerViewC {
	l.margin = [4]int16{a, b, c, d}
	return l
}

// ============================================================================
// Overlay - Modal/popup overlay
// ============================================================================

type OverlayC struct {
	centered   bool
	backdrop   bool
	x, y       int
	width      int
	height     int
	backdropFG Color
	bg         Color
	children   []any
}

type OverlayFn func(children ...any) OverlayC

func (f OverlayFn) Centered() OverlayFn {
	return func(children ...any) OverlayC {
		o := f(children...)
		o.centered = true
		return o
	}
}

func (f OverlayFn) Backdrop() OverlayFn {
	return func(children ...any) OverlayC {
		o := f(children...)
		o.backdrop = true
		return o
	}
}

func (f OverlayFn) At(x, y int) OverlayFn {
	return func(children ...any) OverlayC {
		o := f(children...)
		o.x = x
		o.y = y
		return o
	}
}

func (f OverlayFn) Size(w, h int) OverlayFn {
	return func(children ...any) OverlayC {
		o := f(children...)
		o.width = w
		o.height = h
		return o
	}
}

func (f OverlayFn) BG(c Color) OverlayFn {
	return func(children ...any) OverlayC {
		o := f(children...)
		o.bg = c
		return o
	}
}

func (f OverlayFn) BackdropFG(c Color) OverlayFn {
	return func(children ...any) OverlayC {
		o := f(children...)
		o.backdropFG = c
		return o
	}
}

var Overlay OverlayFn = func(children ...any) OverlayC {
	return OverlayC{children: children}
}

// ============================================================================
// ForEach - List rendering
// ============================================================================

type ForEachC[T any] struct {
	items    *[]T
	template func(item *T) any
}

func ForEach[T any](items *[]T, template func(item *T) any) ForEachC[T] {
	return ForEachC[T]{items: items, template: template}
}

// compileTo implements forEachCompiler for template compilation
func (f ForEachC[T]) compileTo(t *Template, parent int16, depth int) int16 {
	return t.compileForEach(ForEachNode{Items: f.items, Render: f.template}, parent, depth)
}

// ============================================================================
// SelectionList - Navigable list with selection
// ============================================================================

type ListC[T any] struct {
	items            *[]T
	selected         *int
	internalSel      int // used when no external selection provided
	render           func(*T) any
	marker           string
	markerStyle      Style
	maxVisible       int
	style            Style
	selectedStyle    Style
	cached           *SelectionList // cached instance for consistent reference
	declaredBindings []binding
}

// List creates a selectable list with internal selection management.
// Use .Render() to provide custom item rendering.
func List[T any](items *[]T) *ListC[T] {
	l := &ListC[T]{
		items:  items,
		marker: "> ",
	}
	l.selected = &l.internalSel
	return l
}

func (l *ListC[T]) Ref(f func(*ListC[T])) *ListC[T] { f(l); return l }

// Selection binds the selection index to an external pointer.
func (l *ListC[T]) Selection(sel *int) *ListC[T] {
	l.selected = sel
	return l
}

// Selected returns a pointer to the currently selected item, or nil if empty.
func (l *ListC[T]) Selected() *T {
	if l.items == nil || len(*l.items) == 0 {
		return nil
	}
	idx := *l.selected
	if idx < 0 || idx >= len(*l.items) {
		return nil
	}
	return &(*l.items)[idx]
}

// Index returns the current selection index.
func (l *ListC[T]) Index() int {
	return *l.selected
}

// SetIndex sets the selection index directly.
func (l *ListC[T]) SetIndex(i int) {
	*l.selected = i
}

// ClampSelection ensures the selection index is within bounds.
func (l *ListC[T]) ClampSelection() {
	n := len(*l.items)
	if n == 0 {
		*l.selected = 0
		return
	}
	if *l.selected >= n {
		*l.selected = n - 1
	}
	if *l.selected < 0 {
		*l.selected = 0
	}
}

// Delete removes the currently selected item.
func (l *ListC[T]) Delete() {
	if l.items == nil || len(*l.items) == 0 {
		return
	}
	idx := *l.selected
	if idx < 0 || idx >= len(*l.items) {
		return
	}
	*l.items = append((*l.items)[:idx], (*l.items)[idx+1:]...)
	if *l.selected >= len(*l.items) && *l.selected > 0 {
		*l.selected--
	}
}

// Render sets a custom render function for each item.
func (l *ListC[T]) Render(fn func(*T) any) *ListC[T] {
	l.render = fn
	return l
}

// Marker sets the selection marker (default "> ").
func (l *ListC[T]) Marker(m string) *ListC[T] {
	l.marker = m
	return l
}

// MarkerStyle sets the style for the marker text.
func (l *ListC[T]) MarkerStyle(s Style) *ListC[T] {
	l.markerStyle = s
	return l
}

// MaxVisible sets the maximum visible items (0 = show all).
func (l *ListC[T]) MaxVisible(n int) *ListC[T] {
	l.maxVisible = n
	return l
}

// Style sets the default style for non-selected rows.
func (l *ListC[T]) Style(s Style) *ListC[T] {
	l.style = s
	return l
}

// SelectedStyle sets the style for the selected row.
func (l *ListC[T]) SelectedStyle(s Style) *ListC[T] {
	l.selectedStyle = s
	return l
}

// toSelectionList returns the internal SelectionList (creates on first call).
// Same instance is returned for both template compilation and method calls.
func (l *ListC[T]) toSelectionList() *SelectionList {
	if l.cached == nil {
		l.cached = &SelectionList{
			Items:         l.items,
			Selected:      l.selected,
			Render:        l.render,
			Marker:        l.marker,
			MarkerStyle:   l.markerStyle,
			MaxVisible:    l.maxVisible,
			Style:         l.style,
			SelectedStyle: l.selectedStyle,
		}
	}
	return l.cached
}

// Up moves selection up by one.
func (l *ListC[T]) Up(m any) { l.toSelectionList().Up(m) }

// Down moves selection down by one.
func (l *ListC[T]) Down(m any) { l.toSelectionList().Down(m) }

// PageUp moves selection up by page size.
func (l *ListC[T]) PageUp(m any) { l.toSelectionList().PageUp(m) }

// PageDown moves selection down by page size.
func (l *ListC[T]) PageDown(m any) { l.toSelectionList().PageDown(m) }

// First moves selection to first item.
func (l *ListC[T]) First(m any) { l.toSelectionList().First(m) }

// Last moves selection to last item.
func (l *ListC[T]) Last(m any) { l.toSelectionList().Last(m) }

func (l *ListC[T]) BindNav(down, up string) *ListC[T] {
	l.declaredBindings = append(l.declaredBindings,
		binding{pattern: down, handler: l.Down},
		binding{pattern: up, handler: l.Up},
	)
	return l
}

func (l *ListC[T]) BindPageNav(pageDown, pageUp string) *ListC[T] {
	l.declaredBindings = append(l.declaredBindings,
		binding{pattern: pageDown, handler: l.PageDown},
		binding{pattern: pageUp, handler: l.PageUp},
	)
	return l
}

func (l *ListC[T]) BindFirstLast(first, last string) *ListC[T] {
	l.declaredBindings = append(l.declaredBindings,
		binding{pattern: first, handler: l.First},
		binding{pattern: last, handler: l.Last},
	)
	return l
}

// BindVimNav wires the standard vim-style navigation keys:
// j/k for line movement, Ctrl-d/Ctrl-u for page, g/G for first/last.
func (l *ListC[T]) BindVimNav() *ListC[T] {
	return l.BindNav("j", "k").BindPageNav("<C-d>", "<C-u>").BindFirstLast("g", "G")
}

func (l *ListC[T]) BindDelete(key string) *ListC[T] {
	l.declaredBindings = append(l.declaredBindings,
		binding{pattern: key, handler: l.Delete},
	)
	return l
}

// Handle registers a key binding that passes the currently selected item
// to the callback. If nothing is selected, the callback is not called.
func (l *ListC[T]) Handle(key string, fn func(*T)) *ListC[T] {
	l.declaredBindings = append(l.declaredBindings,
		binding{pattern: key, handler: func() {
			if item := l.Selected(); item != nil {
				fn(item)
			}
		}},
	)
	return l
}

func (l *ListC[T]) bindings() []binding { return l.declaredBindings }

// ============================================================================
// Tabs - Tab headers
// ============================================================================

type TabsC struct {
	labels        []string
	selected      *int
	tabStyle      TabsStyle
	gap           int
	activeStyle   Style
	inactiveStyle Style
	margin        [4]int16
}

func Tabs(labels []string, selected *int) TabsC {
	return TabsC{labels: labels, selected: selected, gap: 2}
}

func (t TabsC) Style(s TabsStyle) TabsC {
	t.tabStyle = s
	return t
}

func (t TabsC) Gap(g int) TabsC {
	t.gap = g
	return t
}

func (t TabsC) ActiveStyle(s Style) TabsC {
	t.activeStyle = s
	return t
}

func (t TabsC) InactiveStyle(s Style) TabsC {
	t.inactiveStyle = s
	return t
}

func (t TabsC) Margin(all int16) TabsC            { t.margin = [4]int16{all, all, all, all}; return t }
func (t TabsC) MarginXY(v, h int16) TabsC         { t.margin = [4]int16{v, h, v, h}; return t }
func (t TabsC) MarginTRBL(a, b, c, d int16) TabsC { t.margin = [4]int16{a, b, c, d}; return t }

// ============================================================================
// Scrollbar
// ============================================================================

type ScrollbarC struct {
	contentSize int
	viewSize    int
	position    *int
	length      int16
	horizontal  bool
	trackChar   rune
	thumbChar   rune
	trackStyle  Style
	thumbStyle  Style
	margin      [4]int16
}

func Scroll(contentSize, viewSize int, position *int) ScrollbarC {
	return ScrollbarC{
		contentSize: contentSize,
		viewSize:    viewSize,
		position:    position,
		trackChar:   '│',
		thumbChar:   '█',
	}
}

func (s ScrollbarC) Length(l int16) ScrollbarC {
	s.length = l
	return s
}

func (s ScrollbarC) Horizontal() ScrollbarC {
	s.horizontal = true
	s.trackChar = '─'
	return s
}

func (s ScrollbarC) TrackChar(c rune) ScrollbarC {
	s.trackChar = c
	return s
}

func (s ScrollbarC) ThumbChar(c rune) ScrollbarC {
	s.thumbChar = c
	return s
}

func (s ScrollbarC) TrackStyle(st Style) ScrollbarC {
	s.trackStyle = st
	return s
}

func (s ScrollbarC) ThumbStyle(st Style) ScrollbarC {
	s.thumbStyle = st
	return s
}

func (s ScrollbarC) Margin(all int16) ScrollbarC    { s.margin = [4]int16{all, all, all, all}; return s }
func (s ScrollbarC) MarginXY(v, h int16) ScrollbarC { s.margin = [4]int16{v, h, v, h}; return s }
func (s ScrollbarC) MarginTRBL(a, b, c, d int16) ScrollbarC {
	s.margin = [4]int16{a, b, c, d}
	return s
}

// ============================================================================
// AutoTable - Automatic table from slice of structs
// ============================================================================

type AutoTableC struct {
	data        any      // slice of structs
	columns     []string // field names to display (nil = all exported)
	headers     []string // custom header names (parallel to columns)
	headerStyle Style
	rowStyle    Style
	altRowStyle *Style
	gap         int8
	border      BorderStyle
	margin      [4]int16
}

// AutoTable creates a table from a slice of structs.
// Pass a slice like []MyStruct or []*MyStruct.
func AutoTable(data any) AutoTableC {
	return AutoTableC{
		data:        data,
		headerStyle: Style{Attr: AttrBold},
		gap:         1,
	}
}

// Columns selects which struct fields to display and in what order.
// Field names are case-sensitive and must match exported struct fields.
func (t AutoTableC) Columns(names ...string) AutoTableC {
	t.columns = names
	return t
}

// Headers sets custom header labels for the columns.
// Must be called after Columns() and have the same number of entries.
func (t AutoTableC) Headers(names ...string) AutoTableC {
	t.headers = names
	return t
}

func (t AutoTableC) HeaderStyle(s Style) AutoTableC {
	t.headerStyle = s
	return t
}

func (t AutoTableC) RowStyle(s Style) AutoTableC {
	t.rowStyle = s
	return t
}

func (t AutoTableC) AltRowStyle(s Style) AutoTableC {
	t.altRowStyle = &s
	return t
}

func (t AutoTableC) Gap(g int8) AutoTableC {
	t.gap = g
	return t
}

func (t AutoTableC) Border(b BorderStyle) AutoTableC {
	t.border = b
	return t
}

func (t AutoTableC) Margin(all int16) AutoTableC    { t.margin = [4]int16{all, all, all, all}; return t }
func (t AutoTableC) MarginXY(v, h int16) AutoTableC { t.margin = [4]int16{v, h, v, h}; return t }
func (t AutoTableC) MarginTRBL(a, b, c, d int16) AutoTableC {
	t.margin = [4]int16{a, b, c, d}
	return t
}

// ============================================================================
// Form Components - Checkbox, Radio, CheckList, Input
// ============================================================================

// CheckboxC is a toggleable checkbox bound to a *bool.
type CheckboxC struct {
	checked          *bool
	label            string
	labelPtr         *string
	checkedMark      string
	unchecked        string
	style            Style
	declaredBindings []binding
}

// Checkbox creates a checkbox bound to a bool pointer.
func Checkbox(checked *bool, label string) *CheckboxC {
	return &CheckboxC{
		checked:     checked,
		label:       label,
		checkedMark: "☑",
		unchecked:   "☐",
	}
}

// CheckboxPtr creates a checkbox with a dynamic label.
func CheckboxPtr(checked *bool, label *string) *CheckboxC {
	return &CheckboxC{
		checked:     checked,
		labelPtr:    label,
		checkedMark: "☑",
		unchecked:   "☐",
	}
}

func (c *CheckboxC) Ref(f func(*CheckboxC)) *CheckboxC { f(c); return c }

func (c *CheckboxC) Marks(checked, unchecked string) *CheckboxC {
	c.checkedMark = checked
	c.unchecked = unchecked
	return c
}

func (c *CheckboxC) Style(s Style) *CheckboxC {
	c.style = s
	return c
}

func (c *CheckboxC) BindToggle(key string) *CheckboxC {
	c.declaredBindings = append(c.declaredBindings,
		binding{pattern: key, handler: c.Toggle},
	)
	return c
}

func (c *CheckboxC) bindings() []binding { return c.declaredBindings }

// Toggle flips the checked state.
func (c *CheckboxC) Toggle() {
	*c.checked = !*c.checked
}

// Checked returns the current state.
func (c *CheckboxC) Checked() bool {
	return *c.checked
}

// RadioC is a single-selection group bound to *int (selected index).
type RadioC struct {
	selected         *int
	options          []string
	optionsPtr       *[]string
	selectedMark     string
	unselected       string
	style            Style
	gap              int8
	horizontal       bool
	declaredBindings []binding
}

// Radio creates a radio group with static options.
func Radio(selected *int, options ...string) *RadioC {
	return &RadioC{
		selected:     selected,
		options:      options,
		selectedMark: "◉",
		unselected:   "○",
	}
}

// RadioPtr creates a radio group with dynamic options.
func RadioPtr(selected *int, options *[]string) *RadioC {
	return &RadioC{
		selected:     selected,
		optionsPtr:   options,
		selectedMark: "◉",
		unselected:   "○",
	}
}

func (r *RadioC) Ref(f func(*RadioC)) *RadioC { f(r); return r }

func (r *RadioC) Marks(selected, unselected string) *RadioC {
	r.selectedMark = selected
	r.unselected = unselected
	return r
}

func (r *RadioC) Style(s Style) *RadioC {
	r.style = s
	return r
}

func (r *RadioC) Gap(g int8) *RadioC {
	r.gap = g
	return r
}

func (r *RadioC) Horizontal() *RadioC {
	r.horizontal = true
	return r
}

func (r *RadioC) BindNav(next, prev string) *RadioC {
	r.declaredBindings = append(r.declaredBindings,
		binding{pattern: next, handler: func() { r.Next() }},
		binding{pattern: prev, handler: func() { r.Prev() }},
	)
	return r
}

func (r *RadioC) bindings() []binding { return r.declaredBindings }

// Next moves selection to next option.
func (r *RadioC) Next() {
	opts := r.getOptions()
	if *r.selected < len(opts)-1 {
		*r.selected++
	}
}

// Prev moves selection to previous option.
func (r *RadioC) Prev() {
	if *r.selected > 0 {
		*r.selected--
	}
}

// Selected returns the currently selected option text.
func (r *RadioC) Selected() string {
	opts := r.getOptions()
	if *r.selected >= 0 && *r.selected < len(opts) {
		return opts[*r.selected]
	}
	return ""
}

// Index returns the selected index.
func (r *RadioC) Index() int {
	return *r.selected
}

func (r *RadioC) getOptions() []string {
	if r.optionsPtr != nil {
		return *r.optionsPtr
	}
	return r.options
}

// CheckListC is a list with per-item checkboxes, similar to todo lists.
type CheckListC[T any] struct {
	items            *[]T
	checked          func(*T) *bool
	render           func(*T) any
	selected         *int
	internalSel      int
	checkedMark      string
	uncheckedMark    string
	marker           string
	markerStyle      Style
	style            Style
	selectedStyle    Style
	gap              int8
	declaredBindings []binding
	cached           *SelectionList
}

// CheckList creates a list where each item has a checkbox.
func CheckList[T any](items *[]T) *CheckListC[T] {
	c := &CheckListC[T]{
		items:         items,
		checkedMark:   "☑",
		uncheckedMark: "☐",
		marker:        "> ",
	}
	c.selected = &c.internalSel
	return c
}

// Checked sets the function to get the checked state for each item.
func (c *CheckListC[T]) Checked(fn func(*T) *bool) *CheckListC[T] {
	c.checked = fn
	return c
}

// Render sets a custom render function for item content (after the checkbox).
func (c *CheckListC[T]) Render(fn func(*T) any) *CheckListC[T] {
	c.render = fn
	return c
}

// Marks sets the checkbox characters.
func (c *CheckListC[T]) Marks(checked, unchecked string) *CheckListC[T] {
	c.checkedMark = checked
	c.uncheckedMark = unchecked
	return c
}

// Marker sets the selection indicator.
func (c *CheckListC[T]) Marker(m string) *CheckListC[T] {
	c.marker = m
	return c
}

func (c *CheckListC[T]) MarkerStyle(s Style) *CheckListC[T] {
	c.markerStyle = s
	return c
}

func (c *CheckListC[T]) Style(s Style) *CheckListC[T] {
	c.style = s
	return c
}

func (c *CheckListC[T]) SelectedStyle(s Style) *CheckListC[T] {
	c.selectedStyle = s
	return c
}

func (c *CheckListC[T]) Gap(g int8) *CheckListC[T] {
	c.gap = g
	return c
}

func (c *CheckListC[T]) BindNav(down, up string) *CheckListC[T] {
	c.declaredBindings = append(c.declaredBindings,
		binding{pattern: down, handler: c.Down},
		binding{pattern: up, handler: c.Up},
	)
	return c
}

func (c *CheckListC[T]) BindPageNav(pageDown, pageUp string) *CheckListC[T] {
	c.declaredBindings = append(c.declaredBindings,
		binding{pattern: pageDown, handler: c.PageDown},
		binding{pattern: pageUp, handler: c.PageUp},
	)
	return c
}

func (c *CheckListC[T]) BindFirstLast(first, last string) *CheckListC[T] {
	c.declaredBindings = append(c.declaredBindings,
		binding{pattern: first, handler: c.First},
		binding{pattern: last, handler: c.Last},
	)
	return c
}

// BindVimNav wires the standard vim-style navigation keys:
// j/k for line movement, Ctrl-d/Ctrl-u for page, g/G for first/last.
func (c *CheckListC[T]) BindVimNav() *CheckListC[T] {
	return c.BindNav("j", "k").BindPageNav("<C-d>", "<C-u>").BindFirstLast("g", "G")
}

func (c *CheckListC[T]) BindToggle(key string) *CheckListC[T] {
	c.declaredBindings = append(c.declaredBindings,
		binding{pattern: key, handler: func() {
			if c.checked != nil {
				if item := c.SelectedItem(); item != nil {
					ptr := c.checked(item)
					*ptr = !*ptr
				}
			}
		}},
	)
	return c
}

func (c *CheckListC[T]) BindDelete(key string) *CheckListC[T] {
	c.declaredBindings = append(c.declaredBindings,
		binding{pattern: key, handler: c.Delete},
	)
	return c
}

// Handle registers a key binding that passes the currently selected item
// to the callback. If nothing is selected, the callback is not called.
func (c *CheckListC[T]) Handle(key string, fn func(*T)) *CheckListC[T] {
	c.declaredBindings = append(c.declaredBindings,
		binding{pattern: key, handler: func() {
			if item := c.SelectedItem(); item != nil {
				fn(item)
			}
		}},
	)
	return c
}

func (c *CheckListC[T]) bindings() []binding { return c.declaredBindings }

func (c *CheckListC[T]) Ref(f func(*CheckListC[T])) *CheckListC[T] { f(c); return c }

// SelectedItem returns a pointer to the currently selected item.
func (c *CheckListC[T]) SelectedItem() *T {
	if c.items == nil || len(*c.items) == 0 {
		return nil
	}
	idx := *c.selected
	if idx < 0 || idx >= len(*c.items) {
		return nil
	}
	return &(*c.items)[idx]
}

// Index returns the current selection index.
func (c *CheckListC[T]) Index() int {
	return *c.selected
}

// Delete removes the currently selected item.
func (c *CheckListC[T]) Delete() {
	if c.items == nil || len(*c.items) == 0 {
		return
	}
	idx := *c.selected
	if idx < 0 || idx >= len(*c.items) {
		return
	}
	*c.items = append((*c.items)[:idx], (*c.items)[idx+1:]...)
	if *c.selected >= len(*c.items) && *c.selected > 0 {
		*c.selected--
	}
}

func (c *CheckListC[T]) Up(m any)       { c.toSelectionList().Up(m) }
func (c *CheckListC[T]) Down(m any)     { c.toSelectionList().Down(m) }
func (c *CheckListC[T]) PageUp(m any)   { c.toSelectionList().PageUp(m) }
func (c *CheckListC[T]) PageDown(m any) { c.toSelectionList().PageDown(m) }
func (c *CheckListC[T]) First(m any)    { c.toSelectionList().First(m) }
func (c *CheckListC[T]) Last(m any)     { c.toSelectionList().Last(m) }

func (c *CheckListC[T]) toSelectionList() *SelectionList {
	if c.cached == nil {
		// Start with explicit functions (may be nil)
		checkedFn := c.checked
		renderFn := c.render

		// Infer from struct tags if not explicitly set
		if checkedFn == nil || renderFn == nil {
			var sample T
			t := reflect.TypeOf(sample)
			if t.Kind() == reflect.Struct {
				for i := 0; i < t.NumField(); i++ {
					field := t.Field(i)
					tag := field.Tag.Get("forme")

					if tag == "checked" && field.Type.Kind() == reflect.Bool && checkedFn == nil {
						idx := i
						checkedFn = func(item *T) *bool {
							v := reflect.ValueOf(item).Elem().Field(idx)
							return v.Addr().Interface().(*bool)
						}
					}

					if tag == "render" && field.Type.Kind() == reflect.String && renderFn == nil {
						idx := i
						renderFn = func(item *T) any {
							v := reflect.ValueOf(item).Elem().Field(idx)
							return Text(v.Addr().Interface().(*string))
						}
					}
				}
			}
		}

		// Store inferred functions so BindToggle etc. can use them
		c.checked = checkedFn
		c.render = renderFn

		c.cached = &SelectionList{
			Items:         c.items,
			Selected:      c.selected,
			Marker:        c.marker,
			MarkerStyle:   c.markerStyle,
			Style:         c.style,
			SelectedStyle: c.selectedStyle,
		}

		// Build the render function with checkbox marks
		if checkedFn != nil && renderFn != nil {
			checkedMark := c.checkedMark
			uncheckedMark := c.uncheckedMark
			c.cached.Render = func(item *T) any {
				mark := If(checkedFn(item)).Then(Text(checkedMark)).Else(Text(uncheckedMark))
				return HBox.Gap(1)(mark, renderFn(item))
			}
		} else if checkedFn != nil {
			checkedMark := c.checkedMark
			uncheckedMark := c.uncheckedMark
			c.cached.Render = func(item *T) any {
				return If(checkedFn(item)).Then(Text(checkedMark)).Else(Text(uncheckedMark))
			}
		}
	}
	return c.cached
}

// InputC is a text input with internal state management.
type InputC struct {
	field       Field
	placeholder string
	width       int
	mask        rune
	style       Style
	declaredTIB *textInputBinding
}

// Input creates a text input with internal state.
func Input() *InputC {
	return &InputC{}
}

func (i *InputC) Ref(f func(*InputC)) *InputC { f(i); return i }

// Placeholder sets the placeholder text.
func (i *InputC) Placeholder(p string) *InputC {
	i.placeholder = p
	return i
}

// Width sets the input width.
func (i *InputC) Width(w int) *InputC {
	i.width = w
	return i
}

// Mask sets a password mask character.
func (i *InputC) Mask(m rune) *InputC {
	i.mask = m
	return i
}

func (i *InputC) Style(s Style) *InputC {
	i.style = s
	return i
}

func (i *InputC) Bind() *InputC {
	i.declaredTIB = &textInputBinding{
		value:  &i.field.Value,
		cursor: &i.field.Cursor,
	}
	return i
}

func (i *InputC) textBinding() *textInputBinding { return i.declaredTIB }

// Value returns the current text value.
func (i *InputC) Value() string {
	return i.field.Value
}

// SetValue sets the text value.
func (i *InputC) SetValue(v string) {
	i.field.Value = v
	i.field.Cursor = len(v)
}

// Clear resets the input.
func (i *InputC) Clear() {
	i.field.Clear()
}

// Field returns a pointer to the internal field (for TextInput compatibility).
func (i *InputC) Field() *Field {
	return &i.field
}

// toTextInput converts to the underlying TextInput for rendering.
func (i *InputC) toTextInput() TextInput {
	return TextInput{
		Field:       &i.field,
		Placeholder: i.placeholder,
		Width:       i.width,
		Mask:        i.mask,
		Style:       i.style,
	}
}
