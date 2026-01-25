package forme

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/mattn/go-runewidth"
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
	inlineMode  bool // Inline mode (no alternate buffer)

	// Resize handling
	resizeChan chan Size
	sigChan    chan os.Signal

	// Rendering state
	lastStyle Style        // Last style we emitted (for optimization)
	buf       bytes.Buffer // Reusable buffer for building output

	// Synchronization - protects buffer access during resize
	mu sync.Mutex
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

	// Enter alternate screen, hide cursor, enable bracketed paste
	s.writeString("\x1b[?1049h") // Enter alternate screen
	s.writeString("\x1b[2J")     // Clear screen (ensures front buffer matches actual screen)
	s.writeString("\x1b[H")      // Move cursor to home position
	s.writeString("\x1b[?25l")   // Hide cursor
	s.writeString("\x1b[?2004h") // Enable bracketed paste mode

	return nil
}

// ExitRawMode restores the terminal to its original state.
func (s *Screen) ExitRawMode() error {
	if !s.inRawMode {
		return nil
	}

	// Disable bracketed paste, show cursor, exit alternate screen
	s.writeString("\x1b[?2004l") // Disable bracketed paste mode
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

// EnterInlineMode puts the terminal into raw mode WITHOUT alternate buffer.
// Use this for inline UI elements (progress bars, menus, etc.) that render
// in the normal terminal flow rather than taking over the screen.
func (s *Screen) EnterInlineMode() error {
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
	s.inlineMode = true

	// Start listening for resize signals
	signal.Notify(s.sigChan, syscall.SIGWINCH)
	go s.handleSignals()

	// NO alternate screen switch for inline mode
	// Keep cursor visible

	return nil
}

// ExitInlineMode restores the terminal from inline mode.
// If clear is true, clears the lines used.
// If clear is false, moves cursor below the rendered content.
func (s *Screen) ExitInlineMode(linesUsed int, clear bool) error {
	if !s.inRawMode {
		return nil
	}

	// After FlushInline, cursor is at start of our content (row 0 of inline area)
	if clear && linesUsed > 0 {
		// Build all clear commands into a single write
		var clearBuf bytes.Buffer
		for i := 0; i < linesUsed; i++ {
			clearBuf.WriteString("\r\x1b[2K") // Start of line, clear entire line
			if i < linesUsed-1 {
				clearBuf.WriteString("\x1b[1B") // Move down to next line
			}
		}
		// Move back to first line
		if linesUsed > 1 {
			clearBuf.WriteString(fmt.Sprintf("\x1b[%dA", linesUsed-1))
		}
		clearBuf.WriteString("\r") // Ensure at start of line
		clearBuf.WriteString("\x1b[0m") // Reset style
		s.writer.Write(clearBuf.Bytes())
	} else if linesUsed > 0 {
		// Move cursor below content
		var moveBuf bytes.Buffer
		if linesUsed > 1 {
			moveBuf.WriteString(fmt.Sprintf("\x1b[%dB", linesUsed-1)) // Move to last line of content
		}
		moveBuf.WriteString("\r\n") // New line after content
		moveBuf.WriteString("\x1b[0m") // Reset style
		s.writer.Write(moveBuf.Bytes())
	} else {
		// Reset style
		s.writeString("\x1b[0m")
	}

	signal.Stop(s.sigChan)

	if s.origTermios != nil {
		if err := unix.IoctlSetTermios(s.fd, unix.TIOCSETA, s.origTermios); err != nil {
			return fmt.Errorf("failed to restore termios: %w", err)
		}
	}

	s.inRawMode = false
	s.inlineMode = false
	return nil
}

// IsInlineMode returns true if the screen is in inline mode.
func (s *Screen) IsInlineMode() bool {
	return s.inlineMode
}

// handleSignals processes OS signals.
func (s *Screen) handleSignals() {
	for range s.sigChan {
		width, height, err := getTerminalSize(s.fd)
		if err != nil {
			continue
		}
		if width != s.width || height != s.height {
			s.mu.Lock()
			s.width = width
			s.height = height
			s.front.Resize(width, height)
			s.back.Resize(width, height)
			// Clear BOTH buffers to avoid stale content
			s.front.Clear()
			s.back.Clear()
			// Clear the actual terminal screen
			s.writeString("\x1b[2J")
			s.mu.Unlock()
			// Non-blocking send (outside lock to avoid potential deadlock)
			select {
			case s.resizeChan <- Size{Width: width, Height: height}:
			default:
			}
		}
	}
}

// FlushStats holds statistics from the last flush.
type FlushStats struct {
	DirtyRows   int
	ChangedRows int
}

// lastFlushStats holds stats from the most recent flush.
var lastFlushStats FlushStats

// GetFlushStats returns stats from the last flush.
func GetFlushStats() FlushStats {
	return lastFlushStats
}

// debugFlush enables detailed flush debugging via TUI_DEBUG_FLUSH env var
var debugFlush = os.Getenv("TUI_DEBUG_FLUSH") != ""

// Flush renders the back buffer to the terminal using per-cell diff.
// Only cells that actually changed are written, with cursor positioning for each run.
// Uses dirty row tracking to skip rows that haven't been modified.
func (s *Screen) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buf.Reset()

	dirtyCount := 0
	changedCount := 0
	cursorX, cursorY := -1, -1
	positionCount := 0

	for y := 0; y < s.height; y++ {
		// Fast path: skip rows not marked dirty (no writes since last frame)
		if !s.back.RowDirty(y) {
			continue
		}
		dirtyCount++

		rowChanged := false
		for x := 0; x < s.width; x++ {
			backCell := s.back.Get(x, y)
			if backCell == s.front.Get(x, y) {
				continue
			}

			// skip placeholder cells (second half of double-width chars)
			if backCell.Rune == 0 {
				s.front.Set(x, y, backCell)
				continue
			}

			// Cell changed - need to write it
			if !rowChanged {
				rowChanged = true
				changedCount++
			}

			// Position cursor if not already there
			if cursorX != x || cursorY != y {
				if debugFlush && positionCount < 50 {
					rw := runewidth.RuneWidth(backCell.Rune)
					fmt.Fprintf(os.Stderr, "Flush: pos(%d,%d) cursor was (%d,%d) writing '%c' (U+%04X) width=%d\n",
						x, y, cursorX, cursorY, backCell.Rune, backCell.Rune, rw)
				}
				positionCount++
				s.buf.WriteString("\x1b[")
				s.writeIntToBuf(y + 1)
				s.buf.WriteByte(';')
				s.writeIntToBuf(x + 1)
				s.buf.WriteByte('H')
			}

			s.writeCell(&s.buf, backCell)
			s.front.Set(x, y, backCell)
			// cursor advances by the display width of the character
			rw := runewidth.RuneWidth(backCell.Rune)
			if rw == 0 {
				rw = 1 // zero-width chars still advance cursor by 1 in most terminals
			}
			cursorX = x + rw
			cursorY = y
		}
	}

	if debugFlush {
		fmt.Fprintf(os.Stderr, "Flush: %d dirty rows, %d changed rows, %d cursor positions, buf size %d\n",
			dirtyCount, changedCount, positionCount, s.buf.Len())
	}

	// Reset style at end if we have changes
	if changedCount > 0 {
		s.buf.WriteString("\x1b[0m")
		s.lastStyle = DefaultStyle()
	}
	// Note: Don't write here - let FlushBuffer() do it so we can batch cursor ops

	// Clear dirty flags for next frame
	s.back.ClearDirtyFlags()

	// Record stats
	lastFlushStats = FlushStats{DirtyRows: dirtyCount, ChangedRows: changedCount}
}

