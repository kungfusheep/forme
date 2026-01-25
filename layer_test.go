package tui

import (
	"testing"
)

func TestLayerBlit(t *testing.T) {
	t.Run("single layer blits to correct position", func(t *testing.T) {
		// Create a layer with content
		layer := NewLayer()
		layerBuf := NewBuffer(10, 5)
		for y := 0; y < 5; y++ {
			layerBuf.WriteStringFast(0, y, string(rune('A'+y))+"----", Style{}, 10)
		}
		layer.SetBuffer(layerBuf)

		// Create screen and view
		screen := NewBuffer(20, 10)

		// Build view with layer at position
		view := VBoxNode{Children: []any{
			TextNode{Content: "Header"},
			LayerViewNode{Layer: layer, ViewHeight: 3},
			TextNode{Content: "Footer"},
		}}

		tmpl := Build(view)
		tmpl.Execute(screen, 20, 10)

		// Verify header at line 0
		if got := screen.GetLine(0); got != "Header" {
			t.Errorf("line 0: got %q, want %q", got, "Header")
		}

		// Verify layer content at lines 1-3
		if got := screen.GetLine(1); got != "A----" {
			t.Errorf("line 1: got %q, want %q", got, "A----")
		}
		if got := screen.GetLine(2); got != "B----" {
			t.Errorf("line 2: got %q, want %q", got, "B----")
		}
		if got := screen.GetLine(3); got != "C----" {
			t.Errorf("line 3: got %q, want %q", got, "C----")
		}

		// Verify footer at line 4
		if got := screen.GetLine(4); got != "Footer" {
			t.Errorf("line 4: got %q, want %q", got, "Footer")
		}
	})

	t.Run("multiple layers blit to correct positions", func(t *testing.T) {
		// Create first layer
		layer1 := NewLayer()
		buf1 := NewBuffer(10, 5)
		for y := 0; y < 5; y++ {
			buf1.WriteStringFast(0, y, "111111", Style{}, 10)
		}
		layer1.SetBuffer(buf1)

		// Create second layer
		layer2 := NewLayer()
		buf2 := NewBuffer(10, 5)
		for y := 0; y < 5; y++ {
			buf2.WriteStringFast(0, y, "222222", Style{}, 10)
		}
		layer2.SetBuffer(buf2)

		// Create third layer
		layer3 := NewLayer()
		buf3 := NewBuffer(10, 5)
		for y := 0; y < 5; y++ {
			buf3.WriteStringFast(0, y, "333333", Style{}, 10)
		}
		layer3.SetBuffer(buf3)

		screen := NewBuffer(20, 15)

		view := VBoxNode{Children: []any{
			TextNode{Content: "=TOP="},
			LayerViewNode{Layer: layer1, ViewHeight: 2},
			TextNode{Content: "=MID1="},
			LayerViewNode{Layer: layer2, ViewHeight: 2},
			TextNode{Content: "=MID2="},
			LayerViewNode{Layer: layer3, ViewHeight: 2},
			TextNode{Content: "=BOT="},
		}}

		tmpl := Build(view)
		tmpl.Execute(screen, 20, 15)

		expected := []struct {
			line int
			want string
		}{
			{0, "=TOP="},
			{1, "111111"},
			{2, "111111"},
			{3, "=MID1="},
			{4, "222222"},
			{5, "222222"},
			{6, "=MID2="},
			{7, "333333"},
			{8, "333333"},
			{9, "=BOT="},
		}

		for _, tc := range expected {
			if got := screen.GetLine(tc.line); got != tc.want {
				t.Errorf("line %d: got %q, want %q", tc.line, got, tc.want)
			}
		}
	})

	t.Run("layers scroll independently", func(t *testing.T) {
		// Create two layers with different content
		layer1 := NewLayer()
		buf1 := NewBuffer(10, 10)
		for y := 0; y < 10; y++ {
			buf1.WriteStringFast(0, y, string(rune('A'+y))+"AAAA", Style{}, 10)
		}
		layer1.SetBuffer(buf1)

		layer2 := NewLayer()
		buf2 := NewBuffer(10, 10)
		for y := 0; y < 10; y++ {
			buf2.WriteStringFast(0, y, string(rune('0'+y))+"0000", Style{}, 10)
		}
		layer2.SetBuffer(buf2)

		screen := NewBuffer(20, 10)

		view := VBoxNode{Children: []any{
			LayerViewNode{Layer: layer1, ViewHeight: 3},
			TextNode{Content: "---"},
			LayerViewNode{Layer: layer2, ViewHeight: 3},
		}}

		tmpl := Build(view)

		// Initial render - both at scroll 0
		tmpl.Execute(screen, 20, 10)

		if got := screen.GetLine(0); got != "AAAAA" {
			t.Errorf("initial layer1 line 0: got %q, want %q", got, "AAAAA")
		}
		if got := screen.GetLine(4); got != "00000" {
			t.Errorf("initial layer2 line 4: got %q, want %q", got, "00000")
		}

		// Scroll layer1 down by 2
		layer1.ScrollDown(2)
		tmpl.Execute(screen, 20, 10)

		// Layer1 should now show C, D, E (indices 2, 3, 4)
		if got := screen.GetLine(0); got != "CAAAA" {
			t.Errorf("after scroll layer1 line 0: got %q, want %q", got, "CAAAA")
		}
		if got := screen.GetLine(1); got != "DAAAA" {
			t.Errorf("after scroll layer1 line 1: got %q, want %q", got, "DAAAA")
		}

		// Layer2 should still be at scroll 0
		if got := screen.GetLine(4); got != "00000" {
			t.Errorf("layer2 should be unchanged: got %q, want %q", got, "00000")
		}

		// Now scroll layer2
		layer2.ScrollDown(5)
		tmpl.Execute(screen, 20, 10)

		// Layer2 should now show 5, 6, 7
		if got := screen.GetLine(4); got != "50000" {
			t.Errorf("after scroll layer2 line 4: got %q, want %q", got, "50000")
		}

		// Layer1 should still be at its scroll position
		if got := screen.GetLine(0); got != "CAAAA" {
			t.Errorf("layer1 should be unchanged: got %q, want %q", got, "CAAAA")
		}
	})

	t.Run("layer with nil buffer renders empty", func(t *testing.T) {
		layer := NewLayer()
		// Don't set any buffer

		screen := NewBuffer(20, 5)

		view := VBoxNode{Children: []any{
			TextNode{Content: "Before"},
			LayerViewNode{Layer: layer, ViewHeight: 2},
			TextNode{Content: "After"},
		}}

		tmpl := Build(view)
		screen.Clear()
		tmpl.Execute(screen, 20, 5)

		// Text should render - key is it shouldn't crash with nil buffer
		if got := screen.GetLine(0); got != "Before" {
			t.Errorf("line 0: got %q, want %q", got, "Before")
		}

		// After should be at line 3 (0=Before, 1-2=layer, 3=After)
		if got := screen.GetLine(3); got != "After" {
			t.Errorf("line 3: got %q, want %q", got, "After")
		}
	})
}

