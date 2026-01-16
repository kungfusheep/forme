package main

import (
	"fmt"
	"runtime"
	"time"

	"tui"
)

const (
	screenWidth  = 120
	screenHeight = 40
	contentLines = 1000 // Simulated large content (like a webpage)
	viewHeight   = 30   // Visible viewport
	frames       = 1000
)

var showVisual = true // Set to true to see rendered output

func main() {
	fmt.Println("=== Layer/Blit Performance Test ===")
	fmt.Printf("Screen: %dx%d, Content: %d lines, Viewport: %d lines\n\n",
		screenWidth, screenHeight, contentLines, viewHeight)

	// Warm up
	runtime.GC()

	// Test 1: Full re-render every frame (baseline)
	fmt.Println("Test 1: Full re-render every frame")
	testFullRerender()

	runtime.GC()

	// Test 2: Pre-rendered layer + blit
	fmt.Println("\nTest 2: Pre-rendered layer + blit (scroll via srcY)")
	testLayerBlit()

	runtime.GC()

	// Test 3: Layer blit + dynamic overlay
	fmt.Println("\nTest 3: Layer blit + dynamic region (simulated cursor/status)")
	testLayerWithDynamic()

	runtime.GC()

	// Test 4: Measure layer creation cost
	fmt.Println("\nTest 4: Layer creation cost (simulated page navigation)")
	testLayerCreation()
}

func showBuffer(label string, buf *tui.Buffer, maxLines int) {
	if !showVisual {
		return
	}
	fmt.Printf("\n  --- %s ---\n", label)
	for y := 0; y < maxLines && y < buf.Height(); y++ {
		fmt.Print("  │")
		for x := 0; x < buf.Width(); x++ {
			cell := buf.Get(x, y)
			if cell.Rune == 0 {
				fmt.Print(" ")
			} else {
				fmt.Print(string(cell.Rune))
			}
		}
		fmt.Println("│")
	}
	if buf.Height() > maxLines {
		fmt.Printf("  │... (%d more lines) ...│\n", buf.Height()-maxLines)
	}
}

// generateContent creates styled content lines (simulating rendered HTML)
func generateContent() []tui.Span {
	spans := make([]tui.Span, 0, contentLines*3)
	for i := 0; i < contentLines; i++ {
		// Simulate mixed styled content like a webpage
		spans = append(spans,
			tui.Span{Text: fmt.Sprintf("Line %04d: ", i+1), Style: tui.Style{Attr: tui.AttrBold}},
			tui.Span{Text: "The quick brown ", Style: tui.Style{FG: tui.White}},
			tui.Span{Text: "fox", Style: tui.Style{FG: tui.Red, Attr: tui.AttrBold}},
		)
	}
	return spans
}

// testFullRerender: render all visible content every frame
func testFullRerender() {
	screen := tui.NewBuffer(screenWidth, screenHeight)
	content := generateContent()
	spansPerLine := 3

	// Show a single frame first
	screen.Clear()
	scrollY := 50 // Show from line 50
	for y := 0; y < viewHeight; y++ {
		lineIdx := scrollY + y
		if lineIdx >= contentLines {
			break
		}
		spanStart := lineIdx * spansPerLine
		spanEnd := spanStart + spansPerLine
		if spanEnd > len(content) {
			spanEnd = len(content)
		}
		screen.WriteSpans(0, y, content[spanStart:spanEnd], screenWidth)
	}
	showBuffer(fmt.Sprintf("Screen at scrollY=%d", scrollY), screen, 8)

	var totalAllocs uint64
	var m runtime.MemStats

	runtime.ReadMemStats(&m)
	startAllocs := m.Mallocs

	start := time.Now()
	for frame := 0; frame < frames; frame++ {
		screen.Clear()

		// Simulate scroll position
		scrollY := (frame * 2) % (contentLines - viewHeight)

		// Render visible lines
		for y := 0; y < viewHeight; y++ {
			lineIdx := scrollY + y
			if lineIdx >= contentLines {
				break
			}
			spanStart := lineIdx * spansPerLine
			spanEnd := spanStart + spansPerLine
			if spanEnd > len(content) {
				spanEnd = len(content)
			}
			screen.WriteSpans(0, y, content[spanStart:spanEnd], screenWidth)
		}
	}
	elapsed := time.Since(start)

	runtime.ReadMemStats(&m)
	totalAllocs = m.Mallocs - startAllocs

	fps := float64(frames) / elapsed.Seconds()
	frameTime := elapsed / time.Duration(frames)
	fmt.Printf("  %d frames in %v\n", frames, elapsed)
	fmt.Printf("  %.1f FPS, %.2fms/frame\n", fps, float64(frameTime.Microseconds())/1000)
	fmt.Printf("  Allocations: %d total, %.1f per frame\n", totalAllocs, float64(totalAllocs)/float64(frames))
}