// writeIntToBuf writes an integer to the buffer without allocation.
func (s *Screen) writeIntToBuf(n int) {
	if n == 0 {
		s.buf.WriteByte('0')
		return
	}
	if n < 0 {
		s.buf.WriteByte('-')
		n = -n
	}

	// Use scratch space on stack (max 10 digits for int32)
	var scratch [10]byte
	i := len(scratch)
	for n > 0 {
		i--
		scratch[i] = byte('0' + n%10)
		n /= 10
	}
	s.buf.Write(scratch[i:])
}

// FlushFull does a complete redraw without diffing.
func (s *Screen) FlushFull() {
	s.mu.Lock()
	defer s.mu.Unlock()

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

// FlushInline renders the buffer for inline mode (no alternate screen).
// Renders at current cursor position using relative movement.
// Returns the number of lines rendered for cleanup tracking.
func (s *Screen) FlushInline(height int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buf.Reset()

	linesRendered := 0
	for y := 0; y < height && y < s.height; y++ {
		// Move to start of line, clear to end of line
		s.buf.WriteString("\r\x1b[K")

		for x := 0; x < s.width; x++ {
			cell := s.back.Get(x, y)
			if cell.Rune == 0 {
				break // Stop at first empty cell (end of content)
			}
			s.writeCell(&s.buf, cell)
			s.front.Set(x, y, cell)
		}
		linesRendered++

		if y < height-1 {
			s.buf.WriteString("\n") // Move down to next line
		}
	}

	// Reset style
	s.buf.WriteString("\x1b[0m")
	s.lastStyle = DefaultStyle()

	// Move cursor back to start of our content (first line)
	if linesRendered > 1 {
		s.buf.WriteString(fmt.Sprintf("\x1b[%dA", linesRendered-1))
	}
	s.buf.WriteString("\r")

	s.writer.Write(s.buf.Bytes())
	s.back.ClearDirtyFlags()

	return linesRendered
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

// writeColor writes the ANSI escape code for a color (allocation-free).
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
			buf.WriteByte(';')
			s.writeIntToBuf(base + int(c.Index-8))
		} else {
			buf.WriteByte(';')
			s.writeIntToBuf(base + int(c.Index))
		}
	case Color256:
		// 256 color palette
		if fg {
			buf.WriteString(";38;5;")
		} else {
			buf.WriteString(";48;5;")
		}
		s.writeIntToBuf(int(c.Index))
	case ColorRGB:
		// True color
		if fg {
			buf.WriteString(";38;2;")
		} else {
			buf.WriteString(";48;2;")
		}
		s.writeIntToBuf(int(c.R))
		buf.WriteByte(';')
		s.writeIntToBuf(int(c.G))
		buf.WriteByte(';')
		s.writeIntToBuf(int(c.B))
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
	// Build escape sequence without allocation: \x1b[row;colH
	var scratch [32]byte
	b := scratch[:0]
	b = append(b, "\x1b["...)
	b = appendInt(b, y+1)
	b = append(b, ';')
	b = appendInt(b, x+1)
	b = append(b, 'H')
	s.writer.Write(b)
}

