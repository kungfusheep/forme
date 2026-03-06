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

// BenchmarkFlushColor16NoForceRGB flushes a Color16-filled screen without forceRGB.
// baseline: compact ANSI escape codes.
func BenchmarkFlushColor16NoForceRGB(b *testing.B) {
	w := &mockWriter{}
	s := &Screen{
		width:  120,
		height: 40,
		back:   NewBuffer(120, 40),
		front:  NewBuffer(120, 40),
		buf:    bytes.Buffer{},
		writer: w,
	}
	style := Style{FG: Green, BG: Black}
	for y := range 40 {
		for x := range 120 {
			s.back.Set(x, y, Cell{Rune: 'A', Style: style})
		}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.front.Clear()
		s.back.MarkAllDirty()
		s.Flush()
		w.n = 0
		s.FlushBuffer()
	}
	b.ReportMetric(float64(w.n), "bytes/op")
}

// BenchmarkFlushColor16ForceRGB flushes the same screen with forceRGB=true.
// measures the cost of upgrading Color16 → true color at flush time.
func BenchmarkFlushColor16ForceRGB(b *testing.B) {
	w := &mockWriter{}
	s := &Screen{
		width:    120,
		height:   40,
		back:     NewBuffer(120, 40),
		front:    NewBuffer(120, 40),
		buf:      bytes.Buffer{},
		writer:   w,
		forceRGB: true,
	}
	style := Style{FG: Green, BG: Black}
	for y := range 40 {
		for x := range 120 {
			s.back.Set(x, y, Cell{Rune: 'A', Style: style})
		}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.front.Clear()
		s.back.MarkAllDirty()
		s.Flush()
		w.n = 0
		s.FlushBuffer()
	}
	b.ReportMetric(float64(w.n), "bytes/op")
}

// BenchmarkResolveColor16Pass measures the pre-pass that patches Color16 RGB values.
func BenchmarkResolveColor16Pass(b *testing.B) {
	buf := NewBuffer(120, 40)
	style := Style{FG: Green, BG: Yellow}
	for y := range 40 {
		for x := range 120 {
			buf.Set(x, y, Cell{Rune: 'A', Style: style})
		}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resolveColor16(buf, 120, 40)
	}
}

// BenchmarkVignetteTransition measures the first-frame cost when vignette activates —
// full screen rewrite with unique true-color per cell.
func BenchmarkVignetteTransition(b *testing.B) {
	w := &mockWriter{}
	s := &Screen{
		width:    120,
		height:   40,
		back:     NewBuffer(120, 40),
		front:    NewBuffer(120, 40),
		buf:      bytes.Buffer{},
		writer:   w,
		forceRGB: true,
	}
	// pre-compute the vignette output into a stable buffer
	renderBuf := NewBuffer(120, 40)
	style := Style{FG: Green, BG: Black}
	for y := range 40 {
		for x := range 120 {
			renderBuf.Set(x, y, Cell{Rune: 'A', Style: style})
		}
	}
	resolveColor16(renderBuf, 120, 40)
	PPVignette(1.0)(renderBuf, PostContext{Width: 120, Height: 40})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// simulate: front has pre-vignette state, back has post-vignette
		s.front.Clear()
		copy(s.back.cells, renderBuf.cells)
		s.back.MarkAllDirty()
		s.Flush()
		w.n = 0
		s.FlushBuffer()
	}
	b.ReportMetric(float64(w.n), "bytes/op")
}

// BenchmarkVignetteSteadyState measures the per-frame cost when vignette is stable —
// diff should find nothing changed and write nothing.
func BenchmarkVignetteSteadyState(b *testing.B) {
	w := &mockWriter{}
	s := &Screen{
		width:    120,
		height:   40,
		back:     NewBuffer(120, 40),
		front:    NewBuffer(120, 40),
		buf:      bytes.Buffer{},
		writer:   w,
		forceRGB: true,
	}
	// pre-compute vignette output
	renderBuf := NewBuffer(120, 40)
	style := Style{FG: Green, BG: Black}
	for y := range 40 {
		for x := range 120 {
			renderBuf.Set(x, y, Cell{Rune: 'A', Style: style})
		}
	}
	resolveColor16(renderBuf, 120, 40)
	PPVignette(1.0)(renderBuf, PostContext{Width: 120, Height: 40})

	// prime both buffers with the same vignette output (steady state)
	copy(s.back.cells, renderBuf.cells)
	copy(s.front.cells, renderBuf.cells)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		copy(s.back.cells, renderBuf.cells)
		s.back.MarkAllDirty()
		s.Flush()
		w.n = 0
		s.FlushBuffer()
	}
	b.ReportMetric(float64(w.n), "bytes/op")
}

