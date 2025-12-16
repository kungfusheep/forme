package tui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// Screen manages the terminal display with double buffering and diff-based updates.
type Screen struct {
	front  *Buffer   // What's currently displayed
	back   *Buffer   // What we're drawing to
	writer io.Writer // Output destination (usually os.Stdout)
	fd     int       // File descriptor for terminal operations

	width  int
	height int

	// Terminal state
	origTermios *unix.Termios
	inRawMode   bool

	// Resize handling
	resizeChan chan Size
	sigChan    chan os.Signal

	// Rendering state
	lastStyle Style      // Last style we emitted (for optimization)
	buf       bytes.Buffer // Reusable buffer for building output
}

// Size represents dimensions.
type Size struct {
	Width  int
	Height int
}

// NewScreen creates a new screen writing to the given writer.
// Pass nil to use os.Stdout.
func NewScreen(w io.Writer) (*Screen, error) {
	if w == nil {
		w = os.Stdout
	}

	fd := int(os.Stdout.Fd())
	width, height, err := getTerminalSize(fd)
	if err != nil {
		// Default fallback
		width, height = 80, 24
	}

	s := &Screen{
		front:      NewBuffer(width, height),
		back:       NewBuffer(width, height),
		writer:     w,
		fd:         fd,
		width:      width,
		height:     height,
		resizeChan: make(chan Size, 1),
		sigChan:    make(chan os.Signal, 1),
		lastStyle:  DefaultStyle(),
	}

	return s, nil
}

// getTerminalSize returns the current terminal dimensions.
func getTerminalSize(fd int) (int, int, error) {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, err
	}
	return int(ws.Col), int(ws.Row), nil
}

// Size returns the current screen dimensions.
func (s *Screen) Size() Size {
	return Size{Width: s.width, Height: s.height}
}

// Width returns the screen width.
func (s *Screen) Width() int {
	return s.width
}

// Height returns the screen height.
func (s *Screen) Height() int {
	return s.height
}

// Buffer returns the back buffer for drawing.
func (s *Screen) Buffer() *Buffer {
	return s.back
}

// ResizeChan returns a channel that receives size updates on terminal resize.
func (s *Screen) ResizeChan() <-chan Size {
	return s.resizeChan
}

// EnterRawMode puts the terminal into raw mode for TUI operation.
func (s *Screen) EnterRawMode() error {
	if s.inRawMode {
		return nil
	}

	termios, err := unix.IoctlGetTermios(s.fd, unix.TIOCGETA)
	if err != nil {
		return fmt.Errorf("failed to get termios: %w", err)
	}
	s.origTermios = termios

	raw := *termios
	// Input flags: disable break, CR to NL, parity, strip, flow control
	raw.Iflag &^= unix.BRKINT | unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXON
	// Output flags: disable post processing
	raw.Oflag &^= unix.OPOST
	// Control flags: set 8 bit chars
	raw.Cflag |= unix.CS8
	// Local flags: disable echo, canonical mode, signals, extended input
	raw.Lflag &^= unix.ECHO | unix.ICANON | unix.ISIG | unix.IEXTEN
	// Control chars: min bytes = 1, timeout = 0
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(s.fd, unix.TIOCSETA, &raw); err != nil {
		return fmt.Errorf("failed to set raw mode: %w", err)
	}

	s.inRawMode = true

	// Start listening for resize signals
	signal.Notify(s.sigChan, syscall.SIGWINCH)
	go s.handleSignals()

	// Enter alternate screen, hide cursor
	s.writeString("\x1b[?1049h") // Enter alternate screen
	s.writeString("\x1b[?25l")   // Hide cursor

	return nil
}

// ExitRawMode restores the terminal to its original state.
func (s *Screen) ExitRawMode() error {
	if !s.inRawMode {
		return nil
	}

	// Show cursor, exit alternate screen
	s.writeString("\x1b[?25h")   // Show cursor
	s.writeString("\x1b[?1049l") // Exit alternate screen

	signal.Stop(s.sigChan)

	if s.origTermios != nil {
		if err := unix.IoctlSetTermios(s.fd, unix.TIOCSETA, s.origTermios); err != nil {
			return fmt.Errorf("failed to restore termios: %w", err)
		}
	}

	s.inRawMode = false
	return nil
}

// handleSignals processes OS signals.
func (s *Screen) handleSignals() {
	for range s.sigChan {
		width, height, err := getTerminalSize(s.fd)
		if err != nil {
			continue
		}
		if width != s.width || height != s.height {
			s.width = width
			s.height = height
			s.front.Resize(width, height)
			s.back.Resize(width, height)
			// Clear front buffer to force full redraw
			s.front.Clear()
			// Non-blocking send
			select {
			case s.resizeChan <- Size{Width: width, Height: height}:
			default:
			}
		}
	}
}

