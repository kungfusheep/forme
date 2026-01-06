// minivim: A tiny vim-like editor demonstrating riffkey TextInput with tui framework
//
// Normal mode: j/k=move, i=insert, a=append, o=new line, dd=delete line, q=quit
// Insert mode: Type text, Esc=back to normal, all standard editing keys work
package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"riffkey"
	"tui"
)

// Layout constants
const (
	headerRows     = 0  // Content starts at row 0 now
	footerRows     = 2  // Status bar + message line
	renderBuffer   = 30 // Lines to render above/below viewport for smooth scrolling
)

// Buffer holds file content (can be shared across windows)
type Buffer struct {
	Lines     []string
	FileName  string
	undoStack []EditorState
	redoStack []EditorState
}

// Window is a view into a buffer
type Window struct {
	buffer *Buffer

	// Cursor position
	Cursor int
	Col    int

	// Viewport
	topLine        int
	viewportHeight int
	viewportWidth  int // for vertical splits

	// Visual mode selection (per-window)
	visualStart    int
	visualStartCol int
	visualLineMode bool

	// Rendering
	contentLayer      *tui.Layer
	lineNumWidth      int
	StatusBar         []tui.Span
	renderedMin       int
	renderedMax       int

	// Debug stats
	debugMode          bool
	lastRenderTime     time.Duration
	lastLinesRendered  int
	totalRenders       int
	totalLinesRendered int
}

// SplitDir indicates the split direction
type SplitDir int

const (
	SplitNone SplitDir = iota
	SplitHorizontal // windows stacked vertically (like :sp)
	SplitVertical   // windows side by side (like :vs)
)

// SplitNode is a binary tree node for window layout.
// Either Window is set (leaf) or Children are set (branch).
type SplitNode struct {
	// For branch nodes (splits)
	Direction SplitDir
	Children  [2]*SplitNode

	// For leaf nodes (windows)
	Window *Window

	// Parent pointer for navigation
	Parent *SplitNode
}

// IsLeaf returns true if this node contains a window
func (n *SplitNode) IsLeaf() bool {
	return n.Window != nil
}

// FindWindow returns the node containing the given window
func (n *SplitNode) FindWindow(w *Window) *SplitNode {
	if n.IsLeaf() {
		if n.Window == w {
			return n
		}
		return nil
	}
	if found := n.Children[0].FindWindow(w); found != nil {
		return found
	}
	return n.Children[1].FindWindow(w)
}

// AllWindows returns all windows in the tree (in-order)
func (n *SplitNode) AllWindows() []*Window {
	if n.IsLeaf() {
		return []*Window{n.Window}
	}
	result := n.Children[0].AllWindows()
	return append(result, n.Children[1].AllWindows()...)
}

// FirstWindow returns the first (top-left-most) window
func (n *SplitNode) FirstWindow() *Window {
	if n.IsLeaf() {
		return n.Window
	}
	return n.Children[0].FirstWindow()
}

// LastWindow returns the last (bottom-right-most) window
func (n *SplitNode) LastWindow() *Window {
	if n.IsLeaf() {
		return n.Window
	}
	return n.Children[1].LastWindow()
}

// Editor manages windows and global state
type Editor struct {
	root          *SplitNode // root of the split tree
	focusedWindow *Window    // currently focused window

	app *tui.App // reference for cursor control

	// Global state
	Mode       string // "NORMAL", "INSERT", or "VISUAL"
	StatusLine string // command/message line (bottom)

	// Search (global)
	searchPattern   string
	searchDirection int
	lastSearch      string

	// f/F/t/T (global)
	lastFindChar rune
	lastFindDir  int
	lastFindTill bool

	// Command line mode (global)
	cmdLineActive bool
	cmdLinePrompt string
	cmdLineInput  string
}

// Helper methods to access current window/buffer
func (ed *Editor) win() *Window { return ed.focusedWindow }
func (ed *Editor) buf() *Buffer { return ed.win().buffer }

// EditorState captures state for undo/redo
type EditorState struct {
	Lines  []string
	Cursor int
	Col    int
}


