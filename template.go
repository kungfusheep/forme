package glyph

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

// Component is the extension interface for custom components.
// External packages can implement this to create custom components
// that expand to built-in primitives at compile time.
type Component interface {
	Build() any
}

// Renderer is the extension interface for components that render directly.
// Unlike Component (which expands to primitives), Renderer draws to the
// buffer itself. This is useful for custom widgets like charts, sparklines, etc.
type Renderer interface {
	// MinSize returns the minimum dimensions needed by this component.
	// Called during layout phase.
	MinSize() (width, height int)

	// Render draws the component to the buffer at the given position.
	// w and h are the allocated dimensions (may be larger than MinSize).
	Render(buf *Buffer, x, y, w, h int)
}

// forEachCompiler is implemented by generic ForEach types to compile themselves
type forEachCompiler interface {
	compileTo(t *Template, parent int16, depth int) int16
}

// listCompiler is implemented by generic List types to compile themselves
type listCompiler interface {
	toSelectionList() *SelectionList
}

// bindable is implemented by components that declare key bindings as data.
type bindable interface {
	bindings() []binding
}

// textInputBindable is implemented by InputC for text input routing.
type textInputBindable interface {
	textBinding() *textInputBinding
}

// templateTree is implemented by compound components that compose existing
// building blocks into a template subtree.
type templateTree interface {
	toTemplate() any
}

// LayoutFunc positions children given their sizes and available space.
type LayoutFunc func(children []ChildSize, availW, availH int) []Rect

// ChildSize represents a child's computed minimum dimensions.
type ChildSize struct {
	MinW, MinH int
}

// Rect represents a positioned rectangle.
type Rect struct {
	X, Y, W, H int
}

// NodeRef holds a node's rendered screen bounds, populated each frame after layout.
// Declare one, attach it to a node with .NodeRef(), then read it in effects or
// anywhere that needs to know where something actually rendered.
type NodeRef = Rect

// Box is a container with a custom layout function.
// Use this when HBox/VBox don't fit your needs.
type Box struct {
	Layout   LayoutFunc
	Children []any
}

// Template is a compiled UI template.
// Compile does all reflection. Execute is pure pointer arithmetic.
type Template struct {
	ops  []Op
	geom []Geom // parallel to ops, filled at runtime

	// For bottom-up layout traversal
	maxDepth int
	byDepth  [][]int16 // ops grouped by tree depth

	// Current element base for ForEach context (set during layout/render)
	elemBase unsafe.Pointer

	// App reference for jump mode coordination
	app *App

	// Row background for SelectionList selected rows (merged with cell styles)
	rowBG Color

	// Style inheritance - current inherited style during render
	inheritedStyle *Style
	inheritedFill  Color // cascades through nested containers

	// vertical clip: maximum Y coordinate for rendering (exclusive, 0 = no clip)
	clipMaxY int16

	// Pending overlays to render after main content (cleared each frame)
	pendingOverlays []pendingOverlay

	// Pending screen effects collected from tree (cleared each frame)
	pendingScreenEffects []Effect

	// scratch buffers for per-frame reuse (avoid nil-slice allocs in hot paths)
	flexScratchIdx  []int16   // flex child indices (shared by VBox + HBox phases)
	flexScratchGrow []float32 // flex grow values (shared by VBox + HBox phases)
	flexScratchImpl []int16   // implicit flex children (HBox only)
	treeScratchPfx  []bool    // tree node line prefix

	// ext pools — contiguous allocations for cache-friendly render access.

	// Declarative bindings collected during compile, wired during setup
	pendingBindings     []binding
	pendingTIB          *textInputBinding
	pendingLogs         []*LogC       // Logs that need app.RequestRender wiring
	pendingFocusManager *FocusManager // Focus manager for multi-input routing

	// per-frame evaluators — conditions, animations, etc. run at start of Execute
	evals []func()

	// frame timing — single timestamp per frame, shared by all animations
	frameTime time.Time
	animating bool

	// animation ticker — runs at ~60fps only while animations are active
	animTicker    *time.Ticker
	requestRender func()
}

// pendingOverlay stores info needed to render an overlay after main content
type pendingOverlay struct {
	op *Op // pointer to the overlay op
}

// SetApp links this template to an App for jump mode support.
func (t *Template) SetApp(a *App) {
	t.app = a
}

func (t *Template) collectBindings(node any) {
	if b, ok := node.(bindable); ok {
		t.pendingBindings = append(t.pendingBindings, b.bindings()...)
	}
}

func (t *Template) collectTextInputBinding(node any) {
	if tib, ok := node.(textInputBindable); ok {
		t.pendingTIB = tib.textBinding()
	}
}

func (t *Template) collectFocusManager(node any) {
	// check if InputC or FilterLogC has a manager
	switch v := node.(type) {
	case *InputC:
		if v.manager != nil && t.pendingFocusManager == nil {
			t.pendingFocusManager = v.manager
		}
	case *FilterLogC:
		if v.manager != nil && t.pendingFocusManager == nil {
			t.pendingFocusManager = v.manager
		}
	}
}

// Geom holds runtime geometry for an op.
// Filled during execute, parallel array to ops.
type Geom struct {
	W, H           int16 // dimensions
	LocalX, LocalY int16 // position relative to parent
	ContentH       int16 // natural content height (before flex distribution)
}

// Op represents a single compiled template instruction.
// The template compiler produces a flat array of these; Execute walks them to render.
type Op struct {
	Kind   OpKind
	Depth  int8  // tree depth (root children = 0)
	Parent int16 // parent op index, -1 for root children

	// Layout hints
	Width        int16   // explicit width
	Height       int16   // explicit height
	PercentWidth float32 // 0.0-1.0
	FlexGrow     float32 // share of remaining space
	Gap          int8    // gap between children
	ContentSized bool    // has fixed-width children (don't implicit flex)
	FitContent   bool    // size to content instead of filling available space

	// Container
	IsRow        bool        // true=HBox, false=VBox
	Border       BorderStyle // border style
	BorderFG     *Color      // border foreground color
	BorderBG     *Color      // border background color
	Title        string      // border title
	ChildStart   int16       // first child op index
	ChildEnd     int16       // last child op index (exclusive)
	CascadeStyle *Style      // style inherited by children (pointer for dynamic themes)
	Fill         Color       // container fill color (fills entire area)
	Margin       [4]int16    // outer margin: top, right, bottom, left
	NodeRef      *NodeRef    // if set, populated with rendered screen bounds each frame

	// kind-specific data — type-assert based on Kind.
	// we use a Kind switch + type assertion instead of interface dispatch because
	// concrete method calls after assertion are inlinable. interface calls are not,
	// and cause parameters to escape to heap. verified via go build -gcflags='-m -m'.
	Ext any

	// dynamic layout property overrides — nil for static ops
	Dyn *OpDyn
}

// OpDyn holds pointer overrides for shared layout properties.
// only allocated for ops that use dynamic values (e.g. Height(&h)).
type OpDyn struct {
	Height       *int16
	Width        *int16
	FlexGrow     *float32
	PercentWidth *float32
	Gap          *int8
}

// resolver methods — inlinable nil-check + deref, zero cost when Dyn is nil

func (op *Op) height() int16 {
	if op.Dyn != nil {
		if p := op.Dyn.Height; p != nil {
			return *p
		}
	}
	return op.Height
}

func (op *Op) width() int16 {
	if op.Dyn != nil {
		if p := op.Dyn.Width; p != nil {
			return *p
		}
	}
	return op.Width
}

func (op *Op) flexGrow() float32 {
	if op.Dyn != nil {
		if p := op.Dyn.FlexGrow; p != nil {
			return *p
		}
	}
	return op.FlexGrow
}

func (op *Op) percentWidth() float32 {
	if op.Dyn != nil {
		if p := op.Dyn.PercentWidth; p != nil {
			return *p
		}
	}
	return op.PercentWidth
}

func (op *Op) gap() int8 {
	if op.Dyn != nil {
		if p := op.Dyn.Gap; p != nil {
			return *p
		}
	}
	return op.Gap
}

// compileCond registers a conditional evaluator and returns a pointer to its storage.
// the evaluator runs each frame, resolving the condition and writing the active value.

func (t *Template) compileCondInt16(cond conditionNode) *int16 {
	storage := new(int16)
	thenVal := cond.getThen()
	elseVal := cond.getElse()
	eval := func() {
		if cond.evaluate() {
			*storage = anyToInt16(thenVal)
		} else {
			*storage = anyToInt16(elseVal)
		}
	}
	eval() // set initial value
	t.evals = append(t.evals, eval)
	return storage
}

func (t *Template) compileCondFloat32(cond conditionNode) *float32 {
	storage := new(float32)
	thenVal := cond.getThen()
	elseVal := cond.getElse()
	eval := func() {
		if cond.evaluate() {
			*storage = anyToFloat32(thenVal)
		} else {
			*storage = anyToFloat32(elseVal)
		}
	}
	eval()
	t.evals = append(t.evals, eval)
	return storage
}

func (t *Template) compileCondInt8(cond conditionNode) *int8 {
	storage := new(int8)
	thenVal := cond.getThen()
	elseVal := cond.getElse()
	eval := func() {
		if cond.evaluate() {
			*storage = anyToInt8(thenVal)
		} else {
			*storage = anyToInt8(elseVal)
		}
	}
	eval()
	t.evals = append(t.evals, eval)
	return storage
}

func anyToInt16(v any) int16 {
	switch val := v.(type) {
	case int16:
		return val
	case int:
		return int16(val)
	case *int16:
		return *val
	}
	return 0
}

func anyToFloat32(v any) float32 {
	switch val := v.(type) {
	case float32:
		return val
	case float64:
		return float32(val)
	case int:
		return float32(val)
	case *float32:
		return *val
	}
	return 0
}

func anyToInt8(v any) int8 {
	switch val := v.(type) {
	case int8:
		return val
	case int:
		return int8(val)
	case *int8:
		return *val
	}
	return 0
}

// compileDyn resolves a dynamic property value (conditionNode or tweenNode) to a pointer.
// used by compile sites where Cond fields can hold either type.

func (t *Template) compileDynInt16(v any) *int16 {
	switch c := v.(type) {
	case conditionNode:
		return t.compileCondInt16(c)
	case tweenNode:
		return t.compileTweenInt16(c)
	}
	return nil
}

func (t *Template) compileDynFloat32(v any) *float32 {
	switch c := v.(type) {
	case conditionNode:
		return t.compileCondFloat32(c)
	case tweenNode:
		return t.compileTweenFloat32(c)
	}
	return nil
}

func (t *Template) compileDynInt8(v any) *int8 {
	switch c := v.(type) {
	case conditionNode:
		return t.compileCondInt8(c)
	case tweenNode:
		return t.compileTweenInt8(c)
	}
	return nil
}

// Animating returns true if any tween is currently in progress.
// check this after Execute to determine if another frame is needed.
func (t *Template) Animating() bool { return t.animating }

// compileTween resolves a tweenNode's target to a typed pointer, allocates
// interpolation storage, and registers a per-frame evaluator that watches the
// target and lerps toward it. all tweens in a frame share t.frameTime.

func (t *Template) compileTweenInt16(tw tweenNode) *int16 {
	watchPtr := t.resolveTweenTargetInt16(tw.getTarget())
	storage := new(int16)
	*storage = *watchPtr
	dur := tw.getTweenDuration()
	ease := tw.getTweenEasing()

	lastTarget := *watchPtr
	startVal := float64(*watchPtr)
	var startTime time.Time

	t.evals = append(t.evals, func() {
		target := *watchPtr
		now := t.frameTime
		if target != lastTarget {
			startVal = float64(*storage)
			lastTarget = target
			startTime = now
		}
		if startTime.IsZero() {
			return
		}
		elapsed := now.Sub(startTime)
		if elapsed >= dur {
			*storage = target
			startTime = time.Time{}
			return
		}
		progress := float64(elapsed) / float64(dur)
		if ease != nil {
			progress = ease(progress)
		}
		*storage = int16(startVal + progress*(float64(target)-startVal))
		t.animating = true
	})
	return storage
}

func (t *Template) compileTweenFloat32(tw tweenNode) *float32 {
	watchPtr := t.resolveTweenTargetFloat32(tw.getTarget())
	storage := new(float32)
	*storage = *watchPtr
	dur := tw.getTweenDuration()
	ease := tw.getTweenEasing()

	lastTarget := *watchPtr
	startVal := float64(*watchPtr)
	var startTime time.Time

	t.evals = append(t.evals, func() {
		target := *watchPtr
		now := t.frameTime
		if target != lastTarget {
			startVal = float64(*storage)
			lastTarget = target
			startTime = now
		}
		if startTime.IsZero() {
			return
		}
		elapsed := now.Sub(startTime)
		if elapsed >= dur {
			*storage = target
			startTime = time.Time{}
			return
		}
		progress := float64(elapsed) / float64(dur)
		if ease != nil {
			progress = ease(progress)
		}
		*storage = float32(startVal + progress*(float64(target)-startVal))
		t.animating = true
	})
	return storage
}

func (t *Template) compileTweenInt8(tw tweenNode) *int8 {
	watchPtr := t.resolveTweenTargetInt8(tw.getTarget())
	storage := new(int8)
	*storage = *watchPtr
	dur := tw.getTweenDuration()
	ease := tw.getTweenEasing()

	lastTarget := *watchPtr
	startVal := float64(*watchPtr)
	var startTime time.Time

	t.evals = append(t.evals, func() {
		target := *watchPtr
		now := t.frameTime
		if target != lastTarget {
			startVal = float64(*storage)
			lastTarget = target
			startTime = now
		}
		if startTime.IsZero() {
			return
		}
		elapsed := now.Sub(startTime)
		if elapsed >= dur {
			*storage = target
			startTime = time.Time{}
			return
		}
		progress := float64(elapsed) / float64(dur)
		if ease != nil {
			progress = ease(progress)
		}
		*storage = int8(startVal + progress*(float64(target)-startVal))
		t.animating = true
	})
	return storage
}

// resolve tween targets — unwrap conditionNode or pointer, same as properties
func (t *Template) resolveTweenTargetInt16(target any) *int16 {
	switch v := target.(type) {
	case *int16:
		return v
	case conditionNode:
		return t.compileCondInt16(v)
	}
	// static fallback: allocate storage with the value
	storage := new(int16)
	*storage = anyToInt16(target)
	return storage
}

func (t *Template) resolveTweenTargetFloat32(target any) *float32 {
	switch v := target.(type) {
	case *float32:
		return v
	case conditionNode:
		return t.compileCondFloat32(v)
	}
	storage := new(float32)
	*storage = anyToFloat32(target)
	return storage
}

func (t *Template) resolveTweenTargetInt8(target any) *int8 {
	switch v := target.(type) {
	case *int8:
		return v
	case conditionNode:
		return t.compileCondInt8(v)
	}
	storage := new(int8)
	*storage = anyToInt8(target)
	return storage
}

// opSparkline holds sparkline-specific data.
type opSparkline struct {
	values    []float64
	valuesPtr *[]float64
	min       float64
	max       float64
	style     Style
}

func (s *opSparkline) resolveValues() []float64 {
	if s.valuesPtr != nil {
		return *s.valuesPtr
	}
	return s.values
}

func (s *opSparkline) render(t *Template, buf *Buffer, x, y, w, h int16) {
	style := t.effectiveStyle(s.style)
	vals := s.resolveValues()
	if len(vals) == 0 {
		return
	}
	if h <= 1 {
		buf.WriteSparkline(int(x), int(y), vals, int(w), s.min, s.max, style)
	} else {
		buf.WriteSparklineMulti(int(x), int(y), vals, int(w), int(h), s.min, s.max, style)
	}
}

func (s *opSparkline) dataLen() int {
	if s.valuesPtr != nil {
		return len(*s.valuesPtr)
	}
	return len(s.values)
}

// text variant modes
const (
	textStatic uint8 = iota
	textPtr
	textOff
	textFn
)

type opIf struct {
	condPtr  *bool
	condNode conditionNode
	thenTmpl *Template
	elseTmpl *Template
}

func (c *opIf) eval(elemBase unsafe.Pointer) bool {
	return (c.condPtr != nil && *c.condPtr) ||
		(c.condNode != nil && c.condNode.evaluateWithBase(elemBase))
}

func (c *opIf) evalStatic() bool {
	return (c.condPtr != nil && *c.condPtr) ||
		(c.condNode != nil && c.condNode.evaluate())
}

type opForEach struct {
	iterTmpl *Template
	slicePtr unsafe.Pointer
	elemSize uintptr
	geoms    []Geom // per-item geometry, reused across frames
}

type opSwitch struct {
	node     switchNodeInterface
	cases    []*Template
	def      *Template
}

type opCustomRenderer struct {
	renderer Renderer
}

type opCustomLayout struct {
	layout LayoutFunc
}

type opText struct {
	mode   uint8
	static string
	ptr    *string
	off    uintptr
	fn     func() string
	style  Style
}

func (tx *opText) resolve(elemBase unsafe.Pointer) string {
	switch tx.mode {
	case textPtr:
		return *tx.ptr
	case textOff:
		return *(*string)(unsafe.Pointer(uintptr(elemBase) + tx.off))
	default:
		return tx.static
	}
}

func (tx *opText) textWidth(elemBase unsafe.Pointer) int16 {
	switch tx.mode {
	case textPtr:
		return int16(utf8.RuneCountInString(*tx.ptr))
	case textOff:
		if elemBase != nil {
			return int16(utf8.RuneCountInString(*(*string)(unsafe.Pointer(uintptr(elemBase) + tx.off))))
		}
		return 10
	case textFn:
		if tx.fn != nil {
			return int16(utf8.RuneCountInString(tx.fn()))
		}
		return 0
	default:
		return int16(utf8.RuneCountInString(tx.static))
	}
}

// progress variant modes
const (
	progStatic uint8 = iota
	progPtr
	progOff
)

type opProgress struct {
	mode  uint8
	static int
	ptr    *int
	off    uintptr
	style  Style
}

func (p *opProgress) resolve(elemBase unsafe.Pointer) int {
	switch p.mode {
	case progPtr:
		return *p.ptr
	case progOff:
		return *(*int)(unsafe.Pointer(uintptr(elemBase) + p.off))
	default:
		return p.static
	}
}

// richtext variant modes
const (
	richStatic uint8 = iota
	richPtr
	richOff
)

type opRichText struct {
	mode        uint8
	staticSpans []Span
	spansPtr    *[]Span
	off         uintptr
	spanStrOffs []uintptr
}

func (rt *opRichText) resolve(elemBase unsafe.Pointer) []Span {
	var spans []Span
	switch rt.mode {
	case richPtr:
		spans = *rt.spansPtr
	case richOff:
		if elemBase == nil {
			return nil
		}
		spans = *(*[]Span)(unsafe.Pointer(uintptr(elemBase) + rt.off))
	default:
		spans = rt.staticSpans
	}
	if rt.spanStrOffs != nil {
		return resolveSpanStrs(spans, rt.spanStrOffs, elemBase)
	}
	return spans
}

// leader variant modes
const (
	leaderStatic uint8 = iota
	leaderPtr
	leaderIntPtr
	leaderFloatPtr
)

type opLeader struct {
	mode     uint8
	label    string
	value    string
	valuePtr *string
	intPtr   *int
	floatPtr *float64
	fill     rune
	style    Style
}

type opCounter struct {
	currentPtr   *int
	totalPtr     *int
	prefix       string
	streamingPtr *bool
	framePtr     *int32
	style        Style
}

type opSpinner struct {
	framePtr *int
	frames   []string
	style    Style
}

type opRule struct {
	char        rune
	style       Style
	extend      bool
	vruleX      int16
	vruleX2     int16
	extendTop   bool
	extendBot   bool
	extendLeft  int16
	extendRight int16
}

type opScrollbar struct {
	contentSize int
	viewSize    int
	posPtr      *int
	horizontal  bool
	trackChar   rune
	thumbChar   rune
	trackStyle  Style
	thumbStyle  Style
}

type opTabs struct {
	labels        []string
	selectedPtr   *int
	styleType     TabsStyle
	gap           int
	activeStyle   Style
	inactiveStyle Style
}

type opTreeView struct {
	root          *TreeNode
	showRoot      bool
	indent        int
	showLines     bool
	expandedChar  rune
	collapsedChar rune
	leafChar      rune
	style         Style
}

type opSelectionList struct {
	opForEach
	listPtr      *SelectionList
	selectedPtr  *int
	marker       string
	markerWidth  int16
	markerSpaces string
}

type opTextInput struct {
	fieldPtr       *InputState
	focusGroupPtr  *FocusGroup
	focusIndex     int
	valuePtr       *string
	cursorPtr      *int
	focusedPtr     *bool
	placeholder    string
	mask           rune
	style          Style
	placeholderSty Style
	cursorStyle    Style
}

type opOverlay struct {
	centered   bool
	x, y       int16
	backdrop   bool
	backdropFG Color
	bg         Color
	childTmpl  *Template
}

type opTable struct {
	columns     []TableColumn
	rowsPtr     *[][]string
	showHeader  bool
	headerStyle Style
	rowStyle    Style
	altStyle    Style
}

type opAutoTable struct {
	slicePtr  any
	fields    []int
	headers   []string
	hdrStyle  Style
	rowStyle  Style
	altStyle  *Style
	gap       int8
	fill      Color
	colCfgs   []*ColumnConfig
	sort      *autoTableSortState
	scroll    *autoTableScroll
}

type opLayer struct {
	ptr    *Layer
	width  int16
	height int16
}

type opJump struct {
	onSelect func()
	style    Style
}

type opScreenEffect struct {
	fns []Effect
}

// margin helpers (avoid repeating [0]/[1]/[2]/[3] everywhere)
func (op *Op) marginH() int16 { return op.Margin[1] + op.Margin[3] } // left + right
func (op *Op) marginV() int16 { return op.Margin[0] + op.Margin[2] } // top + bottom

// OpKind identifies the type of a compiled template instruction.
type OpKind uint8

const (
	OpText     OpKind = iota // Text (data in Ext)
	OpProgress               // Progress bar (data in Ext)
	OpRichText               // RichText (data in Ext)
	OpLeader                 // Leader dots (data in Ext)
	OpCounter                // Counter (data in Ext)

	OpContainer // VBox or HBox (determined by IsRow)

	OpIf
	OpForEach
	OpSwitch

	OpCustom // Custom renderer
	OpLayout // Custom layout
	OpLayer  // LayerView (data in Ext)

	OpSelectionList // SelectionList (data in Ext)

	OpTable     // Table (data in Ext)
	OpAutoTable // AutoTable (data in Ext)

	OpSparkline // Sparkline (data in Ext)

	OpHRule        // Horizontal line (data in Ext)
	OpVRule        // Vertical line (data in Ext)
	OpSpacer       // Empty space (data in Ext)
	OpSpinner      // Animated spinner (data in Ext)
	OpScrollbar    // Scroll indicator (data in Ext)
	OpTabs         // Tab headers (data in Ext)
	OpTreeView     // Hierarchical tree (data in Ext)
	OpJump         // Jump target wrapper (data in Ext)
	OpTextInput    // Single-line text input (data in Ext)
	OpOverlay      // Floating overlay/modal (data in Ext)
	OpScreenEffect // Full-screen post-processing effect (data in Ext)
)

