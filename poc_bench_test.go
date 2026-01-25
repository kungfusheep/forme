package tui

import (
	"testing"
)

// progNode mimics SerialNode layout for progress
type progNode struct {
	kind        uint8
	x, y, width int16
	ratio       float32
}

// BenchmarkPOCProgressDirect measures raw WriteProgressBar performance
func BenchmarkPOCProgressDirect(b *testing.B) {
	nodes := make([]progNode, 100)
	for j := 0; j < 100; j++ {
		nodes[j] = progNode{
			kind:  3,
			x:     int16((j % 10) * 10),
			y:     int16(j / 10),
			width: 8,
			ratio: float32(j) / 100.0,
		}
	}
	buf := NewBuffer(100, 50)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		for j := range nodes {
			n := &nodes[j]
			buf.WriteProgressBar(int(n.x), int(n.y), int(n.width), n.ratio, Style{})
		}
	}
}

// BenchmarkPOCProgressNoClear measures without ClearDirty
func BenchmarkPOCProgressNoClear(b *testing.B) {
	nodes := make([]progNode, 100)
	for j := 0; j < 100; j++ {
		nodes[j] = progNode{
			kind:  3,
			x:     int16((j % 10) * 10),
			y:     int16(j / 10),
			width: 8,
			ratio: float32(j) / 100.0,
		}
	}
	buf := NewBuffer(100, 50)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := range nodes {
			n := &nodes[j]
			buf.WriteProgressBar(int(n.x), int(n.y), int(n.width), n.ratio, Style{})
		}
	}
}

// TestSimpleForEach verifies single-level ForEach works
func TestSimpleForEach(t *testing.T) {
	// Simple non-nested ForEach
	items := make([]StressItem, 10)
	for i := range items {
		items[i] = StressItem{CPU: float32(i) / 10.0}
	}

	ui := VBoxNode{
		Children: []any{
			TextNode{Content: "Simple ForEach"},
			ForEach(&items, func(item *StressItem) any {
				return ProgressNode{Value: &item.CPU, BarWidth: 8}
			}),
		},
	}

	serial := Build(ui)
	buf := NewBuffer(100, 50)
	buf.Clear()
	serial.Execute(buf, 100, 50)

	// Row 1 should have progress bar characters (█, partial blocks, or space with BG)
	cell := buf.Get(0, 1)
	isProgressChar := cell.Rune == '█' || cell.Rune == ' ' || cell.Rune == '░' ||
		cell.Rune == '▏' || cell.Rune == '▎' || cell.Rune == '▍' || cell.Rune == '▌' ||
		cell.Rune == '▋' || cell.Rune == '▊' || cell.Rune == '▉'
	if !isProgressChar {
		t.Errorf("Expected progress bar character at (0,1), got %c", cell.Rune)
	}
}

// TestNestedForEach verifies nested ForEach with SerialOpForEachOffset works
func TestNestedForEach(t *testing.T) {
	buf := NewBuffer(100, 50)

	// 10x10 grid of progress bars using nested ForEach
	rows := make([][]StressItem, 10)
	for i := range rows {
		rows[i] = make([]StressItem, 10)
		for j := range rows[i] {
			rows[i][j] = StressItem{
				CPU: float32((i*10+j)%100) / 100.0,
			}
		}
	}

	ui := VBoxNode{
		Children: []any{
			TextNode{Content: "Dense Grid"},
			ForEach(&rows, func(row *[]StressItem) any {
				return HBoxNode{Children: []any{
					ForEach(row, func(item *StressItem) any {
						return ProgressNode{Value: &item.CPU, BarWidth: 8}
					}),
				}}
			}),
		},
	}

	serial := Build(ui)
	buf.Clear()
	serial.Execute(buf, 100, 50)

	// Row 1 should have progress bar characters (10 bars * 8 chars = 80 chars)
	// Progress bars use █, partial blocks (▏▎▍▌▋▊▉), or spaces with BG color
	progressChars := 0
	for x := 0; x < 80; x++ {
		cell := buf.Get(x, 1)
		isProgressChar := cell.Rune == '█' || cell.Rune == ' ' || cell.Rune == '░' || cell.Rune == '▓' ||
			cell.Rune == '▏' || cell.Rune == '▎' || cell.Rune == '▍' || cell.Rune == '▌' ||
			cell.Rune == '▋' || cell.Rune == '▊' || cell.Rune == '▉'
		if isProgressChar {
			progressChars++
		}
	}
	if progressChars < 70 { // Allow for some variation in filled/empty chars
		t.Errorf("Expected ~80 progress bar characters on row 1, got %d", progressChars)
	}
}

// BenchmarkPOCProgressSwitch mimics SerialTemplate's switch dispatch
func BenchmarkPOCProgressSwitch(b *testing.B) {
	const kindProgress = 3
	type node struct {
		kind        uint8
		W, H        int16
		X, Y        int16
		Text        string
		Ratio       float32
		Width       int16
	}
	nodes := make([]node, 100)
	for j := 0; j < 100; j++ {
		nodes[j] = node{
			kind:  kindProgress,
			X:     int16((j % 10) * 10),
			Y:     int16(j / 10),
			Width: 8,
			Ratio: float32(j) / 100.0,
		}
	}
	buf := NewBuffer(100, 50)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		for j := range nodes {
			n := &nodes[j]
			switch {
			case n.kind == kindProgress:
				buf.WriteProgressBar(int(n.X), int(n.Y), int(n.Width), n.Ratio, Style{})
			}
		}
	}
}
