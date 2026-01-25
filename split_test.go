package forme

import (
	"testing"
)

// TestV2SplitLayout tests the nested Row/Col structure used by minivim splits
func TestV2SplitLayout(t *testing.T) {
	// Simulate minivim split structure:
	// Col {
	//   Row {
	//     Col { LayerView, RichText }  <- window 1
	//     Col { LayerView, RichText }  <- window 2
	//   }
	//   Text (status)
	// }

	layer1 := NewLayer()
	buf1 := NewBuffer(40, 10)
	buf1.WriteStringFast(0, 0, "Window 1 content", Style{}, 40)
	layer1.SetBuffer(buf1)

	layer2 := NewLayer()
	buf2 := NewBuffer(40, 10)
	buf2.WriteStringFast(0, 0, "Window 2 content", Style{}, 40)
	layer2.SetBuffer(buf2)

	spans1 := []Span{{Text: "Status 1"}}
	spans2 := []Span{{Text: "Status 2"}}

	view := VBoxNode{Children: []any{
		HBoxNode{Children: []any{
			VBoxNode{Children: []any{
				LayerViewNode{Layer: layer1, ViewHeight: 5},
				RichTextNode{Spans: spans1},
			}},
			VBoxNode{Children: []any{
				LayerViewNode{Layer: layer2, ViewHeight: 5},
				RichTextNode{Spans: spans2},
			}},
		}},
		TextNode{Content: "Global status"},
	}}

	tmpl := Build(view)
	screen := NewBuffer(80, 20)
	tmpl.Execute(screen, 80, 20)

	t.Log("Output:")
	for y := 0; y < 10; y++ {
		t.Logf("%2d: %q", y, screen.GetLine(y))
	}

	// Check that both windows are visible
	line0 := screen.GetLine(0)
	if line0 == "" {
		t.Error("Line 0 is empty - split layout failed")
	}

	// Window 1 content should be at left
	if !contains(line0, "Window 1") {
		t.Errorf("Window 1 content not found at line 0: %q", line0)
	}

	// Window 2 content should also be visible (at some X position)
	found := false
	for y := 0; y < 6; y++ {
		if contains(screen.GetLine(y), "Window 2") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Window 2 content not found in output")
	}
}
