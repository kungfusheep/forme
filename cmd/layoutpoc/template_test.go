package main

import (
	"testing"

	"tui"
)

func TestStaticText(t *testing.T) {
	ui := tui.VBox{Children: []any{
		tui.Text{Content: "Hello"},
		tui.Text{Content: "World"},
	}}

	tmpl := Compile(ui)
	buf := tui.NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	if buf.Get(0, 0).Rune != 'H' {
		t.Errorf("expected 'H' at (0,0), got '%c'", buf.Get(0, 0).Rune)
	}
	if buf.Get(0, 1).Rune != 'W' {
		t.Errorf("expected 'W' at (0,1), got '%c'", buf.Get(0, 1).Rune)
	}
}

func TestForEach(t *testing.T) {
	items := []string{"AAA", "BBB", "CCC"}

	ui := tui.ForEach(&items, func(s *string) any {
		return tui.Text{Content: s}
	})

	tmpl := Compile(ui)
	buf := tui.NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	for i, expected := range items {
		cell := buf.Get(0, i)
		if cell.Rune != rune(expected[0]) {
			t.Errorf("row %d: expected '%c', got '%c'", i, expected[0], cell.Rune)
		}
	}
}

func TestNestedForEach(t *testing.T) {
	type Item struct {
		CPU float32
	}
	rows := make([][]Item, 3)
	for i := range rows {
		rows[i] = make([]Item, 3)
		for j := range rows[i] {
			rows[i][j] = Item{CPU: 0.5}
		}
	}

	ui := tui.VBox{Children: []any{
		tui.ForEach(&rows, func(row *[]Item) any {
			return tui.HBox{Children: []any{
				tui.ForEach(row, func(item *Item) any {
					return tui.Progress{Value: &item.CPU, BarWidth: 8}
				}),
			}}
		}),
	}}

	tmpl := Compile(ui)
	buf := tui.NewBuffer(100, 10)
	tmpl.Execute(buf, 100, 10)

	// Verify progress bars drawn on each row
	for y := 0; y < 3; y++ {
		cell := buf.Get(0, y)
		if cell.Rune == 0 {
			t.Errorf("expected progress bar at (0,%d), got empty", y)
		}
	}
}

func TestHBoxLayout(t *testing.T) {
	type Item struct {
		Label string
		Value int
	}
	items := []Item{
		{Label: "A:", Value: 1},
		{Label: "B:", Value: 2},
	}

	ui := tui.ForEach(&items, func(item *Item) any {
		return tui.HBox{Gap: 1, Children: []any{
			tui.Text{Content: &item.Label},
			tui.Text{Content: &item.Value},
		}}
	})

	tmpl := Compile(ui)
	buf := tui.NewBuffer(30, 10)
	tmpl.Execute(buf, 30, 10)

	// First row: "A:" at x=0, then gap, then "1"
	if buf.Get(0, 0).Rune != 'A' {
		t.Errorf("expected 'A' at (0,0), got '%c'", buf.Get(0, 0).Rune)
	}
	// Note: for offset-bound strings, width defaults to 10 during layout
	// since we don't have the element pointer yet. So second text is at x=11.
	if buf.Get(11, 0).Rune != '1' {
		t.Errorf("expected '1' at (11,0), got '%c'", buf.Get(11, 0).Rune)
	}
}

func TestViewportCulling(t *testing.T) {
	type Counter struct {
		Value int
	}
	rows := make([]Counter, 100)
	for i := range rows {
		rows[i] = Counter{Value: i}
	}

	ui := tui.ForEach(&rows, func(c *Counter) any {
		return tui.Text{Content: &c.Value}
	})

	tmpl := Compile(ui)
	tmpl.ViewportY = 50
	tmpl.ViewportHeight = 10

	buf := tui.NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	// First visible row should show "50"
	cell0 := buf.Get(0, 0)
	cell1 := buf.Get(1, 0)
	if cell0.Rune != '5' || cell1.Rune != '0' {
		t.Errorf("expected '50' at row 0, got '%c%c'", cell0.Rune, cell1.Rune)
	}
}

