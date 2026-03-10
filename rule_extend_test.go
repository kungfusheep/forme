package glyph

import (
	"testing"
)

// TestMultiVRuleLayout tests that multiple VRules in an HBox produce correct ┼ junctions.
func TestMultiVRuleLayout(t *testing.T) {
	view := VBox.Border(BorderSingle).FitContent()(
		Text("HEADER").Align(AlignCenter),
		HRule().Extend(),
		HBox.MarginVH(0, 1).Gap(1)(
			VBox.Width(8)(
				Text("A"),
				HRule().Extend(),
				Text("B"),
			),
			VRule().Extend(),
			VBox.Width(8)(
				Text("C"),
				HRule().Extend(),
				Text("D"),
			),
			VRule().Extend(),
			VBox(
				Text("E"),
				HRule().Extend(),
				Text("F"),
			),
		),
	)

	tmpl := Build(view)
	buf := NewBuffer(60, 10)
	tmpl.Execute(buf, 60, 10)

	t.Log("Multi-VRule layout output:")
	for y := 0; y < 10; y++ {
		line := buf.GetLine(y)
		if line != "" {
			t.Logf("%2d: %s", y, line)
		}
	}

	// scan all rows for ┼ junctions (inner HRule row position depends on layout)
	junctionCount := 0
	for y := 0; y < 10; y++ {
		for x := 0; x < 60; x++ {
			if buf.Get(x, y).Rune == '┼' {
				junctionCount++
				t.Logf("found ┼ at x=%d y=%d", x, y)
			}
		}
	}
	if junctionCount < 2 {
		t.Errorf("expected at least 2 ┼ junctions, got %d", junctionCount)
	}
}

// TestLeadingTrailingVRuleLayout tests the about-me pattern: VRules before and after
// all containers, verifying that border extensions still detect the actual leftmost/rightmost
// containers (not the leading/trailing VRules).
func TestLeadingTrailingVRuleLayout(t *testing.T) {
	view := VBox.Border(BorderSingle).FitContent()(
		Text("TITLE").Align(AlignCenter),
		HRule().Extend(),
		HBox.MarginVH(0, 1).Gap(1)(
			VRule().Extend(),
			VBox.Width(8)(
				Text("KEY"),
				HRule().Extend(),
				Text("KEY2"),
			),
			VRule().Extend(),
			VBox(
				Text("VALUE"),
				HRule().Extend(),
				Text("VALUE2"),
			),
			VRule().Extend(),
		),
	)

	tmpl := Build(view)
	buf := NewBuffer(60, 10)
	tmpl.Execute(buf, 60, 10)

	t.Log("Leading/trailing VRule layout output:")
	for y := 0; y < 10; y++ {
		line := buf.GetLine(y)
		if line != "" {
			t.Logf("%2d: %s", y, line)
		}
	}

	// expect: ├ on the left border, ┤ on the right border, ┼ between containers
	leftFound, rightFound, crossFound := false, false, false
	for y := 0; y < 10; y++ {
		for x := 0; x < 60; x++ {
			switch buf.Get(x, y).Rune {
			case '├':
				leftFound = true
				t.Logf("found ├ at x=%d y=%d", x, y)
			case '┤':
				rightFound = true
				t.Logf("found ┤ at x=%d y=%d", x, y)
			case '┼':
				crossFound = true
				t.Logf("found ┼ at x=%d y=%d", x, y)
			}
		}
	}
	if !leftFound {
		t.Error("no ├ found (HRule not reaching left border through leading VRule)")
	}
	if !rightFound {
		t.Error("no ┤ found (HRule not reaching right border through trailing VRule)")
	}
	if !crossFound {
		t.Error("no ┼ found (HRule not meeting inner VRule)")
	}
}