func main() {
	// Load own source file for demo
	fileName := "cmd/minivim/main.go"
	lines := loadFile(fileName)
	if lines == nil {
		lines = []string{"Could not load file", "Press 'q' to quit"}
		fileName = "[No Name]"
	}

	// Create initial buffer and window
	buf := &Buffer{
		Lines:    lines,
		FileName: fileName,
	}
	win := &Window{
		buffer:      buf,
		renderedMin: -1,
		renderedMax: -1,
	}

	// Create split tree with single window as root
	root := &SplitNode{Window: win}

	ed := &Editor{
		root:          root,
		focusedWindow: win,
		Mode:          "NORMAL",
		StatusLine:    "", // empty initially, used for messages
	}

	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}
	ed.app = app

	// Initialize viewport and layer
	size := app.Size()
	ed.win().viewportHeight = max(1, size.Height-headerRows-footerRows)
	ed.initLayer(size.Width)

	ed.updateDisplay()

	app.SetView(buildView(ed))

	// Start with block cursor in normal mode
	app.ShowCursor(tui.CursorBlock)
	ed.updateCursor()

	// Normal mode handlers
	app.Handle("j", func(m riffkey.Match) { ed.moveDown(m.Count) })
	app.Handle("k", func(m riffkey.Match) { ed.moveUp(m.Count) })
	app.Handle("h", func(m riffkey.Match) { ed.moveLeft(m.Count) })
	app.Handle("l", func(m riffkey.Match) { ed.moveRight(m.Count) })
	app.Handle("gg", func(_ riffkey.Match) { ed.moveTo(0, ed.win().Col) })
	app.Handle("G", func(_ riffkey.Match) { ed.moveTo(len(ed.buf().Lines)-1, ed.win().Col) })
	app.Handle("0", func(_ riffkey.Match) { ed.moveToCol(0) })
	app.Handle("$", func(_ riffkey.Match) { ed.moveToCol(len(ed.buf().Lines[ed.win().Cursor])) })

	app.Handle("w", func(m riffkey.Match) {
		oldLine := ed.win().Cursor
		for range m.Count {
			ed.wordForward()
		}
		ed.ensureCursorVisible()
		ed.updateCursorHighlight(oldLine)
		ed.updateCursor()
	})

	app.Handle("b", func(m riffkey.Match) {
		oldLine := ed.win().Cursor
		for range m.Count {
			ed.wordBackward()
		}
		ed.ensureCursorVisible()
		ed.updateCursorHighlight(oldLine)
		ed.updateCursor()
	})

	app.Handle("e", func(m riffkey.Match) {
		oldLine := ed.win().Cursor
		for range m.Count {
			ed.wordEnd()
		}
		ed.ensureCursorVisible()
		ed.updateCursorHighlight(oldLine)
		ed.updateCursor()
	})

	app.Handle("i", func(_ riffkey.Match) {
		ed.enterInsertMode(app)
	})

	app.Handle("a", func(_ riffkey.Match) {
		// Append after cursor
		if len(ed.buf().Lines[ed.win().Cursor]) > 0 {
			ed.win().Col++
		}
		ed.enterInsertMode(app)
	})

	app.Handle("A", func(_ riffkey.Match) {
		ed.win().Col = len(ed.buf().Lines[ed.win().Cursor])
		ed.enterInsertMode(app)
	})

	app.Handle("I", func(_ riffkey.Match) {
		ed.win().Col = 0
		ed.enterInsertMode(app)
	})

	app.Handle("o", func(_ riffkey.Match) {
		// Insert new line below
		ed.win().Cursor++
		newLines := make([]string, len(ed.buf().Lines)+1)
		copy(newLines[:ed.win().Cursor], ed.buf().Lines[:ed.win().Cursor])
		newLines[ed.win().Cursor] = ""
		copy(newLines[ed.win().Cursor+1:], ed.buf().Lines[ed.win().Cursor:])
		ed.buf().Lines = newLines
		ed.win().Col = 0
		ed.updateDisplay()
		ed.enterInsertMode(app)
	})

	app.Handle("O", func(_ riffkey.Match) {
		// Insert new line above
		newLines := make([]string, len(ed.buf().Lines)+1)
		copy(newLines[:ed.win().Cursor], ed.buf().Lines[:ed.win().Cursor])
		newLines[ed.win().Cursor] = ""
		copy(newLines[ed.win().Cursor+1:], ed.buf().Lines[ed.win().Cursor:])
		ed.buf().Lines = newLines
		ed.win().Col = 0
		ed.updateDisplay()
		ed.enterInsertMode(app)
	})

	app.Handle("dd", func(m riffkey.Match) {
		ed.saveUndo()
		for i := 0; i < m.Count; i++ {
			if len(ed.buf().Lines) > 1 {
				ed.buf().Lines = append(ed.buf().Lines[:ed.win().Cursor], ed.buf().Lines[ed.win().Cursor+1:]...)
				if ed.win().Cursor >= len(ed.buf().Lines) {
					ed.win().Cursor = len(ed.buf().Lines) - 1
				}
			} else {
				ed.buf().Lines[0] = ""
				break
			}
		}
		ed.win().Col = min(ed.win().Col, max(0, len(ed.buf().Lines[ed.win().Cursor])-1))
		ed.updateDisplay()
		ed.updateCursor()
	})

	app.Handle("x", func(m riffkey.Match) {
		ed.saveUndo()
		// Delete character(s) under cursor
		for i := 0; i < m.Count; i++ {
			line := ed.buf().Lines[ed.win().Cursor]
			if len(line) > 0 && ed.win().Col < len(line) {
				ed.buf().Lines[ed.win().Cursor] = line[:ed.win().Col] + line[ed.win().Col+1:]
			}
		}
		if ed.win().Col >= len(ed.buf().Lines[ed.win().Cursor]) && ed.win().Col > 0 {
			ed.win().Col = max(0, len(ed.buf().Lines[ed.win().Cursor])-1)
		}
		ed.updateDisplay()
		ed.updateCursor()
	})

	app.Handle("q", func(_ riffkey.Match) {
		app.Stop()
	})

	app.Handle("<Esc>", func(_ riffkey.Match) {
		// Already in normal mode, do nothing
	})

	// Register operator + text object combinations (diw, ciw, yaw, etc.)
	registerOperatorTextObjects(app, ed)


	// Paste from yank register
	app.Handle("p", func(_ riffkey.Match) {
		if yankRegister != "" {
			line := ed.buf().Lines[ed.win().Cursor]
			pos := min(ed.win().Col+1, len(line))
			ed.buf().Lines[ed.win().Cursor] = line[:pos] + yankRegister + line[pos:]
			ed.win().Col = pos + len(yankRegister) - 1
			ed.updateDisplay()
			ed.updateCursor()
		}
	})

	app.Handle("P", func(_ riffkey.Match) {
		if yankRegister != "" {
			line := ed.buf().Lines[ed.win().Cursor]
			ed.buf().Lines[ed.win().Cursor] = line[:ed.win().Col] + yankRegister + line[ed.win().Col:]
			ed.updateDisplay()
			ed.updateCursor()
		}
	})

	// Undo/Redo
	app.Handle("u", func(_ riffkey.Match) {
		ed.undo()
	})

	app.Handle("<C-r>", func(_ riffkey.Match) {
		ed.redo()
	})

	// Scrolling
	app.Handle("<C-d>", func(_ riffkey.Match) {
		// Half page down
		ed.ensureCursorVisible()
		half := ed.win().viewportHeight / 2
		ed.win().Cursor = min(ed.win().Cursor+half, len(ed.buf().Lines)-1)
		ed.win().Col = min(ed.win().Col, len(ed.buf().Lines[ed.win().Cursor]))
		ed.updateDisplay()
		ed.updateCursor()
	})

	app.Handle("<C-u>", func(_ riffkey.Match) {
		// Half page up
		ed.ensureCursorVisible()
		half := ed.win().viewportHeight / 2
		ed.win().Cursor = max(ed.win().Cursor-half, 0)
		ed.win().Col = min(ed.win().Col, len(ed.buf().Lines[ed.win().Cursor]))
		ed.updateDisplay()
		ed.updateCursor()
	})

	app.Handle("<C-e>", func(_ riffkey.Match) {
		// Scroll down one line (keep cursor in place if possible)
		ed.ensureCursorVisible()
		if ed.win().topLine < len(ed.buf().Lines)-ed.win().viewportHeight {
			ed.win().topLine++
			if ed.win().Cursor < ed.win().topLine {
				ed.win().Cursor = ed.win().topLine
				ed.win().Col = min(ed.win().Col, len(ed.buf().Lines[ed.win().Cursor]))
			}
			ed.updateDisplay()
			ed.updateCursor()
		}
	})

	app.Handle("<C-y>", func(_ riffkey.Match) {
		// Scroll up one line (keep cursor in place if possible)
		if ed.win().topLine > 0 {
			ed.win().topLine--
			if ed.win().Cursor >= ed.win().topLine+ed.win().viewportHeight {
				ed.win().Cursor = ed.win().topLine + ed.win().viewportHeight - 1
				ed.win().Col = min(ed.win().Col, len(ed.buf().Lines[ed.win().Cursor]))
			}
			ed.updateDisplay()
			ed.updateCursor()
		}
	})

	// f/F/t/T - find character on line
	registerFindChar(app, ed)

	// Visual mode
	app.Handle("v", func(_ riffkey.Match) {
		ed.enterVisualMode(app, false)
	})

	app.Handle("V", func(_ riffkey.Match) {
		ed.enterVisualMode(app, true)
	})

	// Join lines (J)
	app.Handle("J", func(_ riffkey.Match) {
		ed.saveUndo()
		if ed.win().Cursor < len(ed.buf().Lines)-1 {
			// Join current line with next
			ed.buf().Lines[ed.win().Cursor] += " " + ed.buf().Lines[ed.win().Cursor+1]
			ed.buf().Lines = append(ed.buf().Lines[:ed.win().Cursor+1], ed.buf().Lines[ed.win().Cursor+2:]...)
			ed.updateDisplay()
		}
	})

	// Replace single char (r)
	app.Handle("r", func(_ riffkey.Match) {
		// Next key will replace char - push a one-shot router
		replaceRouter := riffkey.NewRouter().Name("replace")
		replaceRouter.HandleUnmatched(func(k riffkey.Key) bool {
			if k.Rune != 0 && k.Mod == riffkey.ModNone {
				ed.saveUndo()
				line := ed.buf().Lines[ed.win().Cursor]
				if ed.win().Col < len(line) {
					ed.buf().Lines[ed.win().Cursor] = line[:ed.win().Col] + string(k.Rune) + line[ed.win().Col+1:]
					ed.updateDisplay()
				}
			}
			app.Pop()
			return true
		})
		replaceRouter.Handle("<Esc>", func(_ riffkey.Match) {
			app.Pop()
		})
		app.Push(replaceRouter)
	})

	// Repeat last change (.) - simplified: just re-insert last deleted text
	app.Handle(".", func(_ riffkey.Match) {
		if yankRegister != "" {
			ed.saveUndo()
			line := ed.buf().Lines[ed.win().Cursor]
			ed.buf().Lines[ed.win().Cursor] = line[:ed.win().Col] + yankRegister + line[ed.win().Col:]
			ed.win().Col += len(yankRegister)
			ed.updateDisplay()
			ed.updateCursor()
		}
	})

	// ~ toggle case
	app.Handle("~", func(_ riffkey.Match) {
		ed.saveUndo()
		line := ed.buf().Lines[ed.win().Cursor]
		if ed.win().Col < len(line) {
			c := line[ed.win().Col]
			if c >= 'a' && c <= 'z' {
				c = c - 'a' + 'A'
			} else if c >= 'A' && c <= 'Z' {
				c = c - 'A' + 'a'
			}
			ed.buf().Lines[ed.win().Cursor] = line[:ed.win().Col] + string(c) + line[ed.win().Col+1:]
			if ed.win().Col < len(line)-1 {
				ed.win().Col++
			}
			ed.updateDisplay()
			ed.updateCursor()
		}
	})

	// Command line mode handlers
	app.Handle(":", func(_ riffkey.Match) {
		ed.enterCommandMode(app, ":")
	})

	app.Handle("/", func(_ riffkey.Match) {
		ed.enterCommandMode(app, "/")
	})

	app.Handle("?", func(_ riffkey.Match) {
		ed.enterCommandMode(app, "?")
	})

	// n/N for search repeat
	app.Handle("n", func(_ riffkey.Match) {
		ed.searchNext(1)
	})

	app.Handle("N", func(_ riffkey.Match) {
		ed.searchNext(-1)
	})

	// Debug mode toggle - shows render stats in status bar
	app.Handle("<C-g>", func(_ riffkey.Match) {
		ed.win().debugMode = !ed.win().debugMode
		if ed.win().debugMode {
			ed.StatusLine = "Debug mode ON - showing render stats"
		} else {
			ed.StatusLine = "Debug mode OFF"
		}
		ed.updateDisplay()
	})

	// Window management: Ctrl-w commands
	app.Handle("<C-w>w", func(_ riffkey.Match) {
		ed.focusNextWindow()
	})
	app.Handle("<C-w>W", func(_ riffkey.Match) {
		ed.focusPrevWindow()
	})
	app.Handle("<C-w>j", func(_ riffkey.Match) {
		// Move to window below - find window with horizontal split parent
		ed.focusDirection(SplitHorizontal, 1)
	})
	app.Handle("<C-w>k", func(_ riffkey.Match) {
		// Move to window above - find window with horizontal split parent
		ed.focusDirection(SplitHorizontal, -1)
	})
	app.Handle("<C-w>h", func(_ riffkey.Match) {
		// Move to window left - find window with vertical split parent
		ed.focusDirection(SplitVertical, -1)
	})
	app.Handle("<C-w>l", func(_ riffkey.Match) {
		// Move to window right - find window with vertical split parent
		ed.focusDirection(SplitVertical, 1)
	})
	app.Handle("<C-w>s", func(_ riffkey.Match) {
		ed.splitHorizontal()
	})
	app.Handle("<C-w>v", func(_ riffkey.Match) {
		ed.splitVertical()
	})
	app.Handle("<C-w>c", func(_ riffkey.Match) {
		ed.closeWindow()
	})
	app.Handle("<C-w>o", func(_ riffkey.Match) {
		ed.closeOtherWindows()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func (ed *Editor) enterInsertMode(app *tui.App) {
	ed.Mode = "INSERT"
	ed.win().Col = min(ed.win().Col, len(ed.buf().Lines[ed.win().Cursor]))
	ed.StatusLine = "-- INSERT --  Esc:normal  Enter:newline  Ctrl+W:delete word"
	ed.updateDisplay()

	// Switch to bar cursor for insert mode
	app.ShowCursor(tui.CursorBar)
	ed.updateCursor()

	// Create insert mode router
	insertRouter := riffkey.NewRouter().Name("insert")

	// TextHandler with OnChange callback for live updates
	th := riffkey.NewTextHandler(&ed.buf().Lines[ed.win().Cursor], &ed.win().Col)
	th.OnChange = func(_ string) {
		ed.updateDisplay()
		ed.updateCursor()
	}

	// Esc exits insert mode
	insertRouter.Handle("<Esc>", func(_ riffkey.Match) {
		ed.exitInsertMode(app)
	})

	// Enter creates a new line
	insertRouter.Handle("<CR>", func(_ riffkey.Match) {
		line := ed.buf().Lines[ed.win().Cursor]
		before := line[:ed.win().Col]
		after := line[ed.win().Col:]
		ed.buf().Lines[ed.win().Cursor] = before

		// Insert new line after
		newLines := make([]string, len(ed.buf().Lines)+1)
		copy(newLines[:ed.win().Cursor+1], ed.buf().Lines[:ed.win().Cursor+1])
		newLines[ed.win().Cursor+1] = after
		copy(newLines[ed.win().Cursor+2:], ed.buf().Lines[ed.win().Cursor+1:])
		ed.buf().Lines = newLines
		ed.win().Cursor++
		ed.win().Col = 0

		// Rebind TextHandler to new line
		th.Value = &ed.buf().Lines[ed.win().Cursor]
		ed.updateDisplay()
		ed.updateCursor()
	})

	// Wire up the text handler for unmatched keys
	insertRouter.HandleUnmatched(th.HandleKey)

	// Push the insert router - takes over input
	app.Push(insertRouter)
}

func (ed *Editor) exitInsertMode(app *tui.App) {
	ed.Mode = "NORMAL"
	ed.StatusLine = "hjkl:move  w/b/e:word  ciw/daw/yi\":text-obj  p:paste  q:quit"

	// Adjust cursor if at end of line (vim behavior)
	if ed.win().Col > 0 && ed.win().Col >= len(ed.buf().Lines[ed.win().Cursor]) {
		ed.win().Col = max(0, len(ed.buf().Lines[ed.win().Cursor])-1)
	}

	// Switch back to block cursor for normal mode
	app.ShowCursor(tui.CursorBlock)

	ed.updateDisplay()
	ed.updateCursor()
	app.Pop() // Back to normal mode router
}

func (ed *Editor) updateCursor() {
	// Calculate screen position relative to viewport
	screenY := headerRows + (ed.win().Cursor - ed.win().topLine)
	screenX := ed.win().lineNumWidth + ed.win().Col

	// Adjust for split windows by traversing tree to find offset
	offsetX, offsetY := ed.getWindowOffset(ed.focusedWindow)
	screenX += offsetX
	screenY += offsetY

	ed.app.SetCursor(screenX, screenY)
}

// getWindowOffset calculates the screen offset for a window by traversing the tree
func (ed *Editor) getWindowOffset(w *Window) (x, y int) {
	node := ed.root.FindWindow(w)
	if node == nil {
		return 0, 0
	}

	// Walk up the tree, accumulating offsets
	for node.Parent != nil {
		parent := node.Parent
		// If we're the second child, add the first child's dimensions
		if parent.Children[1] == node {
			first := parent.Children[0]
			switch parent.Direction {
			case SplitHorizontal:
				// First child is above us, add its height
				y += ed.getNodeHeight(first)
			case SplitVertical:
				// First child is to our left, add its width
				x += ed.getNodeWidth(first)
			}
		}
		node = parent
	}
	return x, y
}

// getNodeHeight returns the total height of a node (sum of all windows + status bars)
func (ed *Editor) getNodeHeight(n *SplitNode) int {
	if n.IsLeaf() {
		return n.Window.viewportHeight + 1 // +1 for status bar
	}
	if n.Direction == SplitHorizontal {
		// Stacked vertically - sum heights
		return ed.getNodeHeight(n.Children[0]) + ed.getNodeHeight(n.Children[1])
	}
	// Side by side - max height
	h0 := ed.getNodeHeight(n.Children[0])
	h1 := ed.getNodeHeight(n.Children[1])
	if h0 > h1 {
		return h0
	}
	return h1
}

// getNodeWidth returns the total width of a node
func (ed *Editor) getNodeWidth(n *SplitNode) int {
	if n.IsLeaf() {
		return n.Window.viewportWidth
	}
	if n.Direction == SplitVertical {
		// Side by side - sum widths
		return ed.getNodeWidth(n.Children[0]) + ed.getNodeWidth(n.Children[1])
	}
	// Stacked - max width
	w0 := ed.getNodeWidth(n.Children[0])
	w1 := ed.getNodeWidth(n.Children[1])
	if w0 > w1 {
		return w0
	}
	return w1
}

// refresh does a full re-render - use for content changes or visual mode.
// For simple cursor movement, use updateCursorHighlight() instead.
func (ed *Editor) refresh() {
	ed.updateDisplay()
	ed.updateCursor()
}

// moveTo sets cursor to absolute position, clamping to valid range
func (ed *Editor) moveTo(line, col int) {
	oldLine := ed.win().Cursor
	ed.win().Cursor = max(0, min(line, len(ed.buf().Lines)-1))
	ed.win().Col = max(0, min(col, len(ed.buf().Lines[ed.win().Cursor])-1))
	ed.ensureCursorVisible()
	ed.updateCursorHighlight(oldLine)
	ed.updateCursor()
}

// moveToCol sets column only, clamping to valid range
func (ed *Editor) moveToCol(col int) {
	ed.win().Col = max(0, min(col, len(ed.buf().Lines[ed.win().Cursor])-1))
	ed.updateCursor()
}

// moveDown moves cursor down by count lines
func (ed *Editor) moveDown(count int) {
	oldLine := ed.win().Cursor
	ed.win().Cursor = min(ed.win().Cursor+count, len(ed.buf().Lines)-1)
	ed.win().Col = min(ed.win().Col, len(ed.buf().Lines[ed.win().Cursor]))
	ed.ensureCursorVisible()
	ed.updateCursorHighlight(oldLine)
	ed.updateCursor()
}

// moveUp moves cursor up by count lines
func (ed *Editor) moveUp(count int) {
	oldLine := ed.win().Cursor
	ed.win().Cursor = max(ed.win().Cursor-count, 0)
	ed.win().Col = min(ed.win().Col, len(ed.buf().Lines[ed.win().Cursor]))
	ed.ensureCursorVisible()
	ed.updateCursorHighlight(oldLine)
	ed.updateCursor()
}

// moveLeft moves cursor left by count columns
func (ed *Editor) moveLeft(count int) {
	ed.win().Col = max(0, ed.win().Col-count)
	ed.updateCursor()
}

// moveRight moves cursor right by count columns
func (ed *Editor) moveRight(count int) {
	ed.win().Col = min(ed.win().Col+count, len(ed.buf().Lines[ed.win().Cursor])-1)
	ed.updateCursor()
}

func (ed *Editor) ensureCursorVisible() {
	// Scroll viewport if cursor is outside visible area
	if ed.win().viewportHeight == 0 {
		// Get viewport height from screen (minus footer for status bar + message line)
		size := ed.app.Size()
		ed.win().viewportHeight = max(1, size.Height-headerRows-footerRows)
	}

	// Scroll up if cursor above viewport
	if ed.win().Cursor < ed.win().topLine {
		ed.win().topLine = ed.win().Cursor
	}

	// Scroll down if cursor below viewport
	if ed.win().Cursor >= ed.win().topLine+ed.win().viewportHeight {
		ed.win().topLine = ed.win().Cursor - ed.win().viewportHeight + 1
	}
}

// Style constants for vim-like appearance
var (
	lineNumStyle       = tui.Style{Attr: tui.AttrDim}
	cursorLineNumStyle = tui.Style{FG: tui.Color{Mode: tui.Color16, Index: 3}} // Yellow for current line number
	tildeStyle         = tui.Style{FG: tui.Color{Mode: tui.Color16, Index: 4}} // Blue for ~ lines
	statusBarStyle     = tui.Style{Attr: tui.AttrInverse}                      // Inverse video like vim
	searchHighlight    = tui.Style{BG: tui.Color{Mode: tui.Color16, Index: 3}} // Yellow background for search matches
)

// highlightSearchMatches splits a line into spans with search matches highlighted
func (ed *Editor) highlightSearchMatches(line string) []tui.Span {
	if ed.searchPattern == "" || len(line) == 0 {
		return []tui.Span{{Text: line}}
	}

	var spans []tui.Span
	remaining := line

	for {
		idx := strings.Index(remaining, ed.searchPattern)
		if idx < 0 {
			// No more matches
			if len(remaining) > 0 {
				spans = append(spans, tui.Span{Text: remaining})
			}
			break
		}

		// Add text before match
		if idx > 0 {
			spans = append(spans, tui.Span{Text: remaining[:idx]})
		}

		// Add highlighted match
		spans = append(spans, tui.Span{Text: ed.searchPattern, Style: searchHighlight})

		// Move past match
		remaining = remaining[idx+len(ed.searchPattern):]
	}

	if len(spans) == 0 {
		return []tui.Span{{Text: line}}
	}
	return spans
}

// updateStatusBar builds the vim-style status bar
func (ed *Editor) updateStatusBar() {
	// Use stored viewport width if set, otherwise full screen width
	width := ed.win().viewportWidth
	if width == 0 {
		width = ed.app.Size().Width
	}

	// Left side: filename (and debug stats if enabled)
	left := " " + ed.buf().FileName
	if ed.win().debugMode {
		avgLines := 0
		if ed.win().totalRenders > 0 {
			avgLines = ed.win().totalLinesRendered / ed.win().totalRenders
		}
		left = fmt.Sprintf(" [%v last:%d avg:%d rng:%d-%d] %s",
			ed.win().lastRenderTime.Round(time.Microsecond),
			ed.win().lastLinesRendered,
			avgLines,
			ed.win().renderedMin, ed.win().renderedMax,
			ed.buf().FileName)
	}

	// Right side: line:col percentage
	percentage := 0
	if len(ed.buf().Lines) > 0 {
		percentage = (ed.win().Cursor + 1) * 100 / len(ed.buf().Lines)
	}
	right := fmt.Sprintf(" %d,%d  %d%% ", ed.win().Cursor+1, ed.win().Col+1, percentage)

	// Calculate padding to fill width
	padding := width - len(left) - len(right)
	if padding < 1 {
		padding = 1
	}
	middle := ""
	for i := 0; i < padding; i++ {
		middle += " "
	}

	// Build single span with inverse style
	ed.win().StatusBar = []tui.Span{
		{Text: left + middle + right, Style: statusBarStyle},
	}
}

// updateWindowStatusBar builds status bar for a specific window
func (ed *Editor) updateWindowStatusBar(w *Window, focused bool) {
	// Use stored viewport width if set, otherwise full screen width
	width := w.viewportWidth
	if width == 0 {
		width = ed.app.Size().Width
	}

	// Left side: filename (and debug stats if enabled)
	left := " " + w.buffer.FileName
	if w.debugMode {
		avgLines := 0
		if w.totalRenders > 0 {
			avgLines = w.totalLinesRendered / w.totalRenders
		}
		left = fmt.Sprintf(" [%v last:%d avg:%d rng:%d-%d] %s",
			w.lastRenderTime.Round(time.Microsecond),
			w.lastLinesRendered,
			avgLines,
			w.renderedMin, w.renderedMax,
			w.buffer.FileName)
	}

	// Right side: line:col percentage
	percentage := 0
	if len(w.buffer.Lines) > 0 {
		percentage = (w.Cursor + 1) * 100 / len(w.buffer.Lines)
	}
	right := fmt.Sprintf(" %d,%d  %d%% ", w.Cursor+1, w.Col+1, percentage)

	// Calculate padding to fill width
	padding := width - len(left) - len(right)
	if padding < 1 {
		padding = 1
	}
	middle := strings.Repeat(" ", padding)

	// Use different style for unfocused windows
	style := statusBarStyle
	if !focused {
		style = tui.Style{Attr: tui.AttrDim | tui.AttrInverse} // Dimmer for unfocused
	}

	w.StatusBar = []tui.Span{
		{Text: left + middle + right, Style: style},
	}
}

// ensureWindowRendered makes sure visible region + buffer is rendered for a specific window
func (ed *Editor) ensureWindowRendered(w *Window) {
	if w.contentLayer == nil || w.contentLayer.Buffer() == nil {
		return
	}

	start := time.Now()
	linesRendered := 0

	// Calculate line number width based on total lines
	maxLineNum := len(w.buffer.Lines)
	w.lineNumWidth = len(fmt.Sprintf("%d", maxLineNum)) + 1

	// Calculate desired render range (visible + buffer)
	wantMin := max(0, w.topLine-renderBuffer)
	wantMax := min(len(w.buffer.Lines)+w.viewportHeight-1, w.topLine+w.viewportHeight+renderBuffer)

	// First time? Render the whole range
	if w.renderedMin < 0 {
		for i := wantMin; i <= wantMax; i++ {
			ed.renderWindowLineToLayer(w, i)
			linesRendered++
		}
		w.renderedMin = wantMin
		w.renderedMax = wantMax
	} else {
		// Expand rendered range if needed
		if wantMin < w.renderedMin {
			for i := wantMin; i < w.renderedMin; i++ {
				ed.renderWindowLineToLayer(w, i)
				linesRendered++
			}
			w.renderedMin = wantMin
		}
		if wantMax > w.renderedMax {
			for i := w.renderedMax + 1; i <= wantMax; i++ {
				ed.renderWindowLineToLayer(w, i)
				linesRendered++
			}
			w.renderedMax = wantMax
		}
	}

	// Set scroll position
	w.contentLayer.ScrollTo(w.topLine)

	// Track stats
	w.lastRenderTime = time.Since(start)
	w.lastLinesRendered = linesRendered
	w.totalRenders++
	w.totalLinesRendered += linesRendered
}

// renderWindowLineToLayer renders a single line for a specific window
func (ed *Editor) renderWindowLineToLayer(w *Window, lineIdx int) {
	if w.contentLayer == nil || w.contentLayer.Buffer() == nil {
		return
	}

	lineNumFmt := fmt.Sprintf("%%%dd ", w.lineNumWidth-1)
	tildeFmt := fmt.Sprintf("%%%ds ", w.lineNumWidth-1)

	var spans []tui.Span

	if lineIdx < len(w.buffer.Lines) {
		// Content line
		line := w.buffer.Lines[lineIdx]
		lineNum := fmt.Sprintf(lineNumFmt, lineIdx+1)
		isCursorLine := lineIdx == w.Cursor

		numStyle := lineNumStyle
		if isCursorLine {
			numStyle = cursorLineNumStyle
		}

		// For the focused window in visual mode, use visual spans
		if ed.Mode == "VISUAL" && ed.win() == w {
			spans = append([]tui.Span{{Text: lineNum, Style: numStyle}}, ed.getVisualSpans(lineIdx, line)...)
		} else {
			// Use search highlighting
			contentSpans := ed.highlightSearchMatches(line)
			spans = append([]tui.Span{{Text: lineNum, Style: numStyle}}, contentSpans...)
		}
	} else {
		// Tilde line (beyond EOF)
		spans = []tui.Span{{Text: fmt.Sprintf(tildeFmt, "~"), Style: tildeStyle}}
	}

	w.contentLayer.SetLine(lineIdx, spans)
}

func (ed *Editor) updateDisplay() {
	// Ensure viewport height is set and cursor is visible
	ed.ensureCursorVisible()

	// Build vim-style status bar
	ed.updateStatusBar()

	// Re-render visible region (invalidate first for content changes)
	ed.invalidateRenderedRange()
	ed.ensureRendered()

	// Sync other windows viewing the same buffer
	ed.syncOtherWindows()
}

// syncOtherWindows re-renders other windows that share the same buffer
func (ed *Editor) syncOtherWindows() {
	allWindows := ed.root.AllWindows()
	if len(allWindows) <= 1 {
		return
	}
	currentBuf := ed.buf()
	for _, w := range allWindows {
		if w != ed.focusedWindow && w.buffer == currentBuf {
			// Invalidate and re-render this window
			w.renderedMin = -1
			w.renderedMax = -1
			ed.ensureWindowRendered(w)
			ed.updateWindowStatusBar(w, false)
		}
	}
}

// initLayer creates and sizes the content layer
func (ed *Editor) initLayer(width int) {
	ed.initWindowLayer(ed.win(), width)
}

// initWindowLayer sets up a layer for a specific window
func (ed *Editor) initWindowLayer(w *Window, width int) {
	w.viewportWidth = width
	w.contentLayer = tui.NewLayer()
	// Layer holds ALL lines plus some buffer for scrolling
	w.contentLayer.EnsureSize(width, len(w.buffer.Lines)+w.viewportHeight)
	w.renderedMin = -1
	w.renderedMax = -1
	ed.ensureWindowRendered(w)
}

// splitHorizontal creates a horizontal split (windows stacked vertically like :sp)
func (ed *Editor) splitHorizontal() {
	// Find the current window's node
	currentNode := ed.root.FindWindow(ed.focusedWindow)
	if currentNode == nil {
		return
	}

	// Calculate available height for the current node's area
	// For simplicity, split the current window's viewport in half
	totalHeight := ed.focusedWindow.viewportHeight
	halfHeight := max(1, totalHeight/2)

	// Update existing window height
	ed.focusedWindow.viewportHeight = halfHeight

	// Create new window viewing the same buffer
	newWin := &Window{
		buffer:         ed.buf(),
		Cursor:         ed.focusedWindow.Cursor,
		Col:            ed.focusedWindow.Col,
		topLine:        ed.focusedWindow.topLine,
		viewportHeight: totalHeight - halfHeight,
		viewportWidth:  ed.focusedWindow.viewportWidth,
		renderedMin:    -1,
		renderedMax:    -1,
	}

	// Initialize layer for new window
	ed.initWindowLayer(newWin, newWin.viewportWidth)

	// Create new nodes
	newWindowNode := &SplitNode{Window: newWin}
	currentWindowNode := &SplitNode{Window: ed.focusedWindow}

	// Replace current leaf with a split branch
	currentNode.Direction = SplitHorizontal
	currentNode.Window = nil
	currentNode.Children = [2]*SplitNode{currentWindowNode, newWindowNode}
	currentWindowNode.Parent = currentNode
	newWindowNode.Parent = currentNode

	// Refresh display
	ed.app.SetView(buildView(ed))
	ed.updateAllWindows()
	ed.StatusLine = ""
}

// splitVertical creates a vertical split (windows side by side like :vs)
func (ed *Editor) splitVertical() {
	// Find the current window's node
	currentNode := ed.root.FindWindow(ed.focusedWindow)
	if currentNode == nil {
		return
	}

	// Split the current window's width in half
	totalWidth := ed.focusedWindow.viewportWidth
	halfWidth := max(1, totalWidth/2-1) // -1 for separator space

	// Update existing window width
	ed.focusedWindow.viewportWidth = halfWidth

	// Create new window viewing the same buffer
	newWin := &Window{
		buffer:         ed.buf(),
		Cursor:         ed.focusedWindow.Cursor,
		Col:            ed.focusedWindow.Col,
		topLine:        ed.focusedWindow.topLine,
		viewportHeight: ed.focusedWindow.viewportHeight,
		viewportWidth:  totalWidth - halfWidth - 1, // -1 for separator
		renderedMin:    -1,
		renderedMax:    -1,
	}

	// Reinitialize layers for both windows with new widths
	ed.initWindowLayer(ed.focusedWindow, halfWidth)
	ed.initWindowLayer(newWin, newWin.viewportWidth)

	// Create new nodes
	newWindowNode := &SplitNode{Window: newWin}
	currentWindowNode := &SplitNode{Window: ed.focusedWindow}

	// Replace current leaf with a split branch
	currentNode.Direction = SplitVertical
	currentNode.Window = nil
	currentNode.Children = [2]*SplitNode{currentWindowNode, newWindowNode}
	currentWindowNode.Parent = currentNode
	newWindowNode.Parent = currentNode

	// Refresh display
	ed.app.SetView(buildView(ed))
	ed.updateAllWindows()
	ed.StatusLine = ""
}

// closeWindow closes the current window
func (ed *Editor) closeWindow() {
	// Can't close if this is the only window
	if ed.root.IsLeaf() {
		ed.StatusLine = "E444: Cannot close last window"
		ed.updateDisplay()
		return
	}

	// Find the current window's node and its parent
	node := ed.root.FindWindow(ed.focusedWindow)
	if node == nil || node.Parent == nil {
		return
	}

	parent := node.Parent

	// Find the sibling
	var sibling *SplitNode
	if parent.Children[0] == node {
		sibling = parent.Children[1]
	} else {
		sibling = parent.Children[0]
	}

	// Promote sibling to parent's position
	parent.Direction = sibling.Direction
	parent.Window = sibling.Window
	parent.Children = sibling.Children

	// Update parent pointers for promoted children
	if parent.Children[0] != nil {
		parent.Children[0].Parent = parent
	}
	if parent.Children[1] != nil {
		parent.Children[1].Parent = parent
	}

	// Focus the first window in the sibling subtree
	ed.focusedWindow = parent.FirstWindow()

	// Recalculate viewport sizes based on available space
	size := ed.app.Size()
	ed.recalculateViewports(ed.root, size.Width, size.Height-headerRows-footerRows)

	ed.app.SetView(buildView(ed))
	ed.updateAllWindows()
	ed.updateCursor()
}

// closeOtherWindows closes all windows except the current one
func (ed *Editor) closeOtherWindows() {
	// If already just one window, nothing to do
	if ed.root.IsLeaf() {
		return
	}

	// Reset to single window
	ed.root = &SplitNode{Window: ed.focusedWindow}

	// Reclaim full viewport
	size := ed.app.Size()
	ed.focusedWindow.viewportHeight = max(1, size.Height-headerRows-footerRows)
	ed.focusedWindow.viewportWidth = size.Width
	ed.initWindowLayer(ed.focusedWindow, size.Width)

	ed.app.SetView(buildView(ed))
	ed.updateAllWindows()
	ed.updateCursor()
}

// focusNextWindow moves focus to the next window
func (ed *Editor) focusNextWindow() {
	allWindows := ed.root.AllWindows()
	if len(allWindows) <= 1 {
		return
	}
	// Find current window index
	for i, w := range allWindows {
		if w == ed.focusedWindow {
			ed.focusedWindow = allWindows[(i+1)%len(allWindows)]
			break
		}
	}
	ed.updateAllWindows()
	ed.updateCursor()
}

// focusPrevWindow moves focus to the previous window
func (ed *Editor) focusPrevWindow() {
	allWindows := ed.root.AllWindows()
	if len(allWindows) <= 1 {
		return
	}
	// Find current window index
	for i, w := range allWindows {
		if w == ed.focusedWindow {
			ed.focusedWindow = allWindows[(i-1+len(allWindows))%len(allWindows)]
			break
		}
	}
	ed.updateAllWindows()
	ed.updateCursor()
}

// focusDirection moves focus to an adjacent window in the specified direction
func (ed *Editor) focusDirection(dir SplitDir, delta int) {
	// Find the current window's node
	node := ed.root.FindWindow(ed.focusedWindow)
	if node == nil {
		return
	}

	// Walk up to find a parent with the matching direction
	for node.Parent != nil {
		parent := node.Parent
		if parent.Direction == dir {
			// Found a split in the right direction
			var target *SplitNode
			if delta > 0 {
				// Move forward (down/right) - if we're first child, go to second
				if parent.Children[0] == node {
					target = parent.Children[1]
				}
			} else {
				// Move backward (up/left) - if we're second child, go to first
				if parent.Children[1] == node {
					target = parent.Children[0]
				}
			}
			if target != nil {
				// Focus the appropriate window in the target subtree
				if delta > 0 {
					ed.focusedWindow = target.FirstWindow()
				} else {
					ed.focusedWindow = target.LastWindow()
				}
				ed.updateAllWindows()
				ed.updateCursor()
				return
			}
		}
		node = parent
	}
}

// updateAllWindows updates the display for all windows
func (ed *Editor) updateAllWindows() {
	for _, w := range ed.root.AllWindows() {
		ed.updateWindowDisplay(w, w == ed.focusedWindow)
	}
}

// recalculateViewports recursively calculates viewport sizes for the tree
func (ed *Editor) recalculateViewports(node *SplitNode, width, height int) {
	if node.IsLeaf() {
		node.Window.viewportWidth = width
		node.Window.viewportHeight = height - 1 // -1 for status bar
		ed.initWindowLayer(node.Window, width)
		return
	}

	if node.Direction == SplitHorizontal {
		// Stack vertically - split height
		halfHeight := height / 2
		ed.recalculateViewports(node.Children[0], width, halfHeight)
		ed.recalculateViewports(node.Children[1], width, height-halfHeight)
	} else {
		// Side by side - split width
		halfWidth := width / 2
		ed.recalculateViewports(node.Children[0], halfWidth, height)
		ed.recalculateViewports(node.Children[1], width-halfWidth, height)
	}
}

// updateWindowDisplay updates a specific window's display
func (ed *Editor) updateWindowDisplay(w *Window, focused bool) {
	ed.ensureWindowRendered(w)
	ed.updateWindowStatusBar(w, focused)
}

// ensureRendered makes sure visible region + buffer is rendered.
// This is the lazy rendering entry point - call after any scroll or cursor move.
func (ed *Editor) ensureRendered() {
	if ed.win().contentLayer == nil || ed.win().contentLayer.Buffer() == nil {
		return
	}

	start := time.Now()
	linesRendered := 0

	// Calculate line number width based on total lines
	maxLineNum := len(ed.buf().Lines)
	ed.win().lineNumWidth = len(fmt.Sprintf("%d", maxLineNum)) + 1

	// Calculate desired render range (visible + buffer)
	wantMin := max(0, ed.win().topLine-renderBuffer)
	wantMax := min(len(ed.buf().Lines)+ed.win().viewportHeight-1, ed.win().topLine+ed.win().viewportHeight+renderBuffer)

	// First time? Render the whole range
	if ed.win().renderedMin < 0 {
		for i := wantMin; i <= wantMax; i++ {
			ed.renderLineToLayer(i)
			linesRendered++
		}
		ed.win().renderedMin = wantMin
		ed.win().renderedMax = wantMax
	} else {
		// Expand rendered range if needed
		// Render any lines below current min
		if wantMin < ed.win().renderedMin {
			for i := wantMin; i < ed.win().renderedMin; i++ {
				ed.renderLineToLayer(i)
				linesRendered++
			}
			ed.win().renderedMin = wantMin
		}
		// Render any lines above current max
		if wantMax > ed.win().renderedMax {
			for i := ed.win().renderedMax + 1; i <= wantMax; i++ {
				ed.renderLineToLayer(i)
				linesRendered++
			}
			ed.win().renderedMax = wantMax
		}
	}

	// Set scroll position
	ed.win().contentLayer.ScrollTo(ed.win().topLine)

	// Track stats
	ed.win().lastRenderTime = time.Since(start)
	ed.win().lastLinesRendered = linesRendered
	ed.win().totalRenders++
	ed.win().totalLinesRendered += linesRendered
}

// invalidateRenderedRange marks that content has changed and needs re-render.
// Call after insert/delete operations that modify line content.
func (ed *Editor) invalidateRenderedRange() {
	ed.win().renderedMin = -1
	ed.win().renderedMax = -1
}

// renderLineToLayer renders a single line to the layer buffer
func (ed *Editor) renderLineToLayer(lineIdx int) {
	if ed.win().contentLayer == nil || ed.win().contentLayer.Buffer() == nil {
		return
	}

	lineNumFmt := fmt.Sprintf("%%%dd ", ed.win().lineNumWidth-1)
	tildeFmt := fmt.Sprintf("%%%ds ", ed.win().lineNumWidth-1)

	var spans []tui.Span

	if lineIdx < len(ed.buf().Lines) {
		// Content line
		line := ed.buf().Lines[lineIdx]
		lineNum := fmt.Sprintf(lineNumFmt, lineIdx+1)
		isCursorLine := lineIdx == ed.win().Cursor

		numStyle := lineNumStyle
		if isCursorLine {
			numStyle = cursorLineNumStyle
		}

		if ed.Mode == "VISUAL" {
			spans = append([]tui.Span{{Text: lineNum, Style: numStyle}}, ed.getVisualSpans(lineIdx, line)...)
		} else {
			contentSpans := ed.highlightSearchMatches(line)
			spans = append([]tui.Span{{Text: lineNum, Style: numStyle}}, contentSpans...)
		}
	} else {
		// Tilde line (beyond EOF)
		spans = []tui.Span{{Text: fmt.Sprintf(tildeFmt, "~"), Style: tildeStyle}}
	}

	ed.win().contentLayer.SetLine(lineIdx, spans)
}

// updateLine efficiently updates just the changed lines (for cursor movement)
func (ed *Editor) updateLine(lineIdx int) {
	ed.renderLineToLayer(lineIdx)
}

// updateCursorHighlight efficiently updates highlight when cursor moves between lines
// Returns the old cursor line index for callers that need it
func (ed *Editor) updateCursorHighlight(oldLine int) {
	if ed.win().contentLayer == nil {
		return
	}

	// Ensure visible region is rendered (lazy render on scroll)
	ed.ensureRendered()

	// Only update if cursor actually moved to a different line
	if oldLine != ed.win().Cursor && oldLine >= 0 && oldLine < len(ed.buf().Lines) {
		ed.renderLineToLayer(oldLine) // Remove yellow from old line
	}
	ed.renderLineToLayer(ed.win().Cursor) // Add yellow to new line

	// Sync layer scroll position
	ed.win().contentLayer.ScrollTo(ed.win().topLine)

	// Update status bar (needed for debug stats to refresh)
	ed.updateStatusBar()
}

// getVisualSpans splits a line into styled spans for visual mode highlighting
func (ed *Editor) getVisualSpans(lineIdx int, line string) []tui.Span {
	inverseStyle := tui.Style{Attr: tui.AttrInverse}
	normalStyle := tui.Style{}

	if len(line) == 0 {
		if ed.isLineSelected(lineIdx) {
			return []tui.Span{{Text: " ", Style: inverseStyle}} // Show at least a space for empty lines
		}
		return []tui.Span{{Text: " ", Style: normalStyle}}
	}

	if ed.win().visualLineMode {
		// Line mode: entire line is selected or not
		if ed.isLineSelected(lineIdx) {
			return []tui.Span{{Text: line, Style: inverseStyle}}
		}
		return []tui.Span{{Text: line, Style: normalStyle}}
	}

	// Character mode: need to calculate per-character selection
	// Simplified: only works for single-line selection for now
	if lineIdx != ed.win().Cursor && lineIdx != ed.win().visualStart {
		// Line is either fully selected or not (if between start and cursor)
		if ed.isLineSelected(lineIdx) {
			return []tui.Span{{Text: line, Style: inverseStyle}}
		}
		return []tui.Span{{Text: line, Style: normalStyle}}
	}

	// This is the line with the cursor or visual start - split into spans
	startCol := min(ed.win().visualStartCol, ed.win().Col)
	endCol := max(ed.win().visualStartCol, ed.win().Col) + 1

	if lineIdx != ed.win().Cursor || lineIdx != ed.win().visualStart {
		// Multi-line selection - this line is start or end
		if lineIdx == min(ed.win().visualStart, ed.win().Cursor) {
			// First line - select from startCol to end
			col := ed.win().visualStartCol
			if lineIdx == ed.win().Cursor {
				col = ed.win().Col
			}
			if lineIdx != ed.win().visualStart {
				col = ed.win().Col
			} else {
				col = ed.win().visualStartCol
			}
			// Simplified: highlight from col to end
			startCol = min(col, len(line))
			endCol = len(line)
		} else {
			// Last line - select from start to col
			col := ed.win().Col
			if lineIdx == ed.win().visualStart {
				col = ed.win().visualStartCol
			}
			startCol = 0
			endCol = min(col+1, len(line))
		}
	}

	// Clamp
	startCol = max(0, min(startCol, len(line)))
	endCol = max(0, min(endCol, len(line)))

	var spans []tui.Span
	if startCol > 0 {
		spans = append(spans, tui.Span{Text: line[:startCol], Style: normalStyle})
	}
	if startCol < endCol {
		spans = append(spans, tui.Span{Text: line[startCol:endCol], Style: inverseStyle})
	}
	if endCol < len(line) {
		spans = append(spans, tui.Span{Text: line[endCol:], Style: normalStyle})
	}
	return spans
}

// isLineSelected returns true if a line is within the visual selection
func (ed *Editor) isLineSelected(lineIdx int) bool {
	minLine := min(ed.win().visualStart, ed.win().Cursor)
	maxLine := max(ed.win().visualStart, ed.win().Cursor)
	return lineIdx >= minLine && lineIdx <= maxLine
}

// buildWindowView builds the view for a single window
func buildWindowView(w *Window, focused bool) any {
	return tui.Col{Children: []any{
		// Content area - imperative layer, efficiently updated
		// Width is set for vertical splits to constrain each window's area
		tui.LayerView{
			Layer:  w.contentLayer,
			Height: int16(w.viewportHeight),
			Width:  int16(w.viewportWidth),
		},
		// Vim-style status bar (inverse video, shows filename and position)
		tui.RichText{Spans: &w.StatusBar},
	}}
}

// buildNodeView recursively builds the view for a split node
func buildNodeView(node *SplitNode, focusedWindow *Window) any {
	if node.IsLeaf() {
		return buildWindowView(node.Window, node.Window == focusedWindow)
	}

	// Build children recursively
	child0 := buildNodeView(node.Children[0], focusedWindow)
	child1 := buildNodeView(node.Children[1], focusedWindow)

	if node.Direction == SplitHorizontal {
		// Stack vertically (Col)
		return tui.Col{Children: []any{child0, child1}}
	}
	// Side by side (Row)
	return tui.Row{Children: []any{child0, child1}}
}

func buildView(ed *Editor) any {
	// Build the window tree
	windowTree := buildNodeView(ed.root, ed.focusedWindow)

	// Wrap in Col to add global status line at bottom
	return tui.Col{Children: []any{
		windowTree,
		tui.Text{Content: &ed.StatusLine},
	}}
}

// Text object functions return (start, end) range - end is exclusive
type TextObjectFunc func(line string, col int) (start, end int)

// Multi-line text object functions return line and column ranges
type MultiLineTextObjectFunc func(ed *Editor) (startLine, startCol, endLine, endCol int)

// Operator functions act on a range within a single line
type OperatorFunc func(ed *Editor, app *tui.App, start, end int)

// Multi-line operator functions act on a range across lines
type MultiLineOperatorFunc func(ed *Editor, app *tui.App, startLine, startCol, endLine, endCol int)

// registerOperatorTextObjects sets up all operator+textobject combinations
func registerOperatorTextObjects(app *tui.App, ed *Editor) {
	operators := []struct {
		key string
		fn  OperatorFunc
	}{
		{"d", opDelete},
		{"c", opChange},
		{"y", opYank},
	}

	// Single-line text objects (words, quotes)
	// Note: brackets/braces/parens are handled by multi-line versions in registerParagraphTextObjects
	textObjects := []struct {
		key string
		fn  TextObjectFunc
	}{
		{"iw", toInnerWord},
		{"aw", toAWord},
		{"iW", toInnerWORD},
		{"aW", toAWORD},
		{"i\"", toInnerDoubleQuote},
		{"a\"", toADoubleQuote},
		{"i'", toInnerSingleQuote},
		{"a'", toASingleQuote},
	}

	for _, op := range operators {
		for _, obj := range textObjects {
			pattern := op.key + obj.key
			opFn, objFn := op.fn, obj.fn // capture for closure
			app.Handle(pattern, func(m riffkey.Match) {
				line := ed.buf().Lines[ed.win().Cursor]
				start, end := objFn(line, ed.win().Col)
				if start < end {
					opFn(ed, app, start, end)
				}
			})
		}
	}

	// Multi-line text objects (paragraphs)
	// ip = inner paragraph, ap = a paragraph (includes trailing blank lines)
	registerParagraphTextObjects(app, ed)
}

// Operators
func opDelete(ed *Editor, app *tui.App, start, end int) {
	ed.saveUndo()
	line := ed.buf().Lines[ed.win().Cursor]
	ed.buf().Lines[ed.win().Cursor] = line[:start] + line[end:]
	ed.win().Col = start
	if ed.win().Col >= len(ed.buf().Lines[ed.win().Cursor]) && ed.win().Col > 0 {
		ed.win().Col = max(0, len(ed.buf().Lines[ed.win().Cursor])-1)
	}
	ed.updateDisplay()
	ed.updateCursor()
}

func opChange(ed *Editor, app *tui.App, start, end int) {
	ed.saveUndo()
	line := ed.buf().Lines[ed.win().Cursor]
	ed.buf().Lines[ed.win().Cursor] = line[:start] + line[end:]
	ed.win().Col = start
	ed.updateDisplay()
	ed.enterInsertMode(app)
}

var yankRegister string

func opYank(ed *Editor, app *tui.App, start, end int) {
	line := ed.buf().Lines[ed.win().Cursor]
	yankRegister = line[start:end]
	ed.StatusLine = fmt.Sprintf("Yanked: %q", yankRegister)
	ed.updateDisplay()
}

// Text objects

// Inner word: just the word characters
func toInnerWord(line string, col int) (int, int) {
	if col >= len(line) {
		return col, col
	}
	start, end := col, col
	// Expand left
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}
	// Expand right
	for end < len(line) && isWordChar(line[end]) {
		end++
	}
	return start, end
}

// A word: word + trailing whitespace
func toAWord(line string, col int) (int, int) {
	start, end := toInnerWord(line, col)
	// Include trailing whitespace
	for end < len(line) && line[end] == ' ' {
		end++
	}
	return start, end
}

// Inner WORD: non-whitespace characters
func toInnerWORD(line string, col int) (int, int) {
	if col >= len(line) {
		return col, col
	}
	start, end := col, col
	for start > 0 && line[start-1] != ' ' {
		start--
	}
	for end < len(line) && line[end] != ' ' {
		end++
	}
	return start, end
}

// A WORD: WORD + trailing whitespace
func toAWORD(line string, col int) (int, int) {
	start, end := toInnerWORD(line, col)
	for end < len(line) && line[end] == ' ' {
		end++
	}
	return start, end
}

// Inner quotes helper
func toInnerQuoteChar(line string, col int, quote byte) (int, int) {
	// Find opening quote
	start := col
	for start >= 0 && (start >= len(line) || line[start] != quote) {
		start--
	}
	if start < 0 {
		return col, col
	}
	// Find closing quote
	end := col
	if end <= start {
		end = start + 1
	}
	for end < len(line) && line[end] != quote {
		end++
	}
	if end >= len(line) {
		return col, col
	}
	return start + 1, end // exclude quotes
}

// A quote helper
func toAQuoteChar(line string, col int, quote byte) (int, int) {
	start, end := toInnerQuoteChar(line, col, quote)
	if start > 0 && line[start-1] == quote {
		start--
	}
	if end < len(line) && line[end] == quote {
		end++
	}
	return start, end
}

func toInnerDoubleQuote(line string, col int) (int, int) { return toInnerQuoteChar(line, col, '"') }
func toADoubleQuote(line string, col int) (int, int)     { return toAQuoteChar(line, col, '"') }
func toInnerSingleQuote(line string, col int) (int, int) { return toInnerQuoteChar(line, col, '\'') }
func toASingleQuote(line string, col int) (int, int) { return toAQuoteChar(line, col, '\'') }

func isSentenceEnd(c byte) bool {
	return c == '.' || c == '!' || c == '?'
}

// Multi-line text objects (paragraphs, sentences)
func registerParagraphTextObjects(app *tui.App, ed *Editor) {
	// Multi-line operators
	mlOperators := []struct {
		key string
		fn  MultiLineOperatorFunc
	}{
		{"d", mlOpDelete},
		{"c", mlOpChange},
		{"y", mlOpYank},
	}

	// Multi-line text objects
	mlTextObjects := []struct {
		key string
		fn  MultiLineTextObjectFunc
	}{
		{"ip", toInnerParagraphML},
		{"ap", toAParagraphML},
		{"is", toInnerSentenceML},
		{"as", toASentenceML},
		// Brackets - multi-line versions
		{"i(", toInnerParenML},
		{"a(", toAParenML},
		{"i)", toInnerParenML},
		{"a)", toAParenML},
		{"i[", toInnerBracketML},
		{"a[", toABracketML},
		{"i]", toInnerBracketML},
		{"a]", toABracketML},
		{"i{", toInnerBraceML},
		{"a{", toABraceML},
		{"i}", toInnerBraceML},
		{"a}", toABraceML},
		{"i<", toInnerAngleML},
		{"a<", toAAngleML},
		{"i>", toInnerAngleML},
		{"a>", toAAngleML},
	}

	// Register all text object combinations
	for _, op := range mlOperators {
		for _, obj := range mlTextObjects {
			pattern := op.key + obj.key
			opFn, objFn := op.fn, obj.fn // capture for closure
			app.Handle(pattern, func(_ riffkey.Match) {
				startLine, startCol, endLine, endCol := objFn(ed)
				if startLine >= 0 {
					opFn(ed, app, startLine, startCol, endLine, endCol)
				}
			})
		}
	}

	// Motion functions for operator + motion (dj, yk, cw, etc.)
	// All motions return (startLine, startCol, endLine, endCol) and reuse mlOperators
	mlMotions := []struct {
		key string
		fn  func(ed *Editor, count int) (startLine, startCol, endLine, endCol int)
	}{
		// Linewise motions (full lines: col 0 to end)
		{"j", func(ed *Editor, count int) (int, int, int, int) {
			endLine := min(ed.win().Cursor+count, len(ed.buf().Lines)-1)
			return ed.win().Cursor, 0, endLine, len(ed.buf().Lines[endLine])
		}},
		{"k", func(ed *Editor, count int) (int, int, int, int) {
			startLine := max(ed.win().Cursor-count, 0)
			return startLine, 0, ed.win().Cursor, len(ed.buf().Lines[ed.win().Cursor])
		}},
		{"gg", func(ed *Editor, count int) (int, int, int, int) {
			return 0, 0, ed.win().Cursor, len(ed.buf().Lines[ed.win().Cursor])
		}},
		{"G", func(ed *Editor, count int) (int, int, int, int) {
			endLine := len(ed.buf().Lines) - 1
			return ed.win().Cursor, 0, endLine, len(ed.buf().Lines[endLine])
		}},
		// Characterwise motions
		{"w", func(ed *Editor, count int) (int, int, int, int) {
			startLine, startCol := ed.win().Cursor, ed.win().Col
			for range count {
				ed.wordForward()
			}
			endLine, endCol := ed.win().Cursor, ed.win().Col
			ed.win().Cursor, ed.win().Col = startLine, startCol
			return startLine, startCol, endLine, endCol
		}},
		{"b", func(ed *Editor, count int) (int, int, int, int) {
			endLine, endCol := ed.win().Cursor, ed.win().Col
			for range count {
				ed.wordBackward()
			}
			startLine, startCol := ed.win().Cursor, ed.win().Col
			ed.win().Cursor, ed.win().Col = endLine, endCol
			return startLine, startCol, endLine, endCol
		}},
		{"e", func(ed *Editor, count int) (int, int, int, int) {
			startLine, startCol := ed.win().Cursor, ed.win().Col
			for range count {
				ed.wordEnd()
			}
			endLine, endCol := ed.win().Cursor, ed.win().Col+1
			ed.win().Cursor, ed.win().Col = startLine, startCol
			return startLine, startCol, endLine, endCol
		}},
		{"$", func(ed *Editor, count int) (int, int, int, int) {
			return ed.win().Cursor, ed.win().Col, ed.win().Cursor, len(ed.buf().Lines[ed.win().Cursor])
		}},
		{"0", func(ed *Editor, count int) (int, int, int, int) {
			return ed.win().Cursor, 0, ed.win().Cursor, ed.win().Col
		}},
	}

	// Register operator + motion combinations (reuses mlOperators)
	for _, op := range mlOperators {
		for _, mot := range mlMotions {
			pattern := op.key + mot.key
			opFn, motFn := op.fn, mot.fn
			app.Handle(pattern, func(m riffkey.Match) {
				startLine, startCol, endLine, endCol := motFn(ed, m.Count)
				opFn(ed, app, startLine, startCol, endLine, endCol)
			})
		}
	}

	// cc - change whole line
	app.Handle("cc", func(_ riffkey.Match) {
		ed.saveUndo()
		ed.buf().Lines[ed.win().Cursor] = ""
		ed.win().Col = 0
		ed.updateDisplay()
		ed.enterInsertMode(app)
	})

	// S - same as cc
	app.Handle("S", func(_ riffkey.Match) {
		ed.saveUndo()
		ed.buf().Lines[ed.win().Cursor] = ""
		ed.win().Col = 0
		ed.updateDisplay()
		ed.enterInsertMode(app)
	})

	// yy - yank whole line
	app.Handle("yy", func(_ riffkey.Match) {
		yankRegister = ed.buf().Lines[ed.win().Cursor]
		ed.StatusLine = fmt.Sprintf("Yanked: %q", yankRegister)
		ed.updateDisplay()
	})

	// Y - same as yy
	app.Handle("Y", func(_ riffkey.Match) {
		yankRegister = ed.buf().Lines[ed.win().Cursor]
		ed.StatusLine = fmt.Sprintf("Yanked: %q", yankRegister)
		ed.updateDisplay()
	})
}

// findInnerParagraph returns the line range of the current paragraph (non-blank lines)
func (ed *Editor) findInnerParagraph() (startLine, endLine int) {
	// If on a blank line, return just this line
	if strings.TrimSpace(ed.buf().Lines[ed.win().Cursor]) == "" {
		return ed.win().Cursor, ed.win().Cursor
	}

	// Find start of paragraph (first non-blank line going backward)
	startLine = ed.win().Cursor
	for startLine > 0 && strings.TrimSpace(ed.buf().Lines[startLine-1]) != "" {
		startLine--
	}

	// Find end of paragraph (last non-blank line going forward)
	endLine = ed.win().Cursor
	for endLine < len(ed.buf().Lines)-1 && strings.TrimSpace(ed.buf().Lines[endLine+1]) != "" {
		endLine++
	}

	return startLine, endLine
}

// findAParagraph returns the line range including trailing blank lines
func (ed *Editor) findAParagraph() (startLine, endLine int) {
	startLine, endLine = ed.findInnerParagraph()

	// Include trailing blank lines
	for endLine < len(ed.buf().Lines)-1 && strings.TrimSpace(ed.buf().Lines[endLine+1]) == "" {
		endLine++
	}

	return startLine, endLine
}

// Multi-line operators
func mlOpDelete(ed *Editor, app *tui.App, startLine, startCol, endLine, endCol int) {
	ed.saveUndo()

	// Extract the text being deleted for yank register
	yankRegister = ed.extractRange(startLine, startCol, endLine, endCol)

	// Delete the range
	ed.deleteRange(startLine, startCol, endLine, endCol)

	ed.updateDisplay()
	ed.updateCursor()
}

func mlOpChange(ed *Editor, app *tui.App, startLine, startCol, endLine, endCol int) {
	ed.saveUndo()

	// Extract for yank register
	yankRegister = ed.extractRange(startLine, startCol, endLine, endCol)

	// Delete the range
	ed.deleteRange(startLine, startCol, endLine, endCol)

	ed.updateDisplay()
	ed.enterInsertMode(app)
}

func mlOpYank(ed *Editor, app *tui.App, startLine, startCol, endLine, endCol int) {
	yankRegister = ed.extractRange(startLine, startCol, endLine, endCol)
	ed.StatusLine = fmt.Sprintf("Yanked: %q", yankRegister)
	ed.updateDisplay()
}

// extractRange extracts text from a multi-line range
func (ed *Editor) extractRange(startLine, startCol, endLine, endCol int) string {
	if startLine == endLine {
		// Same line
		line := ed.buf().Lines[startLine]
		endCol = min(endCol, len(line))
		startCol = min(startCol, len(line))
		return line[startCol:endCol]
	}

	// Multiple lines
	var parts []string

	// First line (from startCol to end)
	if startLine < len(ed.buf().Lines) {
		line := ed.buf().Lines[startLine]
		startCol = min(startCol, len(line))
		parts = append(parts, line[startCol:])
	}

	// Middle lines (full lines)
	for i := startLine + 1; i < endLine && i < len(ed.buf().Lines); i++ {
		parts = append(parts, ed.buf().Lines[i])
	}

	// Last line (from start to endCol)
	if endLine < len(ed.buf().Lines) && endLine > startLine {
		line := ed.buf().Lines[endLine]
		endCol = min(endCol, len(line))
		parts = append(parts, line[:endCol])
	}

	return strings.Join(parts, "\n")
}

// deleteRange deletes text from a multi-line range
func (ed *Editor) deleteRange(startLine, startCol, endLine, endCol int) {
	if startLine == endLine {
		// Same line - simple case
		line := ed.buf().Lines[startLine]
		endCol = min(endCol, len(line))
		startCol = min(startCol, len(line))
		ed.buf().Lines[startLine] = line[:startCol] + line[endCol:]
		ed.win().Cursor = startLine
		ed.win().Col = startCol
		return
	}

	// Multiple lines - join first and last line remnants
	firstPart := ""
	if startLine < len(ed.buf().Lines) {
		line := ed.buf().Lines[startLine]
		startCol = min(startCol, len(line))
		firstPart = line[:startCol]
	}

	lastPart := ""
	if endLine < len(ed.buf().Lines) {
		line := ed.buf().Lines[endLine]
		endCol = min(endCol, len(line))
		lastPart = line[endCol:]
	}

	// Create new lines array
	newLines := make([]string, 0, len(ed.buf().Lines)-(endLine-startLine))
	newLines = append(newLines, ed.buf().Lines[:startLine]...)
	newLines = append(newLines, firstPart+lastPart)
	if endLine+1 < len(ed.buf().Lines) {
		newLines = append(newLines, ed.buf().Lines[endLine+1:]...)
	}

	ed.buf().Lines = newLines
	if len(ed.buf().Lines) == 0 {
		ed.buf().Lines = []string{""}
	}
	ed.win().Cursor = min(startLine, len(ed.buf().Lines)-1)
	ed.win().Col = startCol
}

// Multi-line text object functions

// toInnerParagraphML returns the range of the inner paragraph
func toInnerParagraphML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	start, end := ed.findInnerParagraph()
	// For paragraph, we delete whole lines (col 0 to end of last line)
	return start, 0, end, len(ed.buf().Lines[end])
}

