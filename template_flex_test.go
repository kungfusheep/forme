package tui

import (
	"strings"
	"testing"
)

func TestSerialFlexPercentWidth(t *testing.T) {
	// Test that PercentWidth distributes space correctly in a Row
	tmpl := Build(HBoxNode{
		Children: []any{
			VBoxNode{Children: []any{TextNode{Content: "Left"}}}.WidthPct(0.5),
			VBoxNode{Children: []any{TextNode{Content: "Right"}}}.WidthPct(0.5),
		},
	})

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	output := buf.String()
	t.Logf("PercentWidth output:\n%s", output)

	// Left should be at position 0
	if !strings.Contains(output, "Left") {
		t.Error("Output should contain 'Left'")
	}
	// Right should be at position 20 (50% of 40)
	if !strings.Contains(output, "Right") {
		t.Error("Output should contain 'Right'")
	}
}

func TestSerialFlexGrow(t *testing.T) {
	// Test that FlexGrow distributes remaining space
	tmpl := Build(VBoxNode{
		Children: []any{
			TextNode{Content: "Header"}, // H=1
			VBoxNode{Children: []any{TextNode{Content: "Content"}}}.Grow(1),
		},
	})

	buf := NewBuffer(40, 20)
	tmpl.Execute(buf, 40, 20)

	output := buf.String()
	t.Logf("FlexGrow output:\n%s", output)

	// Header should be at Y=0
	if !strings.Contains(output, "Header") {
		t.Error("Output should contain 'Header'")
	}
	// Content should be somewhere below
	if !strings.Contains(output, "Content") {
		t.Error("Output should contain 'Content'")
	}
}

func TestSerialFlexBorder(t *testing.T) {
	// Test that borders are drawn correctly
	tmpl := Build(VBoxNode{
		Title: "Panel",
		Children: []any{
			TextNode{Content: "Inside"},
		},
	}.Border(BorderSingle))

	buf := NewBuffer(30, 5)
	tmpl.Execute(buf, 30, 5)

	output := buf.String()
	t.Logf("Border output:\n%s", output)

	// Check border corner
	if buf.Get(0, 0).Rune != '┌' {
		t.Errorf("Top-left should be ┌, got %c", buf.Get(0, 0).Rune)
	}

	// Check content is inside
	if !strings.Contains(output, "Inside") {
		t.Error("Output should contain 'Inside'")
	}
}

func TestSerialFlexExplicitHeight(t *testing.T) {
	// Test explicit height is respected
	tmpl := Build(VBoxNode{
		Children: []any{
			TextNode{Content: "Line 1"},
			TextNode{Content: "Line 2"},
		},
	}.Height(5))

	buf := NewBuffer(40, 20)
	tmpl.Execute(buf, 40, 20)

	output := buf.String()
	t.Logf("ExplicitHeight output:\n%s", output)

	if !strings.Contains(output, "Line 1") {
		t.Error("Output should contain 'Line 1'")
	}
}

func TestSerialFlexCombined(t *testing.T) {
	// Test combining PercentWidth, FlexGrow, and Border
	tmpl := Build(VBoxNode{
		Children: []any{
			HBoxNode{
				Children: []any{
					VBoxNode{
						Title:    "Left",
						Children: []any{TextNode{Content: "L1"}},
					}.WidthPct(0.5).Border(BorderSingle),
					VBoxNode{
						Title:    "Right",
						Children: []any{TextNode{Content: "R1"}},
					}.WidthPct(0.5).Border(BorderSingle),
				},
			},
			VBoxNode{
				Title:    "Log",
				Children: []any{TextNode{Content: "Log entry"}},
			}.Grow(1).Border(BorderSingle),
		},
	})

	buf := NewBuffer(60, 20)
	tmpl.Execute(buf, 60, 20)

	output := buf.StringTrimmed()
	t.Logf("Combined layout:\n%s", output)

	// Check all content is present
	if !strings.Contains(output, "L1") {
		t.Error("Output should contain 'L1'")
	}
	if !strings.Contains(output, "R1") {
		t.Error("Output should contain 'R1'")
	}
	if !strings.Contains(output, "Log entry") {
		t.Error("Output should contain 'Log entry'")
	}
}

