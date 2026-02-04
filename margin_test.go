package forme

import (
	"testing"
)

func TestMarginUniform_VBox(t *testing.T) {
	// VBox with margin=1 around "Hello"
	// On a 20x5 buffer, content should appear at (1,1) instead of (0,0)
	tmpl := Build(VBox.Margin(1)(
		Text("Hello"),
	))

	buf := NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	// Row 0 should be empty (top margin)
	if got := buf.GetLine(0); got != "" {
		t.Errorf("line 0 (top margin): got %q, want empty", got)
	}
	// Row 1 should have "Hello" starting at column 1 (left margin)
	if got := buf.GetLine(1); got != " Hello" {
		t.Errorf("line 1: got %q, want %q", got, " Hello")
	}
}

func TestMarginUniform_HBox(t *testing.T) {
	tmpl := Build(HBox.Margin(1)(
		Text("AB"),
	))

	buf := NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	if got := buf.GetLine(0); got != "" {
		t.Errorf("line 0 (top margin): got %q, want empty", got)
	}
	if got := buf.GetLine(1); got != " AB" {
		t.Errorf("line 1: got %q, want %q", got, " AB")
	}
}

func TestMarginXY(t *testing.T) {
	// vertical=0, horizontal=3
	tmpl := Build(VBox.MarginXY(0, 3)(
		Text("Hi"),
	))

	buf := NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	// No vertical margin, so content on row 0, but offset by 3 cols
	if got := buf.GetLine(0); got != "   Hi" {
		t.Errorf("line 0: got %q, want %q", got, "   Hi")
	}
}

func TestMarginTRBL(t *testing.T) {
	// top=2, right=0, bottom=0, left=4
	tmpl := Build(VBox.MarginTRBL(2, 0, 0, 4)(
		Text("X"),
	))

	buf := NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	// Rows 0-1 empty (top margin = 2)
	for y := 0; y < 2; y++ {
		if got := buf.GetLine(y); got != "" {
			t.Errorf("line %d (top margin): got %q, want empty", y, got)
		}
	}
	// Row 2 should have "X" starting at column 4
	if got := buf.GetLine(2); got != "    X" {
		t.Errorf("line 2: got %q, want %q", got, "    X")
	}
}

func TestMarginWithBorder(t *testing.T) {
	// Margin outside border: margin=1, then border, then content
	tmpl := Build(VBox.Border(BorderSingle).Margin(1).Width(10)(
		Text("Hi"),
	))

	buf := NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	// Row 0 empty (top margin)
	if got := buf.GetLine(0); got != "" {
		t.Errorf("line 0 (top margin): got %q, want empty", got)
	}

	// Row 1 should have the border, starting at column 1 (left margin)
	line1 := buf.GetLine(1)
	if len(line1) == 0 || line1[0] != ' ' {
		t.Errorf("line 1: expected leading space for left margin, got %q", line1)
	}

	// Check that border character appears at position (1, 1)
	cell := buf.Get(1, 1)
	if cell.Rune != BorderSingle.TopLeft {
		t.Errorf("cell (1,1): got rune %q, want border top-left %q", cell.Rune, BorderSingle.TopLeft)
	}

	// Check that content "Hi" appears inside the border, offset by margin+border
	// margin left=1, border left=1, so content starts at column 2
	line2 := buf.GetLine(2)
	if len(line2) < 4 {
		t.Fatalf("line 2 too short: got %q", line2)
	}
	// cell at (2, 2) should be 'H'
	hCell := buf.Get(2, 2)
	if hCell.Rune != 'H' {
		t.Errorf("cell (2,2): got rune %q, want 'H'", hCell.Rune)
	}
}

func TestMarginWithFill(t *testing.T) {
	// Fill should only apply inside the margin, not in the moat
	tmpl := Build(VBox.Fill(Red).Margin(1).Width(6).Height(4)(
		Text("X"),
	))

	buf := NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	// Margin area (0,0) should have default BG (moat is transparent)
	marginCell := buf.Get(0, 0)
	if marginCell.Style.BG.Mode != ColorDefault {
		t.Errorf("margin cell (0,0): expected default BG, got %+v", marginCell.Style.BG)
	}

	// Empty cell inside the box (not where text renders) should have Red BG
	// Text "X" renders at (1,1), so check (2,1) which is an empty filled cell
	fillCell := buf.Get(2, 1)
	if fillCell.Style.BG != Red {
		t.Errorf("fill cell (2,1): expected Red BG, got %+v", fillCell.Style.BG)
	}

	// Row 2 is entirely fill (no text content), check (1,2)
	fillCell2 := buf.Get(1, 2)
	if fillCell2.Style.BG != Red {
		t.Errorf("fill cell (1,2): expected Red BG, got %+v", fillCell2.Style.BG)
	}

	// Outside the box on the right (column 5 = right margin) should be transparent
	rightMargin := buf.Get(5, 1)
	if rightMargin.Style.BG.Mode != ColorDefault {
		t.Errorf("right margin cell (5,1): expected default BG, got %+v", rightMargin.Style.BG)
	}
}

