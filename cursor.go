package tui

// Cursor represents a cursor position and style.
// Use this to read full cursor state. For setting, use the individual
// methods (SetCursor, SetCursorStyle, ShowCursor, HideCursor) which
// are optimized for their typical usage patterns.
type Cursor struct {
	X, Y    int
	Style   CursorShape
	Visible bool
}

// DefaultCursor returns a cursor with sensible defaults.
func DefaultCursor() Cursor {
	return Cursor{
		Style:   CursorBlock,
		Visible: true,
	}
}
