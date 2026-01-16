package tui

import (
	"reflect"
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

// Box is a container with a custom layout function.
// Use this when Row/Col don't fit your needs.
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
}

// Geom holds runtime geometry for an op.
// Filled during execute, parallel array to ops.
type Geom struct {
	W, H           int16 // dimensions
	LocalX, LocalY int16 // position relative to parent
	ContentH       int16 // natural content height (before flex distribution)
}

// Op represents a single instruction.
type Op struct {
	Kind   OpKind
	Depth  int8  // tree depth (root children = 0)
	Parent int16 // parent op index, -1 for root children

	// Value access - one used based on Kind
	StaticStr string
	StrPtr    *string
	StrOff    uintptr // offset from element base (for ForEach)

	StaticInt int
	IntPtr    *int
	IntOff    uintptr

	// Layout hints
	Width        int16   // explicit width
	Height       int16   // explicit height
	PercentWidth float32 // 0.0-1.0
	FlexGrow     float32 // share of remaining space
	Gap          int8    // gap between children

	// Container
	IsRow       bool        // true=Row, false=Col
	Border      BorderStyle // border style
	BorderFG    *Color      // border color
	Title       string      // border title
	ChildStart  int16       // first child op index
	ChildEnd    int16       // last child op index (exclusive)

	// Control flow
	CondPtr  *bool         // for If (simple bool pointer)
	CondNode ConditionNode // for If (builder-style conditions)
	ThenTmpl *Template   // for If
	ElseTmpl *Template   // for If/Else
	IterTmpl *Template  // for ForEach
	SlicePtr unsafe.Pointer
	ElemSize uintptr

	// ForEach runtime - reused across frames
	iterGeoms []Geom // per-item geometry

	// Switch
	SwitchNode  SwitchNodeInterface
	SwitchCases []*Template
	SwitchDef   *Template

	// Custom renderer
	CustomRenderer Renderer

	// Custom layout
	CustomLayout LayoutFunc

	// Layer
	LayerPtr    *Layer // pointer to Layer
	LayerWidth  int16  // viewport width (0 = fill available)
	LayerHeight int16  // viewport height (0 = fill available)

	// RichText
	StaticSpans []Span   // for static spans
	SpansPtr    *[]Span  // for pointer to spans
	SpansOff    uintptr  // for ForEach offset

	// SelectionList
	SelectionListPtr *SelectionList // pointer to the list for len/offset updates
	SelectedPtr      *int           // pointer to selected index
	Marker           string         // selection marker (e.g., "> ")
	MarkerWidth      int16          // cached rune count of marker
}

type OpKind uint8

const (
	OpText OpKind = iota
	OpTextPtr
	OpTextOff

	OpProgress
	OpProgressPtr
	OpProgressOff

	OpContainer // Col or Row (determined by IsRow)

	OpIf
	OpForEach
	OpSwitch

	OpCustom // Custom renderer
	OpLayout // Custom layout
	OpLayer  // LayerView (scrollable off-screen buffer)

	OpRichText    // RichText with static spans
	OpRichTextPtr // RichText with pointer to spans
	OpRichTextOff // RichText with offset (ForEach)

	OpSelectionList // SelectionList with marker and windowing
)

// Build compiles a declarative UI into a Template.
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
	case Text:
		return t.compileText(v, parent, depth, elemBase, elemSize)
	case Progress:
		return t.compileProgress(v, parent, depth, elemBase, elemSize)
	case Row:
		return t.compileContainer(v.Children, v.Gap, true, v.flex, v.border, v.Title, v.borderFG, parent, depth, elemBase, elemSize)
	case Col:
		return t.compileContainer(v.Children, v.Gap, false, v.flex, v.border, v.Title, v.borderFG, parent, depth, elemBase, elemSize)
	case IfNode:
		return t.compileIf(v, parent, depth, elemBase, elemSize)
	case ForEachNode:
		return t.compileForEach(v, parent, depth)
	case Renderer:
		return t.compileRenderer(v, parent, depth)
	case Box:
		return t.compileBox(v, parent, depth, elemBase, elemSize)
	case ConditionNode:
		return t.compileCondition(v, parent, depth, elemBase, elemSize)
	case LayerView:
		return t.compileLayer(v, parent, depth)
	case RichText:
		return t.compileRichText(v, parent, depth, elemBase, elemSize)
	case SelectionList:
		return t.compileSelectionList(&v, parent, depth, elemBase, elemSize)
	case *SelectionList:
		return t.compileSelectionList(v, parent, depth, elemBase, elemSize)
	case Component:
		return t.compile(v.Build(), parent, depth, elemBase, elemSize)
	}

	// Check for SwitchNodeInterface (generic Switch)
	if sw, ok := node.(SwitchNodeInterface); ok {
		return t.compileSwitch(sw, parent, depth, elemBase, elemSize)
	}

	return -1
}

func (t *Template) compileRenderer(r Renderer, parent int16, depth int) int16 {
	return t.addOp(Op{
		Kind:           OpCustom,
		Parent:         parent,
		CustomRenderer: r,
	}, depth)
}

func (t *Template) compileBox(box Box, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	// Add layout op first (will fill in ChildStart/ChildEnd)
	idx := t.addOp(Op{
		Kind:         OpLayout,
		Parent:       parent,
		CustomLayout: box.Layout,
		ChildStart:   int16(len(t.ops)),
	}, depth)

	// Compile children
	for _, child := range box.Children {
		t.compile(child, idx, depth+1, elemBase, elemSize)
	}

	// Set child end
	t.ops[idx].ChildEnd = int16(len(t.ops))

	return idx
}

