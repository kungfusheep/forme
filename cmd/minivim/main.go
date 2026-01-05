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

	"riffkey"
	"tui"
)

// Layout constants
const (
	headerRows = 0 // Content starts at row 0 now
	footerRows = 2 // Status bar + message line
)

type Editor struct {
	Lines      []string
	Cursor     int    // current line
	Col        int    // cursor column
	Mode       string // "NORMAL", "INSERT", or "VISUAL"
	StatusLine string // command/message line (bottom)
	FileName   string // displayed in status bar

	app *tui.App // reference for cursor control

	// Viewport scrolling
	topLine        int // first visible line
	viewportHeight int // number of visible content lines

	// Undo/redo stacks
	undoStack []EditorState
	redoStack []EditorState

	// Visual mode
	visualStart    int  // line where visual selection started
	visualStartCol int  // column where visual selection started
	visualLineMode bool // V (line mode) vs v (char mode)

	// Search
	searchPattern   string
	searchDirection int // 1 forward, -1 backward
	lastSearch      string

	// f/F/t/T last char for ; and ,
	lastFindChar rune
	lastFindDir  int  // 1=f/t forward, -1=F/T backward
	lastFindTill bool // true for t/T

	// Command line mode
	cmdLineActive bool   // currently in command mode
	cmdLinePrompt string // ":" or "/" or "?"
	cmdLineInput  string // current input

	// Layer-based rendering (imperative, efficient partial updates)
	contentLayer *tui.Layer // pre-rendered content buffer
	lineNumWidth int        // cached width of line number column
	StatusBar    []tui.Span // vim-style status bar (inverse, full width)
}

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

	ed := &Editor{
		Lines:      lines,
		Mode:       "NORMAL",
		FileName:   fileName,
		StatusLine: "", // empty initially, used for messages
	}

	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}
	ed.app = app

	// Initialize viewport and layer
	size := app.Size()
	ed.viewportHeight = max(1, size.Height-headerRows-footerRows)
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
	app.Handle("gg", func(_ riffkey.Match) { ed.moveTo(0, ed.Col) })
	app.Handle("G", func(_ riffkey.Match) { ed.moveTo(len(ed.Lines)-1, ed.Col) })
	app.Handle("0", func(_ riffkey.Match) { ed.moveToCol(0) })
	app.Handle("$", func(_ riffkey.Match) { ed.moveToCol(len(ed.Lines[ed.Cursor])) })

	app.Handle("w", func(m riffkey.Match) {
		oldLine := ed.Cursor
		for range m.Count {
			ed.wordForward()
		}
		ed.ensureCursorVisible()
		ed.updateCursorHighlight(oldLine)
		ed.updateCursor()
	})

	app.Handle("b", func(m riffkey.Match) {
		oldLine := ed.Cursor
		for range m.Count {
			ed.wordBackward()
		}
		ed.ensureCursorVisible()
		ed.updateCursorHighlight(oldLine)
		ed.updateCursor()
	})

	app.Handle("e", func(m riffkey.Match) {
		oldLine := ed.Cursor
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
		if len(ed.Lines[ed.Cursor]) > 0 {
			ed.Col++
		}
		ed.enterInsertMode(app)
	})

	app.Handle("A", func(_ riffkey.Match) {
		ed.Col = len(ed.Lines[ed.Cursor])
		ed.enterInsertMode(app)
	})

	app.Handle("I", func(_ riffkey.Match) {
		ed.Col = 0
		ed.enterInsertMode(app)
	})

	app.Handle("o", func(_ riffkey.Match) {
		// Insert new line below
		ed.Cursor++
		newLines := make([]string, len(ed.Lines)+1)
		copy(newLines[:ed.Cursor], ed.Lines[:ed.Cursor])
		newLines[ed.Cursor] = ""
		copy(newLines[ed.Cursor+1:], ed.Lines[ed.Cursor:])
		ed.Lines = newLines
		ed.Col = 0
		ed.updateDisplay()
		ed.enterInsertMode(app)
	})

	app.Handle("O", func(_ riffkey.Match) {
		// Insert new line above
		newLines := make([]string, len(ed.Lines)+1)
		copy(newLines[:ed.Cursor], ed.Lines[:ed.Cursor])
		newLines[ed.Cursor] = ""
		copy(newLines[ed.Cursor+1:], ed.Lines[ed.Cursor:])
		ed.Lines = newLines
		ed.Col = 0
		ed.updateDisplay()
		ed.enterInsertMode(app)
	})

	app.Handle("dd", func(m riffkey.Match) {
		ed.saveUndo()
		for i := 0; i < m.Count; i++ {
			if len(ed.Lines) > 1 {
				ed.Lines = append(ed.Lines[:ed.Cursor], ed.Lines[ed.Cursor+1:]...)
				if ed.Cursor >= len(ed.Lines) {
					ed.Cursor = len(ed.Lines) - 1
				}
			} else {
				ed.Lines[0] = ""
				break
			}
		}
		ed.Col = min(ed.Col, max(0, len(ed.Lines[ed.Cursor])-1))
		ed.updateDisplay()
		ed.updateCursor()
	})

	app.Handle("x", func(m riffkey.Match) {
		ed.saveUndo()
		// Delete character(s) under cursor
		for i := 0; i < m.Count; i++ {
			line := ed.Lines[ed.Cursor]
			if len(line) > 0 && ed.Col < len(line) {
				ed.Lines[ed.Cursor] = line[:ed.Col] + line[ed.Col+1:]
			}
		}
		if ed.Col >= len(ed.Lines[ed.Cursor]) && ed.Col > 0 {
			ed.Col = max(0, len(ed.Lines[ed.Cursor])-1)
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
			line := ed.Lines[ed.Cursor]
			pos := min(ed.Col+1, len(line))
			ed.Lines[ed.Cursor] = line[:pos] + yankRegister + line[pos:]
			ed.Col = pos + len(yankRegister) - 1
			ed.updateDisplay()
			ed.updateCursor()
		}
	})

	app.Handle("P", func(_ riffkey.Match) {
		if yankRegister != "" {
			line := ed.Lines[ed.Cursor]
			ed.Lines[ed.Cursor] = line[:ed.Col] + yankRegister + line[ed.Col:]
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
		half := ed.viewportHeight / 2
		ed.Cursor = min(ed.Cursor+half, len(ed.Lines)-1)
		ed.Col = min(ed.Col, len(ed.Lines[ed.Cursor]))
		ed.updateDisplay()
		ed.updateCursor()
	})

	app.Handle("<C-u>", func(_ riffkey.Match) {
		// Half page up
		ed.ensureCursorVisible()
		half := ed.viewportHeight / 2
		ed.Cursor = max(ed.Cursor-half, 0)
		ed.Col = min(ed.Col, len(ed.Lines[ed.Cursor]))
		ed.updateDisplay()
		ed.updateCursor()
	})

	app.Handle("<C-e>", func(_ riffkey.Match) {
		// Scroll down one line (keep cursor in place if possible)
		ed.ensureCursorVisible()
		if ed.topLine < len(ed.Lines)-ed.viewportHeight {
			ed.topLine++
			if ed.Cursor < ed.topLine {
				ed.Cursor = ed.topLine
				ed.Col = min(ed.Col, len(ed.Lines[ed.Cursor]))
			}
			ed.updateDisplay()
			ed.updateCursor()
		}
	})

	app.Handle("<C-y>", func(_ riffkey.Match) {
		// Scroll up one line (keep cursor in place if possible)
		if ed.topLine > 0 {
			ed.topLine--
			if ed.Cursor >= ed.topLine+ed.viewportHeight {
				ed.Cursor = ed.topLine + ed.viewportHeight - 1
				ed.Col = min(ed.Col, len(ed.Lines[ed.Cursor]))
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
		if ed.Cursor < len(ed.Lines)-1 {
			// Join current line with next
			ed.Lines[ed.Cursor] += " " + ed.Lines[ed.Cursor+1]
			ed.Lines = append(ed.Lines[:ed.Cursor+1], ed.Lines[ed.Cursor+2:]...)
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
				line := ed.Lines[ed.Cursor]
				if ed.Col < len(line) {
					ed.Lines[ed.Cursor] = line[:ed.Col] + string(k.Rune) + line[ed.Col+1:]
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
			line := ed.Lines[ed.Cursor]
			ed.Lines[ed.Cursor] = line[:ed.Col] + yankRegister + line[ed.Col:]
			ed.Col += len(yankRegister)
			ed.updateDisplay()
			ed.updateCursor()
		}
	})

	// ~ toggle case
	app.Handle("~", func(_ riffkey.Match) {
		ed.saveUndo()
		line := ed.Lines[ed.Cursor]
		if ed.Col < len(line) {
			c := line[ed.Col]
			if c >= 'a' && c <= 'z' {
				c = c - 'a' + 'A'
			} else if c >= 'A' && c <= 'Z' {
				c = c - 'A' + 'a'
			}
			ed.Lines[ed.Cursor] = line[:ed.Col] + string(c) + line[ed.Col+1:]
			if ed.Col < len(line)-1 {
				ed.Col++
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

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func (ed *Editor) enterInsertMode(app *tui.App) {
	ed.Mode = "INSERT"
	ed.Col = min(ed.Col, len(ed.Lines[ed.Cursor]))
	ed.StatusLine = "-- INSERT --  Esc:normal  Enter:newline  Ctrl+W:delete word"
	ed.updateDisplay()

	// Switch to bar cursor for insert mode
	app.ShowCursor(tui.CursorBar)
	ed.updateCursor()

	// Create insert mode router
	insertRouter := riffkey.NewRouter().Name("insert")

	// TextHandler with OnChange callback for live updates
	th := riffkey.NewTextHandler(&ed.Lines[ed.Cursor], &ed.Col)
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
		line := ed.Lines[ed.Cursor]
		before := line[:ed.Col]
		after := line[ed.Col:]
		ed.Lines[ed.Cursor] = before

		// Insert new line after
		newLines := make([]string, len(ed.Lines)+1)
		copy(newLines[:ed.Cursor+1], ed.Lines[:ed.Cursor+1])
		newLines[ed.Cursor+1] = after
		copy(newLines[ed.Cursor+2:], ed.Lines[ed.Cursor+1:])
		ed.Lines = newLines
		ed.Cursor++
		ed.Col = 0

		// Rebind TextHandler to new line
		th.Value = &ed.Lines[ed.Cursor]
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
	if ed.Col > 0 && ed.Col >= len(ed.Lines[ed.Cursor]) {
		ed.Col = max(0, len(ed.Lines[ed.Cursor])-1)
	}

	// Switch back to block cursor for normal mode
	app.ShowCursor(tui.CursorBlock)

	ed.updateDisplay()
	ed.updateCursor()
	app.Pop() // Back to normal mode router
}

func (ed *Editor) updateCursor() {
	// Calculate screen position relative to viewport
	// Row: headerRows + (cursor line - topLine)
	screenY := headerRows + (ed.Cursor - ed.topLine)

	// Column: offset by line number width (cached from updateDisplay)
	screenX := ed.lineNumWidth + ed.Col

	ed.app.SetCursor(screenX, screenY)
}

// refresh does a full re-render - use for content changes or visual mode.
// For simple cursor movement, use updateCursorHighlight() instead.
func (ed *Editor) refresh() {
	ed.updateDisplay()
	ed.updateCursor()
}

// moveTo sets cursor to absolute position, clamping to valid range
func (ed *Editor) moveTo(line, col int) {
	oldLine := ed.Cursor
	ed.Cursor = max(0, min(line, len(ed.Lines)-1))
	ed.Col = max(0, min(col, len(ed.Lines[ed.Cursor])-1))
	ed.ensureCursorVisible()
	ed.updateCursorHighlight(oldLine)
	ed.updateCursor()
}

// moveToCol sets column only, clamping to valid range
func (ed *Editor) moveToCol(col int) {
	ed.Col = max(0, min(col, len(ed.Lines[ed.Cursor])-1))
	ed.updateCursor()
}

// moveDown moves cursor down by count lines
func (ed *Editor) moveDown(count int) {
	oldLine := ed.Cursor
	ed.Cursor = min(ed.Cursor+count, len(ed.Lines)-1)
	ed.Col = min(ed.Col, len(ed.Lines[ed.Cursor]))
	ed.ensureCursorVisible()
	ed.updateCursorHighlight(oldLine)
	ed.updateCursor()
}

// moveUp moves cursor up by count lines
func (ed *Editor) moveUp(count int) {
	oldLine := ed.Cursor
	ed.Cursor = max(ed.Cursor-count, 0)
	ed.Col = min(ed.Col, len(ed.Lines[ed.Cursor]))
	ed.ensureCursorVisible()
	ed.updateCursorHighlight(oldLine)
	ed.updateCursor()
}

// moveLeft moves cursor left by count columns
func (ed *Editor) moveLeft(count int) {
	ed.Col = max(0, ed.Col-count)
	ed.updateCursor()
}

// moveRight moves cursor right by count columns
func (ed *Editor) moveRight(count int) {
	ed.Col = min(ed.Col+count, len(ed.Lines[ed.Cursor])-1)
	ed.updateCursor()
}

func (ed *Editor) ensureCursorVisible() {
	// Scroll viewport if cursor is outside visible area
	if ed.viewportHeight == 0 {
		// Get viewport height from screen (minus footer for status bar + message line)
		size := ed.app.Size()
		ed.viewportHeight = max(1, size.Height-headerRows-footerRows)
	}

	// Scroll up if cursor above viewport
	if ed.Cursor < ed.topLine {
		ed.topLine = ed.Cursor
	}

	// Scroll down if cursor below viewport
	if ed.Cursor >= ed.topLine+ed.viewportHeight {
		ed.topLine = ed.Cursor - ed.viewportHeight + 1
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
	// Get terminal width for padding
	size := ed.app.Size()
	width := size.Width

	// Left side: filename
	left := " " + ed.FileName

	// Right side: line:col percentage
	percentage := 0
	if len(ed.Lines) > 0 {
		percentage = (ed.Cursor + 1) * 100 / len(ed.Lines)
	}
	right := fmt.Sprintf(" %d,%d  %d%% ", ed.Cursor+1, ed.Col+1, percentage)

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
	ed.StatusBar = []tui.Span{
		{Text: left + middle + right, Style: statusBarStyle},
	}
}

func (ed *Editor) updateDisplay() {
	// Ensure viewport height is set and cursor is visible
	ed.ensureCursorVisible()

	// Build vim-style status bar
	ed.updateStatusBar()

	// Render all lines to the layer
	ed.renderAllLines()
}

// initLayer creates and sizes the content layer
func (ed *Editor) initLayer(width int) {
	ed.contentLayer = tui.NewLayer()
	// Layer holds ALL lines plus some buffer for scrolling
	ed.contentLayer.EnsureSize(width, len(ed.Lines)+ed.viewportHeight)
	ed.renderAllLines()
}

// renderAllLines renders every line to the layer (used on init or major changes)
func (ed *Editor) renderAllLines() {
	if ed.contentLayer == nil || ed.contentLayer.Buffer() == nil {
		return
	}

	// Calculate line number width based on total lines
	maxLineNum := len(ed.Lines)
	ed.lineNumWidth = len(fmt.Sprintf("%d", maxLineNum)) + 1

	// Render each line
	for i := 0; i < len(ed.Lines)+ed.viewportHeight; i++ {
		ed.renderLineToLayer(i)
	}

	// Set scroll position
	ed.contentLayer.ScrollTo(ed.topLine)
}

// renderLineToLayer renders a single line to the layer buffer
func (ed *Editor) renderLineToLayer(lineIdx int) {
	if ed.contentLayer == nil || ed.contentLayer.Buffer() == nil {
		return
	}

	lineNumFmt := fmt.Sprintf("%%%dd ", ed.lineNumWidth-1)
	tildeFmt := fmt.Sprintf("%%%ds ", ed.lineNumWidth-1)

	var spans []tui.Span

	if lineIdx < len(ed.Lines) {
		// Content line
		line := ed.Lines[lineIdx]
		lineNum := fmt.Sprintf(lineNumFmt, lineIdx+1)
		isCursorLine := lineIdx == ed.Cursor

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

	ed.contentLayer.SetLine(lineIdx, spans)
}

// updateLine efficiently updates just the changed lines (for cursor movement)
func (ed *Editor) updateLine(lineIdx int) {
	ed.renderLineToLayer(lineIdx)
}

// updateCursorHighlight efficiently updates highlight when cursor moves between lines
// Returns the old cursor line index for callers that need it
func (ed *Editor) updateCursorHighlight(oldLine int) {
	if ed.contentLayer == nil {
		return
	}

	// Only update if cursor actually moved to a different line
	if oldLine != ed.Cursor && oldLine >= 0 && oldLine < len(ed.Lines) {
		ed.renderLineToLayer(oldLine) // Remove yellow from old line
	}
	ed.renderLineToLayer(ed.Cursor) // Add yellow to new line

	// Sync layer scroll position
	ed.contentLayer.ScrollTo(ed.topLine)
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

	if ed.visualLineMode {
		// Line mode: entire line is selected or not
		if ed.isLineSelected(lineIdx) {
			return []tui.Span{{Text: line, Style: inverseStyle}}
		}
		return []tui.Span{{Text: line, Style: normalStyle}}
	}

	// Character mode: need to calculate per-character selection
	// Simplified: only works for single-line selection for now
	if lineIdx != ed.Cursor && lineIdx != ed.visualStart {
		// Line is either fully selected or not (if between start and cursor)
		if ed.isLineSelected(lineIdx) {
			return []tui.Span{{Text: line, Style: inverseStyle}}
		}
		return []tui.Span{{Text: line, Style: normalStyle}}
	}

	// This is the line with the cursor or visual start - split into spans
	startCol := min(ed.visualStartCol, ed.Col)
	endCol := max(ed.visualStartCol, ed.Col) + 1

	if lineIdx != ed.Cursor || lineIdx != ed.visualStart {
		// Multi-line selection - this line is start or end
		if lineIdx == min(ed.visualStart, ed.Cursor) {
			// First line - select from startCol to end
			col := ed.visualStartCol
			if lineIdx == ed.Cursor {
				col = ed.Col
			}
			if lineIdx != ed.visualStart {
				col = ed.Col
			} else {
				col = ed.visualStartCol
			}
			// Simplified: highlight from col to end
			startCol = min(col, len(line))
			endCol = len(line)
		} else {
			// Last line - select from start to col
			col := ed.Col
			if lineIdx == ed.visualStart {
				col = ed.visualStartCol
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
	minLine := min(ed.visualStart, ed.Cursor)
	maxLine := max(ed.visualStart, ed.Cursor)
	return lineIdx >= minLine && lineIdx <= maxLine
}

func buildView(ed *Editor) any {
	return tui.Col{Children: []any{
		// Content area - imperative layer, efficiently updated
		tui.LayerView{Layer: ed.contentLayer, Height: int16(ed.viewportHeight)},

		// Vim-style status bar (inverse video, shows filename and position)
		tui.RichText{Spans: &ed.StatusBar},

		// Command/message line (shows mode indicator and messages)
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
				line := ed.Lines[ed.Cursor]
				start, end := objFn(line, ed.Col)
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
	line := ed.Lines[ed.Cursor]
	ed.Lines[ed.Cursor] = line[:start] + line[end:]
	ed.Col = start
	if ed.Col >= len(ed.Lines[ed.Cursor]) && ed.Col > 0 {
		ed.Col = max(0, len(ed.Lines[ed.Cursor])-1)
	}
	ed.updateDisplay()
	ed.updateCursor()
}

func opChange(ed *Editor, app *tui.App, start, end int) {
	ed.saveUndo()
	line := ed.Lines[ed.Cursor]
	ed.Lines[ed.Cursor] = line[:start] + line[end:]
	ed.Col = start
	ed.updateDisplay()
	ed.enterInsertMode(app)
}

var yankRegister string

func opYank(ed *Editor, app *tui.App, start, end int) {
	line := ed.Lines[ed.Cursor]
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
			endLine := min(ed.Cursor+count, len(ed.Lines)-1)
			return ed.Cursor, 0, endLine, len(ed.Lines[endLine])
		}},
		{"k", func(ed *Editor, count int) (int, int, int, int) {
			startLine := max(ed.Cursor-count, 0)
			return startLine, 0, ed.Cursor, len(ed.Lines[ed.Cursor])
		}},
		{"gg", func(ed *Editor, count int) (int, int, int, int) {
			return 0, 0, ed.Cursor, len(ed.Lines[ed.Cursor])
		}},
		{"G", func(ed *Editor, count int) (int, int, int, int) {
			endLine := len(ed.Lines) - 1
			return ed.Cursor, 0, endLine, len(ed.Lines[endLine])
		}},
		// Characterwise motions
		{"w", func(ed *Editor, count int) (int, int, int, int) {
			startLine, startCol := ed.Cursor, ed.Col
			for range count {
				ed.wordForward()
			}
			endLine, endCol := ed.Cursor, ed.Col
			ed.Cursor, ed.Col = startLine, startCol
			return startLine, startCol, endLine, endCol
		}},
		{"b", func(ed *Editor, count int) (int, int, int, int) {
			endLine, endCol := ed.Cursor, ed.Col
			for range count {
				ed.wordBackward()
			}
			startLine, startCol := ed.Cursor, ed.Col
			ed.Cursor, ed.Col = endLine, endCol
			return startLine, startCol, endLine, endCol
		}},
		{"e", func(ed *Editor, count int) (int, int, int, int) {
			startLine, startCol := ed.Cursor, ed.Col
			for range count {
				ed.wordEnd()
			}
			endLine, endCol := ed.Cursor, ed.Col+1
			ed.Cursor, ed.Col = startLine, startCol
			return startLine, startCol, endLine, endCol
		}},
		{"$", func(ed *Editor, count int) (int, int, int, int) {
			return ed.Cursor, ed.Col, ed.Cursor, len(ed.Lines[ed.Cursor])
		}},
		{"0", func(ed *Editor, count int) (int, int, int, int) {
			return ed.Cursor, 0, ed.Cursor, ed.Col
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
		ed.Lines[ed.Cursor] = ""
		ed.Col = 0
		ed.updateDisplay()
		ed.enterInsertMode(app)
	})

	// S - same as cc
	app.Handle("S", func(_ riffkey.Match) {
		ed.saveUndo()
		ed.Lines[ed.Cursor] = ""
		ed.Col = 0
		ed.updateDisplay()
		ed.enterInsertMode(app)
	})

	// yy - yank whole line
	app.Handle("yy", func(_ riffkey.Match) {
		yankRegister = ed.Lines[ed.Cursor]
		ed.StatusLine = fmt.Sprintf("Yanked: %q", yankRegister)
		ed.updateDisplay()
	})

	// Y - same as yy
	app.Handle("Y", func(_ riffkey.Match) {
		yankRegister = ed.Lines[ed.Cursor]
		ed.StatusLine = fmt.Sprintf("Yanked: %q", yankRegister)
		ed.updateDisplay()
	})
}

// findInnerParagraph returns the line range of the current paragraph (non-blank lines)
func (ed *Editor) findInnerParagraph() (startLine, endLine int) {
	// If on a blank line, return just this line
	if strings.TrimSpace(ed.Lines[ed.Cursor]) == "" {
		return ed.Cursor, ed.Cursor
	}

	// Find start of paragraph (first non-blank line going backward)
	startLine = ed.Cursor
	for startLine > 0 && strings.TrimSpace(ed.Lines[startLine-1]) != "" {
		startLine--
	}

	// Find end of paragraph (last non-blank line going forward)
	endLine = ed.Cursor
	for endLine < len(ed.Lines)-1 && strings.TrimSpace(ed.Lines[endLine+1]) != "" {
		endLine++
	}

	return startLine, endLine
}

// findAParagraph returns the line range including trailing blank lines
func (ed *Editor) findAParagraph() (startLine, endLine int) {
	startLine, endLine = ed.findInnerParagraph()

	// Include trailing blank lines
	for endLine < len(ed.Lines)-1 && strings.TrimSpace(ed.Lines[endLine+1]) == "" {
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
		line := ed.Lines[startLine]
		endCol = min(endCol, len(line))
		startCol = min(startCol, len(line))
		return line[startCol:endCol]
	}

	// Multiple lines
	var parts []string

	// First line (from startCol to end)
	if startLine < len(ed.Lines) {
		line := ed.Lines[startLine]
		startCol = min(startCol, len(line))
		parts = append(parts, line[startCol:])
	}

	// Middle lines (full lines)
	for i := startLine + 1; i < endLine && i < len(ed.Lines); i++ {
		parts = append(parts, ed.Lines[i])
	}

	// Last line (from start to endCol)
	if endLine < len(ed.Lines) && endLine > startLine {
		line := ed.Lines[endLine]
		endCol = min(endCol, len(line))
		parts = append(parts, line[:endCol])
	}

	return strings.Join(parts, "\n")
}

// deleteRange deletes text from a multi-line range
func (ed *Editor) deleteRange(startLine, startCol, endLine, endCol int) {
	if startLine == endLine {
		// Same line - simple case
		line := ed.Lines[startLine]
		endCol = min(endCol, len(line))
		startCol = min(startCol, len(line))
		ed.Lines[startLine] = line[:startCol] + line[endCol:]
		ed.Cursor = startLine
		ed.Col = startCol
		return
	}

	// Multiple lines - join first and last line remnants
	firstPart := ""
	if startLine < len(ed.Lines) {
		line := ed.Lines[startLine]
		startCol = min(startCol, len(line))
		firstPart = line[:startCol]
	}

	lastPart := ""
	if endLine < len(ed.Lines) {
		line := ed.Lines[endLine]
		endCol = min(endCol, len(line))
		lastPart = line[endCol:]
	}

	// Create new lines array
	newLines := make([]string, 0, len(ed.Lines)-(endLine-startLine))
	newLines = append(newLines, ed.Lines[:startLine]...)
	newLines = append(newLines, firstPart+lastPart)
	if endLine+1 < len(ed.Lines) {
		newLines = append(newLines, ed.Lines[endLine+1:]...)
	}

	ed.Lines = newLines
	if len(ed.Lines) == 0 {
		ed.Lines = []string{""}
	}
	ed.Cursor = min(startLine, len(ed.Lines)-1)
	ed.Col = startCol
}

// Multi-line text object functions

// toInnerParagraphML returns the range of the inner paragraph
func toInnerParagraphML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	start, end := ed.findInnerParagraph()
	// For paragraph, we delete whole lines (col 0 to end of last line)
	return start, 0, end, len(ed.Lines[end])
}

// toAParagraphML returns the range including trailing blank lines
func toAParagraphML(ed *Editor) (startLine, startCol, endLine, endCol int) {
	start, end := ed.findAParagraph()
	return start, 0, end, len(ed.Lines[end])
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
	startLine = ed.Cursor
	startCol = ed.Col
	endLine = ed.Cursor
	endCol = ed.Col

	// Search backward for sentence start (after previous sentence end or start of paragraph)
	for {
		line := ed.Lines[startLine]
		for startCol > 0 {
			startCol--
			if startCol < len(line) && isSentenceEnd(line[startCol]) {
				// Found previous sentence end - sentence starts after this
				startCol++
				// Skip whitespace
				for startCol < len(line) && (line[startCol] == ' ' || line[startCol] == '\t') {
					startCol++
				}
				if startCol >= len(line) && startLine < len(ed.Lines)-1 {
					// Move to next line
					startLine++
					startCol = 0
					line = ed.Lines[startLine]
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
			if strings.TrimSpace(ed.Lines[startLine-1]) == "" {
				startCol = 0
				goto foundStart
			}
			startLine--
			startCol = len(ed.Lines[startLine])
		} else {
			// Start of file
			startCol = 0
			goto foundStart
		}
	}
foundStart:

	// Search forward for sentence end
	for {
		line := ed.Lines[endLine]
		for endCol < len(line) {
			if isSentenceEnd(line[endCol]) {
				endCol++ // Include the punctuation
				goto foundEnd
			}
			endCol++
		}
		// Reached end of line, check next line
		if endLine < len(ed.Lines)-1 {
			// Check if next line is blank (paragraph boundary)
			if strings.TrimSpace(ed.Lines[endLine+1]) == "" {
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
			line := ed.Lines[endLine]
			for endCol < len(line) && (line[endCol] == ' ' || line[endCol] == '\t') {
				endCol++
			}
			if endCol < len(line) {
				break // Found non-whitespace
			}
			// Check next line
			if endLine < len(ed.Lines)-1 && strings.TrimSpace(ed.Lines[endLine+1]) != "" {
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

// findPairBoundsML finds matching bracket pairs across multiple lines
func (ed *Editor) findPairBoundsML(open, close byte, inner bool) (startLine, startCol, endLine, endCol int) {
	// Search backward for opening bracket
	startLine = ed.Cursor
	startCol = ed.Col
	depth := 0

	// First, search backward from cursor
	for {
		line := ed.Lines[startLine]
		for startCol >= 0 {
			if startCol < len(line) {
				ch := line[startCol]
				if ch == close {
					depth++
				} else if ch == open {
					if depth == 0 {
						// Found the opening bracket
						goto foundOpen
					}
					depth--
				}
			}
			startCol--
		}
		// Move to previous line
		if startLine > 0 {
			startLine--
			startCol = len(ed.Lines[startLine]) - 1
		} else {
			// Not found
			return -1, -1, -1, -1
		}
	}
foundOpen:

	// Now search forward for closing bracket
	endLine = ed.Cursor
	endCol = ed.Col
	depth = 0

	for {
		line := ed.Lines[endLine]
		for endCol < len(line) {
			ch := line[endCol]
			if ch == open {
				depth++
			} else if ch == close {
				if depth == 0 {
					// Found the closing bracket
					goto foundClose
				}
				depth--
			}
			endCol++
		}
		// Move to next line
		if endLine < len(ed.Lines)-1 {
			endLine++
			endCol = 0
		} else {
			// Not found
			return -1, -1, -1, -1
		}
	}
foundClose:

	if inner {
		// Exclude the brackets themselves
		startCol++
		// If startCol goes past end of line, move to next line
		if startCol >= len(ed.Lines[startLine]) && startLine < endLine {
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
	line := ed.Lines[ed.Cursor]
	n := len(line)

	// Try to find next word on current line
	col := ed.Col
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
		ed.Col = col
		return
	}

	// Move to next line
	for ed.Cursor < len(ed.Lines)-1 {
		ed.Cursor++
		line = ed.Lines[ed.Cursor]
		// Find first word char on new line
		col = 0
		for col < len(line) && !isWordChar(line[col]) {
			col++
		}
		if col < len(line) {
			ed.Col = col
			return
		}
	}
	// At end, go to end of last line
	ed.Col = max(0, len(ed.Lines[ed.Cursor])-1)
}

// wordBackward moves to the start of the previous word, crossing lines
func (ed *Editor) wordBackward() {
	line := ed.Lines[ed.Cursor]
	col := ed.Col

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
			ed.Col = col
			return
		}
	}

	// Move to previous line
	for ed.Cursor > 0 {
		ed.Cursor--
		line = ed.Lines[ed.Cursor]
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
		ed.Col = col
		return
	}
	// At start
	ed.Col = 0
}

// wordEnd moves to the end of the current/next word, crossing lines
func (ed *Editor) wordEnd() {
	line := ed.Lines[ed.Cursor]
	n := len(line)
	col := ed.Col

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
			ed.Col = col
			return
		}
	}

	// Move to next line
	for ed.Cursor < len(ed.Lines)-1 {
		ed.Cursor++
		line = ed.Lines[ed.Cursor]
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
		ed.Col = col
		return
	}
	// At end
	ed.Col = max(0, len(ed.Lines[ed.Cursor])-1)
}

// Undo/Redo implementation
func (ed *Editor) saveUndo() {
	// Deep copy current state
	linesCopy := make([]string, len(ed.Lines))
	copy(linesCopy, ed.Lines)
	ed.undoStack = append(ed.undoStack, EditorState{
		Lines:  linesCopy,
		Cursor: ed.Cursor,
		Col:    ed.Col,
	})
	// Clear redo stack on new change
	ed.redoStack = nil
}

func (ed *Editor) undo() {
	if len(ed.undoStack) == 0 {
		ed.StatusLine = "Already at oldest change"
		ed.updateDisplay()
		return
	}
	// Save current state to redo stack
	linesCopy := make([]string, len(ed.Lines))
	copy(linesCopy, ed.Lines)
	ed.redoStack = append(ed.redoStack, EditorState{
		Lines:  linesCopy,
		Cursor: ed.Cursor,
		Col:    ed.Col,
	})
	// Pop from undo stack
	state := ed.undoStack[len(ed.undoStack)-1]
	ed.undoStack = ed.undoStack[:len(ed.undoStack)-1]
	ed.Lines = state.Lines
	ed.Cursor = state.Cursor
	ed.Col = state.Col
	ed.StatusLine = fmt.Sprintf("Undo (%d more)", len(ed.undoStack))
	ed.updateDisplay()
	ed.updateCursor()
}

func (ed *Editor) redo() {
	if len(ed.redoStack) == 0 {
		ed.StatusLine = "Already at newest change"
		ed.updateDisplay()
		return
	}
	// Save current state to undo stack
	linesCopy := make([]string, len(ed.Lines))
	copy(linesCopy, ed.Lines)
	ed.undoStack = append(ed.undoStack, EditorState{
		Lines:  linesCopy,
		Cursor: ed.Cursor,
		Col:    ed.Col,
	})
	// Pop from redo stack
	state := ed.redoStack[len(ed.redoStack)-1]
	ed.redoStack = ed.redoStack[:len(ed.redoStack)-1]
	ed.Lines = state.Lines
	ed.Cursor = state.Cursor
	ed.Col = state.Col
	ed.StatusLine = fmt.Sprintf("Redo (%d more)", len(ed.redoStack))
	ed.updateDisplay()
	ed.updateCursor()
}

// Visual mode implementation
func (ed *Editor) enterVisualMode(app *tui.App, lineMode bool) {
	ed.Mode = "VISUAL"
	ed.visualStart = ed.Cursor
	ed.visualStartCol = ed.Col
	ed.visualLineMode = lineMode
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
		ed.Cursor = min(ed.Cursor+m.Count, len(ed.Lines)-1)
		ed.Col = min(ed.Col, len(ed.Lines[ed.Cursor]))
		ed.ensureCursorVisible()
		ed.refresh()
	})
	visualRouter.Handle("k", func(m riffkey.Match) {
		ed.Cursor = max(ed.Cursor-m.Count, 0)
		ed.Col = min(ed.Col, len(ed.Lines[ed.Cursor]))
		ed.ensureCursorVisible()
		ed.refresh()
	})
	visualRouter.Handle("h", func(m riffkey.Match) {
		ed.Col = max(0, ed.Col-m.Count)
		ed.refresh()
	})
	visualRouter.Handle("l", func(m riffkey.Match) {
		ed.Col = min(ed.Col+m.Count, max(0, len(ed.Lines[ed.Cursor])-1))
		ed.refresh()
	})
	visualRouter.Handle("gg", func(_ riffkey.Match) {
		ed.Cursor = 0
		ed.Col = 0
		ed.ensureCursorVisible()
		ed.refresh()
	})
	visualRouter.Handle("G", func(_ riffkey.Match) {
		ed.Cursor = len(ed.Lines) - 1
		ed.Col = len(ed.Lines[ed.Cursor])
		ed.ensureCursorVisible()
		ed.refresh()
	})
	visualRouter.Handle("0", func(_ riffkey.Match) {
		ed.Col = 0
		ed.refresh()
	})
	visualRouter.Handle("$", func(_ riffkey.Match) {
		ed.Col = max(0, len(ed.Lines[ed.Cursor])-1)
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
		ed.Cursor, ed.visualStart = ed.visualStart, ed.Cursor
		ed.Col, ed.visualStartCol = ed.visualStartCol, ed.Col
		ed.refresh()
	})
	visualRouter.Handle("O", func(_ riffkey.Match) {
		ed.Cursor, ed.visualStart = ed.visualStart, ed.Cursor
		ed.Col, ed.visualStartCol = ed.visualStartCol, ed.Col
		ed.refresh()
	})

	// d deletes selection
	visualRouter.Handle("d", func(_ riffkey.Match) {
		ed.saveUndo()
		if ed.visualLineMode {
			// Delete entire lines
			startLine := min(ed.visualStart, ed.Cursor)
			endLine := max(ed.visualStart, ed.Cursor)
			if endLine-startLine+1 >= len(ed.Lines) {
				ed.Lines = []string{""}
				ed.Cursor = 0
			} else {
				ed.Lines = append(ed.Lines[:startLine], ed.Lines[endLine+1:]...)
				ed.Cursor = min(startLine, len(ed.Lines)-1)
			}
			ed.Col = min(ed.Col, max(0, len(ed.Lines[ed.Cursor])-1))
		} else {
			// Character mode - can span multiple lines
			startLine := min(ed.visualStart, ed.Cursor)
			endLine := max(ed.visualStart, ed.Cursor)
			var startCol, endCol int
			if ed.visualStart < ed.Cursor || (ed.visualStart == ed.Cursor && ed.visualStartCol <= ed.Col) {
				startCol = ed.visualStartCol
				endCol = ed.Col + 1
			} else {
				startCol = ed.Col
				endCol = ed.visualStartCol + 1
			}
			yankRegister = ed.extractRange(startLine, startCol, endLine, endCol)
			ed.deleteRange(startLine, startCol, endLine, endCol)
		}
		ed.exitVisualMode(app)
	})

	// c changes selection (delete and enter insert mode)
	visualRouter.Handle("c", func(_ riffkey.Match) {
		ed.saveUndo()
		if ed.visualLineMode {
			// Change entire lines - delete and insert on first line
			startLine := min(ed.visualStart, ed.Cursor)
			endLine := max(ed.visualStart, ed.Cursor)
			yankRegister = ed.extractRange(startLine, 0, endLine, len(ed.Lines[endLine]))
			if endLine-startLine+1 >= len(ed.Lines) {
				ed.Lines = []string{""}
				ed.Cursor = 0
			} else {
				ed.Lines = append(ed.Lines[:startLine], ed.Lines[endLine+1:]...)
				ed.Cursor = min(startLine, len(ed.Lines)-1)
			}
			// Insert a blank line to type on
			newLines := make([]string, len(ed.Lines)+1)
			copy(newLines[:ed.Cursor], ed.Lines[:ed.Cursor])
			newLines[ed.Cursor] = ""
			copy(newLines[ed.Cursor+1:], ed.Lines[ed.Cursor:])
			ed.Lines = newLines
			ed.Col = 0
		} else {
			// Character mode - delete selection
			startLine := min(ed.visualStart, ed.Cursor)
			endLine := max(ed.visualStart, ed.Cursor)
			var startCol, endCol int
			if ed.visualStart < ed.Cursor || (ed.visualStart == ed.Cursor && ed.visualStartCol <= ed.Col) {
				startCol = ed.visualStartCol
				endCol = ed.Col + 1
			} else {
				startCol = ed.Col
				endCol = ed.visualStartCol + 1
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
		if ed.visualLineMode {
			startLine := min(ed.visualStart, ed.Cursor)
			endLine := max(ed.visualStart, ed.Cursor)
			var yanked string
			for i := startLine; i <= endLine; i++ {
				yanked += ed.Lines[i]
				if i < endLine {
					yanked += "\n"
				}
			}
			yankRegister = yanked
			ed.StatusLine = fmt.Sprintf("Yanked %d lines", endLine-startLine+1)
		} else {
			// Character mode - can span multiple lines
			startLine := min(ed.visualStart, ed.Cursor)
			endLine := max(ed.visualStart, ed.Cursor)
			var startCol, endCol int
			if ed.visualStart < ed.Cursor || (ed.visualStart == ed.Cursor && ed.visualStartCol <= ed.Col) {
				startCol = ed.visualStartCol
				endCol = ed.Col + 1
			} else {
				startCol = ed.Col
				endCol = ed.visualStartCol + 1
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
				ed.visualStart = startLine
				ed.visualStartCol = startCol
				ed.Cursor = endLine
				ed.Col = max(0, endCol-1) // endCol is exclusive, cursor should be on last char
				ed.visualLineMode = false
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
			line := ed.Lines[ed.Cursor]
			start, end := objFn(line, ed.Col)
			if start < end {
				ed.visualStart = ed.Cursor
				ed.visualStartCol = start
				ed.Col = end - 1
				ed.visualLineMode = false
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
		app.Stop()
	case "w", "write":
		ed.StatusLine = "E37: No write since last change (use :w! to override)"
		ed.updateDisplay()
	case "wq", "x":
		ed.StatusLine = "E37: No write since last change (use :wq! to override)"
		ed.updateDisplay()
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
			if lineNum > 0 && lineNum <= len(ed.Lines) {
				ed.Cursor = lineNum - 1
				ed.Col = 0
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
	startLine := ed.Cursor
	startCol := ed.Col + 1
	if actualDir < 0 {
		startCol = ed.Col - 1
	}

	// Search through all lines
	for i := 0; i < len(ed.Lines); i++ {
		lineIdx := startLine
		if actualDir > 0 {
			lineIdx = (startLine + i) % len(ed.Lines)
		} else {
			lineIdx = (startLine - i + len(ed.Lines)) % len(ed.Lines)
		}

		line := ed.Lines[lineIdx]
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
			ed.Cursor = lineIdx
			ed.Col = col
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
	line := ed.Lines[ed.Cursor]
	if forward {
		for i := ed.Col + 1; i < len(line); i++ {
			if rune(line[i]) == ch {
				if till {
					ed.Col = i - 1
				} else {
					ed.Col = i
				}
				ed.updateCursor()
				return
			}
		}
	} else {
		for i := ed.Col - 1; i >= 0; i-- {
			if rune(line[i]) == ch {
				if till {
					ed.Col = i + 1
				} else {
					ed.Col = i
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