func TestLayerScrollBounds(t *testing.T) {
	t.Run("scroll clamps to bounds", func(t *testing.T) {
		layer := NewLayer()
		buf := NewBuffer(10, 20) // 20 lines of content
		layer.SetBuffer(buf)
		layer.SetViewport(10, 5) // 5 line viewport

		// MaxScroll should be 20 - 5 = 15
		if got := layer.MaxScroll(); got != 15 {
			t.Errorf("MaxScroll: got %d, want 15", got)
		}

		// Scroll past end should clamp
		layer.ScrollTo(100)
		if got := layer.ScrollY(); got != 15 {
			t.Errorf("ScrollY after overflow: got %d, want 15", got)
		}

		// Scroll before start should clamp
		layer.ScrollTo(-10)
		if got := layer.ScrollY(); got != 0 {
			t.Errorf("ScrollY after underflow: got %d, want 0", got)
		}
	})

	t.Run("page scroll methods", func(t *testing.T) {
		layer := NewLayer()
		buf := NewBuffer(10, 100)
		layer.SetBuffer(buf)
		layer.SetViewport(10, 10)

		layer.PageDown()
		if got := layer.ScrollY(); got != 10 {
			t.Errorf("after PageDown: got %d, want 10", got)
		}

		layer.PageUp()
		if got := layer.ScrollY(); got != 0 {
			t.Errorf("after PageUp: got %d, want 0", got)
		}

		layer.HalfPageDown()
		if got := layer.ScrollY(); got != 5 {
			t.Errorf("after HalfPageDown: got %d, want 5", got)
		}

		layer.ScrollToEnd()
		if got := layer.ScrollY(); got != 90 {
			t.Errorf("after ScrollToEnd: got %d, want 90", got)
		}

		layer.ScrollToTop()
		if got := layer.ScrollY(); got != 0 {
			t.Errorf("after ScrollToTop: got %d, want 0", got)
		}
	})
}

// BenchmarkLayerWithCursor measures rendering a layer with cursor tracking.
func BenchmarkLayerWithCursor(b *testing.B) {
	layer := NewLayer()
	buf := NewBuffer(80, 100)
	for y := 0; y < 100; y++ {
		buf.WriteStringFast(0, y, "Line content here", Style{}, 80)
	}
	layer.SetBuffer(buf)
	layer.SetViewport(80, 24)
	layer.ShowCursor()
	layer.SetCursorStyle(CursorBlock)

	screen := NewBuffer(80, 24)

	view := VBoxNode{Children: []any{
		LayerViewNode{Layer: layer, ViewHeight: 24},
	}}
	tmpl := Build(view)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// simulate cursor movement each frame
		layer.SetCursor(i%80, (i/80)%100)
		screen.ClearDirty()
		tmpl.Execute(screen, 80, 24)
	}
}

// BenchmarkLayerScrollingWithCursor measures scrolling + cursor updates.
func BenchmarkLayerScrollingWithCursor(b *testing.B) {
	layer := NewLayer()
	buf := NewBuffer(80, 1000)
	for y := 0; y < 1000; y++ {
		buf.WriteStringFast(0, y, "Line content that we scroll through", Style{}, 80)
	}
	layer.SetBuffer(buf)
	layer.SetViewport(80, 24)
	layer.ShowCursor()

	screen := NewBuffer(80, 24)

	view := VBoxNode{Children: []any{
		LayerViewNode{Layer: layer, ViewHeight: 24},
	}}
	tmpl := Build(view)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		layer.ScrollTo(i % 976) // scroll within bounds
		layer.SetCursor(i%80, layer.ScrollY()+(i%24))
		screen.ClearDirty()
		tmpl.Execute(screen, 80, 24)
	}
}

// BenchmarkLayerCursorScreenTranslation measures ScreenCursor() translation.
func BenchmarkLayerCursorScreenTranslation(b *testing.B) {
	layer := NewLayer()
	layer.SetViewport(80, 24)
	layer.ShowCursor()

	// simulate being positioned at screen offset
	layer.screenX = 10
	layer.screenY = 5

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		layer.SetCursor(i%80, i%100)
		layer.scrollY = (i / 10) % 50
		_, _, _ = layer.ScreenCursor()
	}
}