// toAParagraphML returns the range including trailing blank lines
func toAParagraphML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	start, end := ed.findAParagraph()
	return start, 0, end, len(ed.buf().Lines[end])
}

// toInnerSentenceML finds the current sentence boundaries across lines
func toInnerSentenceML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	return ed.findSentenceBounds(false)
}

// toASentenceML finds the sentence including trailing whitespace
func toASentenceML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	return ed.findSentenceBounds(true)
}

// findSentenceBounds finds sentence boundaries across lines
func (ed *Editor) findSentenceBounds(includeTrailing bool) (startLine, startCol, endLine, endCol int) {
	// Start from cursor position
	startLine = ed.win().Cursor
	startCol = ed.win().Col
	endLine = ed.win().Cursor
	endCol = ed.win().Col

	// Search backward for sentence start (after previous sentence end or start of paragraph)
	for {
		line := ed.buf().Lines[startLine]
		for startCol > 0 {
			startCol--
			if startCol < len(line) && isSentenceEnd(line[startCol]) {
				// Found previous sentence end - sentence starts after this
				startCol++
				// Skip whitespace
				for startCol < len(line) && (line[startCol] == ' ' || line[startCol] == '\t') {
					startCol++
				}
				if startCol >= len(line) && startLine < len(ed.buf().Lines)-1 {
					// Move to next line
					startLine++
					startCol = 0
					line = ed.buf().Lines[startLine]
					// Skip leading whitespace on next line
					for startCol < len(line) && (line[startCol] == ' ' || line[startCol] == '\t') {
						startCol++
					}
				}
				goto foundStart
			}
		}
		// Reached start of line, check previous line
		if startLine > 0 {
			// Check if previous line is blank (paragraph boundary)
			if strings.TrimSpace(ed.buf().Lines[startLine-1]) == "" {
				startCol = 0
				goto foundStart
			}
			startLine--
			startCol = len(ed.buf().Lines[startLine])
		} else {
			// Start of file
			startCol = 0
			goto foundStart
		}
	}
foundStart:

	// Search forward for sentence end
	for {
		line := ed.buf().Lines[endLine]
		for endCol < len(line) {
			if isSentenceEnd(line[endCol]) {
				endCol++ // Include the punctuation
				goto foundEnd
			}
			endCol++
		}
		// Reached end of line, check next line
		if endLine < len(ed.buf().Lines)-1 {
			// Check if next line is blank (paragraph boundary)
			if strings.TrimSpace(ed.buf().Lines[endLine+1]) == "" {
				endCol = len(line)
				goto foundEnd
			}
			endLine++
			endCol = 0
		} else {
			// End of file
			endCol = len(line)
			goto foundEnd
		}
	}
foundEnd:

	// Include trailing whitespace if requested
	if includeTrailing {
		for {
			line := ed.buf().Lines[endLine]
			for endCol < len(line) && (line[endCol] == ' ' || line[endCol] == '\t') {
				endCol++
			}
			if endCol < len(line) {
				break // Found non-whitespace
			}
			// Check next line
			if endLine < len(ed.buf().Lines)-1 && strings.TrimSpace(ed.buf().Lines[endLine+1]) != "" {
				endLine++
				endCol = 0
			} else {
				break
			}
		}
	}

	return startLine, startCol, endLine, endCol
}

