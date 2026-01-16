package tui

import (
	"reflect"
	"unicode/utf8"
	"unsafe"
)

// SerialTemplate is a compiled, phase-aware UI template.
// Build does all reflection. Execute is pure pointer arithmetic across three phases.
type SerialTemplate struct {
	ops      []SerialOp
	byLevel  [][]int16 // byLevel[depth] = op indices at that depth
	maxLevel int
	nodes    []SerialNode    // pre-allocated node storage
	ctxStack []layoutContext // pre-allocated layout context stack (reused each frame)
	geom     []opGeom        // computed geometry per op (reused each frame)
}

// opGeom stores computed layout geometry for an op during execute
type opGeom struct {
	X, Y, W, H   int16 // computed position and size
	contentH     int16 // content height before flex grow
	borderOffset int8  // 1 if bordered, 0 otherwise
}

// SerialNode is minimal runtime node data
type SerialNode struct {
	Kind uint8 // for render dispatch

	// Geometry
	W, H int16
	X, Y int16

	// Values (populated during measure)
	Text  string
	Spans []Span  // for RichText
	Ratio float32
	Width int16 // fixed width for progress bars

	// Layer reference (for blit during render)
	Layer *Layer

	// Custom component render function
	CustomRender func(buf *Buffer, x, y, w, h int16)

	// Styling
	Bold bool
}

// SerialOp represents a single operation. Kind encodes WHAT and HOW (no runtime mode switches).
type SerialOp struct {
	Kind   uint8
	Level  int8
	Parent int16 // parent op index

	// Value access - exactly one used based on Kind
	StaticStr string
	StrPtr    *string
	StrOff    uintptr

	StaticInt int
	IntPtr    *int
	IntOff    uintptr

	BoolPtr *bool

	// Layout hints
	Width        int16
	Height       int16       // explicit height
	PercentWidth float32     // fraction of parent width
	FlexGrow     float32     // share of remaining space
	Border       BorderStyle // border for containers
	Title        string      // title for bordered containers
	BorderFG     *Color      // border foreground color
	IsRow        bool
	Gap          int8
	Bold         bool
	Style        Style

	// ForEach
	SlicePtr unsafe.Pointer
	SliceOff uintptr // offset from elemBase for nested ForEach
	ElemSize uintptr
	IterTmpl *SerialTemplate // sub-template for each element

	// Conditional branches
	ThenTmpl *SerialTemplate
	ElseTmpl *SerialTemplate

	// RichText
	Spans    []Span
	SpansPtr *[]Span
	SpansOff uintptr // offset for ForEach context

	// Layer
	LayerPtr    *Layer
	LayerHeight int16
	LayerWidth  int16

	// Generic condition (If[T])
	CondNode ConditionNode
	CondThen *SerialTemplate
	CondElse *SerialTemplate

	// Switch
	SwitchNode  SwitchNodeInterface
	SwitchCases []*SerialTemplate
	SwitchDef   *SerialTemplate

	// SelectionList
	SelectionListPtr *SelectionList // pointer to the SelectionList for len updates
	SelectedPtr      *int           // pointer to selected index
	Marker           string         // selection marker
	MarkerWidth      int16          // cached marker width

	// Leader
	LeaderLabelStr string   // static label
	LeaderLabelPtr *string  // label pointer
	LeaderValueStr string   // static value
	LeaderValuePtr *string  // value pointer
	LeaderWidth    int16    // total width (0 = fill available)
	LeaderFill     rune     // fill character
	LeaderStyle    Style    // styling

	// Custom component
	CustomMeasure func(availW int16) (w, h int16)
	CustomRender  func(buf *Buffer, x, y, w, h int16)
}

// Op kinds - each encodes both WHAT and HOW to access values
const (
	SerialOpTextStatic uint8 = iota
	SerialOpTextPtr
	SerialOpTextOffset

	SerialOpProgressStatic
	SerialOpProgressPtr
	SerialOpProgressOffset

	SerialOpContainerStart
	SerialOpContainerEnd

	SerialOpForEach
	SerialOpForEachOffset // for nested ForEach within parent element

	SerialOpIf
	SerialOpElse

	SerialOpRichTextStatic
	SerialOpRichTextPtr
	SerialOpRichTextOffset // for ForEach context

	SerialOpLayer

	SerialOpCondition

	SerialOpSwitch

	SerialOpSelectionList

	SerialOpLeaderStatic // both Label and Value are static strings
	SerialOpLeaderPtr    // at least one is a pointer

	SerialOpCustom // user-defined component (function pointer path)
)

// BuildSerial compiles a declarative UI into a SerialTemplate
func BuildSerial(ui any) *SerialTemplate {
	t := &SerialTemplate{
		ops:     make([]SerialOp, 0, 64),
		byLevel: make([][]int16, 16), // pre-alloc 16 levels
	}

	// Initialize level buckets
	for i := range t.byLevel {
		t.byLevel[i] = make([]int16, 0, 8)
	}

	t.compile(ui, -1, 0, nil, 0)

	// Trim unused levels
	for t.maxLevel >= 0 && len(t.byLevel[t.maxLevel]) == 0 {
		t.maxLevel--
	}
	t.byLevel = t.byLevel[:t.maxLevel+1]

	// Pre-allocate nodes (estimate: same as ops count)
	t.nodes = make([]SerialNode, 0, len(t.ops))

	return t
}

func (t *SerialTemplate) addOp(op SerialOp, level int) int16 {
	idx := int16(len(t.ops))
	op.Level = int8(level)
	t.ops = append(t.ops, op)

	// Add to level bucket
	if level >= len(t.byLevel) {
		// Extend byLevel if needed
		for len(t.byLevel) <= level {
			t.byLevel = append(t.byLevel, make([]int16, 0, 8))
		}
	}
	t.byLevel[level] = append(t.byLevel[level], idx)

	if level > t.maxLevel {
		t.maxLevel = level
	}

	return idx
}

func (t *SerialTemplate) compile(node any, parentIdx int16, level int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	if node == nil {
		return -1
	}

	switch v := node.(type) {
	case Text:
		return t.compileText(v, parentIdx, level, elemBase, elemSize)
	case Progress:
		return t.compileProgress(v, parentIdx, level, elemBase, elemSize)
	case Row:
		return t.compileContainer(v.flex, v.Gap, v.Children, v.border, v.Title, v.borderFG, parentIdx, level, true, elemBase, elemSize)
	case Col:
		return t.compileContainer(v.flex, v.Gap, v.Children, v.border, v.Title, v.borderFG, parentIdx, level, false, elemBase, elemSize)
	case IfNode:
		return t.compileIf(v, parentIdx, level, elemBase, elemSize)
	case ElseNode:
		return t.compileElse(v, parentIdx, level, elemBase, elemSize)
	case ForEachNode:
		return t.compileForEach(v, parentIdx, level, elemBase, elemSize)
	case SelectionList:
		return t.compileSelectionList(&v, parentIdx, level)
	case *SelectionList:
		return t.compileSelectionList(v, parentIdx, level)
	case RichText:
		return t.compileRichText(v, parentIdx, level, elemBase, elemSize)
	case LayerView:
		return t.compileLayer(v, parentIdx, level)
	case Leader:
		return t.compileLeader(v, parentIdx, level, elemBase, elemSize)
	case Custom:
		return t.compileCustom(v, parentIdx, level)
	default:
		// Check for ConditionNode (generic If conditions)
		if cond, ok := node.(ConditionNode); ok {
			return t.compileCondition(cond, parentIdx, level, elemBase, elemSize)
		}
		// Check for SwitchNodeInterface (generic Switch)
		if sw, ok := node.(SwitchNodeInterface); ok {
			return t.compileSwitch(sw, parentIdx, level, elemBase, elemSize)
		}
	}

	return -1
}