// Build compiles a declarative UI tree into a Template ready for Execute.
// All reflection happens here at compile time; Execute is pure pointer reads.
func Build(ui any) *Template {
	t := &Template{
		ops:     make([]Op, 0, 32),
		byDepth: make([][]int16, 16),
	}

	for i := range t.byDepth {
		t.byDepth[i] = make([]int16, 0, 8)
	}

	t.compile(ui, -1, 0, nil, 0)

	// Trim unused depths
	for t.maxDepth >= 0 && len(t.byDepth[t.maxDepth]) == 0 {
		t.maxDepth--
	}
	if t.maxDepth >= 0 {
		t.byDepth = t.byDepth[:t.maxDepth+1]
	}

	// Pre-allocate geometry array
	t.geom = make([]Geom, len(t.ops))

	return t
}

func (t *Template) addOp(op Op, depth int) int16 {
	idx := int16(len(t.ops))
	op.Depth = int8(depth)
	t.ops = append(t.ops, op)

	// Track by depth for bottom-up traversal
	if depth >= 0 {
		if depth >= len(t.byDepth) {
			for len(t.byDepth) <= depth {
				t.byDepth = append(t.byDepth, make([]int16, 0, 8))
			}
		}
		t.byDepth[depth] = append(t.byDepth[depth], idx)
		if depth > t.maxDepth {
			t.maxDepth = depth
		}
	}

	return idx
}

func (t *Template) compile(node any, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	if node == nil {
		return -1
	}

	switch v := node.(type) {
	case Renderer:
		return t.compileRenderer(v, parent, depth)
	case Box:
		return t.compileBox(v, parent, depth, elemBase, elemSize)
	case conditionNode:
		return t.compileCondition(v, parent, depth, elemBase, elemSize)
	case RichTextNode:
		return t.compileRichText(v, parent, depth, elemBase, elemSize)
	case SelectionList:
		return t.compileSelectionList(&v, parent, depth, elemBase, elemSize)
	case *SelectionList:
		return t.compileSelectionList(v, parent, depth, elemBase, elemSize)
	case Table:
		return t.compileTable(v, parent, depth)
	case TabsNode:
		return t.compileTabs(v, parent, depth)
	case TreeView:
		return t.compileTreeView(v, parent, depth)
	case TextInput:
		return t.compileTextInput(v, parent, depth)
	case OverlayNode:
		return t.compileOverlay(v, parent, depth)
	case ScreenEffectNode:
		ext := &opScreenEffect{fns: v.Effects}
		return t.addOp(Op{Kind: OpScreenEffect, Parent: parent, Ext: ext}, depth)
	case Component:
		return t.compile(v.Build(), parent, depth, elemBase, elemSize)

	case VBoxC:
		return t.compileVBoxC(v, parent, depth, elemBase, elemSize)
	case HBoxC:
		return t.compileHBoxC(v, parent, depth, elemBase, elemSize)
	case TextC:
		return t.compileTextC(v, parent, depth, elemBase, elemSize)
	case SpacerC:
		return t.compileSpacerC(v, parent, depth)
	case HRuleC:
		return t.compileHRuleC(v, parent, depth)
	case VRuleC:
		return t.compileVRuleC(v, parent, depth)
	case ProgressC:
		return t.compileProgressC(v, parent, depth, elemBase, elemSize)
	case SpinnerC:
		return t.compileSpinnerC(v, parent, depth)
	case LeaderC:
		return t.compileLeaderC(v, parent, depth)
	case counterC:
		return t.compileCounterC(v, parent, depth)
	case SparklineC:
		return t.compileSparklineC(v, parent, depth)
	case JumpC:
		return t.compileJumpC(v, parent, depth, elemBase, elemSize)
	case LayerViewC:
		return t.compileLayerViewC(v, parent, depth)
	case OverlayC:
		return t.compileOverlayC(v, parent, depth)
	case TabsC:
		return t.compileTabsC(v, parent, depth)
	case ScrollbarC:
		return t.compileScrollbarC(v, parent, depth)
	case AutoTableC:
		t.collectBindings(v)
		return t.compileAutoTableC(v, parent, depth)
	case *CheckboxC:
		t.collectBindings(v)
		return t.compileCheckboxC(v, parent, depth, elemBase)
	case *RadioC:
		t.collectBindings(v)
		return t.compileRadioC(v, parent, depth)
	case *InputC:
		t.collectTextInputBinding(v)
		t.collectFocusManager(v)
		return t.compileInputC(v, parent, depth)
	case *LogC:
		t.collectBindings(v)
		return t.compileLogC(v, parent, depth)
	case *TextViewC:
		t.collectBindings(v)
		return t.compileTextViewC(v, parent, depth)
	case *FilterLogC:
		t.collectFocusManager(v)
		return t.compileFilterLogC(v, parent, depth)
	case Custom:
		return t.compileCustom(v, parent, depth)
	}

	// Check for ForEachC[T] via interface
	if fe, ok := node.(forEachCompiler); ok {
		return fe.compileTo(t, parent, depth)
	}

	// Check for compound components that produce a template subtree
	if tc, ok := node.(templateTree); ok {
		t.collectBindings(node)
		t.collectTextInputBinding(node)
		return t.compile(tc.toTemplate(), parent, depth, elemBase, elemSize)
	}

	// Check for ListC[T] or CheckListC[T] via interface
	// Both implement toSelectionList() which sets up their render functions appropriately
	if lc, ok := node.(listCompiler); ok {
		t.collectBindings(node)
		return t.compileSelectionList(lc.toSelectionList(), parent, depth, elemBase, elemSize)
	}

	// Check for SwitchNodeInterface (generic Switch)
	if sw, ok := node.(switchNodeInterface); ok {
		return t.compileSwitch(sw, parent, depth, elemBase, elemSize)
	}

	return -1
}

func (t *Template) compileRenderer(r Renderer, parent int16, depth int) int16 {
	return t.addOp(Op{
		Kind:   OpCustom,
		Parent: parent,
		Ext:    &opCustomRenderer{renderer: r},
	}, depth)
}

// customWrapper adapts the Custom struct to the Renderer interface
type customWrapper struct {
	measure func(availW int16) (w, h int16)
	render  func(buf *Buffer, x, y, w, h int16)
}

func (c *customWrapper) MinSize() (width, height int) {
	if c.measure == nil {
		return 0, 0
	}
	// Pass -1 to signal "fill available" - widget should return desired minimum
	// or pass back -1 to indicate it wants to fill
	w, h := c.measure(-1)
	if w < 0 {
		w = 0 // will be expanded by parent layout
	}
	return int(w), int(h)
}

// MeasureWithAvail calls measure with actual available width
func (c *customWrapper) MeasureWithAvail(availW int16) (w, h int16) {
	if c.measure == nil {
		return 0, 0
	}
	w, h = c.measure(availW)
	if w < 0 {
		w = availW
	}
	return w, h
}

func (c *customWrapper) Render(buf *Buffer, x, y, w, h int) {
	if c.render != nil {
		c.render(buf, int16(x), int16(y), int16(w), int16(h))
	}
}

func (t *Template) compileCustom(v Custom, parent int16, depth int) int16 {
	wrapper := &customWrapper{
		measure: v.Measure,
		render:  v.Render,
	}
	return t.addOp(Op{
		Kind:   OpCustom,
		Parent: parent,
		Ext:    &opCustomRenderer{renderer: wrapper},
	}, depth)
}

func (t *Template) compileBox(box Box, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	// Add layout op first (will fill in ChildStart/ChildEnd)
	idx := t.addOp(Op{
		Kind:       OpLayout,
		Parent:     parent,
		Ext:        &opCustomLayout{layout: box.Layout},
		ChildStart: int16(len(t.ops)),
	}, depth)

	// Compile children
	for _, child := range box.Children {
		t.compile(child, idx, depth+1, elemBase, elemSize)
	}

	// Set child end
	t.ops[idx].ChildEnd = int16(len(t.ops))

	return idx
}

func (t *Template) compileRichText(v RichTextNode, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	ext := &opRichText{}

	switch spans := v.Spans.(type) {
	case []Span:
		ext.mode = richStatic
		ext.staticSpans = spans
	case *[]Span:
		if elemBase != nil && isWithinRange(unsafe.Pointer(spans), elemBase, elemSize) {
			ext.mode = richOff
			ext.off = uintptr(unsafe.Pointer(spans)) - uintptr(elemBase)
		} else {
			ext.mode = richPtr
			ext.spansPtr = spans
		}
	default:
		ext.mode = richStatic
	}

	// compute per-span *string offsets for Textf
	if v.spanPtrs != nil && elemBase != nil {
		noOffset := ^uintptr(0)
		offs := make([]uintptr, len(v.spanPtrs))
		for i, ptr := range v.spanPtrs {
			if ptr != nil && isWithinRange(unsafe.Pointer(ptr), elemBase, elemSize) {
				offs[i] = uintptr(unsafe.Pointer(ptr)) - uintptr(elemBase)
			} else {
				offs[i] = noOffset
			}
		}
		ext.spanStrOffs = offs
	} else if v.spanPtrs != nil {
		noOffset := ^uintptr(0)
		offs := make([]uintptr, len(v.spanPtrs))
		for i, ptr := range v.spanPtrs {
			if ptr != nil {
				offs[i] = uintptr(unsafe.Pointer(ptr))
			} else {
				offs[i] = noOffset
			}
		}
		ext.spanStrOffs = offs
	}

	return t.addOp(Op{
		Kind:   OpRichText,
		Parent: parent,
		Ext:    ext,
	}, depth)
}

// resolveSpanStrs returns a copy of spans with dynamic *string values re-read.
// elemBase is the ForEach element pointer (nil when outside ForEach).
// When elemBase is nil, offs[i] stores the raw uintptr of the *string.
// When elemBase is non-nil, offs[i] stores the offset from elemBase.
// ^uintptr(0) sentinel means that span's text is static.
func resolveSpanStrs(spans []Span, offs []uintptr, elemBase unsafe.Pointer) []Span {
	noOffset := ^uintptr(0)
	resolved := make([]Span, len(spans))
	copy(resolved, spans)
	for i, off := range offs {
		if off == noOffset {
			continue
		}
		var ptr *string
		if elemBase != nil {
			ptr = (*string)(unsafe.Pointer(uintptr(elemBase) + off))
		} else {
			ptr = (*string)(unsafe.Pointer(off))
		}
		resolved[i].Text = *ptr
	}
	return resolved
}

func (t *Template) compileSelectionList(v *SelectionList, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	// Analyze slice using reflection
	sliceRV := reflect.ValueOf(v.Items)
	if sliceRV.Kind() != reflect.Ptr {
		panic("SelectionList Items must be pointer to slice")
	}
	sliceType := sliceRV.Type().Elem()
	if sliceType.Kind() != reflect.Slice {
		panic("SelectionList Items must be pointer to slice")
	}
	elemType := sliceType.Elem()
	sliceElemSize := elemType.Size()
	slicePtr := unsafe.Pointer(sliceRV.Pointer())

	// Default marker
	marker := v.Marker
	if marker == "" {
		marker = "> "
	}
	markerWidth := int16(utf8.RuneCountInString(marker))

	// Create iteration template if Render function provided
	var iterTmpl *Template
	if v.Render != nil && !reflect.ValueOf(v.Render).IsNil() {
		renderRV := reflect.ValueOf(v.Render)
		takesPtr := renderRV.Type().In(0).Kind() == reflect.Ptr

		var dummyElem reflect.Value
		var dummyBase unsafe.Pointer
		if takesPtr {
			dummyElem = reflect.New(elemType)
			dummyBase = unsafe.Pointer(dummyElem.Pointer())
		} else {
			dummyElem = reflect.New(elemType).Elem()
			dummyBase = unsafe.Pointer(dummyElem.Addr().Pointer())
		}

		// Call render to get template structure
		templateResult := renderRV.Call([]reflect.Value{dummyElem})[0].Interface()

		// Compile iteration template
		iterTmpl = &Template{
			ops:     make([]Op, 0, 16),
			byDepth: make([][]int16, 8),
		}
		for i := range iterTmpl.byDepth {
			iterTmpl.byDepth[i] = make([]int16, 0, 4)
		}
		iterTmpl.compile(templateResult, -1, 0, dummyBase, sliceElemSize)
		if iterTmpl.maxDepth >= 0 {
			iterTmpl.byDepth = iterTmpl.byDepth[:iterTmpl.maxDepth+1]
		}
		iterTmpl.geom = make([]Geom, len(iterTmpl.ops))
	}

	ext := &opSelectionList{
		listPtr:      v,
		selectedPtr:  v.Selected,
		marker:       marker,
		markerWidth:  markerWidth,
		markerSpaces: strings.Repeat(" ", int(markerWidth)),
	}

	ext.opForEach = opForEach{
		iterTmpl: iterTmpl,
		slicePtr: slicePtr,
		elemSize: sliceElemSize,
	}
	op := Op{
		Kind:   OpSelectionList,
		Parent: parent,
		Margin: v.Style.margin,
		Ext:    ext,
	}

	return t.addOp(op, depth)
}

func (t *Template) compileTable(v Table, parent int16, depth int) int16 {
	var rowsPtr *[][]string
	switch rows := v.Rows.(type) {
	case *[][]string:
		rowsPtr = rows
	case [][]string:
		rowsPtr = &rows
	}

	ext := &opTable{
		columns:     v.Columns,
		rowsPtr:     rowsPtr,
		showHeader:  v.ShowHeader,
		headerStyle: v.HeaderStyle,
		rowStyle:    v.RowStyle,
		altStyle:    v.AltRowStyle,
	}

	return t.addOp(Op{
		Kind:   OpTable,
		Parent: parent,
		Ext:    ext,
	}, depth)
}

func (t *Template) compileTabs(v TabsNode, parent int16, depth int) int16 {
	gap := v.Gap
	if gap == 0 {
		gap = 2
	}
	ext := &opTabs{
		labels:        v.Labels,
		selectedPtr:   v.Selected,
		styleType:     v.Style,
		gap:           gap,
		activeStyle:   v.ActiveStyle,
		inactiveStyle: v.InactiveStyle,
	}
	return t.addOp(Op{
		Kind:   OpTabs,
		Parent: parent,
		Ext:    ext,
	}, depth)
}

func (t *Template) compileTreeView(v TreeView, parent int16, depth int) int16 {
	indent := v.Indent
	if indent == 0 {
		indent = 2
	}
	expandedChar := v.ExpandedChar
	if expandedChar == 0 {
		expandedChar = '▼'
	}
	collapsedChar := v.CollapsedChar
	if collapsedChar == 0 {
		collapsedChar = '▶'
	}
	leafChar := v.LeafChar
	if leafChar == 0 {
		leafChar = ' '
	}
	ext := &opTreeView{
		root:          v.Root,
		showRoot:      v.ShowRoot,
		indent:        indent,
		showLines:     v.ShowLines,
		expandedChar:  expandedChar,
		collapsedChar: collapsedChar,
		leafChar:      leafChar,
		style:         v.Style,
	}
	return t.addOp(Op{
		Kind:   OpTreeView,
		Parent: parent,
		Ext:    ext,
	}, depth)
}

func (t *Template) compileTextInput(v TextInput, parent int16, depth int) int16 {
	ext := &opTextInput{
		fieldPtr:       v.Field,
		focusGroupPtr:  v.FocusGroup,
		focusIndex:     v.FocusIndex,
		valuePtr:       v.Value,
		cursorPtr:      v.Cursor,
		focusedPtr:     v.Focused,
		placeholder:    v.Placeholder,
		mask:           v.Mask,
		style:          v.Style,
		placeholderSty: v.PlaceholderStyle,
		cursorStyle:    v.CursorStyle,
	}

	if ext.placeholderSty.Equal(Style{}) {
		ext.placeholderSty = Style{Attr: AttrDim}
	}
	if ext.cursorStyle.Equal(Style{}) {
		ext.cursorStyle = Style{Attr: AttrInverse}
	}

	return t.addOp(Op{
		Kind:   OpTextInput,
		Parent: parent,
		Width:  int16(v.Width),
		Margin: v.Style.margin,
		Ext:    ext,
	}, depth)
}

func (t *Template) compileOverlay(v OverlayNode, parent int16, depth int) int16 {
	var childTmpl *Template
	if v.Child != nil {
		childTmpl = Build(v.Child)
	}

	centered := v.Centered || (v.X == 0 && v.Y == 0)

	backdropFG := v.BackdropFG
	if backdropFG.Mode == ColorDefault && v.Backdrop {
		backdropFG = BrightBlack
	}

	ext := &opOverlay{
		centered:   centered,
		x:          int16(v.X),
		y:          int16(v.Y),
		backdrop:   v.Backdrop,
		backdropFG: backdropFG,
		bg:         v.BG,
		childTmpl:  childTmpl,
	}

	return t.addOp(Op{
		Kind:   OpOverlay,
		Parent: parent,
		Width:  int16(v.Width),
		Height: int16(v.Height),
		Ext:    ext,
	}, depth)
}

func (t *Template) compileContainer(children []any, gap int8, isRow bool, f flex, border BorderStyle, title string, borderFG, borderBG *Color, fill Color, inheritStyle *Style, margin [4]int16, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	op := Op{
		Kind:         OpContainer,
		Parent:       parent,
		IsRow:        isRow,
		Gap:          gap,
		PercentWidth: f.percentWidth,
		Width:        f.width,
		Height:       f.height,
		FlexGrow:     f.flexGrow,
		FitContent:   f.fitContent,
		Border:       border,
		Title:        title,
		BorderFG:     borderFG,
		BorderBG:     borderBG,
		Fill:         fill,
		CascadeStyle: inheritStyle,
		Margin:       margin,
	}

	if f.widthPtr != nil || f.heightPtr != nil || f.flexGrowPtr != nil || f.percentWidthPtr != nil {
		op.Dyn = &OpDyn{
			Width:        f.widthPtr,
			Height:       f.heightPtr,
			FlexGrow:     f.flexGrowPtr,
			PercentWidth: f.percentWidthPtr,
		}
	}

	idx := t.addOp(op, depth)

	// Track child range
	childStart := int16(len(t.ops))
	for _, child := range children {
		t.compile(child, idx, depth+1, elemBase, elemSize)
	}
	childEnd := int16(len(t.ops))

	// Update op with child range
	t.ops[idx].ChildStart = childStart
	t.ops[idx].ChildEnd = childEnd

	// Check if any direct child has explicit fixed width (bubble up for layout decisions)
	// Only explicit Width counts - not content-based width like text
	for i := childStart; i < childEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Parent != idx {
			continue
		}
		if childOp.Width > 0 || (childOp.Dyn != nil && childOp.Dyn.Width != nil) || childOp.ContentSized {
			t.ops[idx].ContentSized = true
			break
		}
	}

	return idx
}

func (t *Template) compileCondition(cond conditionNode, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	// Check if condition pointer is within element range (ForEach context)
	if elemBase != nil && elemSize > 0 {
		ptrAddr := cond.getPtrAddr()
		baseAddr := uintptr(elemBase)
		if ptrAddr >= baseAddr && ptrAddr < baseAddr+elemSize {
			// Set offset for rebinding during render
			cond.setOffset(ptrAddr - baseAddr)
		}
	}

	ext := &opIf{
		condNode: cond,
	}

	// Compile then branch as sub-template
	if cond.getThen() != nil {
		thenTmpl := &Template{
			ops:     make([]Op, 0, 16),
			byDepth: make([][]int16, 8),
		}
		for i := range thenTmpl.byDepth {
			thenTmpl.byDepth[i] = make([]int16, 0, 4)
		}
		thenTmpl.compile(cond.getThen(), -1, 0, elemBase, elemSize)
		if thenTmpl.maxDepth >= 0 {
			thenTmpl.byDepth = thenTmpl.byDepth[:thenTmpl.maxDepth+1]
		}
		thenTmpl.geom = make([]Geom, len(thenTmpl.ops))
		ext.thenTmpl = thenTmpl
		t.pendingBindings = append(t.pendingBindings, thenTmpl.pendingBindings...)
	}

	// Compile else branch if present
	if cond.getElse() != nil {
		elseTmpl := &Template{
			ops:     make([]Op, 0, 16),
			byDepth: make([][]int16, 8),
		}
		for i := range elseTmpl.byDepth {
			elseTmpl.byDepth[i] = make([]int16, 0, 4)
		}
		elseTmpl.compile(cond.getElse(), -1, 0, elemBase, elemSize)
		if elseTmpl.maxDepth >= 0 {
			elseTmpl.byDepth = elseTmpl.byDepth[:elseTmpl.maxDepth+1]
		}
		elseTmpl.geom = make([]Geom, len(elseTmpl.ops))
		ext.elseTmpl = elseTmpl
		t.pendingBindings = append(t.pendingBindings, elseTmpl.pendingBindings...)
	}

	return t.addOp(Op{
		Kind:   OpIf,
		Parent: parent,
		Ext:    ext,
	}, depth)
}

func (t *Template) compileSwitch(sw switchNodeInterface, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	// record offset from element base so getMatchIndexWithBase works inside ForEach
	if elemBase != nil && elemSize > 0 {
		ptrAddr := sw.getPtrAddr()
		baseAddr := uintptr(elemBase)
		if ptrAddr >= baseAddr && ptrAddr < baseAddr+elemSize {
			sw.setPtrOffset(ptrAddr - baseAddr)
		}
	}

	ext := &opSwitch{
		node: sw,
	}

	// Compile each case branch
	caseNodes := sw.getCaseNodes()
	ext.cases = make([]*Template, len(caseNodes))
	for i, caseNode := range caseNodes {
		if caseNode != nil {
			caseTmpl := &Template{
				ops:     make([]Op, 0, 16),
				byDepth: make([][]int16, 8),
			}
			for j := range caseTmpl.byDepth {
				caseTmpl.byDepth[j] = make([]int16, 0, 4)
			}
			caseTmpl.compile(caseNode, -1, 0, elemBase, elemSize)
			if caseTmpl.maxDepth >= 0 {
				caseTmpl.byDepth = caseTmpl.byDepth[:caseTmpl.maxDepth+1]
			}
			caseTmpl.geom = make([]Geom, len(caseTmpl.ops))
			ext.cases[i] = caseTmpl
			t.pendingBindings = append(t.pendingBindings, caseTmpl.pendingBindings...)
		}
	}

	// Compile default branch
	if defNode := sw.getDefaultNode(); defNode != nil {
		defTmpl := &Template{
			ops:     make([]Op, 0, 16),
			byDepth: make([][]int16, 8),
		}
		for i := range defTmpl.byDepth {
			defTmpl.byDepth[i] = make([]int16, 0, 4)
		}
		defTmpl.compile(defNode, -1, 0, elemBase, elemSize)
		if defTmpl.maxDepth >= 0 {
			defTmpl.byDepth = defTmpl.byDepth[:defTmpl.maxDepth+1]
		}
		defTmpl.geom = make([]Geom, len(defTmpl.ops))
		ext.def = defTmpl
		t.pendingBindings = append(t.pendingBindings, defTmpl.pendingBindings...)
	}

	return t.addOp(Op{
		Kind:   OpSwitch,
		Parent: parent,
		Ext:    ext,
	}, depth)
}

