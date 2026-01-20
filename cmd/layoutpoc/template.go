package main

import (
	"reflect"
	"unsafe"

	"tui"
)

// Instruction kinds - const for switch jump table optimization
const (
	KindText uint8 = iota
	KindProgress
	KindForEach
	KindRowStart
	KindRowEnd
	KindColStart
	KindColEnd
	KindCustom
)

// Binding kinds for offset-based data access
const (
	BindNone uint8 = iota
	BindString
	BindInt
	BindFloat32
)

// Inst is a single UI operation - compiled once, executed every frame.
type Inst struct {
	Kind uint8

	// Position - computed during Layout phase
	X, Y int16

	// Computed size - set during Layout phase
	Width  int16
	Height int16

	// Data binding (one of these used based on Kind + BindKind)
	StrPtr   *string
	IntPtr   *int
	FloatPtr *float32
	Offset   uintptr // Offset from ForEach element base
	BindKind uint8

	// Static content
	Text     string
	BarWidth int16

	// Container layout
	Gap int16

	// ForEach specific
	SlicePtr   unsafe.Pointer
	ElemSize   uintptr
	ElemCount  int   // Number of element instructions following this one
	ElemWidth  int16 // Computed element width
	ElemHeight int16 // Computed element height
	Horizontal bool  // True if inside a Row (horizontal layout)

	// Custom component callbacks
	Measure func(availW int16) (w, h int16)
	Render  func(buf *tui.Buffer, x, y, w, h int16)
}

// Template holds compiled instructions.
type Template struct {
	instructions []Inst

	// Viewport for culling
	ViewportY      int
	ViewportHeight int
}

// sliceHeader for unsafe slice access
type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

// Compile builds a Template from declarative UI.
// Uses reflect once to analyze structure and capture data pointers.
func Compile(ui any) *Template {
	t := &Template{
		instructions: make([]Inst, 0, 32),
	}
	t.compile(ui, nil, 0, false) // Start with vertical (Col) context
	return t
}

// compile recursively processes UI nodes.
// horizontal indicates if we're inside a Row (affects ForEach layout direction)
func (t *Template) compile(node any, elemBase unsafe.Pointer, elemSize uintptr, horizontal bool) {
	switch v := node.(type) {
	case tui.Text:
		t.compileText(v, elemBase, elemSize)
	case tui.Progress:
		t.compileProgress(v, elemBase, elemSize)
	case tui.HBox:
		t.compileHBox(v, elemBase, elemSize)
	case tui.VBox:
		t.compileVBox(v, elemBase, elemSize)
	case tui.ForEachNode:
		t.compileForEach(v, elemBase, elemSize, horizontal)
	case tui.Custom:
		t.compileCustom(v)
	}
}

func (t *Template) compileText(v tui.Text, elemBase unsafe.Pointer, elemSize uintptr) {
	inst := Inst{Kind: KindText}

	switch content := v.Content.(type) {
	case string:
		inst.Text = content
	case *string:
		if elemBase != nil && inRange(unsafe.Pointer(content), elemBase, elemSize) {
			inst.Offset = uintptr(unsafe.Pointer(content)) - uintptr(elemBase)
			inst.BindKind = BindString
		} else {
			inst.StrPtr = content
		}
	case *int:
		if elemBase != nil && inRange(unsafe.Pointer(content), elemBase, elemSize) {
			inst.Offset = uintptr(unsafe.Pointer(content)) - uintptr(elemBase)
			inst.BindKind = BindInt
		} else {
			inst.IntPtr = content
		}
	case *float32:
		if elemBase != nil && inRange(unsafe.Pointer(content), elemBase, elemSize) {
			inst.Offset = uintptr(unsafe.Pointer(content)) - uintptr(elemBase)
			inst.BindKind = BindFloat32
		} else {
			inst.FloatPtr = content
		}
	}

	t.instructions = append(t.instructions, inst)
}

