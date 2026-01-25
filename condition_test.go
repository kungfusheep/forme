package tui

import "testing"

func TestConditionEq(t *testing.T) {
	t.Run("If comparable Eq true", func(t *testing.T) {
		val := 5
		cond := If(&val).Eq(5)
		if !cond.evaluate() {
			t.Error("expected condition to be true when val == 5")
		}
	})

	t.Run("If comparable Eq false", func(t *testing.T) {
		val := 5
		cond := If(&val).Eq(10)
		if cond.evaluate() {
			t.Error("expected condition to be false when val != 10")
		}
	})

	t.Run("If comparable Ne", func(t *testing.T) {
		val := 5
		cond := If(&val).Ne(10)
		if !cond.evaluate() {
			t.Error("expected condition to be true when val != 10")
		}
	})
}

func TestOrdCondition(t *testing.T) {
	t.Run("Gt true", func(t *testing.T) {
		val := 10
		cond := IfOrd(&val).Gt(5)
		if !cond.evaluate() {
			t.Error("expected 10 > 5 to be true")
		}
	})

	t.Run("Gt false", func(t *testing.T) {
		val := 3
		cond := IfOrd(&val).Gt(5)
		if cond.evaluate() {
			t.Error("expected 3 > 5 to be false")
		}
	})

	t.Run("Lt true", func(t *testing.T) {
		val := 3
		cond := IfOrd(&val).Lt(5)
		if !cond.evaluate() {
			t.Error("expected 3 < 5 to be true")
		}
	})

	t.Run("Gte", func(t *testing.T) {
		val := 5
		cond := IfOrd(&val).Gte(5)
		if !cond.evaluate() {
			t.Error("expected 5 >= 5 to be true")
		}
	})

	t.Run("Lte", func(t *testing.T) {
		val := 5
		cond := IfOrd(&val).Lte(5)
		if !cond.evaluate() {
			t.Error("expected 5 <= 5 to be true")
		}
	})
}

func TestConditionThenElse(t *testing.T) {
	t.Run("Then branch accessible", func(t *testing.T) {
		val := true
		cond := If(&val).Eq(true).Then("yes").Else("no")
		if cond.getThen() != "yes" {
			t.Error("expected then to be 'yes'")
		}
		if cond.getElse() != "no" {
			t.Error("expected else to be 'no'")
		}
	})

	t.Run("Evaluates dynamically", func(t *testing.T) {
		val := 0
		cond := IfOrd(&val).Eq(0)

		if !cond.evaluate() {
			t.Error("expected true when val == 0")
		}

		val = 1
		if cond.evaluate() {
			t.Error("expected false when val == 1")
		}

		val = 0
		if !cond.evaluate() {
			t.Error("expected true again when val == 0")
		}
	})
}

func TestConditionInSerialTemplate(t *testing.T) {
	t.Run("If renders correct branch", func(t *testing.T) {
		activeLayer := 0

		view := VBoxNode{Children: []any{
			IfOrd(&activeLayer).Eq(0).Then(TextNode{Content: "LAYER0"}).Else(TextNode{Content: "OTHER"}),
			IfOrd(&activeLayer).Eq(1).Then(TextNode{Content: "LAYER1"}).Else(TextNode{Content: "OTHER"}),
		}}

		tmpl := Build(view)
		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		// Check first line has "LAYER0"
		line0 := extractLine(buf, 0, 10)
		if line0 != "LAYER0    " {
			t.Errorf("expected 'LAYER0    ', got %q", line0)
		}

		// Check second line has "OTHER" (since activeLayer != 1)
		line1 := extractLine(buf, 1, 10)
		if line1 != "OTHER     " {
			t.Errorf("expected 'OTHER     ', got %q", line1)
		}

		// Now change activeLayer and re-render
		activeLayer = 1
		buf.Clear()
		tmpl.Execute(buf, 20, 5)

		// Check first line now has "OTHER"
		line0 = extractLine(buf, 0, 10)
		if line0 != "OTHER     " {
			t.Errorf("after change: expected 'OTHER     ', got %q", line0)
		}

		// Check second line now has "LAYER1"
		line1 = extractLine(buf, 1, 10)
		if line1 != "LAYER1    " {
			t.Errorf("after change: expected 'LAYER1    ', got %q", line1)
		}
	})
}