func (t *Template) compileLayer(v LayerView, parent int16, depth int) int16 {
	return t.addOp(Op{
		Kind:        OpLayer,
		Parent:      parent,
		LayerPtr:    v.Layer,
		LayerWidth:  v.ViewWidth,
		LayerHeight: v.ViewHeight,
		FlexGrow:    v.FlexGrow, // Allow layers to participate in flex
	}, depth)
}

func (t *Template) compileRichText(v RichText, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	op := Op{
		Parent: parent,
	}

	switch spans := v.Spans.(type) {
	case []Span:
		op.Kind = OpRichText
		op.StaticSpans = spans
	case *[]Span:
		if elemBase != nil && isWithinRange(unsafe.Pointer(spans), elemBase, elemSize) {
			op.Kind = OpRichTextOff
			op.SpansOff = uintptr(unsafe.Pointer(spans)) - uintptr(elemBase)
		} else {
			op.Kind = OpRichTextPtr
			op.SpansPtr = spans
		}
	default:
		// Empty RichText
		op.Kind = OpRichText
		op.StaticSpans = nil
	}

	return t.addOp(op, depth)
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
	if v.Render != nil {
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

	op := Op{
		Kind:             OpSelectionList,
		Parent:           parent,
		SlicePtr:         slicePtr,
		ElemSize:         sliceElemSize,
		IterTmpl:         iterTmpl,
		SelectionListPtr: v,
		SelectedPtr:      v.Selected,
		Marker:           marker,
		MarkerWidth:      markerWidth,
	}

	return t.addOp(op, depth)
}

func (t *Template) compileText(v Text, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	op := Op{
		Parent: parent,
	}

	switch val := v.Content.(type) {
	case string:
		op.Kind = OpText
		op.StaticStr = val
	case *string:
		if elemBase != nil && isWithinRange(unsafe.Pointer(val), elemBase, elemSize) {
			op.Kind = OpTextOff
			op.StrOff = uintptr(unsafe.Pointer(val)) - uintptr(elemBase)
		} else {
			op.Kind = OpTextPtr
			op.StrPtr = val
		}
	}

	return t.addOp(op, depth)
}

func (t *Template) compileProgress(v Progress, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	width := v.BarWidth
	if width == 0 {
		width = 20
	}

	op := Op{
		Parent: parent,
		Width:  width,
	}

	switch val := v.Value.(type) {
	case int:
		op.Kind = OpProgress
		op.StaticInt = val
	case *int:
		if elemBase != nil && isWithinRange(unsafe.Pointer(val), elemBase, elemSize) {
			op.Kind = OpProgressOff
			op.IntOff = uintptr(unsafe.Pointer(val)) - uintptr(elemBase)
		} else {
			op.Kind = OpProgressPtr
			op.IntPtr = val
		}
	}

	return t.addOp(op, depth)
}

func (t *Template) compileContainer(children []any, gap int8, isRow bool, f flex, border BorderStyle, title string, borderFG *Color, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	op := Op{
		Kind:         OpContainer,
		Parent:       parent,
		IsRow:        isRow,
		Gap:          gap,
		PercentWidth: f.percentWidth,
		Width:        f.width,
		Height:       f.height,
		FlexGrow:     f.flexGrow,
		Border:       border,
		Title:        title,
		BorderFG:     borderFG,
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

	return idx
}

func (t *Template) compileIf(v IfNode, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	op := Op{
		Kind:   OpIf,
		Parent: parent,
	}

	// Compile condition pointer
	switch val := v.Cond.(type) {
	case *bool:
		op.CondPtr = val
	}

	// Compile then branch as sub-template
	if v.Then != nil {
		thenTmpl := &Template{
			ops:     make([]Op, 0, 16),
			byDepth: make([][]int16, 8),
		}
		for i := range thenTmpl.byDepth {
			thenTmpl.byDepth[i] = make([]int16, 0, 4)
		}
		thenTmpl.compile(v.Then, -1, 0, elemBase, elemSize)
		if thenTmpl.maxDepth >= 0 {
			thenTmpl.byDepth = thenTmpl.byDepth[:thenTmpl.maxDepth+1]
		}
		thenTmpl.geom = make([]Geom, len(thenTmpl.ops))
		op.ThenTmpl = thenTmpl
	}

	return t.addOp(op, depth)
}

func (t *Template) compileCondition(cond ConditionNode, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	// Check if condition pointer is within element range (ForEach context)
	if elemBase != nil && elemSize > 0 {
		ptrAddr := cond.getPtrAddr()
		baseAddr := uintptr(elemBase)
		if ptrAddr >= baseAddr && ptrAddr < baseAddr+elemSize {
			// Set offset for rebinding during render
			cond.setOffset(ptrAddr - baseAddr)
		}
	}

	op := Op{
		Kind:     OpIf,
		Parent:   parent,
		CondNode: cond,
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
		op.ThenTmpl = thenTmpl
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
		op.ElseTmpl = elseTmpl
	}

	return t.addOp(op, depth)
}

func (t *Template) compileSwitch(sw SwitchNodeInterface, parent int16, depth int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	op := Op{
		Kind:       OpSwitch,
		Parent:     parent,
		SwitchNode: sw,
	}

	// Compile each case branch
	caseNodes := sw.getCaseNodes()
	op.SwitchCases = make([]*Template, len(caseNodes))
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
			op.SwitchCases[i] = caseTmpl
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
		op.SwitchDef = defTmpl
	}

	return t.addOp(op, depth)
}

func (t *Template) compileForEach(v ForEachNode, parent int16, depth int) int16 {
	// Analyze slice
	sliceRV := reflect.ValueOf(v.Items)
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
		Kind:     OpForEach,
		Parent:   parent,
		SlicePtr: slicePtr,
		ElemSize: elemSize,
		IterTmpl: iterTmpl,
	}

	return t.addOp(op, depth)
}