func TestSerialFlexWithPointerBindings(t *testing.T) {
	// Test that flex works with dynamic pointer bindings
	status := "OK"
	level := 75

	tmpl := Build(VBoxNode{
		Children: []any{
			HBoxNode{
				Children: []any{
					VBoxNode{Children: []any{TextNode{Content: &status}}}.WidthPct(0.5),
					VBoxNode{Children: []any{ProgressNode{Value: &level, BarWidth: 10}}}.WidthPct(0.5),
				},
			},
		},
	})

	buf := NewBuffer(40, 5)

	// Initial render
	tmpl.Execute(buf, 40, 5)
	output1 := buf.String()
	t.Logf("Initial:\n%s", output1)

	if !strings.Contains(output1, "OK") {
		t.Error("Initial output should contain 'OK'")
	}

	// Update values and re-render
	status = "ERROR"
	level = 25

	buf.Clear()
	tmpl.Execute(buf, 40, 5)
	output2 := buf.String()
	t.Logf("Updated:\n%s", output2)

	if !strings.Contains(output2, "ERROR") {
		t.Error("Updated output should contain 'ERROR'")
	}
}

func TestLeaderComponent(t *testing.T) {
	t.Run("static leader renders correctly", func(t *testing.T) {
		tmpl := Build(VBoxNode{
			Children: []any{
				LeaderNode{Label: "CPU", Value: "75%", Width: 20},
				LeaderNode{Label: "MEM", Value: "4.2GB", Width: 20},
			},
		})

		buf := NewBuffer(40, 5)
		tmpl.Execute(buf, 40, 5)

		output := buf.String()
		t.Logf("Leader output:\n%s", output)

		// Should contain label and value connected by dots
		if !strings.Contains(output, "CPU") {
			t.Error("Output should contain 'CPU'")
		}
		if !strings.Contains(output, "75%") {
			t.Error("Output should contain '75%'")
		}
		if !strings.Contains(output, "...") {
			t.Error("Output should contain dots")
		}
	})

	t.Run("pointer binding updates dynamically", func(t *testing.T) {
		value := "PASS"
		tmpl := Build(VBoxNode{
			Children: []any{
				LeaderNode{Label: "STATUS", Value: &value, Width: 25},
			},
		})

		buf := NewBuffer(40, 5)
		tmpl.Execute(buf, 40, 5)

		output1 := buf.String()
		t.Logf("Initial:\n%s", output1)

		if !strings.Contains(output1, "PASS") {
			t.Error("Initial output should contain 'PASS'")
		}

		// Update value and re-render
		value = "FAIL"
		buf.Clear()
		tmpl.Execute(buf, 40, 5)

		output2 := buf.String()
		t.Logf("Updated:\n%s", output2)

		if !strings.Contains(output2, "FAIL") {
			t.Error("Updated output should contain 'FAIL'")
		}
		if strings.Contains(output2, "PASS") {
			t.Error("Updated output should NOT contain 'PASS'")
		}
	})

	t.Run("custom fill character", func(t *testing.T) {
		tmpl := Build(VBoxNode{
			Children: []any{
				LeaderNode{Label: "ITEM", Value: "OK", Width: 15, Fill: '-'},
			},
		})

		buf := NewBuffer(40, 5)
		tmpl.Execute(buf, 40, 5)

		output := buf.String()
		t.Logf("Custom fill output:\n%s", output)

		if !strings.Contains(output, "-") {
			t.Error("Output should contain dash fill")
		}
	})

	t.Run("leader in bordered panel", func(t *testing.T) {
		tmpl := Build(VBoxNode{
			Title: "STATUS",
			Children: []any{
				LeaderNode{Label: "RAM", Value: "PASS", Width: 20},
				LeaderNode{Label: "CPU", Value: "OK", Width: 20},
			},
		}.Border(BorderSingle))

		buf := NewBuffer(30, 6)
		tmpl.Execute(buf, 30, 6)

		output := buf.String()
		t.Logf("Bordered panel:\n%s", output)

		if !strings.Contains(output, "RAM") {
			t.Error("Output should contain 'RAM'")
		}
		if !strings.Contains(output, "STATUS") {
			t.Error("Output should contain panel title 'STATUS'")
		}
	})
}