func TestCustomComponent(t *testing.T) {
	// Custom component that draws a box
	box := tui.Custom{
		Measure: func(availW int16) (w, h int16) {
			return 5, 3
		},
		Render: func(buf *tui.Buffer, x, y, w, h int16) {
			// Draw corners
			buf.Set(int(x), int(y), tui.Cell{Rune: '+'})
			buf.Set(int(x+w-1), int(y), tui.Cell{Rune: '+'})
			buf.Set(int(x), int(y+h-1), tui.Cell{Rune: '+'})
			buf.Set(int(x+w-1), int(y+h-1), tui.Cell{Rune: '+'})
		},
	}

	ui := tui.VBox{Children: []any{
		tui.Text{Content: "Header"},
		box,
		tui.Text{Content: "Footer"},
	}}

	tmpl := Compile(ui)
	buf := tui.NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	// Header at row 0
	if buf.Get(0, 0).Rune != 'H' {
		t.Errorf("expected 'H' at (0,0), got '%c'", buf.Get(0, 0).Rune)
	}

	// Box corners at rows 1-3
	if buf.Get(0, 1).Rune != '+' {
		t.Errorf("expected '+' at (0,1), got '%c'", buf.Get(0, 1).Rune)
	}
	if buf.Get(4, 1).Rune != '+' {
		t.Errorf("expected '+' at (4,1), got '%c'", buf.Get(4, 1).Rune)
	}
	if buf.Get(0, 3).Rune != '+' {
		t.Errorf("expected '+' at (0,3), got '%c'", buf.Get(0, 3).Rune)
	}

	// Footer at row 4 (after 3-row box)
	if buf.Get(0, 4).Rune != 'F' {
		t.Errorf("expected 'F' at (0,4), got '%c'", buf.Get(0, 4).Rune)
	}
}

func TestCustomInForEach(t *testing.T) {
	type Item struct {
		Value int
	}
	items := []Item{{Value: 1}, {Value: 2}, {Value: 3}}

	// Custom that shows value as repeated chars
	ui := tui.ForEach(&items, func(item *Item) any {
		return tui.Custom{
			Measure: func(availW int16) (w, h int16) {
				return 5, 1
			},
			Render: func(buf *tui.Buffer, x, y, w, h int16) {
				// Draw char based on item value
				buf.Set(int(x), int(y), tui.Cell{Rune: 'X'})
			},
		}
	})

	tmpl := Compile(ui)
	buf := tui.NewBuffer(20, 10)
	tmpl.Execute(buf, 20, 10)

	// Each row should have 'X' at start
	for i := 0; i < 3; i++ {
		if buf.Get(0, i).Rune != 'X' {
			t.Errorf("expected 'X' at (0,%d), got '%c'", i, buf.Get(0, i).Rune)
		}
	}
}

// TestSliceReassignment tests that ForEach works when slice is reassigned after compile
func TestSliceReassignment(t *testing.T) {
	type State struct {
		AllItems     []string
		VisibleItems []string
	}

	state := &State{
		AllItems: []string{"A", "B", "C", "D", "E"},
	}

	// Build UI BEFORE initializing VisibleItems (like the demo does)
	ui := tui.ForEach(&state.VisibleItems, func(s *string) any {
		return tui.Text{Content: s}
	})

	tmpl := Compile(ui)
	buf := tui.NewBuffer(20, 10)

	// Now assign VisibleItems (like the demo's render loop does)
	state.VisibleItems = state.AllItems[:3] // "A", "B", "C"

	tmpl.Execute(buf, 20, 10)

	// Should render 3 rows
	if buf.Get(0, 0).Rune != 'A' {
		t.Errorf("row 0: expected 'A', got '%c'", buf.Get(0, 0).Rune)
	}
	if buf.Get(0, 1).Rune != 'B' {
		t.Errorf("row 1: expected 'B', got '%c'", buf.Get(0, 1).Rune)
	}
	if buf.Get(0, 2).Rune != 'C' {
		t.Errorf("row 2: expected 'C', got '%c'", buf.Get(0, 2).Rune)
	}
}

// TestHBoxChildPositions tests that Row children are positioned correctly
func TestHBoxChildPositions(t *testing.T) {
	ui := tui.HBox{Gap: 1, Children: []any{
		tui.Text{Content: "AAA"},
		tui.Text{Content: "BBB"},
	}}

	tmpl := Compile(ui)
	buf := tui.NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	// AAA at X=0, BBB at X=4 (3 chars + 1 gap)
	if buf.Get(0, 0).Rune != 'A' {
		t.Errorf("expected 'A' at x=0, got '%c'", buf.Get(0, 0).Rune)
	}
	if buf.Get(4, 0).Rune != 'B' {
		t.Errorf("expected 'B' at x=4, got '%c'", buf.Get(4, 0).Rune)
	}
}

