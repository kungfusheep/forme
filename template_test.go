package forme

import (
	"fmt"
	"strings"
	"testing"
)

func TestV2BasicCol(t *testing.T) {
	// Simple vertical layout
	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Line 1"},
		TextNode{Content: "Line 2"},
		TextNode{Content: "Line 3"},
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
	tmpl := Build(HBoxNode{Children: []any{
		TextNode{Content: "A"},
		TextNode{Content: "B"},
		TextNode{Content: "C"},
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
	tmpl := Build(HBoxNode{Gap: 2, Children: []any{
		TextNode{Content: "A"},
		TextNode{Content: "B"},
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
	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		HBoxNode{Children: []any{
			TextNode{Content: "Left"},
			TextNode{Content: "Right"},
		}},
		TextNode{Content: "Footer"},
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

	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: &title},
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

	tmpl := Build(VBoxNode{Children: []any{
		ProgressNode{Value: &pct, BarWidth: 10},
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
	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Inside"},
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
	tmpl := Build(VBoxNode{Gap: 1, Children: []any{
		TextNode{Content: "A"},
		TextNode{Content: "B"},
		TextNode{Content: "C"},
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

	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		IfNode{
			Cond: &showDetails,
			Then: TextNode{Content: "Details shown"},
		},
		TextNode{Content: "Footer"},
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

	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		IfNode{
			Cond: &showDetails,
			Then: TextNode{Content: "Details shown"},
		},
		TextNode{Content: "Footer"},
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

	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		IfNode{
			Cond: &showDetails,
			Then: TextNode{Content: "Details"},
		},
		TextNode{Content: "Footer"},
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

	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "List:"},
		ForEachNode{
			Items: &items,
			Render: func(item *testItem) any {
				return TextNode{Content: &item.Name}
			},
		},
		TextNode{Content: "End"},
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

	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "List:"},
		ForEachNode{
			Items: &items,
			Render: func(item *testItem) any {
				return TextNode{Content: &item.Name}
			},
		},
		TextNode{Content: "End"},
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

	tmpl := Build(VBoxNode{Children: []any{
		ForEachNode{
			Items: &items,
			Render: func(item *testItem) any {
				return TextNode{Content: &item.Name}
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
			children = append(children, TextNode{Content: " | "})
		}
		children = append(children, TextNode{Content: item.Label + ": "})
		children = append(children, TextNode{Content: item.Value})
	}
	return HBoxNode{Children: children}
}

func TestV2CustomComponent(t *testing.T) {
	fps := "60.0"
	frame := "1234"

	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
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
		return VBoxNode{Children: []any{
			TextNode{Content: "[" + c.Card.Title + "]"},
			TextNode{Content: c.Card.Content},
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

	// Direct test with StatusBar nested in HBox
	fps := "60"
	tmpl := Build(HBoxNode{Gap: 2, Children: []any{
		TextNode{Content: "Stats:"},
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

// CustomSparkline is a custom renderer example that draws a mini chart
// (Used to test the Renderer interface)
type CustomSparkline struct {
	Values *[]float64
	Width  int
}

func (s CustomSparkline) MinSize() (width, height int) {
	return s.Width, 1
}

func (s CustomSparkline) Render(buf *Buffer, x, y, w, h int) {
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

func TestCustomRenderer(t *testing.T) {
	values := []float64{1, 3, 5, 7, 5, 3, 1, 2, 4, 6}

	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "CPU:"},
		CustomSparkline{Values: &values, Width: 10},
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

	tmpl := Build(HBoxNode{Gap: 1, Children: []any{
		TextNode{Content: "CPU:"},
		SparklineNode{Values: &values, Width: 5},
		TextNode{Content: "MEM:"},
		SparklineNode{Values: &values, Width: 5},
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
	tmpl := Build(Box{
		Layout: Grid(3, 10, 1),
		Children: []any{
			TextNode{Content: "A"},
			TextNode{Content: "B"},
			TextNode{Content: "C"},
			TextNode{Content: "D"},
			TextNode{Content: "E"},
			TextNode{Content: "F"},
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
	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		Box{
			Layout: Grid(2, 15, 1),
			Children: []any{
				TextNode{Content: "Item1"},
				TextNode{Content: "Item2"},
				TextNode{Content: "Item3"},
				TextNode{Content: "Item4"},
			},
		},
		TextNode{Content: "Footer"},
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
	tmpl := Build(Box{
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
			TextNode{Content: "A"},
			TextNode{Content: "B"},
			TextNode{Content: "C"},
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

	tmpl := Build(VBoxNode{Children: []any{
		ForEach(&items, func(item *Item) any {
			return HBoxNode{Children: []any{
				If(&item.Selected).Eq(true).Then(
					TextNode{Content: ">"},
				).Else(
					TextNode{Content: " "},
				),
				TextNode{Content: &item.Name},
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
// using forme.If(&x).Eq(true).Then(...) syntax
func TestV2ConditionNodeBuilder(t *testing.T) {
	showGraph := true
	showProcs := false

	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		If(&showGraph).Eq(true).Then(
			TextNode{Content: "Graph visible"},
		),
		If(&showProcs).Eq(true).Then(
			TextNode{Content: "Procs visible"},
		),
		TextNode{Content: "Footer"},
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

	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		VBoxNode{Children: []any{
			TextNode{Content: "Content"},
		}}.Grow(1), // This should expand
		TextNode{Content: "Footer"},
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

	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		VBoxNode{Children: []any{TextNode{Content: "A"}}}.Grow(1),
		VBoxNode{Children: []any{TextNode{Content: "B"}}}.Grow(2),
		TextNode{Content: "Footer"},
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

	tmpl := Build(HBoxNode{Children: []any{
		TextNode{Content: "Left"},
		VBoxNode{Children: []any{
			TextNode{Content: "X"},
		}}.Grow(1), // This should expand horizontally
		TextNode{Content: "Right"},
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

	tmpl := Build(HBoxNode{Children: []any{
		VBoxNode{Children: []any{TextNode{Content: "A"}}}.Grow(1),
		VBoxNode{Children: []any{TextNode{Content: "B"}}}.Grow(2),
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

func TestJumpWrapsVBox(t *testing.T) {
	// Jump should be a transparent wrapper - VBox content should display
	called := false
	tmpl := Build(VBoxNode{Children: []any{
		JumpNode{
			Child: VBoxNode{Children: []any{
				TextNode{Content: "Line 1"},
				TextNode{Content: "Line 2"},
			}},
			OnSelect: func() { called = true },
		},
	}})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// Line 0 should have "Line 1"
	line0 := buf.GetLine(0)
	if len(line0) < 6 || line0[:6] != "Line 1" {
		t.Errorf("Line 0 should start with 'Line 1', got %q", line0)
	}

	// Line 1 should have "Line 2"
	line1 := buf.GetLine(1)
	if len(line1) < 6 || line1[:6] != "Line 2" {
		t.Errorf("Line 1 should start with 'Line 2', got %q", line1)
	}

	_ = called // suppress unused warning
}

func TestJumpInHBoxWithSibling(t *testing.T) {
	// Both VBox children in HBox should get width - even when one is wrapped in Jump
	tmpl := Build(HBoxNode{Gap: 2, Children: []any{
		VBoxNode{Children: []any{
			TextNode{Content: "Panel 1"},
		}},
		JumpNode{
			Child: VBoxNode{Children: []any{
				TextNode{Content: "Panel 2"},
			}},
			OnSelect: func() {},
		},
	}})

	buf := NewBuffer(40, 5)
	tmpl.Execute(buf, 40, 5)

	line := buf.GetLine(0)
	// Panel 1 should be at the start
	if len(line) < 7 || line[:7] != "Panel 1" {
		t.Errorf("Line should start with 'Panel 1', got %q", line)
	}
	// Panel 2 should appear after Panel 1 + gap
	// With implicit flex, each gets ~19 chars, so Panel 2 starts around position 21
	if !strings.Contains(line, "Panel 2") {
		t.Errorf("Line should contain 'Panel 2', got %q", line)
	}
}

// TestForEachMultipleStringFields tests ForEach with multiple string field pointers
// This was reported as a bug: rendering multiple fields from the same struct produces
// garbled output instead of the correct values.
func TestForEachMultipleStringFields(t *testing.T) {
	type Item struct {
		Icon        string
		Label       string
		Description string
	}

	items := []Item{
		{Icon: "📁", Label: "Open", Description: "Open a file"},
		{Icon: "💾", Label: "Save", Description: "Save changes"},
		{Icon: "🔍", Label: "Find", Description: "Search text"},
	}

	tmpl := Build(VBoxNode{Children: []any{
		ForEach(&items, func(item *Item) any {
			return HBoxNode{Gap: 1, Children: []any{
				TextNode{Content: &item.Icon},
				TextNode{Content: &item.Label},
				TextNode{Content: &item.Description},
			}}
		}),
	}})

	buf := NewBuffer(60, 10)
	tmpl.Execute(buf, 60, 10)

	// Each line should have: Icon Label Description
	line0 := buf.GetLine(0)
	line1 := buf.GetLine(1)
	line2 := buf.GetLine(2)

	t.Logf("Line 0: %q", line0)
	t.Logf("Line 1: %q", line1)
	t.Logf("Line 2: %q", line2)

	// Check that each line contains the expected strings (not garbled)
	if !strings.Contains(line0, "Open") {
		t.Errorf("Line 0 should contain 'Open', got %q", line0)
	}
	if !strings.Contains(line0, "Open a file") {
		t.Errorf("Line 0 should contain 'Open a file', got %q", line0)
	}

	if !strings.Contains(line1, "Save") {
		t.Errorf("Line 1 should contain 'Save', got %q", line1)
	}
	if !strings.Contains(line1, "Save changes") {
		t.Errorf("Line 1 should contain 'Save changes', got %q", line1)
	}

	if !strings.Contains(line2, "Find") {
		t.Errorf("Line 2 should contain 'Find', got %q", line2)
	}
	if !strings.Contains(line2, "Search text") {
		t.Errorf("Line 2 should contain 'Search text', got %q", line2)
	}
}

// TestSelectionListMultipleFields tests SelectionList with complex HBox render
// SelectionList now supports complex layouts (HBox/VBox) in the Render function
func TestSelectionListMultipleFields(t *testing.T) {
	type Item struct {
		Icon        string
		Label       string
		Description string
	}

	items := []Item{
		{Icon: "F", Label: "Open File", Description: "Opens a file"},
		{Icon: "S", Label: "Save", Description: "Saves changes"},
		{Icon: "Z", Label: "Toggle Zen Mode", Description: "Focus mode"},
	}

	selected := 0

	list := &SelectionList{
		Items:      &items,
		Selected:   &selected,
		Marker:     "> ",
		MaxVisible: 10,
		Render: func(item *Item) any {
			return HBoxNode{Gap: 1, Children: []any{
				TextNode{Content: &item.Icon},
				TextNode{Content: &item.Label},
				TextNode{Content: &item.Description},
			}}
		},
	}

	tmpl := Build(VBoxNode{Children: []any{list}})
	buf := NewBuffer(60, 10)
	tmpl.Execute(buf, 60, 10)

	line0 := buf.GetLine(0)
	line1 := buf.GetLine(1)
	line2 := buf.GetLine(2)

	t.Logf("SelectionList Line 0: %q", line0)
	t.Logf("SelectionList Line 1: %q", line1)
	t.Logf("SelectionList Line 2: %q", line2)

	// Verify that each line contains the expected content
	// Line 0 is selected, should have "> " marker
	if !strings.HasPrefix(line0, "> ") {
		t.Errorf("Line 0 should start with '> ', got %q", line0)
	}
	if !strings.Contains(line0, "Open File") {
		t.Errorf("Line 0 should contain 'Open File', got %q", line0)
	}
	if !strings.Contains(line0, "Opens a file") {
		t.Errorf("Line 0 should contain 'Opens a file', got %q", line0)
	}

	// Line 1 not selected, should have "  " (spaces)
	if !strings.HasPrefix(line1, "  ") {
		t.Errorf("Line 1 should start with spaces, got %q", line1)
	}
	if !strings.Contains(line1, "Save") {
		t.Errorf("Line 1 should contain 'Save', got %q", line1)
	}

	// Line 2 should have "Toggle Zen Mode"
	if !strings.Contains(line2, "Toggle Zen Mode") {
		t.Errorf("Line 2 should contain 'Toggle Zen Mode', got %q", line2)
	}
}

// TestSelectionListDefaultStyle tests that Style applies background to non-selected rows
func TestSelectionListDefaultStyle(t *testing.T) {
	items := []string{"Apple", "Banana", "Cherry"}
	selected := 1 // Banana selected

	bgColor := PaletteColor(236)      // Default background
	selectedBG := PaletteColor(240)   // Selected background

	list := &SelectionList{
		Items:         &items,
		Selected:      &selected,
		Marker:        "> ",
		Style:         Style{BG: bgColor},
		SelectedStyle: Style{BG: selectedBG},
	}

	tmpl := Build(VBoxNode{Children: []any{list}})
	buf := NewBuffer(20, 3)
	tmpl.Execute(buf, 20, 3)

	// Line 0 (Apple - non-selected) should have default background
	cell0 := buf.Get(0, 0)
	if cell0.Style.BG != bgColor {
		t.Errorf("Non-selected row (0) should have default BG, got %v", cell0.Style.BG)
	}

	// Line 1 (Banana - selected) should have selected background
	cell1 := buf.Get(0, 1)
	if cell1.Style.BG != selectedBG {
		t.Errorf("Selected row (1) should have selected BG, got %v", cell1.Style.BG)
	}

	// Line 2 (Cherry - non-selected) should have default background
	cell2 := buf.Get(0, 2)
	if cell2.Style.BG != bgColor {
		t.Errorf("Non-selected row (2) should have default BG, got %v", cell2.Style.BG)
	}
}

// TestSpacerGrow tests that SpacerNode{} defaults to grow and fills available space
func TestSpacerGrow(t *testing.T) {
	// SpacerNode{} should grow to fill remaining space in HBox
	tmpl := Build(HBoxNode{Children: []any{
		TextNode{Content: "Left"},
		SpacerNode{},
		TextNode{Content: "Right"},
	}})

	buf := NewBuffer(20, 1)
	tmpl.Execute(buf, 20, 1)

	line := buf.GetLine(0)
	t.Logf("Line: %q", line)

	// "Left" at start, "Right" at end, space in between
	if !strings.HasPrefix(line, "Left") {
		t.Errorf("Should start with 'Left', got %q", line)
	}
	if !strings.HasSuffix(strings.TrimRight(line, " "), "Right") {
		t.Errorf("Should end with 'Right', got %q", line)
	}
}

// TestSpacerWithChar tests that SpacerNode{Char: '.'} fills with dots
func TestSpacerWithChar(t *testing.T) {
	tmpl := Build(HBoxNode{Children: []any{
		TextNode{Content: "A"},
		SpacerNode{Char: '.'},
		TextNode{Content: "B"},
	}})

	buf := NewBuffer(10, 1)
	tmpl.Execute(buf, 10, 1)

	line := buf.GetLine(0)
	t.Logf("Line: %q", line)

	// Should be "A........B" (8 dots between A and B)
	if line[0] != 'A' {
		t.Errorf("Should start with 'A', got %q", line)
	}
	if line[9] != 'B' {
		t.Errorf("Should end with 'B', got %c at position 9", line[9])
	}
	// Check for dots in between
	if !strings.Contains(line, "...") {
		t.Errorf("Should contain dots, got %q", line)
	}
}

// TestSpacerFixed tests that SpacerNode{Height: 1} is fixed (no grow)
func TestSpacerFixed(t *testing.T) {
	tmpl := Build(HBoxNode{Children: []any{
		TextNode{Content: "A"},
		SpacerNode{Width: 3}, // fixed 3-char spacer
		TextNode{Content: "B"},
	}})

	buf := NewBuffer(20, 1)
	tmpl.Execute(buf, 20, 1)

	line := buf.GetLine(0)
	t.Logf("Line: %q", line)

	// Should be "A   B" - exactly 3 spaces between
	if line != "A   B" {
		t.Errorf("Expected 'A   B', got %q", line)
	}
}

// TestComplexNestedLayouts is a stress test for deeply nested layouts
func TestComplexNestedLayouts(t *testing.T) {
	t.Run("deeply nested VBox in HBox", func(t *testing.T) {
		// HBox containing multiple VBoxes
		tmpl := Build(HBoxNode{Gap: 1, Children: []any{
			VBoxNode{Children: []any{
				TextNode{Content: "A1"},
				TextNode{Content: "A2"},
				TextNode{Content: "A3"},
			}},
			VBoxNode{Children: []any{
				TextNode{Content: "B1"},
				TextNode{Content: "B2"},
			}},
			VBoxNode{Children: []any{
				TextNode{Content: "C1"},
			}},
		}})

		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		// Row 0 should have A1, B1, C1
		line0 := buf.GetLine(0)
		t.Logf("Line 0: %q", line0)
		if !strings.Contains(line0, "A1") || !strings.Contains(line0, "B1") || !strings.Contains(line0, "C1") {
			t.Errorf("Line 0 should contain A1, B1, C1: got %q", line0)
		}

		// Row 1 should have A2, B2
		line1 := buf.GetLine(1)
		t.Logf("Line 1: %q", line1)
		if !strings.Contains(line1, "A2") || !strings.Contains(line1, "B2") {
			t.Errorf("Line 1 should contain A2, B2: got %q", line1)
		}
	})

	t.Run("HBox inside VBox inside HBox", func(t *testing.T) {
		// 3 levels of nesting
		tmpl := Build(HBoxNode{Gap: 1, Children: []any{
			TextNode{Content: "["},
			VBoxNode{Children: []any{
				HBoxNode{Children: []any{
					TextNode{Content: "X"},
					TextNode{Content: "Y"},
				}},
				HBoxNode{Children: []any{
					TextNode{Content: "1"},
					TextNode{Content: "2"},
				}},
			}},
			TextNode{Content: "]"},
		}})

		buf := NewBuffer(20, 3)
		tmpl.Execute(buf, 20, 3)

		line0 := buf.GetLine(0)
		line1 := buf.GetLine(1)
		t.Logf("Line 0: %q", line0)
		t.Logf("Line 1: %q", line1)

		if !strings.Contains(line0, "X") || !strings.Contains(line0, "Y") {
			t.Errorf("Line 0 should contain XY: got %q", line0)
		}
		if !strings.Contains(line1, "1") || !strings.Contains(line1, "2") {
			t.Errorf("Line 1 should contain 12: got %q", line1)
		}
	})

	t.Run("bordered container with nested content", func(t *testing.T) {
		tmpl := Build(VBoxNode{Children: []any{
			HBoxNode{Gap: 1, Children: []any{
				TextNode{Content: "Name:"},
				TextNode{Content: "Value"},
			}},
			HBoxNode{Gap: 1, Children: []any{
				TextNode{Content: "Foo:"},
				SpacerNode{},
				TextNode{Content: "Bar"},
			}},
		}}.Border(BorderRounded))

		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		for y := 0; y < 5; y++ {
			t.Logf("Line %d: %q", y, buf.GetLine(y))
		}

		// Top border
		line0 := buf.GetLine(0)
		if !strings.Contains(line0, "╭") {
			t.Errorf("Should have top-left corner: %q", line0)
		}

		// Content line should have Name: Value
		line1 := buf.GetLine(1)
		if !strings.Contains(line1, "Name:") || !strings.Contains(line1, "Value") {
			t.Errorf("Line 1 should contain 'Name:' and 'Value': %q", line1)
		}

		// Content with spacer
		line2 := buf.GetLine(2)
		if !strings.Contains(line2, "Foo:") || !strings.Contains(line2, "Bar") {
			t.Errorf("Line 2 should contain 'Foo:' and 'Bar': %q", line2)
		}
	})

	t.Run("nested borders", func(t *testing.T) {
		tmpl := Build(VBoxNode{Children: []any{
			VBoxNode{Children: []any{
				TextNode{Content: "Inner"},
			}}.Border(BorderSingle),
		}}.Border(BorderDouble))

		buf := NewBuffer(20, 6)
		tmpl.Execute(buf, 20, 6)

		for y := 0; y < 6; y++ {
			t.Logf("Line %d: %q", y, buf.GetLine(y))
		}

		// Outer border should use double lines
		line0 := buf.GetLine(0)
		if !strings.Contains(line0, "╔") {
			t.Errorf("Should have double top-left corner: %q", line0)
		}

		// Inner border should use single lines
		line1 := buf.GetLine(1)
		if !strings.Contains(line1, "┌") {
			t.Errorf("Should have single top-left corner in inner: %q", line1)
		}
	})

	t.Run("multiple spacers in HBox", func(t *testing.T) {
		tmpl := Build(HBoxNode{Children: []any{
			TextNode{Content: "L"},
			SpacerNode{},
			TextNode{Content: "M"},
			SpacerNode{},
			TextNode{Content: "R"},
		}})

		buf := NewBuffer(21, 1)
		tmpl.Execute(buf, 21, 1)

		line := buf.GetLine(0)
		t.Logf("Line: %q", line)

		// L should be at start, M in middle, R at end
		if line[0] != 'L' {
			t.Errorf("Should start with L: %q", line)
		}
		if line[10] != 'M' {
			t.Errorf("M should be at position 10: got %c at 10, line: %q", line[10], line)
		}
		if line[20] != 'R' {
			t.Errorf("R should be at position 20: got %c at 20, line: %q", line[20], line)
		}
	})

	t.Run("ForEach with nested HBox", func(t *testing.T) {
		type Row struct {
			Key   string
			Value string
		}
		rows := []Row{
			{Key: "alpha", Value: "1"},
			{Key: "beta", Value: "2"},
			{Key: "gamma", Value: "3"},
		}

		tmpl := Build(VBoxNode{Children: []any{
			ForEach(&rows, func(r *Row) any {
				return HBoxNode{Gap: 1, Children: []any{
					TextNode{Content: &r.Key},
					SpacerNode{Char: '.'},
					TextNode{Content: &r.Value},
				}}
			}),
		}})

		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		for y := 0; y < 3; y++ {
			t.Logf("Line %d: %q", y, buf.GetLine(y))
		}

		// Each line should have key...value pattern
		line0 := buf.GetLine(0)
		if !strings.HasPrefix(line0, "alpha") || !strings.Contains(line0, "...") || !strings.HasSuffix(strings.TrimSpace(line0), "1") {
			t.Errorf("Line 0 should be 'alpha...1' pattern: %q", line0)
		}
	})

	t.Run("SelectionList with deeply nested Render", func(t *testing.T) {
		type MenuItem struct {
			Icon     string
			Label    string
			Shortcut string
			Enabled  bool
		}
		items := []MenuItem{
			{Icon: "*", Label: "New File", Shortcut: "Ctrl+N", Enabled: true},
			{Icon: "#", Label: "Open", Shortcut: "Ctrl+O", Enabled: true},
			{Icon: "!", Label: "Save", Shortcut: "Ctrl+S", Enabled: false},
		}
		selected := 1

		list := &SelectionList{
			Items:      &items,
			Selected:   &selected,
			Marker:     "> ",
			MaxVisible: 10,
			Render: func(item *MenuItem) any {
				// Complex nested layout: HBox with VBox inside
				return HBoxNode{Gap: 1, Children: []any{
					TextNode{Content: &item.Icon},
					VBoxNode{Children: []any{
						HBoxNode{Children: []any{
							TextNode{Content: &item.Label},
							SpacerNode{},
							TextNode{Content: &item.Shortcut},
						}},
					}},
				}}
			},
		}

		tmpl := Build(VBoxNode{Children: []any{list}})
		buf := NewBuffer(40, 5)
		tmpl.Execute(buf, 40, 5)

		for y := 0; y < 3; y++ {
			t.Logf("Line %d: %q", y, buf.GetLine(y))
		}

		// Line 0: not selected, should have icon and label
		line0 := buf.GetLine(0)
		if !strings.Contains(line0, "*") || !strings.Contains(line0, "New File") {
			t.Errorf("Line 0 should have icon and label: %q", line0)
		}

		// Line 1: selected, should have marker
		line1 := buf.GetLine(1)
		if !strings.HasPrefix(line1, "> ") {
			t.Errorf("Line 1 should have selection marker: %q", line1)
		}
		if !strings.Contains(line1, "#") || !strings.Contains(line1, "Open") {
			t.Errorf("Line 1 should have icon and label: %q", line1)
		}
	})

	t.Run("grow factors compete correctly", func(t *testing.T) {
		// Two spacers with different grow factors
		tmpl := Build(HBoxNode{Children: []any{
			TextNode{Content: "A"},
			SpacerNode{}.Grow(1),
			TextNode{Content: "B"},
			SpacerNode{}.Grow(2), // should get 2x the space
			TextNode{Content: "C"},
		}})

		buf := NewBuffer(30, 1)
		tmpl.Execute(buf, 30, 1)

		line := buf.GetLine(0)
		t.Logf("Line: %q", line)

		// A at 0, C at 29
		if line[0] != 'A' {
			t.Errorf("A should be at 0: %q", line)
		}
		if line[29] != 'C' {
			t.Errorf("C should be at 29: %q", line)
		}

		// B should be closer to A than C (due to 1:2 ratio)
		bPos := strings.Index(line, "B")
		if bPos < 5 || bPos > 15 {
			t.Errorf("B should be around position 9-10 (1/3 of remaining space): at %d in %q", bPos, line)
		}
	})

	t.Run("styled text in nested containers", func(t *testing.T) {
		style1 := Style{FG: Red}
		style2 := Style{FG: Green, Attr: AttrBold}

		tmpl := Build(VBoxNode{Children: []any{
			HBoxNode{Children: []any{
				TextNode{Content: "Red", Style: style1},
				SpacerNode{Width: 1},
				TextNode{Content: "Green", Style: style2},
			}},
		}}.Border(BorderSingle))

		buf := NewBuffer(20, 3)
		tmpl.Execute(buf, 20, 3)

		for y := 0; y < 3; y++ {
			t.Logf("Line %d: %q", y, buf.GetLine(y))
		}

		// Check content exists
		line1 := buf.GetLine(1)
		if !strings.Contains(line1, "Red") || !strings.Contains(line1, "Green") {
			t.Errorf("Should contain styled text: %q", line1)
		}

		// Check styles are applied (via cell inspection)
		// Find "Red" text position and check its style
		redPos := strings.Index(line1, "Red")
		if redPos >= 0 {
			cell := buf.Get(redPos, 1)
			if cell.Style.FG != Red {
				t.Errorf("'Red' text should have red foreground: got %v", cell.Style.FG)
			}
		}
	})

	t.Run("zero-width edge case", func(t *testing.T) {
		tmpl := Build(HBoxNode{Children: []any{
			TextNode{Content: "X"},
		}})

		// Execute with zero width - should not panic
		buf := NewBuffer(0, 1)
		tmpl.Execute(buf, 0, 1)
		// If we get here without panic, test passes
	})

	t.Run("very narrow container with nested content", func(t *testing.T) {
		tmpl := Build(VBoxNode{Children: []any{
			HBoxNode{Children: []any{
				TextNode{Content: "TooLongText"},
			}},
		}}.Border(BorderSingle))

		buf := NewBuffer(8, 3) // Very narrow - content will be clipped
		tmpl.Execute(buf, 8, 3)

		for y := 0; y < 3; y++ {
			t.Logf("Line %d: %q", y, buf.GetLine(y))
		}

		// Should render border and clip content, not panic
		line0 := buf.GetLine(0)
		if !strings.Contains(line0, "┌") {
			t.Errorf("Should have border: %q", line0)
		}
	})

	t.Run("HRule inside bordered VBox", func(t *testing.T) {
		tmpl := Build(VBoxNode{Children: []any{
			TextNode{Content: "Header"},
			HRuleNode{},
			TextNode{Content: "Body"},
		}}.Border(BorderSingle))

		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		for y := 0; y < 5; y++ {
			t.Logf("Line %d: %q", y, buf.GetLine(y))
		}

		// HRule should not bleed outside border
		line2 := buf.GetLine(2)
		// First char should be border │, not ─
		runes := []rune(line2)
		if len(runes) > 0 && runes[0] != '│' {
			t.Errorf("HRule should be inside border, first char should be │: %q (first rune: %c)", line2, runes[0])
		}
	})

	t.Run("deeply nested 5 levels", func(t *testing.T) {
		tmpl := Build(
			VBoxNode{Children: []any{
				HBoxNode{Children: []any{
					VBoxNode{Children: []any{
						HBoxNode{Children: []any{
							VBoxNode{Children: []any{
								TextNode{Content: "DEEP"},
							}},
						}},
					}},
				}},
			}},
		)

		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		line0 := buf.GetLine(0)
		t.Logf("Line 0: %q", line0)

		if !strings.Contains(line0, "DEEP") {
			t.Errorf("Deeply nested text should render: %q", line0)
		}
	})
}

// TestHBoxWithLayerView tests the scenario that caused blank screens in the editor
// when adding a sidebar (HBox) alongside a LayerView with custom content.
func TestHBoxWithLayerView(t *testing.T) {
	t.Run("basic HBox with LayerView", func(t *testing.T) {
		// Create a layer with some content
		layer := NewLayer()
		layer.EnsureSize(40, 5)
		layer.SetLineString(0, "Editor line 1", DefaultStyle())
		layer.SetLineString(1, "Editor line 2", DefaultStyle())
		layer.SetLineString(2, "Editor line 3", DefaultStyle())

		tmpl := Build(HBoxNode{Children: []any{
			VBoxNode{Children: []any{
				TextNode{Content: "Sidebar"},
			}}.Width(10),
			LayerViewNode{
				Layer:      layer,
				ViewWidth:  30,
				ViewHeight: 5,
			},
		}})

		buf := NewBuffer(50, 6)
		tmpl.Execute(buf, 50, 6)

		// Check sidebar rendered
		line0 := buf.GetLine(0)
		t.Logf("Line 0: %q", line0)

		if !strings.Contains(line0, "Sidebar") {
			t.Errorf("Sidebar should render, got: %q", line0)
		}

		// Check layer content rendered
		if !strings.Contains(line0, "Editor line 1") {
			t.Errorf("Layer content should render, got: %q", line0)
		}
	})

	t.Run("HBox with bordered sidebar and LayerView", func(t *testing.T) {
		layer := NewLayer()
		layer.EnsureSize(40, 10)
		for i := 0; i < 10; i++ {
			layer.SetLineString(i, fmt.Sprintf("Line %d of editor content", i+1), DefaultStyle())
		}

		tmpl := Build(HBoxNode{Gap: 1, Children: []any{
			VBoxNode{Children: []any{
				TextNode{Content: "Files", Style: Style{FG: Yellow}},
				HRuleNode{},
				TextNode{Content: "main.go"},
				TextNode{Content: "utils.go"},
			}}.Border(BorderSingle).Width(15),
			VBoxNode{Children: []any{
				TextNode{Content: "Editor", Style: Style{FG: Cyan}},
				HRuleNode{},
				LayerViewNode{
					Layer:      layer,
					ViewWidth:  0, // fill available
					ViewHeight: 8,
				},
			}}.Border(BorderRounded).Grow(1),
		}})

		buf := NewBuffer(60, 12)
		tmpl.Execute(buf, 60, 12)

		// Log all lines for debugging
		for y := 0; y < 12; y++ {
			t.Logf("Line %d: %q", y, buf.GetLine(y))
		}

		// Check sidebar header rendered (inside border, so line 1)
		line1 := buf.GetLine(1)
		if !strings.Contains(line1, "Files") {
			t.Errorf("Sidebar header should render on line 1, got: %q", line1)
		}

		// Check editor header rendered (inside border, so line 1)
		if !strings.Contains(line1, "Editor") {
			t.Errorf("Editor header should render on line 1, got: %q", line1)
		}

		// Check layer content visible (after headers and HRule)
		// Layer content should start around line 3 (border + header + hrule)
		hasLayerContent := false
		for y := 0; y < 12; y++ {
			line := buf.GetLine(y)
			if strings.Contains(line, "Line 1 of editor") {
				hasLayerContent = true
				break
			}
		}
		if !hasLayerContent {
			t.Error("Layer content should be visible somewhere in the output")
		}
	})

	t.Run("LayerView with SelectionList sidebar", func(t *testing.T) {
		// This is the exact pattern that was causing issues
		layer := NewLayer()
		layer.EnsureSize(50, 10)
		layer.SetLineString(0, "Custom rendered content here", DefaultStyle())
		layer.SetLineString(1, "More custom content", DefaultStyle())

		type FileItem struct {
			Name    string
			Display string
		}
		items := []FileItem{
			{Name: "file1.go", Display: "  file1.go"},
			{Name: "file2.go", Display: "  file2.go"},
			{Name: "file3.go", Display: "  file3.go"},
		}
		selected := 0

		tmpl := Build(HBoxNode{Children: []any{
			VBoxNode{Children: []any{
				TextNode{Content: "Browser"},
				&SelectionList{
					Items:    &items,
					Selected: &selected,
					Marker:   "> ",
					Render: func(item *FileItem) any {
						return TextNode{Content: &item.Display}
					},
				},
			}}.Border(BorderSingle).Width(20),
			LayerViewNode{
				Layer:      layer,
				ViewWidth:  40,
				ViewHeight: 6,
			}.Grow(1),
		}})

		buf := NewBuffer(70, 10)
		tmpl.Execute(buf, 70, 10)

		for y := 0; y < 10; y++ {
			t.Logf("Line %d: %q", y, buf.GetLine(y))
		}

		// Browser header is inside the border on line 1
		line1 := buf.GetLine(1)
		if !strings.Contains(line1, "Browser") {
			t.Errorf("Browser header should render on line 1, got: %q", line1)
		}

		// Check layer content - should be on line 0 (to the right of the border)
		line0 := buf.GetLine(0)
		if !strings.Contains(line0, "Custom rendered") {
			t.Errorf("Layer content should be visible on line 0, got: %q", line0)
		}
	})

	t.Run("toggle sidebar visibility with If", func(t *testing.T) {
		layer := NewLayer()
		layer.EnsureSize(50, 5)
		layer.SetLineString(0, "Editor content", DefaultStyle())

		sidebarVisible := true

		tmpl := Build(HBoxNode{Gap: 1, Children: []any{
			// Sidebar - conditionally rendered
			If(&sidebarVisible).Eq(true).Then(
				VBoxNode{Children: []any{
					TextNode{Content: "Sidebar"},
				}}.Border(BorderSingle).Width(15),
			),
			// Editor always visible
			LayerViewNode{
				Layer:      layer,
				ViewWidth:  40,
				ViewHeight: 5,
			}.Grow(1),
		}})

		buf := NewBuffer(60, 6)

		// With sidebar visible
		tmpl.Execute(buf, 60, 6)
		for y := 0; y < 6; y++ {
			t.Logf("Sidebar visible - Line %d: %q", y, buf.GetLine(y))
		}

		// Sidebar border should be on line 0, "Sidebar" text inside on line 1
		line0 := buf.GetLine(0)
		line1 := buf.GetLine(1)
		if !strings.Contains(line0, "┌") {
			t.Errorf("Sidebar border should be visible on line 0, got: %q", line0)
		}
		if !strings.Contains(line1, "Sidebar") {
			t.Errorf("Sidebar text should be inside border on line 1, got: %q", line1)
		}
		// Editor content should be visible
		if !strings.Contains(line0, "Editor content") {
			t.Errorf("Editor content should be visible on line 0, got: %q", line0)
		}

		// Toggle sidebar off
		sidebarVisible = false
		buf.Clear()
		tmpl.Execute(buf, 60, 6)
		for y := 0; y < 6; y++ {
			t.Logf("Sidebar hidden - Line %d: %q", y, buf.GetLine(y))
		}

		line0 = buf.GetLine(0)
		line1 = buf.GetLine(1)
		// No sidebar border should be visible
		if strings.Contains(line0, "┌") || strings.Contains(line1, "Sidebar") {
			t.Errorf("Sidebar should NOT be visible when hidden, line0: %q, line1: %q", line0, line1)
		}
		// Editor should still be visible
		if !strings.Contains(line0, "Editor content") {
			t.Errorf("Editor content should be visible on line 0 when sidebar hidden, got: %q", line0)
		}
	})

	t.Run("HBox with If conditional and Grow sibling - the actual bug", func(t *testing.T) {
		// This test replicates the exact issue from nestdemo case 11
		layer := NewLayer()
		layer.EnsureSize(80, 20)
		for y := 0; y < 20; y++ {
			layer.SetLineString(y, fmt.Sprintf("Line %d of editor", y+1), DefaultStyle())
		}

		sidebarVisible := true

		type FileItem struct {
			Display string
		}
		items := []FileItem{
			{Display: "file1.go"},
			{Display: "file2.go"},
		}
		selected := 0

		sidebarList := &SelectionList{
			Items:    &items,
			Selected: &selected,
			Marker:   "> ",
			Render: func(item *FileItem) any {
				return TextNode{Content: &item.Display}
			},
		}

		tmpl := Build(VBoxNode{Children: []any{
			TextNode{Content: "Header"},
			SpacerNode{Height: 1},
			HBoxNode{Gap: 1, Children: []any{
				// Sidebar with If conditional - THIS IS THE KEY DIFFERENCE
				If(&sidebarVisible).Eq(true).Then(
					VBoxNode{Children: []any{
						TextNode{Content: "Files"},
						HRuleNode{},
						sidebarList,
					}}.Border(BorderSingle).Width(25),
				),
				// Editor with Grow(1)
				VBoxNode{Children: []any{
					TextNode{Content: "Editor"},
					HRuleNode{},
					LayerViewNode{
						Layer:      layer,
						ViewWidth:  0,
						ViewHeight: 10,
					},
				}}.Border(BorderRounded).Grow(1),
			}},
		}})

		buf := NewBuffer(100, 20)
		tmpl.Execute(buf, 100, 20)

		t.Log("Full output with If conditional:")
		for y := 0; y < 15; y++ {
			t.Logf("Line %d: %q", y, buf.GetLine(y))
		}

		// Check sidebar is visible
		foundSidebar := false
		foundEditor := false
		for y := 0; y < 15; y++ {
			line := buf.GetLine(y)
			if strings.Contains(line, "Files") {
				foundSidebar = true
			}
			if strings.Contains(line, "Editor") {
				foundEditor = true
			}
		}

		if !foundSidebar {
			t.Error("Sidebar should be visible")
		}
		if !foundEditor {
			t.Error("CRITICAL: Editor is not visible - this is the bug!")
		}

		// Also check that editor content is visible
		foundContent := false
		for y := 0; y < 15; y++ {
			if strings.Contains(buf.GetLine(y), "Line 1 of editor") {
				foundContent = true
				break
			}
		}
		if !foundContent {
			t.Error("Editor content (Line 1 of editor) should be visible")
		}
	})

	t.Run("If with Grow content should flex like content without If", func(t *testing.T) {
		// Test that If is truly transparent - Grow inside If should work same as Grow without If

		// WITHOUT If wrapper
		tmplWithout := Build(HBoxNode{Children: []any{
			VBoxNode{Children: []any{TextNode{Content: "Left"}}}.Border(BorderSingle).Width(20),
			VBoxNode{Children: []any{TextNode{Content: "Right"}}}.Border(BorderSingle).Grow(1),
		}})

		bufWithout := NewBuffer(60, 5)
		tmplWithout.Execute(bufWithout, 60, 5)

		// WITH If wrapper (condition true)
		visible := true
		tmplWith := Build(HBoxNode{Children: []any{
			If(&visible).Eq(true).Then(
				VBoxNode{Children: []any{TextNode{Content: "Left"}}}.Border(BorderSingle).Width(20),
			),
			VBoxNode{Children: []any{TextNode{Content: "Right"}}}.Border(BorderSingle).Grow(1),
		}})

		bufWith := NewBuffer(60, 5)
		tmplWith.Execute(bufWith, 60, 5)

		t.Log("WITHOUT If wrapper:")
		for y := 0; y < 5; y++ {
			t.Logf("  Line %d: %q", y, bufWithout.GetLine(y))
		}
		t.Log("WITH If wrapper:")
		for y := 0; y < 5; y++ {
			t.Logf("  Line %d: %q", y, bufWith.GetLine(y))
		}

		// Both should render identically
		for y := 0; y < 5; y++ {
			if bufWithout.GetLine(y) != bufWith.GetLine(y) {
				t.Errorf("Line %d differs!\n  without: %q\n  with:    %q",
					y, bufWithout.GetLine(y), bufWith.GetLine(y))
			}
		}
	})

	t.Run("If with Grow content - both sides flex", func(t *testing.T) {
		// Both sides have Grow - should split evenly
		visible := true
		tmpl := Build(HBoxNode{Children: []any{
			If(&visible).Eq(true).Then(
				VBoxNode{Children: []any{TextNode{Content: "A"}}}.Border(BorderSingle).Grow(1),
			),
			VBoxNode{Children: []any{TextNode{Content: "B"}}}.Border(BorderSingle).Grow(1),
		}})

		buf := NewBuffer(60, 5)
		tmpl.Execute(buf, 60, 5)

		t.Log("Both sides Grow(1):")
		for y := 0; y < 5; y++ {
			t.Logf("  Line %d: %q", y, buf.GetLine(y))
		}

		// Both A and B should be visible and roughly equal width
		line1 := buf.GetLine(1)
		if !strings.Contains(line1, "A") {
			t.Error("Left panel (A) should be visible")
		}
		if !strings.Contains(line1, "B") {
			t.Error("Right panel (B) should be visible")
		}

		// Check that both have similar width (each should be ~30 chars)
		aIdx := strings.Index(line1, "A")
		bIdx := strings.Index(line1, "B")
		if aIdx < 0 || bIdx < 0 {
			t.Fatal("Could not find A and B in output")
		}
		// B should be somewhere in the right half (after position 20)
		// and both should have roughly equal space
		if bIdx < 20 {
			t.Errorf("B should be in right half, got pos %d", bIdx)
		}
		// The panels should be roughly equal - B position should be around half
		// Allow some tolerance for border characters
		if bIdx < 25 || bIdx > 40 {
			t.Logf("Note: B at position %d (expected ~30, acceptable range 25-40)", bIdx)
		}
	})
}

func TestGapWithInvisibleIf(t *testing.T) {
	// Test: gaps should only appear between visible children
	visible := true

	tmpl := Build(HBoxNode{Gap: 2, Children: []any{
		TextNode{Content: "A"},
		If(&visible).Eq(true).Then(TextNode{Content: "B"}),
		TextNode{Content: "C"},
	}})

	buf := NewBuffer(20, 1)

	// Visible: A [gap=2] B [gap=2] C = "A  B  C"
	tmpl.Execute(buf, 20, 1)
	line := buf.GetLine(0)
	t.Logf("visible=true:  %q", line)

	// Should be "A  B  C" (A at 0, B at 3, C at 6)
	if len(line) < 7 || line[0] != 'A' || line[3] != 'B' || line[6] != 'C' {
		t.Errorf("visible=true layout wrong: %q (expected A at 0, B at 3, C at 6)", line)
	}

	// Now hide B
	visible = false
	buf.Clear()
	tmpl.Execute(buf, 20, 1)
	line = buf.GetLine(0)
	t.Logf("visible=false: %q", line)

	// Should be "A  C" (A at 0, C at 3) - only ONE gap
	if len(line) < 4 || line[0] != 'A' || line[3] != 'C' {
		t.Errorf("visible=false layout wrong: %q (expected A at 0, C at 3)", line)
	}
}

func TestGapWithMultipleIfsAtEnd(t *testing.T) {
	// 3 non-If children followed by 2 If children
	// Tests that width distribution correctly handles Ifs at the end
	if1Visible := true
	if2Visible := true

	tmpl := Build(HBoxNode{Gap: 1, Children: []any{
		TextNode{Content: "A"},
		TextNode{Content: "B"},
		TextNode{Content: "C"},
		If(&if1Visible).Eq(true).Then(TextNode{Content: "D"}),
		If(&if2Visible).Eq(true).Then(TextNode{Content: "E"}),
	}})

	buf := NewBuffer(20, 1)

	// All visible: A B C D E (with gap=1)
	// Positions: A=0, B=2, C=4, D=6, E=8
	tmpl.Execute(buf, 20, 1)
	line := buf.GetLine(0)
	t.Logf("all visible: %q", line)

	expected := "A B C D E"
	if !strings.HasPrefix(line, expected) {
		t.Errorf("all visible: got %q, want prefix %q", line, expected)
	}

	// Hide first If (D)
	if1Visible = false
	buf.Clear()
	tmpl.Execute(buf, 20, 1)
	line = buf.GetLine(0)
	t.Logf("if1 hidden: %q", line)

	expected = "A B C E"
	if !strings.HasPrefix(line, expected) {
		t.Errorf("if1 hidden: got %q, want prefix %q", line, expected)
	}

	// Hide second If too (E)
	if2Visible = false
	buf.Clear()
	tmpl.Execute(buf, 20, 1)
	line = buf.GetLine(0)
	t.Logf("both hidden: %q", line)

	expected = "A B C"
	if !strings.HasPrefix(line, expected) {
		t.Errorf("both hidden: got %q, want prefix %q", line, expected)
	}

	// Only second If visible
	if1Visible = false
	if2Visible = true
	buf.Clear()
	tmpl.Execute(buf, 20, 1)
	line = buf.GetLine(0)
	t.Logf("only if2 visible: %q", line)

	expected = "A B C E"
	if !strings.HasPrefix(line, expected) {
		t.Errorf("only if2 visible: got %q, want prefix %q", line, expected)
	}
}

func TestSwitchWithHBoxChildren(t *testing.T) {
	// Mimics Demo 1: Switch containing HBox with bordered VBoxes
	// Tests that re-rendering produces identical output
	currentDemo := 0

	// First test: just Switch with simple Text - only child (string type like working test)
	t.Run("switch string type", func(t *testing.T) {
		tab := "home"
		tmpl := Build(VBoxNode{Children: []any{
			Switch(&tab).
				Case("home", TextNode{Content: "HOME_CONTENT"}).
				Case("settings", TextNode{Content: "SETTINGS_CONTENT"}).
				Default(TextNode{Content: "DEFAULT_CONTENT"}),
		}})

		buf := NewBuffer(60, 5)
		tmpl.Execute(buf, 60, 5)
		line0 := buf.GetLine(0)
		t.Logf("String type - Line 0: %q", line0)
		if !strings.HasPrefix(line0, "HOME_CONTENT") {
			t.Errorf("expected 'HOME_CONTENT', got %q", line0)
		}
	})

	// Second test: int type (my original test)
	t.Run("switch int type", func(t *testing.T) {
		tmpl := Build(VBoxNode{Children: []any{
			Switch(&currentDemo).
				Case(0, TextNode{Content: "Demo 0 content"}).
				Case(1, TextNode{Content: "Demo 1 content"}).
				Default(TextNode{Content: "Default content"}),
		}})

		buf := NewBuffer(60, 5)
		tmpl.Execute(buf, 60, 5)
		line0 := buf.GetLine(0)
		t.Logf("Int type - Line 0: %q", line0)
		if !strings.HasPrefix(line0, "Demo 0 content") {
			t.Errorf("expected 'Demo 0 content', got %q", line0)
		}
	})

	// Second test: Switch with Header before it
	t.Run("switch with header before", func(t *testing.T) {
		tmpl := Build(VBoxNode{Children: []any{
			TextNode{Content: "Header"},
			Switch(&currentDemo).
				Case(0, TextNode{Content: "Demo 0 content"}).
				Case(1, TextNode{Content: "Demo 1 content"}).
				Default(TextNode{Content: "Default"}),
		}})

		buf := NewBuffer(60, 5)
		tmpl.Execute(buf, 60, 5)
		line0 := buf.GetLine(0)
		line1 := buf.GetLine(1)
		t.Logf("Line 0: %q", line0)
		t.Logf("Line 1: %q", line1)
		if !strings.HasPrefix(line0, "Header") {
			t.Errorf("expected Header, got %q", line0)
		}
		if !strings.HasPrefix(line1, "Demo 0 content") {
			t.Errorf("expected 'Demo 0 content', got %q", line1)
		}
	})

	// Second test: Switch with HBox
	t.Run("switch with HBox", func(t *testing.T) {
		tmpl := Build(VBoxNode{Children: []any{
			TextNode{Content: "Header"},
			Switch(&currentDemo).
				Case(0, HBoxNode{Gap: 1, Children: []any{
					TextNode{Content: "A"},
					TextNode{Content: "B"},
					TextNode{Content: "C"},
				}}).
				Case(1, TextNode{Content: "Demo 1"}).
				Default(TextNode{Content: "Default"}),
		}})

		buf := NewBuffer(60, 5)
		tmpl.Execute(buf, 60, 5)
		line0 := buf.GetLine(0)
		line1 := buf.GetLine(1)
		t.Logf("Line 0: %q", line0)
		t.Logf("Line 1: %q", line1)
		if !strings.HasPrefix(line1, "A B C") {
			t.Errorf("expected 'A B C', got %q", line1)
		}
	})

	// Third test: Switch with bordered VBox
	t.Run("switch with bordered VBox", func(t *testing.T) {
		tmpl := Build(VBoxNode{Children: []any{
			TextNode{Content: "Header"},
			Switch(&currentDemo).
				Case(0, VBoxNode{Children: []any{
					TextNode{Content: "Col A"},
					TextNode{Content: "A1"},
				}}.Border(BorderSingle)).
				Case(1, TextNode{Content: "Demo 1"}).
				Default(TextNode{Content: "Default"}),
		}})

		buf := NewBuffer(60, 10)
		tmpl.Execute(buf, 60, 10)
		for i := 0; i < 5; i++ {
			t.Logf("Line %d: %q", i, buf.GetLine(i))
		}
	})

	// Full test: Switch with HBox containing bordered VBoxes
	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		Switch(&currentDemo).
			Case(0,
				HBoxNode{Gap: 3, Children: []any{
					VBoxNode{Children: []any{
						TextNode{Content: "Col A"},
						TextNode{Content: "A1"},
						TextNode{Content: "A2"},
					}}.Border(BorderSingle),
					VBoxNode{Children: []any{
						TextNode{Content: "Col B"},
						TextNode{Content: "B1"},
					}}.Border(BorderSingle),
					VBoxNode{Children: []any{
						TextNode{Content: "Col C"},
						TextNode{Content: "C1"},
					}}.Border(BorderSingle),
				}}).
			Case(1, TextNode{Content: "Demo 2"}).
			Default(TextNode{Content: "Unknown demo"}),
	}})

	buf := NewBuffer(60, 10)

	// First render
	tmpl.Execute(buf, 60, 10)
	firstRender := make([]string, 10)
	for i := 0; i < 10; i++ {
		firstRender[i] = buf.GetLine(i)
	}
	t.Logf("First render:")
	for i, line := range firstRender {
		t.Logf("  Line %d: %q", i, line)
	}

	// Second render (simulating key press - same demo)
	buf.Clear()
	tmpl.Execute(buf, 60, 10)
	secondRender := make([]string, 10)
	for i := 0; i < 10; i++ {
		secondRender[i] = buf.GetLine(i)
	}
	t.Logf("Second render:")
	for i, line := range secondRender {
		t.Logf("  Line %d: %q", i, line)
	}

	// Compare renders
	for i := 0; i < 10; i++ {
		if firstRender[i] != secondRender[i] {
			t.Errorf("Line %d differs between renders:\n  First:  %q\n  Second: %q", i, firstRender[i], secondRender[i])
		}
	}
}

// TestIfWithFixedWidthSidebar verifies that an If containing a fixed-width
// VBox (wrapper pattern) doesn't cause the sibling flex component to disappear.
// This mimics the sidebar pattern: HBoxNode{If{sidebar}, LayerView.Grow(1)}
func TestIfWithFixedWidthSidebar(t *testing.T) {
	sidebarVisible := true
	const sidebarWidth = 10

	// Build: HBox containing conditional sidebar + main content
	// The sidebar is wrapped in a VBox (like wed's structure)
	tmpl := Build(HBoxNode{Children: []any{
		// Conditional sidebar wrapper
		If(&sidebarVisible).Eq(true).Then(
			VBoxNode{Children: []any{
				VBoxNode{Children: []any{
					TextNode{Content: "Sidebar"},
				}}.Width(sidebarWidth),
			}},
		),
		// Main content with Grow(1)
		VBoxNode{Children: []any{
			TextNode{Content: "MAIN_CONTENT"},
		}}.Grow(1),
	}})

	buf := NewBuffer(50, 3)

	// First render with sidebar visible
	tmpl.Execute(buf, 50, 3)
	line := buf.GetLine(0)
	t.Logf("sidebar visible: %q", line)

	// Sidebar should take 10 chars, main content should have the rest
	if !strings.HasPrefix(line, "Sidebar") {
		t.Errorf("expected sidebar at start, got %q", line)
	}
	if !strings.Contains(line, "MAIN_CONTENT") {
		t.Errorf("main content missing! got %q", line)
	}

	// Hide sidebar
	sidebarVisible = false
	buf.Clear()
	tmpl.Execute(buf, 50, 3)
	line = buf.GetLine(0)
	t.Logf("sidebar hidden: %q", line)

	// Main content should start at position 0
	if !strings.HasPrefix(line, "MAIN_CONTENT") {
		t.Errorf("expected main content at start when sidebar hidden, got %q", line)
	}

	// Show sidebar again
	sidebarVisible = true
	buf.Clear()
	tmpl.Execute(buf, 50, 3)
	line = buf.GetLine(0)
	t.Logf("sidebar visible again: %q", line)

	if !strings.HasPrefix(line, "Sidebar") {
		t.Errorf("expected sidebar at start again, got %q", line)
	}
	if !strings.Contains(line, "MAIN_CONTENT") {
		t.Errorf("main content missing after re-show! got %q", line)
	}
}

// TestStyleInheritance verifies that CascadeStyle propagates to children.
func TestStyleInheritance(t *testing.T) {
	baseStyle := Style{FG: Red, BG: Blue}
	accentStyle := Style{FG: Green}

	t.Run("children inherit parent style", func(t *testing.T) {
		tmpl := Build(VBoxNode{
			CascadeStyle: &baseStyle,
			Children: []any{
				TextNode{Content: "Inherited"},
				TextNode{Content: "Also inherited"},
			},
		})

		buf := NewBuffer(20, 3)
		tmpl.Execute(buf, 20, 3)

		// Check that text was rendered with inherited style
		cell := buf.Get(0, 0)
		if cell.Style.FG != Red {
			t.Errorf("expected FG Red, got %v", cell.Style.FG)
		}
		if cell.Style.BG != Blue {
			t.Errorf("expected BG Blue, got %v", cell.Style.BG)
		}
	})

	t.Run("explicit style overrides inherited", func(t *testing.T) {
		tmpl := Build(VBoxNode{
			CascadeStyle: &baseStyle,
			Children: []any{
				TextNode{Content: "Override", Style: accentStyle},
			},
		})

		buf := NewBuffer(20, 3)
		tmpl.Execute(buf, 20, 3)

		cell := buf.Get(0, 0)
		if cell.Style.FG != Green {
			t.Errorf("expected FG Green (override), got %v", cell.Style.FG)
		}
	})

	t.Run("nested containers can override inherited style", func(t *testing.T) {
		nestedStyle := Style{FG: Yellow}

		tmpl := Build(VBoxNode{
			CascadeStyle: &baseStyle,
			Children: []any{
				TextNode{Content: "Uses base"},
				VBoxNode{
					CascadeStyle: &nestedStyle,
					Children: []any{
						TextNode{Content: "Uses nested"},
					},
				},
				TextNode{Content: "Back to base"},
			},
		})

		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		// Line 0: base style
		cell0 := buf.Get(0, 0)
		if cell0.Style.FG != Red {
			t.Errorf("line 0: expected FG Red, got %v", cell0.Style.FG)
		}

		// Line 1: nested style
		cell1 := buf.Get(0, 1)
		if cell1.Style.FG != Yellow {
			t.Errorf("line 1: expected FG Yellow, got %v", cell1.Style.FG)
		}

		// Line 2: back to base style
		cell2 := buf.Get(0, 2)
		if cell2.Style.FG != Red {
			t.Errorf("line 2: expected FG Red (back to base), got %v", cell2.Style.FG)
		}
	})

	t.Run("style inheritance works through If", func(t *testing.T) {
		visible := true

		tmpl := Build(VBoxNode{
			CascadeStyle: &baseStyle,
			Children: []any{
				If(&visible).Eq(true).Then(
					TextNode{Content: "Conditional"},
				),
			},
		})

		buf := NewBuffer(20, 3)
		tmpl.Execute(buf, 20, 3)

		cell := buf.Get(0, 0)
		if cell.Style.FG != Red {
			t.Errorf("expected FG Red through If, got %v", cell.Style.FG)
		}
	})

	t.Run("dynamic theme switching", func(t *testing.T) {
		theme := Style{FG: Cyan}

		tmpl := Build(VBoxNode{
			CascadeStyle: &theme,
			Children: []any{
				TextNode{Content: "Themed"},
			},
		})

		buf := NewBuffer(20, 3)

		// First render
		tmpl.Execute(buf, 20, 3)
		cell := buf.Get(0, 0)
		if cell.Style.FG != Cyan {
			t.Errorf("first render: expected FG Cyan, got %v", cell.Style.FG)
		}

		// Change theme
		theme = Style{FG: Magenta}
		buf.Clear()
		tmpl.Execute(buf, 20, 3)

		cell = buf.Get(0, 0)
		if cell.Style.FG != Magenta {
			t.Errorf("after theme change: expected FG Magenta, got %v", cell.Style.FG)
		}
	})
}

// TestContainerFill verifies that containers fill their area when CascadeStyle has Fill color.
func TestContainerFill(t *testing.T) {
	t.Run("container fills area with Fill color", func(t *testing.T) {
		fillStyle := Style{Fill: Red}

		tmpl := Build(VBoxNode{
			CascadeStyle: &fillStyle,
			Children: []any{
				TextNode{Content: "Hi"},
			},
		})

		buf := NewBuffer(10, 3)
		tmpl.Execute(buf, 10, 3)

		// Check that empty cells have Fill color as BG
		// Cell at (5, 0) should have the fill color
		cell := buf.Get(5, 0)
		if cell.Style.BG != Red {
			t.Errorf("expected BG Red (fill), got %v", cell.Style.BG)
		}

		// Cell at (0, 2) should also have fill (empty row)
		cell2 := buf.Get(0, 2)
		if cell2.Style.BG != Red {
			t.Errorf("expected BG Red on empty row, got %v", cell2.Style.BG)
		}
	})

	t.Run("nested container with different fill", func(t *testing.T) {
		outerFill := Style{Fill: Blue}
		innerFill := Style{Fill: Green}

		tmpl := Build(VBoxNode{
			CascadeStyle: &outerFill,
			Children: []any{
				TextNode{Content: "Outer"},
				HBoxNode{
					CascadeStyle: &innerFill,
					Children: []any{
						TextNode{Content: "In"},
					},
				}.Width(5).Height(1),
			},
		})

		buf := NewBuffer(10, 3)
		tmpl.Execute(buf, 10, 3)

		// Outer area should have blue fill
		outerCell := buf.Get(8, 0)
		if outerCell.Style.BG != Blue {
			t.Errorf("outer area: expected BG Blue, got %v", outerCell.Style.BG)
		}

		// Inner area should have green fill
		innerCell := buf.Get(4, 1) // within the HBox
		if innerCell.Style.BG != Green {
			t.Errorf("inner area: expected BG Green, got %v", innerCell.Style.BG)
		}
	})

	t.Run("fill does not affect containers without CascadeStyle", func(t *testing.T) {
		buf := NewBuffer(10, 3)

		tmpl := Build(VBoxNode{
			Children: []any{
				TextNode{Content: "No fill"},
			},
		})
		tmpl.Execute(buf, 10, 3)

		// Should have default (no fill)
		cell := buf.Get(9, 0)
		if cell.Style.BG.Mode != ColorDefault {
			t.Errorf("expected default BG, got %v", cell.Style.BG)
		}
	})

	t.Run("fill cascades when nested container overrides only FG", func(t *testing.T) {
		// Root has Fill, nested container only overrides FG
		rootStyle := Style{FG: White, Fill: Blue}
		nestedStyle := Style{FG: Yellow} // no Fill - should inherit parent's Fill

		tmpl := Build(VBoxNode{
			CascadeStyle: &rootStyle,
			Children: []any{
				TextNode{Content: "Root"},
				VBoxNode{
					CascadeStyle: &nestedStyle,
					Children: []any{
						TextNode{Content: "Nested"},
					},
				},
			},
		})

		buf := NewBuffer(10, 3)
		tmpl.Execute(buf, 10, 3)

		// Root text should have Blue BG (from Fill)
		rootCell := buf.Get(0, 0)
		if rootCell.Style.BG != Blue {
			t.Errorf("root text: expected BG Blue, got %v", rootCell.Style.BG)
		}

		// Nested text should ALSO have Blue BG (Fill cascades)
		nestedCell := buf.Get(0, 1)
		if nestedCell.Style.BG != Blue {
			t.Errorf("nested text: expected BG Blue (cascaded), got %v", nestedCell.Style.BG)
		}

		// But nested text should have Yellow FG (from its CascadeStyle)
		if nestedCell.Style.FG != Yellow {
			t.Errorf("nested text: expected FG Yellow, got %v", nestedCell.Style.FG)
		}
	})
}

// TestTextTransform verifies text transforms are applied via CascadeStyle.
func TestTextTransform(t *testing.T) {
	t.Run("uppercase transform", func(t *testing.T) {
		style := Style{Transform: TransformUppercase}
		tmpl := Build(VBoxNode{
			CascadeStyle: &style,
			Children: []any{
				TextNode{Content: "hello world"},
			},
		})

		buf := NewBuffer(20, 1)
		tmpl.Execute(buf, 20, 1)

		line := buf.GetLine(0)
		if !strings.Contains(line, "HELLO WORLD") {
			t.Errorf("expected uppercase, got: %s", line)
		}
	})

	t.Run("lowercase transform", func(t *testing.T) {
		style := Style{Transform: TransformLowercase}
		tmpl := Build(VBoxNode{
			CascadeStyle: &style,
			Children: []any{
				TextNode{Content: "HELLO WORLD"},
			},
		})

		buf := NewBuffer(20, 1)
		tmpl.Execute(buf, 20, 1)

		line := buf.GetLine(0)
		if !strings.Contains(line, "hello world") {
			t.Errorf("expected lowercase, got: %s", line)
		}
	})

	t.Run("transform cascades to children", func(t *testing.T) {
		style := Style{Transform: TransformUppercase}
		tmpl := Build(VBoxNode{
			CascadeStyle: &style,
			Children: []any{
				TextNode{Content: "parent"},
				VBoxNode{
					Children: []any{
						TextNode{Content: "child"},
					},
				},
			},
		})

		buf := NewBuffer(20, 2)
		tmpl.Execute(buf, 20, 2)

		line0 := buf.GetLine(0)
		line1 := buf.GetLine(1)
		if !strings.Contains(line0, "PARENT") {
			t.Errorf("expected PARENT, got: %s", line0)
		}
		if !strings.Contains(line1, "CHILD") {
			t.Errorf("expected CHILD (cascaded), got: %s", line1)
		}
	})
}

// TestAttrInheritance verifies attributes cascade via CascadeStyle.
func TestAttrInheritance(t *testing.T) {
	t.Run("attr merges with child style", func(t *testing.T) {
		parentStyle := Style{Attr: AttrBold}
		childStyle := Style{FG: Red} // has FG but not Attr

		tmpl := Build(VBoxNode{
			CascadeStyle: &parentStyle,
			Children: []any{
				TextNode{Content: "X", Style: childStyle},
			},
		})

		buf := NewBuffer(5, 1)
		tmpl.Execute(buf, 5, 1)

		cell := buf.Get(0, 0)
		if !cell.Style.Attr.Has(AttrBold) {
			t.Errorf("expected Bold attr inherited, got: %v", cell.Style.Attr)
		}
		if cell.Style.FG != Red {
			t.Errorf("expected FG Red from child, got: %v", cell.Style.FG)
		}
	})
}

// ============================================================================
// Functional API Tests
// ============================================================================

func TestFunctionalAPI_VBox(t *testing.T) {
	// Test the new V() functional API
	tmpl := Build(VBox(
		Text("Line 1"),
		Text("Line 2"),
		Text("Line 3"),
	))

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

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

func TestFunctionalAPI_HBox(t *testing.T) {
	// Test the new H() functional API
	tmpl := Build(HBox(
		Text("A"),
		Text("B"),
		Text("C"),
	))

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	line := buf.GetLine(0)
	if line != "ABC" {
		t.Errorf("line 0: got %q, want %q", line, "ABC")
	}
}

func TestFunctionalAPI_HBoxWithGap(t *testing.T) {
	// Test HBox.Gap()
	tmpl := Build(HBox.Gap(2)(
		Text("A"),
		Text("B"),
	))

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	line := buf.GetLine(0)
	if line != "A  B" {
		t.Errorf("line 0: got %q, want %q", line, "A  B")
	}
}

func TestFunctionalAPI_VBoxWithStyle(t *testing.T) {
	// Test VBox.CascadeStyle() for inheritance
	style := Style{FG: Red}
	tmpl := Build(VBox.CascadeStyle(&style)(
		Text("Red text"),
	))

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	cell := buf.Get(0, 0)
	if cell.Style.FG != Red {
		t.Errorf("expected FG Red, got: %v", cell.Style.FG)
	}
}

func TestFunctionalAPI_Nested(t *testing.T) {
	// Test nested functional containers
	tmpl := Build(VBox(
		Text("Top"),
		HBox(
			Text("Left"),
			Text("Right"),
		),
		Text("Bottom"),
	))

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "Top" {
		t.Errorf("line 0: got %q, want %q", got, "Top")
	}
	if got := buf.GetLine(1); got != "LeftRight" {
		t.Errorf("line 1: got %q, want %q", got, "LeftRight")
	}
	if got := buf.GetLine(2); got != "Bottom" {
		t.Errorf("line 2: got %q, want %q", got, "Bottom")
	}
}

func TestFunctionalAPI_TextStyling(t *testing.T) {
	// Test text styling methods
	tmpl := Build(VBox(
		Text("Bold").Bold(),
		Text("Red").FG(Red),
	))

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// Check bold
	cell := buf.Get(0, 0)
	if !cell.Style.Attr.Has(AttrBold) {
		t.Errorf("expected Bold attr, got: %v", cell.Style.Attr)
	}

	// Check color
	cell = buf.Get(0, 1)
	if cell.Style.FG != Red {
		t.Errorf("expected FG Red, got: %v", cell.Style.FG)
	}
}

func TestFunctionalAPI_Spacer(t *testing.T) {
	// Test spacer functions
	tmpl := Build(VBox(
		Text("Line 1"),
		SpaceH(2),
		Text("Line 4"),
	))

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "Line 1" {
		t.Errorf("line 0: got %q, want %q", got, "Line 1")
	}
	if got := buf.GetLine(3); got != "Line 4" {
		t.Errorf("line 3: got %q, want %q", got, "Line 4")
	}
}

func TestFunctionalAPI_HRule(t *testing.T) {
	// Test HRule() function
	tmpl := Build(VBox(
		Text("Above"),
		HRule(),
		Text("Below"),
	))

	buf := NewBuffer(10, 10)
	tmpl.Execute(buf, 10, 10)

	if got := buf.GetLine(0); got != "Above" {
		t.Errorf("line 0: got %q, want %q", got, "Above")
	}
	// HR should be on line 1
	line1 := buf.GetLine(1)
	if !strings.Contains(line1, "─") {
		t.Errorf("line 1: expected hrule chars, got %q", line1)
	}
}

func TestFunctionalAPI_Conditional(t *testing.T) {
	// Test If().Then()
	show := true
	tmpl := Build(VBox(
		If(&show).Then(Text("Visible")),
	))

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "Visible" {
		t.Errorf("line 0: got %q, want %q", got, "Visible")
	}

	// Now test when false
	show = false
	buf = NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); strings.TrimSpace(got) != "" {
		t.Errorf("line 0: expected empty when hidden, got %q", got)
	}
}

func TestFunctionalAPI_ConditionalWithElse(t *testing.T) {
	// Test If().Then().Else()
	show := false
	tmpl := Build(VBox(
		If(&show).Then(Text("Yes")).Else(Text("No")),
	))

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	if got := buf.GetLine(0); got != "No" {
		t.Errorf("line 0: got %q, want %q", got, "No")
	}
}

func TestFunctionalAPI_VBoxWithBorder(t *testing.T) {
	// Test VBox.Border()
	tmpl := Build(VBox.Border(BorderSingle)(
		Text("Inside"),
	))

	buf := NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	// Check for border characters
	line0 := buf.GetLine(0)
	if !strings.Contains(line0, "┌") || !strings.Contains(line0, "┐") {
		t.Errorf("line 0: expected border corners, got %q", line0)
	}
}

func TestFunctionalAPI_ChainedMethods(t *testing.T) {
	// Test chaining multiple modifiers
	style := Style{FG: Blue}
	tmpl := Build(VBox.CascadeStyle(&style).Gap(1).Border(BorderSingle)(
		Text("Line 1"),
		Text("Line 2"),
	))

	buf := NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	// Should compile and render without error
	line0 := buf.GetLine(0)
	if !strings.Contains(line0, "┌") {
		t.Errorf("expected border, got %q", line0)
	}
}

// TestWidgetReceivesAvailWidth tests that Widget's measure function receives
// the correct available width from VBox parent.
func TestWidgetReceivesAvailWidth(t *testing.T) {
	t.Run("widget fills available width in VBox", func(t *testing.T) {
		// Track what width the measure function receives
		var receivedWidth int16

		widget := Widget(
			func(availW int16) (w, h int16) {
				receivedWidth = availW
				return availW, 1 // fill width, 1 line tall
			},
			func(buf *Buffer, x, y, w, h int16) {
				// Draw a bar filling the width
				for i := int16(0); i < w; i++ {
					buf.Set(int(x+i), int(y), Cell{Rune: '█', Style: Style{FG: Green}})
				}
			},
		)

		tmpl := Build(VBox(
			Text("Header"),
			widget,
			Text("Footer"),
		))

		buf := NewBuffer(40, 5)
		tmpl.Execute(buf, 40, 5)

		// Widget should receive the full width (40)
		if receivedWidth != 40 {
			t.Errorf("widget measure received availW=%d, want 40", receivedWidth)
		}

		// Check that widget rendered something on line 1
		line1 := buf.GetLine(1)
		if !strings.Contains(line1, "█") {
			t.Errorf("widget should render bars, got %q", line1)
		}

		// Verify widget fills the full width
		count := strings.Count(line1, "█")
		if count != 40 {
			t.Errorf("widget should fill 40 chars, got %d", count)
		}
	})

	t.Run("widget with fixed width in HBox", func(t *testing.T) {
		// Widget that returns fixed width
		widget := Widget(
			func(availW int16) (w, h int16) {
				return 10, 1 // fixed 10 chars wide
			},
			func(buf *Buffer, x, y, w, h int16) {
				for i := int16(0); i < w; i++ {
					buf.Set(int(x+i), int(y), Cell{Rune: 'X', Style: Style{}})
				}
			},
		)

		tmpl := Build(HBox.Gap(1)(
			Text("A"),
			widget,
			Text("B"),
		))

		buf := NewBuffer(30, 1)
		tmpl.Execute(buf, 30, 1)

		line := buf.GetLine(0)
		t.Logf("line: %q", line)

		// Should have A, then 10 X's, then B
		if line[0] != 'A' {
			t.Errorf("expected A at start, got %q", line)
		}
		if !strings.Contains(line, "XXXXXXXXXX") {
			t.Errorf("expected 10 X's, got %q", line)
		}
	})

	t.Run("widget inside bordered VBox", func(t *testing.T) {
		var receivedWidth int16

		widget := Widget(
			func(availW int16) (w, h int16) {
				receivedWidth = availW
				return availW, 1
			},
			func(buf *Buffer, x, y, w, h int16) {
				for i := int16(0); i < w; i++ {
					buf.Set(int(x+i), int(y), Cell{Rune: '-', Style: Style{}})
				}
			},
		)

		tmpl := Build(VBox.Border(BorderSingle)(
			widget,
		))

		buf := NewBuffer(20, 3)
		tmpl.Execute(buf, 20, 3)

		// Widget should receive width minus border (20 - 2 = 18)
		if receivedWidth != 18 {
			t.Errorf("widget measure received availW=%d, want 18", receivedWidth)
		}
	})
}

func TestAutoTable(t *testing.T) {
	type Person struct {
		Name string
		Age  int
		City string
	}

	people := []Person{
		{"Alice", 30, "NYC"},
		{"Bob", 25, "LA"},
		{"Charlie", 35, "Chicago"},
	}

	t.Run("auto columns from struct", func(t *testing.T) {
		tmpl := Build(AutoTable(people))
		buf := NewBuffer(40, 10)
		tmpl.Execute(buf, 40, 10)

		// Header row should have field names
		header := buf.GetLine(0)
		if !containsSubstring(header, "Name") {
			t.Errorf("header missing 'Name': got %q", header)
		}
		if !containsSubstring(header, "Age") {
			t.Errorf("header missing 'Age': got %q", header)
		}
		if !containsSubstring(header, "City") {
			t.Errorf("header missing 'City': got %q", header)
		}

		// Data rows
		row1 := buf.GetLine(1)
		if !containsSubstring(row1, "Alice") {
			t.Errorf("row 1 missing 'Alice': got %q", row1)
		}

		row2 := buf.GetLine(2)
		if !containsSubstring(row2, "Bob") {
			t.Errorf("row 2 missing 'Bob': got %q", row2)
		}
	})

	t.Run("select columns", func(t *testing.T) {
		tmpl := Build(AutoTable(people).Columns("Name", "City"))
		buf := NewBuffer(40, 10)
		tmpl.Execute(buf, 40, 10)

		header := buf.GetLine(0)
		if !containsSubstring(header, "Name") {
			t.Errorf("header missing 'Name': got %q", header)
		}
		if !containsSubstring(header, "City") {
			t.Errorf("header missing 'City': got %q", header)
		}
		// Age should NOT be present
		if containsSubstring(header, "Age") {
			t.Errorf("header should not have 'Age': got %q", header)
		}
	})

	t.Run("custom headers", func(t *testing.T) {
		tmpl := Build(AutoTable(people).Columns("Name", "Age").Headers("Person", "Years"))
		buf := NewBuffer(40, 10)
		tmpl.Execute(buf, 40, 10)

		header := buf.GetLine(0)
		if !containsSubstring(header, "Person") {
			t.Errorf("header missing 'Person': got %q", header)
		}
		if !containsSubstring(header, "Years") {
			t.Errorf("header missing 'Years': got %q", header)
		}
	})

	t.Run("pointer slice", func(t *testing.T) {
		ptrs := []*Person{
			{"Dave", 40, "Boston"},
			{"Eve", 28, "Seattle"},
		}

		tmpl := Build(AutoTable(ptrs))
		buf := NewBuffer(40, 10)
		tmpl.Execute(buf, 40, 10)

		row1 := buf.GetLine(1)
		if !containsSubstring(row1, "Dave") {
			t.Errorf("row 1 missing 'Dave': got %q", row1)
		}
	})
}