func (t *Template) compileProgress(v tui.Progress, elemBase unsafe.Pointer, elemSize uintptr) {
	inst := Inst{
		Kind:     KindProgress,
		BarWidth: v.BarWidth,
	}
	if inst.BarWidth == 0 {
		inst.BarWidth = 20
	}

	switch val := v.Value.(type) {
	case *float32:
		if elemBase != nil && inRange(unsafe.Pointer(val), elemBase, elemSize) {
			inst.Offset = uintptr(unsafe.Pointer(val)) - uintptr(elemBase)
			inst.BindKind = BindFloat32
		} else {
			inst.FloatPtr = val
		}
	}

	t.instructions = append(t.instructions, inst)
}

func (t *Template) compileHBox(v tui.HBox, elemBase unsafe.Pointer, elemSize uintptr) {
	startIdx := len(t.instructions)
	t.instructions = append(t.instructions, Inst{
		Kind: KindRowStart,
		Gap:  int16(v.Gap),
	})

	for _, child := range v.Children {
		t.compile(child, elemBase, elemSize, true) // Row children are horizontal
	}

	t.instructions = append(t.instructions, Inst{Kind: KindRowEnd})

	// Store count of instructions between start and end
	t.instructions[startIdx].ElemCount = len(t.instructions) - startIdx - 2
}

func (t *Template) compileVBox(v tui.VBox, elemBase unsafe.Pointer, elemSize uintptr) {
	startIdx := len(t.instructions)
	t.instructions = append(t.instructions, Inst{
		Kind: KindColStart,
		Gap:  int16(v.Gap),
	})

	for _, child := range v.Children {
		t.compile(child, elemBase, elemSize, false) // Col children are vertical
	}

	t.instructions = append(t.instructions, Inst{Kind: KindColEnd})

	// Store count of instructions between start and end
	t.instructions[startIdx].ElemCount = len(t.instructions) - startIdx - 2
}

func (t *Template) compileForEach(v tui.ForEachNode, elemBase unsafe.Pointer, elemSize uintptr, horizontal bool) {
	// Analyze slice type using reflect (only at compile time)
	sliceRV := reflect.ValueOf(v.Items)
	if sliceRV.Kind() != reflect.Ptr {
		panic("ForEach Items must be pointer to slice")
	}
	sliceType := sliceRV.Type().Elem()
	if sliceType.Kind() != reflect.Slice {
		panic("ForEach Items must be pointer to slice")
	}
	elemType := sliceType.Elem()
	elemSz := elemType.Size()
	slicePtr := unsafe.Pointer(sliceRV.Pointer())

	// Create dummy element to compile element template
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

	// Get template UI from render function (reflect.Call - only at compile time)
	templateUI := renderRV.Call([]reflect.Value{dummyElem})[0].Interface()

	// Record ForEach instruction
	forEachIdx := len(t.instructions)
	t.instructions = append(t.instructions, Inst{
		Kind:       KindForEach,
		SlicePtr:   slicePtr,
		ElemSize:   elemSz,
		Horizontal: horizontal, // Store layout direction
	})

	// Compile element template inline (element children inherit vertical context)
	elemStartIdx := len(t.instructions)
	t.compile(templateUI, dummyBase, elemSz, false)
	elemInstCount := len(t.instructions) - elemStartIdx

	// Update ForEach with element instruction count
	t.instructions[forEachIdx].ElemCount = elemInstCount
}

func (t *Template) compileCustom(v tui.Custom) {
	t.instructions = append(t.instructions, Inst{
		Kind:    KindCustom,
		Measure: v.Measure,
		Render:  v.Render,
	})
}

func inRange(ptr, base unsafe.Pointer, size uintptr) bool {
	p := uintptr(ptr)
	b := uintptr(base)
	return p >= b && p < b+size
}