// TestForEachWithRowElements tests ForEach where each element is a Row
func TestForEachWithRowElements(t *testing.T) {
	type Item struct {
		A string
		B string
	}
	items := []Item{
		{A: "X1", B: "Y1"},
		{A: "X2", B: "Y2"},
	}

	ui := tui.ForEach(&items, func(item *Item) any {
		return tui.HBox{Gap: 1, Children: []any{
			tui.Text{Content: &item.A},
			tui.Text{Content: &item.B},
		}}
	})

	tmpl := Compile(ui)
	buf := tui.NewBuffer(20, 5)
	tmpl.Execute(buf, 20, 5)

	// Row 0: X1 at 0, Y1 at 11 (10 default width + 1 gap)
	// Row 1: X2 at 0, Y2 at 11
	if buf.Get(0, 0).Rune != 'X' {
		t.Errorf("row 0 col 0: expected 'X', got '%c'", buf.Get(0, 0).Rune)
	}
	if buf.Get(0, 1).Rune != 'X' {
		t.Errorf("row 1 col 0: expected 'X', got '%c'", buf.Get(0, 1).Rune)
	}
}

// TestForEachRowWithProgress tests ForEach with Row containing Text and Progress
func TestForEachRowWithProgress(t *testing.T) {
	type Item struct {
		Name string
		Val  float32
	}
	items := []Item{
		{Name: "A", Val: 0.5},
		{Name: "B", Val: 0.5},
	}

	ui := tui.ForEach(&items, func(item *Item) any {
		return tui.HBox{Gap: 1, Children: []any{
			tui.Text{Content: &item.Name},
			tui.Progress{Value: &item.Val, BarWidth: 5},
		}}
	})

	tmpl := Compile(ui)
	buf := tui.NewBuffer(30, 5)
	tmpl.Execute(buf, 30, 5)

	// Debug: print buffer
	for y := 0; y < 3; y++ {
		t.Logf("HBox %d: ", y)
		for x := 0; x < 20; x++ {
			r := buf.Get(x, y).Rune
			if r == 0 {
				t.Logf(".")
			} else {
				t.Logf("%c", r)
			}
		}
	}

	// Text should be at X=0, Progress at X=11 (10 default + 1 gap)
	if buf.Get(0, 0).Rune != 'A' {
		t.Errorf("row 0 col 0: expected 'A', got '%c'", buf.Get(0, 0).Rune)
	}
	if buf.Get(0, 1).Rune != 'B' {
		t.Errorf("row 1 col 0: expected 'B', got '%c'", buf.Get(0, 1).Rune)
	}
}

// TestProcessListPattern tests the exact pattern used in the demo
func TestProcessListPattern(t *testing.T) {
	type Process struct {
		Name string
		CPU  float32
	}
	type State struct {
		AllProcesses     []Process
		VisibleProcesses []Process
	}

	state := &State{}
	state.AllProcesses = make([]Process, 10)
	for i := range state.AllProcesses {
		state.AllProcesses[i] = Process{
			Name: string(rune('A' + i)),
			CPU:  0.5,
		}
	}

	// Build UI with nil VisibleProcesses (like demo)
	ui := tui.VBox{Children: []any{
		tui.Text{Content: "Header"},
		tui.ForEach(&state.VisibleProcesses, func(p *Process) any {
			return tui.HBox{Gap: 1, Children: []any{
				tui.Text{Content: &p.Name},
				tui.Progress{Value: &p.CPU, BarWidth: 10},
			}}
		}),
	}}

	tmpl := Compile(ui)
	buf := tui.NewBuffer(40, 20)

	// Assign visible slice (like demo render loop)
	state.VisibleProcesses = state.AllProcesses[:5]

	tmpl.Execute(buf, 40, 20)

	// Header at row 0
	if buf.Get(0, 0).Rune != 'H' {
		t.Errorf("row 0: expected 'H', got '%c'", buf.Get(0, 0).Rune)
	}

	// 5 process rows starting at row 1
	for i := 0; i < 5; i++ {
		expected := rune('A' + i)
		actual := buf.Get(0, i+1).Rune
		if actual != expected {
			t.Errorf("row %d: expected '%c', got '%c'", i+1, expected, actual)
		}
	}
}