func (t *SerialTemplate) compileText(v Text, parentIdx int16, level int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	op := SerialOp{
		Parent: parentIdx,
		Bold:   v.Style.Attr&AttrBold != 0,
		Style:  v.Style,
	}

	// Determine access pattern at compile time - no runtime mode switch
	switch val := v.Content.(type) {
	case string:
		op.Kind = SerialOpTextStatic
		op.StaticStr = val
	case *string:
		if elemBase != nil && isWithinRange(unsafe.Pointer(val), elemBase, elemSize) {
			op.Kind = SerialOpTextOffset
			op.StrOff = uintptr(unsafe.Pointer(val)) - uintptr(elemBase)
		} else {
			op.Kind = SerialOpTextPtr
			op.StrPtr = val
		}
	default:
		// Handle reflection case for other pointer types
		rv := reflect.ValueOf(v.Content)
		if rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.String {
			ptr := (*string)(unsafe.Pointer(rv.Pointer()))
			if elemBase != nil && isWithinRange(unsafe.Pointer(ptr), elemBase, elemSize) {
				op.Kind = SerialOpTextOffset
				op.StrOff = uintptr(unsafe.Pointer(ptr)) - uintptr(elemBase)
			} else {
				op.Kind = SerialOpTextPtr
				op.StrPtr = ptr
			}
		} else {
			op.Kind = SerialOpTextStatic
			op.StaticStr = ""
		}
	}

	return t.addOp(op, level)
}

func (t *SerialTemplate) compileProgress(v Progress, parentIdx int16, level int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	width := v.BarWidth
	if width == 0 {
		width = 20
	}

	op := SerialOp{
		Parent: parentIdx,
		Width:  width,
	}

	switch val := v.Value.(type) {
	case int:
		op.Kind = SerialOpProgressStatic
		op.StaticInt = val
	case *int:
		if elemBase != nil && isWithinRange(unsafe.Pointer(val), elemBase, elemSize) {
			op.Kind = SerialOpProgressOffset
			op.IntOff = uintptr(unsafe.Pointer(val)) - uintptr(elemBase)
		} else {
			op.Kind = SerialOpProgressPtr
			op.IntPtr = val
		}
	default:
		rv := reflect.ValueOf(v.Value)
		if rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.Int {
			ptr := (*int)(unsafe.Pointer(rv.Pointer()))
			if elemBase != nil && isWithinRange(unsafe.Pointer(ptr), elemBase, elemSize) {
				op.Kind = SerialOpProgressOffset
				op.IntOff = uintptr(unsafe.Pointer(ptr)) - uintptr(elemBase)
			} else {
				op.Kind = SerialOpProgressPtr
				op.IntPtr = ptr
			}
		} else {
			op.Kind = SerialOpProgressStatic
			op.StaticInt = 0
		}
	}

	return t.addOp(op, level)
}

func (t *SerialTemplate) compileContainer(f flex, gap int8, children []any, border BorderStyle, title string, borderFG *Color, parentIdx int16, level int, isRow bool, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	startIdx := t.addOp(SerialOp{
		Kind:         SerialOpContainerStart,
		Parent:       parentIdx,
		IsRow:        isRow,
		Gap:          gap,
		PercentWidth: f.percentWidth,
		Width:        f.width,
		Height:       f.height,
		FlexGrow:     f.flexGrow,
		Border:       border,
		Title:        title,
		BorderFG:     borderFG,
	}, level)

	for _, child := range children {
		t.compile(child, startIdx, level+1, elemBase, elemSize)
	}

	t.addOp(SerialOp{
		Kind:   SerialOpContainerEnd,
		Parent: startIdx, // points back to start
	}, level)

	return startIdx
}

func (t *SerialTemplate) compileIf(v IfNode, parentIdx int16, level int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	op := SerialOp{
		Kind:   SerialOpIf,
		Parent: parentIdx,
	}

	// Compile condition
	switch val := v.Cond.(type) {
	case *bool:
		op.BoolPtr = val
	default:
		rv := reflect.ValueOf(v.Cond)
		if rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.Bool {
			op.BoolPtr = (*bool)(unsafe.Pointer(rv.Pointer()))
		}
	}

	// Compile then branch as sub-template
	if v.Then != nil {
		thenTmpl := &SerialTemplate{
			ops:     make([]SerialOp, 0, 16),
			byLevel: make([][]int16, 8),
		}
		for i := range thenTmpl.byLevel {
			thenTmpl.byLevel[i] = make([]int16, 0, 4)
		}
		thenTmpl.compile(v.Then, -1, 0, elemBase, elemSize)
		thenTmpl.byLevel = thenTmpl.byLevel[:thenTmpl.maxLevel+1]
		thenTmpl.nodes = make([]SerialNode, 0, len(thenTmpl.ops))
		op.ThenTmpl = thenTmpl
	}

	return t.addOp(op, level)
}

func (t *SerialTemplate) compileElse(v ElseNode, parentIdx int16, level int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	op := SerialOp{
		Kind:   SerialOpElse,
		Parent: parentIdx,
	}

	// Compile else branch as sub-template
	if v.Then != nil {
		elseTmpl := &SerialTemplate{
			ops:     make([]SerialOp, 0, 16),
			byLevel: make([][]int16, 8),
		}
		for i := range elseTmpl.byLevel {
			elseTmpl.byLevel[i] = make([]int16, 0, 4)
		}
		elseTmpl.compile(v.Then, -1, 0, elemBase, elemSize)
		elseTmpl.byLevel = elseTmpl.byLevel[:elseTmpl.maxLevel+1]
		elseTmpl.nodes = make([]SerialNode, 0, len(elseTmpl.ops))
		op.ElseTmpl = elseTmpl
	}

	return t.addOp(op, level)
}

func (t *SerialTemplate) compileRichText(v RichText, parentIdx int16, level int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	op := SerialOp{
		Parent: parentIdx,
	}

	switch spans := v.Spans.(type) {
	case []Span:
		op.Kind = SerialOpRichTextStatic
		op.Spans = spans
	case *[]Span:
		// Check if pointer is within ForEach element range
		if elemBase != nil && isWithinRange(unsafe.Pointer(spans), elemBase, elemSize) {
			op.Kind = SerialOpRichTextOffset
			op.SpansOff = uintptr(unsafe.Pointer(spans)) - uintptr(elemBase)
		} else {
			op.Kind = SerialOpRichTextPtr
			op.SpansPtr = spans
		}
	default:
		// Empty RichText
		op.Kind = SerialOpRichTextStatic
		op.Spans = nil
	}

	return t.addOp(op, level)
}

func (t *SerialTemplate) compileLayer(v LayerView, parentIdx int16, level int) int16 {
	op := SerialOp{
		Kind:        SerialOpLayer,
		Parent:      parentIdx,
		LayerPtr:    v.Layer,
		LayerHeight: v.ViewHeight,
		LayerWidth:  v.ViewWidth,
	}
	return t.addOp(op, level)
}

func (t *SerialTemplate) compileLeader(v Leader, parentIdx int16, level int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	op := SerialOp{
		Parent:      parentIdx,
		LeaderWidth: v.Width,
		LeaderFill:  v.Fill,
		LeaderStyle: v.Style,
	}

	// Default fill character
	if op.LeaderFill == 0 {
		op.LeaderFill = '.'
	}

	// Determine if static or pointer-based
	labelIsPtr := false
	valueIsPtr := false

	// Handle Label
	switch val := v.Label.(type) {
	case string:
		op.LeaderLabelStr = val
	case *string:
		op.LeaderLabelPtr = val
		labelIsPtr = true
	}

	// Handle Value
	switch val := v.Value.(type) {
	case string:
		op.LeaderValueStr = val
	case *string:
		op.LeaderValuePtr = val
		valueIsPtr = true
	}

	// Set kind based on whether we have any pointers
	if labelIsPtr || valueIsPtr {
		op.Kind = SerialOpLeaderPtr
	} else {
		op.Kind = SerialOpLeaderStatic
	}

	return t.addOp(op, level)
}

