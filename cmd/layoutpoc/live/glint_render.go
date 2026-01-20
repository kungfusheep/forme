package main

import (
	"reflect"
	"unsafe"

	"tui"
)

// Instruction kinds - const for jump table optimization
const (
	// Static content - value known at compile time
	KindTextStatic uint8 = iota
	KindProgressStatic

	// Pointer bindings - absolute pointer to value
	KindTextPtr
	KindProgressPtr
	KindTextIntPtr
	KindTextFloatPtr

	// Offset bindings - offset from data base pointer (for ForEach elements)
	KindTextStrOff
	KindTextIntOff
	KindTextFloatOff
	KindProgressOff

	// Containers
	KindRowStart
	KindRowEnd
	KindColStart
	KindColEnd

	// Dynamic
	KindForEach
)

// RenderInstruction is a single rendering operation.
// Designed to be small and cache-friendly.
type RenderInstruction struct {
	Kind   uint8
	X, Y   int16
	Width  int16
	Height int16

	// Static content (only one used based on Kind)
	Text       string
	StaticRatio float32

	// Pointer bindings (absolute pointers)
	StrPtr   *string
	IntPtr   *int
	FloatPtr *float32

	// Offset bindings (for ForEach element fields)
	Offset uintptr

	// ForEach specific
	SlicePtr   unsafe.Pointer
	SliceOff   uintptr // offset from parent data (for nested ForEach)
	ElemSize   uintptr
	ElemTmpl   *GlintTemplate // pre-compiled element template
	IsSliceOff bool           // true if SliceOff should be used instead of SlicePtr
}

// GlintTemplate holds pre-compiled render instructions
type GlintTemplate struct {
	instructions []RenderInstruction
	width        int16
	height       int16

	// Fast path for nested ForEach grids - set at compile time
	isNestedGrid   bool
	gridOuterSlice unsafe.Pointer // pointer to outer [][]T
	gridInnerOff   uintptr        // offset to inner slice within outer element
	gridElemOff    uintptr        // offset to value within inner element
	gridElemSize   uintptr        // size of inner element
	gridOuterSize  uintptr        // size of outer element (inner slice)
	gridWidth      int            // progress bar width

	// Viewport culling - set before Render()
	ViewportY      int // first visible row (for scrolling)
	ViewportHeight int // number of visible rows (0 = no culling)
}

// sliceHeader matches runtime slice structure
type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

// Compile builds a GlintTemplate from a declarative UI definition
func Compile(ui any) *GlintTemplate {
	t := &GlintTemplate{
		instructions: make([]RenderInstruction, 0, 32),
	}
	t.compile(ui, 0, 0, nil, 0)
	t.optimizeForGrid()
	return t
}

// optimizeForGrid detects nested ForEach grid patterns and enables fast path
func (t *GlintTemplate) optimizeForGrid() {
	// Pattern: single ForEach instruction where element template is a single ForEach
	// with a single Progress instruction
	if len(t.instructions) != 1 {
		return
	}
	outer := &t.instructions[0]
	if outer.Kind != KindForEach || outer.ElemTmpl == nil {
		return
	}

	// Check if outer element template has single ForEach
	outerElemInsts := outer.ElemTmpl.instructions
	if len(outerElemInsts) != 1 {
		return
	}
	inner := &outerElemInsts[0]
	if inner.Kind != KindForEach || inner.ElemTmpl == nil {
		return
	}

	// Check if inner element template has single Progress
	innerElemInsts := inner.ElemTmpl.instructions
	if len(innerElemInsts) != 1 {
		return
	}
	progress := &innerElemInsts[0]
	if progress.Kind != KindProgressOff {
		return
	}

	// Pattern matched! Enable fast path
	t.isNestedGrid = true
	t.gridOuterSlice = outer.SlicePtr
	t.gridInnerOff = inner.SliceOff
	t.gridElemOff = progress.Offset
	t.gridElemSize = inner.ElemSize
	t.gridOuterSize = outer.ElemSize
	t.gridWidth = int(progress.Width)
}