// TestForEachInRow tests horizontal ForEach inside HBox
func TestForEachInRow(t *testing.T) {
	values := []float32{0.5, 0.5, 0.5}

	ui := tui.HBox{Children: []any{
		tui.ForEach(&values, func(v *float32) any {
			return tui.Progress{Value: v, BarWidth: 5}
		}),
	}}

	tmpl := Compile(ui)
	buf := tui.NewBuffer(30, 5)
	tmpl.Execute(buf, 30, 5)

	// Three progress bars should be at X=0, X=5, X=10 (each width 5)
	// Progress bar at 50% should have filled char at position 2-3
	cell0 := buf.Get(0, 0)
	cell5 := buf.Get(5, 0)
	cell10 := buf.Get(10, 0)

	if cell0.Rune == 0 {
		t.Errorf("expected progress bar at x=0, got empty")
	}
	if cell5.Rune == 0 {
		t.Errorf("expected progress bar at x=5, got empty")
	}
	if cell10.Rune == 0 {
		t.Errorf("expected progress bar at x=10, got empty")
	}
}

func TestDynamicSliceLength(t *testing.T) {
	items := []string{"A", "B", "C"}

	ui := tui.ForEach(&items, func(s *string) any {
		return tui.Text{Content: s}
	})

	tmpl := Compile(ui)
	buf := tui.NewBuffer(20, 10)

	// First execute with 3 items
	tmpl.Execute(buf, 20, 10)
	if buf.Get(0, 0).Rune != 'A' {
		t.Errorf("expected 'A' at (0,0)")
	}
	if buf.Get(0, 2).Rune != 'C' {
		t.Errorf("expected 'C' at (0,2)")
	}

	// Modify underlying data and execute again
	items[0] = "X"
	items[1] = "Y"
	items[2] = "Z"
	buf.Clear()
	tmpl.Execute(buf, 20, 10)

	// Should reflect new data
	if buf.Get(0, 0).Rune != 'X' {
		t.Errorf("expected 'X' at (0,0), got '%c'", buf.Get(0, 0).Rune)
	}
}

// Benchmarks

func BenchmarkExecute10x10(b *testing.B) {
	type Item struct {
		CPU float32
	}
	rows := make([][]Item, 10)
	for i := range rows {
		rows[i] = make([]Item, 10)
		for j := range rows[i] {
			rows[i][j] = Item{CPU: float32((i*10+j)%100) / 100.0}
		}
	}

	ui := tui.VBox{Children: []any{
		tui.ForEach(&rows, func(row *[]Item) any {
			return tui.HBox{Children: []any{
				tui.ForEach(row, func(item *Item) any {
					return tui.Progress{Value: &item.CPU, BarWidth: 8}
				}),
			}}
		}),
	}}

	tmpl := Compile(ui)
	buf := tui.NewBuffer(100, 50)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		tmpl.Execute(buf, 100, 50)
	}
}

func BenchmarkExecuteWithCulling(b *testing.B) {
	type Item struct {
		CPU float32
	}
	rows := make([][]Item, 1000)
	for i := range rows {
		rows[i] = make([]Item, 10)
		for j := range rows[i] {
			rows[i][j] = Item{CPU: float32((i*10+j)%100) / 100.0}
		}
	}

	ui := tui.VBox{Children: []any{
		tui.ForEach(&rows, func(row *[]Item) any {
			return tui.HBox{Children: []any{
				tui.ForEach(row, func(item *Item) any {
					return tui.Progress{Value: &item.CPU, BarWidth: 8}
				}),
			}}
		}),
	}}

	tmpl := Compile(ui)
	tmpl.ViewportY = 500
	tmpl.ViewportHeight = 10
	buf := tui.NewBuffer(100, 10)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		tmpl.Execute(buf, 100, 10)
	}
}

func BenchmarkExecuteText100(b *testing.B) {
	type Item struct {
		Name string
	}
	items := make([]Item, 100)
	for i := range items {
		items[i] = Item{Name: "test"}
	}

	ui := tui.ForEach(&items, func(item *Item) any {
		return tui.Text{Content: &item.Name}
	})

	tmpl := Compile(ui)
	buf := tui.NewBuffer(100, 100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		tmpl.Execute(buf, 100, 100)
	}
}

// === STRESS BENCHMARKS ===

// BenchmarkStress1000Rows simulates an htop-style process list with 1000 rows
func BenchmarkStress1000Rows(b *testing.B) {
	type Process struct {
		PID   int
		Name  string
		CPU   float32
		Count int
	}
	procs := make([]Process, 1000)
	for i := range procs {
		procs[i] = Process{
			PID:   i,
			Name:  "process-name",
			CPU:   float32(i%100) / 100.0,
			Count: i * 100,
		}
	}

	ui := tui.ForEach(&procs, func(p *Process) any {
		return tui.HBox{Gap: 1, Children: []any{
			tui.Text{Content: &p.PID},
			tui.Text{Content: &p.Name},
			tui.Progress{Value: &p.CPU, BarWidth: 10},
			tui.Text{Content: &p.Count},
		}}
	})

	tmpl := Compile(ui)
	tmpl.ViewportY = 500 // Middle of the list
	tmpl.ViewportHeight = 30

	buf := tui.NewBuffer(100, 30)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		tmpl.Execute(buf, 100, 30)
	}
}