// testLayerBlit: pre-render to layer, blit visible portion each frame
func testLayerBlit() {
	screen := tui.NewBuffer(screenWidth, screenHeight)
	content := generateContent()
	spansPerLine := 3

	// Pre-render ALL content to layer (like uploading texture)
	layer := tui.NewBuffer(screenWidth, contentLines)
	for y := 0; y < contentLines; y++ {
		spanStart := y * spansPerLine
		spanEnd := spanStart + spansPerLine
		if spanEnd > len(content) {
			spanEnd = len(content)
		}
		layer.WriteSpans(0, y, content[spanStart:spanEnd], screenWidth)
	}
	fmt.Printf("  Layer created: %dx%d (%d cells)\n", screenWidth, contentLines, screenWidth*contentLines)

	// Show the full layer (first few lines)
	showBuffer("Layer (pre-rendered, first 5 lines)", layer, 5)

	// Show blitting at different scroll positions
	screen.Blit(layer, 0, 0, 0, 0, screenWidth, viewHeight)
	showBuffer("Screen after Blit(scrollY=0)", screen, 5)

	screen.Blit(layer, 0, 500, 0, 0, screenWidth, viewHeight)
	showBuffer("Screen after Blit(scrollY=500)", screen, 5)

	var totalAllocs uint64
	var m runtime.MemStats

	runtime.ReadMemStats(&m)
	startAllocs := m.Mallocs

	start := time.Now()
	for frame := 0; frame < frames; frame++ {
		// Simulate scroll position
		scrollY := (frame * 2) % (contentLines - viewHeight)

		// Just blit the visible portion - O(viewport) not O(content)
		screen.Blit(layer, 0, scrollY, 0, 0, screenWidth, viewHeight)
	}
	elapsed := time.Since(start)

	runtime.ReadMemStats(&m)
	totalAllocs = m.Mallocs - startAllocs

	fps := float64(frames) / elapsed.Seconds()
	frameTime := elapsed / time.Duration(frames)
	fmt.Printf("  %d frames in %v\n", frames, elapsed)
	fmt.Printf("  %.1f FPS, %.2fms/frame\n", fps, float64(frameTime.Microseconds())/1000)
	fmt.Printf("  Allocations: %d total, %.1f per frame\n", totalAllocs, float64(totalAllocs)/float64(frames))
}

// testLayerWithDynamic: blit layer + render small dynamic region
func testLayerWithDynamic() {
	screen := tui.NewBuffer(screenWidth, screenHeight)
	content := generateContent()
	spansPerLine := 3

	// Pre-render layer
	layer := tui.NewBuffer(screenWidth, contentLines)
	for y := 0; y < contentLines; y++ {
		spanStart := y * spansPerLine
		spanEnd := spanStart + spansPerLine
		if spanEnd > len(content) {
			spanEnd = len(content)
		}
		layer.WriteSpans(0, y, content[spanStart:spanEnd], screenWidth)
	}

	// Animation frames for dynamic content
	spinnerFrames := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

	// Show a sample frame with dynamic status bar
	scrollY := 100
	screen.Blit(layer, 0, scrollY, 0, 0, screenWidth, viewHeight)
	statusY := viewHeight // Put status bar right after viewport
	spinner := spinnerFrames[3]
	status := fmt.Sprintf(" %c Loading... | Line %d/%d ", spinner, scrollY+1, contentLines)
	screen.WriteStringFast(0, statusY, status, tui.Style{Attr: tui.AttrInverse}, screenWidth)
	showBuffer("Layer + Dynamic status bar", screen, 8)

	var totalAllocs uint64
	var m runtime.MemStats

	runtime.ReadMemStats(&m)
	startAllocs := m.Mallocs

	start := time.Now()
	for frame := 0; frame < frames; frame++ {
		scrollY := (frame * 2) % (contentLines - viewHeight)

		// Blit static layer
		screen.Blit(layer, 0, scrollY, 0, 0, screenWidth, viewHeight)

		// Render dynamic status bar (simulating cursor blink, spinner, etc.)
		statusY := screenHeight - 1
		spinner := spinnerFrames[frame%len(spinnerFrames)]
		status := fmt.Sprintf(" %c Loading... | Line %d/%d ", spinner, scrollY+1, contentLines)
		screen.WriteStringFast(0, statusY, status, tui.Style{Attr: tui.AttrInverse}, screenWidth)
	}
	elapsed := time.Since(start)

	runtime.ReadMemStats(&m)
	totalAllocs = m.Mallocs - startAllocs

	fps := float64(frames) / elapsed.Seconds()
	frameTime := elapsed / time.Duration(frames)
	fmt.Printf("  %d frames in %v\n", frames, elapsed)
	fmt.Printf("  %.1f FPS, %.2fms/frame\n", fps, float64(frameTime.Microseconds())/1000)
	fmt.Printf("  Allocations: %d total, %.1f per frame\n", totalAllocs, float64(totalAllocs)/float64(frames))
}

// testLayerCreation: measure cost of creating new layer (page navigation)
func testLayerCreation() {
	content := generateContent()
	spansPerLine := 3
	navigations := 100

	var totalAllocs uint64
	var m runtime.MemStats

	runtime.ReadMemStats(&m)
	startAllocs := m.Mallocs

	start := time.Now()
	for nav := 0; nav < navigations; nav++ {
		// Create new layer (simulating navigation to new page)
		layer := tui.NewBuffer(screenWidth, contentLines)
		for y := 0; y < contentLines; y++ {
			spanStart := y * spansPerLine
			spanEnd := spanStart + spansPerLine
			if spanEnd > len(content) {
				spanEnd = len(content)
			}
			layer.WriteSpans(0, y, content[spanStart:spanEnd], screenWidth)
		}
		_ = layer // use it
	}
	elapsed := time.Since(start)

	runtime.ReadMemStats(&m)
	totalAllocs = m.Mallocs - startAllocs

	navTime := elapsed / time.Duration(navigations)
	fmt.Printf("  %d layer creations in %v\n", navigations, elapsed)
	fmt.Printf("  %.2fms per layer creation\n", float64(navTime.Microseconds())/1000)
	fmt.Printf("  Allocations: %d total, %.1f per creation\n", totalAllocs, float64(totalAllocs)/float64(navigations))
}
