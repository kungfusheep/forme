package glyph

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestLogBasic(t *testing.T) {
	// create a reader with some lines
	input := "line 1\nline 2\nline 3\n"
	reader := strings.NewReader(input)

	lv := Log(reader).MaxLines(100)

	// compile it to start the reader goroutine
	Build(VBox(lv))

	// give goroutine time to read
	time.Sleep(50 * time.Millisecond)

	// check internal state
	lv.mu.Lock()
	lineCount := len(lv.lines)
	lines := make([]string, len(lv.lines))
	copy(lines, lv.lines)
	lv.mu.Unlock()

	if lineCount != 3 {
		t.Errorf("expected 3 lines, got %d", lineCount)
	}
	if lines[0] != "line 1" {
		t.Errorf("expected lines[0]='line 1', got %q", lines[0])
	}
	if lines[1] != "line 2" {
		t.Errorf("expected lines[1]='line 2', got %q", lines[1])
	}
	if lines[2] != "line 3" {
		t.Errorf("expected lines[2]='line 3', got %q", lines[2])
	}

	// also check the layer got the lines
	layerBuf := lv.Layer().Buffer()
	if layerBuf == nil {
		t.Fatal("layer buffer is nil")
	}
	layerLine0 := layerBuf.GetLine(0)
	if !strings.Contains(layerLine0, "line 1") {
		t.Errorf("expected layer line 0 to contain 'line 1', got %q", layerLine0)
	}
}

func TestLogRingBuffer(t *testing.T) {
	// create a pipe for streaming
	pr, pw := io.Pipe()

	lv := Log(pr).MaxLines(3)

	// compile to start reader
	Build(VBox(lv))

	// write 5 lines - should only keep last 3
	for i := 1; i <= 5; i++ {
		pw.Write([]byte("line " + string(rune('0'+i)) + "\n"))
	}

	time.Sleep(50 * time.Millisecond)

	// check internal state
	lv.mu.Lock()
	lineCount := len(lv.lines)
	var firstLine, lastLine string
	if lineCount > 0 {
		firstLine = lv.lines[0]
		lastLine = lv.lines[lineCount-1]
	}
	lv.mu.Unlock()

	if lineCount != 3 {
		t.Errorf("expected 3 lines in buffer, got %d", lineCount)
	}
	if firstLine != "line 3" {
		t.Errorf("expected first line 'line 3', got %q", firstLine)
	}
	if lastLine != "line 5" {
		t.Errorf("expected last line 'line 5', got %q", lastLine)
	}

	pw.Close()
}

func TestLogScrollControl(t *testing.T) {
	input := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	reader := strings.NewReader(input)

	lv := Log(reader).MaxLines(100)
	layer := lv.Layer()

	// compile and wait for read
	Build(VBox(lv))
	time.Sleep(50 * time.Millisecond)

	// set viewport so we can scroll
	layer.SetViewport(40, 2)

	// should be able to scroll
	layer.ScrollDown(1)
	if layer.ScrollY() != 1 {
		t.Errorf("expected scrollY=1, got %d", layer.ScrollY())
	}

	layer.ScrollToEnd()
	// with 5 lines and viewport of 2, max scroll is 3
	if layer.ScrollY() != 3 {
		t.Errorf("expected scrollY=3 at end, got %d", layer.ScrollY())
	}

	layer.ScrollToTop()
	if layer.ScrollY() != 0 {
		t.Errorf("expected scrollY=0 at top, got %d", layer.ScrollY())
	}
}

func TestLogAutoScroll(t *testing.T) {
	pr, pw := io.Pipe()

	lv := Log(pr).MaxLines(100).AutoScroll(true)
	layer := lv.Layer()

	Build(VBox(lv))

	// set viewport
	layer.SetViewport(40, 2)

	// write lines and check auto-scroll
	for i := 1; i <= 5; i++ {
		pw.Write([]byte("line\n"))
		time.Sleep(20 * time.Millisecond)
	}

	// should have auto-scrolled to show latest
	// with 5 lines and viewport 2, if at bottom, scrollY should be near maxScroll
	if layer.ScrollY() < 2 {
		t.Errorf("expected auto-scroll to bottom, scrollY=%d", layer.ScrollY())
	}

	pw.Close()
}