func (t *SerialTemplate) compileCustom(v Custom, parentIdx int16, level int) int16 {
	op := SerialOp{
		Kind:          SerialOpCustom,
		Parent:        parentIdx,
		CustomMeasure: v.Measure,
		CustomRender:  v.Render,
	}
	return t.addOp(op, level)
}

func (t *SerialTemplate) compileCondition(cond ConditionNode, parentIdx int16, level int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	// If inside ForEach, calculate the offset for pointer adjustment at runtime
	if elemBase != nil {
		ptrAddr := cond.getPtrAddr()
		elemAddr := uintptr(elemBase)
		if ptrAddr >= elemAddr && ptrAddr < elemAddr+elemSize {
			// Pointer is within the element - store offset for runtime adjustment
			cond.setOffset(ptrAddr - elemAddr)
		}
	}

	op := SerialOp{
		Kind:     SerialOpCondition,
		Parent:   parentIdx,
		CondNode: cond,
	}

	// Compile then branch
	if thenNode := cond.getThen(); thenNode != nil {
		thenTmpl := &SerialTemplate{
			ops:     make([]SerialOp, 0, 16),
			byLevel: make([][]int16, 8),
		}
		for i := range thenTmpl.byLevel {
			thenTmpl.byLevel[i] = make([]int16, 0, 4)
		}
		thenTmpl.compile(thenNode, -1, 0, elemBase, elemSize)
		thenTmpl.byLevel = thenTmpl.byLevel[:thenTmpl.maxLevel+1]
		thenTmpl.nodes = make([]SerialNode, 0, len(thenTmpl.ops))
		op.CondThen = thenTmpl
	}

	// Compile else branch
	if elseNode := cond.getElse(); elseNode != nil {
		elseTmpl := &SerialTemplate{
			ops:     make([]SerialOp, 0, 16),
			byLevel: make([][]int16, 8),
		}
		for i := range elseTmpl.byLevel {
			elseTmpl.byLevel[i] = make([]int16, 0, 4)
		}
		elseTmpl.compile(elseNode, -1, 0, elemBase, elemSize)
		elseTmpl.byLevel = elseTmpl.byLevel[:elseTmpl.maxLevel+1]
		elseTmpl.nodes = make([]SerialNode, 0, len(elseTmpl.ops))
		op.CondElse = elseTmpl
	}

	return t.addOp(op, level)
}

func (t *SerialTemplate) compileSwitch(sw SwitchNodeInterface, parentIdx int16, level int, elemBase unsafe.Pointer, elemSize uintptr) int16 {
	op := SerialOp{
		Kind:       SerialOpSwitch,
		Parent:     parentIdx,
		SwitchNode: sw,
	}

	// Compile each case branch
	caseNodes := sw.getCaseNodes()
	op.SwitchCases = make([]*SerialTemplate, len(caseNodes))
	for i, caseNode := range caseNodes {
		if caseNode != nil {
			caseTmpl := &SerialTemplate{
				ops:     make([]SerialOp, 0, 16),
				byLevel: make([][]int16, 8),
			}
			for j := range caseTmpl.byLevel {
				caseTmpl.byLevel[j] = make([]int16, 0, 4)
			}
			caseTmpl.compile(caseNode, -1, 0, elemBase, elemSize)
			caseTmpl.byLevel = caseTmpl.byLevel[:caseTmpl.maxLevel+1]
			caseTmpl.nodes = make([]SerialNode, 0, len(caseTmpl.ops))
			op.SwitchCases[i] = caseTmpl
		}
	}

	// Compile default branch
	if defNode := sw.getDefaultNode(); defNode != nil {
		defTmpl := &SerialTemplate{
			ops:     make([]SerialOp, 0, 16),
			byLevel: make([][]int16, 8),
		}
		for i := range defTmpl.byLevel {
			defTmpl.byLevel[i] = make([]int16, 0, 4)
		}
		defTmpl.compile(defNode, -1, 0, elemBase, elemSize)
		defTmpl.byLevel = defTmpl.byLevel[:defTmpl.maxLevel+1]
		defTmpl.nodes = make([]SerialNode, 0, len(defTmpl.ops))
		op.SwitchDef = defTmpl
	}

	return t.addOp(op, level)
}

func (t *SerialTemplate) compileForEach(v ForEachNode, parentIdx int16, level int, parentElemBase unsafe.Pointer, parentElemSize uintptr) int16 {
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
	iterTmpl := &SerialTemplate{
		ops:     make([]SerialOp, 0, 16),
		byLevel: make([][]int16, 8),
	}
	for i := range iterTmpl.byLevel {
		iterTmpl.byLevel[i] = make([]int16, 0, 4)
	}
	iterTmpl.compile(templateResult, -1, 0, dummyBase, elemSize)
	iterTmpl.byLevel = iterTmpl.byLevel[:iterTmpl.maxLevel+1]
	iterTmpl.nodes = make([]SerialNode, 0, len(iterTmpl.ops))

	// Determine if this is a nested ForEach (slice ptr is within parent element)
	opKind := uint8(SerialOpForEach)
	var sliceOff uintptr
	if parentElemBase != nil && isWithinRange(slicePtr, parentElemBase, parentElemSize) {
		opKind = SerialOpForEachOffset
		sliceOff = uintptr(slicePtr) - uintptr(parentElemBase)
		slicePtr = nil // don't store absolute pointer for offset-based
	}

	op := SerialOp{
		Kind:     opKind,
		Parent:   parentIdx,
		SlicePtr: slicePtr,
		SliceOff: sliceOff,
		ElemSize: elemSize,
		IterTmpl: iterTmpl,
	}

	return t.addOp(op, level)
}

func (t *SerialTemplate) compileSelectionList(v *SelectionList, parentIdx int16, level int) int16 {
	// Analyze slice
	sliceRV := reflect.ValueOf(v.Items)
	if sliceRV.Kind() != reflect.Ptr {
		panic("SelectionList Items must be pointer to slice")
	}
	sliceType := sliceRV.Type().Elem()
	if sliceType.Kind() != reflect.Slice {
		panic("SelectionList Items must be pointer to slice")
	}
	elemType := sliceType.Elem()
	elemSize := elemType.Size()
	slicePtr := unsafe.Pointer(sliceRV.Pointer())

	// Default marker
	marker := v.Marker
	if marker == "" {
		marker = "> "
	}
	markerWidth := int16(len(marker))

	// Create iteration template
	var iterTmpl *SerialTemplate
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
		iterTmpl = &SerialTemplate{
			ops:     make([]SerialOp, 0, 16),
			byLevel: make([][]int16, 8),
		}
		for i := range iterTmpl.byLevel {
			iterTmpl.byLevel[i] = make([]int16, 0, 4)
		}
		iterTmpl.compile(templateResult, -1, 0, dummyBase, elemSize)
		iterTmpl.byLevel = iterTmpl.byLevel[:iterTmpl.maxLevel+1]
		iterTmpl.nodes = make([]SerialNode, 0, len(iterTmpl.ops))
	}

	op := SerialOp{
		Kind:             SerialOpSelectionList,
		Parent:           parentIdx,
		SlicePtr:         slicePtr,
		ElemSize:         elemSize,
		IterTmpl:         iterTmpl,
		SelectionListPtr: v,
		SelectedPtr:      v.Selected,
		Marker:           marker,
		MarkerWidth:      markerWidth,
	}

	return t.addOp(op, level)
}