// extractLine returns the text content from a buffer row
func extractLine(buf *Buffer, row, width int) string {
	result := make([]rune, width)
	for x := 0; x < width; x++ {
		result[x] = buf.Get(x, row).Rune
	}
	return string(result)
}

func TestSwitch(t *testing.T) {
	t.Run("Switch string matches case", func(t *testing.T) {
		tab := "home"
		sw := Switch(&tab).
			Case("home", "HOME_VIEW").
			Case("settings", "SETTINGS_VIEW").
			Default("DEFAULT_VIEW")

		if sw.getMatchIndex() != 0 {
			t.Errorf("expected match index 0, got %d", sw.getMatchIndex())
		}
		if sw.evaluateSwitch() != "HOME_VIEW" {
			t.Errorf("expected HOME_VIEW, got %v", sw.evaluateSwitch())
		}
	})

	t.Run("Switch string matches second case", func(t *testing.T) {
		tab := "settings"
		sw := Switch(&tab).
			Case("home", "HOME_VIEW").
			Case("settings", "SETTINGS_VIEW").
			Default("DEFAULT_VIEW")

		if sw.getMatchIndex() != 1 {
			t.Errorf("expected match index 1, got %d", sw.getMatchIndex())
		}
		if sw.evaluateSwitch() != "SETTINGS_VIEW" {
			t.Errorf("expected SETTINGS_VIEW, got %v", sw.evaluateSwitch())
		}
	})

	t.Run("Switch falls through to default", func(t *testing.T) {
		tab := "unknown"
		sw := Switch(&tab).
			Case("home", "HOME_VIEW").
			Case("settings", "SETTINGS_VIEW").
			Default("DEFAULT_VIEW")

		if sw.getMatchIndex() != -1 {
			t.Errorf("expected match index -1, got %d", sw.getMatchIndex())
		}
		if sw.evaluateSwitch() != "DEFAULT_VIEW" {
			t.Errorf("expected DEFAULT_VIEW, got %v", sw.evaluateSwitch())
		}
	})

	t.Run("Switch int type", func(t *testing.T) {
		mode := 2
		sw := Switch(&mode).
			Case(1, "MODE_ONE").
			Case(2, "MODE_TWO").
			Default("MODE_DEFAULT")

		if sw.evaluateSwitch() != "MODE_TWO" {
			t.Errorf("expected MODE_TWO, got %v", sw.evaluateSwitch())
		}
	})

	t.Run("Switch evaluates dynamically", func(t *testing.T) {
		tab := "home"
		sw := Switch(&tab).
			Case("home", "HOME").
			Case("settings", "SETTINGS").
			Default("DEFAULT")

		if sw.evaluateSwitch() != "HOME" {
			t.Errorf("expected HOME, got %v", sw.evaluateSwitch())
		}

		tab = "settings"
		if sw.evaluateSwitch() != "SETTINGS" {
			t.Errorf("expected SETTINGS after change, got %v", sw.evaluateSwitch())
		}

		tab = "other"
		if sw.evaluateSwitch() != "DEFAULT" {
			t.Errorf("expected DEFAULT for unknown, got %v", sw.evaluateSwitch())
		}
	})
}