func TestTableComponent(t *testing.T) {
	t.Run("basic table renders correctly", func(t *testing.T) {
		rows := [][]string{
			{"Alice", "30", "Engineer"},
			{"Bob", "25", "Designer"},
			{"Carol", "35", "Manager"},
		}
		tmpl := Build(VBoxNode{
			Children: []any{
				Table{
					Columns: []TableColumn{
						{Header: "Name", Width: 10},
						{Header: "Age", Width: 5, Align: AlignRight},
						{Header: "Role", Width: 12},
					},
					Rows:       &rows,
					ShowHeader: true,
				},
			},
		})

		buf := NewBuffer(40, 10)
		tmpl.Execute(buf, 40, 10)

		output := buf.String()
		t.Logf("Table output:\n%s", output)

		// Check header
		if !strings.Contains(output, "Name") {
			t.Error("Output should contain header 'Name'")
		}
		if !strings.Contains(output, "Age") {
			t.Error("Output should contain header 'Age'")
		}
		if !strings.Contains(output, "Role") {
			t.Error("Output should contain header 'Role'")
		}

		// Check data
		if !strings.Contains(output, "Alice") {
			t.Error("Output should contain 'Alice'")
		}
		if !strings.Contains(output, "Bob") {
			t.Error("Output should contain 'Bob'")
		}
		if !strings.Contains(output, "Manager") {
			t.Error("Output should contain 'Manager'")
		}
	})

	t.Run("table updates dynamically", func(t *testing.T) {
		rows := [][]string{
			{"Initial", "100"},
		}
		tmpl := Build(Table{
			Columns: []TableColumn{
				{Header: "Status", Width: 15},
				{Header: "Value", Width: 10},
			},
			Rows: &rows,
		})

		buf := NewBuffer(30, 5)
		tmpl.Execute(buf, 30, 5)
		output1 := buf.String()
		t.Logf("Initial:\n%s", output1)

		if !strings.Contains(output1, "Initial") {
			t.Error("Output should contain 'Initial'")
		}

		// Update data
		rows[0] = []string{"Updated", "200"}
		buf.Clear()
		tmpl.Execute(buf, 30, 5)
		output2 := buf.String()
		t.Logf("Updated:\n%s", output2)

		if !strings.Contains(output2, "Updated") {
			t.Error("Output should contain 'Updated'")
		}
		if strings.Contains(output2, "Initial") {
			t.Error("Output should NOT contain 'Initial'")
		}
	})

	t.Run("table with alignment", func(t *testing.T) {
		rows := [][]string{
			{"L", "C", "R"},
		}
		tmpl := Build(Table{
			Columns: []TableColumn{
				{Header: "Left", Width: 10, Align: AlignLeft},
				{Header: "Center", Width: 10, Align: AlignCenter},
				{Header: "Right", Width: 10, Align: AlignRight},
			},
			Rows:       &rows,
			ShowHeader: true,
		})

		buf := NewBuffer(40, 5)
		tmpl.Execute(buf, 40, 5)
		output := buf.String()
		t.Logf("Aligned table:\n%s", output)

		// Just verify it renders without crashing and contains data
		if !strings.Contains(output, "L") {
			t.Error("Output should contain 'L'")
		}
	})
}