func TestMarginLayoutHeight(t *testing.T) {
	// VBox with margin=1 containing one text line.
	// Total height: 1 (top) + 1 (content) + 1 (bottom) = 3
	tmpl := Build(VBox.FitContent().Margin(1)(
		Text("A"),
	))

	buf := NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	// Geom height should be 3 (1+1+1)
	geom := tmpl.geom[0]
	if geom.H != 3 {
		t.Errorf("container height: got %d, want 3", geom.H)
	}
}

func TestMarginVBoxMultiChild(t *testing.T) {
	// margin around a VBox with two children
	tmpl := Build(VBox.Margin(1)(
		Text("One"),
		Text("Two"),
	))

	buf := NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	if got := buf.GetLine(0); got != "" {
		t.Errorf("line 0 (top margin): got %q, want empty", got)
	}
	if got := buf.GetLine(1); got != " One" {
		t.Errorf("line 1: got %q, want %q", got, " One")
	}
	if got := buf.GetLine(2); got != " Two" {
		t.Errorf("line 2: got %q, want %q", got, " Two")
	}
}

func TestMarginNestedContainers(t *testing.T) {
	// Outer VBox has margin=1, inner VBox has margin=1
	// Content should be at (2, 2) — double offset
	tmpl := Build(VBox.Margin(1)(
		VBox.Margin(1)(
			Text("Deep"),
		),
	))

	buf := NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	// Rows 0-1 should be empty
	for y := 0; y < 2; y++ {
		if got := buf.GetLine(y); got != "" {
			t.Errorf("line %d: got %q, want empty", y, got)
		}
	}
	// Row 2 should have "Deep" at column 2
	if got := buf.GetLine(2); got != "  Deep" {
		t.Errorf("line 2: got %q, want %q", got, "  Deep")
	}
}

func TestMarginVBoxNode(t *testing.T) {
	// Test the struct-based API too
	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Hello"},
	}}.Margin(1))

	buf := NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	if got := buf.GetLine(0); got != "" {
		t.Errorf("line 0 (top margin): got %q, want empty", got)
	}
	if got := buf.GetLine(1); got != " Hello" {
		t.Errorf("line 1: got %q, want %q", got, " Hello")
	}
}

func TestMarginHBoxNode(t *testing.T) {
	tmpl := Build(HBoxNode{Children: []any{
		TextNode{Content: "AB"},
	}}.Margin(1))

	buf := NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	if got := buf.GetLine(0); got != "" {
		t.Errorf("line 0 (top margin): got %q, want empty", got)
	}
	if got := buf.GetLine(1); got != " AB" {
		t.Errorf("line 1: got %q, want %q", got, " AB")
	}
}

func TestMarginZero(t *testing.T) {
	// Zero margin should be identical to no margin
	tmpl := Build(VBox.Margin(0)(
		Text("Same"),
	))

	buf := NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	if got := buf.GetLine(0); got != "Same" {
		t.Errorf("line 0: got %q, want %q", got, "Same")
	}
}

func TestMarginAsymmetric(t *testing.T) {
	// top=0, right=0, bottom=0, left=5
	tmpl := Build(VBox.MarginTRBL(0, 0, 0, 5)(
		Text("Hi"),
	))

	buf := NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	if got := buf.GetLine(0); got != "     Hi" {
		t.Errorf("line 0: got %q, want %q", got, "     Hi")
	}
}

func TestMarginIntrinsicWidth(t *testing.T) {
	// FitContent with margin — intrinsic width should include margin
	tmpl := Build(VBox.FitContent().Margin(1)(
		Text("ABC"),
	))

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// Width: 1 (left) + 3 (content) + 1 (right) = 5
	geom := tmpl.geom[0]
	if geom.W != 5 {
		t.Errorf("container width: got %d, want 5", geom.W)
	}
}

// --- Leaf margin tests ---

func TestMarginText(t *testing.T) {
	// Text with margin=1 inside a VBox
	// The text itself should render offset by (1,1)
	tmpl := Build(VBox(
		Text("Hi").Margin(1),
	))

	buf := NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	// Row 0 empty (top margin of text)
	if got := buf.GetLine(0); got != "" {
		t.Errorf("line 0 (text top margin): got %q, want empty", got)
	}
	// Row 1: left margin (1 space) + "Hi"
	if got := buf.GetLine(1); got != " Hi" {
		t.Errorf("line 1: got %q, want %q", got, " Hi")
	}
}

func TestMarginTextLayout(t *testing.T) {
	// Text with margin inside a FitContent VBox should expand the container
	tmpl := Build(VBox.FitContent()(
		Text("AB").Margin(1),
	))

	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	// Container width: 1 (left) + 2 (text) + 1 (right) = 4
	geom := tmpl.geom[0]
	if geom.W != 4 {
		t.Errorf("container width: got %d, want 4", geom.W)
	}
	// Container height: 1 (top) + 1 (text) + 1 (bottom) = 3
	if geom.H != 3 {
		t.Errorf("container height: got %d, want 3", geom.H)
	}
}