// Multi-line bracket/brace/paren text objects

// findPairBoundsML finds matching bracket pairs across multiple lines.
// If cursor is inside a pair, uses that. Otherwise searches forward for next pair.
func (ed *Editor) findPairBoundsML(open, close byte, inner bool) (startLine, startCol, endLine, endCol int) {
	// Try to find a pair containing the cursor, or search forward for next pair
	startLine, startCol, endLine, endCol = ed.findPairContaining(open, close)
	if startLine < 0 {
		// Not inside a pair - search forward for next opening bracket
		startLine, startCol, endLine, endCol = ed.findNextPair(open, close)
	}
	if startLine < 0 {
		return -1, -1, -1, -1
	}

	if inner {
		// Exclude the brackets themselves
		startCol++
		// If startCol goes past end of line, move to next line
		if startCol >= len(ed.buf().Lines[startLine]) && startLine < endLine {
			startLine++
			startCol = 0
		}
		// endCol already points at closing bracket, so we don't include it
	} else {
		// Include both brackets
		endCol++
	}

	return startLine, startCol, endLine, endCol
}

// findPairContaining searches backward for an opening bracket and forward for its match.
// Returns the bracket positions if cursor is inside a pair, or -1,-1,-1,-1 if not.
func (ed *Editor) findPairContaining(open, close byte) (startLine, startCol, endLine, endCol int) {
	// Search backward for opening bracket
	startLine = ed.win().Cursor
	startCol = ed.win().Col
	depth := 0

	for {
		line := ed.buf().Lines[startLine]
		for startCol >= 0 {
			if startCol < len(line) {
				ch := line[startCol]
				if ch == close {
					depth++
				} else if ch == open {
					if depth == 0 {
						// Found opening bracket - now verify there's a matching close after cursor
						endLine, endCol = ed.findMatchingClose(open, close, startLine, startCol)
						if endLine >= 0 {
							return startLine, startCol, endLine, endCol
						}
						// No matching close, keep searching backward
					}
					depth--
				}
			}
			startCol--
		}
		if startLine > 0 {
			startLine--
			startCol = len(ed.buf().Lines[startLine]) - 1
		} else {
			return -1, -1, -1, -1
		}
	}
}