// compile recursively processes UI nodes, tracking position
func (t *GlintTemplate) compile(node any, x, y int16, elemBase unsafe.Pointer, elemSize uintptr) (w, h int16) {
	switch v := node.(type) {
	case tui.Text:
		return t.compileText(v, x, y, elemBase, elemSize)
	case tui.Progress:
		return t.compileProgress(v, x, y, elemBase, elemSize)
	case tui.HBox:
		return t.compileHBox(v, x, y, elemBase, elemSize)
	case tui.VBox:
		return t.compileVBox(v, x, y, elemBase, elemSize)
	case tui.ForEachNode:
		return t.compileForEach(v, x, y, elemBase, elemSize)
	}
	return 0, 0
}

func (t *GlintTemplate) compileText(v tui.Text, x, y int16, elemBase unsafe.Pointer, elemSize uintptr) (w, h int16) {
	inst := RenderInstruction{
		X:      x,
		Y:      y,
		Height: 1,
	}

	switch content := v.Content.(type) {
	case string:
		inst.Kind = KindTextStatic
		inst.Text = content
		inst.Width = int16(len(content))

	case *string:
		ptr := unsafe.Pointer(content)
		if elemBase != nil && isWithinRange(ptr, elemBase, elemSize) {
			inst.Kind = KindTextStrOff
			inst.Offset = uintptr(ptr) - uintptr(elemBase)
			inst.Width = 20 // estimate for dynamic text
		} else {
			inst.Kind = KindTextPtr
			inst.StrPtr = content
			inst.Width = 20
		}

	case *int:
		ptr := unsafe.Pointer(content)
		if elemBase != nil && isWithinRange(ptr, elemBase, elemSize) {
			inst.Kind = KindTextIntOff
			inst.Offset = uintptr(ptr) - uintptr(elemBase)
		} else {
			inst.Kind = KindTextIntPtr
			inst.IntPtr = content
		}
		inst.Width = 10

	case *float32:
		ptr := unsafe.Pointer(content)
		if elemBase != nil && isWithinRange(ptr, elemBase, elemSize) {
			inst.Kind = KindTextFloatOff
			inst.Offset = uintptr(ptr) - uintptr(elemBase)
		} else {
			inst.Kind = KindTextFloatPtr
			inst.FloatPtr = content
		}
		inst.Width = 8
	}

	t.instructions = append(t.instructions, inst)
	return inst.Width, 1
}

func (t *GlintTemplate) compileProgress(v tui.Progress, x, y int16, elemBase unsafe.Pointer, elemSize uintptr) (w, h int16) {
	width := v.BarWidth
	if width == 0 {
		width = 20
	}

	inst := RenderInstruction{
		X:      x,
		Y:      y,
		Width:  width,
		Height: 1,
	}

	switch val := v.Value.(type) {
	case float32:
		inst.Kind = KindProgressStatic
		inst.StaticRatio = val

	case *float32:
		ptr := unsafe.Pointer(val)
		if elemBase != nil && isWithinRange(ptr, elemBase, elemSize) {
			inst.Kind = KindProgressOff
			inst.Offset = uintptr(ptr) - uintptr(elemBase)
		} else {
			inst.Kind = KindProgressPtr
			inst.FloatPtr = val
		}

	case *int:
		// Convert int pointer to progress (0-100 range)
		ptr := unsafe.Pointer(val)
		if elemBase != nil && isWithinRange(ptr, elemBase, elemSize) {
			inst.Kind = KindProgressOff
			inst.Offset = uintptr(ptr) - uintptr(elemBase)
		} else {
			inst.Kind = KindProgressPtr
			// Store as float pointer (will need conversion at render time)
			inst.IntPtr = val
		}
	}

	t.instructions = append(t.instructions, inst)
	return width, 1
}

func (t *GlintTemplate) compileHBox(v tui.HBox, x, y int16, elemBase unsafe.Pointer, elemSize uintptr) (w, h int16) {
	currentX := x
	maxH := int16(0)

	for _, child := range v.Children {
		childW, childH := t.compile(child, currentX, y, elemBase, elemSize)
		currentX += childW
		if v.Gap > 0 {
			currentX += int16(v.Gap)
		}
		if childH > maxH {
			maxH = childH
		}
	}

	return currentX - x, maxH
}