func TestSwitchInSerialTemplate(t *testing.T) {
	t.Run("Switch renders correct case", func(t *testing.T) {
		tab := "home"

		view := VBoxNode{Children: []any{
			Switch(&tab).
				Case("home", TextNode{Content: "HOME_CONTENT"}).
				Case("settings", TextNode{Content: "SETTINGS_CONTENT"}).
				Default(TextNode{Content: "DEFAULT_CONTENT"}),
		}}

		tmpl := Build(view)
		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		line := extractLine(buf, 0, 15)
		if line != "HOME_CONTENT   " {
			t.Errorf("expected 'HOME_CONTENT   ', got %q", line)
		}

		// Change tab and re-render
		tab = "settings"
		buf.Clear()
		tmpl.Execute(buf, 20, 5)

		line = extractLine(buf, 0, 18)
		if line != "SETTINGS_CONTENT  " {
			t.Errorf("expected 'SETTINGS_CONTENT  ', got %q", line)
		}

		// Change to unknown tab
		tab = "unknown"
		buf.Clear()
		tmpl.Execute(buf, 20, 5)

		line = extractLine(buf, 0, 17)
		if line != "DEFAULT_CONTENT  " {
			t.Errorf("expected 'DEFAULT_CONTENT  ', got %q", line)
		}
	})
}