// findMatchingClose searches forward from an opening bracket for its matching close.
func (ed *Editor) findMatchingClose(open, close byte, fromLine, fromCol int) (endLine, endCol int) {
	endLine = fromLine
	endCol = fromCol + 1 // Start after the opening bracket
	depth := 0

	for {
		line := ed.buf().Lines[endLine]
		for endCol < len(line) {
			ch := line[endCol]
			if ch == open {
				depth++
			} else if ch == close {
				if depth == 0 {
					return endLine, endCol
				}
				depth--
			}
			endCol++
		}
		if endLine < len(ed.buf().Lines)-1 {
			endLine++
			endCol = 0
		} else {
			return -1, -1
		}
	}
}

// findNextPair searches forward for the next opening bracket and its matching close.
func (ed *Editor) findNextPair(open, close byte) (startLine, startCol, endLine, endCol int) {
	startLine = ed.win().Cursor
	startCol = ed.win().Col + 1 // Start after cursor

	for {
		line := ed.buf().Lines[startLine]
		for startCol < len(line) {
			if line[startCol] == open {
				// Found opening bracket - find its match
				endLine, endCol = ed.findMatchingClose(open, close, startLine, startCol)
				if endLine >= 0 {
					return startLine, startCol, endLine, endCol
				}
			}
			startCol++
		}
		if startLine < len(ed.buf().Lines)-1 {
			startLine++
			startCol = 0
		} else {
			return -1, -1, -1, -1
		}
	}
}