// BufferCursor writes cursor positioning and visibility to the internal buffer.
// Call this before FlushBuffer() to batch cursor ops with content in one syscall.
func (s *Screen) BufferCursor(x, y int, visible bool, shape CursorShape) {
	// Cursor shape: \x1b[N q
	s.buf.WriteString("\x1b[")
	s.writeIntToBuf(int(shape))
	s.buf.WriteString(" q")

	// Cursor position: \x1b[row;colH
	s.buf.WriteString("\x1b[")
	s.writeIntToBuf(y + 1)
	s.buf.WriteByte(';')
	s.writeIntToBuf(x + 1)
	s.buf.WriteByte('H')

	// Cursor visibility
	if visible {
		s.buf.WriteString("\x1b[?25h")
	} else {
		s.buf.WriteString("\x1b[?25l")
	}
}

// BufferCursorColor sets cursor color using OSC 12 escape sequence.
// Format: OSC 12 ; #RRGGBB BEL
func (s *Screen) BufferCursorColor(c Color) {
	if c.Mode == ColorRGB {
		s.buf.WriteString("\x1b]12;#")
		s.buf.WriteByte(hexDigit(c.R >> 4))
		s.buf.WriteByte(hexDigit(c.R & 0xF))
		s.buf.WriteByte(hexDigit(c.G >> 4))
		s.buf.WriteByte(hexDigit(c.G & 0xF))
		s.buf.WriteByte(hexDigit(c.B >> 4))
		s.buf.WriteByte(hexDigit(c.B & 0xF))
		s.buf.WriteByte('\x07') // BEL terminator
	}
}

func hexDigit(n uint8) byte {
	if n < 10 {
		return '0' + n
	}
	return 'a' + n - 10
}

// FlushBuffer writes the accumulated buffer to the terminal in one syscall.
func (s *Screen) FlushBuffer() {
	if s.buf.Len() > 0 {
		s.writer.Write(s.buf.Bytes())
	}
}

// CursorShape represents the terminal cursor shape.
type CursorShape int

const (
	CursorDefault        CursorShape = 0 // Terminal default
	CursorBlockBlink     CursorShape = 1 // Blinking block
	CursorBlock          CursorShape = 2 // Steady block
	CursorUnderlineBlink CursorShape = 3 // Blinking underline
	CursorUnderline      CursorShape = 4 // Steady underline
	CursorBarBlink       CursorShape = 5 // Blinking bar (line)
	CursorBar            CursorShape = 6 // Steady bar (line)
)

// SetCursorShape changes the cursor shape.
func (s *Screen) SetCursorShape(shape CursorShape) {
	// Build escape sequence without allocation: \x1b[N q
	var scratch [16]byte
	b := scratch[:0]
	b = append(b, "\x1b["...)
	b = appendInt(b, int(shape))
	b = append(b, " q"...)
	s.writer.Write(b)
}

// appendInt appends an integer to a byte slice without allocation.
func appendInt(b []byte, n int) []byte {
	if n == 0 {
		return append(b, '0')
	}
	if n < 0 {
		b = append(b, '-')
		n = -n
	}
	// Find number of digits
	var scratch [10]byte
	i := len(scratch)
	for n > 0 {
		i--
		scratch[i] = byte('0' + n%10)
		n /= 10
	}
	return append(b, scratch[i:]...)
}