// Execute runs all three phases and renders to the buffer.
func (t *Template) Execute(buf *Buffer, screenW, screenH int16) {
	// Phase 1: Width distribution (top → down)
	t.distributeWidths(screenW, nil)

	// Phase 2: Layout (bottom → up) - computes content heights
	t.layout(screenH)

	// Phase 2b: Flex distribution (top → down) - expand flex children
	t.distributeFlexGrow(screenH)

	// Phase 3: Render (top → down)
	t.render(buf, 0, 0, screenW)
}

// distributeWidths assigns W to all ops, top-down.
// Each container sets its children's widths. For Rows, this includes flex distribution.
// elemBase is optional - used for offset-based text in ForEach sub-templates.
func (t *Template) distributeWidths(screenW int16, elemBase unsafe.Pointer) {
	// Set root-level ops to screen width first
	for _, idx := range t.byDepth[0] {
		op := &t.ops[idx]
		geom := &t.geom[idx]
		t.setOpWidth(op, geom, screenW, elemBase)
	}

	// Process containers depth-by-depth, each setting its children's widths
	for depth := 0; depth <= t.maxDepth; depth++ {
		for _, idx := range t.byDepth[depth] {
			op := &t.ops[idx]
			geom := &t.geom[idx]

			if op.Kind == OpContainer {
				t.distributeWidthsToChildren(idx, op, geom, elemBase)
			}
		}
	}
}

// setOpWidth sets a single op's width based on available space.
func (t *Template) setOpWidth(op *Op, geom *Geom, availW int16, elemBase unsafe.Pointer) {
	switch op.Kind {
	case OpText:
		geom.W = int16(utf8.RuneCountInString(op.StaticStr))

	case OpTextPtr:
		geom.W = int16(utf8.RuneCountInString(*op.StrPtr))

	case OpTextOff:
		if elemBase != nil {
			strPtr := (*string)(unsafe.Pointer(uintptr(elemBase) + op.StrOff))
			geom.W = int16(utf8.RuneCountInString(*strPtr))
		} else {
			geom.W = 10
		}

	case OpProgress, OpProgressPtr, OpProgressOff:
		geom.W = op.Width

	case OpCustom:
		if op.CustomRenderer != nil {
			w, _ := op.CustomRenderer.MinSize()
			geom.W = int16(w)
		}

	case OpLayout:
		geom.W = availW

	case OpLayer:
		if op.LayerWidth > 0 {
			geom.W = op.LayerWidth
		} else {
			geom.W = availW
		}

	case OpSelectionList:
		geom.W = availW

	case OpContainer:
		if op.Width > 0 {
			geom.W = op.Width
		} else if op.PercentWidth > 0 {
			geom.W = int16(float32(availW) * op.PercentWidth)
		} else {
			geom.W = availW
		}

	default:
		geom.W = availW
	}
}

// distributeWidthsToChildren sets widths for all children of a container.
// For Rows: two-pass (non-flex first, then flex distribution).
// For Cols: children fill available width.
func (t *Template) distributeWidthsToChildren(idx int16, op *Op, geom *Geom, elemBase unsafe.Pointer) {
	// Calculate content width (subtract border)
	contentW := geom.W
	if op.Border.Horizontal != 0 {
		contentW -= 2
	}

	if op.IsRow {
		t.distributeRowChildWidths(idx, op, contentW, elemBase)
	} else {
		t.distributeColChildWidths(idx, op, contentW, elemBase)
	}
}

// distributeColChildWidths sets widths for children of a Col (they fill available width).
func (t *Template) distributeColChildWidths(idx int16, op *Op, availW int16, elemBase unsafe.Pointer) {
	for i := op.ChildStart; i < op.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Parent != idx {
			continue
		}
		childGeom := &t.geom[i]
		t.setOpWidth(childOp, childGeom, availW, elemBase)
	}
}