func (t *Template) compileForEach(items any, render any, parent int16, depth int) int16 {
	// Analyze slice
	sliceRV := reflect.ValueOf(items)
	if sliceRV.Kind() != reflect.Ptr {
		panic("ForEach Items must be pointer to slice")
	}
	sliceType := sliceRV.Type().Elem()
	if sliceType.Kind() != reflect.Slice {
		panic("ForEach Items must be pointer to slice")
	}
	elemType := sliceType.Elem()
	elemSize := elemType.Size()
	slicePtr := unsafe.Pointer(sliceRV.Pointer())

	// Create dummy element for template compilation
	renderRV := reflect.ValueOf(render)
	takesPtr := renderRV.Type().In(0).Kind() == reflect.Ptr

	var dummyElem reflect.Value
	var dummyBase unsafe.Pointer
	if takesPtr {
		dummyElem = reflect.New(elemType)
		dummyBase = unsafe.Pointer(dummyElem.Pointer())
	} else {
		dummyElem = reflect.New(elemType).Elem()
		dummyBase = unsafe.Pointer(dummyElem.Addr().Pointer())
	}

	// Call render to get template structure
	templateResult := renderRV.Call([]reflect.Value{dummyElem})[0].Interface()

	// Compile iteration template
	iterTmpl := &Template{
		ops:     make([]Op, 0, 16),
		byDepth: make([][]int16, 8),
	}
	for i := range iterTmpl.byDepth {
		iterTmpl.byDepth[i] = make([]int16, 0, 4)
	}
	iterTmpl.compile(templateResult, -1, 0, dummyBase, elemSize)
	if iterTmpl.maxDepth >= 0 {
		iterTmpl.byDepth = iterTmpl.byDepth[:iterTmpl.maxDepth+1]
	}
	iterTmpl.geom = make([]Geom, len(iterTmpl.ops))

	op := Op{
		Kind:   OpForEach,
		Parent: parent,
		Ext: &opForEach{
			iterTmpl: iterTmpl,
			slicePtr: slicePtr,
			elemSize: elemSize,
		},
	}

	return t.addOp(op, depth)
}

// ============================================================================
// Compile functions for new functional API types
// ============================================================================

func (t *Template) compileVBoxC(v VBoxC, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	f := flex{percentWidth: v.percentWidth, width: v.width, height: v.height, flexGrow: v.flexGrow, fitContent: v.fitContent, widthPtr: v.widthPtr, heightPtr: v.heightPtr, percentWidthPtr: v.percentWidthPtr, flexGrowPtr: v.flexGrowPtr}
	if v.heightCond != nil {
		f.heightPtr = t.compileDynInt16(v.heightCond)
	}
	if v.widthCond != nil {
		f.widthPtr = t.compileDynInt16(v.widthCond)
	}
	if v.percentWidthCond != nil {
		f.percentWidthPtr = t.compileDynFloat32(v.percentWidthCond)
	}
	if v.flexGrowCond != nil {
		f.flexGrowPtr = t.compileDynFloat32(v.flexGrowCond)
	}
	idx := t.compileContainer(
		v.children,
		v.gap,
		false, // isRow
		f,
		v.border,
		v.title,
		v.borderFG,
		v.borderBG,
		v.fill,
		v.inheritStyle,
		v.margin,
		parent,
		depth,
		elemBase,
		elemSize,
	)
	if v.nodeRef != nil {
		t.ops[idx].NodeRef = v.nodeRef
	}
	if v.gapPtr != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Gap = v.gapPtr
	}
	if v.gapCond != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Gap = t.compileDynInt8(v.gapCond)
	}
	return idx
}

func (t *Template) compileHBoxC(v HBoxC, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	f := flex{percentWidth: v.percentWidth, width: v.width, height: v.height, flexGrow: v.flexGrow, fitContent: v.fitContent, widthPtr: v.widthPtr, heightPtr: v.heightPtr, percentWidthPtr: v.percentWidthPtr, flexGrowPtr: v.flexGrowPtr}
	if v.heightCond != nil {
		f.heightPtr = t.compileDynInt16(v.heightCond)
	}
	if v.widthCond != nil {
		f.widthPtr = t.compileDynInt16(v.widthCond)
	}
	if v.percentWidthCond != nil {
		f.percentWidthPtr = t.compileDynFloat32(v.percentWidthCond)
	}
	if v.flexGrowCond != nil {
		f.flexGrowPtr = t.compileDynFloat32(v.flexGrowCond)
	}
	idx := t.compileContainer(
		v.children,
		v.gap,
		true, // isRow
		f,
		v.border,
		v.title,
		v.borderFG,
		v.borderBG,
		v.fill,
		v.inheritStyle,
		v.margin,
		parent,
		depth,
		elemBase,
		elemSize,
	)
	if v.nodeRef != nil {
		t.ops[idx].NodeRef = v.nodeRef
	}
	if v.gapPtr != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Gap = v.gapPtr
	}
	if v.gapCond != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Gap = t.compileDynInt8(v.gapCond)
	}
	return idx
}

func (t *Template) compileTextC(v TextC, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	ext := &opText{style: v.style}

	switch val := v.content.(type) {
	case string:
		ext.mode = textStatic
		ext.static = val
	case *string:
		if elemBase != nil && isWithinRange(unsafe.Pointer(val), elemBase, elemSize) {
			ext.mode = textOff
			ext.off = uintptr(unsafe.Pointer(val)) - uintptr(elemBase)
		} else {
			ext.mode = textPtr
			ext.ptr = val
		}
	case func() string:
		ext.mode = textFn
		ext.fn = val
	}

	idx := t.addOp(Op{
		Kind:   OpText,
		Parent: parent,
		Width:  v.width,
		Margin: v.style.margin,
		Ext:    ext,
	}, depth)
	if v.widthCond != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Width = t.compileDynInt16(v.widthCond)
	} else if v.widthPtr != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Width = v.widthPtr
	}
	return idx
}

func (t *Template) compileSpacerC(v SpacerC, parent int16, depth int) int16 {
	grow := v.flexGrow
	if grow == 0 && v.width == 0 && v.height == 0 && v.widthPtr == nil && v.heightPtr == nil && v.flexGrowPtr == nil && v.widthCond == nil && v.heightCond == nil && v.flexGrowCond == nil {
		grow = 1
	}
	ext := &opRule{char: v.char, style: v.style}
	idx := t.addOp(Op{
		Kind:     OpSpacer,
		Parent:   parent,
		Width:    v.width,
		Height:   v.height,
		FlexGrow: grow,
		Margin:   v.style.margin,
		Ext:      ext,
	}, depth)
	hasDyn := v.widthPtr != nil || v.heightPtr != nil || v.flexGrowPtr != nil || v.widthCond != nil || v.heightCond != nil || v.flexGrowCond != nil
	if hasDyn {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		if v.widthCond != nil {
			t.ops[idx].Dyn.Width = t.compileDynInt16(v.widthCond)
		} else if v.widthPtr != nil {
			t.ops[idx].Dyn.Width = v.widthPtr
		}
		if v.heightCond != nil {
			t.ops[idx].Dyn.Height = t.compileDynInt16(v.heightCond)
		} else if v.heightPtr != nil {
			t.ops[idx].Dyn.Height = v.heightPtr
		}
		if v.flexGrowCond != nil {
			t.ops[idx].Dyn.FlexGrow = t.compileDynFloat32(v.flexGrowCond)
		} else if v.flexGrowPtr != nil {
			t.ops[idx].Dyn.FlexGrow = v.flexGrowPtr
		}
	}
	return idx
}

func (t *Template) compileHRuleC(v HRuleC, parent int16, depth int) int16 {
	char := v.char
	if char == 0 {
		char = '─'
	}
	ext := &opRule{char: char, style: v.style, extend: v.extend}
	return t.addOp(Op{
		Kind:   OpHRule,
		Parent: parent,
		Margin: v.style.margin,
		Ext:    ext,
	}, depth)
}

func (t *Template) compileVRuleC(v VRuleC, parent int16, depth int) int16 {
	char := v.char
	if char == 0 {
		char = '│'
	}
	ext := &opRule{char: char, style: v.style, extend: v.extend}
	idx := t.addOp(Op{
		Kind:   OpVRule,
		Parent: parent,
		Height: v.height,
		Margin: v.style.margin,
		Ext:    ext,
	}, depth)
	if v.heightCond != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Height = t.compileDynInt16(v.heightCond)
	} else if v.heightPtr != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Height = v.heightPtr
	}
	return idx
}

func (t *Template) compileProgressC(v ProgressC, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	width := v.width
	if width == 0 {
		width = 20
	}

	ext := &opProgress{style: v.style}

	switch val := v.value.(type) {
	case int:
		ext.mode = progStatic
		ext.static = val
	case *int:
		if elemBase != nil && isWithinRange(unsafe.Pointer(val), elemBase, elemSize) {
			ext.mode = progOff
			ext.off = uintptr(unsafe.Pointer(val)) - uintptr(elemBase)
		} else {
			ext.mode = progPtr
			ext.ptr = val
		}
	}

	idx := t.addOp(Op{
		Kind:   OpProgress,
		Parent: parent,
		Width:  width,
		Margin: v.style.margin,
		Ext:    ext,
	}, depth)
	if v.widthCond != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Width = t.compileDynInt16(v.widthCond)
	} else if v.widthPtr != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Width = v.widthPtr
	}
	return idx
}

func (t *Template) compileSpinnerC(v SpinnerC, parent int16, depth int) int16 {
	frames := v.frames
	if frames == nil {
		frames = SpinnerBraille
	}
	ext := &opSpinner{framePtr: v.frame, frames: frames, style: v.style}
	return t.addOp(Op{
		Kind:   OpSpinner,
		Parent: parent,
		Margin: v.style.margin,
		Ext:    ext,
	}, depth)
}

func (t *Template) compileLeaderC(v LeaderC, parent int16, depth int) int16 {
	fill := v.fill
	if fill == 0 {
		fill = '.'
	}

	ext := &opLeader{fill: fill, style: v.style}

	switch label := v.label.(type) {
	case string:
		ext.label = label
	case *string:
		ext.label = *label
	}

	switch val := v.value.(type) {
	case string:
		ext.mode = leaderStatic
		ext.value = val
	case *string:
		ext.mode = leaderPtr
		ext.valuePtr = val
	case *int:
		ext.mode = leaderIntPtr
		ext.intPtr = val
	case *float64:
		ext.mode = leaderFloatPtr
		ext.floatPtr = val
	case int:
		ext.mode = leaderStatic
		ext.value = fmt.Sprintf("%d", val)
	case float64:
		ext.mode = leaderStatic
		ext.value = fmt.Sprintf("%.1f", val)
	default:
		ext.mode = leaderStatic
		ext.value = fmt.Sprintf("%v", val)
	}

	idx := t.addOp(Op{
		Kind:   OpLeader,
		Parent: parent,
		Width:  v.width,
		Margin: v.style.margin,
		Ext:    ext,
	}, depth)
	if v.widthCond != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Width = t.compileDynInt16(v.widthCond)
	} else if v.widthPtr != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Width = v.widthPtr
	}
	return idx
}

func (t *Template) compileCounterC(v counterC, parent int16, depth int) int16 {
	ext := &opCounter{
		currentPtr:   v.current,
		totalPtr:     v.total,
		prefix:       v.prefix,
		streamingPtr: v.streaming,
		framePtr:     v.framePtr,
		style:        v.style,
	}
	return t.addOp(Op{
		Kind:   OpCounter,
		Parent: parent,
		Margin: v.style.margin,
		Ext:    ext,
	}, depth)
}

func (t *Template) compileSparklineC(v SparklineC, parent int16, depth int) int16 {
	ext := &opSparkline{min: v.min, max: v.max, style: v.style}
	switch vals := v.values.(type) {
	case []float64:
		ext.values = vals
	case *[]float64:
		ext.valuesPtr = vals
	}

	op := Op{
		Kind:   OpSparkline,
		Parent: parent,
		Width:  v.width,
		Height: v.height,
		Margin: v.style.margin,
		Ext:    ext,
	}
	idx := t.addOp(op, depth)
	hasDyn := v.widthPtr != nil || v.heightPtr != nil || v.widthCond != nil || v.heightCond != nil
	if hasDyn {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		if v.widthCond != nil {
			t.ops[idx].Dyn.Width = t.compileDynInt16(v.widthCond)
		} else if v.widthPtr != nil {
			t.ops[idx].Dyn.Width = v.widthPtr
		}
		if v.heightCond != nil {
			t.ops[idx].Dyn.Height = t.compileDynInt16(v.heightCond)
		} else if v.heightPtr != nil {
			t.ops[idx].Dyn.Height = v.heightPtr
		}
	}
	return idx
}

func (t *Template) compileJumpC(v JumpC, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	ext := &opJump{onSelect: v.onSelect, style: v.style}
	idx := t.addOp(Op{
		Kind:       OpJump,
		Parent:     parent,
		ChildStart: int16(len(t.ops)),
		Margin:     v.margin,
		Ext:        ext,
	}, depth)

	if v.child != nil {
		t.compile(v.child, idx, depth+1, elemBase, elemSize)
	}

	t.ops[idx].ChildEnd = int16(len(t.ops))
	return idx
}

func (t *Template) compileLayerViewC(v LayerViewC, parent int16, depth int) int16 {
	ext := &opLayer{ptr: v.layer, width: v.viewWidth, height: v.viewHeight}
	idx := t.addOp(Op{
		Kind:     OpLayer,
		Parent:   parent,
		FlexGrow: v.flexGrow,
		Margin:   v.margin,
		Ext:      ext,
	}, depth)
	if v.flexGrowCond != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.FlexGrow = t.compileDynFloat32(v.flexGrowCond)
	} else if v.flexGrowPtr != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.FlexGrow = v.flexGrowPtr
	}
	return idx
}

func (t *Template) compileOverlayC(v OverlayC, parent int16, depth int) int16 {
	var childTmpl *Template
	if len(v.children) == 1 {
		childTmpl = Build(v.children[0])
	} else if len(v.children) > 1 {
		childTmpl = Build(VBox(v.children...))
	}

	centered := v.centered || (v.x == 0 && v.y == 0)

	backdropFG := v.backdropFG
	if backdropFG.Mode == ColorDefault && v.backdrop {
		backdropFG = BrightBlack
	}

	ext := &opOverlay{
		centered:   centered,
		x:          int16(v.x),
		y:          int16(v.y),
		backdrop:   v.backdrop,
		backdropFG: backdropFG,
		bg:         v.bg,
		childTmpl:  childTmpl,
	}

	return t.addOp(Op{
		Kind:   OpOverlay,
		Parent: parent,
		Width:  int16(v.width),
		Height: int16(v.height),
		Ext:    ext,
	}, depth)
}

func (t *Template) compileTabsC(v TabsC, parent int16, depth int) int16 {
	ext := &opTabs{
		labels:        v.labels,
		selectedPtr:   v.selected,
		styleType:     v.tabStyle,
		gap:           int(v.gap),
		activeStyle:   v.activeStyle,
		inactiveStyle: v.inactiveStyle,
	}
	idx := t.addOp(Op{
		Kind:   OpTabs,
		Parent: parent,
		Gap:    v.gap,
		Margin: v.margin,
		Ext:    ext,
	}, depth)
	if v.gapCond != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Gap = t.compileDynInt8(v.gapCond)
	} else if v.gapPtr != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Gap = v.gapPtr
	}
	return idx
}

func (t *Template) compileScrollbarC(v ScrollbarC, parent int16, depth int) int16 {
	trackChar := v.trackChar
	thumbChar := v.thumbChar
	if trackChar == 0 {
		if v.horizontal {
			trackChar = '─'
		} else {
			trackChar = '│'
		}
	}
	if thumbChar == 0 {
		thumbChar = '█'
	}
	ext := &opScrollbar{
		contentSize: v.contentSize,
		viewSize:    v.viewSize,
		posPtr:      v.position,
		horizontal:  v.horizontal,
		trackChar:   trackChar,
		thumbChar:   thumbChar,
		trackStyle:  v.trackStyle,
		thumbStyle:  v.thumbStyle,
	}
	return t.addOp(Op{
		Kind:   OpScrollbar,
		Parent: parent,
		Width:  v.length,
		Height: v.length,
		Margin: v.margin,
		Ext:    ext,
	}, depth)
}

func (t *Template) compileAutoTableC(v AutoTableC, parent int16, depth int) int16 {
	rv := reflect.ValueOf(v.data)

	// pointer to slice -> reactive mode (reads data each frame)
	if rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.Slice {
		return t.compileAutoTableReactive(v, rv, parent, depth)
	}

	if rv.Kind() != reflect.Slice {
		return t.compileTextC(Text("AutoTable: expected slice or *slice"), parent, depth, nil, 0)
	}

	// static slice -> snapshot mode (existing behaviour)
	return t.compileAutoTableStatic(v, rv, parent, depth)
}

// compileAutoTableReactive compiles an AutoTable backed by *[]T into a single
// OpAutoTable that reads through the pointer on every render frame.
func (t *Template) compileAutoTableReactive(v AutoTableC, rv reflect.Value, parent int16, depth int) int16 {
	sliceType := rv.Elem().Type() // []T
	elemType := sliceType.Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return t.compileTextC(Text("AutoTable: expected *[]struct"), parent, depth, nil, 0)
	}

	columns, fieldIndices := autoTableResolveColumns(v.columns, elemType)
	if len(columns) == 0 {
		return t.compileTextC(Text("AutoTable: no columns"), parent, depth, nil, 0)
	}

	headers := v.headers
	if len(headers) == 0 {
		headers = make([]string, len(columns))
		copy(headers, columns)
	}

	// resolve per-column configs
	colCfgs := make([]*ColumnConfig, len(columns))
	for i, name := range columns {
		cfg := &ColumnConfig{}
		fi := fieldIndices[i]

		// apply type-based default alignment
		ft := elemType.Field(fi).Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		cfg.align = autoTableDefaultAlign(ft)

		// apply user config on top (overrides type defaults)
		if opt, ok := v.columnConfigs[name]; ok {
			opt(cfg)
		}

		colCfgs[i] = cfg
	}

	var altFill Color
	if v.altRowStyle != nil && v.altRowStyle.BG.Mode != ColorDefault {
		altFill = v.altRowStyle.BG
	}

	// resolve SortBy field name to column index (once)
	if ss := v.sortState; ss != nil && ss.initialCol != "" && !ss.initialDone {
		for i, name := range columns {
			if name == ss.initialCol {
				ss.col = i
				ss.asc = ss.initialAsc
				break
			}
		}
		ss.initialDone = true
	}

	ext := &opAutoTable{
		slicePtr: v.data,
		fields:   fieldIndices,
		headers:  headers,
		hdrStyle: v.headerStyle,
		rowStyle: v.rowStyle,
		altStyle: v.altRowStyle,
		gap:      v.gap,
		fill:     altFill,
		colCfgs:  colCfgs,
		sort:     v.sortState,
		scroll:   v.scroll,
	}

	idx := t.addOp(Op{
		Kind:   OpAutoTable,
		Parent: parent,
		Gap:    v.gap,
		Margin: v.margin,
		Ext:    ext,
	}, depth)
	if v.gapCond != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Gap = t.compileDynInt8(v.gapCond)
	} else if v.gapPtr != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Gap = v.gapPtr
	}
	return idx
}

// alignOffset returns the x offset needed to align text within the given width.
func alignOffset(text string, width int, align Align) int {
	textLen := utf8.RuneCountInString(text)
	if textLen >= width {
		return 0
	}
	pad := width - textLen
	switch align {
	case AlignRight:
		return pad
	case AlignCenter:
		return pad / 2
	default:
		return 0
	}
}

// autoTableDefaultAlign returns sensible default alignment based on type.
func autoTableDefaultAlign(t reflect.Type) Align {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return AlignRight
	case reflect.Bool:
		return AlignCenter
	default:
		return AlignLeft
	}
}

// autoTableResolveColumns resolves column names to struct field indices.
func autoTableResolveColumns(explicit []string, elemType reflect.Type) (names []string, indices []int) {
	if len(explicit) > 0 {
		for _, name := range explicit {
			f, ok := elemType.FieldByName(name)
			if ok {
				names = append(names, name)
				indices = append(indices, f.Index[0])
			}
		}
		return
	}
	// all exported fields
	for i := 0; i < elemType.NumField(); i++ {
		f := elemType.Field(i)
		if f.PkgPath == "" {
			names = append(names, f.Name)
			indices = append(indices, i)
		}
	}
	return
}

// compileAutoTableStatic compiles a static (non-pointer) slice into a VBox tree (original behaviour).
func (t *Template) compileAutoTableStatic(v AutoTableC, rv reflect.Value, parent int16, depth int) int16 {
	elemType := rv.Type().Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return t.compileTextC(Text("AutoTable: expected slice of structs"), parent, depth, nil, 0)
	}

	columns := v.columns
	if len(columns) == 0 {
		for i := 0; i < elemType.NumField(); i++ {
			f := elemType.Field(i)
			if f.PkgPath == "" {
				columns = append(columns, f.Name)
			}
		}
	}

	if len(columns) == 0 {
		return t.compileTextC(Text("AutoTable: no columns"), parent, depth, nil, 0)
	}

	headers := v.headers
	if len(headers) == 0 {
		headers = columns
	}

	// resolve column configs for static path
	colCfgs := make([]*ColumnConfig, len(columns))
	for i, col := range columns {
		cfg := &ColumnConfig{}
		if f, ok := elemType.FieldByName(col); ok {
			ft := f.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			cfg.align = autoTableDefaultAlign(ft)
		}
		if opt, ok := v.columnConfigs[col]; ok {
			opt(cfg)
		}
		colCfgs[i] = cfg
	}

	widths := make([]int, len(columns))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		for j, col := range columns {
			field := elem.FieldByName(col)
			if field.IsValid() {
				var str string
				if cfg := colCfgs[j]; cfg != nil && cfg.format != nil {
					str = cfg.format(field.Interface())
				} else {
					str = fmt.Sprintf("%v", field.Interface())
				}
				if len(str) > widths[j] {
					widths[j] = len(str)
				}
			}
		}
	}

	var rows []any

	var headerCells []any
	for i, h := range headers {
		hdrStyle := v.headerStyle
		if cfg := colCfgs[i]; cfg != nil {
			hdrStyle.Align = cfg.align
		}
		headerCells = append(headerCells, Text(h).Width(int16(widths[i])).Style(hdrStyle))
	}
	// use cond > pointer > static gap
	var tableGap any
	if v.gapCond != nil {
		tableGap = v.gapCond
	} else if v.gapPtr != nil {
		tableGap = v.gapPtr
	} else {
		tableGap = v.gap
	}
	rows = append(rows, HBox.Gap(tableGap)(headerCells...))

	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		isAlt := v.altRowStyle != nil && i%2 == 1
		rowStyle := v.rowStyle
		if isAlt {
			rowStyle = *v.altRowStyle
		}

		var cells []any
		for j, col := range columns {
			field := elem.FieldByName(col)
			var str string
			cellStyle := rowStyle
			if field.IsValid() {
				val := field.Interface()
				cfg := colCfgs[j]
				if cfg != nil && cfg.format != nil {
					str = cfg.format(val)
				} else {
					str = fmt.Sprintf("%v", val)
				}
				if cfg != nil && cfg.style != nil {
					cellStyle = cfg.style(val)
				}
			}
			if cfg := colCfgs[j]; cfg != nil {
				cellStyle.Align = cfg.align
			}
			cells = append(cells, Text(str).Width(int16(widths[j])).Style(cellStyle))
		}

		row := HBox.Gap(tableGap)
		if isAlt && rowStyle.BG.Mode != ColorDefault {
			row = HBox.Gap(tableGap).Fill(rowStyle.BG)
		}
		rows = append(rows, row(cells...))
	}

	var vbox VBoxC
	if v.border.Horizontal != 0 {
		vbox = VBox.Border(v.border)(rows...)
	} else {
		vbox = VBox(rows...)
	}
	vbox.margin = v.margin

	return t.compileVBoxC(vbox, parent, depth, nil, 0)
}

