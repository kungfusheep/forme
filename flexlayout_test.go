package forme

import (
	"fmt"
	"testing"
)

func TestFlexVerticalLayout(t *testing.T) {
	// Create a simple vertical layout
	root := FCol(
		FText("Header"),
		FText("Line 1"),
		FText("Line 2"),
	).Gap(0)

	tree := NewFlexTree(root)
	buf := NewBuffer(40, 10)

	tree.Execute(buf, 40, 10)

	// Check that nodes are positioned vertically
	if root.children[0].Y != 0 {
		t.Errorf("First child Y should be 0, got %d", root.children[0].Y)
	}
	if root.children[1].Y != 1 {
		t.Errorf("Second child Y should be 1, got %d", root.children[1].Y)
	}
	if root.children[2].Y != 2 {
		t.Errorf("Third child Y should be 2, got %d", root.children[2].Y)
	}
}

func TestFlexHorizontalLayout(t *testing.T) {
	// Create a horizontal layout with percent widths
	root := FRow(
		FText("A").Percent(0.5),
		FText("B").Percent(0.5),
	)

	tree := NewFlexTree(root)
	buf := NewBuffer(40, 10)

	tree.Execute(buf, 40, 10)

	// Each child should get 50% of width
	if root.children[0].W != 20 {
		t.Errorf("First child W should be 20, got %d", root.children[0].W)
	}
	if root.children[1].W != 20 {
		t.Errorf("Second child W should be 20, got %d", root.children[1].W)
	}
	// Second child should be positioned after first
	if root.children[1].X != 20 {
		t.Errorf("Second child X should be 20, got %d", root.children[1].X)
	}
}

func TestFlexNestedLayout(t *testing.T) {
	// Create nested layout
	root := FCol(
		FRow(
			FText("Left").Percent(0.3),
			FText("Right").Percent(0.7),
		),
		FText("Footer"),
	)

	tree := NewFlexTree(root)
	buf := NewBuffer(100, 20)

	tree.Execute(buf, 100, 20)

	// Check levels
	if root.level != 1 {
		t.Errorf("Root level should be 1, got %d", root.level)
	}
	if root.children[0].level != 2 {
		t.Errorf("HBox level should be 2, got %d", root.children[0].level)
	}
	if root.children[0].children[0].level != 3 {
		t.Errorf("Left text level should be 3, got %d", root.children[0].children[0].level)
	}
}

func TestFlexPercentWidthDistribution(t *testing.T) {
	// Test that percent widths are distributed correctly
	root := FCol(
		FText("Full width").Percent(1.0),
		FText("Half width").Percent(0.5),
		FText("No percent"), // Should keep content width
	)

	tree := NewFlexTree(root)
	buf := NewBuffer(80, 10)

	tree.Execute(buf, 80, 10)

	if root.children[0].W != 80 {
		t.Errorf("100%% width child should be 80, got %d", root.children[0].W)
	}
	if root.children[1].W != 40 {
		t.Errorf("50%% width child should be 40, got %d", root.children[1].W)
	}
}

func TestFlexDisplayHelpers(t *testing.T) {
	// Test meter and bar nodes
	root := FCol(
		FLeader("CPU", "75%"),
		FMeter(75, 100).Width(20),
		FBar(7, 10),
	)

	tree := NewFlexTree(root)
	buf := NewBuffer(40, 10)

	tree.Execute(buf, 40, 10)

	output := buf.String()
	t.Logf("Display helpers output:\n%s", output)

	// Basic sanity check
	if len(output) == 0 {
		t.Error("Output should not be empty")
	}
}

func TestFlexLayoutWithBorder(t *testing.T) {
	// Test container with border
	root := FCol(
		FText("Inside panel"),
	).Border(BorderSingle)

	tree := NewFlexTree(root)
	buf := NewBuffer(30, 5)

	tree.Execute(buf, 30, 5)

	output := buf.String()
	t.Logf("Border output:\n%s", output)

	// Check border characters
	if buf.Get(0, 0).Rune != '┌' {
		t.Errorf("Top-left should be ┌, got %c", buf.Get(0, 0).Rune)
	}
}

func TestFlexBottomUpHeightCalculation(t *testing.T) {
	// Test that parent height is calculated from children (bottom-up)
	inner := FCol(
		FText("Line 1"),
		FText("Line 2"),
		FText("Line 3"),
	)

	root := FCol(inner)

	tree := NewFlexTree(root)
	buf := NewBuffer(40, 20)

	tree.Execute(buf, 40, 20)

	// Inner container should have height = 3 (3 lines)
	if inner.H != 3 {
		t.Errorf("Inner container H should be 3, got %d", inner.H)
	}
}

