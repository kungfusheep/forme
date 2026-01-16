package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/term"
	"tui"
)

type Process struct {
	PID   int
	Name  string
	CPU   float32
	Count int
}

func main() {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 80, 24
	}

	// Leave room for header (2) + footer (1) + safety (1)
	visibleRows := height - 4
	totalRows := 1000
	contentWidth := 50 // Fixed content width, don't fill full terminal

	procs := make([]Process, totalRows)
	for i := range procs {
		procs[i] = Process{
			PID:   i,
			Name:  fmt.Sprintf("proc-%04d", i),
			CPU:   float32(i%100) / 100.0,
			Count: i * 100,
		}
	}

	buf := tui.NewBuffer(contentWidth, visibleRows)

	var mu sync.Mutex
	viewportY := 0
	running := true

	// Switch to alternate screen, hide cursor
	os.Stdout.WriteString("\033[?1049h\033[?25l\033[2J")
	defer os.Stdout.WriteString("\033[?25h\033[?1049l")

	// Raw mode
	oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Input
	go func() {
		b := make([]byte, 3)
		for running {
			n, _ := os.Stdin.Read(b)
			if n > 0 {
				mu.Lock()
				switch {
				case b[0] == 'q' || b[0] == 3:
					running = false
				case b[0] == 'j':
					viewportY += 5
					if viewportY > totalRows-visibleRows {
						viewportY = totalRows - visibleRows
					}
				case b[0] == 'k':
					viewportY -= 5
					if viewportY < 0 {
						viewportY = 0
					}
				case b[0] == 'g':
					viewportY = 0
				case b[0] == 'G':
					viewportY = totalRows - visibleRows
				}
				mu.Unlock()
			}
		}
	}()

	// Pre-allocate
	outBuf := make([]byte, 0, 32768)

	frameCount := 0
	lastFPS := time.Now()
	fps := 0.0
	_ = width // silence unused

	for running {
		time.Sleep(33 * time.Millisecond)

		for i := range procs {
			procs[i].Count++
		}

		mu.Lock()
		vp := viewportY
		mu.Unlock()

		start := time.Now()
		buf.Clear()

		endRow := vp + visibleRows
		if endRow > totalRows {
			endRow = totalRows
		}

		for i := vp; i < endRow; i++ {
			p := &procs[i]
			y := i - vp
			writeIntPadded(buf, 0, y, p.PID, 5)
			buf.WriteStringFast(6, y, p.Name, tui.Style{}, 10)
			buf.WriteProgressBar(17, y, 8, p.CPU, tui.Style{})
			writeIntPadded(buf, 26, y, p.Count, 10)
		}
		renderTime := time.Since(start)

		frameCount++
		if time.Since(lastFPS) >= time.Second {
			fps = float64(frameCount)
			frameCount = 0
			lastFPS = time.Now()
		}

		// Build frame
		outBuf = outBuf[:0]
		outBuf = append(outBuf, "\033[?2026h\033[H"...)

		// Row 1: header
		outBuf = append(outBuf, "=== Live Demo: "...)
		outBuf = strconv.AppendInt(outBuf, int64(totalRows), 10)
		outBuf = append(outBuf, " rows, "...)
		outBuf = strconv.AppendInt(outBuf, int64(visibleRows), 10)
		outBuf = append(outBuf, " visible ===\033[K\r\n"...)

		// Row 2: column headers
		outBuf = append(outBuf, "  PID Name       CPU      Count\033[K\r\n"...)

		// Content rows
		for y := 0; y < visibleRows; y++ {
			for x := 0; x < contentWidth; x++ {
				cell := buf.Get(x, y)
				if cell.Rune == 0 {
					outBuf = append(outBuf, ' ')
				} else if cell.Rune < 128 {
					outBuf = append(outBuf, byte(cell.Rune))
				} else {
					var rb [4]byte
					n := encodeRune(rb[:], cell.Rune)
					outBuf = append(outBuf, rb[:n]...)
				}
			}
			outBuf = append(outBuf, "\033[K\r\n"...)
		}

		// Footer
		outBuf = append(outBuf, "Row "...)
		outBuf = strconv.AppendInt(outBuf, int64(vp+1), 10)
		outBuf = append(outBuf, "-"...)
		outBuf = strconv.AppendInt(outBuf, int64(endRow), 10)
		outBuf = append(outBuf, " | "...)
		outBuf = append(outBuf, renderTime.String()...)
		outBuf = append(outBuf, " | FPS:"...)
		outBuf = strconv.AppendInt(outBuf, int64(fps), 10)
		outBuf = append(outBuf, " | j/k/g/G/q\033[K"...)
		outBuf = append(outBuf, "\033[?2026l"...)

		os.Stdout.Write(outBuf)
	}
}

func writeIntPadded(buf *tui.Buffer, x, y, val, w int) {
	s := strconv.Itoa(val)
	pad := w - len(s)
	for i := 0; i < pad; i++ {
		buf.Set(x+i, y, tui.Cell{Rune: ' '})
	}
	buf.WriteStringFast(x+pad, y, s, tui.Style{}, len(s))
}

func encodeRune(b []byte, r rune) int {
	if r < 0x80 {
		b[0] = byte(r)
		return 1
	} else if r < 0x800 {
		b[0] = 0xC0 | byte(r>>6)
		b[1] = 0x80 | byte(r&0x3F)
		return 2
	} else if r < 0x10000 {
		b[0] = 0xE0 | byte(r>>12)
		b[1] = 0x80 | byte((r>>6)&0x3F)
		b[2] = 0x80 | byte(r&0x3F)
		return 3
	}
	b[0] = 0xF0 | byte(r>>18)
	b[1] = 0x80 | byte((r>>12)&0x3F)
	b[2] = 0x80 | byte((r>>6)&0x3F)
	b[3] = 0x80 | byte(r&0x3F)
	return 4
}