// TestAboutMeLayout tests the exact about-me layout with leading/trailing VRules
// and double HRule, matching the cmd/about-me main.go structure.
func TestAboutMeLayout(t *testing.T) {
	view := VBox.Border(BorderSingle).FitContent()(
		HRule().Extend(),
		Text("GLYPH FRAMEWORK").Align(AlignCenter),
		Text("DIAGNOSTIC REPORT").Align(AlignCenter),
		HRule().Extend(),
		HRule().Extend(),
		HBox.MarginVH(0, 1).Gap(1)(
			VRule().Extend(),
			VBox.Width(11)(
				Text("GO"), Text("OS"), Text("ARCH"),
				HRule().Extend(),
				Text("TERMINAL"), Text("SHELL"), Text("COLOUR"), Text("WIDTH"),
				HRule().Extend(),
				Text("GLYPH"), Text("RENDER"),
			),
			VRule().Extend(),
			VBox(
				Text("go1.23.4"), Text("darwin"), Text("arm64"),
				HRule().Extend(),
				Text("iTerm.app"), Text("zsh"), Text("truecolor"), Text("80c"),
				HRule().Extend(),
				Text("v0.1.0"), Text("OK"),
			),
			VRule().Extend(),
		),
	)

	tmpl := Build(view)
	buf := NewBuffer(60, 25)
	tmpl.Execute(buf, 60, 25)

	t.Log("About-me layout output:")
	for y := 0; y < 25; y++ {
		line := buf.GetLine(y)
		if line != "" {
			t.Logf("%2d: %s", y, line)
		}
	}

	// check no character is set outside the box (above y=0 is impossible, below bottom border is blank)
	// verify all rows before the box top border are blank (should be none since box starts at y=0)
	// verify correct junction characters exist
	topBorderFound := false
	for x := 0; x < 60; x++ {
		c := buf.Get(x, 0)
		if c.Rune == '┌' || c.Rune == '─' || c.Rune == '┐' {
			topBorderFound = true
		}
	}
	if !topBorderFound {
		t.Error("no top border at y=0 (box not anchored at row 0)")
	}

	// check for ├/┤ at the HRule rows (border junctions)
	var leftFound, rightFound bool
	for y := 0; y < 25; y++ {
		if buf.Get(0, y).Rune == '├' {
			leftFound = true
			t.Logf("found ├ at x=0 y=%d", y)
		}
		for x := 0; x < 60; x++ {
			if buf.Get(x, y).Rune == '┤' {
				rightFound = true
				t.Logf("found ┤ at x=%d y=%d", x, y)
			}
		}
	}
	if !leftFound {
		t.Error("no ├ at left border")
	}
	if !rightFound {
		t.Error("no ┤ at right border")
	}

	// check ┬/┴ junctions for the leading/trailing VRules at border rows
	var topJunctionFound, botJunctionFound bool
	for y := 0; y < 25; y++ {
		for x := 0; x < 60; x++ {
			r := buf.Get(x, y).Rune
			if r == '┬' {
				topJunctionFound = true
				t.Logf("found ┬ at x=%d y=%d", x, y)
			}
			if r == '┴' {
				botJunctionFound = true
				t.Logf("found ┴ at x=%d y=%d", x, y)
			}
		}
	}
	if !topJunctionFound {
		t.Error("no ┬ junction (VRules not connecting to top HRule)")
	}
	if !botJunctionFound {
		t.Error("no ┴ junction (VRules not connecting to bottom border)")
	}
}

// TestDiagnosticLayout renders the about-me diagnostic report layout and dumps
// the buffer so we can visually inspect the border/rule junctions.
func TestDiagnosticLayout(t *testing.T) {
	view := VBox.Border(BorderSingle).FitContent()(
		Text("GLYPH FRAMEWORK").Align(AlignCenter),
		Text("DIAGNOSTIC REPORT").Align(AlignCenter),
		HRule().Extend(),
		HBox.MarginVH(0, 1).Gap(1)(
			VBox.Width(11)(
				Text("GO"), Text("OS"), Text("ARCH"),
				HRule().Extend(),
				Text("TERMINAL"), Text("SHELL"), Text("COLOUR"), Text("WIDTH"),
				HRule().Extend(),
				Text("GLYPH"), Text("RENDER"),
			),
			VRule().Extend(),
			VBox(
				Text("go1.23.4"), Text("darwin"), Text("arm64"),
				HRule().Extend(),
				Text("iTerm.app"), Text("zsh"), Text("truecolor"), Text("██████░░  80c"),
				HRule().Extend(),
				Text("v0.1.0"), Text("OK"),
			),
		),
	)

	tmpl := Build(view)
	buf := NewBuffer(80, 20)
	tmpl.Execute(buf, 80, 20)

	t.Log("Diagnostic layout output:")
	for y := 0; y < 20; y++ {
		line := buf.GetLine(y)
		if line != "" {
			t.Logf("%2d: %s", y, line)
		}
	}


	// Check for ┬ (VRule meets top separator HRule) at row 3
	topJunctionFound := false
	for x := 0; x < 80; x++ {
		c := buf.Get(x, 3)
		if c.Rune == '┬' {
			topJunctionFound = true
			t.Logf("found ┬ junction at x=%d y=3", x)
		}
	}
	if !topJunctionFound {
		t.Error("no ┬ junction at row 3 (VRule top extension not meeting HRule separator)")
	}

	// Check for ├ (HRule meets left border) somewhere
	leftJunctionFound := false
	for y := 0; y < 20; y++ {
		c := buf.Get(0, y)
		if c.Rune == '├' {
			leftJunctionFound = true
			t.Logf("found ├ at x=0 y=%d", y)
		}
	}
	if !leftJunctionFound {
		t.Error("no ├ junction at left border (HRule.Extend() not reaching left border)")
	}

	// Check for ┤ (HRule meets right border)
	rightJunctionFound := false
	for y := 0; y < 20; y++ {
		for x := 0; x < 80; x++ {
			c := buf.Get(x, y)
			if c.Rune == '┤' {
				rightJunctionFound = true
				t.Logf("found ┤ at x=%d y=%d", x, y)
			}
		}
	}
	if !rightJunctionFound {
		t.Error("no ┤ junction (HRule.Extend() not reaching right border)")
	}

	// Check for ┴ (VRule meets bottom border)
	botJunctionFound := false
	for x := 0; x < 80; x++ {
		for y := 0; y < 20; y++ {
			c := buf.Get(x, y)
			if c.Rune == '┴' {
				botJunctionFound = true
				t.Logf("found ┴ at x=%d y=%d", x, y)
			}
		}
	}
	if !botJunctionFound {
		t.Error("no ┴ junction (VRule bottom extension not meeting bottom border)")
	}
}