// measureOp creates nodes with natural sizes.
// elemBase is only used for ForEach iterations (element base pointer for offset access).
func (t *SerialTemplate) measureOp(opIdx int16, elemBase unsafe.Pointer, ifSatisfied *bool) {
	op := &t.ops[opIdx]

	switch op.Kind {
	case SerialOpTextStatic:
		t.measureTextStatic(op)
	case SerialOpTextPtr:
		t.measureTextPtr(op)
	case SerialOpTextOffset:
		t.measureTextOffset(op, elemBase)
	case SerialOpProgressStatic:
		t.measureProgressStatic(op)
	case SerialOpProgressPtr:
		t.measureProgressPtr(op)
	case SerialOpProgressOffset:
		t.measureProgressOffset(op, elemBase)
	case SerialOpContainerStart:
		t.measureContainerStart(op)
	case SerialOpContainerEnd:
		t.measureContainerEnd(op)
	case SerialOpForEach:
		t.measureForEach(op, nil)
	case SerialOpForEachOffset:
		t.measureForEach(op, elemBase)
	case SerialOpIf:
		t.measureIf(op, ifSatisfied)
	case SerialOpElse:
		t.measureElse(op, ifSatisfied)
	case SerialOpRichTextStatic:
		t.measureRichTextStatic(op)
	case SerialOpRichTextPtr:
		t.measureRichTextPtr(op)
	case SerialOpRichTextOffset:
		t.measureRichTextOffset(op, elemBase)
	case SerialOpLayer:
		t.measureLayer(op)
	case SerialOpCondition:
		t.measureCondition(op, elemBase)
	case SerialOpSwitch:
		t.measureSwitch(op)
	case SerialOpSelectionList:
		t.measureSelectionList(op)
	case SerialOpLeaderStatic:
		t.measureLeaderStatic(op)
	case SerialOpLeaderPtr:
		t.measureLeaderPtr(op)
	case SerialOpCustom:
		t.measureCustom(op)
	}
}

func (t *SerialTemplate) measureTextStatic(op *SerialOp) {
	t.nodes = append(t.nodes, SerialNode{
		Kind: op.Kind,
		Text: op.StaticStr,
		W:    int16(utf8.RuneCountInString(op.StaticStr)),
		H:    1,
		Bold: op.Bold,
	})
}

func (t *SerialTemplate) measureTextPtr(op *SerialOp) {
	text := *op.StrPtr
	t.nodes = append(t.nodes, SerialNode{
		Kind: op.Kind,
		Text: text,
		W:    int16(utf8.RuneCountInString(text)),
		H:    1,
		Bold: op.Bold,
	})
}

func (t *SerialTemplate) measureTextOffset(op *SerialOp, elemBase unsafe.Pointer) {
	text := *(*string)(unsafe.Add(elemBase, op.StrOff))
	t.nodes = append(t.nodes, SerialNode{
		Kind: op.Kind,
		Text: text,
		W:    int16(utf8.RuneCountInString(text)),
		H:    1,
		Bold: op.Bold,
	})
}

func (t *SerialTemplate) measureProgressStatic(op *SerialOp) {
	t.nodes = append(t.nodes, SerialNode{
		Kind:  op.Kind,
		Ratio: float32(op.StaticInt) / 100.0,
		W:     op.Width,
		H:     1,
		Width: op.Width,
	})
}

func (t *SerialTemplate) measureProgressPtr(op *SerialOp) {
	t.nodes = append(t.nodes, SerialNode{
		Kind:  op.Kind,
		Ratio: float32(*op.IntPtr) / 100.0,
		W:     op.Width,
		H:     1,
		Width: op.Width,
	})
}

func (t *SerialTemplate) measureProgressOffset(op *SerialOp, elemBase unsafe.Pointer) {
	val := *(*int)(unsafe.Add(elemBase, op.IntOff))
	t.nodes = append(t.nodes, SerialNode{
		Kind:  op.Kind,
		Ratio: float32(val) / 100.0,
		W:     op.Width,
		H:     1,
		Width: op.Width,
	})
}

func (t *SerialTemplate) measureContainerStart(op *SerialOp) {
	// Container nodes are not rendered in simple vertical layout
}

func (t *SerialTemplate) measureContainerEnd(op *SerialOp) {
	// Container end doesn't create a node, just signals end of children
	// Actual measurement happens in layout phase
}

func (t *SerialTemplate) measureForEach(op *SerialOp, elemBase unsafe.Pointer) {
	// For offset-based ForEach (nested within parent element), compute actual slice ptr
	var slicePtr unsafe.Pointer
	if op.Kind == SerialOpForEachOffset {
		if elemBase == nil {
			return // Can't resolve offset without elemBase
		}
		slicePtr = unsafe.Add(elemBase, op.SliceOff)
	} else {
		slicePtr = op.SlicePtr
	}

	sliceHdr := *(*sliceHeader)(slicePtr)
	if sliceHdr.Len == 0 || op.IterTmpl == nil {
		return
	}

	// Track accumulated height across iterations (for stacking rows vertically)
	accumulatedY := int16(0)

	// Execute sub-template for each element with proper layout
	for i := 0; i < sliceHdr.Len; i++ {
		elemPtr := unsafe.Add(sliceHdr.Data, uintptr(i)*op.ElemSize)

		// Reset sub-template nodes
		op.IterTmpl.nodes = op.IterTmpl.nodes[:0]

		// Track layout context for containers within each iteration
		ctxStack := make([]layoutContext, 0, 4)
		ctxStack = append(ctxStack, layoutContext{x: 0, y: 0, isRow: false, firstChild: true})

		ifSat := false
		iterMaxH := int16(1) // Track max height for this iteration

		// Process ops in document order with layout tracking
		for j := range op.IterTmpl.ops {
			iterOp := &op.IterTmpl.ops[j]

			switch iterOp.Kind {
			case SerialOpContainerStart:
				ctx := ctxStack[len(ctxStack)-1]
				ctxStack = append(ctxStack, layoutContext{
					x:          ctx.x,
					y:          ctx.y,
					startX:     ctx.x,
					startY:     ctx.y,
					isRow:      iterOp.IsRow,
					gap:        iterOp.Gap,
					firstChild: true,
				})

			case SerialOpContainerEnd:
				childCtx := ctxStack[len(ctxStack)-1]
				ctxStack = ctxStack[:len(ctxStack)-1]

				// Calculate container dimensions
				var containerW, containerH int16
				if childCtx.isRow {
					containerW = childCtx.x - childCtx.startX
					containerH = childCtx.maxH
					if containerH == 0 {
						containerH = 1
					}
				} else {
					containerW = childCtx.maxW
					containerH = childCtx.y - childCtx.startY
				}

				// Track iteration max height
				if containerH > iterMaxH {
					iterMaxH = containerH
				}

				// Update parent context
				if len(ctxStack) > 0 {
					parentIdx := len(ctxStack) - 1
					if ctxStack[parentIdx].isRow {
						if !ctxStack[parentIdx].firstChild && ctxStack[parentIdx].gap > 0 {
							ctxStack[parentIdx].x += int16(ctxStack[parentIdx].gap)
						}
						ctxStack[parentIdx].x += containerW
						if containerH > ctxStack[parentIdx].maxH {
							ctxStack[parentIdx].maxH = containerH
						}
					} else {
						if !ctxStack[parentIdx].firstChild && ctxStack[parentIdx].gap > 0 {
							ctxStack[parentIdx].y += int16(ctxStack[parentIdx].gap)
						}
						ctxStack[parentIdx].y += containerH
						if containerW > ctxStack[parentIdx].maxW {
							ctxStack[parentIdx].maxW = containerW
						}
					}
					ctxStack[parentIdx].firstChild = false
				}

			default:
				// Measure the op and position any new nodes
				nodeStart := len(op.IterTmpl.nodes)
				op.IterTmpl.measureOp(int16(j), elemPtr, &ifSat)

				// Position new nodes within current container
				if len(ctxStack) > 0 {
					ctxIdx := len(ctxStack) - 1
					for k := nodeStart; k < len(op.IterTmpl.nodes); k++ {
						node := &op.IterTmpl.nodes[k]

						if !ctxStack[ctxIdx].firstChild && ctxStack[ctxIdx].gap > 0 {
							if ctxStack[ctxIdx].isRow {
								ctxStack[ctxIdx].x += int16(ctxStack[ctxIdx].gap)
							} else {
								ctxStack[ctxIdx].y += int16(ctxStack[ctxIdx].gap)
							}
						}

						node.X = ctxStack[ctxIdx].x
						node.Y = ctxStack[ctxIdx].y

						if ctxStack[ctxIdx].isRow {
							ctxStack[ctxIdx].x += node.W
							if node.H > ctxStack[ctxIdx].maxH {
								ctxStack[ctxIdx].maxH = node.H
							}
						} else {
							ctxStack[ctxIdx].y += node.H
							if node.W > ctxStack[ctxIdx].maxW {
								ctxStack[ctxIdx].maxW = node.W
							}
						}
						ctxStack[ctxIdx].firstChild = false

						// Track max height for this iteration
						if node.Y+node.H > iterMaxH {
							iterMaxH = node.Y + node.H
						}
					}
				}
			}
		}

		// Copy nodes to main template with Y offset for this iteration
		for _, node := range op.IterTmpl.nodes {
			node.Y += accumulatedY
			t.nodes = append(t.nodes, node)
		}

		// Accumulate height for next iteration
		accumulatedY += iterMaxH
	}
}