// distributeRowChildWidths sets widths for children of a Row using two-pass flex.
func (t *Template) distributeRowChildWidths(idx int16, op *Op, availW int16, elemBase unsafe.Pointer) {
	// Pass 1: Set widths for non-flex children, collect flex children
	// Containers without explicit width/flex are treated as implicit flex (share remaining space)
	var usedW int16
	var totalFlex float32
	var flexChildren []int16
	var implicitFlexChildren []int16 // containers without explicit width

	for i := op.ChildStart; i < op.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Parent != idx {
			continue
		}
		childGeom := &t.geom[i]

		if childOp.FlexGrow > 0 {
			// Explicit flex child - defer to pass 2
			totalFlex += childOp.FlexGrow
			flexChildren = append(flexChildren, i)
		} else if childOp.Kind == OpContainer && childOp.Width == 0 && childOp.PercentWidth == 0 {
			// Container without explicit width - treat as implicit flex
			implicitFlexChildren = append(implicitFlexChildren, i)
		} else {
			// Non-flex child with explicit or content-based width
			t.setOpWidth(childOp, childGeom, availW, elemBase)
			usedW += childGeom.W
		}
	}

	// Account for gaps
	childCount := int16(0)
	for i := op.ChildStart; i < op.ChildEnd; i++ {
		if t.ops[i].Parent == idx {
			childCount++
		}
	}
	if childCount > 1 && op.Gap > 0 {
		usedW += int16(op.Gap) * (childCount - 1)
	}

	// Pass 2: Distribute remaining width to flex children
	remaining := availW - usedW
	if remaining > 0 && totalFlex > 0 {
		// Explicit flex children
		distributed := int16(0)
		for i, childIdx := range flexChildren {
			childOp := &t.ops[childIdx]
			childGeom := &t.geom[childIdx]

			flexShare := childOp.FlexGrow / totalFlex
			flexW := int16(float32(remaining) * flexShare)

			// Last flex child gets remainder (avoid rounding loss)
			if i == len(flexChildren)-1 {
				flexW = remaining - distributed
			}
			distributed += flexW

			// Set the flex child's width
			childGeom.W = flexW
		}
	} else if remaining > 0 && len(implicitFlexChildren) > 0 {
		// No explicit flex, but implicit flex containers - share remaining evenly
		shareW := remaining / int16(len(implicitFlexChildren))
		distributed := int16(0)
		for i, childIdx := range implicitFlexChildren {
			childGeom := &t.geom[childIdx]

			w := shareW
			// Last child gets remainder
			if i == len(implicitFlexChildren)-1 {
				w = remaining - distributed
			}
			distributed += w
			childGeom.W = w
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
			case OpText, OpTextPtr, OpTextOff:
				geom.H = 1

			case OpProgress, OpProgressPtr, OpProgressOff:
				geom.H = 1

			case OpRichText, OpRichTextPtr, OpRichTextOff:
				geom.H = 1

			case OpSelectionList:
				// Calculate height based on slice length and MaxVisible
				sliceHdr := *(*sliceHeader)(op.SlicePtr)
				// Update len for helper methods
				if op.SelectionListPtr != nil {
					op.SelectionListPtr.len = sliceHdr.Len
					op.SelectionListPtr.ensureVisible()
				}
				visibleCount := sliceHdr.Len
				if op.SelectionListPtr != nil && op.SelectionListPtr.MaxVisible > 0 && visibleCount > op.SelectionListPtr.MaxVisible {
					visibleCount = op.SelectionListPtr.MaxVisible
				}
				geom.H = int16(visibleCount)
				if geom.H == 0 {
					geom.H = 1 // Minimum height
				}

			case OpCustom:
				// Custom renderer provides its own size
				if op.CustomRenderer != nil {
					_, h := op.CustomRenderer.MinSize()
					geom.H = int16(h)
				}

			case OpLayer:
				// Layer height calculation
				if op.LayerHeight > 0 {
					// Explicit viewport height
					geom.H = op.LayerHeight
				} else if op.FlexGrow > 0 {
					// Flex layer - use minimal height, will expand via flex
					geom.H = 1
				} else if op.LayerPtr != nil && op.LayerPtr.viewHeight > 0 {
					// Use pre-set viewport height
					geom.H = int16(op.LayerPtr.viewHeight)
				} else {
					// Default to 1 line
					geom.H = 1
				}
				// Store content height for flex distribution
				geom.ContentH = geom.H

			case OpLayout:
				t.layoutCustom(idx, op, geom)

			case OpContainer:
				t.layoutContainer(idx, op, geom)
			}
		}
	}
}