func (t *GlintTemplate) compileVBox(v tui.VBox, x, y int16, elemBase unsafe.Pointer, elemSize uintptr) (w, h int16) {
	currentY := y
	maxW := int16(0)

	for _, child := range v.Children {
		childW, childH := t.compile(child, x, currentY, elemBase, elemSize)
		currentY += childH
		if v.Gap > 0 {
			currentY += int16(v.Gap)
		}
		if childW > maxW {
			maxW = childW
		}
	}

	return maxW, currentY - y
}

func (t *GlintTemplate) compileForEach(v tui.ForEachNode, x, y int16, parentElemBase unsafe.Pointer, parentElemSize uintptr) (w, h int16) {
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

	// Compile element template
	elemTmpl := &GlintTemplate{
		instructions: make([]RenderInstruction, 0, 16),
	}
	elemW, elemH := elemTmpl.compile(templateResult, 0, 0, dummyBase, elemSize)
	elemTmpl.width = elemW
	elemTmpl.height = elemH

	// Create ForEach instruction
	inst := RenderInstruction{
		Kind:     KindForEach,
		X:        x,
		Y:        y,
		Width:    elemW,
		Height:   elemH,
		ElemSize: elemSize,
		ElemTmpl: elemTmpl,
	}

	// Check if slice pointer is within parent element (nested ForEach)
	if parentElemBase != nil && isWithinRange(slicePtr, parentElemBase, parentElemSize) {
		inst.IsSliceOff = true
		inst.SliceOff = uintptr(slicePtr) - uintptr(parentElemBase)
	} else {
		inst.SlicePtr = slicePtr
	}

	t.instructions = append(t.instructions, inst)

	// Return estimated size (1 row for now, actual size determined at runtime)
	return elemW, elemH
}

// isWithinRange checks if ptr is within [base, base+size)
func isWithinRange(ptr, base unsafe.Pointer, size uintptr) bool {
	p := uintptr(ptr)
	b := uintptr(base)
	return p >= b && p < b+size
}

// Render executes the pre-compiled template, writing directly to the buffer
func (t *GlintTemplate) Render(buf *tui.Buffer, dataPtr unsafe.Pointer) {
	// Fast path for nested progress grids - avoids 4 function calls
	if t.isNestedGrid {
		t.renderGridFast(buf)
		return
	}
	t.renderInstructions(buf, t.instructions, 0, 0, dataPtr)
}

// renderGridFast is the ultra-fast path for 2D grids of progress bars
// All parameters are pre-computed at compile time, this is just two nested loops
// Supports viewport culling when ViewportHeight > 0
func (t *GlintTemplate) renderGridFast(buf *tui.Buffer) {
	outerHdr := *(*sliceHeader)(t.gridOuterSlice)
	width := t.gridWidth

	// Viewport culling
	startRow := 0
	endRow := outerHdr.Len
	if t.ViewportHeight > 0 {
		startRow = t.ViewportY
		endRow = t.ViewportY + t.ViewportHeight
		if startRow < 0 {
			startRow = 0
		}
		if endRow > outerHdr.Len {
			endRow = outerHdr.Len
		}
	}

	y := 0
	for i := startRow; i < endRow; i++ {
		outerElemPtr := unsafe.Add(outerHdr.Data, uintptr(i)*t.gridOuterSize)
		innerSlicePtr := unsafe.Add(outerElemPtr, t.gridInnerOff)
		innerHdr := *(*sliceHeader)(innerSlicePtr)

		x := 0
		for j := 0; j < innerHdr.Len; j++ {
			innerElemPtr := unsafe.Add(innerHdr.Data, uintptr(j)*t.gridElemSize)
			ratio := *(*float32)(unsafe.Add(innerElemPtr, t.gridElemOff))
			buf.WriteProgressBar(x, y, width, ratio, tui.Style{})
			x += width
		}
		y++
	}
}

