package glyph

import (
	"strings"
	"testing"
)

type TestItem struct {
	Name string
	Done bool
}

func TestListCNavigation(t *testing.T) {
	items := []TestItem{
		{Name: "First", Done: false},
		{Name: "Second", Done: true},
		{Name: "Third", Done: false},
	}

	var list *ListC[TestItem]

	listComp := List(&items).Render(func(item *TestItem) any {
		return Text(&item.Name)
	}).Ref(func(l *ListC[TestItem]) { list = l })

	// Build and execute template to initialize len
	tmpl := Build(VBox(listComp))
	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// Verify initial selection is 0
	if list.Index() != 0 {
		t.Errorf("Expected initial index 0, got %d", list.Index())
	}

	// Verify Selected returns correct item
	if list.Selected() == nil {
		t.Fatal("Selected() returned nil")
	}
	if list.Selected().Name != "First" {
		t.Errorf("Expected 'First', got '%s'", list.Selected().Name)
	}

	// Test navigation
	list.Down(nil)
	if list.Index() != 1 {
		t.Errorf("After Down, expected index 1, got %d", list.Index())
	}
	if list.Selected().Name != "Second" {
		t.Errorf("After Down, expected 'Second', got '%s'", list.Selected().Name)
	}

	list.Down(nil)
	if list.Index() != 2 {
		t.Errorf("After second Down, expected index 2, got %d", list.Index())
	}

	// Can't go past end
	list.Down(nil)
	if list.Index() != 2 {
		t.Errorf("Should stay at 2 (end), got %d", list.Index())
	}

	list.Up(nil)
	if list.Index() != 1 {
		t.Errorf("After Up, expected index 1, got %d", list.Index())
	}

	// Test First/Last
	list.Last(nil)
	if list.Index() != 2 {
		t.Errorf("After Last, expected index 2, got %d", list.Index())
	}

	list.First(nil)
	if list.Index() != 0 {
		t.Errorf("After First, expected index 0, got %d", list.Index())
	}
}

func TestListCRendersText(t *testing.T) {
	items := []TestItem{
		{Name: "Apple", Done: false},
		{Name: "Banana", Done: true},
	}

	listComp := List(&items).Render(func(item *TestItem) any {
		return Text(&item.Name)
	})

	tmpl := Build(VBox(listComp))
	buf := NewBuffer(40, 5)
	tmpl.Execute(buf, 40, 5)

	// Check that text renders correctly
	line0 := buf.GetLine(0)
	line1 := buf.GetLine(1)

	// Should see marker and text
	if line0 == "" {
		t.Error("Line 0 is empty")
	}
	if !strings.Contains(line0, "Apple") {
		t.Errorf("Line 0 should contain 'Apple', got: %q", line0)
	}
	if !strings.Contains(line1, "Banana") {
		t.Errorf("Line 1 should contain 'Banana', got: %q", line1)
	}
}

func TestListCOnSelect(t *testing.T) {
	items := []TestItem{
		{Name: "First"},
		{Name: "Second"},
		{Name: "Third"},
	}

	var list *ListC[TestItem]
	var selected string
	callCount := 0

	listComp := List(&items).Render(func(item *TestItem) any {
		return Text(&item.Name)
	}).OnSelect(func(item *TestItem) {
		selected = item.Name
		callCount++
	}).Ref(func(l *ListC[TestItem]) { list = l })

	tmpl := Build(VBox(listComp))
	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// move down, should fire OnSelect
	list.Down(nil)
	if selected != "Second" {
		t.Errorf("OnSelect should receive 'Second', got %q", selected)
	}
	if callCount != 1 {
		t.Errorf("OnSelect should fire once, fired %d times", callCount)
	}

	// move down again
	list.Down(nil)
	if selected != "Third" {
		t.Errorf("OnSelect should receive 'Third', got %q", selected)
	}

	// move down at end, should NOT fire (no change)
	callCount = 0
	list.Down(nil)
	if callCount != 0 {
		t.Errorf("OnSelect should not fire when selection doesn't change, fired %d", callCount)
	}

	// move up
	callCount = 0
	list.Up(nil)
	if selected != "Second" {
		t.Errorf("OnSelect should receive 'Second', got %q", selected)
	}
	if callCount != 1 {
		t.Errorf("OnSelect should fire once on Up, fired %d", callCount)
	}

	// First/Last
	callCount = 0
	list.Last(nil)
	if selected != "Third" {
		t.Errorf("OnSelect should receive 'Third' after Last, got %q", selected)
	}
	list.First(nil)
	if selected != "First" {
		t.Errorf("OnSelect should receive 'First' after First, got %q", selected)
	}
	if callCount != 2 {
		t.Errorf("Expected 2 calls (Last+First), got %d", callCount)
	}
}

func TestListCDelete(t *testing.T) {
	items := []TestItem{
		{Name: "First", Done: false},
		{Name: "Second", Done: true},
		{Name: "Third", Done: false},
	}

	var list *ListC[TestItem]

	listComp := List(&items).Render(func(item *TestItem) any {
		return Text(&item.Name)
	}).Ref(func(l *ListC[TestItem]) { list = l })

	// Need to compile/execute to set len
	tmpl := Build(VBox(listComp))
	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// Delete first item
	list.Delete()
	if len(items) != 2 {
		t.Errorf("After delete, expected 2 items, got %d", len(items))
	}
	if items[0].Name != "Second" {
		t.Errorf("First item should now be 'Second', got '%s'", items[0].Name)
	}
	if list.Index() != 0 {
		t.Errorf("Selection should stay at 0, got %d", list.Index())
	}

	// Move to end and delete
	list.Down(nil)
	if list.Index() != 1 {
		t.Errorf("Expected index 1, got %d", list.Index())
	}
	list.Delete()
	if len(items) != 1 {
		t.Errorf("After second delete, expected 1 item, got %d", len(items))
	}
	// Selection should adjust to stay in bounds
	if list.Index() != 0 {
		t.Errorf("After deleting last item, selection should be 0, got %d", list.Index())
	}
}