// layoutContainer positions children and computes container height.
func (t *Template) layoutContainer(idx int16, op *Op, geom *Geom) {
	// Content area offset for border
	contentOffX := int16(0)
	contentOffY := int16(0)
	if op.Border.Horizontal != 0 {
		contentOffX = 1
		contentOffY = 1
	}

	availW := geom.W
	if op.Border.Horizontal != 0 {
		availW -= 2
	}

	if op.IsRow {
		// Horizontal layout
		cursor := int16(0)
		maxH := int16(0)
		firstChild := true

		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &t.ops[i]
			if childOp.Parent != idx {
				continue // not direct child
			}

			// Handle gap
			if !firstChild && op.Gap > 0 {
				cursor += int16(op.Gap)
			}
			firstChild = false

			// Control flow ops expand to their content
			switch childOp.Kind {
			case OpIf:
				// Use evaluateWithBase for conditions in ForEach context
				condTrue := (childOp.CondPtr != nil && *childOp.CondPtr) ||
					(childOp.CondNode != nil && childOp.CondNode.evaluateWithBase(t.elemBase))
				if childOp.ThenTmpl != nil && condTrue {
					childOp.ThenTmpl.elemBase = t.elemBase
					childOp.ThenTmpl.distributeWidths(availW, t.elemBase)
					childOp.ThenTmpl.layout(0)
					h := childOp.ThenTmpl.Height()
					t.geom[i].LocalX = contentOffX + cursor
					t.geom[i].LocalY = contentOffY
					t.geom[i].H = h
					// Width = sub-template width (for now, first root op)
					if len(childOp.ThenTmpl.geom) > 0 {
						t.geom[i].W = childOp.ThenTmpl.geom[0].W
						cursor += childOp.ThenTmpl.geom[0].W
					}
					if h > maxH {
						maxH = h
					}
				} else if childOp.ElseTmpl != nil && !condTrue {
					childOp.ElseTmpl.elemBase = t.elemBase
					childOp.ElseTmpl.distributeWidths(availW, t.elemBase)
					childOp.ElseTmpl.layout(0)
					h := childOp.ElseTmpl.Height()
					t.geom[i].LocalX = contentOffX + cursor
					t.geom[i].LocalY = contentOffY
					t.geom[i].H = h
					if len(childOp.ElseTmpl.geom) > 0 {
						t.geom[i].W = childOp.ElseTmpl.geom[0].W
						cursor += childOp.ElseTmpl.geom[0].W
					}
					if h > maxH {
						maxH = h
					}
				}

			case OpForEach:
				h, w := t.layoutForEach(i, childOp, availW)
				t.geom[i].LocalX = contentOffX + cursor
				t.geom[i].LocalY = contentOffY
				t.geom[i].H = h
				t.geom[i].W = w
				cursor += w
				if h > maxH {
					maxH = h
				}

			case OpSwitch:
				// Get matching template
				var tmpl *Template
				matchIdx := childOp.SwitchNode.getMatchIndex()
				if matchIdx >= 0 && matchIdx < len(childOp.SwitchCases) {
					tmpl = childOp.SwitchCases[matchIdx]
				} else {
					tmpl = childOp.SwitchDef
				}
				if tmpl != nil {
					tmpl.elemBase = t.elemBase
					tmpl.distributeWidths(availW, t.elemBase)
					tmpl.layout(0)
					h := tmpl.Height()
					t.geom[i].LocalX = contentOffX + cursor
					t.geom[i].LocalY = contentOffY
					t.geom[i].H = h
					if len(tmpl.geom) > 0 {
						t.geom[i].W = tmpl.geom[0].W
						cursor += tmpl.geom[0].W
					}
					if h > maxH {
						maxH = h
					}
				}

			default:
				childGeom := &t.geom[i]
				childGeom.LocalX = contentOffX + cursor
				childGeom.LocalY = contentOffY
				cursor += childGeom.W
				if childGeom.H > maxH {
					maxH = childGeom.H
				}
			}
		}

		geom.H = maxH
		if op.Border.Horizontal != 0 {
			geom.H += 2
		}
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
			if !firstChild && op.Gap > 0 {
				cursor += int16(op.Gap)
			}
			firstChild = false

			// Control flow ops expand to their content
			switch childOp.Kind {
			case OpIf:
				// Use evaluateWithBase for conditions in ForEach context
				condTrue := (childOp.CondPtr != nil && *childOp.CondPtr) ||
					(childOp.CondNode != nil && childOp.CondNode.evaluateWithBase(t.elemBase))
				if childOp.ThenTmpl != nil && condTrue {
					childOp.ThenTmpl.elemBase = t.elemBase
					childOp.ThenTmpl.distributeWidths(availW, t.elemBase)
					childOp.ThenTmpl.layout(0)
					h := childOp.ThenTmpl.Height()
					t.geom[i].LocalX = contentOffX
					t.geom[i].LocalY = contentOffY + cursor
					t.geom[i].H = h
					t.geom[i].ContentH = h // Track content height for flex
					t.geom[i].W = availW
					cursor += h
				} else if childOp.ElseTmpl != nil && !condTrue {
					childOp.ElseTmpl.elemBase = t.elemBase
					childOp.ElseTmpl.distributeWidths(availW, t.elemBase)
					childOp.ElseTmpl.layout(0)
					h := childOp.ElseTmpl.Height()
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
				var tmpl *Template
				matchIdx := childOp.SwitchNode.getMatchIndex()
				if matchIdx >= 0 && matchIdx < len(childOp.SwitchCases) {
					tmpl = childOp.SwitchCases[matchIdx]
				} else {
					tmpl = childOp.SwitchDef
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

		geom.H = cursor
		if op.Border.Horizontal != 0 {
			geom.H += 2
		}
	}

	// Store content height before any override (for flex distribution)
	geom.ContentH = geom.H

	// Explicit height overrides
	if op.Height > 0 {
		geom.H = op.Height
	}
}

// distributeFlexGrow distributes remaining space to flex children.
// Called top-down after layout phase.
// Vertical containers (Col) distribute height, horizontal containers (Row) distribute width.
// distributeFlexGrow distributes remaining height to Col flex children.
// Row flex is handled during width distribution (single pass).
// Col flex must happen after layout since it needs content heights.
func (t *Template) distributeFlexGrow(rootH int16) {
	for depth := 0; depth <= t.maxDepth; depth++ {
		for _, idx := range t.byDepth[depth] {
			op := &t.ops[idx]

			// Only Cols need height flex distribution here
			// Rows already handled width flex in distributeWidths
			if op.Kind == OpContainer && !op.IsRow {
				t.distributeFlexInCol(idx, op, rootH)
			}
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
	if op.FlexGrow > 0 && geom.H > 0 {
		// This container is a flex child - use its own height (already computed)
		availH = geom.H
		if op.Border.Horizontal != 0 {
			availH -= 2 // Subtract own border from available content space
		}
	} else if op.Parent >= 0 {
		parentGeom := &t.geom[op.Parent]
		availH = parentGeom.H
		if t.ops[op.Parent].Border.Horizontal != 0 {
			availH -= 2 // Account for parent border
		}
	} else {
		availH = rootH
	}

	// If this container has explicit height, use that
	if op.Height > 0 {
		availH = op.Height
		if op.Border.Horizontal != 0 {
			availH -= 2
		}
	}

	// Calculate used height and total flex grow
	var usedH int16
	var totalFlex float32
	var flexChildren []int16
	var flexGrowValues []float32 // Store flex values (may come from nested template)

	for i := op.ChildStart; i < op.ChildEnd; i++ {
		childOp := &t.ops[i]
		if childOp.Parent != idx {
			continue
		}

		childGeom := &t.geom[i]

		// Check for direct flex child (container or layer)
		if (childOp.Kind == OpContainer || childOp.Kind == OpLayer) && childOp.FlexGrow > 0 {
			totalFlex += childOp.FlexGrow
			flexChildren = append(flexChildren, i)
			flexGrowValues = append(flexGrowValues, childOp.FlexGrow)
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

	// Add gaps to used height
	childCount := int16(0)
	for i := op.ChildStart; i < op.ChildEnd; i++ {
		if t.ops[i].Parent == idx {
			childCount++
		}
	}
	if childCount > 1 && op.Gap > 0 {
		usedH += int16(op.Gap) * (childCount - 1)
	}

	// Distribute remaining space
	remaining := availH - usedH
	if remaining > 0 && totalFlex > 0 {
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
			childGeom.H = childGeom.ContentH + extraH
		}

		// Recalculate child positions with new heights
		contentOffY := int16(0)
		if op.Border.Horizontal != 0 {
			contentOffY = 1
		}
		cursor := int16(0)
		firstChild := true

		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &t.ops[i]
			if childOp.Parent != idx {
				continue
			}

			if !firstChild && op.Gap > 0 {
				cursor += int16(op.Gap)
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
	condTrue := (op.CondPtr != nil && *op.CondPtr) ||
		(op.CondNode != nil && op.CondNode.evaluateWithBase(t.elemBase))

	var tmpl *Template
	if condTrue && op.ThenTmpl != nil {
		tmpl = op.ThenTmpl
	} else if !condTrue && op.ElseTmpl != nil {
		tmpl = op.ElseTmpl
	}

	if tmpl == nil || len(tmpl.ops) == 0 {
		return
	}

	// If root is a flex container, update its height and redistribute
	rootOp := &tmpl.ops[0]
	if rootOp.Kind == OpContainer && rootOp.FlexGrow > 0 {
		tmpl.geom[0].H = newH
		tmpl.distributeFlexGrow(newH)
	}
}

// getIfFlexGrow returns the FlexGrow value from an If's active branch, if any.
// This allows If-wrapped containers to participate in flex distribution.
func (t *Template) getIfFlexGrow(op *Op) float32 {
	// Determine which branch is active
	condTrue := (op.CondPtr != nil && *op.CondPtr) ||
		(op.CondNode != nil && op.CondNode.evaluateWithBase(t.elemBase))

	var tmpl *Template
	if condTrue && op.ThenTmpl != nil {
		tmpl = op.ThenTmpl
	} else if !condTrue && op.ElseTmpl != nil {
		tmpl = op.ElseTmpl
	}

	if tmpl == nil || len(tmpl.ops) == 0 {
		return 0
	}

	// Check if root op of the branch is a Container with FlexGrow
	rootOp := &tmpl.ops[0]
	if rootOp.Kind == OpContainer && rootOp.FlexGrow > 0 {
		return rootOp.FlexGrow
	}

	return 0
}

// layoutCustom handles custom layout containers using the Arranger interface.
func (t *Template) layoutCustom(idx int16, op *Op, geom *Geom) {
	if op.CustomLayout == nil {
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
	rects := op.CustomLayout(childSizes, int(geom.W), int(geom.H))

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
	if op.IterTmpl == nil || op.SlicePtr == nil {
		return 0, 0
	}

	sliceHdr := *(*sliceHeader)(op.SlicePtr)
	if sliceHdr.Len == 0 {
		return 0, 0
	}

	// Ensure we have enough geometry slots for items
	if cap(op.iterGeoms) < sliceHdr.Len {
		op.iterGeoms = make([]Geom, sliceHdr.Len)
	}
	op.iterGeoms = op.iterGeoms[:sliceHdr.Len]

	cursor := int16(0)
	for i := 0; i < sliceHdr.Len; i++ {
		// Get element pointer for this item
		elemPtr := unsafe.Pointer(uintptr(sliceHdr.Data) + uintptr(i)*op.ElemSize)

		// Layout sub-template for this item with element base
		op.IterTmpl.elemBase = elemPtr // Set element base for condition evaluation
		op.IterTmpl.distributeWidths(availW, elemPtr)
		op.IterTmpl.layout(0)
		itemH := op.IterTmpl.Height()

		op.iterGeoms[i].LocalX = 0
		op.iterGeoms[i].LocalY = cursor
		op.iterGeoms[i].H = itemH
		op.iterGeoms[i].W = availW

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

func (t *Template) renderOp(buf *Buffer, idx int16, globalX, globalY, maxW int16) {
	if idx < 0 || int(idx) >= len(t.ops) {
		return
	}

	op := &t.ops[idx]
	geom := &t.geom[idx]

	// Compute absolute position
	absX := globalX + geom.LocalX
	absY := globalY + geom.LocalY

	switch op.Kind {
	case OpText:
		buf.WriteStringFast(int(absX), int(absY), op.StaticStr, Style{}, int(maxW))

	case OpTextPtr:
		buf.WriteStringFast(int(absX), int(absY), *op.StrPtr, Style{}, int(maxW))

	case OpTextOff:
		// Would need elemBase passed through for ForEach
		// For now, skip

	case OpProgress:
		ratio := float32(op.StaticInt) / 100.0
		buf.WriteProgressBar(int(absX), int(absY), int(op.Width), ratio, Style{})

	case OpProgressPtr:
		ratio := float32(*op.IntPtr) / 100.0
		buf.WriteProgressBar(int(absX), int(absY), int(op.Width), ratio, Style{})

	case OpRichText:
		buf.WriteSpans(int(absX), int(absY), op.StaticSpans, int(maxW))

	case OpRichTextPtr:
		buf.WriteSpans(int(absX), int(absY), *op.SpansPtr, int(maxW))

	case OpRichTextOff:
		// Would need elemBase passed through for ForEach
		// For now, skip

	case OpSelectionList:
		t.renderSelectionList(buf, op, geom, absX, absY, maxW)

	case OpCustom:
		// Custom renderer draws itself
		if op.CustomRenderer != nil {
			op.CustomRenderer.Render(buf, int(absX), int(absY), int(geom.W), int(geom.H))
		}

	case OpLayout:
		// Custom layout just renders children at their arranged positions
		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &t.ops[i]
			if childOp.Parent != idx {
				continue
			}
			t.renderOp(buf, i, absX, absY, geom.W)
		}

	case OpLayer:
		// Blit the layer's visible portion to the buffer
		if op.LayerPtr != nil {
			layerW := int(geom.W)
			if op.LayerWidth > 0 {
				layerW = int(op.LayerWidth)
			}
			op.LayerPtr.SetViewport(layerW, int(geom.H))
			op.LayerPtr.blit(buf, int(absX), int(absY), layerW, int(geom.H))
		}

	case OpContainer:
		// Draw border if present
		if op.Border.Horizontal != 0 {
			style := DefaultStyle()
			if op.BorderFG != nil {
				style.FG = *op.BorderFG
			}
			buf.DrawBorder(int(absX), int(absY), int(geom.W), int(geom.H), op.Border, style)

			if op.Title != "" {
				titleStr := "─ " + op.Title + " "
				buf.WriteStringFast(int(absX)+1, int(absY), titleStr, style, int(geom.W)-2)
			}
		}

		// Render children with this container's position as their origin
		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &t.ops[i]
			if childOp.Parent != idx {
				continue
			}
			t.renderOp(buf, i, absX, absY, geom.W)
		}

	case OpIf:
		// Render active branch if condition is true
		condTrue := (op.CondPtr != nil && *op.CondPtr) || (op.CondNode != nil && op.CondNode.evaluate())
		if op.ThenTmpl != nil && condTrue {
			op.ThenTmpl.render(buf, absX, absY, geom.W)
		} else if op.ElseTmpl != nil && !condTrue {
			op.ElseTmpl.render(buf, absX, absY, geom.W)
		}

	case OpForEach:
		// Render each item using iterGeoms for positioning
		if op.IterTmpl == nil || op.SlicePtr == nil {
			return
		}
		sliceHdr := *(*sliceHeader)(op.SlicePtr)
		if sliceHdr.Len == 0 {
			return
		}

		for i := 0; i < sliceHdr.Len && i < len(op.iterGeoms); i++ {
			itemGeom := &op.iterGeoms[i]
			itemAbsX := absX + itemGeom.LocalX
			itemAbsY := absY + itemGeom.LocalY

			// Rebind template ops to this element's data
			elemPtr := unsafe.Pointer(uintptr(sliceHdr.Data) + uintptr(i)*op.ElemSize)
			t.renderSubTemplate(buf, op.IterTmpl, itemAbsX, itemAbsY, itemGeom.W, elemPtr)
		}

	case OpSwitch:
		// Render matching case template
		var tmpl *Template
		matchIdx := op.SwitchNode.getMatchIndex()
		if matchIdx >= 0 && matchIdx < len(op.SwitchCases) {
			tmpl = op.SwitchCases[matchIdx]
		} else {
			tmpl = op.SwitchDef
		}
		if tmpl != nil {
			tmpl.render(buf, absX, absY, geom.W)
		}
	}
}

// renderSubTemplate renders a sub-template (for ForEach) with element-bound data.
func (t *Template) renderSubTemplate(buf *Buffer, sub *Template, globalX, globalY, maxW int16, elemBase unsafe.Pointer) {
	// Render root-level ops in sub-template
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

	switch op.Kind {
	case OpText:
		buf.WriteStringFast(int(absX), int(absY), op.StaticStr, Style{}, int(maxW))

	case OpTextPtr:
		buf.WriteStringFast(int(absX), int(absY), *op.StrPtr, Style{}, int(maxW))

	case OpTextOff:
		// Offset from element base
		strPtr := (*string)(unsafe.Pointer(uintptr(elemBase) + op.StrOff))
		buf.WriteStringFast(int(absX), int(absY), *strPtr, Style{}, int(maxW))

	case OpProgress:
		ratio := float32(op.StaticInt) / 100.0
		buf.WriteProgressBar(int(absX), int(absY), int(op.Width), ratio, Style{})

	case OpProgressPtr:
		ratio := float32(*op.IntPtr) / 100.0
		buf.WriteProgressBar(int(absX), int(absY), int(op.Width), ratio, Style{})

	case OpProgressOff:
		intPtr := (*int)(unsafe.Pointer(uintptr(elemBase) + op.IntOff))
		ratio := float32(*intPtr) / 100.0
		buf.WriteProgressBar(int(absX), int(absY), int(op.Width), ratio, Style{})

	case OpRichText:
		buf.WriteSpans(int(absX), int(absY), op.StaticSpans, int(maxW))

	case OpRichTextPtr:
		buf.WriteSpans(int(absX), int(absY), *op.SpansPtr, int(maxW))

	case OpRichTextOff:
		// Offset from element base
		spansPtr := (*[]Span)(unsafe.Pointer(uintptr(elemBase) + op.SpansOff))
		buf.WriteSpans(int(absX), int(absY), *spansPtr, int(maxW))

	case OpSelectionList:
		sub.renderSelectionList(buf, op, geom, absX, absY, maxW)

	case OpCustom:
		// Custom renderer draws itself
		if op.CustomRenderer != nil {
			op.CustomRenderer.Render(buf, int(absX), int(absY), int(geom.W), int(geom.H))
		}

	case OpLayout:
		// Custom layout renders children at their arranged positions
		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &sub.ops[i]
			if childOp.Parent != idx {
				continue
			}
			sub.renderSubOp(buf, i, absX, absY, geom.W, elemBase)
		}

	case OpLayer:
		// Blit the layer's visible portion to the buffer
		if op.LayerPtr != nil {
			layerW := int(geom.W)
			if op.LayerWidth > 0 {
				layerW = int(op.LayerWidth)
			}
			op.LayerPtr.SetViewport(layerW, int(geom.H))
			op.LayerPtr.blit(buf, int(absX), int(absY), layerW, int(geom.H))
		}

	case OpContainer:
		// Draw border if present
		if op.Border.Horizontal != 0 {
			style := DefaultStyle()
			if op.BorderFG != nil {
				style.FG = *op.BorderFG
			}
			buf.DrawBorder(int(absX), int(absY), int(geom.W), int(geom.H), op.Border, style)

			if op.Title != "" {
				titleStr := "─ " + op.Title + " "
				buf.WriteStringFast(int(absX)+1, int(absY), titleStr, style, int(geom.W)-2)
			}
		}

		// Recurse into children with this container's position as their origin
		for i := op.ChildStart; i < op.ChildEnd; i++ {
			childOp := &sub.ops[i]
			if childOp.Parent != idx {
				continue
			}
			sub.renderSubOp(buf, i, absX, absY, geom.W, elemBase)
		}

	case OpIf:
		// Use evaluateWithBase for conditions inside ForEach
		condTrue := (op.CondPtr != nil && *op.CondPtr) || (op.CondNode != nil && op.CondNode.evaluateWithBase(elemBase))
		if op.ThenTmpl != nil && condTrue {
			sub.renderSubTemplate(buf, op.ThenTmpl, absX, absY, geom.W, elemBase)
		} else if op.ElseTmpl != nil && !condTrue {
			sub.renderSubTemplate(buf, op.ElseTmpl, absX, absY, geom.W, elemBase)
		}

	case OpForEach:
		// Nested ForEach - render with nested element base
		if op.IterTmpl != nil && op.SlicePtr != nil {
			sliceHdr := *(*sliceHeader)(op.SlicePtr)
			for j := 0; j < sliceHdr.Len && j < len(op.iterGeoms); j++ {
				itemGeom := &op.iterGeoms[j]
				itemAbsX := absX + itemGeom.LocalX
				itemAbsY := absY + itemGeom.LocalY
				nestedElemPtr := unsafe.Pointer(uintptr(sliceHdr.Data) + uintptr(j)*op.ElemSize)
				sub.renderSubTemplate(buf, op.IterTmpl, itemAbsX, itemAbsY, itemGeom.W, nestedElemPtr)
			}
		}

	case OpSwitch:
		// Render matching case template within ForEach context
		var tmpl *Template
		matchIdx := op.SwitchNode.getMatchIndex()
		if matchIdx >= 0 && matchIdx < len(op.SwitchCases) {
			tmpl = op.SwitchCases[matchIdx]
		} else {
			tmpl = op.SwitchDef
		}
		if tmpl != nil {
			sub.renderSubTemplate(buf, tmpl, absX, absY, geom.W, elemBase)
		}
	}
}

// renderSelectionList renders a selection list with marker and windowing.
func (t *Template) renderSelectionList(buf *Buffer, op *Op, geom *Geom, absX, absY, maxW int16) {
	sliceHdr := *(*sliceHeader)(op.SlicePtr)
	if sliceHdr.Len == 0 {
		return
	}

	// Get selected index
	selectedIdx := -1
	if op.SelectedPtr != nil {
		selectedIdx = *op.SelectedPtr
	}

	// Calculate visible window
	startIdx := 0
	endIdx := sliceHdr.Len
	if op.SelectionListPtr != nil && op.SelectionListPtr.MaxVisible > 0 {
		startIdx = op.SelectionListPtr.offset
		endIdx = startIdx + op.SelectionListPtr.MaxVisible
		if endIdx > sliceHdr.Len {
			endIdx = sliceHdr.Len
		}
	}

	// Spaces for non-selected items (same width as marker)
	spaces := ""
	for i := int16(0); i < op.MarkerWidth; i++ {
		spaces += " "
	}

	contentW := int(maxW) - int(op.MarkerWidth)

	// Render visible items
	y := int(absY)
	for i := startIdx; i < endIdx; i++ {
		// Determine marker or spaces
		var markerText string
		if i == selectedIdx {
			markerText = op.Marker
		} else {
			markerText = spaces
		}

		// Write marker first
		buf.WriteStringFast(int(absX), y, markerText, Style{}, int(maxW))

		// Get content from iteration template
		if op.IterTmpl != nil && len(op.IterTmpl.ops) > 0 {
			elemPtr := unsafe.Pointer(uintptr(sliceHdr.Data) + uintptr(i)*op.ElemSize)

			// Render the first op from iteration template (usually a Text)
			iterOp := &op.IterTmpl.ops[0]
			contentX := int(absX) + int(op.MarkerWidth)

			switch iterOp.Kind {
			case OpText:
				buf.WriteStringFast(contentX, y, iterOp.StaticStr, Style{}, contentW)
			case OpTextPtr:
				buf.WriteStringFast(contentX, y, *iterOp.StrPtr, Style{}, contentW)
			case OpTextOff:
				strPtr := (*string)(unsafe.Pointer(uintptr(elemPtr) + iterOp.StrOff))
				buf.WriteStringFast(contentX, y, *strPtr, Style{}, contentW)
			case OpRichText:
				buf.WriteSpans(contentX, y, iterOp.StaticSpans, contentW)
			case OpRichTextPtr:
				buf.WriteSpans(contentX, y, *iterOp.SpansPtr, contentW)
			case OpRichTextOff:
				spansPtr := (*[]Span)(unsafe.Pointer(uintptr(elemPtr) + iterOp.SpansOff))
				buf.WriteSpans(contentX, y, *spansPtr, contentW)
			}
		}
		y++
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