func TestSparklineComponent(t *testing.T) {
	t.Run("basic sparkline renders", func(t *testing.T) {
		values := []float64{1, 3, 5, 7, 5, 3, 1, 2, 4, 6, 8}
		tmpl := Build(VBoxNode{
			Children: []any{
				TextNode{Content: "CPU:"},
				SparklineNode{Values: values},
			},
		})

		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		output := buf.String()
		t.Logf("Sparkline output:\n%s", output)

		// Should contain sparkline characters
		if !strings.Contains(output, "CPU:") {
			t.Error("Output should contain 'CPU:'")
		}
		// Check for block characters (any of ▁▂▃▄▅▆▇█)
		hasBlock := false
		for _, r := range output {
			if r >= '▁' && r <= '█' {
				hasBlock = true
				break
			}
		}
		if !hasBlock {
			t.Error("Output should contain block characters")
		}
	})

	t.Run("sparkline updates dynamically", func(t *testing.T) {
		values := []float64{1, 2, 3, 4, 5}
		tmpl := Build(SparklineNode{Values: &values, Width: 10})

		buf := NewBuffer(15, 3)
		tmpl.Execute(buf, 15, 3)
		output1 := buf.String()
		t.Logf("Initial:\n%s", output1)

		// Update values - reverse the trend
		values = []float64{5, 4, 3, 2, 1}
		buf.Clear()
		tmpl.Execute(buf, 15, 3)
		output2 := buf.String()
		t.Logf("Updated:\n%s", output2)

		// Just verify it renders without error
		if output1 == "" || output2 == "" {
			t.Error("Sparkline should produce output")
		}
	})

	t.Run("sparkline with fixed min/max", func(t *testing.T) {
		values := []float64{25, 50, 75}
		tmpl := Build(SparklineNode{
			Values: values,
			Min:    0,
			Max:    100,
		})

		buf := NewBuffer(10, 3)
		tmpl.Execute(buf, 10, 3)
		output := buf.String()
		t.Logf("Fixed range:\n%s", output)

		// Should render without error
		if output == "" {
			t.Error("Sparkline should produce output")
		}
	})
}

func TestHRuleVRuleSpacer(t *testing.T) {
	t.Run("HRule fills width", func(t *testing.T) {
		tmpl := Build(VBoxNode{
			Children: []any{
				TextNode{Content: "Above"},
				HRuleNode{},
				TextNode{Content: "Below"},
			},
		})

		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)
		output := buf.String()
		t.Logf("HRule output:\n%s", output)

		if !strings.Contains(output, "Above") {
			t.Error("Output should contain 'Above'")
		}
		if !strings.Contains(output, "─") {
			t.Error("Output should contain horizontal line character")
		}
		if !strings.Contains(output, "Below") {
			t.Error("Output should contain 'Below'")
		}
	})

	t.Run("HRule custom character", func(t *testing.T) {
		tmpl := Build(VBoxNode{
			Children: []any{
				HRuleNode{Char: '═'},
			},
		})

		buf := NewBuffer(10, 3)
		tmpl.Execute(buf, 10, 3)
		output := buf.String()
		t.Logf("Custom HRule:\n%s", output)

		if !strings.Contains(output, "═") {
			t.Error("Output should contain double line character")
		}
	})

	t.Run("VRule in row with height", func(t *testing.T) {
		tmpl := Build(HBoxNode{
			Children: []any{
				VBoxNode{Children: []any{
					TextNode{Content: "Left1"},
					TextNode{Content: "Left2"},
				}}.WidthPct(0.4),
				VRuleNode{},
				VBoxNode{Children: []any{
					TextNode{Content: "Right1"},
					TextNode{Content: "Right2"},
				}}.WidthPct(0.5),
			},
		})

		buf := NewBuffer(20, 3)
		tmpl.Execute(buf, 20, 3)
		output := buf.String()
		t.Logf("VRule output:\n%s", output)

		if !strings.Contains(output, "Left1") {
			t.Error("Output should contain 'Left1'")
		}
		if !strings.Contains(output, "│") {
			t.Error("Output should contain vertical line character")
		}
		if !strings.Contains(output, "Right1") {
			t.Error("Output should contain 'Right1'")
		}
	})

	t.Run("Spacer creates gap", func(t *testing.T) {
		tmpl := Build(VBoxNode{
			Children: []any{
				TextNode{Content: "Line1"},
				SpacerNode{Height: 2},
				TextNode{Content: "Line4"},
			},
		})

		buf := NewBuffer(20, 6)
		tmpl.Execute(buf, 20, 6)

		// Line1 should be at y=0
		if got := buf.GetLine(0); got != "Line1" {
			t.Errorf("line 0: got %q, want %q", got, "Line1")
		}
		// Line4 should be at y=3 (after 2-line spacer)
		if got := buf.GetLine(3); got != "Line4" {
			t.Errorf("line 3: got %q, want %q", got, "Line4")
		}
	})
}