// renderInstructions executes a slice of instructions with position offset
func (t *GlintTemplate) renderInstructions(buf *tui.Buffer, instructions []RenderInstruction, offsetX, offsetY int16, dataPtr unsafe.Pointer) {
	for i := range instructions {
		inst := &instructions[i]
		x := inst.X + offsetX
		y := inst.Y + offsetY

		switch inst.Kind {
		case KindTextStatic:
			buf.WriteStringFast(int(x), int(y), inst.Text, tui.Style{}, int(inst.Width))

		case KindTextPtr:
			if inst.StrPtr != nil {
				buf.WriteStringFast(int(x), int(y), *inst.StrPtr, tui.Style{}, int(inst.Width))
			}

		case KindTextStrOff:
			text := *(*string)(unsafe.Add(dataPtr, inst.Offset))
			buf.WriteStringFast(int(x), int(y), text, tui.Style{}, int(inst.Width))

		case KindTextIntPtr:
			if inst.IntPtr != nil {
				writeInt(buf, int(x), int(y), *inst.IntPtr, int(inst.Width))
			}

		case KindTextIntOff:
			val := *(*int)(unsafe.Add(dataPtr, inst.Offset))
			writeInt(buf, int(x), int(y), val, int(inst.Width))

		case KindTextFloatPtr:
			if inst.FloatPtr != nil {
				writeFloat(buf, int(x), int(y), *inst.FloatPtr, int(inst.Width))
			}

		case KindTextFloatOff:
			val := *(*float32)(unsafe.Add(dataPtr, inst.Offset))
			writeFloat(buf, int(x), int(y), val, int(inst.Width))

		case KindProgressStatic:
			buf.WriteProgressBar(int(x), int(y), int(inst.Width), inst.StaticRatio, tui.Style{})

		case KindProgressPtr:
			if inst.FloatPtr != nil {
				buf.WriteProgressBar(int(x), int(y), int(inst.Width), *inst.FloatPtr, tui.Style{})
			}

		case KindProgressOff:
			ratio := *(*float32)(unsafe.Add(dataPtr, inst.Offset))
			buf.WriteProgressBar(int(x), int(y), int(inst.Width), ratio, tui.Style{})

		case KindForEach:
			t.renderForEach(buf, inst, x, y, dataPtr)
		}
	}
}

// renderForEach handles dynamic slice iteration
func (t *GlintTemplate) renderForEach(buf *tui.Buffer, inst *RenderInstruction, x, y int16, dataPtr unsafe.Pointer) {
	// Get slice pointer
	var slicePtr unsafe.Pointer
	if inst.IsSliceOff {
		slicePtr = unsafe.Add(dataPtr, inst.SliceOff)
	} else {
		slicePtr = inst.SlicePtr
	}

	// Read slice header
	hdr := *(*sliceHeader)(slicePtr)
	if hdr.Len == 0 || inst.ElemTmpl == nil {
		return
	}

	elemInsts := inst.ElemTmpl.instructions

	// Apply viewport culling
	startRow := 0
	endRow := hdr.Len
	if t.ViewportHeight > 0 {
		startRow = t.ViewportY
		endRow = t.ViewportY + t.ViewportHeight
		if startRow < 0 {
			startRow = 0
		}
		if endRow > hdr.Len {
			endRow = hdr.Len
		}
	}

	// Fast path: single-instruction element template (very common)
	if len(elemInsts) == 1 {
		single := &elemInsts[0]
		if single.Kind == KindForEach {
			// Nested ForEach - handle inline to avoid extra renderInstructions call
			t.renderNestedForEachCulled(buf, inst, single, hdr, x, y, startRow, endRow)
			return
		}
		t.renderForEachSingleCulled(buf, inst, single, hdr, x, y, startRow, endRow)
		return
	}

	// General path: multi-instruction element template
	currentY := y
	for i := startRow; i < endRow; i++ {
		elemPtr := unsafe.Add(hdr.Data, uintptr(i)*inst.ElemSize)
		t.renderInstructions(buf, elemInsts, x, currentY, elemPtr)
		currentY += inst.Height
	}
}

// renderNestedForEach handles nested ForEach pattern inline (ForEach within ForEach)
// This is the hot path for grids - fully inlined for maximum performance
func (t *GlintTemplate) renderNestedForEach(buf *tui.Buffer, outerInst *RenderInstruction, innerForEach *RenderInstruction, outerHdr sliceHeader, baseX, baseY int16) {
	t.renderNestedForEachCulled(buf, outerInst, innerForEach, outerHdr, baseX, baseY, 0, outerHdr.Len)
}