func TestSelectionList(t *testing.T) {
	type Item struct {
		Name string
	}

	t.Run("renders items with selection marker", func(t *testing.T) {
		items := []Item{{Name: "One"}, {Name: "Two"}, {Name: "Three"}}
		selected := 1

		list := &SelectionList{
			Items:    &items,
			Selected: &selected,
			Render: func(item *Item) any {
				return TextNode{Content: &item.Name}
			},
		}

		view := VBoxNode{Children: []any{list}}
		tmpl := Build(view)
		buf := NewBuffer(20, 10)
		tmpl.Execute(buf, 20, 10)

		// Row 0: "  One" (not selected)
		// Row 1: "> Two" (selected)
		// Row 2: "  Three" (not selected)
		line0 := extractLine(buf, 0, 7)
		line1 := extractLine(buf, 1, 7)
		line2 := extractLine(buf, 2, 9)

		if line0 != "  One  " {
			t.Errorf("row 0: expected '  One  ', got %q", line0)
		}
		if line1 != "> Two  " {
			t.Errorf("row 1: expected '> Two  ', got %q", line1)
		}
		if line2 != "  Three  " {
			t.Errorf("row 2: expected '  Three  ', got %q", line2)
		}
	})

	t.Run("selection changes dynamically", func(t *testing.T) {
		items := []Item{{Name: "A"}, {Name: "B"}}
		selected := 0

		list := &SelectionList{
			Items:    &items,
			Selected: &selected,
			Render: func(item *Item) any {
				return TextNode{Content: &item.Name}
			},
		}

		view := VBoxNode{Children: []any{list}}
		tmpl := Build(view)
		buf := NewBuffer(20, 10)
		tmpl.Execute(buf, 20, 10)

		line0 := extractLine(buf, 0, 4)
		line1 := extractLine(buf, 1, 4)
		if line0 != "> A " {
			t.Errorf("before: row 0 expected '> A ', got %q", line0)
		}
		if line1 != "  B " {
			t.Errorf("before: row 1 expected '  B ', got %q", line1)
		}

		// Change selection
		selected = 1
		buf.Clear()
		tmpl.Execute(buf, 20, 10)

		line0 = extractLine(buf, 0, 4)
		line1 = extractLine(buf, 1, 4)
		if line0 != "  A " {
			t.Errorf("after: row 0 expected '  A ', got %q", line0)
		}
		if line1 != "> B " {
			t.Errorf("after: row 1 expected '> B ', got %q", line1)
		}
	})

	t.Run("custom marker", func(t *testing.T) {
		items := []Item{{Name: "X"}}
		selected := 0

		list := &SelectionList{
			Items:    &items,
			Selected: &selected,
			Marker:   "→ ",
			Render: func(item *Item) any {
				return TextNode{Content: &item.Name}
			},
		}

		view := VBoxNode{Children: []any{list}}
		tmpl := Build(view)
		buf := NewBuffer(20, 10)
		tmpl.Execute(buf, 20, 10)

		// "→ " is 4 bytes but 2 display chars - for now we just check it renders
		line0 := extractLine(buf, 0, 6)
		// The arrow character might take different width, just check it's not "> "
		if line0 == "> X   " {
			t.Errorf("custom marker not applied, got default")
		}
	})

	t.Run("helper methods", func(t *testing.T) {
		items := []Item{{Name: "A"}, {Name: "B"}, {Name: "C"}}
		selected := 0

		list := &SelectionList{
			Items:    &items,
			Selected: &selected,
			Render: func(item *Item) any {
				return TextNode{Content: &item.Name}
			},
		}

		// Need to render once to populate len
		view := VBoxNode{Children: []any{list}}
		tmpl := Build(view)
		buf := NewBuffer(20, 10)
		tmpl.Execute(buf, 20, 10)

		// Test Down
		list.Down(nil)
		if selected != 1 {
			t.Errorf("Down: expected selected=1, got %d", selected)
		}

		// Test Down again
		list.Down(nil)
		if selected != 2 {
			t.Errorf("Down again: expected selected=2, got %d", selected)
		}

		// Test Down at end (should stay at 2)
		list.Down(nil)
		if selected != 2 {
			t.Errorf("Down at end: expected selected=2, got %d", selected)
		}

		// Test Up
		list.Up(nil)
		if selected != 1 {
			t.Errorf("Up: expected selected=1, got %d", selected)
		}

		// Test First
		list.First(nil)
		if selected != 0 {
			t.Errorf("First: expected selected=0, got %d", selected)
		}

		// Test Last
		list.Last(nil)
		if selected != 2 {
			t.Errorf("Last: expected selected=2, got %d", selected)
		}

		// Test Up at start
		selected = 0
		list.Up(nil)
		if selected != 0 {
			t.Errorf("Up at start: expected selected=0, got %d", selected)
		}
	})

	t.Run("MaxVisible limits displayed items", func(t *testing.T) {
		items := []Item{{Name: "A"}, {Name: "B"}, {Name: "C"}, {Name: "D"}, {Name: "E"}}
		selected := 0

		list := &SelectionList{
			Items:      &items,
			Selected:   &selected,
			MaxVisible: 3, // Only show 3 items at a time
			Render: func(item *Item) any {
				return TextNode{Content: &item.Name}
			},
		}

		view := VBoxNode{Children: []any{list}}
		tmpl := Build(view)
		buf := NewBuffer(20, 10)
		tmpl.Execute(buf, 20, 10)

		// Should show items 0-2 (A, B, C)
		line0 := extractLine(buf, 0, 4)
		line1 := extractLine(buf, 1, 4)
		line2 := extractLine(buf, 2, 4)
		line3 := extractLine(buf, 3, 4) // Should be empty

		if line0 != "> A " {
			t.Errorf("row 0: expected '> A ', got %q", line0)
		}
		if line1 != "  B " {
			t.Errorf("row 1: expected '  B ', got %q", line1)
		}
		if line2 != "  C " {
			t.Errorf("row 2: expected '  C ', got %q", line2)
		}
		if line3 != "    " {
			t.Errorf("row 3: expected empty, got %q", line3)
		}
	})

	t.Run("viewport scrolls with selection", func(t *testing.T) {
		items := []Item{{Name: "A"}, {Name: "B"}, {Name: "C"}, {Name: "D"}, {Name: "E"}}
		selected := 0

		list := &SelectionList{
			Items:      &items,
			Selected:   &selected,
			MaxVisible: 3,
			Render: func(item *Item) any {
				return TextNode{Content: &item.Name}
			},
		}

		view := VBoxNode{Children: []any{list}}
		tmpl := Build(view)
		buf := NewBuffer(20, 10)
		tmpl.Execute(buf, 20, 10)

		// Move down past visible window
		list.Down(nil) // 0 -> 1
		list.Down(nil) // 1 -> 2
		list.Down(nil) // 2 -> 3 (should scroll)

		buf.Clear()
		tmpl.Execute(buf, 20, 10)

		// Now viewport should show B, C, D (items 1-3) with D selected
		line0 := extractLine(buf, 0, 4)
		line1 := extractLine(buf, 1, 4)
		line2 := extractLine(buf, 2, 4)

		if line0 != "  B " {
			t.Errorf("after scroll: row 0 expected '  B ', got %q", line0)
		}
		if line1 != "  C " {
			t.Errorf("after scroll: row 1 expected '  C ', got %q", line1)
		}
		if line2 != "> D " {
			t.Errorf("after scroll: row 2 expected '> D ', got %q", line2)
		}
	})

	t.Run("viewport scrolls up", func(t *testing.T) {
		items := []Item{{Name: "A"}, {Name: "B"}, {Name: "C"}, {Name: "D"}, {Name: "E"}}
		selected := 4 // Start at end

		list := &SelectionList{
			Items:      &items,
			Selected:   &selected,
			MaxVisible: 3,
			Render: func(item *Item) any {
				return TextNode{Content: &item.Name}
			},
		}

		view := VBoxNode{Children: []any{list}}
		tmpl := Build(view)
		buf := NewBuffer(20, 10)
		tmpl.Execute(buf, 20, 10)

		// Should show C, D, E (items 2-4) with E selected
		line2 := extractLine(buf, 2, 4)
		if line2 != "> E " {
			t.Errorf("initial: row 2 expected '> E ', got %q", line2)
		}

		// Move up to beginning
		list.First(nil)
		buf.Clear()
		tmpl.Execute(buf, 20, 10)

		// Now viewport should show A, B, C (items 0-2) with A selected
		line0 := extractLine(buf, 0, 4)
		if line0 != "> A " {
			t.Errorf("after First: row 0 expected '> A ', got %q", line0)
		}
	})
}