func TestSpinnerComponent(t *testing.T) {
	t.Run("Spinner renders current frame", func(t *testing.T) {
		frame := 0
		tmpl := Build(HBoxNode{
			Children: []any{
				SpinnerNode{Frame: &frame},
				TextNode{Content: " Loading..."},
			},
		})

		buf := NewBuffer(20, 1)
		tmpl.Execute(buf, 20, 1)
		output := buf.String()
		t.Logf("Spinner frame 0:\n%s", output)

		// Default spinner is SpinnerBraille, frame 0 is "⠋"
		if !strings.Contains(output, "⠋") {
			t.Error("Output should contain first braille spinner frame")
		}
		if !strings.Contains(output, "Loading") {
			t.Error("Output should contain 'Loading'")
		}
	})

	t.Run("Spinner advances frames", func(t *testing.T) {
		frame := 0
		tmpl := Build(SpinnerNode{Frame: &frame})

		buf := NewBuffer(5, 1)

		// Frame 0
		tmpl.Execute(buf, 5, 1)
		if buf.Get(0, 0).Rune != '⠋' {
			t.Errorf("frame 0: got %c, want ⠋", buf.Get(0, 0).Rune)
		}

		// Frame 1
		frame = 1
		buf.Clear()
		tmpl.Execute(buf, 5, 1)
		if buf.Get(0, 0).Rune != '⠙' {
			t.Errorf("frame 1: got %c, want ⠙", buf.Get(0, 0).Rune)
		}
	})

	t.Run("Spinner with custom frames", func(t *testing.T) {
		frame := 0
		tmpl := Build(SpinnerNode{
			Frame:  &frame,
			Frames: SpinnerLine,
		})

		buf := NewBuffer(5, 1)
		tmpl.Execute(buf, 5, 1)

		// SpinnerLine frame 0 is "-"
		if buf.Get(0, 0).Rune != '-' {
			t.Errorf("got %c, want -", buf.Get(0, 0).Rune)
		}
	})

	t.Run("Spinner with dots frames", func(t *testing.T) {
		frame := 0
		tmpl := Build(SpinnerNode{
			Frame:  &frame,
			Frames: SpinnerDots,
		})

		buf := NewBuffer(5, 1)
		tmpl.Execute(buf, 5, 1)

		// SpinnerDots frame 0 is "⣾"
		if buf.Get(0, 0).Rune != '⣾' {
			t.Errorf("got %c, want ⣾", buf.Get(0, 0).Rune)
		}
	})

	t.Run("Spinner wraps frame index", func(t *testing.T) {
		frame := 10 // SpinnerBraille has 10 frames, so this should wrap to 0
		tmpl := Build(SpinnerNode{Frame: &frame})

		buf := NewBuffer(5, 1)
		tmpl.Execute(buf, 5, 1)

		// Should wrap to frame 0
		if buf.Get(0, 0).Rune != '⠋' {
			t.Errorf("wrapped frame: got %c, want ⠋", buf.Get(0, 0).Rune)
		}
	})
}