func (t *Template) compileCheckboxC(v *CheckboxC, parent int16, depth int, elemBase unsafe.Pointer) int16 {
	// Checkbox is: [mark] [label]
	// The mark is conditional based on checked state
	var labelNode any
	if v.labelPtr != nil {
		labelNode = Text(v.labelPtr)
	} else {
		labelNode = Text(v.label)
	}

	// Use If for the checkbox mark
	mark := If(v.checked).Then(Text(v.checkedMark)).Else(Text(v.unchecked))

	box := HBox.Gap(1)(mark, labelNode)
	box.margin = v.style.margin
	return t.compileHBoxC(box, parent, depth, elemBase, 0)
}

func (t *Template) compileRadioC(v *RadioC, parent int16, depth int) int16 {
	// Radio is a list of options with selection marks
	opts := v.getOptions()
	if len(opts) == 0 {
		return t.compileTextC(Text("(no options)"), parent, depth, nil, 0)
	}

	var items []any
	for i, opt := range opts {
		idx := i // capture for closure
		mark := IfOrd(v.selected).Eq(idx).Then(Text(v.selectedMark)).Else(Text(v.unselected))
		item := HBox.Gap(1)(mark, Text(opt))
		items = append(items, item)
	}

	// use cond > pointer > static gap
	var gap any
	if v.gapCond != nil {
		gap = v.gapCond
	} else if v.gapPtr != nil {
		gap = v.gapPtr
	} else {
		gap = v.gap
	}

	if v.horizontal {
		hbox := HBox.Gap(gap)(items...)
		hbox.margin = v.style.margin
		return t.compileHBoxC(hbox, parent, depth, nil, 0)
	}
	vbox := VBox.Gap(gap)(items...)
	vbox.margin = v.style.margin
	return t.compileVBoxC(vbox, parent, depth, nil, 0)
}

func (t *Template) compileInputC(v *InputC, parent int16, depth int) int16 {
	// Convert to TextInput and compile
	ti := v.toTextInput()
	idx := t.compile(ti, parent, depth, nil, 0)
	if v.widthCond != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Width = t.compileDynInt16(v.widthCond)
	} else if v.widthPtr != nil {
		if t.ops[idx].Dyn == nil {
			t.ops[idx].Dyn = &OpDyn{}
		}
		t.ops[idx].Dyn.Width = v.widthPtr
	}
	return idx
}

// Execute runs all three phases and renders to the buffer.
func (t *Template) Execute(buf *Buffer, screenW, screenH int16) {
	// Clear pending from previous frame
	t.pendingOverlays = t.pendingOverlays[:0]
	t.pendingScreenEffects = t.pendingScreenEffects[:0]

	// Phase 0: Evaluate reactive bindings (conditions, animations)
	t.frameTime = time.Now()
	t.animating = false
	for _, eval := range t.evals {
		eval()
	}

	// manage animation ticker — start at ~60fps when animating, stop when settled
	if t.animating && t.animTicker == nil && t.requestRender != nil {
		t.animTicker = time.NewTicker(16 * time.Millisecond)
		go func() {
			for range t.animTicker.C {
				t.requestRender()
			}
		}()
	} else if !t.animating && t.animTicker != nil {
		t.animTicker.Stop()
		t.animTicker = nil
	}

	// Phase 1: Width distribution (top → down)
	t.distributeWidths(screenW, nil)

	// Phase 2: Layout (bottom → up) - computes content heights
	t.layout(screenH)

	// Phase 2b: Flex distribution (top → down) - expand flex children
	t.distributeFlexGrow(screenH)

	// Phase 3: Render (top → down)
	t.render(buf, 0, 0, screenW)

	// Phase 4: Render overlays (after main content so they appear on top)
	t.renderOverlays(buf, screenW, screenH)
}

// distributeWidths assigns W to all ops, top-down.
// Each container sets its children's widths. For Rows, this includes flex distribution.
// elemBase is optional - used for offset-based text in ForEach sub-templates.
func (t *Template) distributeWidths(screenW int16, elemBase unsafe.Pointer) {
	// Set root-level ops to screen width first (or compute intrinsic width if FitContent)
	for _, idx := range t.byDepth[0] {
		op := &t.ops[idx]
		geom := &t.geom[idx]
		if op.FitContent {
			// Compute intrinsic width from children
			intrinsicW := t.computeIntrinsicWidth(idx)
			geom.W = intrinsicW
		} else {
			t.setOpWidth(op, geom, screenW, elemBase)
		}
	}

	// Process containers depth-by-depth, each setting its children's widths
	for depth := 0; depth <= t.maxDepth; depth++ {
		for _, idx := range t.byDepth[depth] {
			op := &t.ops[idx]
			geom := &t.geom[idx]

			switch op.Kind {
			case OpContainer:
				t.distributeWidthsToChildren(idx, op, geom, elemBase)
			case OpJump:
				// Jump is a transparent wrapper - distribute full width to children (like VBox)
				t.distributeVBoxChildWidths(idx, op, geom.W, elemBase)
			}
		}
	}
}

// computeIntrinsicWidth computes the minimum width needed for a ContentSized container.
// For VBox: maximum width of children (all children stack vertically, need same width)
// For HBox: sum of children widths + gaps
func (t *Template) computeIntrinsicWidth(idx int16) int16 {
	op := &t.ops[idx]

	// If this op has an explicit width, use it
	if w := op.width(); w > 0 {
		return w
	}

	// For containers, compute from children
	if op.Kind == OpContainer {
		var intrinsicW int16

		// Count children and find max/sum
		childCount := int16(0)
		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &t.ops[i]
			if childOp.Parent != idx {
				continue
			}
			childW := t.computeIntrinsicWidth(i)
			childCount++

			if op.IsRow {
				// HBox: sum widths
				intrinsicW += childW
			} else {
				// VBox: max width
				if childW > intrinsicW {
					intrinsicW = childW
				}
			}
		}

		// Add gaps for HBox
		if g := op.gap(); op.IsRow && childCount > 1 && g > 0 {
			intrinsicW += int16(g) * (childCount - 1)
		}

		// Add border
		if op.Border.Horizontal != 0 {
			intrinsicW += 2
		}

		// Add margin
		intrinsicW += op.marginH()

		return intrinsicW
	}

	// For text, compute string width
	if op.Kind == OpText {
		return op.Ext.(*opText).textWidth(nil) + op.marginH()
	}

	return op.marginH()
}

// setOpWidth sets a single op's width based on available space.
func (t *Template) setOpWidth(op *Op, geom *Geom, availW int16, elemBase unsafe.Pointer) {
	switch op.Kind {
	case OpText:
		if w := op.width(); w > 0 {
			geom.W = w
		} else {
			geom.W = op.Ext.(*opText).textWidth(elemBase)
		}

	case OpProgress:
		geom.W = op.width()

	case OpCounter:
		ext := op.Ext.(*opCounter)
		var scratch [48]byte
		b := append(scratch[:0], ext.prefix...)
		b = strconv.AppendInt(b, int64(*ext.currentPtr), 10)
		b = append(b, '/')
		b = strconv.AppendInt(b, int64(*ext.totalPtr), 10)
		geom.W = int16(len(b))

	case OpLeader:
		geom.W = op.width()
		if geom.W == 0 {
			geom.W = 20
		}

	case OpAutoTable:
		geom.W = availW

	case OpTable:
		ext := op.Ext.(*opTable)
		totalW := 0
		for _, col := range ext.columns {
			if col.Width > 0 {
				totalW += col.Width
			} else {
				totalW += 10
			}
		}
		geom.W = int16(totalW)

	case OpSparkline:
		geom.W = op.width()
		if geom.W == 0 {
			if availW > 0 {
				geom.W = availW
			} else {
				geom.W = int16(op.Ext.(*opSparkline).dataLen())
			}
		}

	case OpHRule:
		geom.W = 0 // fill available

	case OpVRule:
		geom.W = 1 // single column

	case OpSpacer:
		geom.W = op.width() // 0 = fill available

	case OpSpinner:
		geom.W = 1 // single character width

	case OpScrollbar:
		ext := op.Ext.(*opScrollbar)
		if ext.horizontal {
			if w := op.width(); w > 0 {
				geom.W = w
			} else {
				geom.W = availW
			}
		} else {
			geom.W = 1
		}

	case OpTabs:
		ext := op.Ext.(*opTabs)
		totalW := 0
		for i, label := range ext.labels {
			labelW := utf8.RuneCountInString(label)
			switch ext.styleType {
			case TabsStyleBox:
				labelW += 4
			case TabsStyleBracket:
				labelW += 2
			}
			totalW += labelW
			if i < len(ext.labels)-1 {
				totalW += ext.gap
			}
		}
		geom.W = int16(totalW)

	case OpTreeView:
		ext := op.Ext.(*opTreeView)
		maxW := 0
		if ext.root != nil {
			startLevel := 0
			if !ext.showRoot {
				startLevel = -1
			}
			maxW = t.treeMaxWidth(ext.root, startLevel, ext.indent, ext.showRoot)
		}
		geom.W = int16(maxW)

	case OpCustom:
		ext := op.Ext.(*opCustomRenderer)
		if ext.renderer != nil {
			if cw, ok := ext.renderer.(*customWrapper); ok {
				w, _ := cw.MeasureWithAvail(availW)
				geom.W = w
			} else {
				w, _ := ext.renderer.MinSize()
				geom.W = int16(w)
			}
		}

	case OpLayout:
		geom.W = availW

	case OpLayer:
		ext := op.Ext.(*opLayer)
		if ext.width > 0 {
			geom.W = ext.width
		} else {
			geom.W = availW
		}

	case OpSelectionList:
		geom.W = availW

	case OpJump:
		// Jump is a transparent wrapper - uses full available width
		// Children will be laid out within this width
		geom.W = availW

	case OpTextInput:
		// TextInput uses explicit width or fills available
		if w := op.width(); w > 0 {
			geom.W = w
		} else {
			geom.W = availW
		}

	case OpOverlay, OpScreenEffect:
		// Overlays and screen effects take zero space in layout
		geom.W = 0

	case OpIf:
		// Calculate width from the active branch content
		ifExt := op.Ext.(*opIf)
		condTrue := ifExt.eval(elemBase)
		var subTmpl *Template
		if condTrue {
			subTmpl = ifExt.thenTmpl
		} else {
			subTmpl = ifExt.elseTmpl
		}
		if subTmpl != nil {
			subTmpl.elemBase = elemBase
			// computeIntrinsicWidth handles both ContentSized containers and
			// leaf nodes (OpText, etc.) that have a computable fixed width.
			// Falls back to 0 for truly flexible content (Space, unsized containers).
			intrinsicW := subTmpl.computeIntrinsicWidth(0)
			if intrinsicW > 0 {
				subTmpl.distributeWidths(intrinsicW, elemBase)
				geom.W = intrinsicW
			} else {
				subTmpl.distributeWidths(availW, elemBase)
				if len(subTmpl.geom) > 0 {
					geom.W = subTmpl.geom[0].W
				}
			}
		} else {
			// Condition false with no else branch - takes no space
			geom.W = 0
		}

	case OpContainer:
		if w := op.width(); w > 0 {
			geom.W = w
		} else if pw := op.percentWidth(); pw > 0 {
			geom.W = int16(float32(availW) * pw)
		} else {
			geom.W = availW
		}

	default:
		geom.W = availW
	}

	// generic margin: non-container ops include margin in their outer width
	if op.Kind != OpContainer && op.marginH() > 0 {
		geom.W += op.marginH()
	}
}

// distributeWidthsToChildren sets widths for all children of a container.
// For Rows: two-pass (non-flex first, then flex distribution).
// For Cols: children fill available width.
func (t *Template) distributeWidthsToChildren(idx int16, op *Op, geom *Geom, elemBase unsafe.Pointer) {
	// Calculate content width (subtract margin + border)
	contentW := geom.W - op.marginH()
	if op.Border.Horizontal != 0 {
		contentW -= 2
	}

	if op.IsRow {
		t.distributeHBoxChildWidths(idx, op, contentW, elemBase)
	} else {
		t.distributeVBoxChildWidths(idx, op, contentW, elemBase)
	}
}

// distributeVBoxChildWidths sets widths for children of a VBox (they fill available width).
func (t *Template) distributeVBoxChildWidths(idx int16, op *Op, availW int16, elemBase unsafe.Pointer) {
	for i := op.ChildStart; i < op.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Parent != idx {
			continue
		}
		childGeom := &t.geom[i]
		t.setOpWidth(childOp, childGeom, availW, elemBase)
	}
}

// getIfContentOp returns the root op of an If's active branch content.
// Returns nil if condition is false and no else branch, or if template is empty.
func (t *Template) getIfContentOp(childOp *Op, elemBase unsafe.Pointer) *Op {
	childIfExt := childOp.Ext.(*opIf)
	condTrue := childIfExt.eval(elemBase)

	if condTrue && childIfExt.thenTmpl != nil && len(childIfExt.thenTmpl.ops) > 0 {
		return &childIfExt.thenTmpl.ops[0]
	} else if !condTrue && childIfExt.elseTmpl != nil && len(childIfExt.elseTmpl.ops) > 0 {
		return &childIfExt.elseTmpl.ops[0]
	}
	return nil
}

// distributeHBoxChildWidths sets widths for children of a HBox using two-pass flex.
func (t *Template) distributeHBoxChildWidths(idx int16, op *Op, availW int16, elemBase unsafe.Pointer) {
	// Pass 1: Set widths for non-flex children, collect flex children
	// Containers without explicit width/flex are treated as implicit flex (share remaining space)
	// OpIf is transparent - we look at its content's properties
	var usedW int16
	var totalFlex float32
	var fixedWidthCount int16 // count of non-flex children with width

	flexChildren := t.flexScratchIdx[:0]
	flexGrowValues := t.flexScratchGrow[:0]
	implicitFlexChildren := t.flexScratchImpl[:0]

	for i := op.ChildStart; i < op.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Parent != idx {
			continue
		}
		childGeom := &t.geom[i]

		// For OpIf, look at the content's properties (transparent wrapper)
		effectiveOp := childOp
		if childOp.Kind == OpIf {
			contentOp := t.getIfContentOp(childOp, elemBase)
			if contentOp == nil {
				// Condition false with no else - takes no space
				childGeom.W = 0
				continue
			}
			effectiveOp = contentOp
		}

		if fg := effectiveOp.flexGrow(); fg > 0 {
			// Explicit flex child - defer to pass 2
			totalFlex += fg
			flexChildren = append(flexChildren, i)
			flexGrowValues = append(flexGrowValues, fg)
		} else if !effectiveOp.ContentSized && (effectiveOp.Kind == OpContainer || effectiveOp.Kind == OpJump) && effectiveOp.width() == 0 && effectiveOp.percentWidth() == 0 {
			// Container/Jump without explicit width or fixed-content children - implicit flex
			implicitFlexChildren = append(implicitFlexChildren, i)
		} else {
			// Non-flex child with explicit or content-based width
			t.setOpWidth(childOp, childGeom, availW, elemBase)
			usedW += childGeom.W
			if childGeom.W > 0 {
				fixedWidthCount++
			}
		}
	}

	t.flexScratchIdx = flexChildren
	t.flexScratchGrow = flexGrowValues
	t.flexScratchImpl = implicitFlexChildren

	// Account for gaps - total children that will take space
	// Note: we track fixedWidthCount during the loop above to avoid double-counting
	// flex children that might have non-zero W from a previous render
	childCount := fixedWidthCount + int16(len(flexChildren)) + int16(len(implicitFlexChildren))
	if g := op.gap(); childCount > 1 && g > 0 {
		usedW += int16(g) * (childCount - 1)
	}

	// Pass 2: Distribute remaining width to flex children
	remaining := availW - usedW
	if remaining > 0 && totalFlex > 0 {
		// Explicit flex children
		distributed := int16(0)
		for i, childIdx := range flexChildren {
			childOp := &t.ops[childIdx]
			childGeom := &t.geom[childIdx]

			flexShare := flexGrowValues[i] / totalFlex
			flexW := int16(float32(remaining) * flexShare)

			// Last flex child gets remainder (avoid rounding loss)
			if i == len(flexChildren)-1 {
				flexW = remaining - distributed
			}
			distributed += flexW

			// Set the flex child's width
			childGeom.W = flexW

			// For OpIf, also distribute to sub-template
			if childOp.Kind == OpIf {
				childIfExt := childOp.Ext.(*opIf)
				condTrue := childIfExt.eval(elemBase)
				if condTrue && childIfExt.thenTmpl != nil {
					childIfExt.thenTmpl.elemBase = elemBase
					childIfExt.thenTmpl.distributeWidths(flexW, elemBase)
				} else if !condTrue && childIfExt.elseTmpl != nil {
					childIfExt.elseTmpl.elemBase = elemBase
					childIfExt.elseTmpl.distributeWidths(flexW, elemBase)
				}
			}
		}
	} else if remaining > 0 && len(implicitFlexChildren) > 0 {
		// No explicit flex, but implicit flex containers - share remaining evenly
		shareW := remaining / int16(len(implicitFlexChildren))
		distributed := int16(0)
		for i, childIdx := range implicitFlexChildren {
			childOp := &t.ops[childIdx]
			childGeom := &t.geom[childIdx]

			w := shareW
			// Last child gets remainder
			if i == len(implicitFlexChildren)-1 {
				w = remaining - distributed
			}
			distributed += w
			childGeom.W = w

			// For OpIf, also distribute to sub-template
			if childOp.Kind == OpIf {
				childIfExt := childOp.Ext.(*opIf)
				condTrue := childIfExt.eval(elemBase)
				if condTrue && childIfExt.thenTmpl != nil {
					childIfExt.thenTmpl.elemBase = elemBase
					childIfExt.thenTmpl.distributeWidths(w, elemBase)
				} else if !condTrue && childIfExt.elseTmpl != nil {
					childIfExt.elseTmpl.elemBase = elemBase
					childIfExt.elseTmpl.distributeWidths(w, elemBase)
				}
			}
		}
	}

	// Annotate Extend HRules: find VRule siblings and stamp their X delta onto
	// any HRule with RuleExtend=true in sibling container children.
	t.annotateHRuleExtensions(idx, op, availW)
}

// annotateHRuleExtensions finds VRule children of the HBox at idx and, for each,
// walks sibling container subtrees to set RuleVRuleX on HRules with RuleExtend=true.
// It also checks whether the HBox's parent has a border and stamps border extension
// deltas (RuleExtendLeft/Right) onto the outermost container HRules.
func (t *Template) annotateHRuleExtensions(hboxIdx int16, hboxOp *Op, availW int16) {
	// compute each direct child's X offset within the HBox content area
	cursor := int16(0)
	type childInfo struct {
		idx    int16
		xStart int16
	}
	var children []childInfo
	for i := hboxOp.ChildStart; i < hboxOp.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Parent != hboxIdx {
			continue
		}
		w := t.geom[i].W
		children = append(children, childInfo{idx: i, xStart: cursor})
		if w > 0 {
			cursor += w + int16(hboxOp.gap())
		}
	}

	// for each container child, stamp deltas to its nearest left and right VRules only.
	// using nearest-neighbor (not all VRules) prevents extending through other content.
	// track hasLeft/hasRight per container so border extension can be skipped when a
	// VRule already terminates the HRule on that side.
	type containerSides struct{ hasLeft, hasRight bool }
	sides := map[int16]containerSides{}
	for ci := range children {
		c := &children[ci]
		if t.ops[c.idx].Kind != OpContainer {
			continue
		}
		var leftDelta, rightDelta int16
		hasLeft, hasRight := false, false
		for _, v := range children {
			if t.ops[v.idx].Kind != OpVRule {
				continue
			}
			d := v.xStart - c.xStart
			if d < 0 {
				// VRule to the left — take nearest (largest xStart, i.e. least negative delta)
				if !hasLeft || d > leftDelta {
					leftDelta = d
					hasLeft = true
				}
			} else if d > 0 {
				// VRule to the right — take nearest (smallest xStart, i.e. smallest delta)
				if !hasRight || d < rightDelta {
					rightDelta = d
					hasRight = true
				}
			}
		}
		sides[c.idx] = containerSides{hasLeft: hasLeft, hasRight: hasRight}
		if hasLeft && hasRight {
			t.stampVRuleXPair(c.idx, leftDelta, rightDelta)
		} else if hasLeft {
			t.stampVRuleX(c.idx, leftDelta)
		} else if hasRight {
			t.stampVRuleX(c.idx, rightDelta)
		}
	}

	// if the HBox's direct parent has a border, extend the leftmost and rightmost
	// container HRules to meet the border walls (producing ├ and ┤ junctions).
	// skip border extension when a VRule already terminates the HRule on that side —
	// the VRule endpoint cap will produce ├/┤ via buffer merge instead.
	if hboxOp.Parent >= 0 && int(hboxOp.Parent) < len(t.ops) {
		parentOp := &t.ops[hboxOp.Parent]
		if parentOp.Kind == OpContainer && parentOp.Border.Horizontal != 0 {
			var leftmost, rightmost *childInfo
			for i := range children {
				c := &children[i]
				if t.ops[c.idx].Kind != OpContainer {
					continue
				}
				if leftmost == nil || c.xStart < leftmost.xStart {
					leftmost = &children[i]
				}
				if rightmost == nil || c.xStart > rightmost.xStart {
					rightmost = &children[i]
				}
			}
			if leftmost != nil && !sides[leftmost.idx].hasLeft {
				leftExt := leftmost.xStart + int16(hboxOp.Margin[3]) + 1
				t.stampHRuleExtendBorder(leftmost.idx, leftExt, 0)
			}
			if rightmost != nil && !sides[rightmost.idx].hasRight {
				rightExt := (availW - rightmost.xStart - t.geom[rightmost.idx].W) + int16(hboxOp.Margin[1]) + 1
				t.stampHRuleExtendBorder(rightmost.idx, 0, rightExt)
			}
		}
	}
}

// stampHRuleExtendBorder recursively sets RuleExtendLeft/Right on HRules with
// RuleExtend=true within the subtree rooted at containerIdx.
func (t *Template) stampHRuleExtendBorder(containerIdx int16, left, right int16) {
	containerOp := &t.ops[containerIdx]
	for i := containerOp.ChildStart; i < containerOp.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Kind == OpHRule {
			ext := childOp.Ext.(*opRule)
			if ext.extend {
				if left > 0 {
					ext.extendLeft = left
				}
				if right > 0 {
					ext.extendRight = right
				}
			}
		}
		if childOp.Kind == OpContainer {
			t.stampHRuleExtendBorder(i, left, right)
		}
	}
}

// stampVRuleX recursively sets RuleVRuleX on all HRules with RuleExtend=true
// within the subtree rooted at containerIdx.
func (t *Template) stampVRuleX(containerIdx int16, delta int16) {
	containerOp := &t.ops[containerIdx]
	for i := containerOp.ChildStart; i < containerOp.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Kind == OpHRule {
			if ext := childOp.Ext.(*opRule); ext.extend {
				ext.vruleX = delta
			}
		}
		if childOp.Kind == OpContainer {
			t.stampVRuleX(i, delta)
		}
	}
}

// stampVRuleXPair recursively sets both RuleVRuleX and RuleVRuleX2 on all HRules
// with RuleExtend=true within the subtree rooted at containerIdx.
// Used when a container is flanked by VRules on both sides.
func (t *Template) stampVRuleXPair(containerIdx int16, delta1, delta2 int16) {
	containerOp := &t.ops[containerIdx]
	for i := containerOp.ChildStart; i < containerOp.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Kind == OpHRule {
			if ext := childOp.Ext.(*opRule); ext.extend {
				ext.vruleX = delta1
				ext.vruleX2 = delta2
			}
		}
		if childOp.Kind == OpContainer {
			t.stampVRuleXPair(i, delta1, delta2)
		}
	}
}

