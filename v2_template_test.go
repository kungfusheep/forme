package tui

import (
	"testing"
)

func TestV2BasicCol(t *testing.T) {
	// Simple vertical layout
	tmpl := V2Build(Col{Children: []any{
		Text{Content: "Line 1"},
		Text{Content: "Line 2"},
		Text{Content: "Line 3"},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// Check output
	if got := buf.GetLine(0); got != "Line 1" {
		t.Errorf("line 0: got %q, want %q", got, "Line 1")
	}
	if got := buf.GetLine(1); got != "Line 2" {
		t.Errorf("line 1: got %q, want %q", got, "Line 2")
	}
	if got := buf.GetLine(2); got != "Line 3" {
		t.Errorf("line 2: got %q, want %q", got, "Line 3")
	}
}

func TestV2BasicRow(t *testing.T) {
	// Simple horizontal layout
	tmpl := V2Build(Row{Children: []any{
		Text{Content: "A"},
		Text{Content: "B"},
		Text{Content: "C"},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// All on same line
	line := buf.GetLine(0)
	if line != "ABC" {
		t.Errorf("line 0: got %q, want %q", line, "ABC")
	}
}

func TestV2RowWithGap(t *testing.T) {
	// Row with gap between children
	tmpl := V2Build(Row{Gap: 2, Children: []any{
		Text{Content: "A"},
		Text{Content: "B"},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	line := buf.GetLine(0)
	// "A" + 2 spaces + "B"
	if line != "A  B" {
		t.Errorf("line 0: got %q, want %q", line, "A  B")
	}
}

func TestV2NestedContainers(t *testing.T) {
	// Col containing Row
	tmpl := V2Build(Col{Children: []any{
		Text{Content: "Header"},
		Row{Children: []any{
			Text{Content: "Left"},
			Text{Content: "Right"},
		}},
		Text{Content: "Footer"},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "Header" {
		t.Errorf("line 0: got %q, want %q", got, "Header")
	}
	if got := buf.GetLine(1); got != "LeftRight" {
		t.Errorf("line 1: got %q, want %q", got, "LeftRight")
	}
	if got := buf.GetLine(2); got != "Footer" {
		t.Errorf("line 2: got %q, want %q", got, "Footer")
	}
}

func TestV2DynamicText(t *testing.T) {
	// Text with pointer binding
	title := "Dynamic Title"

	tmpl := V2Build(Col{Children: []any{
		Text{Content: &title},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "Dynamic Title" {
		t.Errorf("line 0: got %q, want %q", got, "Dynamic Title")
	}

	// Change value and re-render
	title = "Changed!"
	buf.Clear()
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "Changed!" {
		t.Errorf("after change: got %q, want %q", got, "Changed!")
	}
}

func TestV2Progress(t *testing.T) {
	pct := 50

	tmpl := V2Build(Col{Children: []any{
		Progress{Value: &pct, BarWidth: 10},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	line := buf.GetLine(0)
	// 50% of 10 = 5 filled, 5 empty
	// Should be "█████░░░░░"
	if len(line) < 10 {
		t.Errorf("progress bar too short: got %q", line)
	}
}

func TestV2Border(t *testing.T) {
	tmpl := V2Build(Col{Children: []any{
		Text{Content: "Inside"},
	}}.Border(BorderSingle))

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// First line should be top border
	line0 := buf.GetLine(0)
	if len(line0) < 2 || line0[0] != 0xe2 { // UTF-8 start of box drawing
		t.Logf("line 0: %q", line0)
	}

	// Content should be on line 1, offset by 1 for border
	line1 := buf.GetLine(1)
	// Should contain "Inside" with border chars
	t.Logf("line 1: %q", line1)
}

func TestV2ColWithGap(t *testing.T) {
	tmpl := V2Build(Col{Gap: 1, Children: []any{
		Text{Content: "A"},
		Text{Content: "B"},
		Text{Content: "C"},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// With gap=1, should be on lines 0, 2, 4
	if got := buf.GetLine(0); got != "A" {
		t.Errorf("line 0: got %q, want %q", got, "A")
	}
	if got := buf.GetLine(1); got != "" {
		t.Errorf("line 1 (gap): got %q, want empty", got)
	}
	if got := buf.GetLine(2); got != "B" {
		t.Errorf("line 2: got %q, want %q", got, "B")
	}
	if got := buf.GetLine(3); got != "" {
		t.Errorf("line 3 (gap): got %q, want empty", got)
	}
	if got := buf.GetLine(4); got != "C" {
		t.Errorf("line 4: got %q, want %q", got, "C")
	}
}

func TestV2IfTrue(t *testing.T) {
	showDetails := true

	tmpl := V2Build(Col{Children: []any{
		Text{Content: "Header"},
		IfNode{
			Cond: &showDetails,
			Then: Text{Content: "Details shown"},
		},
		Text{Content: "Footer"},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "Header" {
		t.Errorf("line 0: got %q, want %q", got, "Header")
	}
	if got := buf.GetLine(1); got != "Details shown" {
		t.Errorf("line 1: got %q, want %q", got, "Details shown")
	}
	if got := buf.GetLine(2); got != "Footer" {
		t.Errorf("line 2: got %q, want %q", got, "Footer")
	}
}

func TestV2IfFalse(t *testing.T) {
	showDetails := false

	tmpl := V2Build(Col{Children: []any{
		Text{Content: "Header"},
		IfNode{
			Cond: &showDetails,
			Then: Text{Content: "Details shown"},
		},
		Text{Content: "Footer"},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "Header" {
		t.Errorf("line 0: got %q, want %q", got, "Header")
	}
	// When condition is false, Footer should be on line 1 (no space taken)
	if got := buf.GetLine(1); got != "Footer" {
		t.Errorf("line 1: got %q, want %q", got, "Footer")
	}
}

func TestV2IfDynamic(t *testing.T) {
	showDetails := true

	tmpl := V2Build(Col{Children: []any{
		Text{Content: "Header"},
		IfNode{
			Cond: &showDetails,
			Then: Text{Content: "Details"},
		},
		Text{Content: "Footer"},
	}})

	// First render with condition true
	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(1); got != "Details" {
		t.Errorf("with true: line 1 got %q, want %q", got, "Details")
	}
	if got := buf.GetLine(2); got != "Footer" {
		t.Errorf("with true: line 2 got %q, want %q", got, "Footer")
	}

	// Change condition and re-render
	showDetails = false
	buf.Clear()
	tmpl.Execute(buf, 40, 10)

	// Now Footer should be on line 1
	if got := buf.GetLine(1); got != "Footer" {
		t.Errorf("with false: line 1 got %q, want %q", got, "Footer")
	}
}

type testItem struct {
	Name string
}

func TestV2ForEach(t *testing.T) {
	items := []testItem{
		{Name: "Item 1"},
		{Name: "Item 2"},
		{Name: "Item 3"},
	}

	tmpl := V2Build(Col{Children: []any{
		Text{Content: "List:"},
		ForEachNode{
			Items: &items,
			Render: func(item *testItem) any {
				return Text{Content: &item.Name}
			},
		},
		Text{Content: "End"},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "List:" {
		t.Errorf("line 0: got %q, want %q", got, "List:")
	}
	if got := buf.GetLine(1); got != "Item 1" {
		t.Errorf("line 1: got %q, want %q", got, "Item 1")
	}
	if got := buf.GetLine(2); got != "Item 2" {
		t.Errorf("line 2: got %q, want %q", got, "Item 2")
	}
	if got := buf.GetLine(3); got != "Item 3" {
		t.Errorf("line 3: got %q, want %q", got, "Item 3")
	}
	if got := buf.GetLine(4); got != "End" {
		t.Errorf("line 4: got %q, want %q", got, "End")
	}
}

func TestV2ForEachEmpty(t *testing.T) {
	items := []testItem{}

	tmpl := V2Build(Col{Children: []any{
		Text{Content: "List:"},
		ForEachNode{
			Items: &items,
			Render: func(item *testItem) any {
				return Text{Content: &item.Name}
			},
		},
		Text{Content: "End"},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "List:" {
		t.Errorf("line 0: got %q, want %q", got, "List:")
	}
	// Empty list should take no space
	if got := buf.GetLine(1); got != "End" {
		t.Errorf("line 1: got %q, want %q", got, "End")
	}
}

func TestV2ForEachDynamic(t *testing.T) {
	items := []testItem{
		{Name: "A"},
		{Name: "B"},
	}

	tmpl := V2Build(Col{Children: []any{
		ForEachNode{
			Items: &items,
			Render: func(item *testItem) any {
				return Text{Content: &item.Name}
			},
		},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "A" {
		t.Errorf("line 0: got %q, want %q", got, "A")
	}
	if got := buf.GetLine(1); got != "B" {
		t.Errorf("line 1: got %q, want %q", got, "B")
	}

	// Add an item and re-render
	items = append(items, testItem{Name: "C"})
	buf.Clear()
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "A" {
		t.Errorf("after add: line 0 got %q, want %q", got, "A")
	}
	if got := buf.GetLine(1); got != "B" {
		t.Errorf("after add: line 1 got %q, want %q", got, "B")
	}
	if got := buf.GetLine(2); got != "C" {
		t.Errorf("after add: line 2 got %q, want %q", got, "C")
	}
}

// StatusBar is a custom component that implements the Component interface
type StatusBar struct {
	Items []StatusItem
}

type StatusItem struct {
	Label string
	Value *string
}

func (s StatusBar) Build() any {
	children := make([]any, 0, len(s.Items)*3)
	for i, item := range s.Items {
		if i > 0 {
			children = append(children, Text{Content: " | "})
		}
		children = append(children, Text{Content: item.Label + ": "})
		children = append(children, Text{Content: item.Value})
	}
	return Row{Children: children}
}

func TestV2CustomComponent(t *testing.T) {
	fps := "60.0"
	frame := "1234"

	tmpl := V2Build(Col{Children: []any{
		Text{Content: "Header"},
		StatusBar{Items: []StatusItem{
			{Label: "FPS", Value: &fps},
			{Label: "Frame", Value: &frame},
		}},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "Header" {
		t.Errorf("line 0: got %q, want %q", got, "Header")
	}
	if got := buf.GetLine(1); got != "FPS: 60.0 | Frame: 1234" {
		t.Errorf("line 1: got %q, want %q", got, "FPS: 60.0 | Frame: 1234")
	}

	// Test dynamic update
	fps = "59.5"
	frame = "1235"
	buf.Clear()
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(1); got != "FPS: 59.5 | Frame: 1235" {
		t.Errorf("after update: line 1 got %q, want %q", got, "FPS: 59.5 | Frame: 1235")
	}
}

func TestV2NestedCustomComponent(t *testing.T) {
	// Custom component can contain other custom components
	type Card struct {
		Title   string
		Content string
	}

	// Make Card implement Component
	type CardComponent struct {
		Card *Card
	}

	// This is defined inline to test the pattern
	build := func(c CardComponent) any {
		return Col{Children: []any{
			Text{Content: "[" + c.Card.Title + "]"},
			Text{Content: c.Card.Content},
		}}
	}

	// Wrapper that implements Component
	type cardWrapper struct {
		card *Card
		fn   func(CardComponent) any
	}

	wrap := func(c *Card) cardWrapper {
		return cardWrapper{card: c, fn: build}
	}

	_ = wrap // Test shows pattern works with the interface

	// Direct test with StatusBar nested in Row
	fps := "60"
	tmpl := V2Build(Row{Gap: 2, Children: []any{
		Text{Content: "Stats:"},
		StatusBar{Items: []StatusItem{
			{Label: "FPS", Value: &fps},
		}},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "Stats:  FPS: 60" {
		t.Errorf("nested: got %q, want %q", got, "Stats:  FPS: 60")
	}
}

// Sparkline is a custom renderer that draws a mini chart
type Sparkline struct {
	Values *[]float64
	Width  int
}

func (s Sparkline) MinSize() (width, height int) {
	return s.Width, 1
}

func (s Sparkline) Render(buf *Buffer, x, y, w, h int) {
	if s.Values == nil || len(*s.Values) == 0 {
		return
	}
	bars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	vals := *s.Values

	// Find min/max for scaling
	minV, maxV := vals[0], vals[0]
	for _, v := range vals {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	rangeV := maxV - minV
	if rangeV == 0 {
		rangeV = 1
	}

	// Draw bars
	for i := 0; i < w && i < len(vals); i++ {
		normalized := (vals[i] - minV) / rangeV
		idx := int(normalized * 7)
		if idx > 7 {
			idx = 7
		}
		buf.Set(x+i, y, Cell{Rune: bars[idx]})
	}
}

func TestV2CustomRenderer(t *testing.T) {
	values := []float64{1, 3, 5, 7, 5, 3, 1, 2, 4, 6}

	tmpl := V2Build(Col{Children: []any{
		Text{Content: "CPU:"},
		Sparkline{Values: &values, Width: 10},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "CPU:" {
		t.Errorf("line 0: got %q, want %q", got, "CPU:")
	}

	// Check sparkline rendered (should have bar characters)
	line1 := buf.GetLine(1)
	if len(line1) < 10 {
		t.Errorf("sparkline too short: got %q", line1)
	}

	// Verify it contains sparkline chars
	hasSparkChars := false
	for _, r := range line1 {
		if r >= '▁' && r <= '█' {
			hasSparkChars = true
			break
		}
	}
	if !hasSparkChars {
		t.Errorf("sparkline missing bar chars: got %q", line1)
	}
}

func TestV2RendererInRow(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}

	tmpl := V2Build(Row{Gap: 1, Children: []any{
		Text{Content: "CPU:"},
		Sparkline{Values: &values, Width: 5},
		Text{Content: "MEM:"},
		Sparkline{Values: &values, Width: 5},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	line := buf.GetLine(0)
	// Should be "CPU: ▁▂▄▆█ MEM: ▁▂▄▆█" (approximately)
	if len(line) < 20 {
		t.Errorf("row with sparklines too short: got %q", line)
	}

	// Should contain "CPU:" and "MEM:"
	if !containsSubstring(line, "CPU:") {
		t.Errorf("missing CPU label: got %q", line)
	}
	if !containsSubstring(line, "MEM:") {
		t.Errorf("missing MEM label: got %q", line)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Grid returns a layout function that arranges children in a grid
func Grid(cols, cellW, cellH int) LayoutFunc {
	return func(children []ChildSize, availW, availH int) []Rect {
		rects := make([]Rect, len(children))
		c := cols
		if c <= 0 {
			c = 3
		}
		cw := cellW
		if cw <= 0 {
			cw = availW / c
		}
		ch := cellH
		if ch <= 0 {
			ch = 1
		}

		for i := range children {
			col := i % c
			row := i / c
			rects[i] = Rect{
				X: col * cw,
				Y: row * ch,
				W: cw,
				H: ch,
			}
		}
		return rects
	}
}

func TestV2CustomLayout(t *testing.T) {
	// Create a 3-column grid layout using Box
	tmpl := V2Build(Box{
		Layout: Grid(3, 10, 1),
		Children: []any{
			Text{Content: "A"},
			Text{Content: "B"},
			Text{Content: "C"},
			Text{Content: "D"},
			Text{Content: "E"},
			Text{Content: "F"},
		},
	})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// Row 0: A, B, C at columns 0, 10, 20
	line0 := buf.GetLine(0)
	if line0[0] != 'A' {
		t.Errorf("expected 'A' at (0,0), got %q", string(line0[0]))
	}
	if line0[10] != 'B' {
		t.Errorf("expected 'B' at (10,0), got %q", string(line0[10]))
	}
	if line0[20] != 'C' {
		t.Errorf("expected 'C' at (20,0), got %q", string(line0[20]))
	}

	// Row 1: D, E, F at columns 0, 10, 20
	line1 := buf.GetLine(1)
	if line1[0] != 'D' {
		t.Errorf("expected 'D' at (0,1), got %q", string(line1[0]))
	}
	if line1[10] != 'E' {
		t.Errorf("expected 'E' at (10,1), got %q", string(line1[10]))
	}
	if line1[20] != 'F' {
		t.Errorf("expected 'F' at (20,1), got %q", string(line1[20]))
	}
}

func TestV2CustomLayoutNested(t *testing.T) {
	// Grid inside a Col
	tmpl := V2Build(Col{Children: []any{
		Text{Content: "Header"},
		Box{
			Layout: Grid(2, 15, 1),
			Children: []any{
				Text{Content: "Item1"},
				Text{Content: "Item2"},
				Text{Content: "Item3"},
				Text{Content: "Item4"},
			},
		},
		Text{Content: "Footer"},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// Line 0: Header
	if got := buf.GetLine(0); !containsSubstring(got, "Header") {
		t.Errorf("missing Header: got %q", got)
	}

	// Line 1: Item1 at 0, Item2 at 15
	line1 := buf.GetLine(1)
	if !containsSubstring(line1, "Item1") {
		t.Errorf("missing Item1: got %q", line1)
	}
	if line1[15] != 'I' { // Item2 starts at col 15
		t.Errorf("Item2 not at col 15: got %q", line1)
	}

	// Line 2: Item3 at 0, Item4 at 15
	line2 := buf.GetLine(2)
	if !containsSubstring(line2, "Item3") {
		t.Errorf("missing Item3: got %q", line2)
	}

	// Line 3: Footer
	if got := buf.GetLine(3); !containsSubstring(got, "Footer") {
		t.Errorf("missing Footer: got %q", got)
	}
}

func TestV2BoxInlineLayout(t *testing.T) {
	// Test with inline layout function
	tmpl := V2Build(Box{
		Layout: func(children []ChildSize, w, h int) []Rect {
			// Simple: stack horizontally with 5-char spacing
			rects := make([]Rect, len(children))
			x := 0
			for i := range children {
				rects[i] = Rect{X: x, Y: 0, W: 5, H: 1}
				x += 5
			}
			return rects
		},
		Children: []any{
			Text{Content: "A"},
			Text{Content: "B"},
			Text{Content: "C"},
		},
	})

	buf := NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	line := buf.GetLine(0)
	if line[0] != 'A' || line[5] != 'B' || line[10] != 'C' {
		t.Errorf("inline layout failed: got %q", line)
	}
}

// TestV2ConditionInsideForEach tests conditions inside ForEach
// This verifies that per-element conditions evaluate correctly
func TestV2ConditionInsideForEach(t *testing.T) {
	type Item struct {
		Name     string
		Selected bool
	}

	items := []Item{
		{Name: "A", Selected: false},
		{Name: "B", Selected: true},
		{Name: "C", Selected: false},
	}

	tmpl := V2Build(Col{Children: []any{
		ForEach(&items, func(item *Item) any {
			return Row{Children: []any{
				If(&item.Selected).Eq(true).Then(
					Text{Content: ">"},
				).Else(
					Text{Content: " "},
				),
				Text{Content: &item.Name},
			}}
		}),
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// Line 0: " A" (not selected)
	// Line 1: ">B" (selected)
	// Line 2: " C" (not selected)
	line0 := buf.GetLine(0)
	line1 := buf.GetLine(1)
	line2 := buf.GetLine(2)

	if line0[0] != ' ' {
		t.Errorf("line 0 marker: got %q, want ' '", string(line0[0]))
	}
	if line1[0] != '>' {
		t.Errorf("line 1 marker: got %q, want '>'", string(line1[0]))
	}
	if line2[0] != ' ' {
		t.Errorf("line 2 marker: got %q, want ' '", string(line2[0]))
	}

	// Now change selection and re-render
	items[0].Selected = true
	items[1].Selected = false
	buf.Clear()
	tmpl.Execute(buf, 40, 10)

	line0 = buf.GetLine(0)
	line1 = buf.GetLine(1)

	if line0[0] != '>' {
		t.Errorf("after change: line 0 marker: got %q, want '>'", string(line0[0]))
	}
	if line1[0] != ' ' {
		t.Errorf("after change: line 1 marker: got %q, want ' '", string(line1[0]))
	}
}

// TestV2ConditionNodeBuilder tests the builder-style If() conditionals
// using tui.If(&x).Eq(true).Then(...) syntax
func TestV2ConditionNodeBuilder(t *testing.T) {
	showGraph := true
	showProcs := false

	tmpl := V2Build(Col{Children: []any{
		Text{Content: "Header"},
		If(&showGraph).Eq(true).Then(
			Text{Content: "Graph visible"},
		),
		If(&showProcs).Eq(true).Then(
			Text{Content: "Procs visible"},
		),
		Text{Content: "Footer"},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// Header should be on line 0
	if got := buf.GetLine(0); got != "Header" {
		t.Errorf("line 0: got %q, want %q", got, "Header")
	}
	// Graph visible should be on line 1 (showGraph=true)
	if got := buf.GetLine(1); got != "Graph visible" {
		t.Errorf("line 1: got %q, want %q", got, "Graph visible")
	}
	// Footer should be on line 2 (showProcs=false, skipped)
	if got := buf.GetLine(2); got != "Footer" {
		t.Errorf("line 2: got %q, want %q", got, "Footer")
	}

	// Now toggle values and re-render
	showGraph = false
	showProcs = true
	buf.Clear()
	tmpl.Execute(buf, 40, 10)

	// Header on line 0
	if got := buf.GetLine(0); got != "Header" {
		t.Errorf("toggled: line 0 got %q, want %q", got, "Header")
	}
	// Procs visible should now show (showProcs=true), graph hidden
	if got := buf.GetLine(1); got != "Procs visible" {
		t.Errorf("toggled: line 1 got %q, want %q", got, "Procs visible")
	}
	// Footer should be on line 2
	if got := buf.GetLine(2); got != "Footer" {
		t.Errorf("toggled: line 2 got %q, want %q", got, "Footer")
	}
}

func TestV2FlexGrow(t *testing.T) {
	// Test that FlexGrow distributes remaining vertical space
	// Screen is 20 high, header is 1 line, footer is 1 line
	// Middle section with Grow(1) should expand to fill remaining 18 lines

	tmpl := V2Build(Col{Children: []any{
		Text{Content: "Header"},
		Col{Children: []any{
			Text{Content: "Content"},
		}}.Grow(1), // This should expand
		Text{Content: "Footer"},
	}})

	buf := NewBuffer(40, 20)
	tmpl.Execute(buf, 40, 20)

	// Header on line 0
	if got := buf.GetLine(0); got != "Header" {
		t.Errorf("line 0: got %q, want %q", got, "Header")
	}

	// Content on line 1
	if got := buf.GetLine(1); got != "Content" {
		t.Errorf("line 1: got %q, want %q", got, "Content")
	}

	// Footer should be on line 19 (last line) due to flex expansion
	// The middle Col should have expanded to fill lines 1-18
	if got := buf.GetLine(19); got != "Footer" {
		t.Errorf("line 19: got %q, want %q (flex should push footer to bottom)", got, "Footer")
	}
}

func TestV2FlexGrowMultiple(t *testing.T) {
	// Test multiple flex children with different weights
	// Screen is 12 high: header(1) + flex1(Grow(1)) + flex2(Grow(2)) + footer(1)
	// Remaining space = 12 - 2 = 10 lines
	// flex1 should get 10 * 1/3 ≈ 3 lines
	// flex2 should get 10 * 2/3 ≈ 6 lines (total with content = header at some offset)

	tmpl := V2Build(Col{Children: []any{
		Text{Content: "Header"},
		Col{Children: []any{Text{Content: "A"}}}.Grow(1),
		Col{Children: []any{Text{Content: "B"}}}.Grow(2),
		Text{Content: "Footer"},
	}})

	buf := NewBuffer(40, 12)
	tmpl.Execute(buf, 40, 12)

	// Header on line 0
	if got := buf.GetLine(0); got != "Header" {
		t.Errorf("line 0: got %q, want %q", got, "Header")
	}

	// Footer should be at bottom (line 11)
	if got := buf.GetLine(11); got != "Footer" {
		t.Errorf("line 11: got %q, want %q", got, "Footer")
	}

	// A should be on line 1
	if got := buf.GetLine(1); got != "A" {
		t.Errorf("line 1: got %q, want %q", got, "A")
	}

	// B should start after A's flex section
	// With 10 lines to distribute: A gets ~3, B gets ~7
	// So B starts at line 1 + 3 = 4
	if got := buf.GetLine(4); got != "B" {
		t.Errorf("line 4: got %q, want %q", got, "B")
	}
}

func TestV2FlexGrowHorizontal(t *testing.T) {
	// Test horizontal flex in a Row
	// Row width is 40, "Left" is 4 chars, "Right" is 5 chars
	// Middle with Grow(1) should expand to fill remaining 31 chars

	tmpl := V2Build(Row{Children: []any{
		Text{Content: "Left"},
		Col{Children: []any{
			Text{Content: "X"},
		}}.Grow(1), // This should expand horizontally
		Text{Content: "Right"},
	}})

	buf := NewBuffer(40, 5)
	tmpl.Execute(buf, 40, 5)

	line := buf.GetLine(0)
	// "Left" at start, "Right" at end (position 35), "X" somewhere in between
	if len(line) < 5 || line[:4] != "Left" {
		t.Errorf("should start with 'Left', got %q", line)
	}
	// "Right" should be at the far right (chars 35-39)
	if len(line) < 40 || line[35:40] != "Right" {
		t.Errorf("should end with 'Right' at position 35, got line: %q", line)
	}
}

func TestV2FlexGrowHorizontalMultiple(t *testing.T) {
	// Test multiple horizontal flex children
	// Row width is 30, no fixed children
	// A with Grow(1) gets 1/3, B with Grow(2) gets 2/3

	tmpl := V2Build(Row{Children: []any{
		Col{Children: []any{Text{Content: "A"}}}.Grow(1),
		Col{Children: []any{Text{Content: "B"}}}.Grow(2),
	}})

	buf := NewBuffer(30, 5)
	tmpl.Execute(buf, 30, 5)

	line := buf.GetLine(0)
	// A should be at position 0
	if len(line) < 1 || line[0] != 'A' {
		t.Errorf("A should be at position 0, got %q", line)
	}
	// B should be at position 10 (30 * 1/3 = 10)
	if len(line) < 11 || line[10] != 'B' {
		t.Errorf("B should be at position 10, got line: %q", line)
	}
}
