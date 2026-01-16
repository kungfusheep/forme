package tui

import (
	"strings"
	"testing"
)

func TestSerialFlexPercentWidth(t *testing.T) {
	// Test that PercentWidth distributes space correctly in a Row
	tmpl := BuildSerial(Row{
		Children: []any{
			Col{Children: []any{Text{Content: "Left"}}}.WidthPct(0.5),
			Col{Children: []any{Text{Content: "Right"}}}.WidthPct(0.5),
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
	tmpl := BuildSerial(Col{
		Children: []any{
			Text{Content: "Header"}, // H=1
			Col{Children: []any{Text{Content: "Content"}}}.Grow(1),
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
	tmpl := BuildSerial(Col{
		Title: "Panel",
		Children: []any{
			Text{Content: "Inside"},
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
	tmpl := BuildSerial(Col{
		Children: []any{
			Text{Content: "Line 1"},
			Text{Content: "Line 2"},
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
	tmpl := BuildSerial(Col{
		Children: []any{
			Row{
				Children: []any{
					Col{
						Title:    "Left",
						Children: []any{Text{Content: "L1"}},
					}.WidthPct(0.5).Border(BorderSingle),
					Col{
						Title:    "Right",
						Children: []any{Text{Content: "R1"}},
					}.WidthPct(0.5).Border(BorderSingle),
				},
			},
			Col{
				Title:    "Log",
				Children: []any{Text{Content: "Log entry"}},
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

	tmpl := BuildSerial(Col{
		Children: []any{
			Row{
				Children: []any{
					Col{Children: []any{Text{Content: &status}}}.WidthPct(0.5),
					Col{Children: []any{Progress{Value: &level, BarWidth: 10}}}.WidthPct(0.5),
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
		tmpl := BuildSerial(Col{
			Children: []any{
				Leader{Label: "CPU", Value: "75%", Width: 20},
				Leader{Label: "MEM", Value: "4.2GB", Width: 20},
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
		tmpl := BuildSerial(Col{
			Children: []any{
				Leader{Label: "STATUS", Value: &value, Width: 25},
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
		tmpl := BuildSerial(Col{
			Children: []any{
				Leader{Label: "ITEM", Value: "OK", Width: 15, Fill: '-'},
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
		tmpl := BuildSerial(Col{
			Title: "STATUS",
			Children: []any{
				Leader{Label: "RAM", Value: "PASS", Width: 20},
				Leader{Label: "CPU", Value: "OK", Width: 20},
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