// Execute runs the three-phase render cycle: Update, Layout, Render.
func (t *Template) Execute(buf *tui.Buffer, width, height int) {
	if len(t.instructions) == 0 {
		return
	}

	// Phase 1: Update - read current data (slice lengths, etc.)
	t.phaseUpdate(0, len(t.instructions))

	// Phase 2: Layout - compute positions bottom-up
	t.phaseLayout(0, len(t.instructions), 0, 0)

	// Phase 3: Render - draw to buffer
	t.phaseRender(buf, 0, len(t.instructions), nil, 0, 0)
}

// phaseUpdate reads current data values (slice lengths).
// This is the "top-down constraints" phase.
func (t *Template) phaseUpdate(start, end int) {
	for i := start; i < end; {
		inst := &t.instructions[i]

		switch inst.Kind {
		case KindForEach:
			// Read current slice length
			hdr := *(*sliceHeader)(inst.SlicePtr)
			_ = hdr.Len // Used in layout

			// Update element template
			elemEnd := i + 1 + inst.ElemCount
			t.phaseUpdate(i+1, elemEnd)
			i = elemEnd

		case KindRowStart, KindColStart:
			// Skip to matching end
			i = t.skipContainer(i) + 1

		default:
			i++
		}
	}
}

// phaseLayout computes sizes and positions.
// Returns (width, height) of laid out content.
func (t *Template) phaseLayout(start, end int, x, y int16) (w, h int16) {
	for i := start; i < end; {
		inst := &t.instructions[i]

		switch inst.Kind {
		case KindText:
			inst.X = x
			inst.Y = y
			// Compute width from content
			if inst.Text != "" {
				inst.Width = int16(len(inst.Text))
			} else if inst.StrPtr != nil {
				inst.Width = int16(len(*inst.StrPtr))
			} else {
				inst.Width = 10 // Default for int/float
			}
			inst.Height = 1

			if inst.Width > w {
				w = inst.Width
			}
			y++
			h++
			i++

		case KindProgress:
			inst.X = x
			inst.Y = y
			inst.Height = 1
			if inst.Width == 0 {
				inst.Width = inst.BarWidth
			}

			if inst.Width > w {
				w = inst.Width
			}
			y++
			h++
			i++

		case KindForEach:
			inst.X = x
			inst.Y = y

			// Layout element template to get dimensions
			elemEnd := i + 1 + inst.ElemCount
			elemW, elemH := t.phaseLayout(i+1, elemEnd, 0, 0)
			inst.ElemWidth = elemW
			inst.ElemHeight = elemH

			// Get slice length
			hdr := *(*sliceHeader)(inst.SlicePtr)

			if inst.Horizontal {
				// Horizontal layout: width = sum of elements, height = element height
				inst.Width = int16(hdr.Len) * elemW
				inst.Height = elemH
			} else {
				// Vertical layout: width = element width, height = sum of elements
				inst.Width = elemW
				inst.Height = int16(hdr.Len) * elemH
			}

			if inst.Width > w {
				w = inst.Width
			}
			h += inst.Height
			i = elemEnd

		case KindRowStart:
			inst.X = x
			inst.Y = y

			// Layout children horizontally
			endIdx := t.skipContainer(i)
			childX := int16(0)
			maxH := int16(0)
			gap := inst.Gap
			first := true

			for j := i + 1; j < endIdx; {
				if !first && gap > 0 {
					childX += gap
				}
				childW, childH := t.layoutSingle(j, childX, 0)
				childX += childW
				if childH > maxH {
					maxH = childH
				}
				j = t.skipInstruction(j)
				first = false
			}

			inst.Width = childX
			inst.Height = maxH

			if inst.Width > w {
				w = inst.Width
			}
			h += inst.Height
			i = endIdx + 1

		case KindColStart:
			inst.X = x
			inst.Y = y

			// Layout children vertically
			endIdx := t.skipContainer(i)
			childY := int16(0)
			maxW := int16(0)
			gap := inst.Gap
			first := true

			for j := i + 1; j < endIdx; {
				if !first && gap > 0 {
					childY += gap
				}
				childW, childH := t.layoutSingle(j, 0, childY)
				childY += childH
				if childW > maxW {
					maxW = childW
				}
				j = t.skipInstruction(j)
				first = false
			}

			inst.Width = maxW
			inst.Height = childY

			if inst.Width > w {
				w = inst.Width
			}
			h += inst.Height
			i = endIdx + 1

		case KindCustom:
			inst.X = x
			inst.Y = y
			if inst.Measure != nil {
				inst.Width, inst.Height = inst.Measure(1000) // availW placeholder
			} else {
				inst.Width, inst.Height = 10, 1 // default
			}

			if inst.Width > w {
				w = inst.Width
			}
			h += inst.Height
			i++

		default:
			i++
		}
	}
	return w, h
}