// Flush renders the back buffer to the terminal, only updating changed cells.
func (s *Screen) Flush() {
	s.buf.Reset()

	// Reset cursor position
	s.buf.WriteString("\x1b[H") // Move to home position

	for y := 0; y < s.height; y++ {
		for x := 0; x < s.width; x++ {
			backCell := s.back.Get(x, y)
			frontCell := s.front.Get(x, y)

			if !backCell.Equal(frontCell) {
				// Move cursor to position
				s.buf.WriteString(fmt.Sprintf("\x1b[%d;%dH", y+1, x+1))
				// Apply style and write character
				s.writeCell(&s.buf, backCell)
				// Update front buffer
				s.front.Set(x, y, backCell)
			}
		}
	}

	// Reset style at end
	s.buf.WriteString("\x1b[0m")
	s.lastStyle = DefaultStyle()

	// Write to terminal
	s.writer.Write(s.buf.Bytes())
}

// FlushFull does a complete redraw without diffing.
func (s *Screen) FlushFull() {
	s.buf.Reset()

	// Clear screen and move to home
	s.buf.WriteString("\x1b[2J\x1b[H")

	for y := 0; y < s.height; y++ {
		for x := 0; x < s.width; x++ {
			cell := s.back.Get(x, y)
			s.writeCell(&s.buf, cell)
			s.front.Set(x, y, cell)
		}
		if y < s.height-1 {
			s.buf.WriteString("\r\n")
		}
	}

	// Reset style at end
	s.buf.WriteString("\x1b[0m")
	s.lastStyle = DefaultStyle()

	s.writer.Write(s.buf.Bytes())
}

// writeCell writes a cell's style and rune to the buffer.
func (s *Screen) writeCell(buf *bytes.Buffer, cell Cell) {
	// Only emit style changes
	if !cell.Style.Equal(s.lastStyle) {
		s.writeStyle(buf, cell.Style)
		s.lastStyle = cell.Style
	}
	buf.WriteRune(cell.Rune)
}

// writeStyle writes ANSI escape codes for the given style.
func (s *Screen) writeStyle(buf *bytes.Buffer, style Style) {
	// Reset first if we need to turn off attributes
	buf.WriteString("\x1b[0")

	// Attributes
	if style.Attr.Has(AttrBold) {
		buf.WriteString(";1")
	}
	if style.Attr.Has(AttrDim) {
		buf.WriteString(";2")
	}
	if style.Attr.Has(AttrItalic) {
		buf.WriteString(";3")
	}
	if style.Attr.Has(AttrUnderline) {
		buf.WriteString(";4")
	}
	if style.Attr.Has(AttrBlink) {
		buf.WriteString(";5")
	}
	if style.Attr.Has(AttrInverse) {
		buf.WriteString(";7")
	}
	if style.Attr.Has(AttrStrikethrough) {
		buf.WriteString(";9")
	}

	// Foreground color
	s.writeColor(buf, style.FG, true)

	// Background color
	s.writeColor(buf, style.BG, false)

	buf.WriteString("m")
}

// writeColor writes the ANSI escape code for a color.
func (s *Screen) writeColor(buf *bytes.Buffer, c Color, fg bool) {
	switch c.Mode {
	case ColorDefault:
		// Use default color (39 for fg, 49 for bg)
		if fg {
			buf.WriteString(";39")
		} else {
			buf.WriteString(";49")
		}
	case Color16:
		// Basic 16 colors
		base := 30
		if !fg {
			base = 40
		}
		if c.Index >= 8 {
			// Bright colors
			base += 60
			buf.WriteString(fmt.Sprintf(";%d", base+int(c.Index-8)))
		} else {
			buf.WriteString(fmt.Sprintf(";%d", base+int(c.Index)))
		}
	case Color256:
		// 256 color palette
		if fg {
			buf.WriteString(fmt.Sprintf(";38;5;%d", c.Index))
		} else {
			buf.WriteString(fmt.Sprintf(";48;5;%d", c.Index))
		}
	case ColorRGB:
		// True color
		if fg {
			buf.WriteString(fmt.Sprintf(";38;2;%d;%d;%d", c.R, c.G, c.B))
		} else {
			buf.WriteString(fmt.Sprintf(";48;2;%d;%d;%d", c.R, c.G, c.B))
		}
	}
}

// writeString is a helper to write a string directly to the terminal.
func (s *Screen) writeString(str string) {
	io.WriteString(s.writer, str)
}

// Clear clears the back buffer.
func (s *Screen) Clear() {
	s.back.Clear()
}

// ShowCursor makes the cursor visible.
func (s *Screen) ShowCursor() {
	s.writeString("\x1b[?25h")
}

// HideCursor hides the cursor.
func (s *Screen) HideCursor() {
	s.writeString("\x1b[?25l")
}

// MoveCursor moves the cursor to the given position (0-indexed).
func (s *Screen) MoveCursor(x, y int) {
	s.writeString(fmt.Sprintf("\x1b[%d;%dH", y+1, x+1))
}