func TestMarginTextAsymmetric(t *testing.T) {
	// top=0, right=0, bottom=0, left=3
	tmpl := Build(VBox(
		Text("X").MarginTRBL(0, 0, 0, 3),
	))

	buf := NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	if got := buf.GetLine(0); got != "   X" {
		t.Errorf("line 0: got %q, want %q", got, "   X")
	}
}

func TestMarginTextXY(t *testing.T) {
	// vertical=2, horizontal=0 — text pushed down by 2 rows
	tmpl := Build(VBox(
		Text("Go").MarginXY(2, 0),
	))

	buf := NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	for y := 0; y < 2; y++ {
		if got := buf.GetLine(y); got != "" {
			t.Errorf("line %d (top margin): got %q, want empty", y, got)
		}
	}
	if got := buf.GetLine(2); got != "Go" {
		t.Errorf("line 2: got %q, want %q", got, "Go")
	}
}

func TestMarginHRule(t *testing.T) {
	// HRule with left margin=2 should render the rule starting at column 2
	tmpl := Build(VBox.Width(10)(
		HRule().MarginTRBL(1, 0, 0, 2),
	))

	buf := NewBuffer(10, 5)
	tmpl.Execute(buf, 10, 5)

	// Row 0 should be empty (top margin)
	if got := buf.GetLine(0); got != "" {
		t.Errorf("line 0 (hrule top margin): got %q, want empty", got)
	}

	// Row 1: 2 spaces + rule chars filling remaining width
	cell := buf.Get(0, 1)
	if cell.Rune != 0 && cell.Rune != ' ' {
		t.Errorf("cell (0,1): expected space/empty in left margin, got %q", cell.Rune)
	}
	cell = buf.Get(2, 1)
	if cell.Rune != '─' {
		t.Errorf("cell (2,1): expected '─', got %q", cell.Rune)
	}
}

func TestMarginMultipleTextsInVBox(t *testing.T) {
	// Two texts, second one has margin — should push it down and indent it
	tmpl := Build(VBox(
		Text("One"),
		Text("Two").MarginTRBL(1, 0, 0, 2),
	))

	buf := NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	if got := buf.GetLine(0); got != "One" {
		t.Errorf("line 0: got %q, want %q", got, "One")
	}
	// Row 1 should be empty (top margin of "Two")
	if got := buf.GetLine(1); got != "" {
		t.Errorf("line 1 (top margin): got %q, want empty", got)
	}
	// Row 2: left margin (2 spaces) + "Two"
	if got := buf.GetLine(2); got != "  Two" {
		t.Errorf("line 2: got %q, want %q", got, "  Two")
	}
}

func TestMarginTextInHBox(t *testing.T) {
	// Text with left margin inside HBox — should offset horizontally
	tmpl := Build(HBox(
		Text("A"),
		Text("B").MarginTRBL(0, 0, 0, 2),
	))

	buf := NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	// "A" at column 0, then 2 spaces margin, then "B"
	if got := buf.GetLine(0); got != "A  B" {
		t.Errorf("line 0: got %q, want %q", got, "A  B")
	}
}

func TestMarginLeaderWidth(t *testing.T) {
	// regression: leader with margin must not overflow past its content area.
	// use a simple VBox without border to keep layout predictable.
	cpu := 53
	mem := 37
	tmpl := Build(VBox(
		Leader("CPU", &cpu),
		Leader("MEM", &mem).Margin(1),
	))

	buf := NewBuffer(30, 10)
	tmpl.Execute(buf, 30, 10)

	// CPU at row 0, no margin, fills available width (30)
	cpuLine := buf.GetLine(0)
	if len(cpuLine) == 0 || cpuLine[0] != 'C' {
		t.Errorf("row 0: expected CPU leader, got %q", cpuLine)
	}

	// Row 1: empty (MEM top margin)
	if got := buf.GetLine(1); got != "" {
		t.Errorf("row 1 (MEM top margin): expected empty, got %q", got)
	}

	// Row 2: MEM leader with left margin=1
	memLine := buf.GetLine(2)
	if len(memLine) < 2 || memLine[0] != ' ' || memLine[1] != 'M' {
		t.Errorf("row 2: expected ' MEM...', got %q", memLine)
	}

	// key: CPU and MEM content widths. CPU renders 30 chars wide.
	// MEM renders 28 chars wide (30 - 2 margin), starting at column 1.
	// so MEM's last rendered character should be at column 28 (1 + 28 - 1).
	// CPU's last rendered character should be at column 29 (30 - 1).
	cpuLen := len(cpuLine)
	memLen := len(memLine)
	// MEM rendered content: 1 (left margin space) + 28 (content) = 29 chars total
	// CPU rendered content: 30 chars
	if memLen > cpuLen {
		t.Errorf("MEM line length (%d) > CPU line length (%d) — overflow!", memLen, cpuLen)
	}
}
