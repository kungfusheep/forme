package glyph

import (
	"fmt"
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

func TestSelectionListOverflowClipped(t *testing.T) {
	// Reproduces the pprofin bug: a bordered VBox with header, Grow(1) list,
	// and footer. The list has MaxVisible(20) but the terminal is only 15 lines
	// tall. Without the fix, the list renders past the border and pushes
	// the footer off-screen.
	items := make([]string, 25)
	for i := range items {
		items[i] = fmt.Sprintf("Item %02d", i+1)
	}
	selected := 0

	list := &SelectionList{
		Items:      &items,
		Selected:   &selected,
		Marker:     "> ",
		MaxVisible: 20,
		Render: func(s *string) any {
			return TextNode{Content: s}
		},
	}

	// Layout mirrors pprofin: bordered root, header, grow list, footer
	view := VBoxNode{
		Title:    "test",
		Children: []any{
			TextNode{Content: "header"},                       // 1 line
			HRuleNode{},                                       // 1 line
			VBoxNode{Children: []any{list}}.Grow(1),           // flex fill
			HRuleNode{},                                       // 1 line
			TextNode{Content: "footer"},                       // 1 line
		},
	}.Border(BorderSingle)

	screenH := int16(15)
	tmpl := Build(view)
	buf := NewBuffer(40, int(screenH))
	tmpl.Execute(buf, 40, screenH)

	// Dump for debugging
	for y := 0; y < int(screenH); y++ {
		t.Logf("Line %2d: %q", y, buf.GetLine(y))
	}

	// The bottom border must be on the last line
	bottomLeft := buf.Get(0, int(screenH)-1)
	if bottomLeft.Rune != '└' {
		t.Errorf("Bottom-left corner should be └, got %c (border overwritten by list overflow)", bottomLeft.Rune)
	}

	// The footer text must be on the second-to-last line (inside border)
	footerLine := buf.GetLine(int(screenH) - 2)
	if !contains(footerLine, "footer") {
		t.Errorf("Footer should be visible on line %d, got: %q", screenH-2, footerLine)
	}

	// The header must be on line 1 (inside top border)
	headerLine := buf.GetLine(1)
	if !contains(headerLine, "header") {
		t.Errorf("Header should be on line 1, got: %q", headerLine)
	}

	// List items should NOT appear on the footer or border lines
	footerArea := buf.GetLine(int(screenH) - 2)
	if contains(footerArea, "Item") {
		t.Errorf("List items should not overflow into footer area: %q", footerArea)
	}
	borderLine := buf.GetLine(int(screenH) - 1)
	if contains(borderLine, "Item") {
		t.Errorf("List items should not overflow into border: %q", borderLine)
	}
}

func TestSelectionListOverflowScrolling(t *testing.T) {
	// When the list is clipped to fewer rows than MaxVisible, scrolling
	// to a selected item near the bottom should still work correctly.
	items := make([]string, 25)
	for i := range items {
		items[i] = fmt.Sprintf("Item %02d", i+1)
	}
	selected := 24 // last item

	list := &SelectionList{
		Items:      &items,
		Selected:   &selected,
		Marker:     "> ",
		MaxVisible: 20,
		Render: func(s *string) any {
			return TextNode{Content: s}
		},
	}

	view := VBoxNode{
		Title:    "test",
		Children: []any{
			TextNode{Content: "header"},
			HRuleNode{},
			VBoxNode{Children: []any{list}}.Grow(1),
			HRuleNode{},
			TextNode{Content: "footer"},
		},
	}.Border(BorderSingle)

	screenH := int16(15)
	tmpl := Build(view)
	buf := NewBuffer(40, int(screenH))
	tmpl.Execute(buf, 40, screenH)

	for y := 0; y < int(screenH); y++ {
		t.Logf("Line %2d: %q", y, buf.GetLine(y))
	}

	// The selected item (Item 25) should be visible somewhere in the list area
	found := false
	for y := 3; y < int(screenH)-3; y++ { // list area is between header+hrule and hrule+footer
		line := buf.GetLine(y)
		if contains(line, "> Item 25") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Selected item 'Item 25' should be visible in the clipped list area")
	}

	// Footer must still be visible
	footerLine := buf.GetLine(int(screenH) - 2)
	if !contains(footerLine, "footer") {
		t.Errorf("Footer should still be visible: %q", footerLine)
	}
}