func TestScrollbarComponent(t *testing.T) {
	t.Run("Vertical scrollbar at top", func(t *testing.T) {
		pos := 0
		tmpl := Build(HBoxNode{
			Children: []any{
				TextNode{Content: "Content"},
				ScrollbarNode{
					ContentSize: 100,
					ViewSize:    10,
					Position:    &pos,
					Length:      10,
				},
			},
		})

		buf := NewBuffer(20, 10)
		tmpl.Execute(buf, 20, 10)

		// Thumb should be at top (position 0)
		// Default thumb char is '█', track is '│'
		if buf.Get(7, 0).Rune != '█' {
			t.Errorf("thumb at pos 0: got %c, want █", buf.Get(7, 0).Rune)
		}
		// Track should be below the thumb
		if buf.Get(7, 9).Rune != '│' {
			t.Errorf("track: got %c, want │", buf.Get(7, 9).Rune)
		}
	})

	t.Run("Vertical scrollbar at bottom", func(t *testing.T) {
		pos := 90 // scrolled to bottom
		tmpl := Build(ScrollbarNode{
			ContentSize: 100,
			ViewSize:    10,
			Position:    &pos,
			Length:      10,
		})

		buf := NewBuffer(5, 10)
		tmpl.Execute(buf, 5, 10)

		// Thumb should be at bottom
		// At pos=90 of range 0-90, thumb should be at track end
		if buf.Get(0, 9).Rune != '█' {
			t.Errorf("thumb at bottom: got %c, want █", buf.Get(0, 9).Rune)
		}
		// Track should be above
		if buf.Get(0, 0).Rune != '│' {
			t.Errorf("track at top: got %c, want │", buf.Get(0, 0).Rune)
		}
	})

	t.Run("Horizontal scrollbar", func(t *testing.T) {
		pos := 0
		tmpl := Build(ScrollbarNode{
			ContentSize: 100,
			ViewSize:    10,
			Position:    &pos,
			Length:      10,
			Horizontal:  true,
		})

		buf := NewBuffer(10, 3)
		tmpl.Execute(buf, 10, 3)

		// Thumb should be at left (position 0)
		if buf.Get(0, 0).Rune != '█' {
			t.Errorf("thumb at left: got %c, want █", buf.Get(0, 0).Rune)
		}
		// Track should be to the right
		if buf.Get(9, 0).Rune != '─' {
			t.Errorf("track at right: got %c, want ─", buf.Get(9, 0).Rune)
		}
	})

	t.Run("Scrollbar thumb moves with position", func(t *testing.T) {
		pos := 0
		tmpl := Build(ScrollbarNode{
			ContentSize: 100,
			ViewSize:    10,
			Position:    &pos,
			Length:      10,
		})

		buf := NewBuffer(5, 10)

		// At top
		pos = 0
		tmpl.Execute(buf, 5, 10)
		topThumbPos := findThumbPosition(buf, 0, 10, false)
		if topThumbPos != 0 {
			t.Errorf("thumb at top: got position %d, want 0", topThumbPos)
		}

		// At middle
		buf.Clear()
		pos = 45 // middle of scroll range
		tmpl.Execute(buf, 5, 10)
		midThumbPos := findThumbPosition(buf, 0, 10, false)
		if midThumbPos <= 0 || midThumbPos >= 9 {
			t.Errorf("thumb at middle: got position %d, expected between 1-8", midThumbPos)
		}

		// At bottom
		buf.Clear()
		pos = 90
		tmpl.Execute(buf, 5, 10)
		bottomThumbPos := findThumbPosition(buf, 0, 10, false)
		if bottomThumbPos != 9 {
			t.Errorf("thumb at bottom: got position %d, want 9", bottomThumbPos)
		}
	})

	t.Run("Custom scrollbar characters", func(t *testing.T) {
		pos := 0
		tmpl := Build(ScrollbarNode{
			ContentSize: 20,
			ViewSize:    5,
			Position:    &pos,
			Length:      4,
			TrackChar:   '░',
			ThumbChar:   '▓',
		})

		buf := NewBuffer(5, 4)
		tmpl.Execute(buf, 5, 4)

		// Check custom characters
		if buf.Get(0, 0).Rune != '▓' {
			t.Errorf("custom thumb: got %c, want ▓", buf.Get(0, 0).Rune)
		}
		if buf.Get(0, 3).Rune != '░' {
			t.Errorf("custom track: got %c, want ░", buf.Get(0, 3).Rune)
		}
	})
}

// findThumbPosition finds the Y position (for vertical) or X position (for horizontal) of the thumb
func findThumbPosition(buf *Buffer, x, length int, horizontal bool) int {
	for i := 0; i < length; i++ {
		var char rune
		if horizontal {
			char = buf.Get(i, 0).Rune
		} else {
			char = buf.Get(x, i).Rune
		}
		if char == '█' || char == '▓' {
			return i
		}
	}
	return -1
}