// BenchmarkStressDashboard simulates a dashboard with multiple panels
func BenchmarkStressDashboard(b *testing.B) {
	type Metric struct {
		Label string
		Value float32
	}

	// 4 panels, each with 25 metrics = 100 total metrics
	panels := make([][]Metric, 4)
	for i := range panels {
		panels[i] = make([]Metric, 25)
		for j := range panels[i] {
			panels[i][j] = Metric{
				Label: "metric",
				Value: float32((i*25+j)%100) / 100.0,
			}
		}
	}

	// Dashboard layout: 2x2 grid of panels
	ui := tui.VBox{Children: []any{
		tui.Text{Content: "=== Dashboard ==="},
		tui.HBox{Gap: 2, Children: []any{
			// Left column
			tui.VBox{Children: []any{
				tui.Text{Content: "Panel 1"},
				tui.ForEach(&panels[0], func(m *Metric) any {
					return tui.HBox{Gap: 1, Children: []any{
						tui.Text{Content: &m.Label},
						tui.Progress{Value: &m.Value, BarWidth: 15},
					}}
				}),
			}},
			// Right column
			tui.VBox{Children: []any{
				tui.Text{Content: "Panel 2"},
				tui.ForEach(&panels[1], func(m *Metric) any {
					return tui.HBox{Gap: 1, Children: []any{
						tui.Text{Content: &m.Label},
						tui.Progress{Value: &m.Value, BarWidth: 15},
					}}
				}),
			}},
		}},
		tui.HBox{Gap: 2, Children: []any{
			// Left column
			tui.VBox{Children: []any{
				tui.Text{Content: "Panel 3"},
				tui.ForEach(&panels[2], func(m *Metric) any {
					return tui.HBox{Gap: 1, Children: []any{
						tui.Text{Content: &m.Label},
						tui.Progress{Value: &m.Value, BarWidth: 15},
					}}
				}),
			}},
			// Right column
			tui.VBox{Children: []any{
				tui.Text{Content: "Panel 4"},
				tui.ForEach(&panels[3], func(m *Metric) any {
					return tui.HBox{Gap: 1, Children: []any{
						tui.Text{Content: &m.Label},
						tui.Progress{Value: &m.Value, BarWidth: 15},
					}}
				}),
			}},
		}},
	}}

	tmpl := Compile(ui)
	buf := tui.NewBuffer(120, 60)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		tmpl.Execute(buf, 120, 60)
	}
}

// BenchmarkStress10x100Grid tests a large grid (1000 cells)
func BenchmarkStress10x100Grid(b *testing.B) {
	type Cell struct {
		Value float32
	}
	rows := make([][]Cell, 100)
	for i := range rows {
		rows[i] = make([]Cell, 10)
		for j := range rows[i] {
			rows[i][j] = Cell{Value: float32((i*10+j)%100) / 100.0}
		}
	}

	ui := tui.VBox{Children: []any{
		tui.ForEach(&rows, func(row *[]Cell) any {
			return tui.HBox{Children: []any{
				tui.ForEach(row, func(cell *Cell) any {
					return tui.Progress{Value: &cell.Value, BarWidth: 10}
				}),
			}}
		}),
	}}

	tmpl := Compile(ui)
	buf := tui.NewBuffer(120, 100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		tmpl.Execute(buf, 120, 100)
	}
}

// BenchmarkStress10x100GridCulled tests viewport culling on large grid
func BenchmarkStress10x100GridCulled(b *testing.B) {
	type Cell struct {
		Value float32
	}
	rows := make([][]Cell, 100)
	for i := range rows {
		rows[i] = make([]Cell, 10)
		for j := range rows[i] {
			rows[i][j] = Cell{Value: float32((i*10+j)%100) / 100.0}
		}
	}

	ui := tui.VBox{Children: []any{
		tui.ForEach(&rows, func(row *[]Cell) any {
			return tui.HBox{Children: []any{
				tui.ForEach(row, func(cell *Cell) any {
					return tui.Progress{Value: &cell.Value, BarWidth: 10}
				}),
			}}
		}),
	}}

	tmpl := Compile(ui)
	tmpl.ViewportY = 50
	tmpl.ViewportHeight = 20
	buf := tui.NewBuffer(120, 20)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		tmpl.Execute(buf, 120, 20)
	}
}