// layoutSingle layouts one instruction and returns its size.
func (t *Template) layoutSingle(idx int, x, y int16) (w, h int16) {
	inst := &t.instructions[idx]
	inst.X = x
	inst.Y = y

	switch inst.Kind {
	case KindText:
		if inst.Text != "" {
			inst.Width = int16(len(inst.Text))
		} else if inst.StrPtr != nil {
			inst.Width = int16(len(*inst.StrPtr))
		} else {
			inst.Width = 10
		}
		inst.Height = 1
		return inst.Width, inst.Height

	case KindProgress:
		inst.Height = 1
		if inst.Width == 0 {
			inst.Width = inst.BarWidth
		}
		return inst.Width, inst.Height

	case KindForEach:
		// Layout element template to get dimensions
		elemEnd := idx + 1 + inst.ElemCount
		elemW, elemH := t.phaseLayout(idx+1, elemEnd, 0, 0)
		inst.ElemWidth = elemW
		inst.ElemHeight = elemH

		hdr := *(*sliceHeader)(inst.SlicePtr)
		if inst.Horizontal {
			inst.Width = int16(hdr.Len) * inst.ElemWidth
			inst.Height = inst.ElemHeight
		} else {
			inst.Width = inst.ElemWidth
			inst.Height = int16(hdr.Len) * inst.ElemHeight
		}
		return inst.Width, inst.Height

	case KindRowStart, KindColStart:
		endIdx := t.skipContainer(idx)
		_, _ = t.phaseLayout(idx, endIdx+1, x, y)
		return inst.Width, inst.Height

	case KindCustom:
		if inst.Measure != nil {
			inst.Width, inst.Height = inst.Measure(1000)
		} else {
			inst.Width, inst.Height = 10, 1
		}
		return inst.Width, inst.Height
	}

	return 0, 0
}