// annotateVRuleExtensions finds HRule children of the VBox at idx and, for each,
// walks sibling container subtrees to set RuleExtendTop/Bot on VRules with RuleExtend=true.
func (t *Template) annotateVRuleExtensions(idx int16, op *Op, totalH int16) {
	contentOffY := op.Margin[0]
	if op.Border.Horizontal != 0 {
		contentOffY++
	}

	type childInfo struct {
		idx    int16
		yStart int16
		height int16
	}
	var children []childInfo
	for i := op.ChildStart; i < op.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Parent != idx {
			continue
		}
		children = append(children, childInfo{i, t.geom[i].LocalY, t.geom[i].H})
	}

	// collect HRule Y positions
	hRuleYs := make(map[int16]bool)
	for _, c := range children {
		if t.ops[c.idx].Kind == OpHRule {
			hRuleYs[c.yStart] = true
		}
	}

	hasBorder := op.Border.Horizontal != 0

	for _, c := range children {
		childOp := &t.ops[c.idx]
		if hasBorder && childOp.Kind == OpHRule {
			if ext := childOp.Ext.(*opRule); ext.extend {
				ext.extendLeft = 1
				ext.extendRight = 1
				continue
			}
		}
		if childOp.Kind != OpContainer {
			continue
		}
		extTop := hRuleYs[c.yStart-1] || (hasBorder && c.yStart == contentOffY)
		extBot := hRuleYs[c.yStart+c.height] || (hasBorder && c.yStart+c.height == contentOffY+totalH)
		if extTop || extBot {
			t.stampVRuleExtend(c.idx, extTop, extBot)
		}
	}
}

// stampVRuleExtend recursively sets RuleExtendTop/Bot on all VRules with RuleExtend=true
// within the subtree rooted at containerIdx.
func (t *Template) stampVRuleExtend(containerIdx int16, top, bot bool) {
	containerOp := &t.ops[containerIdx]
	for i := containerOp.ChildStart; i < containerOp.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Kind == OpVRule {
			if ext := childOp.Ext.(*opRule); ext.extend {
				ext.extendTop = top
				ext.extendBot = bot
			}
		}
		if childOp.Kind == OpContainer {
			t.stampVRuleExtend(i, top, bot)
		}
	}
}

// layout computes H and local positions, bottom-up.
func (t *Template) layout(_ int16) {
	// Bottom-up: deepest first
	for depth := t.maxDepth; depth >= 0; depth-- {
		for _, idx := range t.byDepth[depth] {
			op := &t.ops[idx]
			geom := &t.geom[idx]

			switch op.Kind {
			case OpText, OpProgress, OpRichText, OpLeader, OpCounter:
				geom.H = 1

			case OpAutoTable:
				ext := op.Ext.(*opAutoTable)
				dataRows := 0
				if ext.slicePtr != nil {
					dataRows = reflect.ValueOf(ext.slicePtr).Elem().Len()
				}
				visibleRows := dataRows
				if sc := ext.scroll; sc != nil && sc.maxVisible < visibleRows {
					visibleRows = sc.maxVisible
				}
				geom.H = int16(visibleRows + 1)
				if geom.H == 0 {
					geom.H = 1
				}

			case OpTable:
				ext := op.Ext.(*opTable)
				rowCount := 0
				if ext.rowsPtr != nil {
					rowCount = len(*ext.rowsPtr)
				}
				if ext.showHeader {
					rowCount++
				}
				geom.H = int16(rowCount)
				if geom.H == 0 {
					geom.H = 1
				}

			case OpSparkline:
				geom.H = op.height()
				if geom.H <= 0 {
					geom.H = 1
				}

			case OpHRule:
				geom.H = 1

			case OpVRule:
				geom.H = 1 // default height (will be stretched by flex)

			case OpSpacer:
				geom.H = op.height()

			case OpSpinner:
				geom.H = 1 // single line

			case OpScrollbar:
				ext := op.Ext.(*opScrollbar)
				if ext.horizontal {
					geom.H = 1
				} else {
					if h := op.height(); h > 0 {
						geom.H = h
					} else {
						geom.H = 1
					}
				}

			case OpTabs:
				ext := op.Ext.(*opTabs)
				switch ext.styleType {
				case TabsStyleBox:
					geom.H = 3
				default:
					geom.H = 1
				}

			case OpTreeView:
				ext := op.Ext.(*opTreeView)
				count := 0
				if ext.root != nil {
					count = t.treeVisibleCount(ext.root, ext.showRoot)
				}
				geom.H = int16(count)
				if geom.H == 0 {
					geom.H = 1
				}

			case OpSelectionList:
				ext := op.Ext.(*opSelectionList)
				sliceHdr := *(*sliceHeader)(ext.slicePtr)
				if ext.listPtr != nil {
					ext.listPtr.len = sliceHdr.Len
					ext.listPtr.ensureVisible()
				}
				visibleCount := sliceHdr.Len
				if ext.listPtr != nil && ext.listPtr.MaxVisible > 0 && visibleCount > ext.listPtr.MaxVisible {
					visibleCount = ext.listPtr.MaxVisible
				}
				geom.H = int16(visibleCount)
				if geom.H == 0 {
					geom.H = 1
				}

			case OpCustom:
				// Custom renderer provides its own size
				crExt := op.Ext.(*opCustomRenderer)
				if crExt.renderer != nil {
					// Use customWrapper with computed width for better sizing
					if cw, ok := crExt.renderer.(*customWrapper); ok {
						_, h := cw.MeasureWithAvail(geom.W)
						geom.H = h
					} else {
						_, h := crExt.renderer.MinSize()
						geom.H = int16(h)
					}
				}

			case OpLayer:
				ext := op.Ext.(*opLayer)
				if ext.height > 0 {
					geom.H = ext.height
				} else if op.flexGrow() > 0 {
					geom.H = 1
				} else if ext.ptr != nil && ext.ptr.viewHeight > 0 {
					geom.H = int16(ext.ptr.viewHeight)
				} else {
					geom.H = 1
				}
				geom.ContentH = geom.H

			case OpJump:
				// Jump's height is sum of children's heights (like a VBox)
				totalH := int16(0)
				for i := op.ChildStart; i < op.ChildEnd; i++ {
					childOp := &t.ops[i]
					if childOp.Parent == idx {
						childGeom := &t.geom[i]
						childGeom.LocalX = 0
						childGeom.LocalY = totalH
						totalH += childGeom.H
					}
				}
				geom.H = totalH
				if geom.H == 0 {
					geom.H = 1
				}

			case OpTextInput:
				// TextInput is always 1 line
				geom.H = 1

			case OpOverlay, OpScreenEffect:
				// Overlays and screen effects take zero space in layout
				geom.H = 0

			case OpIf:
				// root-level OpIf (e.g. ForEach iter template root); container children
				// are handled inline by layoutContainer and skipped here
				if op.Parent != -1 {
					break
				}
				ifExt := op.Ext.(*opIf)
				condTrue := ifExt.eval(t.elemBase)
				if condTrue && ifExt.thenTmpl != nil {
					ifExt.thenTmpl.elemBase = t.elemBase
					ifExt.thenTmpl.distributeWidths(geom.W, t.elemBase)
					ifExt.thenTmpl.layout(0)
					geom.H = ifExt.thenTmpl.Height()
				} else if !condTrue && ifExt.elseTmpl != nil {
					ifExt.elseTmpl.elemBase = t.elemBase
					ifExt.elseTmpl.distributeWidths(geom.W, t.elemBase)
					ifExt.elseTmpl.layout(0)
					geom.H = ifExt.elseTmpl.Height()
				}

			case OpSwitch:
				// root-level OpSwitch (e.g. ForEach iter template root); container
				// children are handled inline by layoutContainer and skipped here
				if op.Parent != -1 {
					break
				}
				swExt := op.Ext.(*opSwitch)
				matchIdx := swExt.node.getMatchIndexWithBase(t.elemBase)
				var switchTmpl *Template
				if matchIdx >= 0 && matchIdx < len(swExt.cases) {
					switchTmpl = swExt.cases[matchIdx]
				} else {
					switchTmpl = swExt.def
				}
				if switchTmpl != nil {
					switchTmpl.elemBase = t.elemBase
					switchTmpl.distributeWidths(geom.W, t.elemBase)
					switchTmpl.layout(0)
					geom.H = switchTmpl.Height()
				}

			case OpLayout:
				t.layoutCustom(idx, op, geom)

			case OpContainer:
				t.layoutContainer(idx, op, geom)
			}

			// generic margin: non-container ops include margin in their outer height
			if op.Kind != OpContainer && op.marginV() > 0 {
				geom.H += op.marginV()
			}
		}
	}
}

// layoutContainer positions children and computes container height.
func (t *Template) layoutContainer(idx int16, op *Op, geom *Geom) {
	// Content area offset for margin + border
	contentOffX := op.Margin[3] // left margin
	contentOffY := op.Margin[0] // top margin
	if op.Border.Horizontal != 0 {
		contentOffX += 1
		contentOffY += 1
	}

	availW := geom.W - op.marginH()
	if op.Border.Horizontal != 0 {
		availW -= 2
	}

	if op.IsRow {
		// Horizontal layout
		cursor := int16(0)
		maxH := int16(0)
		needGap := false // Add gap before next visible child

		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &t.ops[i]
			if childOp.Parent != idx {
				continue // not direct child
			}

			// Control flow ops expand to their content
			switch childOp.Kind {
			case OpIf:
				// Use evaluateWithBase for conditions in ForEach context
				childIfExt := childOp.Ext.(*opIf)
				condTrue := childIfExt.eval(t.elemBase)
				// Use pre-calculated width if set (from flex distribution), otherwise use availW
				ifWidth := t.geom[i].W
				if ifWidth == 0 {
					ifWidth = availW
				}
				if childIfExt.thenTmpl != nil && condTrue {
					// Add gap before this child if needed
					if g := op.gap(); needGap && g > 0 {
						cursor += int16(g)
					}
					childIfExt.thenTmpl.elemBase = t.elemBase
					childIfExt.thenTmpl.distributeWidths(ifWidth, t.elemBase)
					childIfExt.thenTmpl.layout(0)
					h := childIfExt.thenTmpl.Height()
					t.geom[i].LocalX = contentOffX + cursor
					t.geom[i].LocalY = contentOffY
					t.geom[i].H = h
					// Use sub-template width only if we didn't have a pre-set width
					if t.geom[i].W == 0 && len(childIfExt.thenTmpl.geom) > 0 {
						t.geom[i].W = childIfExt.thenTmpl.geom[0].W
					}
					cursor += t.geom[i].W
					if h > maxH {
						maxH = h
					}
					needGap = true // Next visible child needs gap
				} else if childIfExt.elseTmpl != nil && !condTrue {
					// Add gap before this child if needed
					if g := op.gap(); needGap && g > 0 {
						cursor += int16(g)
					}
					childIfExt.elseTmpl.elemBase = t.elemBase
					childIfExt.elseTmpl.distributeWidths(ifWidth, t.elemBase)
					childIfExt.elseTmpl.layout(0)
					h := childIfExt.elseTmpl.Height()
					t.geom[i].LocalX = contentOffX + cursor
					t.geom[i].LocalY = contentOffY
					t.geom[i].H = h
					if t.geom[i].W == 0 && len(childIfExt.elseTmpl.geom) > 0 {
						t.geom[i].W = childIfExt.elseTmpl.geom[0].W
					}
					cursor += t.geom[i].W
					if h > maxH {
						maxH = h
					}
					needGap = true // Next visible child needs gap
				}
				// If condition false with no else, don't set needGap (takes no space)

			case OpForEach:
				// Add gap before this child if needed
				if g := op.gap(); needGap && g > 0 {
					cursor += int16(g)
				}
				h, w := t.layoutForEach(i, childOp, availW)
				t.geom[i].LocalX = contentOffX + cursor
				t.geom[i].LocalY = contentOffY
				t.geom[i].H = h
				t.geom[i].W = w
				cursor += w
				if h > maxH {
					maxH = h
				}
				if w > 0 {
					needGap = true
				}

			case OpSwitch:
				// Layout all cases to find the maximum width. In a ForEach, all rows
				// share one geom array (last-element wins), so the Switch must reserve
				// enough space for any case that could render — otherwise wider cases
				// get truncated and column positions vary per row, breaking alignment.
				childSwExt := childOp.Ext.(*opSwitch)
				var maxCaseW, maxCaseH int16
				allCaseTmpls := append(childSwExt.cases, childSwExt.def)
				for _, ct := range allCaseTmpls {
					if ct == nil {
						continue
					}
					ct.elemBase = t.elemBase
					ct.distributeWidths(availW, t.elemBase)
					ct.layout(0)
					if len(ct.geom) > 0 && ct.geom[0].W > maxCaseW {
						maxCaseW = ct.geom[0].W
					}
					if h := ct.Height(); h > maxCaseH {
						maxCaseH = h
					}
				}
				if maxCaseW > 0 {
					if g := op.gap(); needGap && g > 0 {
						cursor += int16(g)
					}
					t.geom[i].LocalX = contentOffX + cursor
					t.geom[i].LocalY = contentOffY
					t.geom[i].W = maxCaseW
					t.geom[i].H = maxCaseH
					cursor += maxCaseW
					if maxCaseH > maxH {
						maxH = maxCaseH
					}
					needGap = true
				}

			default:
				childGeom := &t.geom[i]
				// Add gap before this child if needed
				if g := op.gap(); needGap && g > 0 && childGeom.W > 0 {
					cursor += int16(g)
				}
				childGeom.LocalX = contentOffX + cursor
				childGeom.LocalY = contentOffY
				cursor += childGeom.W
				if childGeom.H > maxH {
					maxH = childGeom.H
				}
				if childGeom.W > 0 {
					needGap = true
				}
			}
		}

		geom.H = maxH
		if op.Border.Horizontal != 0 {
			geom.H += 2
		}
		geom.H += op.marginV()
	} else {
		// Vertical layout
		cursor := int16(0)
		firstChild := true

		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &t.ops[i]
			if childOp.Parent != idx {
				continue
			}

			// Handle gap
			if g := op.gap(); !firstChild && g > 0 {
				cursor += int16(g)
			}
			firstChild = false

			// Control flow ops expand to their content
			switch childOp.Kind {
			case OpIf:
				// Use evaluateWithBase for conditions in ForEach context
				childIfExt := childOp.Ext.(*opIf)
				condTrue := childIfExt.eval(t.elemBase)
				if childIfExt.thenTmpl != nil && condTrue {
					childIfExt.thenTmpl.elemBase = t.elemBase
					childIfExt.thenTmpl.distributeWidths(availW, t.elemBase)
					childIfExt.thenTmpl.layout(0)
					h := childIfExt.thenTmpl.Height()
					t.geom[i].LocalX = contentOffX
					t.geom[i].LocalY = contentOffY + cursor
					t.geom[i].H = h
					t.geom[i].ContentH = h // Track content height for flex
					t.geom[i].W = availW
					cursor += h
				} else if childIfExt.elseTmpl != nil && !condTrue {
					childIfExt.elseTmpl.elemBase = t.elemBase
					childIfExt.elseTmpl.distributeWidths(availW, t.elemBase)
					childIfExt.elseTmpl.layout(0)
					h := childIfExt.elseTmpl.Height()
					t.geom[i].LocalX = contentOffX
					t.geom[i].LocalY = contentOffY + cursor
					t.geom[i].H = h
					t.geom[i].ContentH = h // Track content height for flex
					t.geom[i].W = availW
					cursor += h
				} else {
					t.geom[i].H = 0 // condition false and no else, takes no space
					t.geom[i].ContentH = 0
				}

			case OpForEach:
				h, _ := t.layoutForEach(i, childOp, availW)
				t.geom[i].LocalX = contentOffX
				t.geom[i].LocalY = contentOffY + cursor
				t.geom[i].H = h
				t.geom[i].W = availW
				cursor += h

			case OpSwitch:
				// Get matching template
				childSwExt := childOp.Ext.(*opSwitch)
				var tmpl *Template
				matchIdx := childSwExt.node.getMatchIndexWithBase(t.elemBase)
				if matchIdx >= 0 && matchIdx < len(childSwExt.cases) {
					tmpl = childSwExt.cases[matchIdx]
				} else {
					tmpl = childSwExt.def
				}
				if tmpl != nil {
					tmpl.elemBase = t.elemBase
					tmpl.distributeWidths(availW, t.elemBase)
					tmpl.layout(0)
					h := tmpl.Height()
					t.geom[i].LocalX = contentOffX
					t.geom[i].LocalY = contentOffY + cursor
					t.geom[i].H = h
					t.geom[i].W = availW
					cursor += h
				} else {
					t.geom[i].H = 0 // no matching case, takes no space
				}

			default:
				childGeom := &t.geom[i]
				childGeom.LocalX = contentOffX
				childGeom.LocalY = contentOffY + cursor
				cursor += childGeom.H
			}
		}

		// Annotate VRule extensions: find HRule siblings and stamp extend flags onto VRules.
		t.annotateVRuleExtensions(idx, op, cursor)

		geom.H = cursor
		if op.Border.Horizontal != 0 {
			geom.H += 2
		}
		geom.H += op.marginV()
	}

	// Store content height before any override (for flex distribution)
	geom.ContentH = geom.H

	// Explicit height overrides
	if h := op.height(); h > 0 {
		geom.H = h
	}
}

// distributeFlexGrow distributes remaining space to flex children.
// Called top-down after layout phase.
// Vertical containers (VBox) distribute height, horizontal containers (HBox) distribute width.
// distributeFlexGrow distributes remaining height to VBox flex children.
// HBox flex is handled during width distribution (single pass).
// VBox flex must happen after layout since it needs content heights.
func (t *Template) distributeFlexGrow(rootH int16) {
	// First pass: ensure root element fills screen height
	// This makes the common case "just work" without needing VBox wrappers
	if len(t.byDepth[0]) > 0 {
		for _, idx := range t.byDepth[0] {
			op := &t.ops[idx]
			geom := &t.geom[idx]
			if op.Kind == OpContainer && op.Parent == -1 {
				// Root container fills screen height (unless explicit height or FitContent)
				if op.height() == 0 && !op.FitContent {
					geom.H = rootH
				}
			}
		}
	}

	// Second pass: process depth by depth
	for depth := 0; depth <= t.maxDepth; depth++ {
		for _, idx := range t.byDepth[depth] {
			op := &t.ops[idx]

			if op.Kind == OpContainer {
				if op.IsRow {
					// HBox: stretch children to fill HBox height
					t.stretchRowChildren(idx, op)
				} else {
					// VBox: distribute vertical flex space
					t.distributeFlexInCol(idx, op, rootH)
				}
			}
		}
	}
}

// stretchRowChildren stretches HBox children to fill the HBox's height.
// This enables VBox children inside an HBox to use flex for vertical distribution.
func (t *Template) stretchRowChildren(idx int16, op *Op) {
	geom := &t.geom[idx]
	availH := geom.H - op.marginV()
	if op.Border.Horizontal != 0 {
		availH -= 2
	}

	// Stretch each child to fill the row height
	for i := op.ChildStart; i < op.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Parent != idx {
			continue
		}
		childGeom := &t.geom[i]

		// Stretch containers, layers, and VRule to fill height (unless they have explicit height)
		if childOp.Kind == OpContainer || childOp.Kind == OpLayer || childOp.Kind == OpVRule {
			if childOp.height() == 0 && childGeom.H < availH {
				childGeom.H = availH
			}
		}

		// Handle If ops - stretch their content too
		if childOp.Kind == OpIf {
			childGeom.H = availH
			t.stretchIfContent(childOp, availH)
		}
	}
}

// stretchIfContent stretches the active branch of an If to the given height.
func (t *Template) stretchIfContent(op *Op, newH int16) {
	ifExt := op.Ext.(*opIf)
	condTrue := ifExt.eval(t.elemBase)

	var tmpl *Template
	if condTrue && ifExt.thenTmpl != nil {
		tmpl = ifExt.thenTmpl
	} else if !condTrue && ifExt.elseTmpl != nil {
		tmpl = ifExt.elseTmpl
	}

	if tmpl == nil || len(tmpl.ops) == 0 {
		return
	}

	// Stretch root of sub-template
	rootOp := &tmpl.ops[0]
	if rootOp.Kind == OpContainer || rootOp.Kind == OpLayer {
		if rootOp.height() == 0 {
			tmpl.geom[0].H = newH
		}
	}
}

// distributeFlexInCol distributes vertical flex space within a column container.
func (t *Template) distributeFlexInCol(idx int16, op *Op, rootH int16) {
	geom := &t.geom[idx]

	// Calculate available height
	// If this container is a flex child, it already has its height set by parent's distribution
	// Use that height, not the parent's full height
	var availH int16
	if op.flexGrow() > 0 && geom.H > 0 {
		// This container is a flex child - use its own height (already computed)
		availH = geom.H - op.marginV()
		if op.Border.Horizontal != 0 {
			availH -= 2 // Subtract own border from available content space
		}
	} else if op.Parent >= 0 {
		parentGeom := &t.geom[op.Parent]
		parentOp := &t.ops[op.Parent]
		availH = parentGeom.H - parentOp.marginV()
		if parentOp.Border.Horizontal != 0 {
			availH -= 2 // Account for parent border
		}
	} else {
		availH = rootH - op.marginV()
		if op.Border.Horizontal != 0 {
			availH -= 2 // subtract own border from available content space
		}
	}

	// If this container has explicit height, use that
	if h := op.height(); h > 0 {
		availH = h - op.marginV()
		if op.Border.Horizontal != 0 {
			availH -= 2
		}
	}

	// Calculate used height and total flex grow (reuse scratch slices)
	var usedH int16
	var totalFlex float32
	var childCount int16
	flexChildren := t.flexScratchIdx[:0]
	flexGrowValues := t.flexScratchGrow[:0]

	for i := op.ChildStart; i < op.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Parent != idx {
			continue
		}
		childCount++

		childGeom := &t.geom[i]

		// Check for direct flex child (container, layer or spacer)
		if fg := childOp.flexGrow(); (childOp.Kind == OpContainer || childOp.Kind == OpLayer || childOp.Kind == OpSpacer) && fg > 0 {
			totalFlex += fg
			flexChildren = append(flexChildren, i)
			flexGrowValues = append(flexGrowValues, fg)
			usedH += childGeom.ContentH // Use content height for flex children
			continue
		}

		// Check for If containing a flex child in its active branch
		if childOp.Kind == OpIf {
			flexGrow := t.getIfFlexGrow(childOp)
			if flexGrow > 0 {
				totalFlex += flexGrow
				flexChildren = append(flexChildren, i)
				flexGrowValues = append(flexGrowValues, flexGrow)
				usedH += childGeom.ContentH
				continue
			}
		}

		usedH += childGeom.H
	}
	t.flexScratchIdx = flexChildren
	t.flexScratchGrow = flexGrowValues

	// Add gaps to used height
	if g := op.gap(); childCount > 1 && g > 0 {
		usedH += int16(g) * (childCount - 1)
	}

	// Distribute remaining space (handles both expansion and shrinkage)
	remaining := availH - usedH
	if remaining != 0 && totalFlex > 0 {
		distributed := int16(0)
		for i, childIdx := range flexChildren {
			childGeom := &t.geom[childIdx]
			flexShare := flexGrowValues[i] / totalFlex
			extraH := int16(float32(remaining) * flexShare)

			// Give any remainder to the last flex child (avoid rounding loss)
			if i == len(flexChildren)-1 {
				extraH = remaining - distributed
			}
			distributed += extraH
			h := childGeom.ContentH + extraH
			if h < 0 {
				h = 0
			}
			childGeom.H = h
		}

		// Recalculate child positions with new heights
		contentOffY := int16(op.Margin[0])
		if op.Border.Horizontal != 0 {
			contentOffY += 1
		}
		cursor := int16(0)
		firstChild := true

		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &t.ops[i]
			if childOp.Parent != idx {
				continue
			}

			if g := op.gap(); !firstChild && g > 0 {
				cursor += int16(g)
			}
			firstChild = false

			childGeom := &t.geom[i]
			childGeom.LocalY = contentOffY + cursor
			cursor += childGeom.H
		}

		// Propagate extra height to nested templates in If ops
		for _, childIdx := range flexChildren {
			childOp := &t.ops[childIdx]
			if childOp.Kind == OpIf {
				childGeom := &t.geom[childIdx]
				t.propagateFlexToIf(childOp, childGeom.H)
			}
		}

		// Update container height to match available
		geom.H = availH
		if op.Border.Horizontal != 0 {
			geom.H += 2
		}
	}
}