func TestConditionInsideForEach(t *testing.T) {
	t.Skip("TODO: If inside ForEach not fully implemented in new Template")
	t.Run("If evaluates per element in ForEach", func(t *testing.T) {
		type Item struct {
			Name     string
			Selected bool
		}
		items := []Item{
			{Name: "Alpha", Selected: false},
			{Name: "Beta", Selected: true},
			{Name: "Gamma", Selected: false},
		}

		view := VBoxNode{Children: []any{
			ForEach(&items, func(item *Item) any {
				// Use If to conditionally style each item based on its Selected field
				return If(&item.Selected).Eq(true).
					Then(TextNode{Content: &item.Name, Style: Style{Attr: AttrBold}}).
					Else(TextNode{Content: &item.Name})
			}),
		}}

		tmpl := Build(view)
		buf := NewBuffer(20, 10)
		tmpl.Execute(buf, 20, 10)

		// All items should render correctly
		line0 := extractLine(buf, 0, 7)
		line1 := extractLine(buf, 1, 7)
		line2 := extractLine(buf, 2, 7)

		if line0 != "Alpha  " {
			t.Errorf("row 0: expected 'Alpha  ', got %q", line0)
		}
		if line1 != "Beta   " {
			t.Errorf("row 1: expected 'Beta   ', got %q", line1)
		}
		if line2 != "Gamma  " {
			t.Errorf("row 2: expected 'Gamma  ', got %q", line2)
		}

		// Change selection and re-render
		items[0].Selected = true
		items[1].Selected = false
		buf.Clear()
		tmpl.Execute(buf, 20, 10)

		// Items should still render (just with different styles)
		line0 = extractLine(buf, 0, 7)
		line1 = extractLine(buf, 1, 7)
		if line0 != "Alpha  " {
			t.Errorf("after change: row 0 expected 'Alpha  ', got %q", line0)
		}
		if line1 != "Beta   " {
			t.Errorf("after change: row 1 expected 'Beta   ', got %q", line1)
		}
	})

	t.Run("If with same component different style", func(t *testing.T) {
		type Item struct {
			Text     string
			IsActive bool
		}
		items := []Item{
			{Text: "Inactive", IsActive: false},
			{Text: "Active", IsActive: true},
		}

		view := VBoxNode{Children: []any{
			ForEach(&items, func(item *Item) any {
				// Both branches use Text with same pointer, just different styles
				return If(&item.IsActive).Eq(true).
					Then(TextNode{Content: &item.Text, Style: Style{Attr: AttrBold}}).
					Else(TextNode{Content: &item.Text, Style: Style{Attr: AttrDim}})
			}),
		}}

		tmpl := Build(view)
		buf := NewBuffer(20, 10)
		tmpl.Execute(buf, 20, 10)

		// Both items should render their text correctly
		line0 := extractLine(buf, 0, 10)
		line1 := extractLine(buf, 1, 10)

		if line0 != "Inactive  " {
			t.Errorf("row 0: expected 'Inactive  ', got %q", line0)
		}
		if line1 != "Active    " {
			t.Errorf("row 1: expected 'Active    ', got %q", line1)
		}

		// Flip the active states
		items[0].IsActive = true
		items[1].IsActive = false
		buf.Clear()
		tmpl.Execute(buf, 20, 10)

		// Content should be the same (styles would differ if we checked them)
		line0 = extractLine(buf, 0, 10)
		line1 = extractLine(buf, 1, 10)
		if line0 != "Inactive  " {
			t.Errorf("after flip: row 0 expected 'Inactive  ', got %q", line0)
		}
		if line1 != "Active    " {
			t.Errorf("after flip: row 1 expected 'Active    ', got %q", line1)
		}
	})
}

