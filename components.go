package tui

// ============================================================================
// Functional Component API
// ============================================================================
//
// Container components (VBox, HBox) use a function-type-with-methods pattern:
//   VBox(children...)                    - simple usage
//   VBox.Style(&s).Gap(2)(children...)   - with configuration
//
// Leaf components (Text, Spacer, etc.) use simple functions with method chaining:
//   Text("hello")                        - simple usage
//   Text("hello").Bold().FG(Red)         - with styling
//
// ============================================================================

// ============================================================================
// VBox - Vertical container
// ============================================================================

type VBoxC struct {
	style        *Style
	gap          int8
	border       BorderStyle
	borderFG     *Color
	borderBG     *Color
	title        string
	width        int16
	height       int16
	percentWidth float32
	flexGrow     float32
	children     []any
}

type VBoxFn func(children ...any) VBoxC

func (f VBoxFn) Style(s *Style) VBoxFn {
	return func(children ...any) VBoxC {
		v := f(children...)
		v.style = s
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

// VBox is the vertical container constructor
var VBox VBoxFn = func(children ...any) VBoxC {
	return VBoxC{children: children}
}

// ============================================================================
// HBox - Horizontal container
// ============================================================================

type HBoxC struct {
	style        *Style
	gap          int8
	border       BorderStyle
	borderFG     *Color
	borderBG     *Color
	title        string
	width        int16
	height       int16
	percentWidth float32
	flexGrow     float32
	children     []any
}

type HBoxFn func(children ...any) HBoxC

func (f HBoxFn) Style(s *Style) HBoxFn {
	return func(children ...any) HBoxC {
		h := f(children...)
		h.style = s
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

// HBox is the horizontal container constructor
var HBox HBoxFn = func(children ...any) HBoxC {
	return HBoxC{children: children}
}

// ============================================================================
// Text - Text display
// ============================================================================

type TextC struct {
	content any // string or *string
	style   Style
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

// ============================================================================
// Spacer - Empty space
// ============================================================================

type SpacerC struct {
	width    int16
	height   int16
	char     rune
	style    Style
	flexGrow float32
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

// ============================================================================
// HRule - Horizontal line
// ============================================================================

type HRuleC struct {
	char  rune
	style Style
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

// ============================================================================
// VRule - Vertical line
// ============================================================================

type VRuleC struct {
	char   rune
	style  Style
	height int16
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

// ============================================================================
// Progress - Progress bar
// ============================================================================

type ProgressC struct {
	value any   // int (0-100) or *int
	width int16
	style Style
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

// ============================================================================
// Spinner - Animated spinner
// ============================================================================

type SpinnerC struct {
	frame  *int
	frames []string
	style  Style
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

// ============================================================================
// Leader - Label.....Value display
// ============================================================================

type LeaderC struct {
	label any // string or *string
	value any // string or *string
	width int16
	fill  rune
	style Style
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

// ============================================================================
// Sparkline - Mini chart
// ============================================================================

type SparklineC struct {
	values any // []float64 or *[]float64
	width  int16
	min    float64
	max    float64
	style  Style
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

// ============================================================================
// Jump - Jumpable target wrapper
// ============================================================================

type JumpC struct {
	child    any
	onSelect func()
	style    Style
}

func Jump(child any, onSelect func()) JumpC {
	return JumpC{child: child, onSelect: onSelect}
}

func (j JumpC) Style(s Style) JumpC {
	j.style = s
	return j
}

// ============================================================================
// LayerView - Display a pre-rendered layer
// ============================================================================

type LayerViewC struct {
	layer      *Layer
	viewHeight int16
	viewWidth  int16
	flexGrow   float32
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
	items         *[]T
	selected      *int
	render        func(*T) any
	marker        string
	markerStyle   Style
	maxVisible    int
	style         Style
	selectedStyle Style
	cached        *SelectionList // cached instance for consistent reference
}

// List creates a selectable list with the given items and selection pointer.
// Use .Render() to provide custom item rendering.
func List[T any](items *[]T, selected *int) *ListC[T] {
	return &ListC[T]{
		items:    items,
		selected: selected,
		marker:   "> ",
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