// renderNestedForEachCulled handles nested ForEach with viewport culling
func (t *GlintTemplate) renderNestedForEachCulled(buf *tui.Buffer, outerInst *RenderInstruction, innerForEach *RenderInstruction, outerHdr sliceHeader, baseX, baseY int16, startRow, endRow int) {
	outerElemSize := outerInst.ElemSize
	outerHeight := outerInst.Height
	innerElemSize := innerForEach.ElemSize
	innerHeight := innerForEach.Height

	// Check if inner ForEach has single-instruction element (common case: grid of progress bars)
	innerElemInsts := innerForEach.ElemTmpl.instructions
	if len(innerElemInsts) == 1 && innerElemInsts[0].Kind != KindForEach {
		elemInst := &innerElemInsts[0]

		// Dispatch once based on kind, then run specialized loop
		switch elemInst.Kind {
		case KindProgressOff:
			t.renderNestedProgressOffCulled(buf, outerInst, innerForEach, outerHdr, baseX, baseY, elemInst,
				outerElemSize, outerHeight, innerElemSize, innerHeight, startRow, endRow)
		default:
			// Fallback for other types
			t.renderNestedGenericCulled(buf, outerInst, innerForEach, outerHdr, baseX, baseY, elemInst,
				outerElemSize, outerHeight, innerElemSize, innerHeight, startRow, endRow)
		}
		return
	}

	// General path for complex inner elements
	t.renderNestedComplexCulled(buf, outerInst, innerForEach, outerHdr, baseX, baseY,
		outerElemSize, outerHeight, innerElemSize, innerHeight, innerElemInsts, startRow, endRow)
}

// renderNestedProgressOff is the specialized hot path for grids of progress bars
func (t *GlintTemplate) renderNestedProgressOff(buf *tui.Buffer, outerInst, innerForEach *RenderInstruction, outerHdr sliceHeader, baseX, baseY int16, elemInst *RenderInstruction, outerElemSize uintptr, outerHeight int16, innerElemSize uintptr, innerHeight int16) {
	t.renderNestedProgressOffCulled(buf, outerInst, innerForEach, outerHdr, baseX, baseY, elemInst, outerElemSize, outerHeight, innerElemSize, innerHeight, 0, outerHdr.Len)
}

// renderNestedProgressOffCulled is the specialized hot path with viewport culling
func (t *GlintTemplate) renderNestedProgressOffCulled(buf *tui.Buffer, outerInst, innerForEach *RenderInstruction, outerHdr sliceHeader, baseX, baseY int16, elemInst *RenderInstruction, outerElemSize uintptr, outerHeight int16, innerElemSize uintptr, innerHeight int16, startRow, endRow int) {
	offset := elemInst.Offset
	width := int(elemInst.Width)
	sliceOff := innerForEach.SliceOff
	isSliceOff := innerForEach.IsSliceOff
	innerForEachX := innerForEach.X
	innerForEachY := innerForEach.Y

	currentY := baseY
	for i := startRow; i < endRow; i++ {
		outerElemPtr := unsafe.Add(outerHdr.Data, uintptr(i)*outerElemSize)

		// Resolve inner slice
		var innerSlicePtr unsafe.Pointer
		if isSliceOff {
			innerSlicePtr = unsafe.Add(outerElemPtr, sliceOff)
		} else {
			innerSlicePtr = innerForEach.SlicePtr
		}

		innerHdr := *(*sliceHeader)(innerSlicePtr)
		if innerHdr.Len == 0 {
			currentY += outerHeight
			continue
		}

		x := int(baseX + innerForEachX)
		y := int(currentY + innerForEachY)

		// Tight inner loop - no switch, no extra calculations
		for j := 0; j < innerHdr.Len; j++ {
			innerElemPtr := unsafe.Add(innerHdr.Data, uintptr(j)*innerElemSize)
			ratio := *(*float32)(unsafe.Add(innerElemPtr, offset))
			buf.WriteProgressBar(x, y, width, ratio, tui.Style{})
			y += int(innerHeight)
		}

		currentY += outerHeight
	}
}

// renderNestedGeneric handles other single-instruction element types
func (t *GlintTemplate) renderNestedGeneric(buf *tui.Buffer, outerInst, innerForEach *RenderInstruction, outerHdr sliceHeader, baseX, baseY int16, elemInst *RenderInstruction, outerElemSize uintptr, outerHeight int16, innerElemSize uintptr, innerHeight int16) {
	t.renderNestedGenericCulled(buf, outerInst, innerForEach, outerHdr, baseX, baseY, elemInst, outerElemSize, outerHeight, innerElemSize, innerHeight, 0, outerHdr.Len)
}