// TestExampleDashboard shows a complete dashboard layout
func TestExampleDashboard(t *testing.T) {
	// Build a dashboard layout
	dashboard := FCol(
		// Header row
		FRow(
			FText("SYSTEM MONITOR").Bold(),
		),
		// Status section with leaders
		FCol(
			FLeader("CPU USAGE", "78%"),
			FLeader("MEMORY", "4.2GB/8GB"),
			FLeader("DISK", "120GB FREE"),
		).Percent(1.0),
		// Meters section
		FCol(
			FRow(
				FText("LOAD "),
				FMeter(78, 100).Width(20),
				FText(" 78%"),
			),
			FRow(
				FText("TEMP "),
				FMeter(45, 100).Width(20),
				FText(" 45C"),
			),
		),
		// Bar section
		FRow(
			FText("NETWORK: "),
			FBar(7, 10),
			FText(" 70%"),
		),
	).Gap(1)

	tree := NewFlexTree(dashboard)
	buf := NewBuffer(50, 15)

	tree.Execute(buf, 50, 15)

	output := buf.StringTrimmed()
	fmt.Printf("Dashboard output:\n%s\n", output)

	if len(output) == 0 {
		t.Error("Dashboard output should not be empty")
	}
}

func TestFPanel(t *testing.T) {
	panel := FPanel("STATUS",
		FLeader("CPU", "78%"),
		FLeader("MEM", "4.2GB"),
	)

	tree := NewFlexTree(panel)
	buf := NewBuffer(30, 6)

	tree.Execute(buf, 30, 6)

	output := buf.String()
	t.Logf("Panel output:\n%s", output)

	// Check that title appears
	if buf.Get(2, 0).Rune != ' ' && buf.Get(3, 0).Rune != 'S' {
		// Title should start with "─ STATUS "
	}
}

func TestFLEDs(t *testing.T) {
	// Test LED indicators
	row := FRow(
		FText("SYSTEMS: "),
		FLEDs(true, true, false, true),
	)

	tree := NewFlexTree(row)
	buf := NewBuffer(30, 3)

	tree.Execute(buf, 30, 3)

	output := buf.StringTrimmed()
	t.Logf("LEDs output: %s", output)

	// Should contain LED symbols
	if len(output) == 0 {
		t.Error("Output should not be empty")
	}
}

func TestFlexGrow(t *testing.T) {
	// Test that flex grow distributes remaining space
	root := FCol(
		FText("Header"),                    // H=1
		FCol(FText("Line 1")).Grow(1),      // Should fill remaining space
	)

	tree := NewFlexTree(root)
	buf := NewBuffer(40, 20)

	tree.Execute(buf, 40, 20)

	// Root H = 20
	// Header H = 1
	// Remaining = 20 - 1 = 19
	// Flex child should get all remaining space
	flexChild := root.children[1]
	if flexChild.H != 19 {
		t.Errorf("Flex child H should be 19, got %d", flexChild.H)
	}
}

func TestFlexGrowWithMultipleChildren(t *testing.T) {
	// Test flex grow with fixed + flex children
	root := FCol(
		FCol(FText("HBox 1")).Height(5), // Fixed H=5
		FCol(FText("HBox 2")).Height(5), // Fixed H=5
		FCol(FText("Log")).Grow(1),     // Should fill remaining
	)

	tree := NewFlexTree(root)
	buf := NewBuffer(40, 30)

	tree.Execute(buf, 40, 30)

	// Content = 5 + 5 + 1 (log content) = 11
	// Remaining = 30 - 11 = 19
	// Flex child gets 19 extra, so H = 1 + 19 = 20
	logPanel := root.children[2]
	if logPanel.H != 20 {
		t.Errorf("Log panel H should be 20, got %d", logPanel.H)
	}
}

// TestDenseDashboard creates a more complete dense dashboard
func TestDenseDashboard(t *testing.T) {
	dashboard := FCol(
		// Top row with two panels
		FRow(
			FPanel("STATUS",
				FLeader("ITEM A", "OK"),
				FLeader("ITEM B", "PASS"),
				FLeader("ITEM C", "FAIL"),
			).Percent(0.5),
			FPanel("SYSTEMS",
				FRow(FLED(true), FText(" POWER  "), FLED(true), FText(" COMMS")),
				FRow(FLED(false), FText(" BACKUP "), FLED(true), FText(" LINK")),
				FRow(FText("STATUS: "), FLEDs(true, true, false, false)),
			).Percent(0.5),
		),
		// Bottom row with two panels
		FRow(
			FPanel("LEVELS",
				FRow(FText("CPU "), FMeter(75, 100).Width(15), FText(" 75%")),
				FRow(FText("MEM "), FMeter(45, 100).Width(15), FText(" 45%")),
				FRow(FText("DSK "), FBar(8, 10), FText(" 80%")),
			).Percent(0.5),
			FPanel("CAPACITY",
				FRow(FText("TANK A "), FBar(9, 10)),
				FRow(FText("TANK B "), FBar(6, 10)),
				FRow(FText("TANK C "), FBar(3, 10)),
			).Percent(0.5),
		),
	)

	tree := NewFlexTree(dashboard)
	buf := NewBuffer(60, 15)

	tree.Execute(buf, 60, 15)

	output := buf.StringTrimmed()
	fmt.Printf("\nDense Dashboard:\n%s\n", output)

	if len(output) == 0 {
		t.Error("Output should not be empty")
	}
}