// Paren text objects
func toInnerParenML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	return ed.findPairBoundsML('(', ')', true)
}
func toAParenML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	return ed.findPairBoundsML('(', ')', false)
}

// Bracket text objects
func toInnerBracketML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	return ed.findPairBoundsML('[', ']', true)
}
func toABracketML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	return ed.findPairBoundsML('[', ']', false)
}

// Brace text objects
func toInnerBraceML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	return ed.findPairBoundsML('{', '}', true)
}
func toABraceML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	return ed.findPairBoundsML('{', '}', false)
}

// Angle bracket text objects
func toInnerAngleML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	return ed.findPairBoundsML('<', '>', true)
}
func toAAngleML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	return ed.findPairBoundsML('<', '>', false)
}

// Word motion helper
func isWordChar(r byte) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// Cross-line word motions

// wordForward moves to the start of the next word, crossing lines
func (ed *Editor) wordForward() {
	line := ed.buf().Lines[ed.win().Cursor]
	n := len(line)

	// Try to find next word on current line
	col := ed.win().Col
	// Skip current word
	for col < n && isWordChar(line[col]) {
		col++
	}
	// Skip whitespace/punctuation
	for col < n && !isWordChar(line[col]) {
		col++
	}

	if col < n {
		// Found word on this line
		ed.win().Col = col
		return
	}

	// Move to next line
	for ed.win().Cursor < len(ed.buf().Lines)-1 {
		ed.win().Cursor++
		line = ed.buf().Lines[ed.win().Cursor]
		// Find first word char on new line
		col = 0
		for col < len(line) && !isWordChar(line[col]) {
			col++
		}
		if col < len(line) {
			ed.win().Col = col
			return
		}
	}
	// At end, go to end of last line
	ed.win().Col = max(0, len(ed.buf().Lines[ed.win().Cursor])-1)
}

