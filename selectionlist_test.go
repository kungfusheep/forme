package forme

import (
	"testing"
)

func TestV2SelectionListBasic(t *testing.T) {
	items := []string{"Apple", "Banana", "Cherry"}
	selected := 1

	view := VBoxNode{Children: []any{
		&SelectionList{
			Items:    &items,
			Selected: &selected,
			Marker:   "> ",
			Render: func(s *string) any {
				return TextNode{Content: s}
			},
		},
	}}

	tmpl := Build(view)
	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// Check output
	line0 := buf.GetLine(0)
	line1 := buf.GetLine(1)
	line2 := buf.GetLine(2)

	t.Logf("Line 0: %q", line0)
	t.Logf("Line 1: %q", line1)
	t.Logf("Line 2: %q", line2)

	// Apple should not have marker (selected = 1)
	if !contains(line0, "  Apple") {
		t.Errorf("Line 0 should have spaces before Apple: %q", line0)
	}

	// Banana should have marker (selected = 1)
	if !contains(line1, "> Banana") {
		t.Errorf("Line 1 should have marker before Banana: %q", line1)
	}

	// Cherry should not have marker
	if !contains(line2, "  Cherry") {
		t.Errorf("Line 2 should have spaces before Cherry: %q", line2)
	}
}

func TestV2SelectionListWithRender(t *testing.T) {
	items := []string{"First", "Second", "Third"}
	selected := 0

	view := VBoxNode{Children: []any{
		&SelectionList{
			Items:    &items,
			Selected: &selected,
			Marker:   "* ",
			Render: func(s *string) any {
				return TextNode{Content: s}
			},
		},
	}}

	tmpl := Build(view)
	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	line0 := buf.GetLine(0)
	t.Logf("Line 0: %q", line0)

	// First should have marker
	if !contains(line0, "* First") {
		t.Errorf("Line 0 should have marker before First: %q", line0)
	}
}

func TestV2SelectionListMaxVisible(t *testing.T) {
	items := []string{"One", "Two", "Three", "Four", "Five"}
	selected := 0

	list := &SelectionList{
		Items:      &items,
		Selected:   &selected,
		Marker:     "> ",
		MaxVisible: 3,
		Render: func(s *string) any {
			return TextNode{Content: s}
		},
	}

	view := VBoxNode{Children: []any{list}}

	tmpl := Build(view)
	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// Should only show 3 items
	line0 := buf.GetLine(0)
	line1 := buf.GetLine(1)
	line2 := buf.GetLine(2)
	line3 := buf.GetLine(3)

	t.Logf("Line 0: %q", line0)
	t.Logf("Line 1: %q", line1)
	t.Logf("Line 2: %q", line2)
	t.Logf("Line 3: %q", line3)

	if !contains(line0, "One") {
		t.Errorf("Line 0 should contain One: %q", line0)
	}
	if !contains(line2, "Three") {
		t.Errorf("Line 2 should contain Three: %q", line2)
	}
	// Line 3 should be empty (only showing 3 items)
	if contains(line3, "Four") {
		t.Errorf("Line 3 should NOT contain Four (MaxVisible=3): %q", line3)
	}
}

func TestV2SelectionListScrolling(t *testing.T) {
	items := []string{"One", "Two", "Three", "Four", "Five"}
	selected := 3 // Select "Four"

	list := &SelectionList{
		Items:      &items,
		Selected:   &selected,
		Marker:     "> ",
		MaxVisible: 3,
		Render: func(s *string) any {
			return TextNode{Content: s}
		},
	}

	view := VBoxNode{Children: []any{list}}

	tmpl := Build(view)
	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// After ensureVisible, offset should be 1 so we see [Two, Three, Four]
	line0 := buf.GetLine(0)
	line1 := buf.GetLine(1)
	line2 := buf.GetLine(2)

	t.Logf("Line 0: %q", line0)
	t.Logf("Line 1: %q", line1)
	t.Logf("Line 2: %q", line2)

	// "Four" should be visible and selected
	found := false
	for y := 0; y < 3; y++ {
		line := buf.GetLine(y)
		if contains(line, "> Four") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Selected item 'Four' with marker not found in visible window")
	}
}