func (t *SerialTemplate) measureSelectionList(op *SerialOp) {
	sliceHdr := *(*sliceHeader)(op.SlicePtr)

	// Update len in SelectionList for bounds checking in helper methods
	if op.SelectionListPtr != nil {
		op.SelectionListPtr.len = sliceHdr.Len
		// Ensure selection is visible (handles data changes)
		op.SelectionListPtr.ensureVisible()
	}

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

	// Render only visible items: marker prepended to content
	for i := startIdx; i < endIdx; i++ {
		// Determine marker or spaces
		var markerText string
		if i == selectedIdx {
			markerText = op.Marker
		} else {
			markerText = spaces
		}

		// Get content from Render (if provided)
		if op.IterTmpl != nil {
			elemPtr := unsafe.Add(sliceHdr.Data, uintptr(i)*op.ElemSize)

			// Reset sub-template nodes
			op.IterTmpl.nodes = op.IterTmpl.nodes[:0]

			ifSat := false
			for level := 0; level <= op.IterTmpl.maxLevel; level++ {
				for _, idx := range op.IterTmpl.byLevel[level] {
					op.IterTmpl.measureOp(idx, elemPtr, &ifSat)
				}
			}

			// Prepend marker to first node's text, copy all nodes
			for j, node := range op.IterTmpl.nodes {
				if j == 0 {
					// Prepend marker to first node
					node.Text = markerText + node.Text
					node.W += op.MarkerWidth
				}
				t.nodes = append(t.nodes, node)
			}
		} else {
			// No render function, just show marker
			t.nodes = append(t.nodes, SerialNode{
				Kind: SerialOpTextStatic,
				Text: markerText,
				W:    op.MarkerWidth,
				H:    1,
			})
		}
	}
}

func (t *SerialTemplate) measureIf(op *SerialOp, ifSatisfied *bool) {
	if op.BoolPtr != nil && *op.BoolPtr {
		*ifSatisfied = true
		if op.ThenTmpl != nil {
			op.ThenTmpl.nodes = op.ThenTmpl.nodes[:0]
			ifSat := false
			for level := 0; level <= op.ThenTmpl.maxLevel; level++ {
				for _, idx := range op.ThenTmpl.byLevel[level] {
					op.ThenTmpl.measureOp(idx, nil, &ifSat)
				}
			}
			for _, node := range op.ThenTmpl.nodes {
				t.nodes = append(t.nodes, node)
			}
		}
	} else {
		*ifSatisfied = false
	}
}

func (t *SerialTemplate) measureElse(op *SerialOp, ifSatisfied *bool) {
	if !*ifSatisfied {
		if op.ElseTmpl != nil {
			op.ElseTmpl.nodes = op.ElseTmpl.nodes[:0]
			ifSat := false
			for level := 0; level <= op.ElseTmpl.maxLevel; level++ {
				for _, idx := range op.ElseTmpl.byLevel[level] {
					op.ElseTmpl.measureOp(idx, nil, &ifSat)
				}
			}
			for _, node := range op.ElseTmpl.nodes {
				t.nodes = append(t.nodes, node)
			}
		}
	}
}

func (t *SerialTemplate) measureRichTextStatic(op *SerialOp) {
	w := int16(0)
	for _, span := range op.Spans {
		w += int16(utf8.RuneCountInString(span.Text))
	}
	t.nodes = append(t.nodes, SerialNode{
		Kind:  SerialOpRichTextStatic,
		Spans: op.Spans,
		W:     w,
		H:     1,
	})
}

func (t *SerialTemplate) measureRichTextPtr(op *SerialOp) {
	spans := *op.SpansPtr
	w := int16(0)
	for _, span := range spans {
		w += int16(utf8.RuneCountInString(span.Text))
	}
	t.nodes = append(t.nodes, SerialNode{
		Kind:  SerialOpRichTextPtr,
		Spans: spans,
		W:     w,
		H:     1,
	})
}

func (t *SerialTemplate) measureRichTextOffset(op *SerialOp, elemBase unsafe.Pointer) {
	spans := *(*[]Span)(unsafe.Add(elemBase, op.SpansOff))
	w := int16(0)
	for _, span := range spans {
		w += int16(utf8.RuneCountInString(span.Text))
	}
	t.nodes = append(t.nodes, SerialNode{
		Kind:  SerialOpRichTextOffset,
		Spans: spans,
		W:     w,
		H:     1,
	})
}

func (t *SerialTemplate) measureLeaderStatic(op *SerialOp) {
	// For static leader, compute text at measure time
	label := op.LeaderLabelStr
	value := op.LeaderValueStr
	width := int(op.LeaderWidth)
	if width == 0 {
		width = utf8.RuneCountInString(label) + utf8.RuneCountInString(value) + 3 // minimum with some dots
	}

	// Compute fill
	fillCount := width - utf8.RuneCountInString(label) - utf8.RuneCountInString(value)
	if fillCount < 1 {
		fillCount = 1
	}
	fill := string(op.LeaderFill)
	fillStr := ""
	for i := 0; i < fillCount; i++ {
		fillStr += fill
	}

	text := label + fillStr + value
	t.nodes = append(t.nodes, SerialNode{
		Kind: op.Kind,
		Text: text,
		W:    int16(utf8.RuneCountInString(text)),
		H:    1,
	})
}