// BenchmarkPlasmaFrame measures one animated plasma frame — every cell gets a
// unique quantized RGB per frame. Measures steady-state throughput (frames/sec proxy).
func BenchmarkPlasmaFrame(b *testing.B) {
	w := &mockWriter{}
	s := &Screen{
		width:    120,
		height:   40,
		back:     NewBuffer(120, 40),
		front:    NewBuffer(120, 40),
		buf:      bytes.Buffer{},
		writer:   w,
		forceRGB: true,
	}
	style := Style{FG: RGB(100, 100, 100), BG: RGB(10, 10, 30)}
	for y := range 40 {
		for x := range 120 {
			s.back.Set(x, y, Cell{Rune: 'A', Style: style})
			s.front.Set(x, y, Cell{Rune: 'A', Style: style})
		}
	}
	plasma := PPPlasma(0.6)
	ctx := PostContext{Width: 120, Height: 40}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// new frame: render content into a fresh buffer, apply plasma, flush
		renderBuf := NewBuffer(120, 40)
		for y := range 40 {
			for x := range 120 {
				renderBuf.Set(x, y, Cell{Rune: 'A', Style: style})
			}
		}
		ctx.Time += 33 * 1e6 // 33ms advance
		plasma(renderBuf, ctx)
		copy(s.back.cells, renderBuf.cells)
		s.back.MarkAllDirty()
		s.front.Clear() // simulate prior frame having different content
		s.Flush()
		w.n = 0
		s.FlushBuffer()
	}
	b.ReportMetric(float64(w.n), "bytes/op")
}

// ---------------------------------------------------------------------------
// Phase-isolation benchmarks — proves where frame time actually goes.
//
// The three phases of a rendered frame:
//   Phase 1 — Effect:  resolveColor16 + PostProcess passes (pure Go, CPU-bound)
//   Phase 2 — Diff:    Flush() — cell comparison + escape-sequence building (pure Go)
//   Phase 3 — Write:   FlushBuffer() — single Write() syscall to terminal (I/O-bound)
//
// mockWriter makes Phase 3 essentially free (~ns), so BenchmarkPlasmaFrame measures
// Phase 1+2 only. Comparing that against a 33ms frame budget (30fps) shows how much
// headroom our Go code leaves for the terminal.
// ---------------------------------------------------------------------------

// BenchmarkPlasmaComputeOnly measures only Phase 1: effect computation.
// No diff, no write. Pure Go CPU cost.
func BenchmarkPlasmaComputeOnly(b *testing.B) {
	buf := NewBuffer(120, 40)
	style := Style{FG: RGB(204, 204, 204), BG: RGB(10, 10, 30)}
	for y := range 40 {
		for x := range 120 {
			buf.Set(x, y, Cell{Rune: 'A', Style: style})
		}
	}
	plasma := PPPlasma(0.6)
	ctx := PostContext{Width: 120, Height: 40}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx.Time += 33 * 1e6
		plasma(buf, ctx)
	}
}

// BenchmarkPlasmaFlushBuildOnly measures only Phase 2: diff + escape building.
// Uses a pre-computed plasma frame so effect cost is excluded.
// mockWriter makes the Write free, isolating the diff algorithm.
func BenchmarkPlasmaFlushBuildOnly(b *testing.B) {
	w := &mockWriter{}
	s := &Screen{
		width:    120,
		height:   40,
		back:     NewBuffer(120, 40),
		front:    NewBuffer(120, 40),
		buf:      bytes.Buffer{},
		writer:   w,
		forceRGB: true,
	}
	// pre-build a plasma frame
	style := Style{FG: RGB(204, 204, 204), BG: RGB(10, 10, 30)}
	renderBuf := NewBuffer(120, 40)
	for y := range 40 {
		for x := range 120 {
			renderBuf.Set(x, y, Cell{Rune: 'A', Style: style})
		}
	}
	PPPlasma(0.6)(renderBuf, PostContext{Width: 120, Height: 40, Time: 1e9})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		copy(s.back.cells, renderBuf.cells)
		s.back.MarkAllDirty()
		s.front.Clear()
		s.Flush() // diff + escape build only
		w.n = 0
		s.FlushBuffer() // mockWriter — free
	}
	b.ReportMetric(float64(w.n), "bytes/op")
}

// BenchmarkWriteThroughput measures Phase 3 in isolation: how fast can we push
// bytes through Write() when the terminal is the only variable.
// Run with mockWriter (this file) vs piped to /dev/null at the shell to compare:
//   go test -bench=BenchmarkWriteThroughput -benchtime=3s | tee /dev/null
// The difference between the two is terminal parsing overhead.
func BenchmarkWriteThroughput(b *testing.B) {
	// realistic plasma-frame-sized payload: pre-built escape sequence buffer
	w := &mockWriter{}
	s := &Screen{
		width:    120,
		height:   40,
		back:     NewBuffer(120, 40),
		front:    NewBuffer(120, 40),
		buf:      bytes.Buffer{},
		writer:   w,
		forceRGB: true,
	}
	style := Style{FG: RGB(204, 204, 204), BG: RGB(10, 10, 30)}
	renderBuf := NewBuffer(120, 40)
	for y := range 40 {
		for x := range 120 {
			renderBuf.Set(x, y, Cell{Rune: 'A', Style: style})
		}
	}
	PPPlasma(0.6)(renderBuf, PostContext{Width: 120, Height: 40, Time: 1e9})
	copy(s.back.cells, renderBuf.cells)
	s.back.MarkAllDirty()
	s.front.Clear()
	s.Flush()
	prebuilt := make([]byte, s.buf.Len())
	copy(prebuilt, s.buf.Bytes())

	b.SetBytes(int64(len(prebuilt)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w.Write(prebuilt) // isolated Write — no Go computation
	}
	b.ReportMetric(float64(w.n)/float64(b.N), "bytes/op")
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