func TestHBoxLayout(t *testing.T) {
	t.Run("HBox places children horizontally", func(t *testing.T) {
		view := HBoxNode{Children: []any{
			TextNode{Content: "AAA"},
			TextNode{Content: "BBB"},
			TextNode{Content: "CCC"},
		}}

		tmpl := Build(view)
		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		// All three texts should be on row 0, horizontally adjacent
		line := extractLine(buf, 0, 12)
		if line != "AAABBBCCC   " {
			t.Errorf("expected 'AAABBBCCC   ', got %q", line)
		}
	})

	t.Run("HBox with gap", func(t *testing.T) {
		view := HBoxNode{Gap: 2, Children: []any{
			TextNode{Content: "AA"},
			TextNode{Content: "BB"},
		}}

		tmpl := Build(view)
		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		// "AA" then 2 spaces gap then "BB"
		line := extractLine(buf, 0, 10)
		if line != "AA  BB    " {
			t.Errorf("expected 'AA  BB    ', got %q", line)
		}
	})

	t.Run("VBox places children vertically", func(t *testing.T) {
		view := VBoxNode{Children: []any{
			TextNode{Content: "AAA"},
			TextNode{Content: "BBB"},
		}}

		tmpl := Build(view)
		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		line0 := extractLine(buf, 0, 5)
		line1 := extractLine(buf, 1, 5)
		if line0 != "AAA  " {
			t.Errorf("row 0: expected 'AAA  ', got %q", line0)
		}
		if line1 != "BBB  " {
			t.Errorf("row 1: expected 'BBB  ', got %q", line1)
		}
	})

	t.Run("Nested HBox in VBox", func(t *testing.T) {
		view := VBoxNode{Children: []any{
			HBoxNode{Children: []any{
				TextNode{Content: "A"},
				TextNode{Content: "B"},
			}},
			TextNode{Content: "C"},
		}}

		tmpl := Build(view)
		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		line0 := extractLine(buf, 0, 5)
		line1 := extractLine(buf, 1, 5)
		if line0 != "AB   " {
			t.Errorf("row 0: expected 'AB   ', got %q", line0)
		}
		if line1 != "C    " {
			t.Errorf("row 1: expected 'C    ', got %q", line1)
		}
	})
}