func (t *SerialTemplate) measureLeaderPtr(op *SerialOp) {
	// Get current values from pointers
	label := op.LeaderLabelStr
	if op.LeaderLabelPtr != nil {
		label = *op.LeaderLabelPtr
	}
	value := op.LeaderValueStr
	if op.LeaderValuePtr != nil {
		value = *op.LeaderValuePtr
	}

	width := int(op.LeaderWidth)
	if width == 0 {
		width = utf8.RuneCountInString(label) + utf8.RuneCountInString(value) + 3 // minimum with some dots
	}

	// Compute fill
	fillCount := width - utf8.RuneCountInString(label) - utf8.RuneCountInString(value)
	if fillCount < 1 {
		fillCount = 1
	}
	fill := string(op.LeaderFill)
	fillStr := ""
	for i := 0; i < fillCount; i++ {
		fillStr += fill
	}

	text := label + fillStr + value
	t.nodes = append(t.nodes, SerialNode{
		Kind: op.Kind,
		Text: text,
		W:    int16(utf8.RuneCountInString(text)),
		H:    1,
	})
}

func (t *SerialTemplate) measureCustom(op *SerialOp) {
	var w, h int16 = 1, 1
	if op.CustomMeasure != nil {
		w, h = op.CustomMeasure(0) // TODO: pass available width from context
	}
	t.nodes = append(t.nodes, SerialNode{
		Kind:         op.Kind,
		W:            w,
		H:            h,
		CustomRender: op.CustomRender,
	})
}

func (t *SerialTemplate) measureLayer(op *SerialOp) {
	layer := op.LayerPtr
	h := op.LayerHeight
	w := op.LayerWidth
	// Height 0 means we'll use whatever height is available
	// For now, use the layer's content height or the specified height
	if h == 0 && layer != nil && layer.buffer != nil {
		h = int16(layer.viewHeight) // use viewport height set during layout
		if h == 0 {
			h = int16(layer.buffer.Height()) // fallback to content height
		}
	}
	t.nodes = append(t.nodes, SerialNode{
		Kind:  SerialOpLayer,
		Layer: layer,
		H:     h,
		W:     w, // 0 = fill available width during render
	})
}

func (t *SerialTemplate) measureCondition(op *SerialOp, elemBase unsafe.Pointer) {
	// Evaluate the condition at measure time
	// Use evaluateWithBase for ForEach context (when offset is set)
	condResult := op.CondNode.evaluateWithBase(elemBase)

	if condResult {
		// Render then branch
		if op.CondThen != nil {
			op.CondThen.nodes = op.CondThen.nodes[:0]
			ifSat := false
			for level := 0; level <= op.CondThen.maxLevel; level++ {
				for _, idx := range op.CondThen.byLevel[level] {
					op.CondThen.measureOp(idx, elemBase, &ifSat)
				}
			}
			for _, node := range op.CondThen.nodes {
				t.nodes = append(t.nodes, node)
			}
		}
	} else {
		// Render else branch
		if op.CondElse != nil {
			op.CondElse.nodes = op.CondElse.nodes[:0]
			ifSat := false
			for level := 0; level <= op.CondElse.maxLevel; level++ {
				for _, idx := range op.CondElse.byLevel[level] {
					op.CondElse.measureOp(idx, elemBase, &ifSat)
				}
			}
			for _, node := range op.CondElse.nodes {
				t.nodes = append(t.nodes, node)
			}
		}
	}
}

func (t *SerialTemplate) measureSwitch(op *SerialOp) {
	// Get matching case index at measure time
	matchIdx := op.SwitchNode.getMatchIndex()

	var tmpl *SerialTemplate
	if matchIdx >= 0 && matchIdx < len(op.SwitchCases) {
		tmpl = op.SwitchCases[matchIdx]
	} else {
		tmpl = op.SwitchDef
	}

	if tmpl != nil {
		tmpl.nodes = tmpl.nodes[:0]
		ifSat := false
		for level := 0; level <= tmpl.maxLevel; level++ {
			for _, idx := range tmpl.byLevel[level] {
				tmpl.measureOp(idx, nil, &ifSat)
			}
		}
		for _, node := range tmpl.nodes {
			t.nodes = append(t.nodes, node)
		}
	}
}

// layoutContext tracks position during layout
type layoutContext struct {
	x, y       int16 // current position for next child
	startX     int16 // position when container started
	startY     int16
	isRow      bool
	gap        int8
	maxH       int16 // max child height (for calculating row height)
	maxW       int16 // max child width (for calculating col width)
	firstChild bool  // true if no children added yet (for gap handling)
	availW     int16 // available width for children (for PercentWidth)
	availH     int16 // available height for children (for FlexGrow)
	opIdx      int16 // index of ContainerStart op (for geometry tracking)
}

// Execute measures and renders the template to the buffer.
func (t *SerialTemplate) Execute(buf *Buffer, w, h int16) {
	t.executeThreePhase(buf, w, h, false)
}

// ExecutePadded renders with padded writes, allowing caller to skip Clear().
// Only safe when UI structure is stable (no shrinking content).
func (t *SerialTemplate) ExecutePadded(buf *Buffer, w, h int16) {
	t.executeThreePhase(buf, w, h, true)
}

// executeThreePhase implements proper three-phase layout:
// 1. Update (top→down): distribute widths based on PercentWidth
// 2. Layout (bottom→up): calculate heights, handle FlexGrow
// 3. Draw: render nodes to buffer
func (t *SerialTemplate) executeThreePhase(buf *Buffer, w, h int16, padded bool) {
	// Ensure geometry slice is large enough
	if cap(t.geom) < len(t.ops) {
		t.geom = make([]opGeom, len(t.ops))
	}
	t.geom = t.geom[:len(t.ops)]

	// Initialize root geometry
	// Find first container (if any) or set implicit root
	if len(t.ops) > 0 {
		for i := range t.geom {
			t.geom[i] = opGeom{} // reset
		}
	}

	// Phase 1: Update (top→down) - distribute widths and set sizes
	t.phaseUpdate(w)

	// Phase 2: Layout (bottom→up) - calculate heights
	t.phaseLayout(h)

	// Phase 3: Measure leaf nodes and render
	t.phaseDraw(buf, w, h, padded)
}

// phaseUpdate distributes widths AND sets sizes for ALL ops (top-down)
// This is where measurement happens - most sizes are static or derived from parent
func (t *SerialTemplate) phaseUpdate(rootW int16) {
	// Process ops by level, top to bottom
	for level := 0; level <= t.maxLevel; level++ {
		for _, opIdx := range t.byLevel[level] {
			op := &t.ops[opIdx]

			switch op.Kind {
			case SerialOpContainerStart:
				// Determine this container's width
				var containerW int16
				if op.Width > 0 {
					containerW = op.Width
				} else if op.PercentWidth > 0 {
					parentW := rootW
					if op.Parent >= 0 {
						parentW = t.geom[op.Parent].W
					}
					containerW = int16(float32(parentW) * op.PercentWidth)
				} else {
					parentW := rootW
					if op.Parent >= 0 {
						parentW = t.geom[op.Parent].W
					}
					containerW = parentW
				}

				// Account for border
				borderOffset := int8(0)
				if op.Border.Horizontal != 0 {
					borderOffset = 1
					containerW -= 2
				}

				t.geom[opIdx].W = containerW
				t.geom[opIdx].borderOffset = borderOffset
				// Height set later by phaseLayout (depends on children)

			case SerialOpTextStatic, SerialOpTextPtr, SerialOpTextOffset,
				SerialOpProgressStatic, SerialOpProgressPtr, SerialOpProgressOffset,
				SerialOpRichTextStatic, SerialOpRichTextPtr, SerialOpRichTextOffset,
				SerialOpLeaderStatic, SerialOpLeaderPtr,
				SerialOpIf, SerialOpElse, SerialOpCondition, SerialOpSwitch,
				SerialOpSelectionList:
				// Leaf nodes are 1 line
				t.geom[opIdx].H = 1

			case SerialOpLayer:
				// Layer has explicit or content-based height
				h := op.LayerHeight
				if h == 0 {
					h = 1
				}
				t.geom[opIdx].H = h

			case SerialOpCustom:
				// Custom components measure themselves
				if op.CustomMeasure != nil {
					_, h := op.CustomMeasure(rootW)
					t.geom[opIdx].H = h
				} else {
					t.geom[opIdx].H = 1
				}

			case SerialOpForEach, SerialOpForEachOffset:
				// ForEach: height = slice length × iteration height
				var slicePtr unsafe.Pointer
				if op.Kind == SerialOpForEach {
					slicePtr = op.SlicePtr
				}
				if slicePtr != nil && op.IterTmpl != nil {
					sliceHdr := *(*sliceHeader)(slicePtr)
					iterH := t.calculateIterTemplateHeight(op.IterTmpl)
					t.geom[opIdx].H = int16(sliceHdr.Len) * iterH
				} else {
					t.geom[opIdx].H = 0
				}
			}
		}
	}
}

