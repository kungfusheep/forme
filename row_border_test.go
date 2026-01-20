package tui

import (
	"testing"
)

// TestHBoxWithBorderedChildren tests that borders inside HBox flex children
// are drawn correctly.
func TestHBoxWithBorderedChildren(t *testing.T) {
	// Simplified dashboard structure:
	// Row {
	//   Col.Grow(1) {  // Left panel
	//     Col "Stats".Border()
	//     Col "Load".Border()
	//   }
	//   Col.Grow(2) {  // Right panel
	//     Col "Info".Border()
	//   }
	// }
	view := HBox{Gap: 1, Children: []any{
		// Left panel
		VBox{Children: []any{
			VBox{
				Title:    "Stats",
				Children: []any{
					Text{Content: "Tasks: 142"},
					Text{Content: "Sleeping: 138"},
				},
			}.Border(BorderSingle).BorderFG(Cyan),
			VBox{
				Title:    "Load",
				Children: []any{
					Text{Content: "1.17, 0.69, 0.85"},
				},
			}.Border(BorderRounded).BorderFG(Green),
		}}.Grow(1),

		// Right panel
		VBox{Children: []any{
			VBox{
				Title:    "Info",
				Children: []any{
					Text{Content: "Line 1"},
					Text{Content: "Line 2"},
					Text{Content: "Line 3"},
				},
			}.Border(BorderSingle).BorderFG(Magenta),
		}}.Grow(2),
	}}

	tmpl := Build(view)
	buf := NewBuffer(60, 15)
	tmpl.Execute(buf, 60, 15)

	// Debug: print geometry of all ops
	t.Log("Op geometries:")
	for i, op := range tmpl.ops {
		g := tmpl.geom[i]
		name := ""
		if op.Title != "" {
			name = op.Title
		} else if op.Kind == OpContainer {
			if op.IsRow {
				name = "Row"
			} else {
				name = "Col"
			}
		}
		t.Logf("  [%d] %s: LocalX=%d LocalY=%d W=%d H=%d", i, name, g.LocalX, g.LocalY, g.W, g.H)
	}

	t.Log("Buffer contents:")
	for y := 0; y < 15; y++ {
		line := buf.GetLine(y)
		t.Logf("Line %2d: %s", y, line)
	}

	// Check for Stats bottom border (should be cyan single border)
	statsBottomFound := false
	loadBottomFound := false

	for y := 0; y < 15; y++ {
		for x := 0; x < 60; x++ {
			cell := buf.Get(x, y)
			if cell.Rune == BorderSingle.BottomLeft && cell.Style.FG == Cyan {
				statsBottomFound = true
				t.Logf("Found Stats bottom border at y=%d", y)
			}
			if cell.Rune == BorderRounded.BottomLeft && cell.Style.FG == Green {
				loadBottomFound = true
				t.Logf("Found Load bottom border at y=%d", y)
			}
		}
	}

	if !statsBottomFound {
		t.Error("Stats box bottom border not found!")
	}
	if !loadBottomFound {
		t.Error("Load box bottom border not found!")
	}
}