// propagateFlexToIf propagates flex height to an If's active branch template.
func (t *Template) propagateFlexToIf(op *Op, newH int16) {
	ifExt := op.Ext.(*opIf)
	condTrue := ifExt.eval(t.elemBase)

	var tmpl *Template
	if condTrue && ifExt.thenTmpl != nil {
		tmpl = ifExt.thenTmpl
	} else if !condTrue && ifExt.elseTmpl != nil {
		tmpl = ifExt.elseTmpl
	}

	if tmpl == nil || len(tmpl.ops) == 0 {
		return
	}

	// If root is a flex container, update its height and redistribute
	rootOp := &tmpl.ops[0]
	if rootOp.Kind == OpContainer && rootOp.flexGrow() > 0 {
		tmpl.geom[0].H = newH
		tmpl.distributeFlexGrow(newH)
	}
}

// getIfFlexGrow returns the FlexGrow value from an If's active branch, if any.
// This allows If-wrapped containers to participate in flex distribution.
func (t *Template) getIfFlexGrow(op *Op) float32 {
	// Determine which branch is active
	ifExt := op.Ext.(*opIf)
	condTrue := ifExt.eval(t.elemBase)

	var tmpl *Template
	if condTrue && ifExt.thenTmpl != nil {
		tmpl = ifExt.thenTmpl
	} else if !condTrue && ifExt.elseTmpl != nil {
		tmpl = ifExt.elseTmpl
	}

	if tmpl == nil || len(tmpl.ops) == 0 {
		return 0
	}

	// Check if root op of the branch is a Container with FlexGrow
	rootOp := &tmpl.ops[0]
	if fg := rootOp.flexGrow(); rootOp.Kind == OpContainer && fg > 0 {
		return fg
	}

	return 0
}

// layoutCustom handles custom layout containers using the Arranger interface.
func (t *Template) layoutCustom(idx int16, op *Op, geom *Geom) {
	clExt := op.Ext.(*opCustomLayout)
	if clExt.layout == nil {
		return
	}

	// Collect child sizes
	var childSizes []ChildSize
	for i := op.ChildStart; i < op.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Parent != idx {
			continue // not direct child
		}
		childGeom := &t.geom[i]
		childSizes = append(childSizes, ChildSize{
			MinW: int(childGeom.W),
			MinH: int(childGeom.H),
		})
	}

	// Call the layout function
	rects := clExt.layout(childSizes, int(geom.W), int(geom.H))

	// Apply positions to children
	childIdx := 0
	maxH := int16(0)
	for i := op.ChildStart; i < op.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Parent != idx {
			continue
		}
		if childIdx < len(rects) {
			r := rects[childIdx]
			t.geom[i].LocalX = int16(r.X)
			t.geom[i].LocalY = int16(r.Y)
			t.geom[i].W = int16(r.W)
			t.geom[i].H = int16(r.H)
			if int16(r.Y)+int16(r.H) > maxH {
				maxH = int16(r.Y) + int16(r.H)
			}
		}
		childIdx++
	}

	// Set container height to encompass all children
	geom.H = maxH
}

// layoutForEach iterates items, layouts each, returns total height and max width.
func (t *Template) layoutForEach(_ int16, op *Op, availW int16) (totalH, maxW int16) {
	feExt := op.Ext.(*opForEach)
	if feExt.iterTmpl == nil || feExt.slicePtr == nil {
		return 0, 0
	}

	sliceHdr := *(*sliceHeader)(feExt.slicePtr)
	if sliceHdr.Len == 0 {
		return 0, 0
	}

	// Ensure we have enough geometry slots for items
	if cap(feExt.geoms) < sliceHdr.Len {
		feExt.geoms = make([]Geom, sliceHdr.Len)
	}
	feExt.geoms = feExt.geoms[:sliceHdr.Len]

	cursor := int16(0)
	for i := 0; i < sliceHdr.Len; i++ {
		// Get element pointer for this item
		elemPtr := unsafe.Pointer(uintptr(sliceHdr.Data) + uintptr(i)*feExt.elemSize)

		// Layout sub-template for this item with element base
		feExt.iterTmpl.elemBase = elemPtr // Set element base for condition evaluation
		feExt.iterTmpl.distributeWidths(availW, elemPtr)
		feExt.iterTmpl.layout(0)
		itemH := feExt.iterTmpl.Height()

		feExt.geoms[i].LocalX = 0
		feExt.geoms[i].LocalY = cursor
		feExt.geoms[i].H = itemH
		feExt.geoms[i].W = availW

		cursor += itemH

		if availW > maxW {
			maxW = availW
		}
	}

	return cursor, maxW
}

// render draws to buffer, accumulating global positions top-down.
func (t *Template) render(buf *Buffer, globalX, globalY, maxW int16) {
	t.renderOp(buf, 0, globalX, globalY, maxW)
}

// applyTransform applies a text transform to a string.
func applyTransform(s string, transform TextTransform) string {
	switch transform {
	case TransformUppercase:
		return strings.ToUpper(s)
	case TransformLowercase:
		return strings.ToLower(s)
	case TransformCapitalize:
		// capitalize first letter of each word
		var result strings.Builder
		result.Grow(len(s))
		capitalizeNext := true
		for _, r := range s {
			if r == ' ' || r == '\t' || r == '\n' {
				capitalizeNext = true
				result.WriteRune(r)
			} else if capitalizeNext {
				result.WriteRune(unicode.ToUpper(r))
				capitalizeNext = false
			} else {
				result.WriteRune(r)
			}
		}
		return result.String()
	default:
		return s
	}
}

// effectiveStyle returns the style to use, merging with inherited style.
// If s is completely empty, returns the inherited style.
// Otherwise, cascades: Fill→BG, Attr (merged), Transform (if not set).
func (t *Template) effectiveStyle(s Style) Style {
	if t.inheritedStyle == nil && t.inheritedFill.Mode == ColorDefault {
		return s
	}
	// fully empty style inherits everything (except margin, which never cascades)
	if s.Equal(Style{}) && t.inheritedStyle != nil {
		result := *t.inheritedStyle
		result.margin = [4]int16{}
		// use cascaded Fill as BG for text rendering
		if result.BG.Mode == ColorDefault && t.inheritedFill.Mode != ColorDefault {
			result.BG = t.inheritedFill
		}
		return result
	}
	// partial style: merge inherited properties
	if t.inheritedStyle != nil {
		// merge Attr (combine both)
		s.Attr = s.Attr | t.inheritedStyle.Attr
		// inherit FG if not set
		if s.FG.Mode == ColorDefault && t.inheritedStyle.FG.Mode != ColorDefault {
			s.FG = t.inheritedStyle.FG
		}
		// inherit BG if not set (cascaded Fill may override below)
		if s.BG.Mode == ColorDefault && t.inheritedStyle.BG.Mode != ColorDefault {
			s.BG = t.inheritedStyle.BG
		}
		// inherit Transform if not set
		if s.Transform == TransformNone && t.inheritedStyle.Transform != TransformNone {
			s.Transform = t.inheritedStyle.Transform
		}
	}
	// use cascaded Fill as BG if no explicit BG
	if s.BG.Mode == ColorDefault && t.inheritedFill.Mode != ColorDefault {
		s.BG = t.inheritedFill
	}
	return s
}

func (t *Template) renderOp(buf *Buffer, idx int16, globalX, globalY, maxW int16) {
	if idx < 0 || int(idx) >= len(t.ops) {
		return
	}

	op := &t.ops[idx]
	geom := &t.geom[idx]

	// Compute absolute position
	absX := globalX + geom.LocalX
	absY := globalY + geom.LocalY

	// generic margin offset for non-container ops (containers handle margin themselves)
	if op.Kind != OpContainer && op.marginH()+op.marginV() > 0 {
		absX += op.Margin[3] // left
		absY += op.Margin[0] // top
		maxW -= op.marginH()
	}

	// content dimensions exclude margin (for ops without margin, marginH/V == 0)
	contentW := geom.W - op.marginH()
	contentH := geom.H - op.marginV()

	switch op.Kind {
	case OpText:
		ext := op.Ext.(*opText)
		style := t.effectiveStyle(ext.style)
		var raw string
		if ext.mode == textFn {
			raw = ext.fn()
		} else {
			raw = ext.resolve(t.elemBase)
		}
		text := applyTransform(raw, style.Transform)
		x := int(absX)
		if style.Align != AlignLeft {
			alignW := op.width()
			if alignW == 0 {
				alignW = maxW
			}
			x += alignOffset(text, int(alignW), style.Align)
		}
		buf.WriteStringFast(x, int(absY), text, style, int(maxW))

	case OpProgress:
		ext := op.Ext.(*opProgress)
		ratio := float32(ext.resolve(t.elemBase)) / 100.0
		style := t.effectiveStyle(ext.style)
		buf.WriteProgressBar(int(absX), int(absY), int(op.width()), ratio, style)

	case OpRichText:
		ext := op.Ext.(*opRichText)
		spans := ext.resolve(t.elemBase)
		if spans != nil {
			buf.WriteSpans(int(absX), int(absY), spans, int(maxW))
		}

	case OpLeader:
		ext := op.Ext.(*opLeader)
		width := int(op.width())
		if width == 0 {
			width = int(maxW)
		}
		style := t.effectiveStyle(ext.style)
		label := applyTransform(ext.label, style.Transform)
		var value string
		switch ext.mode {
		case leaderPtr:
			value = applyTransform(*ext.valuePtr, style.Transform)
		case leaderIntPtr:
			var scratch [20]byte
			b := strconv.AppendInt(scratch[:0], int64(*ext.intPtr), 10)
			value = applyTransform(unsafe.String(&b[0], len(b)), style.Transform)
		case leaderFloatPtr:
			var scratch [32]byte
			b := strconv.AppendFloat(scratch[:0], *ext.floatPtr, 'f', 1, 64)
			value = applyTransform(unsafe.String(&b[0], len(b)), style.Transform)
		default:
			value = applyTransform(ext.value, style.Transform)
		}
		buf.WriteLeader(int(absX), int(absY), label, value, width, ext.fill, style)

	case OpCounter:
		ext := op.Ext.(*opCounter)
		style := t.effectiveStyle(ext.style)
		var scratch [48]byte
		var b []byte
		prefix := ext.prefix
		if ext.streamingPtr != nil && *ext.streamingPtr && len(prefix) > 0 {
			frame := int(atomic.LoadInt32(ext.framePtr))
			b = append(scratch[:0], SpinnerCircle[frame%len(SpinnerCircle)]...)
			b = append(b, prefix[1:]...)
		} else {
			b = append(scratch[:0], prefix...)
		}
		b = strconv.AppendInt(b, int64(*ext.currentPtr), 10)
		b = append(b, '/')
		b = strconv.AppendInt(b, int64(*ext.totalPtr), 10)
		text := unsafe.String(&b[0], len(b))
		buf.WriteStringFast(int(absX), int(absY), text, style, int(maxW))

	case OpAutoTable:
		t.renderAutoTable(buf, op, absX, absY, maxW)

	case OpTable:
		t.renderTable(buf, op, absX, absY, maxW)

	case OpSparkline:
		op.Ext.(*opSparkline).render(t, buf, absX, absY, contentW, geom.H)

	case OpHRule:
		ext := op.Ext.(*opRule)
		width := int(maxW)
		if contentW > 0 {
			width = int(contentW)
		}
		ruleStyle := t.effectiveStyle(ext.style)
		for i := 0; i < width; i++ {
			buf.Set(int(absX)+i, int(absY), Cell{Rune: ext.char, Style: ruleStyle})
		}
		if ext.extend && ext.vruleX != 0 {
			delta := int(ext.vruleX)
			if delta > 0 {
				for i := width; i <= delta; i++ {
					r := ext.char
					if i == delta {
						r = '╴'
					}
					buf.Set(int(absX)+i, int(absY), Cell{Rune: r, Style: ruleStyle})
				}
			} else {
				for i := delta; i < 0; i++ {
					r := ext.char
					if i == delta {
						r = '╶'
					}
					buf.Set(int(absX)+i, int(absY), Cell{Rune: r, Style: ruleStyle})
				}
			}
		}
		if ext.extend && ext.vruleX2 != 0 {
			delta := int(ext.vruleX2)
			if delta > 0 {
				for i := width; i <= delta; i++ {
					r := ext.char
					if i == delta {
						r = '╴'
					}
					buf.Set(int(absX)+i, int(absY), Cell{Rune: r, Style: ruleStyle})
				}
			} else {
				for i := delta; i < 0; i++ {
					r := ext.char
					if i == delta {
						r = '╶'
					}
					buf.Set(int(absX)+i, int(absY), Cell{Rune: r, Style: ruleStyle})
				}
			}
		}
		if ext.extendLeft > 0 {
			n := int(ext.extendLeft)
			buf.Set(int(absX)-n, int(absY), Cell{Rune: '╶', Style: ruleStyle})
			for i := 1; i < n; i++ {
				buf.Set(int(absX)-i, int(absY), Cell{Rune: ext.char, Style: ruleStyle})
			}
		}
		if ext.extendRight > 0 {
			n := int(ext.extendRight)
			buf.Set(int(absX)+width+n-1, int(absY), Cell{Rune: '╴', Style: ruleStyle})
			for i := 0; i < n-1; i++ {
				buf.Set(int(absX)+width+i, int(absY), Cell{Rune: ext.char, Style: ruleStyle})
			}
		}

	case OpVRule:
		ext := op.Ext.(*opRule)
		ruleStyle := t.effectiveStyle(ext.style)
		for i := 0; i < int(contentH); i++ {
			buf.Set(int(absX), int(absY)+i, Cell{Rune: ext.char, Style: ruleStyle})
		}
		if ext.extendTop {
			buf.Set(int(absX), int(absY)-1, Cell{Rune: '╷', Style: ruleStyle})
		}
		if ext.extendBot {
			buf.Set(int(absX), int(absY)+int(contentH), Cell{Rune: '╵', Style: ruleStyle})
		}

	case OpSpacer:
		ext := op.Ext.(*opRule)
		if ext.char != 0 {
			for x := int16(0); x < contentW; x++ {
				buf.Set(int(absX+x), int(absY), Cell{Rune: ext.char, Style: ext.style})
			}
		}

	case OpSpinner:
		ext := op.Ext.(*opSpinner)
		if ext.framePtr != nil && len(ext.frames) > 0 {
			frameIdx := *ext.framePtr % len(ext.frames)
			frame := ext.frames[frameIdx]
			style := t.effectiveStyle(ext.style)
			buf.WriteStringFast(int(absX), int(absY), frame, style, 1)
		}

	case OpScrollbar:
		t.renderScrollbar(buf, op, geom, absX, absY)

	case OpTabs:
		t.renderTabs(buf, op, geom, absX, absY)

	case OpTreeView:
		t.renderTreeView(buf, op, absX, absY)

	case OpSelectionList:
		t.renderSelectionList(buf, op, geom, absX, absY, maxW)

	case OpJump:
		t.renderJump(buf, op, geom, absX, absY, maxW, idx)

	case OpTextInput:
		t.renderTextInput(buf, op, geom, absX, absY)

	case OpOverlay:
		t.pendingOverlays = append(t.pendingOverlays, pendingOverlay{op: op})

	case OpScreenEffect:
		ext := op.Ext.(*opScreenEffect)
		t.pendingScreenEffects = append(t.pendingScreenEffects, ext.fns...)

	case OpCustom:
		crExt := op.Ext.(*opCustomRenderer)
		if crExt.renderer != nil {
			crExt.renderer.Render(buf, int(absX), int(absY), int(contentW), int(contentH))
		}

	case OpLayout:
		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &t.ops[i]
			if childOp.Parent != idx {
				continue
			}
			t.renderOp(buf, i, absX, absY, contentW)
		}

	case OpLayer:
		ext := op.Ext.(*opLayer)
		if ext.ptr != nil {
			layerW := int(contentW)
			if ext.width > 0 {
				layerW = int(ext.width)
			}
			ext.ptr.SetViewport(layerW, int(contentH))
			ext.ptr.screenX = int(absX)
			ext.ptr.screenY = int(absY)
			ext.ptr.prepare()
			ext.ptr.blit(buf, int(absX), int(absY), layerW, int(contentH))

			if ext.ptr.cursor.Visible && t.app != nil {
				t.app.activeLayer = ext.ptr
			}
		}

	case OpContainer:
		// Margin inset: visible box starts inside the margin
		boxX := absX + op.Margin[3] // left margin
		boxY := absY + op.Margin[0] // top margin
		boxW := geom.W - op.marginH()
		boxH := geom.H - op.marginV()

		if op.NodeRef != nil {
			op.NodeRef.X = int(boxX)
			op.NodeRef.Y = int(boxY)
			op.NodeRef.W = int(boxW)
			op.NodeRef.H = int(boxH)
		}

		// Update inherited Fill - cascades through nested containers
		oldInheritedFill := t.inheritedFill
		if op.CascadeStyle != nil && op.CascadeStyle.Fill.Mode != ColorDefault {
			t.inheritedFill = op.CascadeStyle.Fill
		} else if op.Fill.Mode != ColorDefault {
			t.inheritedFill = op.Fill
		}

		// Update inherited style if this container sets one (before title rendering)
		oldInheritedStyle := t.inheritedStyle
		if op.CascadeStyle != nil {
			t.inheritedStyle = op.CascadeStyle
		}

		// Fill container area - direct Fill takes precedence over inherited
		fillColor := t.inheritedFill
		if op.Fill.Mode != ColorDefault {
			fillColor = op.Fill
		}
		if fillColor.Mode != ColorDefault {
			fillCell := Cell{Rune: ' ', Style: Style{BG: fillColor}}
			buf.FillRect(int(boxX), int(boxY), int(boxW), int(boxH), fillCell)
		}

		// Draw border if present
		if op.Border.Horizontal != 0 {
			style := DefaultStyle()
			if op.BorderFG != nil {
				style.FG = *op.BorderFG
			}
			if op.BorderBG != nil {
				style.BG = *op.BorderBG
			} else if fillColor.Mode != ColorDefault {
				style.BG = fillColor
			}
			buf.DrawBorder(int(boxX), int(boxY), int(boxW), int(boxH), op.Border, style)

			if op.Title != "" {
				titleTransform := TransformNone
				if t.inheritedStyle != nil {
					titleTransform = t.inheritedStyle.Transform
				}
				titleMaxW := int(boxW) - 2
				titleX := int(boxX) + 1
				if titleMaxW > 0 {
					buf.SetFast(titleX, int(boxY), Cell{Rune: op.Border.Horizontal, Style: style})
					titleX++
					buf.SetFast(titleX, int(boxY), Cell{Rune: ' ', Style: style})
					titleX++
					title := applyTransform(op.Title, titleTransform)
					titleW := utf8.RuneCountInString(title)
					availTitleW := titleMaxW - 3 // border char + space before + space after
					if availTitleW > 0 {
						if titleW > availTitleW {
							titleW = availTitleW
						}
						buf.WriteStringFast(titleX, int(boxY), title, style, titleW)
						titleX += titleW
						buf.SetFast(titleX, int(boxY), Cell{Rune: ' ', Style: style})
					}
				}
			}
		}

		// Calculate content width (accounting for margin + border)
		contentW := boxW
		if op.Border.Horizontal != 0 {
			contentW -= 2
		}

		// Set vertical clip for children (content area bottom)
		oldClipMaxY := t.clipMaxY
		contentBottom := boxY + boxH
		if op.Border.Horizontal != 0 {
			contentBottom -= 1 // don't render into bottom border
		}
		if t.clipMaxY == 0 || contentBottom < t.clipMaxY {
			t.clipMaxY = contentBottom
		}

		// Render children with this container's position as their origin
		// children's LocalX/Y already include margin+border offsets from layoutContainer
		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &t.ops[i]
			if childOp.Parent != idx {
				continue
			}
			t.renderOp(buf, i, absX, absY, contentW)
		}

		// Restore inherited style, fill, and clip
		t.inheritedStyle = oldInheritedStyle
		t.inheritedFill = oldInheritedFill
		t.clipMaxY = oldClipMaxY

	case OpIf:
		// Render active branch if condition is true
		ifExt := op.Ext.(*opIf)
		condTrue := ifExt.evalStatic()
		if ifExt.thenTmpl != nil && condTrue {
			ifExt.thenTmpl.app = t.app
			ifExt.thenTmpl.inheritedStyle = t.inheritedStyle // propagate inherited style
			ifExt.thenTmpl.inheritedFill = t.inheritedFill   // propagate inherited fill
			ifExt.thenTmpl.clipMaxY = t.clipMaxY             // propagate vertical clip
			ifExt.thenTmpl.elemBase = t.elemBase             // propagate for offset-based text inside branch templates
			ifExt.thenTmpl.pendingOverlays = ifExt.thenTmpl.pendingOverlays[:0]
			ifExt.thenTmpl.pendingScreenEffects = ifExt.thenTmpl.pendingScreenEffects[:0]
			ifExt.thenTmpl.render(buf, absX, absY, geom.W)
			t.pendingOverlays = append(t.pendingOverlays, ifExt.thenTmpl.pendingOverlays...)
			t.pendingScreenEffects = append(t.pendingScreenEffects, ifExt.thenTmpl.pendingScreenEffects...)
		} else if ifExt.elseTmpl != nil && !condTrue {
			ifExt.elseTmpl.app = t.app
			ifExt.elseTmpl.inheritedStyle = t.inheritedStyle // propagate inherited style
			ifExt.elseTmpl.inheritedFill = t.inheritedFill   // propagate inherited fill
			ifExt.elseTmpl.clipMaxY = t.clipMaxY             // propagate vertical clip
			ifExt.elseTmpl.elemBase = t.elemBase             // propagate for offset-based text inside branch templates
			ifExt.elseTmpl.pendingOverlays = ifExt.elseTmpl.pendingOverlays[:0]
			ifExt.elseTmpl.pendingScreenEffects = ifExt.elseTmpl.pendingScreenEffects[:0]
			ifExt.elseTmpl.render(buf, absX, absY, geom.W)
			t.pendingOverlays = append(t.pendingOverlays, ifExt.elseTmpl.pendingOverlays...)
			t.pendingScreenEffects = append(t.pendingScreenEffects, ifExt.elseTmpl.pendingScreenEffects...)
		}

	case OpForEach:
		// Render each item using iterGeoms for positioning
		feExt := op.Ext.(*opForEach)
		if feExt.iterTmpl == nil || feExt.slicePtr == nil {
			return
		}
		sliceHdr := *(*sliceHeader)(feExt.slicePtr)
		if sliceHdr.Len == 0 {
			return
		}

		for i := 0; i < sliceHdr.Len && i < len(feExt.geoms); i++ {
			itemGeom := &feExt.geoms[i]
			itemAbsX := absX + itemGeom.LocalX
			itemAbsY := absY + itemGeom.LocalY

			// Rebind template ops to this element's data
			elemPtr := unsafe.Pointer(uintptr(sliceHdr.Data) + uintptr(i)*feExt.elemSize)
			t.renderSubTemplate(buf, feExt.iterTmpl, itemAbsX, itemAbsY, itemGeom.W, elemPtr)
		}

	case OpSwitch:
		// Render matching case template
		swExt := op.Ext.(*opSwitch)
		var tmpl *Template
		matchIdx := swExt.node.getMatchIndexWithBase(t.elemBase)
		if matchIdx >= 0 && matchIdx < len(swExt.cases) {
			tmpl = swExt.cases[matchIdx]
		} else {
			tmpl = swExt.def
		}
		if tmpl != nil {
			tmpl.clipMaxY = t.clipMaxY           // propagate vertical clip
			tmpl.inheritedFill = t.inheritedFill // propagate fill so blank cells use parent bg
			tmpl.elemBase = t.elemBase           // propagate for offset-based text inside case templates
			tmpl.render(buf, absX, absY, geom.W)
		}
	}
}