func TestRichTextInsideForEach(t *testing.T) {
	t.Run("RichText with pointer renders per element", func(t *testing.T) {
		type DisplayLine struct {
			LineNum string
			Spans   []Span
		}
		lines := []DisplayLine{
			{LineNum: "1 ", Spans: []Span{{Text: "Hello"}}},
			{LineNum: "2 ", Spans: []Span{{Text: "World"}}},
			{LineNum: "3 ", Spans: []Span{{Text: "Test"}}},
		}

		view := VBoxNode{Children: []any{
			ForEach(&lines, func(dl *DisplayLine) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &dl.LineNum},
					RichTextNode{Spans: &dl.Spans},
				}}
			}),
		}}

		tmpl := Build(view)
		buf := NewBuffer(20, 10)
		tmpl.Execute(buf, 20, 10)

		// Each line should render correctly
		line0 := extractLine(buf, 0, 9)
		line1 := extractLine(buf, 1, 9)
		line2 := extractLine(buf, 2, 9)

		if line0 != "1 Hello  " {
			t.Errorf("row 0: expected '1 Hello  ', got %q", line0)
		}
		if line1 != "2 World  " {
			t.Errorf("row 1: expected '2 World  ', got %q", line1)
		}
		if line2 != "3 Test   " {
			t.Errorf("row 2: expected '3 Test   ', got %q", line2)
		}
	})

	t.Run("RichText updates dynamically", func(t *testing.T) {
		type Line struct {
			Spans []Span
		}
		lines := []Line{
			{Spans: []Span{{Text: "AAA"}}},
			{Spans: []Span{{Text: "BBB"}}},
		}

		view := VBoxNode{Children: []any{
			ForEach(&lines, func(l *Line) any {
				return RichTextNode{Spans: &l.Spans}
			}),
		}}

		tmpl := Build(view)
		buf := NewBuffer(20, 10)
		tmpl.Execute(buf, 20, 10)

		line0 := extractLine(buf, 0, 5)
		line1 := extractLine(buf, 1, 5)
		if line0 != "AAA  " {
			t.Errorf("before: row 0 expected 'AAA  ', got %q", line0)
		}
		if line1 != "BBB  " {
			t.Errorf("before: row 1 expected 'BBB  ', got %q", line1)
		}

		// Update spans
		lines[0].Spans = []Span{{Text: "XXX"}}
		lines[1].Spans = []Span{{Text: "YYY"}}
		buf.Clear()
		tmpl.Execute(buf, 20, 10)

		line0 = extractLine(buf, 0, 5)
		line1 = extractLine(buf, 1, 5)
		if line0 != "XXX  " {
			t.Errorf("after: row 0 expected 'XXX  ', got %q", line0)
		}
		if line1 != "YYY  " {
			t.Errorf("after: row 1 expected 'YYY  ', got %q", line1)
		}
	})

	t.Run("RichText with styled spans", func(t *testing.T) {
		type DisplayLine struct {
			Spans []Span
		}
		// Using styled spans like visual mode in vim
		lines := []DisplayLine{
			{Spans: []Span{
				{Text: "normal", Style: Style{}},
				{Text: "selected", Style: Style{Attr: AttrInverse}},
			}},
		}

		view := VBoxNode{Children: []any{
			ForEach(&lines, func(dl *DisplayLine) any {
				return RichTextNode{Spans: &dl.Spans}
			}),
		}}

		tmpl := Build(view)
		buf := NewBuffer(20, 10)
		tmpl.Execute(buf, 20, 10)

		// Both spans should render
		line0 := extractLine(buf, 0, 16)
		if line0 != "normalselected  " {
			t.Errorf("row 0: expected 'normalselected  ', got %q", line0)
		}
	})
}