// phaseLayout calculates container heights bottom-up by summing children
func (t *SerialTemplate) phaseLayout(rootH int16) {
	// Process ops by level, bottom to top
	for level := t.maxLevel; level >= 0; level-- {
		for _, opIdx := range t.byLevel[level] {
			op := &t.ops[opIdx]

			// Only containers need layout - all other ops have heights set in phaseUpdate
			if op.Kind != SerialOpContainerStart {
				continue
			}

			// Calculate content height from children
			contentH := t.calculateContentHeight(opIdx, op)
			t.geom[opIdx].contentH = contentH

			// Determine final height
			var containerH int16
			if op.Height > 0 {
				// Explicit height - this IS the total height (including borders if any)
				containerH = op.Height
			} else if op.FlexGrow > 0 && op.Parent >= 0 {
				// Will be set by parent during flex distribution
				containerH = contentH
				// Account for border
				if op.Border.Horizontal != 0 {
					containerH += 2 // top and bottom border
				}
			} else {
				// Height from content
				containerH = contentH
				// Account for border
				if op.Border.Horizontal != 0 {
					containerH += 2 // top and bottom border
				}
			}

			t.geom[opIdx].H = containerH
		}
	}

	// Second pass: distribute FlexGrow space
	for level := 0; level <= t.maxLevel; level++ {
		for _, opIdx := range t.byLevel[level] {
			op := &t.ops[opIdx]

			if op.Kind != SerialOpContainerStart || op.IsRow {
				continue // Only vertical containers can distribute flex
			}

			t.distributeFlexGrow(opIdx, op, rootH)
		}
	}
}

// calculateContentHeight calculates content height for a container
// Simply sums t.geom[i].H for all children - no type-switching needed
// because phaseUpdate already set sizes for all ops
func (t *SerialTemplate) calculateContentHeight(opIdx int16, op *SerialOp) int16 {
	var totalH int16
	var maxH int16
	childCount := 0

	// Find children (ops with Parent == opIdx)
	for i, childOp := range t.ops {
		if childOp.Parent != opIdx || childOp.Kind == SerialOpContainerEnd {
			continue
		}

		// All ops already have their H set in phaseUpdate
		childH := t.geom[i].H

		childCount++
		if op.IsRow {
			if childH > maxH {
				maxH = childH
			}
		} else {
			totalH += childH
		}
	}

	// Add gaps for vertical layout
	if childCount > 1 && op.Gap > 0 && !op.IsRow {
		totalH += int16(op.Gap) * int16(childCount-1)
	}

	if op.IsRow {
		return maxH
	}
	return totalH
}

// calculateIterTemplateHeight calculates the height of one ForEach iteration
func (t *SerialTemplate) calculateIterTemplateHeight(iterTmpl *SerialTemplate) int16 {
	if iterTmpl == nil || len(iterTmpl.ops) == 0 {
		return 1
	}

	// Simple calculation: count leaf nodes accounting for Row containers
	var totalH int16
	var rowDepth int

	for _, op := range iterTmpl.ops {
		switch op.Kind {
		case SerialOpContainerStart:
			if op.IsRow {
				rowDepth++
			}
		case SerialOpContainerEnd:
			if rowDepth > 0 {
				rowDepth--
				if rowDepth == 0 {
					// Row ended, count as 1 line
					totalH++
				}
			}
		case SerialOpTextStatic, SerialOpTextPtr, SerialOpTextOffset,
			SerialOpProgressStatic, SerialOpProgressPtr, SerialOpProgressOffset,
			SerialOpRichTextStatic, SerialOpRichTextPtr, SerialOpRichTextOffset,
			SerialOpLeaderStatic, SerialOpLeaderPtr, SerialOpCustom:
			if rowDepth == 0 {
				// Not in a row, each element is 1 line
				totalH++
			}
			// Elements inside rows don't add height
		}
	}

	if totalH == 0 {
		return 1
	}
	return totalH
}

// distributeFlexGrow distributes remaining space to flex children
func (t *SerialTemplate) distributeFlexGrow(opIdx int16, op *SerialOp, rootH int16) {
	// Calculate available height
	availH := rootH
	if op.Parent >= 0 {
		parentGeom := t.geom[op.Parent]
		availH = parentGeom.H
		if t.ops[op.Parent].Border.Horizontal != 0 {
			availH -= 2 // Account for parent border
		}
	}

	// Calculate used height and total flex grow
	var usedH int16
	var totalFlex float32
	var flexChildren []int16

	for i, childOp := range t.ops {
		if childOp.Parent != opIdx || childOp.Kind == SerialOpContainerEnd {
			continue
		}

		if childOp.Kind == SerialOpContainerStart {
			if childOp.FlexGrow > 0 {
				totalFlex += childOp.FlexGrow
				flexChildren = append(flexChildren, int16(i))
				usedH += t.geom[i].contentH // Use content height, not final height
			} else {
				usedH += t.geom[i].H
			}
		} else {
			usedH += 1 // Leaf nodes are 1 line
		}
	}

	// Add gaps to used height
	childCount := 0
	for _, childOp := range t.ops {
		if childOp.Parent == opIdx && childOp.Kind != SerialOpContainerEnd {
			childCount++
		}
	}
	if childCount > 1 && op.Gap > 0 {
		usedH += int16(op.Gap) * int16(childCount-1)
	}

	// Distribute remaining space
	remaining := availH - usedH
	if remaining > 0 && totalFlex > 0 {
		for _, childIdx := range flexChildren {
			childOp := &t.ops[childIdx]
			flexShare := childOp.FlexGrow / totalFlex
			extraH := int16(float32(remaining) * flexShare)
			t.geom[childIdx].H = t.geom[childIdx].contentH + extraH
		}
	}
}