// renderSubTemplate renders a sub-template (for ForEach) with element-bound data.
func (t *Template) renderSubTemplate(buf *Buffer, sub *Template, globalX, globalY, maxW int16, elemBase unsafe.Pointer) {
	sub.app = t.app
	sub.clipMaxY = t.clipMaxY       // propagate vertical clip
	sub.inheritedFill = t.inheritedFill // propagate fill so blank cells use parent bg
	sub.elemBase = elemBase         // ensure renderOp paths (e.g. via renderJump) see the correct element
	for i := range sub.ops {
		if sub.ops[i].Parent == -1 {
			sub.renderSubOp(buf, int16(i), globalX, globalY, maxW, elemBase)
		}
	}
}

// renderSubOp renders a single op in a sub-template, recursing into children.
func (sub *Template) renderSubOp(buf *Buffer, idx int16, globalX, globalY, maxW int16, elemBase unsafe.Pointer) {
	op := &sub.ops[idx]
	geom := &sub.geom[idx]

	absX := globalX + geom.LocalX
	absY := globalY + geom.LocalY

	// generic margin offset for non-container ops
	if op.Kind != OpContainer && op.marginH()+op.marginV() > 0 {
		absX += op.Margin[3]
		absY += op.Margin[0]
		maxW -= op.marginH()
	}

	// content dimensions exclude margin
	contentW := geom.W - op.marginH()
	contentH := geom.H - op.marginV()

	// Helper to merge row background with text style (also applies inherited style)
	mergeStyle := func(s Style) Style {
		s = sub.effectiveStyle(s) // apply inherited style first
		if sub.rowBG.Mode != 0 && s.BG.Mode == 0 {
			s.BG = sub.rowBG
		}
		return s
	}

	switch op.Kind {
	case OpText:
		ext := op.Ext.(*opText)
		style := mergeStyle(ext.style)
		var raw string
		if ext.mode == textFn {
			raw = ext.fn()
		} else {
			raw = ext.resolve(elemBase)
		}
		text := applyTransform(raw, style.Transform)
		buf.WriteStringFast(int(absX), int(absY), text, style, int(maxW))

	case OpProgress:
		ext := op.Ext.(*opProgress)
		ratio := float32(ext.resolve(elemBase)) / 100.0
		style := sub.effectiveStyle(ext.style)
		buf.WriteProgressBar(int(absX), int(absY), int(op.width()), ratio, style)

	case OpRichText:
		ext := op.Ext.(*opRichText)
		spans := ext.resolve(elemBase)
		if spans != nil {
			buf.WriteSpans(int(absX), int(absY), spans, int(maxW))
		}

	case OpLeader:
		ext := op.Ext.(*opLeader)
		width := int(op.width())
		if width == 0 {
			width = int(maxW)
		}
		style := sub.effectiveStyle(ext.style)
		label := applyTransform(ext.label, style.Transform)
		var value string
		switch ext.mode {
		case leaderPtr:
			value = applyTransform(*ext.valuePtr, style.Transform)
		case leaderIntPtr:
			var scratch [20]byte
			b := strconv.AppendInt(scratch[:0], int64(*ext.intPtr), 10)
			value = applyTransform(unsafe.String(&b[0], len(b)), style.Transform)
		case leaderFloatPtr:
			var scratch [32]byte
			b := strconv.AppendFloat(scratch[:0], *ext.floatPtr, 'f', 1, 64)
			value = applyTransform(unsafe.String(&b[0], len(b)), style.Transform)
		default:
			value = applyTransform(ext.value, style.Transform)
		}
		buf.WriteLeader(int(absX), int(absY), label, value, width, ext.fill, style)

	case OpCounter:
		ext := op.Ext.(*opCounter)
		style := sub.effectiveStyle(ext.style)
		var scratch [48]byte
		var b []byte
		prefix := ext.prefix
		if ext.streamingPtr != nil && *ext.streamingPtr && len(prefix) > 0 {
			frame := int(atomic.LoadInt32(ext.framePtr))
			b = append(scratch[:0], SpinnerCircle[frame%len(SpinnerCircle)]...)
			b = append(b, prefix[1:]...)
		} else {
			b = append(scratch[:0], prefix...)
		}
		b = strconv.AppendInt(b, int64(*ext.currentPtr), 10)
		b = append(b, '/')
		b = strconv.AppendInt(b, int64(*ext.totalPtr), 10)
		text := unsafe.String(&b[0], len(b))
		buf.WriteStringFast(int(absX), int(absY), text, style, int(maxW))

	case OpTable:
		sub.renderTable(buf, op, absX, absY, maxW)

	case OpSparkline:
		op.Ext.(*opSparkline).render(sub, buf, absX, absY, contentW, geom.H)

	case OpHRule:
		ext := op.Ext.(*opRule)
		width := int(maxW)
		if contentW > 0 {
			width = int(contentW)
		}
		ruleStyle := sub.effectiveStyle(ext.style)
		for i := 0; i < width; i++ {
			buf.Set(int(absX)+i, int(absY), Cell{Rune: ext.char, Style: ruleStyle})
		}

	case OpVRule:
		ext := op.Ext.(*opRule)
		ruleStyle := sub.effectiveStyle(ext.style)
		for i := 0; i < int(contentH); i++ {
			buf.Set(int(absX), int(absY)+i, Cell{Rune: ext.char, Style: ruleStyle})
		}

	case OpSpacer:
		ext := op.Ext.(*opRule)
		spacerStyle := mergeStyle(ext.style)
		if ext.char != 0 {
			for x := int16(0); x < contentW; x++ {
				buf.Set(int(absX+x), int(absY), Cell{Rune: ext.char, Style: spacerStyle})
			}
		} else if sub.rowBG.Mode != 0 {
			for x := int16(0); x < contentW; x++ {
				buf.Set(int(absX+x), int(absY), Cell{Rune: ' ', Style: spacerStyle})
			}
		}

	case OpSpinner:
		ext := op.Ext.(*opSpinner)
		if ext.framePtr != nil && len(ext.frames) > 0 {
			frameIdx := *ext.framePtr % len(ext.frames)
			frame := ext.frames[frameIdx]
			style := sub.effectiveStyle(ext.style)
			buf.WriteStringFast(int(absX), int(absY), frame, style, 1)
		}

	case OpScrollbar:
		sub.renderScrollbar(buf, op, geom, absX, absY)

	case OpTabs:
		sub.renderTabs(buf, op, geom, absX, absY)

	case OpTreeView:
		sub.renderTreeView(buf, op, absX, absY)

	case OpSelectionList:
		sub.renderSelectionList(buf, op, geom, absX, absY, maxW)

	case OpJump:
		sub.renderJump(buf, op, geom, absX, absY, maxW, idx)

	case OpTextInput:
		sub.renderTextInput(buf, op, geom, absX, absY)

	case OpOverlay:
		sub.pendingOverlays = append(sub.pendingOverlays, pendingOverlay{op: op})

	case OpScreenEffect:
		ext := op.Ext.(*opScreenEffect)
		sub.pendingScreenEffects = append(sub.pendingScreenEffects, ext.fns...)

	case OpCustom:
		crExt := op.Ext.(*opCustomRenderer)
		if crExt.renderer != nil {
			crExt.renderer.Render(buf, int(absX), int(absY), int(contentW), int(contentH))
		}

	case OpLayout:
		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &sub.ops[i]
			if childOp.Parent != idx {
				continue
			}
			sub.renderSubOp(buf, i, absX, absY, contentW, elemBase)
		}

	case OpLayer:
		ext := op.Ext.(*opLayer)
		if ext.ptr != nil {
			layerW := int(contentW)
			if ext.width > 0 {
				layerW = int(ext.width)
			}
			ext.ptr.SetViewport(layerW, int(contentH))
			ext.ptr.screenX = int(absX)
			ext.ptr.screenY = int(absY)
			ext.ptr.prepare()
			ext.ptr.blit(buf, int(absX), int(absY), layerW, int(contentH))

			if ext.ptr.cursor.Visible && sub.app != nil {
				sub.app.activeLayer = ext.ptr
			}
		}

	case OpContainer:
		// Margin inset: visible box starts inside the margin
		boxX := absX + op.Margin[3]
		boxY := absY + op.Margin[0]
		boxW := geom.W - op.marginH()
		boxH := geom.H - op.marginV()

		// Update inherited Fill - cascades through nested containers
		oldInheritedFill := sub.inheritedFill
		if op.CascadeStyle != nil && op.CascadeStyle.Fill.Mode != ColorDefault {
			sub.inheritedFill = op.CascadeStyle.Fill
		} else if op.Fill.Mode != ColorDefault {
			sub.inheritedFill = op.Fill
		}

		// Update inherited style if this container sets one (before title rendering)
		oldInheritedStyle := sub.inheritedStyle
		if op.CascadeStyle != nil {
			sub.inheritedStyle = op.CascadeStyle
		}

		// Fill container area - direct Fill takes precedence over inherited
		fillColor := sub.inheritedFill
		if op.Fill.Mode != ColorDefault {
			fillColor = op.Fill
		}
		if fillColor.Mode != ColorDefault {
			fillCell := Cell{Rune: ' ', Style: Style{BG: fillColor}}
			buf.FillRect(int(boxX), int(boxY), int(boxW), int(boxH), fillCell)
		}

		// Draw border if present
		if op.Border.Horizontal != 0 {
			style := DefaultStyle()
			if op.BorderFG != nil {
				style.FG = *op.BorderFG
			}
			if op.BorderBG != nil {
				style.BG = *op.BorderBG
			} else if fillColor.Mode != ColorDefault {
				style.BG = fillColor
			}
			buf.DrawBorder(int(boxX), int(boxY), int(boxW), int(boxH), op.Border, style)

			if op.Title != "" {
				titleTransform := TransformNone
				if sub.inheritedStyle != nil {
					titleTransform = sub.inheritedStyle.Transform
				}
				titleMaxW := int(boxW) - 2
				titleX := int(boxX) + 1
				if titleMaxW > 0 {
					buf.SetFast(titleX, int(boxY), Cell{Rune: op.Border.Horizontal, Style: style})
					titleX++
					buf.SetFast(titleX, int(boxY), Cell{Rune: ' ', Style: style})
					titleX++
					title := applyTransform(op.Title, titleTransform)
					titleW := utf8.RuneCountInString(title)
					availTitleW := titleMaxW - 3 // border char + space before + space after
					if availTitleW > 0 {
						if titleW > availTitleW {
							titleW = availTitleW
						}
						buf.WriteStringFast(titleX, int(boxY), title, style, titleW)
						titleX += titleW
						buf.SetFast(titleX, int(boxY), Cell{Rune: ' ', Style: style})
					}
				}
			}
		}

		// Calculate content width (accounting for margin + border)
		contentW := boxW
		if op.Border.Horizontal != 0 {
			contentW -= 2
		}

		// Recurse into children with this container's position as their origin
		// children's LocalX/Y already include margin+border offsets
		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &sub.ops[i]
			if childOp.Parent != idx {
				continue
			}
			sub.renderSubOp(buf, i, absX, absY, contentW, elemBase)
		}

		// Restore inherited style and fill
		sub.inheritedStyle = oldInheritedStyle
		sub.inheritedFill = oldInheritedFill

	case OpIf:
		// Use evaluateWithBase for conditions inside ForEach
		ifExt := op.Ext.(*opIf)
		condTrue := ifExt.eval(elemBase)
		if ifExt.thenTmpl != nil && condTrue {
			ifExt.thenTmpl.inheritedStyle = sub.inheritedStyle // propagate inherited style
			ifExt.thenTmpl.inheritedFill = sub.inheritedFill   // propagate inherited fill
			sub.renderSubTemplate(buf, ifExt.thenTmpl, absX, absY, geom.W, elemBase)
		} else if ifExt.elseTmpl != nil && !condTrue {
			ifExt.elseTmpl.inheritedStyle = sub.inheritedStyle // propagate inherited style
			ifExt.elseTmpl.inheritedFill = sub.inheritedFill   // propagate inherited fill
			sub.renderSubTemplate(buf, ifExt.elseTmpl, absX, absY, geom.W, elemBase)
		}

	case OpForEach:
		// Nested ForEach - render with nested element base
		feExt := op.Ext.(*opForEach)
		if feExt.iterTmpl != nil && feExt.slicePtr != nil {
			sliceHdr := *(*sliceHeader)(feExt.slicePtr)
			for j := 0; j < sliceHdr.Len && j < len(feExt.geoms); j++ {
				itemGeom := &feExt.geoms[j]
				itemAbsX := absX + itemGeom.LocalX
				itemAbsY := absY + itemGeom.LocalY
				nestedElemPtr := unsafe.Pointer(uintptr(sliceHdr.Data) + uintptr(j)*feExt.elemSize)
				sub.renderSubTemplate(buf, feExt.iterTmpl, itemAbsX, itemAbsY, itemGeom.W, nestedElemPtr)
			}
		}

	case OpSwitch:
		// Render matching case template within ForEach context
		swExt := op.Ext.(*opSwitch)
		var tmpl *Template
		matchIdx := swExt.node.getMatchIndexWithBase(elemBase)
		if matchIdx >= 0 && matchIdx < len(swExt.cases) {
			tmpl = swExt.cases[matchIdx]
		} else {
			tmpl = swExt.def
		}
		if tmpl != nil {
			sub.renderSubTemplate(buf, tmpl, absX, absY, geom.W, elemBase)
		}
	}
}

// renderSelectionList renders a selection list with marker and windowing.
func (t *Template) renderSelectionList(buf *Buffer, op *Op, geom *Geom, absX, absY, maxW int16) {
	ext := op.Ext.(*opSelectionList)
	sliceHdr := *(*sliceHeader)(ext.slicePtr)
	if sliceHdr.Len == 0 {
		return
	}

	selectedIdx := -1
	if ext.selectedPtr != nil {
		selectedIdx = *ext.selectedPtr
	}

	startIdx := 0
	endIdx := sliceHdr.Len
	if ext.listPtr != nil && ext.listPtr.MaxVisible > 0 {
		startIdx = ext.listPtr.offset
		endIdx = startIdx + ext.listPtr.MaxVisible
		if endIdx > sliceHdr.Len {
			endIdx = sliceHdr.Len
		}
	}

	if t.clipMaxY > 0 {
		availableRows := int(t.clipMaxY - absY)
		if availableRows <= 0 {
			return
		}
		if endIdx-startIdx > availableRows {
			endIdx = startIdx + availableRows
		}
		if ext.listPtr != nil && selectedIdx >= 0 {
			effectiveVisible := endIdx - startIdx
			if selectedIdx < startIdx {
				startIdx = selectedIdx
				endIdx = startIdx + effectiveVisible
				if endIdx > sliceHdr.Len {
					endIdx = sliceHdr.Len
				}
				ext.listPtr.offset = startIdx
			} else if selectedIdx >= startIdx+effectiveVisible {
				startIdx = selectedIdx - effectiveVisible + 1
				if startIdx < 0 {
					startIdx = 0
				}
				endIdx = startIdx + effectiveVisible
				if endIdx > sliceHdr.Len {
					endIdx = sliceHdr.Len
				}
				ext.listPtr.offset = startIdx
			}
		}
	}

	spaces := ext.markerSpaces

	contentW := int16(maxW) - ext.markerWidth
	contentX := absX + ext.markerWidth

	needsFullPipeline := false
	if ext.iterTmpl != nil && len(ext.iterTmpl.ops) > 0 {
		firstOp := &ext.iterTmpl.ops[0]
		needsFullPipeline = firstOp.Kind == OpContainer || firstOp.Kind == OpLayout || firstOp.Kind == OpJump
	}

	var defaultStyle, selectedStyle, markerBaseStyle Style
	if ext.listPtr != nil {
		defaultStyle = ext.listPtr.Style
		selectedStyle = ext.listPtr.SelectedStyle
		markerBaseStyle = ext.listPtr.MarkerStyle
	}

	// Render visible items
	y := int(absY)
	for i := startIdx; i < endIdx; i++ {
		isSelected := i == selectedIdx

		// Fill background for row
		var rowBG Color
		if isSelected && selectedStyle.BG.Mode != 0 {
			rowBG = selectedStyle.BG
		} else if defaultStyle.BG.Mode != 0 {
			rowBG = defaultStyle.BG
		}
		if rowBG.Mode != 0 {
			buf.FillRect(int(absX), y, int(maxW), 1, Cell{Rune: ' ', Style: Style{BG: rowBG}})
		}

		// Determine marker text and style
		var markerText string
		markerStyle := markerBaseStyle
		if isSelected {
			markerText = ext.marker
			// Merge: use MarkerStyle but inherit SelectedStyle background if marker has none
			if markerStyle.BG.Mode == 0 && selectedStyle.BG.Mode != 0 {
				markerStyle.BG = selectedStyle.BG
			}
			// Also inherit foreground from SelectedStyle if MarkerStyle has none
			if markerStyle.FG.Mode == 0 && selectedStyle.FG.Mode != 0 {
				markerStyle.FG = selectedStyle.FG
			}
		} else {
			markerText = spaces
			// For non-selected rows, inherit from default style
			if markerStyle.BG.Mode == 0 && defaultStyle.BG.Mode != 0 {
				markerStyle.BG = defaultStyle.BG
			}
		}

		// Write marker first
		buf.WriteStringFast(int(absX), y, markerText, t.effectiveStyle(markerStyle), int(maxW))

		// Get content from iteration template
		if ext.iterTmpl != nil && len(ext.iterTmpl.ops) > 0 {
			elemPtr := unsafe.Pointer(uintptr(sliceHdr.Data) + uintptr(i)*ext.elemSize)

			if needsFullPipeline {
				// Complex layout: do full width distribution, layout, and render
				ext.iterTmpl.elemBase = elemPtr
				ext.iterTmpl.distributeWidths(contentW, elemPtr)
				ext.iterTmpl.layout(0)
				// Set row background (used by renderSubOp)
				if isSelected && selectedStyle.BG.Mode != 0 {
					ext.iterTmpl.rowBG = selectedStyle.BG
				} else if defaultStyle.BG.Mode != 0 {
					ext.iterTmpl.rowBG = defaultStyle.BG
				} else {
					ext.iterTmpl.rowBG = Color{}
				}
				t.renderSubTemplate(buf, ext.iterTmpl, contentX, int16(y), contentW, elemPtr)
			} else {
				// Simple text: fast path (no layout needed)
				iterOp := &ext.iterTmpl.ops[0]

				switch iterOp.Kind {
				case OpText:
					ext := iterOp.Ext.(*opText)
					textStyle := ext.style
					if textStyle.BG.Mode == 0 {
						if isSelected && selectedStyle.BG.Mode != 0 {
							textStyle.BG = selectedStyle.BG
						} else if defaultStyle.BG.Mode != 0 {
							textStyle.BG = defaultStyle.BG
						}
					}
					effStyle := t.effectiveStyle(textStyle)
					var raw string
					if ext.mode == textFn {
						raw = ext.fn()
					} else {
						raw = ext.resolve(elemPtr)
					}
					txt := applyTransform(raw, effStyle.Transform)
					buf.WriteStringFast(int(contentX), y, txt, effStyle, int(contentW))
				case OpRichText:
					ext := iterOp.Ext.(*opRichText)
					spans := ext.resolve(elemPtr)
					if spans != nil {
						buf.WriteSpans(int(contentX), y, spans, int(contentW))
					}
				}
			}
		}
		y++
	}
}

// treeVisibleCount returns the number of visible nodes in the tree.
func (t *Template) treeVisibleCount(node *TreeNode, includeRoot bool) int {
	if node == nil {
		return 0
	}
	count := 0
	if includeRoot {
		count = 1
	}
	if node.Expanded || !includeRoot {
		for _, child := range node.Children {
			count += t.treeVisibleCount(child, true)
		}
	}
	return count
}

// treeMaxWidth returns the maximum width of visible nodes.
func (t *Template) treeMaxWidth(node *TreeNode, level, indent int, includeRoot bool) int {
	if node == nil {
		return 0
	}
	maxW := 0
	if includeRoot && level >= 0 {
		// 2 for indicator + space, then indent + label
		lineW := 2 + level*indent + utf8.RuneCountInString(node.Label)
		if lineW > maxW {
			maxW = lineW
		}
	}
	if node.Expanded || !includeRoot {
		for _, child := range node.Children {
			childW := t.treeMaxWidth(child, level+1, indent, true)
			if childW > maxW {
				maxW = childW
			}
		}
	}
	return maxW
}

func (t *Template) renderTreeView(buf *Buffer, op *Op, absX, absY int16) {
	ext := op.Ext.(*opTreeView)
	if ext.root == nil {
		return
	}
	y := int(absY)
	t.renderTreeNode(buf, ext, ext.root, int(absX), &y, 0, ext.showRoot, nil)
}

func (t *Template) renderTreeNode(buf *Buffer, ext *opTreeView, node *TreeNode, x int, y *int, level int, render bool, linePrefix []bool) {
	if node == nil {
		return
	}

	if render && level >= 0 {
		posX := x
		if ext.showLines && level > 0 {
			for i := 0; i < level; i++ {
				if i < len(linePrefix) && linePrefix[i] {
					buf.Set(posX, *y, Cell{Rune: '│', Style: ext.style})
				}
				posX += ext.indent
			}
		} else {
			posX += level * ext.indent
		}

		var indicator rune
		if len(node.Children) > 0 {
			if node.Expanded {
				indicator = ext.expandedChar
			} else {
				indicator = ext.collapsedChar
			}
		} else {
			indicator = ext.leafChar
		}
		buf.Set(posX, *y, Cell{Rune: indicator, Style: ext.style})
		posX++
		buf.Set(posX, *y, Cell{Rune: ' ', Style: ext.style})
		posX++

		effStyle := t.effectiveStyle(ext.style)
		labelText := applyTransform(node.Label, effStyle.Transform)
		buf.WriteStringFast(posX, *y, labelText, ext.style, utf8.RuneCountInString(labelText))
		(*y)++
	}

	if node.Expanded || !render {
		childCount := len(node.Children)
		for i, child := range node.Children {
			for len(t.treeScratchPfx) <= level {
				t.treeScratchPfx = append(t.treeScratchPfx, false)
			}
			if level >= 0 {
				t.treeScratchPfx[level] = i < childCount-1
			}
			t.renderTreeNode(buf, ext, child, x, y, level+1, true, t.treeScratchPfx)
		}
	}
}

