package forme

import (
	"testing"
)

// TestIfWithGrowBorder tests that a bordered container inside an If with Grow
// has its bottom border drawn at the correct position.
func TestIfWithGrowBorder(t *testing.T) {
	showProcs := true

	// Mimic the dashboard structure:
	// VBoxNode{Grow(1)} containing:
	//   - VBoxNode{Border, Grow(1)} "Timing"
	//   - If(showProcs).Then(VBoxNode{Border, Grow(2)} "Processes")
	view := VBoxNode{Children: []any{
		VBoxNode{
			Title:    "Timing",
			Children: []any{
				TextNode{Content: "Line 1"},
				TextNode{Content: "Line 2"},
				TextNode{Content: "Line 3"},
			},
		}.Border(BorderSingle).BorderFG(Yellow).Grow(1),

		If(&showProcs).Eq(true).Then(VBoxNode{
			Title:    "Processes",
			Children: []any{
				TextNode{Content: "Process 1"},
				TextNode{Content: "Process 2"},
				TextNode{Content: "Process 3"},
			},
		}.Border(BorderSingle).BorderFG(BrightBlue).Grow(2)),
	}}.Grow(1)

	// Build template
	tmpl := Build(view)

	// Create a buffer with specific dimensions
	buf := NewBuffer(80, 30)

	// Execute template - this does width distribution, layout, and render
	tmpl.Execute(buf, 80, 30)

	// Print the buffer to see the output
	t.Log("Buffer contents:")
	for y := 0; y < 30; y++ {
		line := buf.GetLine(y)
		if line != "" {
			t.Logf("Line %2d: %s", y, line)
		}
	}

	// Check that Timing box bottom border exists
	timingBottomFound := false
	processesBottomFound := false

	for y := 0; y < 30; y++ {
		for x := 0; x < 80; x++ {
			cell := buf.Get(x, y)
			// Check for bottom-left corner of a box
			if cell.Rune == BorderSingle.BottomLeft {
				// Look at the next few chars to see if it's a border
				if x+1 < 80 {
					nextCell := buf.Get(x+1, y)
					if nextCell.Rune == BorderSingle.Horizontal {
						// This is a bottom border
						// Check color to distinguish Timing (Yellow) vs Processes (BrightBlue)
						if cell.Style.FG == Yellow {
							timingBottomFound = true
							t.Logf("Found Timing bottom border at y=%d", y)
						} else if cell.Style.FG == BrightBlue {
							processesBottomFound = true
							t.Logf("Found Processes bottom border at y=%d", y)
						}
					}
				}
			}
		}
	}

	if !timingBottomFound {
		t.Error("Timing box bottom border not found")
	}
	if !processesBottomFound {
		t.Error("Processes box bottom border not found")
	}
}

// TestDashboardLayoutBorders tests the full dashboard-like structure with
// nested grows and conditionals.
func TestDashboardLayoutBorders(t *testing.T) {
	showProcs := true
	showGraph := true

	// Full dashboard structure (simplified):
	// VBoxNode{Children: [
	//   Text "Header"
	//   Text "Progress bars"
	//   HBoxNode{Children: [Left.Grow(1), Right.Grow(2)]}  // Horizontal flex
	//   VBoxNode{Children: [
	//     Col "Timing".Grow(1)
	//     If.Then(Col "Processes".Grow(2))
	//   ]}.Grow(1)  // The OUTER Col also has Grow!
	//   Text "Footer"
	// ]}
	view := VBoxNode{Children: []any{
		// Fixed header
		TextNode{Content: "Dashboard Header"},
		TextNode{Content: "CPU: [████████████________] 60%"},

		// Main content row (horizontal flex)
		HBoxNode{Gap: 1, Children: []any{
			VBoxNode{
				Title:    "Stats",
				Children: []any{
					TextNode{Content: "Tasks: 100"},
					TextNode{Content: "Memory: 4GB"},
				},
			}.Border(BorderSingle).BorderFG(Cyan).Grow(1),
			VBoxNode{
				Title: "Load",
				Children: []any{
					If(&showGraph).Eq(true).Then(
						TextNode{Content: "Graph: ▁▂▃▄▅▆▇█"},
					),
				},
			}.Border(BorderRounded).BorderFG(Green).Grow(2),
		}},

		// Middle section with vertical flex - THIS IS THE KEY PART
		// The outer Col has Grow(1), inner children have Grow(1) and Grow(2)
		VBoxNode{Children: []any{
			VBoxNode{
				Title:    "Timing",
				Children: []any{
					TextNode{Content: "Render: 100µs"},
					TextNode{Content: "Flush: 50µs"},
				},
			}.Border(BorderDouble).BorderFG(Yellow).Grow(1),

			If(&showProcs).Eq(true).Then(VBoxNode{
				Title:    "Processes",
				Children: []any{
					TextNode{Content: "PID    NAME     CPU"},
					TextNode{Content: "1001   nginx    2.5%"},
					TextNode{Content: "1002   node     5.2%"},
				},
			}.Border(BorderSingle).BorderFG(BrightBlue).Grow(2)),
		}}.Grow(1), // <-- OUTER COL HAS GROW!

		// Fixed footer
		TextNode{Content: "Press q to quit"},
	}}

	tmpl := Build(view)
	buf := NewBuffer(80, 40)
	tmpl.Execute(buf, 80, 40)

	t.Log("Buffer contents:")
	for y := 0; y < 40; y++ {
		line := buf.GetLine(y)
		if line != "" {
			t.Logf("Line %2d: %s", y, line)
		}
	}

	// Check for Processes bottom border
	processesBottomFound := false
	for y := 0; y < 40; y++ {
		for x := 0; x < 80; x++ {
			cell := buf.Get(x, y)
			if cell.Rune == BorderSingle.BottomLeft && cell.Style.FG == BrightBlue {
				if x+1 < 80 {
					nextCell := buf.Get(x+1, y)
					if nextCell.Rune == BorderSingle.Horizontal {
						processesBottomFound = true
						t.Logf("Found Processes bottom border at y=%d", y)
					}
				}
			}
		}
	}

	if !processesBottomFound {
		t.Error("Processes box bottom border not found - this is the bug we're debugging!")
	}
}