// phaseDraw measures leaf nodes, positions everything, and renders
func (t *SerialTemplate) phaseDraw(buf *Buffer, w, h int16, padded bool) {
	t.nodes = t.nodes[:0]

	// Reuse pre-allocated layout context stack
	if cap(t.ctxStack) < 16 {
		t.ctxStack = make([]layoutContext, 0, 16)
	}
	t.ctxStack = t.ctxStack[:1]
	t.ctxStack[0] = layoutContext{x: 0, y: 0, isRow: false, firstChild: true, availW: w, availH: h, opIdx: -1}

	ifSatisfied := false

	// Process ops in document order for positioning
	for i := range t.ops {
		op := &t.ops[i]

		switch op.Kind {
		case SerialOpContainerStart:
			ctx := t.ctxStack[len(t.ctxStack)-1]

			// Draw border if present
			if op.Border.Horizontal != 0 {
				// Calculate container position
				containerX := ctx.x
				containerY := ctx.y
				containerW := t.geom[i].W + 2 // Add back border width
				containerH := t.geom[i].H

				// Draw border
				borderStyle := DefaultStyle()
				if op.BorderFG != nil {
					borderStyle.FG = *op.BorderFG
				}
				buf.DrawBorder(int(containerX), int(containerY), int(containerW), int(containerH), op.Border, borderStyle)

				// Draw title if present
				if op.Title != "" {
					titleStr := "─ " + op.Title + " "
					buf.WriteStringFast(int(containerX)+1, int(containerY), titleStr, borderStyle, int(w))
				}
			}

			// Calculate content area position
			contentX := ctx.x
			contentY := ctx.y
			if op.Border.Horizontal != 0 {
				contentX++
				contentY++
			}

			// Push new context for this container
			t.ctxStack = append(t.ctxStack, layoutContext{
				x:          contentX,
				y:          contentY,
				startX:     contentX,
				startY:     contentY,
				isRow:      op.IsRow,
				gap:        op.Gap,
				firstChild: true,
				availW:     t.geom[i].W,
				availH:     t.geom[i].H,
				opIdx:      int16(i),
			})

		case SerialOpContainerEnd:
			// Pop completed container
			childCtx := t.ctxStack[len(t.ctxStack)-1]
			t.ctxStack = t.ctxStack[:len(t.ctxStack)-1]

			// Get container dimensions from computed geometry
			containerW := t.geom[childCtx.opIdx].W
			containerH := t.geom[childCtx.opIdx].H

			// Account for border in parent positioning
			if childCtx.opIdx >= 0 && t.ops[childCtx.opIdx].Border.Horizontal != 0 {
				containerW += 2
			}

			// Update parent context
			if len(t.ctxStack) > 0 {
				parentIdx := len(t.ctxStack) - 1
				if t.ctxStack[parentIdx].isRow {
					if !t.ctxStack[parentIdx].firstChild && t.ctxStack[parentIdx].gap > 0 {
						t.ctxStack[parentIdx].x += int16(t.ctxStack[parentIdx].gap)
					}
					t.ctxStack[parentIdx].x += containerW
					if containerH > t.ctxStack[parentIdx].maxH {
						t.ctxStack[parentIdx].maxH = containerH
					}
				} else {
					if !t.ctxStack[parentIdx].firstChild && t.ctxStack[parentIdx].gap > 0 {
						t.ctxStack[parentIdx].y += int16(t.ctxStack[parentIdx].gap)
					}
					t.ctxStack[parentIdx].y += containerH
					if containerW > t.ctxStack[parentIdx].maxW {
						t.ctxStack[parentIdx].maxW = containerW
					}
				}
				t.ctxStack[parentIdx].firstChild = false
			}

		case SerialOpForEach, SerialOpForEachOffset:
			// ForEach nodes are already positioned internally - just add offset
			nodeStart := len(t.nodes)
			t.measureOp(int16(i), nil, &ifSatisfied)

			if len(t.ctxStack) > 0 && nodeStart < len(t.nodes) {
				ctxIdx := len(t.ctxStack) - 1

				if !t.ctxStack[ctxIdx].firstChild && t.ctxStack[ctxIdx].gap > 0 {
					if t.ctxStack[ctxIdx].isRow {
						t.ctxStack[ctxIdx].x += int16(t.ctxStack[ctxIdx].gap)
					} else {
						t.ctxStack[ctxIdx].y += int16(t.ctxStack[ctxIdx].gap)
					}
				}

				var maxW, maxH int16
				for j := nodeStart; j < len(t.nodes); j++ {
					node := &t.nodes[j]
					node.X += t.ctxStack[ctxIdx].x
					node.Y += t.ctxStack[ctxIdx].y
					if node.X+node.W > maxW {
						maxW = node.X + node.W - t.ctxStack[ctxIdx].x
					}
					if node.Y+node.H > maxH {
						maxH = node.Y + node.H - t.ctxStack[ctxIdx].y
					}
				}

				if t.ctxStack[ctxIdx].isRow {
					t.ctxStack[ctxIdx].x += maxW
					if maxH > t.ctxStack[ctxIdx].maxH {
						t.ctxStack[ctxIdx].maxH = maxH
					}
				} else {
					t.ctxStack[ctxIdx].y += maxH
					if maxW > t.ctxStack[ctxIdx].maxW {
						t.ctxStack[ctxIdx].maxW = maxW
					}
				}
				t.ctxStack[ctxIdx].firstChild = false
			}

		default:
			// Measure the op and position any new nodes
			nodeStart := len(t.nodes)
			t.measureOp(int16(i), nil, &ifSatisfied)

			// Position new nodes within current container
			if len(t.ctxStack) > 0 {
				ctxIdx := len(t.ctxStack) - 1
				for j := nodeStart; j < len(t.nodes); j++ {
					node := &t.nodes[j]

					if !t.ctxStack[ctxIdx].firstChild && t.ctxStack[ctxIdx].gap > 0 {
						if t.ctxStack[ctxIdx].isRow {
							t.ctxStack[ctxIdx].x += int16(t.ctxStack[ctxIdx].gap)
						} else {
							t.ctxStack[ctxIdx].y += int16(t.ctxStack[ctxIdx].gap)
						}
					}

					node.X = t.ctxStack[ctxIdx].x
					node.Y = t.ctxStack[ctxIdx].y

					if t.ctxStack[ctxIdx].isRow {
						t.ctxStack[ctxIdx].x += node.W
						if node.H > t.ctxStack[ctxIdx].maxH {
							t.ctxStack[ctxIdx].maxH = node.H
						}
					} else {
						t.ctxStack[ctxIdx].y += node.H
						if node.W > t.ctxStack[ctxIdx].maxW {
							t.ctxStack[ctxIdx].maxW = node.W
						}
					}
					t.ctxStack[ctxIdx].firstChild = false
				}
			}
		}
	}

	// Render all nodes
	for i := range t.nodes {
		node := &t.nodes[i]
		switch {
		case node.Kind == SerialOpTextStatic || node.Kind == SerialOpTextPtr || node.Kind == SerialOpTextOffset:
			style := Style{}
			if node.Bold {
				style.Attr = AttrBold
			}
			if padded {
				buf.WriteStringPadded(int(node.X), int(node.Y), node.Text, style, int(w))
			} else {
				buf.WriteStringFast(int(node.X), int(node.Y), node.Text, style, int(w))
			}
		case node.Kind == SerialOpProgressStatic || node.Kind == SerialOpProgressPtr || node.Kind == SerialOpProgressOffset:
			buf.WriteProgressBar(int(node.X), int(node.Y), int(node.Width), node.Ratio, Style{})
		case node.Kind == SerialOpRichTextStatic || node.Kind == SerialOpRichTextPtr || node.Kind == SerialOpRichTextOffset:
			buf.WriteSpans(int(node.X), int(node.Y), node.Spans, int(w))
		case node.Kind == SerialOpLayer:
			if node.Layer != nil {
				layerW := w
				if node.W > 0 {
					layerW = node.W
				}
				node.Layer.setViewport(int(layerW), int(node.H))
				node.Layer.blit(buf, int(node.X), int(node.Y), int(layerW), int(node.H))
			}
		case node.Kind == SerialOpLeaderStatic || node.Kind == SerialOpLeaderPtr:
			style := Style{}
			if padded {
				buf.WriteStringPadded(int(node.X), int(node.Y), node.Text, style, int(w))
			} else {
				buf.WriteStringFast(int(node.X), int(node.Y), node.Text, style, int(w))
			}
		case node.Kind == SerialOpCustom:
			if node.CustomRender != nil {
				node.CustomRender(buf, node.X, node.Y, node.W, node.H)
			}
		}
	}
}