// renderJump renders a jump target and its children.
func (t *Template) renderJump(buf *Buffer, op *Op, geom *Geom, absX, absY, maxW int16, idx int16) {
	// Render children first
	for i := op.ChildStart; i < op.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Parent == idx {
			t.renderOp(buf, i, absX, absY, maxW)
		}
	}

	// If jump mode is active, register this target and draw label
	if t.app != nil && t.app.JumpModeActive() {
		ext := op.Ext.(*opJump)
		t.app.AddJumpTarget(absX, absY, ext.onSelect, ext.style)

		// Draw label if assigned
		jm := t.app.JumpMode()
		for i := len(jm.Targets) - 1; i >= 0; i-- {
			target := &jm.Targets[i]
			if target.X == absX && target.Y == absY && target.Label != "" {
				style := t.app.JumpStyle().LabelStyle
				if !target.Style.Equal(Style{}) {
					style = target.Style
				}
				for j, r := range target.Label {
					buf.Set(int(absX)+j, int(absY), Cell{Rune: r, Style: style})
				}
				break
			}
		}
	}
}

func (t *Template) renderTextInput(buf *Buffer, op *Op, geom *Geom, absX, absY int16) {
	width := int(geom.W)
	if width <= 0 {
		return
	}

	ext := op.Ext.(*opTextInput)

	// Resolve styles through the cascade so inheritedFill applies as BG
	textStyle := t.effectiveStyle(ext.style)
	placeholderStyle := t.effectiveStyle(ext.placeholderSty)
	cursorStyle := t.effectiveStyle(ext.cursorStyle)

	// Get value and cursor - prefer Field API, fall back to pointer API
	var value string
	var cursor int
	if ext.fieldPtr != nil {
		value = ext.fieldPtr.Value
		cursor = ext.fieldPtr.Cursor
	} else {
		if ext.valuePtr != nil {
			value = *ext.valuePtr
		}
		cursor = len(value) // default to end
		if ext.cursorPtr != nil {
			cursor = *ext.cursorPtr
		}
	}

	// Clamp cursor to valid range
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(value) {
		cursor = len(value)
	}

	// Determine if cursor should be shown
	// Priority: FocusGroup > Focused > always show
	var showCursor bool
	if ext.focusGroupPtr != nil {
		showCursor = ext.focusGroupPtr.Current == ext.focusIndex
	} else if ext.focusedPtr != nil {
		showCursor = *ext.focusedPtr
	} else {
		// Default: show cursor if we have cursor tracking
		showCursor = ext.fieldPtr != nil || ext.cursorPtr != nil
	}

	// Handle empty state with placeholder
	if value == "" {
		if ext.placeholder != "" {
			buf.WriteStringFast(int(absX), int(absY), ext.placeholder, placeholderStyle, width)
		}
		// Draw cursor at start if focused
		if showCursor {
			buf.Set(int(absX), int(absY), Cell{Rune: ' ', Style: cursorStyle})
		}
		return
	}

	// Apply mask if set
	displayValue := value
	if ext.mask != 0 {
		runes := make([]rune, len([]rune(value)))
		for i := range runes {
			runes[i] = ext.mask
		}
		displayValue = string(runes)
	}

	// Calculate scroll offset for horizontal scrolling
	// Keep cursor visible within the field
	displayRunes := []rune(displayValue)
	cursorRune := cursor
	if cursorRune > len(displayRunes) {
		cursorRune = len(displayRunes)
	}

	scrollOffset := 0
	if showCursor && cursorRune >= width {
		scrollOffset = cursorRune - width + 1
	}

	// Render visible portion
	visibleEnd := scrollOffset + width
	if visibleEnd > len(displayRunes) {
		visibleEnd = len(displayRunes)
	}

	x := int(absX)
	for i := scrollOffset; i < visibleEnd; i++ {
		style := textStyle
		// Highlight cursor position if focused
		if showCursor && i == cursorRune {
			style = cursorStyle
		}
		buf.Set(x, int(absY), Cell{Rune: displayRunes[i], Style: style})
		x++
	}

	// If cursor is at end (after last char), draw cursor there
	if showCursor && cursorRune >= len(displayRunes) && cursorRune-scrollOffset < width {
		buf.Set(int(absX)+cursorRune-scrollOffset, int(absY), Cell{Rune: ' ', Style: cursorStyle})
	}
}

// ScreenEffects returns the post-processing passes collected from the tree
// during the most recent Execute. The returned slice is reused between frames.
func (t *Template) ScreenEffects() []Effect {
	return t.pendingScreenEffects
}

// renderOverlays renders all collected overlays after main content.
func (t *Template) renderOverlays(buf *Buffer, screenW, screenH int16) {
	for _, po := range t.pendingOverlays {
		t.renderOverlay(buf, po.op, screenW, screenH)
	}
}

// renderOverlay renders a single overlay to the buffer.
func (t *Template) renderOverlay(buf *Buffer, op *Op, screenW, screenH int16) {
	ext := op.Ext.(*opOverlay)
	if ext.childTmpl == nil {
		return
	}

	// Link app to child template for jump mode support
	ext.childTmpl.app = t.app

	// Propagate overlay BG as inheritedFill so all child text cells render with
	// the same explicit background, preventing patchy backdrop bleed-through
	if ext.bg.Mode != ColorDefault {
		ext.childTmpl.inheritedFill = ext.bg
	}

	// Calculate content size by doing a dry-run layout
	childTmpl := ext.childTmpl

	// Determine overlay dimensions
	overlayW := op.width()
	overlayH := op.height()

	if overlayW == 0 || overlayH == 0 {
		// Calculate natural size from content
		// DON'T call distributeFlexGrow - overlays should size to content, not expand
		childTmpl.distributeWidths(screenW, nil)
		childTmpl.layout(screenH)

		// Get root content size (natural height, no flex grow distribution)
		if len(childTmpl.geom) > 0 {
			if overlayW == 0 {
				overlayW = childTmpl.geom[0].W
			}
			if overlayH == 0 {
				overlayH = childTmpl.geom[0].H
			}
		}
	}

	// Calculate position
	var posX, posY int16
	if ext.centered {
		posX = (screenW - overlayW) / 2
		posY = (screenH - overlayH) / 2
	} else {
		posX = ext.x
		posY = ext.y
	}

	// Clamp to screen bounds
	if posX < 0 {
		posX = 0
	}
	if posY < 0 {
		posY = 0
	}

	// Draw backdrop if enabled
	if ext.backdrop {
		for y := int16(0); y < screenH; y++ {
			for x := int16(0); x < screenW; x++ {
				cell := buf.Get(int(x), int(y))
				// Dim existing content - preserve background, only modify FG and attr
				cell.Style.FG = ext.backdropFG
				cell.Style.Attr = AttrDim
				buf.Set(int(x), int(y), cell)
			}
		}
	}

	// Fill overlay content area with background color if set
	if ext.bg.Mode != ColorDefault {
		bgStyle := Style{BG: ext.bg}
		for y := posY; y < posY+overlayH && y < screenH; y++ {
			for x := posX; x < posX+overlayW && x < screenW; x++ {
				buf.Set(int(x), int(y), Cell{Rune: ' ', Style: bgStyle})
			}
		}
	}

	// Render the overlay content
	// Re-layout with actual available space
	childTmpl.pendingScreenEffects = childTmpl.pendingScreenEffects[:0]
	childTmpl.distributeWidths(overlayW, nil)
	childTmpl.layout(overlayH)
	childTmpl.distributeFlexGrow(overlayH)
	childTmpl.render(buf, posX, posY, overlayW)

	// bubble screen effects declared inside overlay up to the parent so they
	// run as full-screen passes after all content (including this overlay) is rendered
	t.pendingScreenEffects = append(t.pendingScreenEffects, childTmpl.pendingScreenEffects...)
}

func (t *Template) renderTabs(buf *Buffer, op *Op, geom *Geom, absX, absY int16) {
	ext := op.Ext.(*opTabs)
	selectedIdx := 0
	if ext.selectedPtr != nil {
		selectedIdx = *ext.selectedPtr
	}

	x := int(absX)
	y := int(absY)

	for i, label := range ext.labels {
		isSelected := i == selectedIdx
		style := t.effectiveStyle(ext.inactiveStyle)
		if isSelected {
			style = t.effectiveStyle(ext.activeStyle)
		}

		// apply transform to label text
		label = applyTransform(label, style.Transform)
		labelLen := utf8.RuneCountInString(label)

		switch ext.styleType {
		case TabsStyleBox:
			// Draw box around tab
			// Top border
			buf.Set(x, y, Cell{Rune: '┌', Style: style})
			for j := 0; j < labelLen+2; j++ {
				buf.Set(x+1+j, y, Cell{Rune: '─', Style: style})
			}
			buf.Set(x+labelLen+3, y, Cell{Rune: '┐', Style: style})
			// Content
			buf.Set(x, y+1, Cell{Rune: '│', Style: style})
			buf.Set(x+1, y+1, Cell{Rune: ' ', Style: style})
			buf.WriteStringFast(x+2, y+1, label, style, labelLen)
			buf.Set(x+labelLen+2, y+1, Cell{Rune: ' ', Style: style})
			buf.Set(x+labelLen+3, y+1, Cell{Rune: '│', Style: style})
			// Bottom border
			buf.Set(x, y+2, Cell{Rune: '└', Style: style})
			for j := 0; j < labelLen+2; j++ {
				buf.Set(x+1+j, y+2, Cell{Rune: '─', Style: style})
			}
			buf.Set(x+labelLen+3, y+2, Cell{Rune: '┘', Style: style})
			x += labelLen + 4 + ext.gap

		case TabsStyleBracket:
			buf.Set(x, y, Cell{Rune: '[', Style: style})
			buf.WriteStringFast(x+1, y, label, style, labelLen)
			buf.Set(x+1+labelLen, y, Cell{Rune: ']', Style: style})
			x += labelLen + 2 + ext.gap

		default: // TabsStyleUnderline
			if isSelected {
				// Write label with underline attribute
				underlineStyle := style
				underlineStyle.Attr = underlineStyle.Attr.With(AttrUnderline)
				buf.WriteStringFast(x, y, label, underlineStyle, labelLen)
			} else {
				buf.WriteStringFast(x, y, label, style, labelLen)
			}
			x += labelLen + ext.gap
		}
	}
}

func (t *Template) renderScrollbar(buf *Buffer, op *Op, geom *Geom, absX, absY int16) {
	ext := op.Ext.(*opScrollbar)
	// Calculate scrollbar dimensions
	length := int(geom.H)
	if ext.horizontal {
		length = int(geom.W)
	}

	if length == 0 {
		return
	}

	// Get scroll position
	pos := 0
	if ext.posPtr != nil {
		pos = *ext.posPtr
	}

	// Calculate thumb size and position
	contentSize := ext.contentSize
	viewSize := ext.viewSize
	if contentSize <= 0 {
		contentSize = 1
	}
	if viewSize <= 0 {
		viewSize = 1
	}

	// Thumb size proportional to view/content ratio (minimum 1)
	thumbSize := (viewSize * length) / contentSize
	if thumbSize < 1 {
		thumbSize = 1
	}
	if thumbSize > length {
		thumbSize = length
	}

	// Thumb position
	scrollRange := contentSize - viewSize
	if scrollRange <= 0 {
		scrollRange = 1
	}
	trackSpace := length - thumbSize
	thumbPos := 0
	if trackSpace > 0 {
		thumbPos = (pos * trackSpace) / scrollRange
		if thumbPos < 0 {
			thumbPos = 0
		}
		if thumbPos > trackSpace {
			thumbPos = trackSpace
		}
	}

	// Draw the scrollbar
	if ext.horizontal {
		// Horizontal scrollbar
		for i := 0; i < length; i++ {
			var char rune
			var style Style
			if i >= thumbPos && i < thumbPos+thumbSize {
				char = ext.thumbChar
				style = ext.thumbStyle
			} else {
				char = ext.trackChar
				style = ext.trackStyle
			}
			buf.Set(int(absX)+i, int(absY), Cell{Rune: char, Style: style})
		}
	} else {
		// Vertical scrollbar
		for i := 0; i < length; i++ {
			var char rune
			var style Style
			if i >= thumbPos && i < thumbPos+thumbSize {
				char = ext.thumbChar
				style = ext.thumbStyle
			} else {
				char = ext.trackChar
				style = ext.trackStyle
			}
			buf.Set(int(absX), int(absY)+i, Cell{Rune: char, Style: style})
		}
	}
}

func (t *Template) renderTable(buf *Buffer, op *Op, absX, absY, maxW int16) {
	ext := op.Ext.(*opTable)
	if ext.rowsPtr == nil {
		return
	}
	rows := *ext.rowsPtr
	y := int(absY)

	// Render header if enabled
	if ext.showHeader {
		x := int(absX)
		for _, col := range ext.columns {
			width := col.Width
			if width == 0 {
				width = 10
			}
			t.writeTableCell(buf, x, y, col.Header, width, col.Align, ext.headerStyle)
			x += width
		}
		y++
	}

	// Render data rows
	for rowIdx, row := range rows {
		x := int(absX)
		style := ext.rowStyle
		// Alternating row style (check if AltStyle has any non-default values)
		if rowIdx%2 == 1 && ext.altStyle != (Style{}) {
			style = ext.altStyle
		}

		for colIdx, col := range ext.columns {
			width := col.Width
			if width == 0 {
				width = 10
			}
			cellText := ""
			if colIdx < len(row) {
				cellText = row[colIdx]
			}
			t.writeTableCell(buf, x, y, cellText, width, col.Align, style)
			x += width
		}
		y++
	}
}

func (t *Template) writeTableCell(buf *Buffer, x, y int, text string, width int, align Align, style Style) {
	textLen := utf8.RuneCountInString(text)
	if textLen > width {
		// Truncate
		runes := []rune(text)
		text = string(runes[:width])
		textLen = width
	}

	padding := width - textLen
	var leftPad, rightPad int

	switch align {
	case AlignRight:
		leftPad = padding
	case AlignCenter:
		leftPad = padding / 2
		rightPad = padding - leftPad
	default: // AlignLeft
		rightPad = padding
	}

	// Write padding and text
	pos := x
	for i := 0; i < leftPad; i++ {
		buf.Set(pos, y, Cell{Rune: ' ', Style: style})
		pos++
	}
	for _, r := range text {
		buf.Set(pos, y, Cell{Rune: r, Style: style})
		pos++
	}
	for i := 0; i < rightPad; i++ {
		buf.Set(pos, y, Cell{Rune: ' ', Style: style})
		pos++
	}
}

func (t *Template) renderAutoTable(buf *Buffer, op *Op, absX, absY, maxW int16) {
	ext := op.Ext.(*opAutoTable)
	if ext.slicePtr == nil {
		return
	}

	rv := reflect.ValueOf(ext.slicePtr).Elem()
	nRows := rv.Len()
	nCols := len(ext.fields)
	gap := int(ext.gap)

	// re-apply sort if active (keeps data consistent after mutations)
	if ss := ext.sort; ss != nil && ss.col >= 0 && ss.col < nCols {
		autoTableSort(ext.slicePtr, ext.fields[ss.col], ss.asc)
	}

	// compute natural column widths from current data
	// if sorting is enabled, reserve space for the indicator on every header
	// since the user can cycle to any column
	indicatorW := 0
	if ext.sort != nil {
		indicatorW = 2 // " ▲" or " ▼"
	}
	widths := make([]int, nCols)
	for i, h := range ext.headers {
		widths[i] = len(h) + indicatorW
	}

	for i := 0; i < nRows; i++ {
		elem := rv.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		for j, fi := range ext.fields {
			val := elem.Field(fi).Interface()
			var str string
			if cfg := ext.colCfgs[j]; cfg != nil && cfg.format != nil {
				str = cfg.format(val)
			} else {
				str = fmt.Sprintf("%v", val)
			}
			if len(str) > widths[j] {
				widths[j] = len(str)
			}
		}
	}

	// distribute remaining width proportionally to natural column widths
	availW := int(maxW) - int(absX)
	totalNatural := 0
	for _, w := range widths {
		totalNatural += w
	}
	totalGaps := gap * (nCols - 1)
	remaining := availW - totalNatural - totalGaps

	if remaining > 0 && totalNatural > 0 {
		for i, w := range widths {
			extra := remaining * w / totalNatural
			widths[i] += extra
		}
	}

	hdrStyle := t.effectiveStyle(ext.hdrStyle)
	y := int(absY)

	// header row
	x := int(absX)
	jumpActive := ext.sort != nil && t.app != nil && t.app.JumpModeActive()

	for i, h := range ext.headers {
		text := applyTransform(h, hdrStyle.Transform)
		if ext.sort != nil && ext.sort.col == i {
			if ext.sort.asc {
				text += " ▲"
			} else {
				text += " ▼"
			}
		}
		hdrAlign := AlignLeft
		if cfg := ext.colCfgs[i]; cfg != nil {
			hdrAlign = cfg.align
		}
		t.writeTableCell(buf, x, y, text, widths[i], hdrAlign, hdrStyle)

		// register column header as a jump target for sorting
		if jumpActive {
			colIdx := i
			fieldIdx := ext.fields[i]
			ss := ext.sort
			slicePtr := ext.slicePtr
			t.app.AddJumpTarget(int16(x), int16(y), func() {
				if ss.col == colIdx {
					ss.asc = !ss.asc
				} else {
					ss.col = colIdx
					ss.asc = true
				}
				autoTableSort(slicePtr, fieldIdx, ss.asc)
			}, Style{})

			// draw jump label if assigned (second render pass)
			jm := t.app.JumpMode()
			for j := len(jm.Targets) - 1; j >= 0; j-- {
				target := &jm.Targets[j]
				if target.X == int16(x) && target.Y == int16(y) && target.Label != "" {
					style := t.app.JumpStyle().LabelStyle
					for k, r := range target.Label {
						buf.Set(x+k, y, Cell{Rune: r, Style: style})
					}
					break
				}
			}
		}

		x += widths[i] + gap
	}
	y++

	// data rows -- when scrolling is enabled, render all rows to an internal
	// buffer and blit only the visible viewport to the screen buffer.
	sc := ext.scroll
	if sc != nil {
		sc.clamp(nRows)

		// allocate or resize internal buffer (width = availW, height = nRows)
		if sc.buf == nil || sc.bufW != availW || sc.buf.Height() < nRows {
			sc.buf = NewBuffer(availW, nRows)
			sc.bufW = availW
		} else {
			sc.buf.Clear()
		}

		// render all data rows into internal buffer at y=0..nRows-1
		for i := 0; i < nRows; i++ {
			elem := rv.Index(i)
			if elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}

			rowStyle := t.effectiveStyle(ext.rowStyle)
			isAlt := ext.altStyle != nil && i%2 == 1
			if isAlt {
				rowStyle = t.effectiveStyle(*ext.altStyle)
			}

			if isAlt && ext.fill.Mode != ColorDefault {
				for fx := 0; fx < availW; fx++ {
					sc.buf.Set(fx, i, Cell{Rune: ' ', Style: Style{BG: ext.fill}})
				}
			}

			bx := 0
			for j, fi := range ext.fields {
				val := elem.Field(fi).Interface()
				cfg := ext.colCfgs[j]

				var str string
				if cfg != nil && cfg.format != nil {
					str = cfg.format(val)
				} else {
					str = fmt.Sprintf("%v", val)
				}

				cellStyle := rowStyle
				if cfg != nil && cfg.style != nil {
					cellStyle = cfg.style(val)
				}

				cellAlign := AlignLeft
				if cfg != nil {
					cellAlign = cfg.align
				}

				t.writeTableCell(sc.buf, bx, i, str, widths[j], cellAlign, cellStyle)
				bx += widths[j] + gap
			}
		}

		// blit visible window from internal buffer to screen
		visH := sc.maxVisible
		if visH > nRows {
			visH = nRows
		}
		buf.Blit(sc.buf, 0, sc.offset, int(absX), y, availW, visH)
	} else {
		// no scroll -- render directly (backwards compatible)
		for i := 0; i < nRows; i++ {
			elem := rv.Index(i)
			if elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}

			rowStyle := t.effectiveStyle(ext.rowStyle)
			isAlt := ext.altStyle != nil && i%2 == 1
			if isAlt {
				rowStyle = t.effectiveStyle(*ext.altStyle)
			}

			// fill entire row background for alt rows
			if isAlt && ext.fill.Mode != ColorDefault {
				for fx := int(absX); fx < int(maxW); fx++ {
					buf.Set(fx, y, Cell{Rune: ' ', Style: Style{BG: ext.fill}})
				}
			}

			x = int(absX)
			for j, fi := range ext.fields {
				val := elem.Field(fi).Interface()
				cfg := ext.colCfgs[j]

				var str string
				if cfg != nil && cfg.format != nil {
					str = cfg.format(val)
				} else {
					str = fmt.Sprintf("%v", val)
				}

				cellStyle := rowStyle
				if cfg != nil && cfg.style != nil {
					cellStyle = cfg.style(val)
				}

				cellAlign := AlignLeft
				if cfg != nil {
					cellAlign = cfg.align
				}

				t.writeTableCell(buf, x, y, str, widths[j], cellAlign, cellStyle)
				x += widths[j] + gap
			}
			y++
		}
	}
}

// Height returns the computed height after layout.
// Must call Execute first.
func (t *Template) Height() int16 {
	if len(t.geom) == 0 {
		return 0
	}
	// Find root-level ops and sum their heights
	var totalH int16
	for i, op := range t.ops {
		if op.Parent == -1 {
			totalH += t.geom[i].H
		}
	}
	return totalH
}

// DebugDump prints the template's op tree for debugging layout issues.
func (t *Template) DebugDump(prefix string) {
	fmt.Fprintf(os.Stderr, "%s=== Template Debug (%d ops) ===\n", prefix, len(t.ops))
	for i, op := range t.ops {
		geom := Geom{}
		if i < len(t.geom) {
			geom = t.geom[i]
		}
		kindStr := opKindName(op.Kind)
		flags := ""
		if op.ContentSized {
			flags += " [ContentSized]"
		}
		if op.FlexGrow > 0 {
			flags += fmt.Sprintf(" [Flex:%.1f]", op.FlexGrow)
		}
		if op.Width > 0 {
			flags += fmt.Sprintf(" [W:%d]", op.Width)
		}
		fmt.Fprintf(os.Stderr, "%s  [%d] %s parent=%d geom={W:%d H:%d}%s\n",
			prefix, i, kindStr, op.Parent, geom.W, geom.H, flags)

		// Dump sub-templates for If
		if op.Kind == OpIf {
			ifExt := op.Ext.(*opIf)
			if ifExt.thenTmpl != nil {
				ifExt.thenTmpl.DebugDump(prefix + "    Then: ")
			}
			if ifExt.elseTmpl != nil {
				ifExt.elseTmpl.DebugDump(prefix + "    Else: ")
			}
		}
	}
}

func opKindName(k OpKind) string {
	names := map[OpKind]string{
		OpText: "Text", OpProgress: "Progress", OpRichText: "RichText",
		OpLeader: "Leader", OpCounter: "Counter",
		OpContainer: "Container", OpIf: "If", OpForEach: "ForEach", OpSwitch: "Switch",
		OpCustom: "Custom", OpLayout: "Layout", OpLayer: "Layer",
		OpSelectionList: "SelectionList",
		OpTable: "Table", OpAutoTable: "AutoTable", OpSparkline: "Sparkline",
		OpHRule: "HRule", OpVRule: "VRule", OpSpacer: "Spacer",
		OpSpinner: "Spinner", OpScrollbar: "Scrollbar", OpTabs: "Tabs", OpTreeView: "TreeView",
		OpJump: "Jump", OpTextInput: "TextInput", OpOverlay: "Overlay", OpScreenEffect: "ScreenEffect",
	}
	if name, ok := names[k]; ok {
		return name
	}
	return fmt.Sprintf("Op(%d)", k)
}
