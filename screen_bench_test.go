package forme

import (
	"bytes"
	"testing"
)

// mockWriter discards output but counts bytes
type mockWriter struct {
	n int
}

func (w *mockWriter) Write(p []byte) (int, error) {
	w.n += len(p)
	return len(p), nil
}

// BenchmarkFlushFullScreen benchmarks flushing when entire screen changed
func BenchmarkFlushFullScreen(b *testing.B) {
	w := &mockWriter{}
	s := &Screen{
		width:  120,
		height: 40,
		back:   NewBuffer(120, 40),
		front:  NewBuffer(120, 40),
		buf:    bytes.Buffer{},
		writer: w,
	}

	// Fill back buffer with content
	for y := 0; y < 40; y++ {
		for x := 0; x < 120; x++ {
			s.back.Set(x, y, Cell{Rune: 'A', Style: DefaultStyle()})
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Reset front buffer to force full redraw
		s.front.Clear()
		// Mark back buffer dirty so flush will check all rows
		s.back.MarkAllDirty()
		w.n = 0
		s.Flush()
	}
	b.ReportMetric(float64(w.n), "bytes/op")
}

// BenchmarkFlushSparseChanges benchmarks flushing with only a few changed cells
func BenchmarkFlushSparseChanges(b *testing.B) {
	w := &mockWriter{}
	s := &Screen{
		width:  120,
		height: 40,
		back:   NewBuffer(120, 40),
		front:  NewBuffer(120, 40),
		buf:    bytes.Buffer{},
		writer: w,
	}

	// Fill both buffers identically
	for y := 0; y < 40; y++ {
		for x := 0; x < 120; x++ {
			cell := Cell{Rune: 'A', Style: DefaultStyle()}
			s.back.Set(x, y, cell)
			s.front.Set(x, y, cell)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Change just 10 cells on different rows
		for j := 0; j < 10; j++ {
			s.back.Set(j*10, j*4, Cell{Rune: rune('0' + (i+j)%10), Style: DefaultStyle()})
		}
		w.n = 0
		s.Flush()
	}
	b.ReportMetric(float64(w.n), "bytes/op")
}

// BenchmarkFlushOneLineChanged benchmarks flushing with one line changed
func BenchmarkFlushOneLineChanged(b *testing.B) {
	w := &mockWriter{}
	s := &Screen{
		width:  120,
		height: 40,
		back:   NewBuffer(120, 40),
		front:  NewBuffer(120, 40),
		buf:    bytes.Buffer{},
		writer: w,
	}

	// Fill both buffers identically
	for y := 0; y < 40; y++ {
		for x := 0; x < 120; x++ {
			cell := Cell{Rune: 'A', Style: DefaultStyle()}
			s.back.Set(x, y, cell)
			s.front.Set(x, y, cell)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Change one entire line
		for x := 0; x < 120; x++ {
			s.back.Set(x, 20, Cell{Rune: rune('0' + (i+x)%10), Style: DefaultStyle()})
		}
		w.n = 0
		s.Flush()
	}
	b.ReportMetric(float64(w.n), "bytes/op")
}

// BenchmarkFlushNoChanges benchmarks flushing when nothing changed
func BenchmarkFlushNoChanges(b *testing.B) {
	w := &mockWriter{}
	s := &Screen{
		width:  120,
		height: 40,
		back:   NewBuffer(120, 40),
		front:  NewBuffer(120, 40),
		buf:    bytes.Buffer{},
		writer: w,
	}

	// Fill both buffers identically
	for y := 0; y < 40; y++ {
		for x := 0; x < 120; x++ {
			cell := Cell{Rune: 'A', Style: DefaultStyle()}
			s.back.Set(x, y, cell)
			s.front.Set(x, y, cell)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w.n = 0
		s.Flush()
	}
	b.ReportMetric(float64(w.n), "bytes/op")
}

// BenchmarkWriteIntToBuf benchmarks integer formatting
func BenchmarkWriteIntToBuf(b *testing.B) {
	s := &Screen{
		buf: bytes.Buffer{},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.buf.Reset()
		s.writeIntToBuf(12345)
	}
}

// BenchmarkAppendInt benchmarks the appendInt helper
func BenchmarkAppendInt(b *testing.B) {
	var scratch [32]byte

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := scratch[:0]
		buf = appendInt(buf, 12345)
	}
}
