package glyph

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func newTestScreen(w, h int) (*Screen, *bytes.Buffer) {
	var out bytes.Buffer
	s := &Screen{
		width:  w,
		height: h,
		back:   NewBuffer(w, h),
		front:  NewBuffer(w, h),
		buf:    bytes.Buffer{},
		writer: &out,
	}
	return s, &out
}

func TestFlushInline(t *testing.T) {
	t.Run("clears stale lines when content shrinks", func(t *testing.T) {
		s, out := newTestScreen(40, 20)

		// Frame 1: render 5 lines of content
		for y := 0; y < 5; y++ {
			for x := 0; x < 5; x++ {
				s.back.Set(x, y, Cell{Rune: 'A', Style: DefaultStyle()})
			}
		}
		lines1 := s.FlushInline(5, 0)
		if lines1 != 5 {
			t.Fatalf("expected 5 lines rendered, got %d", lines1)
		}

		// Frame 2: render only 2 lines (simulating a filter reducing results)
		s.back.Clear()
		for y := 0; y < 2; y++ {
			for x := 0; x < 3; x++ {
				s.back.Set(x, y, Cell{Rune: 'B', Style: DefaultStyle()})
			}
		}
		out.Reset()
		lines2 := s.FlushInline(2, lines1)
		if lines2 != 2 {
			t.Fatalf("expected 2 lines rendered, got %d", lines2)
		}

		// 2 for the content lines + 3 for the stale lines = 5 total
		clearCount := strings.Count(out.String(), "\x1b[K")
		if clearCount != 5 {
			t.Errorf("expected 5 clear-line sequences (2 content + 3 stale), got %d", clearCount)
		}
	})

	t.Run("cursor returns to first line after shrinking", func(t *testing.T) {
		s, out := newTestScreen(40, 20)

		// Frame 1: render 5 lines
		for y := 0; y < 5; y++ {
			for x := 0; x < 3; x++ {
				s.back.Set(x, y, Cell{Rune: 'A', Style: DefaultStyle()})
			}
		}
		s.FlushInline(5, 0)

		// Frame 2: shrink to 2 lines
		s.back.Clear()
		for y := 0; y < 2; y++ {
			for x := 0; x < 3; x++ {
				s.back.Set(x, y, Cell{Rune: 'B', Style: DefaultStyle()})
			}
		}
		out.Reset()
		s.FlushInline(2, 5)

		// Cursor must move up by max(2,5)-1 = 4 lines to return to line 0.
		// A bug here would use linesRendered-1 = 1, leaving the cursor
		// stranded 3 rows too low and causing subsequent frames to shift.
		output := out.String()
		moveUp4 := fmt.Sprintf("\x1b[%dA", 4)
		if !strings.Contains(output, moveUp4) {
			t.Errorf("expected cursor-up by 4 (%q) in output, got %q", moveUp4, output)
		}
	})
}
