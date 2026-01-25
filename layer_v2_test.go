package tui

import (
	"testing"
)

func TestV2LayerBlit(t *testing.T) {
	t.Run("single layer blits to correct position", func(t *testing.T) {
		// Create a layer with content
		layer := NewLayer()
		layerBuf := NewBuffer(10, 5)
		for y := 0; y < 5; y++ {
			layerBuf.WriteStringFast(0, y, string(rune('A'+y))+"----", Style{}, 10)
		}
		layer.SetBuffer(layerBuf)

		// Create screen
		screen := NewBuffer(20, 10)

		// Build view with layer using V2Template
		view := VBoxNode{Children: []any{
			TextNode{Content: "Header"},
			LayerViewNode{Layer: layer, ViewHeight: 3},
			TextNode{Content: "Footer"},
		}}

		tmpl := Build(view)
		tmpl.Execute(screen, 20, 10)

		// Verify header at line 0
		if got := screen.GetLine(0); got != "Header" {
			t.Errorf("line 0: got %q, want %q", got, "Header")
		}

		// Verify layer content at lines 1-3
		if got := screen.GetLine(1); got != "A----" {
			t.Errorf("line 1: got %q, want %q", got, "A----")
		}
		if got := screen.GetLine(2); got != "B----" {
			t.Errorf("line 2: got %q, want %q", got, "B----")
		}
		if got := screen.GetLine(3); got != "C----" {
			t.Errorf("line 3: got %q, want %q", got, "C----")
		}

		// Verify footer at line 4
		if got := screen.GetLine(4); got != "Footer" {
			t.Errorf("line 4: got %q, want %q", got, "Footer")
		}
	})

	t.Run("layer scrolling works", func(t *testing.T) {
		// Create a layer with 10 lines
		layer := NewLayer()
		layerBuf := NewBuffer(10, 10)
		for y := 0; y < 10; y++ {
			layerBuf.WriteStringFast(0, y, string(rune('A'+y))+"AAAA", Style{}, 10)
		}
		layer.SetBuffer(layerBuf)

		screen := NewBuffer(20, 10)

		view := VBoxNode{Children: []any{
			LayerViewNode{Layer: layer, ViewHeight: 3},
			TextNode{Content: "---"},
		}}

		tmpl := Build(view)

		// Initial render - scroll 0
		tmpl.Execute(screen, 20, 10)

		if got := screen.GetLine(0); got != "AAAAA" {
			t.Errorf("initial line 0: got %q, want %q", got, "AAAAA")
		}

		// Scroll down by 2
		layer.ScrollDown(2)
		screen.Clear()
		tmpl.Execute(screen, 20, 10)

		// Should now show C, D, E (indices 2, 3, 4)
		if got := screen.GetLine(0); got != "CAAAA" {
			t.Errorf("after scroll line 0: got %q, want %q", got, "CAAAA")
		}
		if got := screen.GetLine(1); got != "DAAAA" {
			t.Errorf("after scroll line 1: got %q, want %q", got, "DAAAA")
		}
	})

	t.Run("layer with nil buffer renders empty", func(t *testing.T) {
		layer := NewLayer()
		// Don't set any buffer

		screen := NewBuffer(20, 5)

		view := VBoxNode{Children: []any{
			TextNode{Content: "Before"},
			LayerViewNode{Layer: layer, ViewHeight: 2},
			TextNode{Content: "After"},
		}}

		tmpl := Build(view)
		screen.Clear()
		tmpl.Execute(screen, 20, 5)

		// Text should render - key is it shouldn't crash with nil buffer
		if got := screen.GetLine(0); got != "Before" {
			t.Errorf("line 0: got %q, want %q", got, "Before")
		}

		// After should be at line 3 (0=Before, 1-2=layer, 3=After)
		if got := screen.GetLine(3); got != "After" {
			t.Errorf("line 3: got %q, want %q", got, "After")
		}
	})

	t.Run("layer inside bordered container", func(t *testing.T) {
		// Create a layer
		layer := NewLayer()
		layerBuf := NewBuffer(30, 5)
		for y := 0; y < 5; y++ {
			layerBuf.WriteStringFast(0, y, string(rune('A'+y))+"----line", Style{}, 30)
		}
		layer.SetBuffer(layerBuf)

		screen := NewBuffer(40, 10)

		view := VBoxNode{Children: []any{
			VBoxNode{
				Title: "Content",
				Children: []any{
					LayerViewNode{Layer: layer, ViewHeight: 3},
				},
			}.Border(BorderSingle),
		}}

		tmpl := Build(view)
		tmpl.Execute(screen, 40, 10)

		// Line 0 should have border top with title
		line0 := screen.GetLine(0)
		if !contains(line0, "Content") {
			t.Errorf("line 0 should have title: got %q", line0)
		}

		// Line 1 should have border + layer content
		line1 := screen.GetLine(1)
		if !contains(line1, "A----line") {
			t.Errorf("line 1 should contain layer content: got %q", line1)
		}

		// Line 4 should have bottom border
		line4 := screen.GetLine(4)
		if !contains(line4, "└") && !contains(line4, "─") {
			t.Errorf("line 4 should have bottom border: got %q", line4)
		}
	})
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