// renderNestedGenericCulled handles other single-instruction element types with viewport culling
func (t *GlintTemplate) renderNestedGenericCulled(buf *tui.Buffer, outerInst, innerForEach *RenderInstruction, outerHdr sliceHeader, baseX, baseY int16, elemInst *RenderInstruction, outerElemSize uintptr, outerHeight int16, innerElemSize uintptr, innerHeight int16, startRow, endRow int) {
	kind := elemInst.Kind
	offset := elemInst.Offset
	width := int(elemInst.Width)
	sliceOff := innerForEach.SliceOff
	isSliceOff := innerForEach.IsSliceOff

	currentY := baseY
	for i := startRow; i < endRow; i++ {
		outerElemPtr := unsafe.Add(outerHdr.Data, uintptr(i)*outerElemSize)

		var innerSlicePtr unsafe.Pointer
		if isSliceOff {
			innerSlicePtr = unsafe.Add(outerElemPtr, sliceOff)
		} else {
			innerSlicePtr = innerForEach.SlicePtr
		}

		innerHdr := *(*sliceHeader)(innerSlicePtr)
		if innerHdr.Len == 0 {
			currentY += outerHeight
			continue
		}

		innerX := baseX + innerForEach.X
		innerY := currentY + innerForEach.Y

		innerCurrentY := innerY
		for j := 0; j < innerHdr.Len; j++ {
			innerElemPtr := unsafe.Add(innerHdr.Data, uintptr(j)*innerElemSize)
			x := int(innerX)
			y := int(innerCurrentY)

			switch kind {
			case KindTextStrOff:
				text := *(*string)(unsafe.Add(innerElemPtr, offset))
				buf.WriteStringFast(x, y, text, tui.Style{}, width)

			case KindTextIntOff:
				val := *(*int)(unsafe.Add(innerElemPtr, offset))
				writeInt(buf, x, y, val, width)

			case KindTextFloatOff:
				val := *(*float32)(unsafe.Add(innerElemPtr, offset))
				writeFloat(buf, x, y, val, width)

			case KindProgressStatic:
				buf.WriteProgressBar(x, y, width, elemInst.StaticRatio, tui.Style{})

			case KindTextStatic:
				buf.WriteStringFast(x, y, elemInst.Text, tui.Style{}, width)
			}

			innerCurrentY += innerHeight
		}

		currentY += outerHeight
	}
}

// renderNestedComplex handles multi-instruction inner elements
func (t *GlintTemplate) renderNestedComplex(buf *tui.Buffer, outerInst, innerForEach *RenderInstruction, outerHdr sliceHeader, baseX, baseY int16, outerElemSize uintptr, outerHeight int16, innerElemSize uintptr, innerHeight int16, innerElemInsts []RenderInstruction) {
	t.renderNestedComplexCulled(buf, outerInst, innerForEach, outerHdr, baseX, baseY, outerElemSize, outerHeight, innerElemSize, innerHeight, innerElemInsts, 0, outerHdr.Len)
}

// renderNestedComplexCulled handles multi-instruction inner elements with viewport culling
func (t *GlintTemplate) renderNestedComplexCulled(buf *tui.Buffer, outerInst, innerForEach *RenderInstruction, outerHdr sliceHeader, baseX, baseY int16, outerElemSize uintptr, outerHeight int16, innerElemSize uintptr, innerHeight int16, innerElemInsts []RenderInstruction, startRow, endRow int) {
	currentY := baseY
	for i := startRow; i < endRow; i++ {
		outerElemPtr := unsafe.Add(outerHdr.Data, uintptr(i)*outerElemSize)

		innerX := baseX + innerForEach.X
		innerY := currentY + innerForEach.Y

		var innerSlicePtr unsafe.Pointer
		if innerForEach.IsSliceOff {
			innerSlicePtr = unsafe.Add(outerElemPtr, innerForEach.SliceOff)
		} else {
			innerSlicePtr = innerForEach.SlicePtr
		}

		innerHdr := *(*sliceHeader)(innerSlicePtr)
		if innerHdr.Len > 0 {
			innerCurrentY := innerY
			for j := 0; j < innerHdr.Len; j++ {
				innerElemPtr := unsafe.Add(innerHdr.Data, uintptr(j)*innerElemSize)
				t.renderInstructions(buf, innerElemInsts, innerX, innerCurrentY, innerElemPtr)
				innerCurrentY += innerHeight
			}
		}

		currentY += outerHeight
	}
}