// skipContainer returns the index of the matching end instruction.
func (t *Template) skipContainer(startIdx int) int {
	kind := t.instructions[startIdx].Kind
	var endKind uint8
	if kind == KindRowStart {
		endKind = KindRowEnd
	} else {
		endKind = KindColEnd
	}

	depth := 1
	for j := startIdx + 1; j < len(t.instructions); j++ {
		k := t.instructions[j].Kind
		if k == kind {
			depth++
		} else if k == endKind {
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return len(t.instructions)
}

// skipInstruction returns the index after this instruction.
func (t *Template) skipInstruction(idx int) int {
	inst := &t.instructions[idx]
	switch inst.Kind {
	case KindForEach:
		return idx + 1 + inst.ElemCount
	case KindRowStart, KindColStart:
		return t.skipContainer(idx) + 1
	default:
		return idx + 1
	}
}

// phaseRender draws to buffer using computed positions.
func (t *Template) phaseRender(buf *tui.Buffer, start, end int, elemPtr unsafe.Pointer, offsetX, offsetY int16) {
	for i := start; i < end; {
		inst := &t.instructions[i]
		x := inst.X + offsetX
		y := inst.Y + offsetY

		switch inst.Kind {
		case KindText:
			text := t.getText(inst, elemPtr)
			buf.WriteStringFast(int(x), int(y), text, tui.Style{}, int(inst.Width))
			i++

		case KindProgress:
			ratio := t.getFloat(inst, elemPtr)
			buf.WriteProgressBar(int(x), int(y), int(inst.BarWidth), ratio, tui.Style{})
			i++

		case KindForEach:
			hdr := *(*sliceHeader)(inst.SlicePtr)
			elemEnd := i + 1 + inst.ElemCount

			// Viewport culling (only for vertical ForEach)
			startIdx := 0
			endIdx := hdr.Len
			if !inst.Horizontal && t.ViewportHeight > 0 {
				startIdx = t.ViewportY
				endIdx = t.ViewportY + t.ViewportHeight
				if startIdx < 0 {
					startIdx = 0
				}
				if endIdx > hdr.Len {
					endIdx = hdr.Len
				}
			}

			// Render visible elements
			for j := startIdx; j < endIdx; j++ {
				ePtr := unsafe.Add(hdr.Data, uintptr(j)*inst.ElemSize)
				if inst.Horizontal {
					// Horizontal: increment X
					elemX := x + int16(j)*inst.ElemWidth
					t.phaseRender(buf, i+1, elemEnd, ePtr, elemX, y)
				} else {
					// Vertical: increment Y
					elemY := y + int16(j-startIdx)*inst.ElemHeight
					t.phaseRender(buf, i+1, elemEnd, ePtr, x, elemY)
				}
			}
			i = elemEnd

		case KindRowStart:
			// Children positioned relative to Row, pass Row's absolute position as offset
			endIdx := t.skipContainer(i)
			t.phaseRender(buf, i+1, endIdx, elemPtr, x, y)
			i = endIdx + 1

		case KindColStart:
			// Children positioned relative to Col, pass Col's absolute position as offset
			endIdx := t.skipContainer(i)
			t.phaseRender(buf, i+1, endIdx, elemPtr, x, y)
			i = endIdx + 1

		case KindCustom:
			if inst.Render != nil {
				inst.Render(buf, x, y, inst.Width, inst.Height)
			}
			i++

		case KindRowEnd, KindColEnd:
			i++

		default:
			i++
		}
	}
}

// getText returns text content, reading from pointer/offset as needed.
func (t *Template) getText(inst *Inst, elemPtr unsafe.Pointer) string {
	if inst.Text != "" {
		return inst.Text
	}
	if inst.StrPtr != nil {
		return *inst.StrPtr
	}
	if elemPtr != nil && inst.BindKind == BindString {
		return *(*string)(unsafe.Add(elemPtr, inst.Offset))
	}
	if inst.IntPtr != nil {
		return intToStr(*inst.IntPtr)
	}
	if elemPtr != nil && inst.BindKind == BindInt {
		return intToStr(*(*int)(unsafe.Add(elemPtr, inst.Offset)))
	}
	if inst.FloatPtr != nil {
		return intToStr(int(*inst.FloatPtr * 100))
	}
	if elemPtr != nil && inst.BindKind == BindFloat32 {
		return intToStr(int(*(*float32)(unsafe.Add(elemPtr, inst.Offset)) * 100))
	}
	return ""
}

// getFloat returns float value, reading from pointer/offset as needed.
func (t *Template) getFloat(inst *Inst, elemPtr unsafe.Pointer) float32 {
	if inst.FloatPtr != nil {
		return *inst.FloatPtr
	}
	if elemPtr != nil && inst.BindKind == BindFloat32 {
		return *(*float32)(unsafe.Add(elemPtr, inst.Offset))
	}
	return 0
}

// intToStr converts int to string without allocation for common values.
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

func intToStr(v int) string {
	if v >= 0 && v <= 100 {
		return intStrings[v]
	}
	// Fallback for larger numbers
	var buf [20]byte
	i := len(buf)
	neg := v < 0
	if neg {
		v = -v
	}
	if v == 0 {
		return "0"
	}
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