// wordBackward moves to the start of the previous word, crossing lines
func (ed *Editor) wordBackward() {
	line := ed.buf().Lines[ed.win().Cursor]
	col := ed.win().Col

	if col > 0 {
		col--
		// Skip whitespace/punctuation backwards
		for col > 0 && !isWordChar(line[col]) {
			col--
		}
		// Skip word backwards to start
		for col > 0 && isWordChar(line[col-1]) {
			col--
		}
		if col > 0 || isWordChar(line[0]) {
			ed.win().Col = col
			return
		}
	}

	// Move to previous line
	for ed.win().Cursor > 0 {
		ed.win().Cursor--
		line = ed.buf().Lines[ed.win().Cursor]
		if len(line) == 0 {
			continue
		}
		// Find last word on this line
		col = len(line) - 1
		// Skip trailing non-word chars
		for col >= 0 && !isWordChar(line[col]) {
			col--
		}
		if col < 0 {
			continue
		}
		// Skip word backwards to start
		for col > 0 && isWordChar(line[col-1]) {
			col--
		}
		ed.win().Col = col
		return
	}
	// At start
	ed.win().Col = 0
}

// wordEnd moves to the end of the current/next word, crossing lines
func (ed *Editor) wordEnd() {
	line := ed.buf().Lines[ed.win().Cursor]
	n := len(line)
	col := ed.win().Col

	if col < n-1 {
		col++
		// Skip whitespace/punctuation
		for col < n && !isWordChar(line[col]) {
			col++
		}
		// Go to end of word
		for col < n-1 && isWordChar(line[col+1]) {
			col++
		}
		if col < n && isWordChar(line[col]) {
			ed.win().Col = col
			return
		}
	}

	// Move to next line
	for ed.win().Cursor < len(ed.buf().Lines)-1 {
		ed.win().Cursor++
		line = ed.buf().Lines[ed.win().Cursor]
		n = len(line)
		// Find first word
		col = 0
		for col < n && !isWordChar(line[col]) {
			col++
		}
		if col >= n {
			continue
		}
		// Go to end of that word
		for col < n-1 && isWordChar(line[col+1]) {
			col++
		}
		ed.win().Col = col
		return
	}
	// At end
	ed.win().Col = max(0, len(ed.buf().Lines[ed.win().Cursor])-1)
}

// Undo/Redo implementation
func (ed *Editor) saveUndo() {
	// Deep copy current state
	linesCopy := make([]string, len(ed.buf().Lines))
	copy(linesCopy, ed.buf().Lines)
	ed.buf().undoStack = append(ed.buf().undoStack, EditorState{
		Lines:  linesCopy,
		Cursor: ed.win().Cursor,
		Col:    ed.win().Col,
	})
	// Clear redo stack on new change
	ed.buf().redoStack = nil
}

func (ed *Editor) undo() {
	if len(ed.buf().undoStack) == 0 {
		ed.StatusLine = "Already at oldest change"
		ed.updateDisplay()
		return
	}
	// Save current state to redo stack
	linesCopy := make([]string, len(ed.buf().Lines))
	copy(linesCopy, ed.buf().Lines)
	ed.buf().redoStack = append(ed.buf().redoStack, EditorState{
		Lines:  linesCopy,
		Cursor: ed.win().Cursor,
		Col:    ed.win().Col,
	})
	// Pop from undo stack
	state := ed.buf().undoStack[len(ed.buf().undoStack)-1]
	ed.buf().undoStack = ed.buf().undoStack[:len(ed.buf().undoStack)-1]
	ed.buf().Lines = state.Lines
	ed.win().Cursor = state.Cursor
	ed.win().Col = state.Col
	ed.StatusLine = fmt.Sprintf("Undo (%d more)", len(ed.buf().undoStack))
	ed.updateDisplay()
	ed.updateCursor()
}

func (ed *Editor) redo() {
	if len(ed.buf().redoStack) == 0 {
		ed.StatusLine = "Already at newest change"
		ed.updateDisplay()
		return
	}
	// Save current state to undo stack
	linesCopy := make([]string, len(ed.buf().Lines))
	copy(linesCopy, ed.buf().Lines)
	ed.buf().undoStack = append(ed.buf().undoStack, EditorState{
		Lines:  linesCopy,
		Cursor: ed.win().Cursor,
		Col:    ed.win().Col,
	})
	// Pop from redo stack
	state := ed.buf().redoStack[len(ed.buf().redoStack)-1]
	ed.buf().redoStack = ed.buf().redoStack[:len(ed.buf().redoStack)-1]
	ed.buf().Lines = state.Lines
	ed.win().Cursor = state.Cursor
	ed.win().Col = state.Col
	ed.StatusLine = fmt.Sprintf("Redo (%d more)", len(ed.buf().redoStack))
	ed.updateDisplay()
	ed.updateCursor()
}

// Visual mode implementation
func (ed *Editor) enterVisualMode(app *tui.App, lineMode bool) {
	ed.Mode = "VISUAL"
	ed.win().visualStart = ed.win().Cursor
	ed.win().visualStartCol = ed.win().Col
	ed.win().visualLineMode = lineMode
	if lineMode {
		ed.StatusLine = "-- VISUAL LINE --  hjkl:select  d/y:operate  Esc:cancel"
	} else {
		ed.StatusLine = "-- VISUAL --  hjkl:select  d/y:operate  Esc:cancel"
	}
	ed.updateDisplay()

	// Create visual mode router
	visualRouter := riffkey.NewRouter().Name("visual")

	// Movement keys update selection
	// Visual mode needs full refresh for multi-line selection highlighting
	visualRouter.Handle("j", func(m riffkey.Match) {
		ed.win().Cursor = min(ed.win().Cursor+m.Count, len(ed.buf().Lines)-1)
		ed.win().Col = min(ed.win().Col, len(ed.buf().Lines[ed.win().Cursor]))
		ed.ensureCursorVisible()
		ed.refresh()
	})
	visualRouter.Handle("k", func(m riffkey.Match) {
		ed.win().Cursor = max(ed.win().Cursor-m.Count, 0)
		ed.win().Col = min(ed.win().Col, len(ed.buf().Lines[ed.win().Cursor]))
		ed.ensureCursorVisible()
		ed.refresh()
	})
	visualRouter.Handle("h", func(m riffkey.Match) {
		ed.win().Col = max(0, ed.win().Col-m.Count)
		ed.refresh()
	})
	visualRouter.Handle("l", func(m riffkey.Match) {
		ed.win().Col = min(ed.win().Col+m.Count, max(0, len(ed.buf().Lines[ed.win().Cursor])-1))
		ed.refresh()
	})
	visualRouter.Handle("gg", func(_ riffkey.Match) {
		ed.win().Cursor = 0
		ed.win().Col = 0
		ed.ensureCursorVisible()
		ed.refresh()
	})
	visualRouter.Handle("G", func(_ riffkey.Match) {
		ed.win().Cursor = len(ed.buf().Lines) - 1
		ed.win().Col = len(ed.buf().Lines[ed.win().Cursor])
		ed.ensureCursorVisible()
		ed.refresh()
	})
	visualRouter.Handle("0", func(_ riffkey.Match) {
		ed.win().Col = 0
		ed.refresh()
	})
	visualRouter.Handle("$", func(_ riffkey.Match) {
		ed.win().Col = max(0, len(ed.buf().Lines[ed.win().Cursor])-1)
		ed.refresh()
	})

	visualRouter.Handle("w", func(m riffkey.Match) {
		for range m.Count {
			ed.wordForward()
		}
		ed.refresh()
	})
	visualRouter.Handle("b", func(m riffkey.Match) {
		for range m.Count {
			ed.wordBackward()
		}
		ed.refresh()
	})
	visualRouter.Handle("e", func(m riffkey.Match) {
		for range m.Count {
			ed.wordEnd()
		}
		ed.refresh()
	})

	// o/O swaps cursor to other end of selection
	visualRouter.Handle("o", func(_ riffkey.Match) {
		ed.win().Cursor, ed.win().visualStart = ed.win().visualStart, ed.win().Cursor
		ed.win().Col, ed.win().visualStartCol = ed.win().visualStartCol, ed.win().Col
		ed.refresh()
	})
	visualRouter.Handle("O", func(_ riffkey.Match) {
		ed.win().Cursor, ed.win().visualStart = ed.win().visualStart, ed.win().Cursor
		ed.win().Col, ed.win().visualStartCol = ed.win().visualStartCol, ed.win().Col
		ed.refresh()
	})

	// d deletes selection
	visualRouter.Handle("d", func(_ riffkey.Match) {
		ed.saveUndo()
		if ed.win().visualLineMode {
			// Delete entire lines
			startLine := min(ed.win().visualStart, ed.win().Cursor)
			endLine := max(ed.win().visualStart, ed.win().Cursor)
			if endLine-startLine+1 >= len(ed.buf().Lines) {
				ed.buf().Lines = []string{""}
				ed.win().Cursor = 0
			} else {
				ed.buf().Lines = append(ed.buf().Lines[:startLine], ed.buf().Lines[endLine+1:]...)
				ed.win().Cursor = min(startLine, len(ed.buf().Lines)-1)
			}
			ed.win().Col = min(ed.win().Col, max(0, len(ed.buf().Lines[ed.win().Cursor])-1))
		} else {
			// Character mode - can span multiple lines
			startLine := min(ed.win().visualStart, ed.win().Cursor)
			endLine := max(ed.win().visualStart, ed.win().Cursor)
			var startCol, endCol int
			if ed.win().visualStart < ed.win().Cursor || (ed.win().visualStart == ed.win().Cursor && ed.win().visualStartCol <= ed.win().Col) {
				startCol = ed.win().visualStartCol
				endCol = ed.win().Col + 1
			} else {
				startCol = ed.win().Col
				endCol = ed.win().visualStartCol + 1
			}
			yankRegister = ed.extractRange(startLine, startCol, endLine, endCol)
			ed.deleteRange(startLine, startCol, endLine, endCol)
		}
		ed.exitVisualMode(app)
	})

	// c changes selection (delete and enter insert mode)
	visualRouter.Handle("c", func(_ riffkey.Match) {
		ed.saveUndo()
		if ed.win().visualLineMode {
			// Change entire lines - delete and insert on first line
			startLine := min(ed.win().visualStart, ed.win().Cursor)
			endLine := max(ed.win().visualStart, ed.win().Cursor)
			yankRegister = ed.extractRange(startLine, 0, endLine, len(ed.buf().Lines[endLine]))
			if endLine-startLine+1 >= len(ed.buf().Lines) {
				ed.buf().Lines = []string{""}
				ed.win().Cursor = 0
			} else {
				ed.buf().Lines = append(ed.buf().Lines[:startLine], ed.buf().Lines[endLine+1:]...)
				ed.win().Cursor = min(startLine, len(ed.buf().Lines)-1)
			}
			// Insert a blank line to type on
			newLines := make([]string, len(ed.buf().Lines)+1)
			copy(newLines[:ed.win().Cursor], ed.buf().Lines[:ed.win().Cursor])
			newLines[ed.win().Cursor] = ""
			copy(newLines[ed.win().Cursor+1:], ed.buf().Lines[ed.win().Cursor:])
			ed.buf().Lines = newLines
			ed.win().Col = 0
		} else {
			// Character mode - delete selection
			startLine := min(ed.win().visualStart, ed.win().Cursor)
			endLine := max(ed.win().visualStart, ed.win().Cursor)
			var startCol, endCol int
			if ed.win().visualStart < ed.win().Cursor || (ed.win().visualStart == ed.win().Cursor && ed.win().visualStartCol <= ed.win().Col) {
				startCol = ed.win().visualStartCol
				endCol = ed.win().Col + 1
			} else {
				startCol = ed.win().Col
				endCol = ed.win().visualStartCol + 1
			}
			yankRegister = ed.extractRange(startLine, startCol, endLine, endCol)
			ed.deleteRange(startLine, startCol, endLine, endCol)
		}
		ed.Mode = "NORMAL" // Clear visual mode state
		app.Pop()          // Pop visual router
		ed.updateDisplay()
		ed.enterInsertMode(app)
	})

	// y yanks selection
	visualRouter.Handle("y", func(_ riffkey.Match) {
		if ed.win().visualLineMode {
			startLine := min(ed.win().visualStart, ed.win().Cursor)
			endLine := max(ed.win().visualStart, ed.win().Cursor)
			var yanked string
			for i := startLine; i <= endLine; i++ {
				yanked += ed.buf().Lines[i]
				if i < endLine {
					yanked += "\n"
				}
			}
			yankRegister = yanked
			ed.StatusLine = fmt.Sprintf("Yanked %d lines", endLine-startLine+1)
		} else {
			// Character mode - can span multiple lines
			startLine := min(ed.win().visualStart, ed.win().Cursor)
			endLine := max(ed.win().visualStart, ed.win().Cursor)
			var startCol, endCol int
			if ed.win().visualStart < ed.win().Cursor || (ed.win().visualStart == ed.win().Cursor && ed.win().visualStartCol <= ed.win().Col) {
				startCol = ed.win().visualStartCol
				endCol = ed.win().Col + 1
			} else {
				startCol = ed.win().Col
				endCol = ed.win().visualStartCol + 1
			}
			yankRegister = ed.extractRange(startLine, startCol, endLine, endCol)
			ed.StatusLine = fmt.Sprintf("Yanked %d chars", len(yankRegister))
		}
		ed.exitVisualMode(app)
	})

	// Text objects expand selection in visual mode
	visualTextObjects := []struct {
		key string
		fn  MultiLineTextObjectFunc
	}{
		{"ip", toInnerParagraphML},
		{"ap", toAParagraphML},
		{"is", toInnerSentenceML},
		{"as", toASentenceML},
		{"i(", toInnerParenML},
		{"a(", toAParenML},
		{"i)", toInnerParenML},
		{"a)", toAParenML},
		{"i[", toInnerBracketML},
		{"a[", toABracketML},
		{"i]", toInnerBracketML},
		{"a]", toABracketML},
		{"i{", toInnerBraceML},
		{"a{", toABraceML},
		{"i}", toInnerBraceML},
		{"a}", toABraceML},
		{"i<", toInnerAngleML},
		{"a<", toAAngleML},
		{"i>", toInnerAngleML},
		{"a>", toAAngleML},
	}

	for _, obj := range visualTextObjects {
		objFn := obj.fn // capture for closure
		visualRouter.Handle(obj.key, func(_ riffkey.Match) {
			startLine, startCol, endLine, endCol := objFn(ed)
			if startLine >= 0 {
				// Expand visual selection to cover the text object
				ed.win().visualStart = startLine
				ed.win().visualStartCol = startCol
				ed.win().Cursor = endLine
				ed.win().Col = max(0, endCol-1) // endCol is exclusive, cursor should be on last char
				ed.win().visualLineMode = false
				ed.updateDisplay()
				ed.updateCursor()
			}
		})
	}

	// Word text objects (single-line)
	visualWordObjects := []struct {
		key string
		fn  TextObjectFunc
	}{
		{"iw", toInnerWord},
		{"aw", toAWord},
		{"iW", toInnerWORD},
		{"aW", toAWORD},
	}

	for _, obj := range visualWordObjects {
		objFn := obj.fn
		visualRouter.Handle(obj.key, func(_ riffkey.Match) {
			line := ed.buf().Lines[ed.win().Cursor]
			start, end := objFn(line, ed.win().Col)
			if start < end {
				ed.win().visualStart = ed.win().Cursor
				ed.win().visualStartCol = start
				ed.win().Col = end - 1
				ed.win().visualLineMode = false
				ed.updateDisplay()
				ed.updateCursor()
			}
		})
	}

	// Escape exits visual mode
	visualRouter.Handle("<Esc>", func(_ riffkey.Match) {
		ed.exitVisualMode(app)
	})

	app.Push(visualRouter)
}

