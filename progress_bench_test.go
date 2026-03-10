package glyph

import (
	"testing"
)

// progNode mimics SerialNode layout for progress
type progNode struct {
	kind        uint8
	x, y, width int16
	ratio       float32
}

// BenchmarkProgressBarDirect measures raw WriteProgressBar performance
func BenchmarkProgressBarDirect(b *testing.B) {
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

// BenchmarkProgressBarNoClear measures without ClearDirty
func BenchmarkProgressBarNoClear(b *testing.B) {
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

// BenchmarkProgressBarSwitch mimics SerialTemplate's switch dispatch
func BenchmarkProgressBarSwitch(b *testing.B) {
	const kindProgress = 3
	type node struct {
		kind  uint8
		W, H  int16
		X, Y  int16
		Text  string
		Ratio float32
		Width int16
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