// renderForEachSingle is an optimized path for single-instruction element templates
func (t *GlintTemplate) renderForEachSingle(buf *tui.Buffer, inst *RenderInstruction, elemInst *RenderInstruction, hdr sliceHeader, baseX, baseY int16) {
	t.renderForEachSingleCulled(buf, inst, elemInst, hdr, baseX, baseY, 0, hdr.Len)
}

// renderForEachSingleCulled is an optimized path with viewport culling
func (t *GlintTemplate) renderForEachSingleCulled(buf *tui.Buffer, inst *RenderInstruction, elemInst *RenderInstruction, hdr sliceHeader, baseX, baseY int16, startRow, endRow int) {
	elemSize := inst.ElemSize
	height := inst.Height
	kind := elemInst.Kind

	// Pre-compute fixed values
	elemX := baseX + elemInst.X
	width := int(elemInst.Width)
	offset := elemInst.Offset

	currentY := baseY
	for i := startRow; i < endRow; i++ {
		elemPtr := unsafe.Add(hdr.Data, uintptr(i)*elemSize)
		y := int(currentY + elemInst.Y)

		switch kind {
		case KindProgressOff:
			ratio := *(*float32)(unsafe.Add(elemPtr, offset))
			buf.WriteProgressBar(int(elemX), y, width, ratio, tui.Style{})

		case KindTextStrOff:
			text := *(*string)(unsafe.Add(elemPtr, offset))
			buf.WriteStringFast(int(elemX), y, text, tui.Style{}, width)

		case KindTextIntOff:
			val := *(*int)(unsafe.Add(elemPtr, offset))
			writeInt(buf, int(elemX), y, val, width)

		case KindTextFloatOff:
			val := *(*float32)(unsafe.Add(elemPtr, offset))
			writeFloat(buf, int(elemX), y, val, width)

		case KindProgressStatic:
			buf.WriteProgressBar(int(elemX), y, width, elemInst.StaticRatio, tui.Style{})

		case KindTextStatic:
			buf.WriteStringFast(int(elemX), y, elemInst.Text, tui.Style{}, width)

		// Note: KindForEach is excluded at the call site, so we don't need to handle it here
		}

		currentY += height
	}
}

// writeInt writes an integer to the buffer (simple implementation)
func writeInt(buf *tui.Buffer, x, y, val, width int) {
	// Simple int to string conversion
	s := intToString(val)
	buf.WriteStringFast(x, y, s, tui.Style{}, width)
}

// writeFloat writes a float to the buffer
func writeFloat(buf *tui.Buffer, x, y int, val float32, width int) {
	s := floatToString(val)
	buf.WriteStringFast(x, y, s, tui.Style{}, width)
}

// intToString converts int to string without allocation (for small numbers)
var intStrings = [...]string{
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
	"10", "11", "12", "13", "14", "15", "16", "17", "18", "19",
	"20", "21", "22", "23", "24", "25", "26", "27", "28", "29",
	"30", "31", "32", "33", "34", "35", "36", "37", "38", "39",
	"40", "41", "42", "43", "44", "45", "46", "47", "48", "49",
	"50", "51", "52", "53", "54", "55", "56", "57", "58", "59",
	"60", "61", "62", "63", "64", "65", "66", "67", "68", "69",
	"70", "71", "72", "73", "74", "75", "76", "77", "78", "79",
	"80", "81", "82", "83", "84", "85", "86", "87", "88", "89",
	"90", "91", "92", "93", "94", "95", "96", "97", "98", "99",
	"100",
}

func intToString(v int) string {
	if v >= 0 && v <= 100 {
		return intStrings[v]
	}
	// Fallback for larger numbers - this will allocate
	return formatInt(v)
}

func formatInt(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// floatToString converts float32 to string (simple implementation)
func floatToString(v float32) string {
	// Convert to percentage-style display (0.XX -> XX%)
	pct := int(v * 100)
	if pct >= 0 && pct <= 100 {
		return intStrings[pct]
	}
	return formatInt(pct)
}