func (ed *Editor) exitVisualMode(app *tui.App) {
	ed.Mode = "NORMAL"
	ed.StatusLine = "hjkl:move  w/b/e:word  ciw/daw/yi\":text-obj  p:paste  q:quit"
	ed.updateDisplay()
	app.Pop()
}

// Command line mode (for :, /, ?)
func (ed *Editor) enterCommandMode(app *tui.App, prompt string) {
	ed.cmdLineActive = true
	ed.cmdLinePrompt = prompt
	ed.cmdLineInput = ""
	ed.StatusLine = prompt
	ed.updateDisplay()

	// Move cursor to command line
	app.ShowCursor(tui.CursorBar)
	size := app.Size()
	app.SetCursor(1, size.Height-1) // After the prompt

	// Create command line router
	cmdRouter := riffkey.NewRouter().Name("cmdline")

	// Enter executes the command
	cmdRouter.Handle("<CR>", func(_ riffkey.Match) {
		cmd := ed.cmdLineInput
		ed.exitCommandMode(app)
		ed.executeCommand(app, ed.cmdLinePrompt, cmd)
	})

	// Escape cancels
	cmdRouter.Handle("<Esc>", func(_ riffkey.Match) {
		ed.exitCommandMode(app)
	})

	// Backspace deletes last char
	cmdRouter.Handle("<BS>", func(_ riffkey.Match) {
		if len(ed.cmdLineInput) > 0 {
			ed.cmdLineInput = ed.cmdLineInput[:len(ed.cmdLineInput)-1]
			ed.StatusLine = ed.cmdLinePrompt + ed.cmdLineInput
			ed.updateDisplay()
			// Update cursor position
			app.SetCursor(1+len(ed.cmdLineInput), size.Height-1)
		}
	})

	// Handle regular character input
	cmdRouter.HandleUnmatched(func(k riffkey.Key) bool {
		if k.Rune != 0 && k.Mod == riffkey.ModNone {
			ed.cmdLineInput += string(k.Rune)
			ed.StatusLine = ed.cmdLinePrompt + ed.cmdLineInput
			ed.updateDisplay()
			// Update cursor position
			size := app.Size()
			app.SetCursor(1+len(ed.cmdLineInput), size.Height-1)
			return true
		}
		return false
	})

	app.Push(cmdRouter)
}

func (ed *Editor) exitCommandMode(app *tui.App) {
	ed.cmdLineActive = false
	ed.StatusLine = ""
	app.ShowCursor(tui.CursorBlock)
	ed.updateDisplay()
	ed.updateCursor()
	app.Pop()
}

func (ed *Editor) executeCommand(app *tui.App, prompt, cmd string) {
	switch prompt {
	case ":":
		ed.executeColonCommand(app, cmd)
	case "/":
		ed.executeSearch(cmd, 1) // forward
	case "?":
		ed.executeSearch(cmd, -1) // backward
	}
}

func (ed *Editor) executeColonCommand(app *tui.App, cmd string) {
	switch cmd {
	case "q", "quit":
		if !ed.root.IsLeaf() {
			// Close current window if there are multiple
			ed.closeWindow()
		} else {
			app.Stop()
		}
	case "qa", "qall":
		app.Stop()
	case "w", "write":
		ed.StatusLine = "E37: No write since last change (use :w! to override)"
		ed.updateDisplay()
	case "wq", "x":
		ed.StatusLine = "E37: No write since last change (use :wq! to override)"
		ed.updateDisplay()
	case "sp", "split":
		// Horizontal split
		ed.splitHorizontal()
	case "vs", "vsplit":
		// Vertical split
		ed.splitVertical()
	case "close":
		ed.closeWindow()
	case "only", "on":
		ed.closeOtherWindows()
	case "noh", "nohlsearch":
		// Clear search highlighting
		ed.searchPattern = ""
		ed.updateDisplay()
	default:
		// Try to parse as line number
		if lineNum := 0; len(cmd) > 0 {
			for _, c := range cmd {
				if c >= '0' && c <= '9' {
					lineNum = lineNum*10 + int(c-'0')
				} else {
					lineNum = -1
					break
				}
			}
			if lineNum > 0 && lineNum <= len(ed.buf().Lines) {
				ed.win().Cursor = lineNum - 1
				ed.win().Col = 0
				ed.updateDisplay()
				ed.updateCursor()
				return
			}
		}
		ed.StatusLine = fmt.Sprintf("E492: Not an editor command: %s", cmd)
		ed.updateDisplay()
	}
}

func (ed *Editor) executeSearch(pattern string, direction int) {
	if pattern == "" {
		// Use last search pattern
		pattern = ed.lastSearch
	}
	if pattern == "" {
		ed.StatusLine = "E35: No previous regular expression"
		ed.updateDisplay()
		return
	}

	ed.lastSearch = pattern
	ed.searchPattern = pattern
	ed.searchDirection = direction

	// Search from current position
	ed.searchNext(direction)
}

func (ed *Editor) searchNext(direction int) {
	if ed.searchPattern == "" {
		ed.StatusLine = "E35: No previous regular expression"
		ed.updateDisplay()
		return
	}

	// Actual direction considering original search direction
	actualDir := ed.searchDirection * direction

	// Start search from next/prev position
	startLine := ed.win().Cursor
	startCol := ed.win().Col + 1
	if actualDir < 0 {
		startCol = ed.win().Col - 1
	}

	// Search through all lines
	for i := 0; i < len(ed.buf().Lines); i++ {
		lineIdx := startLine
		if actualDir > 0 {
			lineIdx = (startLine + i) % len(ed.buf().Lines)
		} else {
			lineIdx = (startLine - i + len(ed.buf().Lines)) % len(ed.buf().Lines)
		}

		line := ed.buf().Lines[lineIdx]
		col := -1

		if i == 0 {
			// First line: search from startCol
			if actualDir > 0 {
				col = strings.Index(line[min(startCol, len(line)):], ed.searchPattern)
				if col >= 0 {
					col += min(startCol, len(line))
				}
			} else {
				// Search backward from startCol
				searchPart := line[:max(0, startCol)]
				col = strings.LastIndex(searchPart, ed.searchPattern)
			}
		} else {
			// Other lines: search whole line
			if actualDir > 0 {
				col = strings.Index(line, ed.searchPattern)
			} else {
				col = strings.LastIndex(line, ed.searchPattern)
			}
		}

		if col >= 0 {
			ed.win().Cursor = lineIdx
			ed.win().Col = col
			ed.StatusLine = fmt.Sprintf("/%s", ed.searchPattern)
			ed.updateDisplay()
			ed.updateCursor()
			return
		}
	}

	ed.StatusLine = fmt.Sprintf("E486: Pattern not found: %s", ed.searchPattern)
	ed.updateDisplay()
}

// f/F/t/T implementation - find character on line
func registerFindChar(app *tui.App, ed *Editor) {
	for _, findType := range []struct {
		key     string
		forward bool
		till    bool
	}{
		{"f", true, false},
		{"F", false, false},
		{"t", true, true},
		{"T", false, true},
	} {
		key := findType.key
		forward := findType.forward
		till := findType.till

		app.Handle(key, func(_ riffkey.Match) {
			// Next key press will be the target character
			findRouter := riffkey.NewRouter().Name("find-char")
			findRouter.HandleUnmatched(func(k riffkey.Key) bool {
				if k.Rune != 0 && k.Mod == riffkey.ModNone {
					ed.lastFindChar = k.Rune
					ed.lastFindDir = 1
					if !forward {
						ed.lastFindDir = -1
					}
					ed.lastFindTill = till
					ed.doFindChar(forward, till, k.Rune)
				}
				app.Pop()
				return true
			})
			findRouter.Handle("<Esc>", func(_ riffkey.Match) {
				app.Pop()
			})
			app.Push(findRouter)
		})
	}

	// ; repeats last f/F/t/T
	app.Handle(";", func(_ riffkey.Match) {
		if ed.lastFindChar != 0 {
			ed.doFindChar(ed.lastFindDir == 1, ed.lastFindTill, ed.lastFindChar)
		}
	})

	// , repeats last f/F/t/T in opposite direction
	app.Handle(",", func(_ riffkey.Match) {
		if ed.lastFindChar != 0 {
			ed.doFindChar(ed.lastFindDir != 1, ed.lastFindTill, ed.lastFindChar)
		}
	})
}

func (ed *Editor) doFindChar(forward, till bool, ch rune) {
	line := ed.buf().Lines[ed.win().Cursor]
	if forward {
		for i := ed.win().Col + 1; i < len(line); i++ {
			if rune(line[i]) == ch {
				if till {
					ed.win().Col = i - 1
				} else {
					ed.win().Col = i
				}
				ed.updateCursor()
				return
			}
		}
	} else {
		for i := ed.win().Col - 1; i >= 0; i-- {
			if rune(line[i]) == ch {
				if till {
					ed.win().Col = i + 1
				} else {
					ed.win().Col = i
				}
				ed.updateCursor()
				return
			}
		}
	}
}

// loadFile reads a file and returns lines, or nil on error
func loadFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	// Remove trailing empty line if present (from final newline)
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	// Expand tabs to spaces (4 spaces per tab)
	for i, line := range lines {
		lines[i] = strings.ReplaceAll(line, "\t", "    ")
	}
	return lines
}