func TestTabsComponent(t *testing.T) {
	t.Run("Tabs with underline style", func(t *testing.T) {
		selected := 0
		tmpl := Build(TabsNode{
			Labels:   []string{"Home", "Settings", "Help"},
			Selected: &selected,
		})

		buf := NewBuffer(30, 3)
		tmpl.Execute(buf, 30, 3)
		output := buf.String()
		t.Logf("Tabs underline:\n%s", output)

		// Check labels are present
		if !strings.Contains(output, "Home") {
			t.Error("Output should contain 'Home'")
		}
		if !strings.Contains(output, "Settings") {
			t.Error("Output should contain 'Settings'")
		}
		if !strings.Contains(output, "Help") {
			t.Error("Output should contain 'Help'")
		}

		// First tab should have underline attribute (active)
		cell := buf.Get(0, 0)
		if cell.Rune != 'H' {
			t.Errorf("First char: got %c, want H", cell.Rune)
		}
		if !cell.Style.Attr.Has(AttrUnderline) {
			t.Error("First tab should be underlined (active)")
		}
	})

	t.Run("Tabs selection changes", func(t *testing.T) {
		selected := 1 // Select "Settings"
		tmpl := Build(TabsNode{
			Labels:   []string{"Home", "Settings"},
			Selected: &selected,
		})

		buf := NewBuffer(20, 3)
		tmpl.Execute(buf, 20, 3)

		// "Home" should NOT be underlined (gap=2 means "Settings" starts at position 6)
		if buf.Get(0, 0).Style.Attr.Has(AttrUnderline) {
			t.Error("First tab should NOT be underlined when not selected")
		}
		// "Settings" should be underlined
		if !buf.Get(6, 0).Style.Attr.Has(AttrUnderline) {
			t.Error("Second tab should be underlined when selected")
		}
	})

	t.Run("Tabs with bracket style", func(t *testing.T) {
		selected := 0
		tmpl := Build(TabsNode{
			Labels:   []string{"Tab1", "Tab2"},
			Selected: &selected,
			Style:    TabsStyleBracket,
		})

		buf := NewBuffer(20, 3)
		tmpl.Execute(buf, 20, 3)
		output := buf.String()
		t.Logf("Tabs bracket:\n%s", output)

		// Check bracket characters
		if !strings.Contains(output, "[Tab1]") {
			t.Error("Output should contain '[Tab1]'")
		}
		if !strings.Contains(output, "[Tab2]") {
			t.Error("Output should contain '[Tab2]'")
		}
	})

	t.Run("Tabs with box style", func(t *testing.T) {
		selected := 0
		tmpl := Build(TabsNode{
			Labels:   []string{"One", "Two"},
			Selected: &selected,
			Style:    TabsStyleBox,
		})

		buf := NewBuffer(30, 5)
		tmpl.Execute(buf, 30, 5)
		output := buf.String()
		t.Logf("Tabs box:\n%s", output)

		// Check for box characters
		if !strings.Contains(output, "┌") {
			t.Error("Output should contain box corner ┌")
		}
		if !strings.Contains(output, "│") {
			t.Error("Output should contain box side │")
		}
		if !strings.Contains(output, "One") {
			t.Error("Output should contain 'One'")
		}
	})

	t.Run("Tabs with custom gap", func(t *testing.T) {
		selected := 0
		tmpl := Build(TabsNode{
			Labels:   []string{"A", "B"},
			Selected: &selected,
			Gap:      5,
		})

		buf := NewBuffer(20, 3)
		tmpl.Execute(buf, 20, 3)

		// "A" at 0, gap of 5, "B" at 6
		if buf.Get(0, 0).Rune != 'A' {
			t.Errorf("First tab: got %c, want A", buf.Get(0, 0).Rune)
		}
		if buf.Get(6, 0).Rune != 'B' {
			t.Errorf("Second tab at pos 6: got %c, want B", buf.Get(6, 0).Rune)
		}
	})

	t.Run("Tabs with styling", func(t *testing.T) {
		selected := 0
		tmpl := Build(TabsNode{
			Labels:        []string{"Active", "Inactive"},
			Selected:      &selected,
			ActiveStyle:   Style{FG: Green},
			InactiveStyle: Style{FG: White},
		})

		buf := NewBuffer(25, 3)
		tmpl.Execute(buf, 25, 3)

		// Active tab should have green FG
		if buf.Get(0, 0).Style.FG != Green {
			t.Error("Active tab should have green foreground")
		}
	})
}

func TestTreeViewComponent(t *testing.T) {
	t.Run("TreeView renders expanded tree", func(t *testing.T) {
		tree := &TreeNode{
			Label:    "Root",
			Expanded: true,
			Children: []*TreeNode{
				{Label: "Child 1"},
				{Label: "Child 2"},
			},
		}
		tmpl := Build(TreeView{
			Root:     tree,
			ShowRoot: true,
		})

		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)
		output := buf.String()
		t.Logf("TreeView:\n%s", output)

		if !strings.Contains(output, "Root") {
			t.Error("Output should contain 'Root'")
		}
		if !strings.Contains(output, "Child 1") {
			t.Error("Output should contain 'Child 1'")
		}
		if !strings.Contains(output, "Child 2") {
			t.Error("Output should contain 'Child 2'")
		}
	})

	t.Run("TreeView collapsed hides children", func(t *testing.T) {
		tree := &TreeNode{
			Label:    "Root",
			Expanded: false, // collapsed
			Children: []*TreeNode{
				{Label: "Hidden"},
			},
		}
		tmpl := Build(TreeView{
			Root:     tree,
			ShowRoot: true,
		})

		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)
		output := buf.String()
		t.Logf("TreeView collapsed:\n%s", output)

		if !strings.Contains(output, "Root") {
			t.Error("Output should contain 'Root'")
		}
		if strings.Contains(output, "Hidden") {
			t.Error("Output should NOT contain 'Hidden' when collapsed")
		}
		// Should show collapsed indicator
		if !strings.Contains(output, "▶") {
			t.Error("Output should contain collapsed indicator ▶")
		}
	})

	t.Run("TreeView shows expand indicator", func(t *testing.T) {
		tree := &TreeNode{
			Label:    "Parent",
			Expanded: true,
			Children: []*TreeNode{
				{Label: "Child"},
			},
		}
		tmpl := Build(TreeView{
			Root:     tree,
			ShowRoot: true,
		})

		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)
		output := buf.String()

		// Should show expanded indicator
		if !strings.Contains(output, "▼") {
			t.Error("Output should contain expanded indicator ▼")
		}
	})

	t.Run("TreeView without root", func(t *testing.T) {
		tree := &TreeNode{
			Label:    "HiddenRoot",
			Expanded: true,
			Children: []*TreeNode{
				{Label: "Child 1"},
				{Label: "Child 2"},
			},
		}
		tmpl := Build(TreeView{
			Root:     tree,
			ShowRoot: false, // don't show root
		})

		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)
		output := buf.String()
		t.Logf("TreeView no root:\n%s", output)

		if strings.Contains(output, "HiddenRoot") {
			t.Error("Output should NOT contain 'HiddenRoot' when ShowRoot=false")
		}
		if !strings.Contains(output, "Child 1") {
			t.Error("Output should contain 'Child 1'")
		}
	})

	t.Run("TreeView with nested levels", func(t *testing.T) {
		tree := &TreeNode{
			Label:    "Root",
			Expanded: true,
			Children: []*TreeNode{
				{
					Label:    "Level 1",
					Expanded: true,
					Children: []*TreeNode{
						{
							Label:    "Level 2",
							Expanded: true,
							Children: []*TreeNode{
								{Label: "Level 3"},
							},
						},
					},
				},
			},
		}
		tmpl := Build(TreeView{
			Root:     tree,
			ShowRoot: true,
			Indent:   2,
		})

		buf := NewBuffer(30, 10)
		tmpl.Execute(buf, 30, 10)
		output := buf.String()
		t.Logf("TreeView nested:\n%s", output)

		if !strings.Contains(output, "Level 3") {
			t.Error("Output should contain 'Level 3'")
		}
	})

	t.Run("TreeView with custom characters", func(t *testing.T) {
		tree := &TreeNode{
			Label:    "Root",
			Expanded: true,
			Children: []*TreeNode{
				{Label: "Leaf"},
			},
		}
		tmpl := Build(TreeView{
			Root:          tree,
			ShowRoot:      true,
			ExpandedChar:  '-',
			CollapsedChar: '+',
			LeafChar:      '*',
		})

		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)
		output := buf.String()
		t.Logf("TreeView custom chars:\n%s", output)

		if !strings.Contains(output, "-") {
			t.Error("Output should contain expanded indicator '-'")
		}
		if !strings.Contains(output, "*") {
			t.Error("Output should contain leaf indicator '*'")
		}
	})
}
