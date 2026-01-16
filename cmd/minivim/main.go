// minivim: A tiny vim-like editor demonstrating riffkey TextInput with tui framework
//
// Normal mode: j/k=move, i=insert, a=append, o=new line, dd=delete line, q=quit
// Insert mode: Type text, Esc=back to normal, all standard editing keys work
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"riffkey"
	"tui"
)

// Layout constants
const (
	headerRows   = 0  // Content starts at row 0 now
	footerRows   = 2  // Status bar + message line
	renderBuffer = 30 // Lines to render above/below viewport for smooth scrolling
)

// VisualMode represents the type of visual selection
type VisualMode int

const (
	VisualNone  VisualMode = iota // Not in visual mode
	VisualChar                    // Character-wise (v)
	VisualLine                    // Line-wise (V)
	VisualBlock                   // Block-wise (Ctrl-V)
)

// Pos represents a position in a buffer
type Pos struct {
	Line int
	Col  int
}

// Range represents a selection in a buffer
type Range struct {
	Start Pos
	End   Pos
}

// GitSign represents a git change sign for a line
type GitSign int

const (
	GitSignNone     GitSign = iota
	GitSignAdded            // new line (not in HEAD)
	GitSignModified         // modified line
	GitSignRemoved          // line(s) deleted below this line
)

// DirEntry represents a file or directory in the file tree
type DirEntry struct {
	Name  string
	IsDir bool
}

// FileTree holds the state for netrw-style file browsing
type FileTree struct {
	Path       string     // current directory path
	Entries    []DirEntry // sorted list of entries
	ShowHidden bool       // show dotfiles
}

// Buffer holds file content (can be shared across windows)
type Buffer struct {
	Lines     []string
	FileName  string
	undoStack []EditorState
	redoStack []EditorState
	marks     map[rune]Pos // a-z marks (per-buffer)
	gitSigns  []GitSign    // git change signs per line
	fileTree  *FileTree    // non-nil if this is a netrw buffer
}

// Window is a view into a buffer
type Window struct {
	buffer *Buffer

	// Cursor position
	Cursor int
	Col    int

	// Viewport
	topLine        int
	leftCol        int // horizontal scroll offset
	viewportHeight int
	viewportWidth  int // for vertical splits

	// Visual mode selection (per-window)
	visualStart    int
	visualStartCol int
	visualMode     VisualMode

	// Rendering
	contentLayer *tui.Layer
	lineNumWidth int
	StatusBar    []tui.Span
	renderedMin  int
	renderedMax  int

	// Debug stats
	debugMode          bool
	lastRenderTime     time.Duration
	lastLinesRendered  int
	totalRenders       int
	totalLinesRendered int

	// Jump list for C-o/C-i
	jumpList  []JumpPos
	jumpIndex int
}

// JumpPos records a position for the jump list
type JumpPos struct {
	Line int
	Col  int
}

// SplitDir indicates the split direction
type SplitDir int

const (
	SplitNone       SplitDir = iota
	SplitHorizontal          // windows stacked vertically (like :sp)
	SplitVertical            // windows side by side (like :vs)
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

	// Display options
	relativeNumber bool // show relative line numbers (like vim's relativenumber)
	cursorLine     bool // highlight the entire cursor line (like vim's cursorline)
	showSignColumn bool // show git gutter signs column

	// Search (global)
	searchPattern   string
	searchDirection int
	lastSearch      string

	// f/F/t/T (global)
	lastFindChar rune
	lastFindDir  int
	lastFindTill bool

	// Command line mode (global)
	cmdLineActive  bool
	cmdLinePrompt  string
	cmdLineInput   string
	lastColonCmd   string // for @: repeat

	// Command completion (wildmenu)
	cmdCompletions      []string   // all available commands (harvested once)
	cmdMatches          []string   // filtered matches for current input
	cmdMatchSelected    int        // selected match index
	cmdCompletionActive bool       // whether completion popup is showing
	cmdWildmenuSpans    []tui.Span // rendered wildmenu line

	// Debounce for C-a/C-x
	lastNumberModify time.Time

	// Macros (app manages storage)
	macros         map[rune]riffkey.Macro
	recordingMacro rune // which register we're recording to
	lastMacro      rune // for @@

	// Block insert mode (for visual block I/A)
	blockInsertLines []int // lines to replicate text to
	blockInsertCol   int   // column where insert started
	blockInsertStart int   // original line length before insert

	// Fuzzy finder (declarative overlay view)
	fuzzy FuzzyState
}

// FuzzyState holds the state for the fuzzy finder overlay
type FuzzyState struct {
	Active     bool     // whether fuzzy finder is showing
	Query      string   // current search query
	AllItems   []string // all available items
	Matches    []string // filtered matches
	Selected   int      // selected index in matches
	SourceDir  string   // directory being searched
	PrevBuffer *Buffer  // buffer to restore on cancel
	PrevCursor int      // cursor position to restore
}

// Helper methods to access current window/buffer
func (ed *Editor) win() *Window { return ed.focusedWindow }
func (ed *Editor) buf() *Buffer { return ed.win().buffer }

// Cursor returns the current cursor position
func (ed *Editor) Cursor() Pos {
	return Pos{Line: ed.win().Cursor, Col: ed.win().Col}
}

// SetCursor moves cursor to position with bounds clamping, viewport scroll, and display update
func (ed *Editor) SetCursor(p Pos) {
	ed.SetCursorQuiet(p)
	ed.ensureCursorVisible()
	ed.updateDisplay()
	ed.updateCursor()
}

// SetCursorQuiet moves cursor with clamping but no display update (for batch operations)
func (ed *Editor) SetCursorQuiet(p Pos) {
	// Clamp line to buffer bounds
	p.Line = max(0, min(p.Line, len(ed.buf().Lines)-1))
	// Clamp column to line length
	lineLen := len(ed.buf().Lines[p.Line])
	if lineLen == 0 {
		p.Col = 0
	} else {
		p.Col = max(0, min(p.Col, lineLen-1))
	}
	ed.win().Cursor = p.Line
	ed.win().Col = p.Col
}

// moveCursor moves cursor with clamping and scroll but NO display update.
// This is the core movement primitive - display updates are handled by middleware.
func (ed *Editor) moveCursor(p Pos) {
	ed.SetCursorQuiet(p)
	ed.ensureCursorVisible()
}

// =============================================================================
// Movement Actions - pure cursor movement, NO display updates
// Display updates are handled by middleware (normal mode vs visual mode)
// =============================================================================

// GotoLine moves to line n (1-indexed for user-facing commands)
func (ed *Editor) GotoLine(n int) Pos {
	ed.addJump()
	ed.moveCursor(Pos{Line: n - 1, Col: 0})
	return ed.Cursor()
}

// BufferStart moves to the first line (gg)
func (ed *Editor) BufferStart() Pos {
	ed.addJump()
	ed.moveCursor(Pos{Line: 0, Col: 0})
	return ed.Cursor()
}

// BufferEnd moves to the last line (G)
func (ed *Editor) BufferEnd() Pos {
	ed.addJump()
	ed.moveCursor(Pos{Line: len(ed.buf().Lines) - 1, Col: 0})
	return ed.Cursor()
}

// LineStart moves to the first column (0)
func (ed *Editor) LineStart() Pos {
	ed.moveCursor(Pos{Line: ed.win().Cursor, Col: 0})
	return ed.Cursor()
}

// LineEnd moves to the last column ($)
func (ed *Editor) LineEnd() Pos {
	line := ed.buf().Lines[ed.win().Cursor]
	ed.moveCursor(Pos{Line: ed.win().Cursor, Col: len(line)})
	return ed.Cursor()
}

// FirstNonBlank moves to the first non-whitespace character (^)
func (ed *Editor) FirstNonBlank() Pos {
	line := ed.buf().Lines[ed.win().Cursor]
	col := 0
	for col < len(line) && (line[col] == ' ' || line[col] == '\t') {
		col++
	}
	ed.moveCursor(Pos{Line: ed.win().Cursor, Col: col})
	return ed.Cursor()
}

// Left moves cursor left by n characters (h)
func (ed *Editor) Left(n int) Pos {
	ed.moveCursor(Pos{Line: ed.win().Cursor, Col: ed.win().Col - n})
	return ed.Cursor()
}

// Right moves cursor right by n characters (l)
func (ed *Editor) Right(n int) Pos {
	ed.moveCursor(Pos{Line: ed.win().Cursor, Col: ed.win().Col + n})
	return ed.Cursor()
}

// Up moves cursor up by n lines (k)
func (ed *Editor) Up(n int) Pos {
	ed.moveCursor(Pos{Line: ed.win().Cursor - n, Col: ed.win().Col})
	return ed.Cursor()
}

// Down moves cursor down by n lines (j)
func (ed *Editor) Down(n int) Pos {
	ed.moveCursor(Pos{Line: ed.win().Cursor + n, Col: ed.win().Col})
	return ed.Cursor()
}

// HalfPageDown moves cursor down by half a page (C-d)
func (ed *Editor) HalfPageDown() Pos {
	ed.addJump()
	half := ed.win().viewportHeight / 2
	ed.moveCursor(Pos{Line: ed.win().Cursor + half, Col: ed.win().Col})
	return ed.Cursor()
}

// HalfPageUp moves cursor up by half a page (C-u)
func (ed *Editor) HalfPageUp() Pos {
	ed.addJump()
	half := ed.win().viewportHeight / 2
	ed.moveCursor(Pos{Line: ed.win().Cursor - half, Col: ed.win().Col})
	return ed.Cursor()
}

// PageDown moves cursor down by a full page (C-f)
func (ed *Editor) PageDown() Pos {
	ed.addJump()
	ed.moveCursor(Pos{Line: ed.win().Cursor + ed.win().viewportHeight, Col: ed.win().Col})
	return ed.Cursor()
}

// PageUp moves cursor up by a full page (C-b)
func (ed *Editor) PageUp() Pos {
	ed.addJump()
	ed.moveCursor(Pos{Line: ed.win().Cursor - ed.win().viewportHeight, Col: ed.win().Col})
	return ed.Cursor()
}

// CenterCursor positions viewport so cursor is in the middle (zz)
func (ed *Editor) CenterCursor() Pos {
	ed.scrollCursorTo(ed.win().viewportHeight / 2)
	return ed.Cursor()
}

// CursorToTop positions viewport so cursor is at the top (zt)
func (ed *Editor) CursorToTop() Pos {
	ed.scrollCursorTo(0)
	return ed.Cursor()
}

// CursorToBottom positions viewport so cursor is at the bottom (zb)
func (ed *Editor) CursorToBottom() Pos {
	ed.scrollCursorTo(ed.win().viewportHeight - 1)
	return ed.Cursor()
}

// JumpBack moves to previous position in jump list (C-o)
func (ed *Editor) JumpBack() Pos {
	ed.jumpBack()
	return ed.Cursor()
}

// JumpForward moves to next position in jump list (C-i)
func (ed *Editor) JumpForward() Pos {
	ed.jumpForward()
	return ed.Cursor()
}

// NextWordStart moves to the start of the next word (w)
func (ed *Editor) NextWordStart(n int) Pos {
	for range n {
		ed.wordForward()
	}
	ed.ensureCursorVisible()
	return ed.Cursor()
}

// PrevWordStart moves to the start of the previous word (b)
func (ed *Editor) PrevWordStart(n int) Pos {
	for range n {
		ed.wordBackward()
	}
	ed.ensureCursorVisible()
	return ed.Cursor()
}

// NextWordEnd moves to the end of the current/next word (e)
func (ed *Editor) NextWordEnd(n int) Pos {
	for range n {
		ed.wordEnd()
	}
	ed.ensureCursorVisible()
	return ed.Cursor()
}

// =============================================================================
// Editing Actions
// =============================================================================

// ToggleCase toggles the case of the character under cursor and moves right
func (ed *Editor) ToggleCase() {
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
}

// JoinLines joins the current line with the next line
func (ed *Editor) JoinLines() {
	ed.saveUndo()
	if ed.win().Cursor < len(ed.buf().Lines)-1 {
		ed.buf().Lines[ed.win().Cursor] += " " + ed.buf().Lines[ed.win().Cursor+1]
		ed.buf().Lines = append(ed.buf().Lines[:ed.win().Cursor+1], ed.buf().Lines[ed.win().Cursor+2:]...)
		ed.updateDisplay()
	}
}

// DeleteChar deletes n characters under and after cursor (x)
func (ed *Editor) DeleteChar(n int) {
	ed.saveUndo()
	for i := 0; i < n; i++ {
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
}

// DeleteLine deletes n lines starting from cursor (dd)
func (ed *Editor) DeleteLine(n int) {
	ed.saveUndo()
	for i := 0; i < n; i++ {
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
}

// Undo reverts the last change
func (ed *Editor) Undo() {
	ed.undo()
}

// Redo reapplies the last undone change
func (ed *Editor) Redo() {
	ed.redo()
}

// SearchNext finds the next match in the current search direction
func (ed *Editor) SearchNext() {
	ed.searchNext(1)
}

// SearchPrev finds the previous match (opposite of search direction)
func (ed *Editor) SearchPrev() {
	ed.searchNext(-1)
}

// IncrementNumber increments the number under or after cursor (C-a)
func (ed *Editor) IncrementNumber() {
	ed.modifyNumber(1)
}

// DecrementNumber decrements the number under or after cursor (C-x)
func (ed *Editor) DecrementNumber() {
	ed.modifyNumber(-1)
}

// ScrollLineDown scrolls viewport down one line (C-e)
func (ed *Editor) ScrollLineDown() {
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
}

// ScrollLineUp scrolls viewport up one line (C-y)
func (ed *Editor) ScrollLineUp() {
	if ed.win().topLine > 0 {
		ed.win().topLine--
		if ed.win().Cursor >= ed.win().topLine+ed.win().viewportHeight {
			ed.win().Cursor = ed.win().topLine + ed.win().viewportHeight - 1
			ed.win().Col = min(ed.win().Col, len(ed.buf().Lines[ed.win().Cursor]))
		}
		ed.updateDisplay()
		ed.updateCursor()
	}
}

// ClearSearchHighlight clears search highlighting (C-l / :nohl)
func (ed *Editor) ClearSearchHighlight() {
	ed.searchPattern = ""
	ed.invalidateRenderedRange()
	ed.updateDisplay()
	ed.updateCursor()
}

// =============================================================================
// Insert Mode Actions - buffer operations for insert mode handlers
// These don't include display updates (caller handles that + TextHandler rebinding)
// =============================================================================

// InsertNewline splits the current line at cursor and moves to the new line (Enter in insert mode)
func (ed *Editor) InsertNewline() {
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
}

// DeleteToLineStart deletes from cursor to start of line (C-u in insert mode)
func (ed *Editor) DeleteToLineStart() {
	line := ed.buf().Lines[ed.win().Cursor]
	ed.buf().Lines[ed.win().Cursor] = line[ed.win().Col:]
	ed.win().Col = 0
}

// DeleteToLineEnd deletes from cursor to end of line (C-k in insert mode)
func (ed *Editor) DeleteToLineEnd() {
	line := ed.buf().Lines[ed.win().Cursor]
	ed.buf().Lines[ed.win().Cursor] = line[:ed.win().Col]
}

// IndentLine adds a tab at the start of the current line (C-t in insert mode)
func (ed *Editor) IndentLine() {
	ed.buf().Lines[ed.win().Cursor] = "\t" + ed.buf().Lines[ed.win().Cursor]
	ed.win().Col++
}

// UnindentLine removes leading whitespace from the current line (C-d in insert mode)
func (ed *Editor) UnindentLine() {
	line := ed.buf().Lines[ed.win().Cursor]
	if len(line) > 0 && (line[0] == '\t' || line[0] == ' ') {
		ed.buf().Lines[ed.win().Cursor] = line[1:]
		if ed.win().Col > 0 {
			ed.win().Col--
		}
	}
}

// =============================================================================
// Mode Entry Actions
// =============================================================================

// EnterInsert enters insert mode at cursor position (i)
func (ed *Editor) EnterInsert() {
	ed.enterInsertMode(ed.app)
}

// Append enters insert mode after cursor (a)
func (ed *Editor) Append() {
	if len(ed.buf().Lines[ed.win().Cursor]) > 0 {
		ed.win().Col++
	}
	ed.enterInsertMode(ed.app)
}

// AppendLine enters insert mode at end of line (A)
func (ed *Editor) AppendLine() {
	ed.win().Col = len(ed.buf().Lines[ed.win().Cursor])
	ed.enterInsertMode(ed.app)
}

// InsertLine enters insert mode at start of line (I)
func (ed *Editor) InsertLine() {
	ed.win().Col = 0
	ed.enterInsertMode(ed.app)
}

// OpenBelow opens a new line below and enters insert mode (o)
func (ed *Editor) OpenBelow() {
	ed.win().Cursor++
	newLines := make([]string, len(ed.buf().Lines)+1)
	copy(newLines[:ed.win().Cursor], ed.buf().Lines[:ed.win().Cursor])
	newLines[ed.win().Cursor] = ""
	copy(newLines[ed.win().Cursor+1:], ed.buf().Lines[ed.win().Cursor:])
	ed.buf().Lines = newLines
	ed.win().Col = 0
	ed.updateDisplay()
	ed.enterInsertMode(ed.app)
}

// OpenAbove opens a new line above and enters insert mode (O)
func (ed *Editor) OpenAbove() {
	newLines := make([]string, len(ed.buf().Lines)+1)
	copy(newLines[:ed.win().Cursor], ed.buf().Lines[:ed.win().Cursor])
	newLines[ed.win().Cursor] = ""
	copy(newLines[ed.win().Cursor+1:], ed.buf().Lines[ed.win().Cursor:])
	ed.buf().Lines = newLines
	ed.win().Col = 0
	ed.updateDisplay()
	ed.enterInsertMode(ed.app)
}

// EnterVisual enters character-wise visual mode (v)
func (ed *Editor) EnterVisual() {
	ed.enterVisualMode(ed.app, VisualChar)
}

// EnterVisualLine enters line-wise visual mode (V)
func (ed *Editor) EnterVisualLine() {
	ed.enterVisualMode(ed.app, VisualLine)
}

// EnterVisualBlock enters block-wise visual mode (Ctrl-V)
func (ed *Editor) EnterVisualBlock() {
	ed.enterVisualMode(ed.app, VisualBlock)
}

// =============================================================================
// Paste Actions
// =============================================================================

// Paste pastes after cursor (p)
func (ed *Editor) Paste() {
	if yankRegister != "" {
		line := ed.buf().Lines[ed.win().Cursor]
		pos := min(ed.win().Col+1, len(line))
		ed.buf().Lines[ed.win().Cursor] = line[:pos] + yankRegister + line[pos:]
		ed.win().Col = pos + len(yankRegister) - 1
		ed.updateDisplay()
		ed.updateCursor()
	}
}

// PasteBefore pastes before cursor (P)
func (ed *Editor) PasteBefore() {
	if yankRegister != "" {
		line := ed.buf().Lines[ed.win().Cursor]
		ed.buf().Lines[ed.win().Cursor] = line[:ed.win().Col] + yankRegister + line[ed.win().Col:]
		ed.updateDisplay()
		ed.updateCursor()
	}
}

// =============================================================================
// Screen Position Actions
// =============================================================================

// ScreenTop moves cursor to top of screen (H)
func (ed *Editor) ScreenTop(offset int) {
	targetLine := ed.win().topLine + offset - 1
	targetLine = max(ed.win().topLine, min(targetLine, ed.win().topLine+ed.win().viewportHeight-1))
	targetLine = min(targetLine, len(ed.buf().Lines)-1)
	ed.win().Cursor = targetLine
	ed.FirstNonBlank()
}

// ScreenMiddle moves cursor to middle of screen (M)
func (ed *Editor) ScreenMiddle() {
	targetLine := ed.win().topLine + ed.win().viewportHeight/2
	targetLine = min(targetLine, len(ed.buf().Lines)-1)
	ed.win().Cursor = targetLine
	ed.FirstNonBlank()
}

// ScreenBottom moves cursor to bottom of screen (L)
func (ed *Editor) ScreenBottom(offset int) {
	targetLine := ed.win().topLine + ed.win().viewportHeight - offset
	targetLine = max(ed.win().topLine, min(targetLine, ed.win().topLine+ed.win().viewportHeight-1))
	targetLine = min(targetLine, len(ed.buf().Lines)-1)
	ed.win().Cursor = targetLine
	ed.FirstNonBlank()
}

// ScrollRight scrolls viewport right by n columns (zl)
func (ed *Editor) ScrollRight(n int) {
	ed.scrollRight(n)
}

// ScrollLeft scrolls viewport left by n columns (zh)
func (ed *Editor) ScrollLeft(n int) {
	ed.scrollLeft(n)
}

// CursorToScreenStart scrolls to put cursor at left edge (zs)
func (ed *Editor) CursorToScreenStart() {
	ed.win().leftCol = ed.win().Col
	ed.updateDisplay()
	ed.updateCursor()
}

// CursorToScreenEnd scrolls to put cursor at right edge (ze)
func (ed *Editor) CursorToScreenEnd() {
	ed.win().leftCol = max(0, ed.win().Col-ed.win().viewportWidth+ed.win().lineNumWidth+1)
	ed.updateDisplay()
	ed.updateCursor()
}

// NextScreenLine moves to first line below window at top (z+)
func (ed *Editor) NextScreenLine() {
	newLine := ed.win().topLine + ed.win().viewportHeight
	if newLine < len(ed.buf().Lines) {
		ed.win().Cursor = newLine
		ed.CursorToTop()
		ed.FirstNonBlank()
	}
}

// PrevScreenLine moves to last line above window at bottom (z^)
func (ed *Editor) PrevScreenLine() {
	newLine := ed.win().topLine - 1
	if newLine >= 0 {
		ed.win().Cursor = newLine
		ed.CursorToBottom()
		ed.FirstNonBlank()
	}
}

// ToggleDebug toggles debug mode (C-g)
func (ed *Editor) ToggleDebug() {
	ed.win().debugMode = !ed.win().debugMode
	if ed.win().debugMode {
		ed.StatusLine = "Debug mode ON - showing render stats"
	} else {
		ed.StatusLine = "Debug mode OFF"
	}
	ed.updateDisplay()
}

// =============================================================================
// Mark Actions
// =============================================================================

// SetMark sets a mark at the current position (m{a-z})
func (ed *Editor) SetMark(reg rune) {
	ed.buf().marks[reg] = Pos{
		Line: ed.win().Cursor,
		Col:  ed.win().Col,
	}
	ed.StatusLine = fmt.Sprintf("Mark '%c' set", reg)
	ed.updateDisplay()
}

// GotoMark jumps to exact mark position (`{a-z})
func (ed *Editor) GotoMark(reg rune) {
	if mark, ok := ed.buf().marks[reg]; ok {
		ed.addJump()
		ed.win().Cursor = min(mark.Line, len(ed.buf().Lines)-1)
		line := ed.buf().Lines[ed.win().Cursor]
		ed.win().Col = min(mark.Col, max(0, len(line)-1))
		ed.ensureCursorVisible()
		ed.StatusLine = fmt.Sprintf("Mark `%c'", reg)
	} else {
		ed.StatusLine = fmt.Sprintf("E20: Mark not set: %c", reg)
	}
	ed.updateDisplay()
	ed.updateCursor()
}

// GotoMarkLine jumps to mark line, first non-blank ('{a-z})
func (ed *Editor) GotoMarkLine(reg rune) {
	if mark, ok := ed.buf().marks[reg]; ok {
		ed.addJump()
		ed.win().Cursor = min(mark.Line, len(ed.buf().Lines)-1)
		line := ed.buf().Lines[ed.win().Cursor]
		col := 0
		for col < len(line) && (line[col] == ' ' || line[col] == '\t') {
			col++
		}
		ed.win().Col = min(col, max(0, len(line)-1))
		ed.ensureCursorVisible()
		ed.StatusLine = fmt.Sprintf("Mark '%c'", reg)
	} else {
		ed.StatusLine = fmt.Sprintf("E20: Mark not set: %c", reg)
	}
	ed.updateDisplay()
	ed.updateCursor()
}

// =============================================================================
// Other Actions
// =============================================================================

// ReplaceChar replaces the character under cursor (r{char})
func (ed *Editor) ReplaceChar(ch rune) {
	ed.saveUndo()
	line := ed.buf().Lines[ed.win().Cursor]
	if ed.win().Col < len(line) {
		ed.buf().Lines[ed.win().Cursor] = line[:ed.win().Col] + string(ch) + line[ed.win().Col+1:]
		ed.updateDisplay()
	}
}

// RepeatLast repeats the last change (.) - simplified version
func (ed *Editor) RepeatLast() {
	if yankRegister != "" {
		ed.saveUndo()
		line := ed.buf().Lines[ed.win().Cursor]
		ed.buf().Lines[ed.win().Cursor] = line[:ed.win().Col] + yankRegister + line[ed.win().Col:]
		ed.win().Col += len(yankRegister)
		ed.updateDisplay()
		ed.updateCursor()
	}
}

// EnterCommand enters command line mode (:, /, ?)
func (ed *Editor) EnterCommand(prompt string) {
	ed.enterCommandMode(ed.app, prompt)
}

// =============================================================================
// Macro Actions
// =============================================================================

// StopMacroRecording stops recording and saves the macro to the register
func (ed *Editor) StopMacroRecording(app *tui.App) {
	macro := app.Input().StopRecording()
	ed.macros[ed.recordingMacro] = macro
	ed.StatusLine = fmt.Sprintf("Recorded @%c (%d keys)", ed.recordingMacro, len(macro))
	ed.recordingMacro = 0
	ed.updateDisplay()
}

// StartMacroRecording begins recording keystrokes to the given register
func (ed *Editor) StartMacroRecording(app *tui.App, reg rune) {
	ed.recordingMacro = reg
	app.Input().StartRecording()
	ed.StatusLine = fmt.Sprintf("Recording @%c...", reg)
	ed.updateDisplay()
}

// PlayMacro executes the macro stored in the given register
func (ed *Editor) PlayMacro(app *tui.App, reg rune) {
	if macro, ok := ed.macros[reg]; ok && len(macro) > 0 {
		ed.lastMacro = reg
		app.Input().ExecuteMacro(macro)
		ed.StatusLine = fmt.Sprintf("Played @%c", reg)
	} else {
		ed.StatusLine = fmt.Sprintf("E35: No recorded macro in register %c", reg)
	}
	ed.updateDisplay()
}

// =============================================================================
// Prompt Helpers - push a router to wait for a character input
// =============================================================================

// promptForRegister pushes a router that waits for a register (a-z) and calls the action
func (ed *Editor) promptForRegister(app *tui.App, action func(rune)) {
	router := riffkey.NewRouter()
	router.HandleUnmatched(func(k riffkey.Key) bool {
		app.Pop()
		if k.Rune >= 'a' && k.Rune <= 'z' {
			action(k.Rune)
		}
		return true
	})
	router.Handle("<Esc>", func(_ riffkey.Match) { app.Pop() })
	app.Push(router)
}

// promptForChar pushes a router that waits for any printable character and calls the action
func (ed *Editor) promptForChar(app *tui.App, action func(rune)) {
	router := riffkey.NewRouter().Name("char-prompt")
	router.HandleUnmatched(func(k riffkey.Key) bool {
		if k.Rune != 0 && k.Mod == riffkey.ModNone {
			action(k.Rune)
		}
		app.Pop()
		return true
	})
	router.Handle("<Esc>", func(_ riffkey.Match) { app.Pop() })
	app.Push(router)
}

// =============================================================================
// Window Actions
// =============================================================================

// FocusNextWindow focuses the next window (C-w w)
func (ed *Editor) FocusNextWindow() {
	ed.focusNextWindow()
}

// FocusPrevWindow focuses the previous window (C-w W)
func (ed *Editor) FocusPrevWindow() {
	ed.focusPrevWindow()
}

// SplitHoriz splits the window horizontally (C-w s)
func (ed *Editor) SplitHoriz() {
	ed.splitHorizontal()
}

// SplitVert splits the window vertically (C-w v)
func (ed *Editor) SplitVert() {
	ed.splitVertical()
}

// CloseWindow closes the current window (C-w c)
func (ed *Editor) CloseWindow() {
	ed.closeWindow()
}

// OnlyWindow closes all other windows (C-w o)
func (ed *Editor) OnlyWindow() {
	ed.closeOtherWindows()
}

// FocusDown moves to the window below (C-w j)
func (ed *Editor) FocusDown() {
	ed.focusDirection(SplitHorizontal, 1)
}

// FocusUp moves to the window above (C-w k)
func (ed *Editor) FocusUp() {
	ed.focusDirection(SplitHorizontal, -1)
}

// FocusLeft moves to the window left (C-w h)
func (ed *Editor) FocusLeft() {
	ed.focusDirection(SplitVertical, -1)
}

// FocusRight moves to the window right (C-w l)
func (ed *Editor) FocusRight() {
	ed.focusDirection(SplitVertical, 1)
}

// =============================================================================
// Motion Ranges - return (startLine, startCol, endLine, endCol) for operators
// =============================================================================

// MotionDown returns range from current line to n lines below (for dj, cj, yj)
func (ed *Editor) MotionDown(count int) Range {
	endLine := min(ed.win().Cursor+count, len(ed.buf().Lines)-1)
	return Range{
		Start: Pos{Line: ed.win().Cursor, Col: 0},
		End:   Pos{Line: endLine, Col: len(ed.buf().Lines[endLine])},
	}
}

// MotionUp returns range from n lines above to current line (for dk, ck, yk)
func (ed *Editor) MotionUp(count int) Range {
	startLine := max(ed.win().Cursor-count, 0)
	return Range{
		Start: Pos{Line: startLine, Col: 0},
		End:   Pos{Line: ed.win().Cursor, Col: len(ed.buf().Lines[ed.win().Cursor])},
	}
}

// MotionToStart returns range from buffer start to current line (for dgg, cgg, ygg)
func (ed *Editor) MotionToStart() Range {
	return Range{
		Start: Pos{Line: 0, Col: 0},
		End:   Pos{Line: ed.win().Cursor, Col: len(ed.buf().Lines[ed.win().Cursor])},
	}
}

// MotionToEnd returns range from current line to buffer end (for dG, cG, yG)
func (ed *Editor) MotionToEnd() Range {
	endLine := len(ed.buf().Lines) - 1
	return Range{
		Start: Pos{Line: ed.win().Cursor, Col: 0},
		End:   Pos{Line: endLine, Col: len(ed.buf().Lines[endLine])},
	}
}

// MotionWordForward returns range from cursor to n words forward (for dw, cw, yw)
func (ed *Editor) MotionWordForward(count int) Range {
	startLine, startCol := ed.win().Cursor, ed.win().Col
	for range count {
		ed.wordForward()
	}
	endLine, endCol := ed.win().Cursor, ed.win().Col
	ed.win().Cursor, ed.win().Col = startLine, startCol
	return Range{
		Start: Pos{Line: startLine, Col: startCol},
		End:   Pos{Line: endLine, Col: endCol},
	}
}

// MotionWordBackward returns range from n words back to cursor (for db, cb, yb)
func (ed *Editor) MotionWordBackward(count int) Range {
	endLine, endCol := ed.win().Cursor, ed.win().Col
	for range count {
		ed.wordBackward()
	}
	startLine, startCol := ed.win().Cursor, ed.win().Col
	ed.win().Cursor, ed.win().Col = endLine, endCol
	return Range{
		Start: Pos{Line: startLine, Col: startCol},
		End:   Pos{Line: endLine, Col: endCol},
	}
}

// MotionWordEnd returns range from cursor to end of nth word (for de, ce, ye)
func (ed *Editor) MotionWordEnd(count int) Range {
	startLine, startCol := ed.win().Cursor, ed.win().Col
	for range count {
		ed.wordEnd()
	}
	endLine, endCol := ed.win().Cursor, ed.win().Col+1
	ed.win().Cursor, ed.win().Col = startLine, startCol
	return Range{
		Start: Pos{Line: startLine, Col: startCol},
		End:   Pos{Line: endLine, Col: endCol},
	}
}

// MotionToLineEnd returns range from cursor to end of line (for d$, c$, y$)
func (ed *Editor) MotionToLineEnd() Range {
	return Range{
		Start: Pos{Line: ed.win().Cursor, Col: ed.win().Col},
		End:   Pos{Line: ed.win().Cursor, Col: len(ed.buf().Lines[ed.win().Cursor])},
	}
}

// MotionToLineStart returns range from start of line to cursor (for d0, c0, y0)
func (ed *Editor) MotionToLineStart() Range {
	return Range{
		Start: Pos{Line: ed.win().Cursor, Col: 0},
		End:   Pos{Line: ed.win().Cursor, Col: ed.win().Col},
	}
}

// =============================================================================
// Line Operations - whole line actions
// =============================================================================

// ChangeLine clears the current line and enters insert mode (cc, S)
func (ed *Editor) ChangeLine(app *tui.App) {
	ed.saveUndo()
	ed.buf().Lines[ed.win().Cursor] = ""
	ed.win().Col = 0
	ed.updateDisplay()
	ed.enterInsertMode(app)
}

// YankLine yanks the current line (yy, Y)
func (ed *Editor) YankLine() {
	yankRegister = ed.buf().Lines[ed.win().Cursor]
	ed.StatusLine = fmt.Sprintf("Yanked: %q", yankRegister)
	ed.updateDisplay()
}

// =============================================================================
// Visual Mode Actions
// =============================================================================

// VisualRange returns normalized selection bounds (start always <= end)
func (ed *Editor) VisualRange() Range {
	startLine := min(ed.win().visualStart, ed.win().Cursor)
	endLine := max(ed.win().visualStart, ed.win().Cursor)

	var startCol, endCol int
	if ed.win().visualStart < ed.win().Cursor ||
		(ed.win().visualStart == ed.win().Cursor && ed.win().visualStartCol <= ed.win().Col) {
		startCol = ed.win().visualStartCol
		endCol = ed.win().Col + 1
	} else {
		startCol = ed.win().Col
		endCol = ed.win().visualStartCol + 1
	}
	return Range{
		Start: Pos{Line: startLine, Col: startCol},
		End:   Pos{Line: endLine, Col: endCol},
	}
}

// SwapVisualEnds swaps cursor to other end of selection (o, O)
func (ed *Editor) SwapVisualEnds() {
	ed.win().Cursor, ed.win().visualStart = ed.win().visualStart, ed.win().Cursor
	ed.win().Col, ed.win().visualStartCol = ed.win().visualStartCol, ed.win().Col
	ed.refresh()
}

// VisualDelete deletes the visual selection (d)
func (ed *Editor) VisualDelete(app *tui.App) {
	ed.saveUndo()
	r := ed.VisualRange()

	switch ed.win().visualMode {
	case VisualLine:
		if r.End.Line-r.Start.Line+1 >= len(ed.buf().Lines) {
			ed.buf().Lines = []string{""}
			ed.win().Cursor = 0
		} else {
			ed.buf().Lines = append(ed.buf().Lines[:r.Start.Line], ed.buf().Lines[r.End.Line+1:]...)
			ed.win().Cursor = min(r.Start.Line, len(ed.buf().Lines)-1)
		}
		ed.win().Col = min(ed.win().Col, max(0, len(ed.buf().Lines[ed.win().Cursor])-1))

	case VisualBlock:
		// Block mode: delete rectangular region (same columns on each line)
		startCol := min(ed.win().visualStartCol, ed.win().Col)
		endCol := max(ed.win().visualStartCol, ed.win().Col) + 1
		yankRegister = ed.extractBlock(r.Start.Line, r.End.Line, startCol, endCol)
		ed.deleteBlock(r.Start.Line, r.End.Line, startCol, endCol)
		ed.win().Cursor = r.Start.Line
		ed.win().Col = min(startCol, max(0, len(ed.buf().Lines[ed.win().Cursor])-1))

	default: // VisualChar
		yankRegister = ed.extractRange(r)
		ed.deleteRange(r)
	}
	ed.exitVisualMode(app)
}

// VisualChange deletes selection and enters insert mode (c)
func (ed *Editor) VisualChange(app *tui.App) {
	ed.saveUndo()
	r := ed.VisualRange()

	switch ed.win().visualMode {
	case VisualLine:
		fullLineRange := Range{
			Start: Pos{Line: r.Start.Line, Col: 0},
			End:   Pos{Line: r.End.Line, Col: len(ed.buf().Lines[r.End.Line])},
		}
		yankRegister = ed.extractRange(fullLineRange)
		if r.End.Line-r.Start.Line+1 >= len(ed.buf().Lines) {
			ed.buf().Lines = []string{""}
			ed.win().Cursor = 0
		} else {
			ed.buf().Lines = append(ed.buf().Lines[:r.Start.Line], ed.buf().Lines[r.End.Line+1:]...)
			ed.win().Cursor = min(r.Start.Line, len(ed.buf().Lines)-1)
		}
		// Insert a blank line to type on
		newLines := make([]string, len(ed.buf().Lines)+1)
		copy(newLines[:ed.win().Cursor], ed.buf().Lines[:ed.win().Cursor])
		newLines[ed.win().Cursor] = ""
		copy(newLines[ed.win().Cursor+1:], ed.buf().Lines[ed.win().Cursor:])
		ed.buf().Lines = newLines
		ed.win().Col = 0

	case VisualBlock:
		// Block mode: delete rectangular region and enter block insert mode
		startCol := min(ed.win().visualStartCol, ed.win().Col)
		endCol := max(ed.win().visualStartCol, ed.win().Col) + 1
		yankRegister = ed.extractBlock(r.Start.Line, r.End.Line, startCol, endCol)
		ed.deleteBlock(r.Start.Line, r.End.Line, startCol, endCol)

		// Set up block insert for the changed region
		ed.blockInsertLines = []int{}
		for i := r.Start.Line; i <= r.End.Line; i++ {
			ed.blockInsertLines = append(ed.blockInsertLines, i)
		}
		ed.blockInsertCol = startCol
		ed.win().Cursor = r.Start.Line
		ed.win().Col = startCol
		ed.Mode = "NORMAL"
		app.Pop()
		ed.updateDisplay()
		ed.enterBlockInsertMode(app)
		return

	default: // VisualChar
		yankRegister = ed.extractRange(r)
		ed.deleteRange(r)
	}

	ed.Mode = "NORMAL"
	app.Pop()
	ed.updateDisplay()
	ed.enterInsertMode(app)
}

// VisualYank yanks the visual selection (y)
func (ed *Editor) VisualYank(app *tui.App) {
	r := ed.VisualRange()

	switch ed.win().visualMode {
	case VisualLine:
		var yanked string
		for i := r.Start.Line; i <= r.End.Line; i++ {
			yanked += ed.buf().Lines[i]
			if i < r.End.Line {
				yanked += "\n"
			}
		}
		yankRegister = yanked
		ed.StatusLine = fmt.Sprintf("Yanked %d lines", r.End.Line-r.Start.Line+1)

	case VisualBlock:
		startCol := min(ed.win().visualStartCol, ed.win().Col)
		endCol := max(ed.win().visualStartCol, ed.win().Col) + 1
		yankRegister = ed.extractBlock(r.Start.Line, r.End.Line, startCol, endCol)
		ed.StatusLine = fmt.Sprintf("Yanked block %dx%d", r.End.Line-r.Start.Line+1, endCol-startCol)

	default: // VisualChar
		yankRegister = ed.extractRange(r)
		ed.StatusLine = fmt.Sprintf("Yanked %d chars", len(yankRegister))
	}

	ed.exitVisualMode(app)
}

// VisualExpandToTextObject expands selection to cover a text object
func (ed *Editor) VisualExpandToTextObject(r Range) {
	if r.Start.Line >= 0 {
		ed.win().visualStart = r.Start.Line
		ed.win().visualStartCol = r.Start.Col
		ed.win().Cursor = r.End.Line
		ed.win().Col = max(0, r.End.Col-1)
		ed.win().visualMode = VisualChar
		ed.updateDisplay()
		ed.updateCursor()
	}
}

// VisualExpandToWordObject expands selection to cover a word text object
func (ed *Editor) VisualExpandToWordObject(start, end int) {
	if start < end {
		ed.win().visualStart = ed.win().Cursor
		ed.win().visualStartCol = start
		ed.win().Col = end - 1
		ed.win().visualMode = VisualChar
		ed.updateDisplay()
		ed.updateCursor()
	}
}

// VisualBlockInsert inserts text at the start of each line in the block (I)
// After exiting insert mode, the typed text is replicated to all lines
func (ed *Editor) VisualBlockInsert(app *tui.App) {
	if ed.win().visualMode != VisualBlock {
		// Fall back to regular insert at line start for non-block modes
		ed.exitVisualMode(app)
		ed.LineStart()
		ed.enterInsertMode(app)
		return
	}

	// Save the block selection info before exiting visual mode
	startLine := min(ed.win().visualStart, ed.win().Cursor)
	endLine := max(ed.win().visualStart, ed.win().Cursor)
	insertCol := min(ed.win().visualStartCol, ed.win().Col)

	// Store for after insert mode completes
	ed.blockInsertLines = []int{}
	for i := startLine; i <= endLine; i++ {
		ed.blockInsertLines = append(ed.blockInsertLines, i)
	}
	ed.blockInsertCol = insertCol
	ed.blockInsertStart = len(ed.buf().Lines[startLine]) // Will track where insert started

	ed.exitVisualMode(app)
	ed.win().Cursor = startLine
	ed.win().Col = insertCol
	ed.updateDisplay()
	ed.enterBlockInsertMode(app)
}

// VisualBlockAppend appends text at the end of each line in the block (A)
// After exiting insert mode, the typed text is replicated to all lines
func (ed *Editor) VisualBlockAppend(app *tui.App) {
	if ed.win().visualMode != VisualBlock {
		// Fall back to regular append at line end for non-block modes
		ed.exitVisualMode(app)
		ed.LineEnd()
		ed.Append()
		return
	}

	// Save the block selection info before exiting visual mode
	startLine := min(ed.win().visualStart, ed.win().Cursor)
	endLine := max(ed.win().visualStart, ed.win().Cursor)
	appendCol := max(ed.win().visualStartCol, ed.win().Col) + 1

	// Store for after insert mode completes
	ed.blockInsertLines = []int{}
	for i := startLine; i <= endLine; i++ {
		ed.blockInsertLines = append(ed.blockInsertLines, i)
	}
	ed.blockInsertCol = appendCol
	ed.blockInsertStart = len(ed.buf().Lines[startLine])

	ed.exitVisualMode(app)
	ed.win().Cursor = startLine
	// Position cursor at the append column (may need padding)
	line := ed.buf().Lines[startLine]
	if appendCol > len(line) {
		ed.win().Col = len(line)
	} else {
		ed.win().Col = appendCol
	}
	ed.updateDisplay()
	ed.enterBlockInsertMode(app)
}

// =============================================================================
// Command Dispatcher - reflects on Editor methods to execute commands
// =============================================================================

// ExecCommand executes a command by name with string arguments.
// Uses reflection to find matching Editor method and convert args.
func (ed *Editor) ExecCommand(name string, args []string) error {
	// Find method by case-insensitive match
	method, methodName := ed.findMethod(name)
	if !method.IsValid() {
		return fmt.Errorf("unknown command: %s", name)
	}
	_ = methodName // for future logging/help

	methodType := method.Type()
	numIn := methodType.NumIn()

	if len(args) < numIn {
		return fmt.Errorf("%s requires %d argument(s)", name, numIn)
	}

	// Convert string args to method parameter types
	in := make([]reflect.Value, numIn)
	for i := 0; i < numIn; i++ {
		paramType := methodType.In(i)
		converted, err := convertArg(args[i], paramType)
		if err != nil {
			return fmt.Errorf("argument %d: %v", i+1, err)
		}
		in[i] = converted
	}

	method.Call(in)
	return nil
}

// findMethod finds an Editor method by case-insensitive name match
func (ed *Editor) findMethod(name string) (reflect.Value, string) {
	nameLower := strings.ToLower(name)
	editorType := reflect.TypeOf(ed)
	editorVal := reflect.ValueOf(ed)

	for i := 0; i < editorType.NumMethod(); i++ {
		method := editorType.Method(i)
		if strings.ToLower(method.Name) == nameLower {
			return editorVal.Method(i), method.Name
		}
	}
	return reflect.Value{}, ""
}

// convertArg converts a string argument to the target type
func convertArg(s string, t reflect.Type) (reflect.Value, error) {
	switch t.Kind() {
	case reflect.Int:
		n, err := strconv.Atoi(s)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("expected integer, got %q", s)
		}
		return reflect.ValueOf(n), nil
	case reflect.Int32: // rune
		if len(s) == 0 {
			return reflect.Value{}, fmt.Errorf("expected character, got empty string")
		}
		return reflect.ValueOf(rune(s[0])), nil
	case reflect.String:
		return reflect.ValueOf(s), nil
	case reflect.Bool:
		return reflect.ValueOf(s == "true" || s == "1" || s == "yes"), nil
	default:
		return reflect.Value{}, fmt.Errorf("unsupported parameter type: %v", t)
	}
}

// =============================================================================
// Command Completion (Wildmenu)
// =============================================================================

// harvestCommands builds the list of available commands from Editor methods.
// Called once at startup.
func (ed *Editor) harvestCommands() {
	editorType := reflect.TypeOf(ed)
	var commands []string

	for i := 0; i < editorType.NumMethod(); i++ {
		method := editorType.Method(i)
		// Skip internal methods (those with unexported-style names or special prefixes)
		name := method.Name
		if len(name) == 0 {
			continue
		}
		// Include methods that look like commands (start with uppercase, no weird prefixes)
		// Exclude obvious internal methods
		switch name {
		case "Cursor", "SetCursor", "SetCursorQuiet":
			continue // internal cursor management
		}
		commands = append(commands, strings.ToLower(name))
	}
	sort.Strings(commands)
	ed.cmdCompletions = commands
}

// updateCompletions filters completions based on current input.
func (ed *Editor) updateCompletions() {
	prefix := strings.ToLower(ed.cmdLineInput)
	if prefix == "" {
		ed.cmdMatches = nil
		ed.cmdCompletionActive = false
		ed.cmdWildmenuSpans = nil
		return
	}

	var matches []string
	for _, cmd := range ed.cmdCompletions {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	ed.cmdMatches = matches
	ed.cmdMatchSelected = 0
	ed.cmdCompletionActive = len(matches) > 0
	ed.buildWildmenuSpans()
}

// completeNext cycles to next completion match.
func (ed *Editor) completeNext() {
	if len(ed.cmdMatches) == 0 {
		return
	}
	// If current input doesn't match selection, apply first (don't advance)
	if ed.cmdLineInput != ed.cmdMatches[ed.cmdMatchSelected] {
		ed.applyCmdCompletion()
		return
	}
	// Already showing a completion, cycle to next
	ed.cmdMatchSelected = (ed.cmdMatchSelected + 1) % len(ed.cmdMatches)
	ed.applyCmdCompletion()
}

// completePrev cycles to previous completion match.
func (ed *Editor) completePrev() {
	if len(ed.cmdMatches) == 0 {
		return
	}
	// If current input doesn't match selection, apply first (don't advance)
	if ed.cmdLineInput != ed.cmdMatches[ed.cmdMatchSelected] {
		ed.applyCmdCompletion()
		return
	}
	// Already showing a completion, cycle to previous
	ed.cmdMatchSelected = (ed.cmdMatchSelected - 1 + len(ed.cmdMatches)) % len(ed.cmdMatches)
	ed.applyCmdCompletion()
}

// applyCmdCompletion applies the selected completion to the command line.
func (ed *Editor) applyCmdCompletion() {
	if ed.cmdMatchSelected < len(ed.cmdMatches) {
		ed.cmdLineInput = ed.cmdMatches[ed.cmdMatchSelected]
		ed.StatusLine = ed.cmdLinePrompt + ed.cmdLineInput
		ed.updateCmdLineCursor()
		ed.buildWildmenuSpans()
	}
}

// buildWildmenuSpans builds the visual representation of the wildmenu.
// Shows matches horizontally with the selected one highlighted.
func (ed *Editor) buildWildmenuSpans() {
	if !ed.cmdCompletionActive || len(ed.cmdMatches) == 0 {
		ed.cmdWildmenuSpans = nil
		return
	}

	// Style for normal and selected items
	normalStyle := tui.Style{}
	selectedStyle := tui.Style{Attr: tui.AttrInverse}

	// Build spans
	var spans []tui.Span
	for i, match := range ed.cmdMatches {
		if i > 0 {
			spans = append(spans, tui.Span{Text: " "})
		}
		style := normalStyle
		if i == ed.cmdMatchSelected {
			style = selectedStyle
		}
		spans = append(spans, tui.Span{Text: match, Style: style})
	}
	ed.cmdWildmenuSpans = spans
}

// EditorState captures state for undo/redo
type EditorState struct {
	Lines  []string
	Cursor int
	Col    int
}

func main() {
	initTextObjectDefs()

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
		marks:    make(map[rune]Pos),
	}
	win := &Window{
		buffer:      buf,
		renderedMin: -1,
		renderedMax: -1,
	}

	// Create split tree with single window as root
	root := &SplitNode{Window: win}

	ed := &Editor{
		root:           root,
		focusedWindow:  win,
		Mode:           "NORMAL",
		StatusLine:     "", // empty initially, used for messages
		relativeNumber: true,
		cursorLine:     true,
		showSignColumn: true,
		macros:         make(map[rune]riffkey.Macro),
	}

	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}
	ed.app = app
	ed.harvestCommands() // Build command list for completion
	ed.refreshGitSigns() // Load initial git diff state

	// Initialize viewport and layer
	size := app.Size()
	ed.win().viewportHeight = max(1, size.Height-headerRows-footerRows)
	ed.initLayer(size.Width)

	ed.updateDisplay()

	app.SetView(buildView(ed))

	// Register fuzzy finder as a pushable overlay view
	app.View("fuzzy", buildFuzzyView(ed)).
		Handle("<BS>", func(_ riffkey.Match) { ed.fuzzyBackspace() }).
		Handle("<Up>", func(_ riffkey.Match) { ed.fuzzyUp() }).
		Handle("<C-p>", func(_ riffkey.Match) { ed.fuzzyUp() }).
		Handle("<C-k>", func(_ riffkey.Match) { ed.fuzzyUp() }).
		Handle("<Down>", func(_ riffkey.Match) { ed.fuzzyDown() }).
		Handle("<C-n>", func(_ riffkey.Match) { ed.fuzzyDown() }).
		Handle("<C-j>", func(_ riffkey.Match) { ed.fuzzyDown() }).
		Handle("<CR>", func(_ riffkey.Match) { ed.fuzzySelect(app) }).
		Handle("<Esc>", func(_ riffkey.Match) { ed.fuzzyCancel(app) }).
		Handle("<C-c>", func(_ riffkey.Match) { ed.fuzzyCancel(app) })

	// Handle text input for fuzzy finder - need to get the router for HandleUnmatched
	if fuzzyRouter, ok := app.ViewRouter("fuzzy"); ok {
		fuzzyRouter.NoCounts()
		fuzzyRouter.HandleUnmatched(func(k riffkey.Key) bool {
			if !ed.fuzzy.Active {
				return false
			}
			if k.Rune != 0 && k.Mod == riffkey.ModNone {
				ed.fuzzy.Query += string(k.Rune)
				ed.fuzzyFilterMatches()
				return true
			}
			return false
		})
	}

	// Start with block cursor in normal mode
	app.ShowCursor(tui.CursorBlock)
	ed.updateCursor()

	// Normal mode handlers - movement actions
	app.Handle("j", func(m riffkey.Match) { ed.Down(m.Count) })
	app.Handle("k", func(m riffkey.Match) { ed.Up(m.Count) })
	app.Handle("h", func(m riffkey.Match) { ed.Left(m.Count) })
	app.Handle("l", func(m riffkey.Match) { ed.Right(m.Count) })
	app.Handle("gg", func(_ riffkey.Match) { ed.BufferStart() })
	app.Handle("G", func(_ riffkey.Match) { ed.BufferEnd() })
	app.Handle("0", func(_ riffkey.Match) { ed.LineStart() })
	app.Handle("$", func(_ riffkey.Match) { ed.LineEnd() })
	app.Handle("^", func(_ riffkey.Match) { ed.FirstNonBlank() })
	app.Handle("_", func(_ riffkey.Match) { ed.FirstNonBlank() })
	app.Handle("w", func(m riffkey.Match) { ed.NextWordStart(m.Count) })
	app.Handle("b", func(m riffkey.Match) { ed.PrevWordStart(m.Count) })
	app.Handle("e", func(m riffkey.Match) { ed.NextWordEnd(m.Count) })

	// Netrw (file explorer) keybindings
	app.Handle("<CR>", func(_ riffkey.Match) {
		if ed.isNetrw() {
			ed.netrwEnter(app)
		}
	})
	app.Handle("-", func(_ riffkey.Match) {
		if ed.isNetrw() {
			ed.netrwUp()
		} else {
			// Normal mode: go up one line and to first non-blank
			ed.Up(1)
			ed.FirstNonBlank()
		}
	})
	app.Handle("gh", func(_ riffkey.Match) {
		if ed.isNetrw() {
			ed.netrwToggleHidden()
		}
	})

	// Fuzzy finder (Ctrl-P like VSCode/Sublime)
	app.Handle("<C-p>", func(_ riffkey.Match) { ed.openFuzzyFinder(app) })

	app.Handle("i", func(_ riffkey.Match) { ed.EnterInsert() })
	app.Handle("a", func(_ riffkey.Match) { ed.Append() })
	app.Handle("A", func(_ riffkey.Match) { ed.AppendLine() })
	app.Handle("I", func(_ riffkey.Match) { ed.InsertLine() })
	app.Handle("o", func(_ riffkey.Match) { ed.OpenBelow() })
	app.Handle("O", func(_ riffkey.Match) { ed.OpenAbove() })

	app.Handle("dd", func(m riffkey.Match) { ed.DeleteLine(m.Count) })
	app.Handle("x", func(m riffkey.Match) { ed.DeleteChar(m.Count) })

	app.Handle("<Esc>", func(_ riffkey.Match) {
		// Already in normal mode, do nothing
	})

	// Register operator + text object combinations (diw, ciw, yaw, etc.)
	registerOperatorTextObjects(app, ed)

	// Paste from yank register
	app.Handle("p", func(_ riffkey.Match) { ed.Paste() })
	app.Handle("P", func(_ riffkey.Match) { ed.PasteBefore() })

	// Undo/Redo
	app.Handle("u", func(_ riffkey.Match) { ed.Undo() })
	app.Handle("<C-r>", func(_ riffkey.Match) { ed.Redo() })

	// Scrolling
	app.Handle("<C-d>", func(_ riffkey.Match) { ed.HalfPageDown() })
	app.Handle("<C-u>", func(_ riffkey.Match) { ed.HalfPageUp() })
	app.Handle("<C-f>", func(_ riffkey.Match) { ed.PageDown() })
	app.Handle("<C-b>", func(_ riffkey.Match) { ed.PageUp() })

	app.Handle("<C-e>", func(_ riffkey.Match) { ed.ScrollLineDown() })
	app.Handle("<C-y>", func(_ riffkey.Match) { ed.ScrollLineUp() })

	app.Handle("<C-l>", func(_ riffkey.Match) { ed.ClearSearchHighlight() })
	app.Handle("<C-a>", func(_ riffkey.Match) { ed.IncrementNumber() })
	app.Handle("<C-x>", func(_ riffkey.Match) { ed.DecrementNumber() })

	app.Handle("<C-o>", func(_ riffkey.Match) { ed.JumpBack() })

	app.Handle("<C-i>", func(_ riffkey.Match) { ed.JumpForward() })

	// f/F/t/T - find character on line
	registerFindChar(app, ed)

	// Visual mode
	app.Handle("v", func(_ riffkey.Match) { ed.EnterVisual() })
	app.Handle("V", func(_ riffkey.Match) { ed.EnterVisualLine() })
	app.Handle("<C-v>", func(_ riffkey.Match) { ed.EnterVisualBlock() })

	// Join lines (J)
	app.Handle("J", func(_ riffkey.Match) { ed.JoinLines() })

	// Replace single char (r)
	app.Handle("r", func(_ riffkey.Match) { ed.promptForChar(app, ed.ReplaceChar) })

	// Repeat last change (.)
	app.Handle(".", func(_ riffkey.Match) { ed.RepeatLast() })

	// ~ toggle case
	app.Handle("~", func(_ riffkey.Match) { ed.ToggleCase() })

	// Macro recording: q{a-z} to start, q to stop
	app.Handle("q", func(_ riffkey.Match) {
		if app.Input().IsRecording() {
			ed.StopMacroRecording(app)
		} else {
			ed.promptForRegister(app, func(reg rune) {
				ed.StartMacroRecording(app, reg)
			})
		}
	})

	// Macro playback: @{a-z} or @@ for last
	app.Handle("@", func(_ riffkey.Match) {
		ed.promptForRegister(app, func(reg rune) {
			ed.PlayMacro(app, reg)
		})
	})

	// Repeat last : command
	app.Handle("@:", func(_ riffkey.Match) {
		if ed.lastColonCmd != "" {
			ed.executeColonCommand(app, ed.lastColonCmd)
		}
	})

	// @@ - repeat last macro (special case)
	app.Handle("@@", func(_ riffkey.Match) {
		if ed.lastMacro != 0 {
			ed.PlayMacro(app, ed.lastMacro)
		}
	})

	// Marks: m{a-z} to set mark
	app.Handle("m", func(_ riffkey.Match) { ed.promptForRegister(app, ed.SetMark) })

	// Marks: '{a-z} to jump to mark line (first non-blank)
	app.Handle("'", func(_ riffkey.Match) { ed.promptForRegister(app, ed.GotoMarkLine) })

	// Marks: `{a-z} to jump to exact mark position
	app.Handle("`", func(_ riffkey.Match) { ed.promptForRegister(app, ed.GotoMark) })

	// Command line mode handlers
	app.Handle(":", func(_ riffkey.Match) { ed.enterCommandMode(app, ":") })
	app.Handle("/", func(_ riffkey.Match) { ed.enterCommandMode(app, "/") })
	app.Handle("?", func(_ riffkey.Match) { ed.enterCommandMode(app, "?") })

	// n/N for search repeat
	app.Handle("n", func(_ riffkey.Match) { ed.SearchNext() })
	app.Handle("N", func(_ riffkey.Match) { ed.SearchPrev() })

	// Debug mode toggle
	app.Handle("<C-g>", func(_ riffkey.Match) { ed.ToggleDebug() })

	// z commands - screen positioning
	app.Handle("zz", func(_ riffkey.Match) { ed.CenterCursor() })
	app.Handle("zt", func(_ riffkey.Match) { ed.CursorToTop() })
	app.Handle("zb", func(_ riffkey.Match) { ed.CursorToBottom() })
	app.Handle("z<CR>", func(_ riffkey.Match) { ed.CursorToTop(); ed.FirstNonBlank() })
	app.Handle("z.", func(_ riffkey.Match) { ed.CenterCursor(); ed.FirstNonBlank() })
	app.Handle("z-", func(_ riffkey.Match) { ed.CursorToBottom(); ed.FirstNonBlank() })
	app.Handle("z+", func(_ riffkey.Match) { ed.NextScreenLine() })
	app.Handle("z^", func(_ riffkey.Match) { ed.PrevScreenLine() })

	// Horizontal scroll commands
	app.Handle("zl", func(_ riffkey.Match) { ed.ScrollRight(1) })
	app.Handle("zh", func(_ riffkey.Match) { ed.ScrollLeft(1) })
	app.Handle("zL", func(_ riffkey.Match) { ed.ScrollRight(ed.win().viewportWidth / 2) })
	app.Handle("zH", func(_ riffkey.Match) { ed.ScrollLeft(ed.win().viewportWidth / 2) })
	app.Handle("zs", func(_ riffkey.Match) { ed.CursorToScreenStart() })
	app.Handle("ze", func(_ riffkey.Match) { ed.CursorToScreenEnd() })

	// H/M/L - move cursor to top/middle/bottom of screen
	app.Handle("H", func(m riffkey.Match) { ed.ScreenTop(m.Count) })
	app.Handle("M", func(_ riffkey.Match) { ed.ScreenMiddle() })
	app.Handle("L", func(m riffkey.Match) { ed.ScreenBottom(m.Count) })

	// Window management: Ctrl-w commands
	app.Handle("<C-w>w", func(_ riffkey.Match) { ed.FocusNextWindow() })
	app.Handle("<C-w>W", func(_ riffkey.Match) { ed.FocusPrevWindow() })
	app.Handle("<C-w>j", func(_ riffkey.Match) { ed.FocusDown() })
	app.Handle("<C-w>k", func(_ riffkey.Match) { ed.FocusUp() })
	app.Handle("<C-w>h", func(_ riffkey.Match) { ed.FocusLeft() })
	app.Handle("<C-w>l", func(_ riffkey.Match) { ed.FocusRight() })
	app.Handle("<C-w>s", func(_ riffkey.Match) { ed.SplitHoriz() })
	app.Handle("<C-w>v", func(_ riffkey.Match) { ed.SplitVert() })
	app.Handle("<C-w>c", func(_ riffkey.Match) { ed.CloseWindow() })
	app.Handle("<C-w>o", func(_ riffkey.Match) { ed.OnlyWindow() })

	// Add hook to normal mode router for display updates.
	// Movement actions are now pure - hooks handle display refresh.
	app.Router().AddOnAfter(func() {
		ed.updateDisplay()
		ed.updateCursor()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func (ed *Editor) enterInsertMode(app *tui.App) {
	ed.Mode = "INSERT"
	ed.win().Col = min(ed.win().Col, len(ed.buf().Lines[ed.win().Cursor]))
	ed.StatusLine = "-- INSERT --"
	ed.updateDisplay()

	// Switch to bar cursor for insert mode
	app.ShowCursor(tui.CursorBar)
	ed.updateCursor()

	// Create insert mode router (NoCounts so digits aren't count prefixes)
	insertRouter := riffkey.NewRouter().Name("insert").NoCounts()

	// TextHandler with OnChange callback for live updates
	th := riffkey.NewTextHandler(&ed.buf().Lines[ed.win().Cursor], &ed.win().Col)
	th.OnChange = func(_ string) {
		ed.updateDisplay()
		ed.updateCursor()
	}

	// Helper to rebind TextHandler and update display after buffer changes
	rebindAndRefresh := func() {
		th.Value = &ed.buf().Lines[ed.win().Cursor]
		ed.updateDisplay()
		ed.updateCursor()
	}

	// Esc exits insert mode
	insertRouter.Handle("<Esc>", func(_ riffkey.Match) { ed.exitInsertMode(app) })

	// Enter creates a new line
	insertRouter.Handle("<CR>", func(_ riffkey.Match) { ed.InsertNewline(); rebindAndRefresh() })

	// C-u: delete from cursor to start of line
	insertRouter.Handle("<C-u>", func(_ riffkey.Match) { ed.DeleteToLineStart(); rebindAndRefresh() })

	// C-k: delete from cursor to end of line
	insertRouter.Handle("<C-k>", func(_ riffkey.Match) { ed.DeleteToLineEnd(); rebindAndRefresh() })

	// C-t: indent line (add tab at start)
	insertRouter.Handle("<C-t>", func(_ riffkey.Match) { ed.IndentLine(); rebindAndRefresh() })

	// C-d: unindent line (remove leading whitespace)
	insertRouter.Handle("<C-d>", func(_ riffkey.Match) { ed.UnindentLine(); rebindAndRefresh() })

	// C-o: execute one normal mode command then return to insert
	insertRouter.Handle("<C-o>", func(_ riffkey.Match) {
		ed.Mode = "INSERT (i)"
		ed.StatusLine = "-- (insert) --"
		ed.updateDisplay()

		// Clone normal mode router with after-hook to return to insert
		oneShot := app.Router().Clone().Name("insert-normal").OnAfter(func() {
			th.Value = &ed.buf().Lines[ed.win().Cursor] // Rebind in case line changed
			ed.Mode = "INSERT"
			ed.StatusLine = "-- INSERT --"
			app.Pop()
		})

		// Esc just returns to insert (empty handler, after-middleware handles cleanup)
		oneShot.Handle("<Esc>", func(_ riffkey.Match) {})

		// Unknown keys return to insert without doing anything
		oneShot.HandleUnmatched(func(_ riffkey.Key) bool { return true })

		app.Push(oneShot)
	})

	// Wire up the text handler for unmatched keys
	insertRouter.HandleUnmatched(th.HandleKey)

	// Push the insert router - takes over input
	app.Push(insertRouter)
}

func (ed *Editor) exitInsertMode(app *tui.App) {
	ed.Mode = "NORMAL"
	ed.StatusLine = ""

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

// enterBlockInsertMode is like insert mode but replicates text to all block lines on exit
func (ed *Editor) enterBlockInsertMode(app *tui.App) {
	ed.Mode = "INSERT"
	ed.win().Col = min(ed.win().Col, len(ed.buf().Lines[ed.win().Cursor]))
	ed.StatusLine = "-- INSERT (block) --"
	ed.updateDisplay()

	// Track the first line's content before and during insert
	firstLine := ed.blockInsertLines[0]
	originalContent := ed.buf().Lines[firstLine]
	insertStartCol := ed.blockInsertCol

	// Switch to bar cursor
	app.ShowCursor(tui.CursorBar)
	ed.updateCursor()

	// Create insert mode router
	insertRouter := riffkey.NewRouter().Name("block-insert").NoCounts()

	// TextHandler for the first line
	th := riffkey.NewTextHandler(&ed.buf().Lines[firstLine], &ed.win().Col)
	th.OnChange = func(_ string) {
		ed.updateDisplay()
		ed.updateCursor()
	}

	rebindAndRefresh := func() {
		th.Value = &ed.buf().Lines[ed.win().Cursor]
		ed.updateDisplay()
		ed.updateCursor()
	}

	// Esc exits and replicates to all lines
	insertRouter.Handle("<Esc>", func(_ riffkey.Match) {
		ed.exitBlockInsertMode(app, originalContent, insertStartCol)
	})

	// Enter creates new line (exits block mode for simplicity)
	insertRouter.Handle("<CR>", func(_ riffkey.Match) {
		ed.InsertNewline()
		rebindAndRefresh()
		// Exit block mode - newlines don't make sense in block insert
		ed.blockInsertLines = nil
	})

	// Standard insert mode bindings
	insertRouter.Handle("<C-u>", func(_ riffkey.Match) { ed.DeleteToLineStart(); rebindAndRefresh() })
	insertRouter.Handle("<C-k>", func(_ riffkey.Match) { ed.DeleteToLineEnd(); rebindAndRefresh() })

	insertRouter.HandleUnmatched(th.HandleKey)
	app.Push(insertRouter)
}

func (ed *Editor) exitBlockInsertMode(app *tui.App, originalContent string, insertStartCol int) {
	// Calculate what was inserted by comparing first line to original
	if len(ed.blockInsertLines) > 1 {
		firstLine := ed.blockInsertLines[0]
		newContent := ed.buf().Lines[firstLine]

		// Find the inserted text (text that was added at insertStartCol)
		var insertedText string
		if len(newContent) > len(originalContent) {
			// Determine what was inserted
			if insertStartCol <= len(originalContent) {
				// Text was inserted in the middle or at insertStartCol
				insertedLen := len(newContent) - len(originalContent)
				if insertStartCol+insertedLen <= len(newContent) {
					insertedText = newContent[insertStartCol : insertStartCol+insertedLen]
				}
			}
		}

		// Replicate to other lines if we have text to insert
		if insertedText != "" {
			for _, lineIdx := range ed.blockInsertLines[1:] {
				if lineIdx < len(ed.buf().Lines) {
					line := ed.buf().Lines[lineIdx]
					// Pad with spaces if line is shorter than insert column
					for len(line) < insertStartCol {
						line += " "
					}
					// Insert the text at the same column
					ed.buf().Lines[lineIdx] = line[:insertStartCol] + insertedText + line[insertStartCol:]
				}
			}
		}
	}

	// Clear block insert state
	ed.blockInsertLines = nil

	// Standard exit insert mode
	ed.Mode = "NORMAL"
	ed.StatusLine = ""

	if ed.win().Col > 0 && ed.win().Col >= len(ed.buf().Lines[ed.win().Cursor]) {
		ed.win().Col = max(0, len(ed.buf().Lines[ed.win().Cursor])-1)
	}

	app.ShowCursor(tui.CursorBlock)
	ed.updateDisplay()
	ed.updateCursor()
	app.Pop()
}

func (ed *Editor) updateCursor() {
	// Calculate screen position relative to viewport
	screenY := headerRows + (ed.win().Cursor - ed.win().topLine)
	// Adjust for horizontal scroll
	screenX := ed.win().lineNumWidth + ed.win().Col - ed.win().leftCol

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

// scrollCursorTo positions the viewport so cursor is at the given screen row
func (ed *Editor) scrollCursorTo(screenRow int) {
	// Calculate new topLine so cursor appears at screenRow
	newTop := ed.win().Cursor - screenRow

	// Clamp to valid range
	maxTop := len(ed.buf().Lines) - ed.win().viewportHeight
	if maxTop < 0 {
		maxTop = 0
	}
	if newTop < 0 {
		newTop = 0
	}
	if newTop > maxTop {
		newTop = maxTop
	}

	ed.win().topLine = newTop
	ed.updateDisplay()
	ed.updateCursor()
}

// addJump adds the current position to the jump list
func (ed *Editor) addJump() {
	pos := JumpPos{Line: ed.win().Cursor, Col: ed.win().Col}

	// Don't add duplicate of current position
	if len(ed.win().jumpList) > 0 && ed.win().jumpIndex > 0 {
		last := ed.win().jumpList[ed.win().jumpIndex-1]
		if last.Line == pos.Line && last.Col == pos.Col {
			return
		}
	}

	// Truncate any forward history when adding new jump
	if ed.win().jumpIndex < len(ed.win().jumpList) {
		ed.win().jumpList = ed.win().jumpList[:ed.win().jumpIndex]
	}

	ed.win().jumpList = append(ed.win().jumpList, pos)
	ed.win().jumpIndex = len(ed.win().jumpList)

	// Limit jump list size
	if len(ed.win().jumpList) > 100 {
		ed.win().jumpList = ed.win().jumpList[1:]
		ed.win().jumpIndex--
	}
}

// jumpBack moves to previous position in jump list (C-o)
func (ed *Editor) jumpBack() {
	if ed.win().jumpIndex <= 0 || len(ed.win().jumpList) == 0 {
		return
	}

	// Save current position if at end of list
	if ed.win().jumpIndex == len(ed.win().jumpList) {
		ed.addJump()
		ed.win().jumpIndex-- // Back up past the one we just added
	}

	ed.win().jumpIndex--
	pos := ed.win().jumpList[ed.win().jumpIndex]
	ed.win().Cursor = min(pos.Line, len(ed.buf().Lines)-1)
	ed.win().Col = min(pos.Col, len(ed.buf().Lines[ed.win().Cursor]))
	ed.ensureCursorVisible()
	ed.updateDisplay()
	ed.updateCursor()
}

// jumpForward moves to next position in jump list (C-i)
func (ed *Editor) jumpForward() {
	if ed.win().jumpIndex >= len(ed.win().jumpList)-1 {
		return
	}

	ed.win().jumpIndex++
	pos := ed.win().jumpList[ed.win().jumpIndex]
	ed.win().Cursor = min(pos.Line, len(ed.buf().Lines)-1)
	ed.win().Col = min(pos.Col, len(ed.buf().Lines[ed.win().Cursor]))
	ed.ensureCursorVisible()
	ed.updateDisplay()
	ed.updateCursor()
}

// modifyNumber increments or decrements the number under/after cursor
func (ed *Editor) modifyNumber(delta int) {
	// Debounce to prevent key repeat flooding
	if time.Since(ed.lastNumberModify) < 100*time.Millisecond {
		return
	}
	ed.lastNumberModify = time.Now()

	line := ed.buf().Lines[ed.win().Cursor]
	if len(line) == 0 {
		return
	}

	// Helper to check if char is a digit
	isDigit := func(c byte) bool { return c >= '0' && c <= '9' }

	// Find number: scan right from cursor to find first digit
	firstDigit := -1
	for i := ed.win().Col; i < len(line); i++ {
		if isDigit(line[i]) {
			firstDigit = i
			break
		}
	}

	// If no digit found scanning right, scan left
	if firstDigit < 0 {
		for i := ed.win().Col - 1; i >= 0; i-- {
			if isDigit(line[i]) {
				firstDigit = i
				break
			}
		}
	}

	if firstDigit < 0 {
		return // No number found
	}

	// Expand left to find start of number (include all digits)
	start := firstDigit
	for start > 0 && isDigit(line[start-1]) {
		start--
	}
	// Check for negative sign
	if start > 0 && line[start-1] == '-' {
		start--
	}

	// Expand right to find end of number
	end := firstDigit + 1
	for end < len(line) && isDigit(line[end]) {
		end++
	}

	// Parse and modify
	numStr := line[start:end]
	var num int
	_, err := fmt.Sscanf(numStr, "%d", &num)
	if err != nil {
		return
	}
	num += delta

	// Replace in line
	ed.saveUndo()
	newNum := fmt.Sprintf("%d", num)
	ed.buf().Lines[ed.win().Cursor] = line[:start] + newNum + line[end:]
	ed.win().Col = max(0, start+len(newNum)-1)
	ed.updateDisplay()
	ed.updateCursor()
}

// scrollRight scrolls the view right by n columns
func (ed *Editor) scrollRight(n int) {
	ed.win().leftCol += n
	ed.ensureHorizontalCursorVisible()
	ed.updateDisplay()
	ed.updateCursor()
}

// scrollLeft scrolls the view left by n columns
func (ed *Editor) scrollLeft(n int) {
	ed.win().leftCol = max(0, ed.win().leftCol-n)
	ed.ensureHorizontalCursorVisible()
	ed.updateDisplay()
	ed.updateCursor()
}

// ensureHorizontalCursorVisible adjusts leftCol if cursor is off screen
func (ed *Editor) ensureHorizontalCursorVisible() {
	textWidth := ed.win().viewportWidth - ed.win().lineNumWidth
	if textWidth <= 0 {
		textWidth = 1
	}

	// Cursor position relative to text area
	cursorScreenPos := ed.win().Col - ed.win().leftCol

	// Scroll right if cursor is past right edge
	if cursorScreenPos >= textWidth {
		ed.win().leftCol = ed.win().Col - textWidth + 1
	}

	// Scroll left if cursor is before left edge
	if cursorScreenPos < 0 {
		ed.win().leftCol = ed.win().Col
	}
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
	cursorLineNumStyle = tui.Style{FG: tui.Color{Mode: tui.Color16, Index: 3}}                                 // Yellow for current line number
	cursorLineStyle    = tui.Style{BG: tui.Color{Mode: tui.Color256, Index: 236}}                              // Subtle dark gray background for cursorline
	tildeStyle         = tui.Style{FG: tui.Color{Mode: tui.Color16, Index: 4}}                                 // Blue for ~ lines
	statusBarStyle     = tui.Style{Attr: tui.AttrInverse}                                                      // Inverse video like vim
	searchHighlight    = tui.Style{BG: tui.Color{Mode: tui.Color16, Index: 3}}                                 // Yellow background for search matches
	gitAddedStyle      = tui.Style{FG: tui.Color{Mode: tui.Color16, Index: 2}}                                 // Green for added lines
	gitModifiedStyle   = tui.Style{FG: tui.Color{Mode: tui.Color16, Index: 3}}                                 // Yellow for modified lines
	gitRemovedStyle    = tui.Style{FG: tui.Color{Mode: tui.Color16, Index: 1}}                                 // Red for removed lines
)

// refreshGitSigns updates the git change signs for a buffer by running git diff
func (ed *Editor) refreshGitSigns() {
	buf := ed.buf()
	if buf.FileName == "" {
		return
	}

	// Initialize signs array to correct size
	buf.gitSigns = make([]GitSign, len(buf.Lines))

	// Check if we're in a git repo by finding .git directory
	dir := filepath.Dir(buf.FileName)
	if dir == "" {
		dir = "."
	}

	// Try to get git root
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		// Not in a git repo
		return
	}

	// Get the diff between HEAD and working directory for this file
	// Use --no-ext-diff to avoid custom diff tools
	cmd = exec.Command("git", "-C", dir, "diff", "--no-ext-diff", "--no-color", "-U0", "HEAD", "--", buf.FileName)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		// File might be untracked - try checking with git status
		cmd = exec.Command("git", "-C", dir, "status", "--porcelain", buf.FileName)
		var statusOut bytes.Buffer
		cmd.Stdout = &statusOut
		if err := cmd.Run(); err == nil {
			status := strings.TrimSpace(statusOut.String())
			if strings.HasPrefix(status, "??") || strings.HasPrefix(status, "A ") {
				// Untracked or newly added file - mark all lines as added
				for i := range buf.gitSigns {
					buf.gitSigns[i] = GitSignAdded
				}
			}
		}
		return
	}

	// Parse unified diff output
	// Format: @@ -oldstart,oldcount +newstart,newcount @@
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "@@") {
			continue
		}

		// Parse hunk header: @@ -oldstart,oldcount +newstart,newcount @@
		// Examples: @@ -1,3 +1,5 @@ or @@ -1 +1,2 @@ or @@ -0,0 +1,10 @@
		parts := strings.Split(line, " ")
		if len(parts) < 3 {
			continue
		}

		// Parse old range (-oldstart,oldcount)
		oldPart := strings.TrimPrefix(parts[1], "-")
		_, oldCount := parseRange(oldPart)

		// Parse new range (+newstart,newcount)
		newPart := strings.TrimPrefix(parts[2], "+")
		newStart, newCount := parseRange(newPart)

		// Determine the type of change
		if oldCount == 0 {
			// Pure addition - no lines removed, just added
			for i := newStart; i < newStart+newCount && i <= len(buf.gitSigns); i++ {
				if i > 0 && i <= len(buf.gitSigns) {
					buf.gitSigns[i-1] = GitSignAdded
				}
			}
		} else if newCount == 0 {
			// Pure deletion - lines removed
			// Mark the line before the deletion (or first line if at start)
			deletionMarker := newStart
			if deletionMarker > 0 && deletionMarker <= len(buf.gitSigns) {
				buf.gitSigns[deletionMarker-1] = GitSignRemoved
			} else if len(buf.gitSigns) > 0 {
				buf.gitSigns[0] = GitSignRemoved
			}
		} else {
			// Modification - some lines changed
			for i := newStart; i < newStart+newCount && i <= len(buf.gitSigns); i++ {
				if i > 0 && i <= len(buf.gitSigns) {
					buf.gitSigns[i-1] = GitSignModified
				}
			}
		}
	}
}

// parseRange parses "start,count" or "start" from diff output
func parseRange(s string) (start, count int) {
	parts := strings.Split(s, ",")
	start, _ = strconv.Atoi(parts[0])
	if len(parts) > 1 {
		count, _ = strconv.Atoi(parts[1])
	} else {
		count = 1
	}
	return
}

// getGitSignForLine returns the sign character and style for a line
func (ed *Editor) getGitSignForLine(lineIdx int) (string, tui.Style) {
	return ed.getGitSignForLineBuffer(ed.buf(), lineIdx)
}

// getGitSignForLineBuffer returns the sign character and style for a line in a specific buffer
func (ed *Editor) getGitSignForLineBuffer(buf *Buffer, lineIdx int) (string, tui.Style) {
	if lineIdx >= len(buf.gitSigns) {
		return " ", tui.Style{}
	}

	switch buf.gitSigns[lineIdx] {
	case GitSignAdded:
		return "│", gitAddedStyle // Green vertical bar for added
	case GitSignModified:
		return "│", gitModifiedStyle // Yellow vertical bar for modified
	case GitSignRemoved:
		return "▁", gitRemovedStyle // Red underscore for deleted below
	default:
		return " ", tui.Style{}
	}
}

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

// applyCursorLineStyle applies the cursorline background to spans
func (ed *Editor) applyCursorLineStyle(spans []tui.Span) []tui.Span {
	result := make([]tui.Span, len(spans))
	for i, span := range spans {
		// Merge cursorline background with existing style
		// Keep foreground and attributes, add cursorline background if no background set
		newStyle := span.Style
		if newStyle.BG.Mode == 0 { // No background set
			newStyle.BG = cursorLineStyle.BG
		}
		result[i] = tui.Span{Text: span.Text, Style: newStyle}
	}
	return result
}

// updateStatusBar builds the vim-style status bar
func (ed *Editor) updateStatusBar() {
	// Use stored viewport width if set, otherwise full screen width
	width := ed.win().viewportWidth
	if width == 0 {
		width = ed.app.Size().Width
	}

	// Left side: filename (and debug stats if enabled)
	filename := ed.buf().FileName
	if ed.isNetrw() {
		filename = "[netrw] " + filename
	}
	left := " " + filename
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
	// Add sign column width if enabled (2 chars: sign + space)
	if ed.showSignColumn {
		w.lineNumWidth += 2
	}

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

	// Calculate format widths
	signWidth := 0
	if ed.showSignColumn {
		signWidth = 2 // sign char + space
	}
	lineNumOnlyWidth := w.lineNumWidth - signWidth
	lineNumFmt := fmt.Sprintf("%%%dd ", lineNumOnlyWidth-1)
	tildeFmt := fmt.Sprintf("%%%ds ", w.lineNumWidth-1)

	var spans []tui.Span

	if lineIdx < len(w.buffer.Lines) {
		// Content line
		line := w.buffer.Lines[lineIdx]
		isCursorLine := lineIdx == w.Cursor

		// Add sign column if enabled
		if ed.showSignColumn {
			sign, signStyle := ed.getGitSignForLineBuffer(w.buffer, lineIdx)
			spans = append(spans, tui.Span{Text: sign + " ", Style: signStyle})
		}

		// Calculate displayed line number (absolute or relative)
		var displayNum int
		if ed.relativeNumber && !isCursorLine {
			// Relative: show distance from cursor
			displayNum = abs(lineIdx - w.Cursor)
		} else {
			// Absolute: show actual line number (1-indexed)
			displayNum = lineIdx + 1
		}
		lineNum := fmt.Sprintf(lineNumFmt, displayNum)

		numStyle := lineNumStyle
		if isCursorLine {
			numStyle = cursorLineNumStyle
		}

		// For the focused window in visual mode, use visual spans
		if ed.Mode == "VISUAL" && ed.win() == w {
			spans = append(spans, tui.Span{Text: lineNum, Style: numStyle})
			spans = append(spans, ed.getVisualSpans(lineIdx, line)...)
		} else {
			// Use search highlighting
			contentSpans := ed.highlightSearchMatches(line)
			// Apply cursorline background if enabled and this is the cursor line
			if ed.cursorLine && isCursorLine {
				contentSpans = ed.applyCursorLineStyle(contentSpans)
			}
			spans = append(spans, tui.Span{Text: lineNum, Style: numStyle})
			spans = append(spans, contentSpans...)
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

// splitHorizontalWithBuffer creates a horizontal split with a specific buffer
func (ed *Editor) splitHorizontalWithBuffer(buf *Buffer) {
	currentNode := ed.root.FindWindow(ed.focusedWindow)
	if currentNode == nil {
		return
	}

	totalHeight := ed.focusedWindow.viewportHeight
	halfHeight := max(1, totalHeight/2)
	ed.focusedWindow.viewportHeight = halfHeight

	newWin := &Window{
		buffer:         buf,
		Cursor:         0,
		Col:            0,
		topLine:        0,
		viewportHeight: totalHeight - halfHeight,
		viewportWidth:  ed.focusedWindow.viewportWidth,
		renderedMin:    -1,
		renderedMax:    -1,
	}

	ed.initWindowLayer(newWin, newWin.viewportWidth)

	newWindowNode := &SplitNode{Window: newWin}
	currentWindowNode := &SplitNode{Window: ed.focusedWindow}

	currentNode.Direction = SplitHorizontal
	currentNode.Window = nil
	currentNode.Children = [2]*SplitNode{currentWindowNode, newWindowNode}
	currentWindowNode.Parent = currentNode
	newWindowNode.Parent = currentNode

	// Focus the new window
	ed.focusedWindow = newWin

	ed.app.SetView(buildView(ed))
	ed.updateAllWindows()
	ed.StatusLine = ""
}

// splitVerticalWithBuffer creates a vertical split with a specific buffer
func (ed *Editor) splitVerticalWithBuffer(buf *Buffer) {
	currentNode := ed.root.FindWindow(ed.focusedWindow)
	if currentNode == nil {
		return
	}

	totalWidth := ed.focusedWindow.viewportWidth
	halfWidth := max(1, totalWidth/2-1)
	ed.focusedWindow.viewportWidth = halfWidth

	newWin := &Window{
		buffer:         buf,
		Cursor:         0,
		Col:            0,
		topLine:        0,
		viewportHeight: ed.focusedWindow.viewportHeight,
		viewportWidth:  totalWidth - halfWidth - 1,
		renderedMin:    -1,
		renderedMax:    -1,
	}

	ed.initWindowLayer(ed.focusedWindow, halfWidth)
	ed.initWindowLayer(newWin, newWin.viewportWidth)

	newWindowNode := &SplitNode{Window: newWin}
	currentWindowNode := &SplitNode{Window: ed.focusedWindow}

	currentNode.Direction = SplitVertical
	currentNode.Window = nil
	currentNode.Children = [2]*SplitNode{currentWindowNode, newWindowNode}
	currentWindowNode.Parent = currentNode
	newWindowNode.Parent = currentNode

	// Focus the new window
	ed.focusedWindow = newWin

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
	// Add sign column width if enabled (2 chars: sign + space)
	if ed.showSignColumn {
		ed.win().lineNumWidth += 2
	}

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

	// Calculate format widths
	signWidth := 0
	if ed.showSignColumn {
		signWidth = 2 // sign char + space
	}
	lineNumOnlyWidth := ed.win().lineNumWidth - signWidth
	lineNumFmt := fmt.Sprintf("%%%dd ", lineNumOnlyWidth-1)
	tildeFmt := fmt.Sprintf("%%%ds ", ed.win().lineNumWidth-1)

	var spans []tui.Span

	if lineIdx < len(ed.buf().Lines) {
		// Content line - apply horizontal scroll
		line := ed.buf().Lines[lineIdx]
		if ed.win().leftCol > 0 && ed.win().leftCol < len(line) {
			line = line[ed.win().leftCol:]
		} else if ed.win().leftCol >= len(line) {
			line = ""
		}

		isCursorLine := lineIdx == ed.win().Cursor

		// Add sign column if enabled
		if ed.showSignColumn {
			sign, signStyle := ed.getGitSignForLine(lineIdx)
			spans = append(spans, tui.Span{Text: sign + " ", Style: signStyle})
		}

		// Calculate displayed line number (absolute or relative)
		var displayNum int
		if ed.relativeNumber && !isCursorLine {
			// Relative: show distance from cursor
			displayNum = abs(lineIdx - ed.win().Cursor)
		} else {
			// Absolute: show actual line number (1-indexed)
			displayNum = lineIdx + 1
		}
		lineNum := fmt.Sprintf(lineNumFmt, displayNum)

		numStyle := lineNumStyle
		if isCursorLine {
			numStyle = cursorLineNumStyle
		}

		if ed.Mode == "VISUAL" {
			// Visual mode needs offset-adjusted highlighting
			spans = append(spans, tui.Span{Text: lineNum, Style: numStyle})
			spans = append(spans, ed.getVisualSpans(lineIdx, line)...)
		} else {
			contentSpans := ed.highlightSearchMatches(line)
			// Apply cursorline background if enabled and this is the cursor line
			if ed.cursorLine && isCursorLine {
				contentSpans = ed.applyCursorLineStyle(contentSpans)
			}
			spans = append(spans, tui.Span{Text: lineNum, Style: numStyle})
			spans = append(spans, contentSpans...)
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

	if ed.win().visualMode == VisualLine {
		// Line mode: entire line is selected or not
		if ed.isLineSelected(lineIdx) {
			return []tui.Span{{Text: line, Style: inverseStyle}}
		}
		return []tui.Span{{Text: line, Style: normalStyle}}
	}

	if ed.win().visualMode == VisualBlock {
		// Block mode: select a rectangular column region
		if ed.isLineSelected(lineIdx) {
			startCol := min(ed.win().visualStartCol, ed.win().Col)
			endCol := max(ed.win().visualStartCol, ed.win().Col) + 1
			return ed.buildBlockSelectionSpans(line, startCol, endCol, normalStyle, inverseStyle)
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

// buildBlockSelectionSpans builds spans for block (column) visual selection
func (ed *Editor) buildBlockSelectionSpans(line string, startCol, endCol int, normalStyle, inverseStyle tui.Style) []tui.Span {
	// Clamp columns to line length
	lineLen := len(line)
	startCol = max(0, min(startCol, lineLen))
	endCol = max(0, min(endCol, lineLen))

	var spans []tui.Span

	// Before selection
	if startCol > 0 {
		spans = append(spans, tui.Span{Text: line[:startCol], Style: normalStyle})
	}

	// Selected block region
	if startCol < endCol {
		spans = append(spans, tui.Span{Text: line[startCol:endCol], Style: inverseStyle})
	} else if startCol == endCol && startCol < lineLen {
		// Single column - still highlight one character
		spans = append(spans, tui.Span{Text: string(line[startCol]), Style: inverseStyle})
	} else if startCol >= lineLen {
		// Selection extends beyond line - show virtual space
		spans = append(spans, tui.Span{Text: " ", Style: inverseStyle})
	}

	// After selection
	if endCol < lineLen {
		spans = append(spans, tui.Span{Text: line[endCol:], Style: normalStyle})
	}

	return spans
}

// buildWindowView builds the view for a single window
func buildWindowView(w *Window, focused bool) any {
	return tui.Col{Children: []any{
		// Content area - imperative layer, efficiently updated
		// Width is set for vertical splits to constrain each window's area
		tui.LayerView{
			Layer:      w.contentLayer,
			ViewHeight: int16(w.viewportHeight),
			ViewWidth:  int16(w.viewportWidth),
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

	// Wrap in Col to add wildmenu and status line at bottom
	return tui.Col{Children: []any{
		windowTree,
		// Wildmenu appears above status line when active
		tui.IfNode{
			Cond: &ed.cmdCompletionActive,
			Then: tui.RichText{Spans: &ed.cmdWildmenuSpans},
		},
		tui.Text{Content: &ed.StatusLine},
	}}
}

// buildFuzzyView creates the declarative fuzzy finder overlay view
func buildFuzzyView(ed *Editor) any {
	return tui.Col{Children: []any{
		// Prompt line with query
		tui.Text{Content: &ed.fuzzy.Query, Style: tui.Style{Attr: tui.AttrBold}},
		// Results list with selection
		&tui.SelectionList{
			Items:      &ed.fuzzy.Matches,
			Selected:   &ed.fuzzy.Selected,
			Marker:     "> ",
			MaxVisible: 20,
			Render: func(s *string) any {
				return tui.Text{Content: s}
			},
		},
		// Status line
		tui.Text{Content: &ed.StatusLine},
	}}
}

// TextObjectFunc return (start, end) range - end is exclusive
type TextObjectFunc func(line string, col int) (start, end int)

// MultiLineTextObjectFunc returns a Range for multi-line text objects
type MultiLineTextObjectFunc func(ed *Editor) Range

// OperatorFunc 's act on a range within a single line
type OperatorFunc func(ed *Editor, app *tui.App, start, end int)

// MultiLineOperatorFunc functions act on a Range across lines
type MultiLineOperatorFunc func(ed *Editor, app *tui.App, r Range)

// Shared text object definitions - used by both normal mode operators and visual mode
type mlTextObjectDef struct {
	key string
	fn  MultiLineTextObjectFunc
}

type wordTextObjectDef struct {
	key string
	fn  TextObjectFunc
}

// These are initialized in init() because they reference Editor methods
var (
	mlTextObjectDefs   []mlTextObjectDef
	wordTextObjectDefs []wordTextObjectDef
)

func initTextObjectDefs() {
	mlTextObjectDefs = []mlTextObjectDef{
		{"ip", toInnerParagraphML}, {"ap", toAParagraphML},
		{"is", toInnerSentenceML}, {"as", toASentenceML},
		{"i(", toInnerParenML}, {"a(", toAParenML},
		{"i)", toInnerParenML}, {"a)", toAParenML},
		{"i[", toInnerBracketML}, {"a[", toABracketML},
		{"i]", toInnerBracketML}, {"a]", toABracketML},
		{"i{", toInnerBraceML}, {"a{", toABraceML},
		{"i}", toInnerBraceML}, {"a}", toABraceML},
		{"i<", toInnerAngleML}, {"a<", toAAngleML},
		{"i>", toInnerAngleML}, {"a>", toAAngleML},
	}
	wordTextObjectDefs = []wordTextObjectDef{
		{"iw", toInnerWord}, {"aw", toAWord},
		{"iW", toInnerWORD}, {"aW", toAWORD},
	}
}

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

// Inner quotes helper - finds quote pair containing cursor or seeks forward
func toInnerQuoteChar(line string, col int, quote byte) (int, int) {
	// Find all quote positions
	var quotes []int
	for i := 0; i < len(line); i++ {
		if line[i] == quote {
			quotes = append(quotes, i)
		}
	}

	// Need at least 2 quotes to form a pair
	if len(quotes) < 2 {
		return col, col
	}

	// Find pair that contains cursor or is first pair after cursor
	for i := 0; i+1 < len(quotes); i += 2 {
		start, end := quotes[i], quotes[i+1]
		// If cursor is at/before end of this pair, use it
		if col <= end {
			return start + 1, end // inner: exclude quotes
		}
	}

	return col, col // No suitable pair found
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
func toASingleQuote(line string, col int) (int, int)     { return toAQuoteChar(line, col, '\'') }

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

	// Register all text object combinations (uses shared mlTextObjectDefs)
	for _, op := range mlOperators {
		for _, obj := range mlTextObjectDefs {
			pattern := op.key + obj.key
			opFn, objFn := op.fn, obj.fn
			app.Handle(pattern, func(_ riffkey.Match) {
				r := objFn(ed)
				if r.Start.Line >= 0 {
					opFn(ed, app, r)
				}
			})
		}
	}

	// Motion methods for operator + motion (dj, yk, cw, etc.)
	// Uses named Motion* methods on Editor for readability and extensibility
	type motionFunc func(count int) Range
	mlMotions := []struct {
		key string
		fn  motionFunc
	}{
		// Linewise motions
		{"j", func(count int) Range { return ed.MotionDown(count) }},
		{"k", func(count int) Range { return ed.MotionUp(count) }},
		{"gg", func(_ int) Range { return ed.MotionToStart() }},
		{"G", func(_ int) Range { return ed.MotionToEnd() }},
		// Characterwise motions
		{"w", func(count int) Range { return ed.MotionWordForward(count) }},
		{"b", func(count int) Range { return ed.MotionWordBackward(count) }},
		{"e", func(count int) Range { return ed.MotionWordEnd(count) }},
		{"$", func(_ int) Range { return ed.MotionToLineEnd() }},
		{"0", func(_ int) Range { return ed.MotionToLineStart() }},
	}

	// Register operator + motion combinations
	for _, op := range mlOperators {
		for _, mot := range mlMotions {
			pattern := op.key + mot.key
			opFn, motFn := op.fn, mot.fn
			app.Handle(pattern, func(m riffkey.Match) {
				r := motFn(m.Count)
				opFn(ed, app, r)
			})
		}
	}

	// Line operations
	app.Handle("cc", func(_ riffkey.Match) { ed.ChangeLine(app) })
	app.Handle("S", func(_ riffkey.Match) { ed.ChangeLine(app) })
	app.Handle("yy", func(_ riffkey.Match) { ed.YankLine() })
	app.Handle("Y", func(_ riffkey.Match) { ed.YankLine() })
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
func mlOpDelete(ed *Editor, app *tui.App, r Range) {
	ed.saveUndo()

	// Extract the text being deleted for yank register
	yankRegister = ed.extractRange(r)

	// Delete the range
	ed.deleteRange(r)

	ed.updateDisplay()
	ed.updateCursor()
}

func mlOpChange(ed *Editor, app *tui.App, r Range) {
	ed.saveUndo()

	// Extract for yank register
	yankRegister = ed.extractRange(r)

	// Delete the range
	ed.deleteRange(r)

	ed.updateDisplay()
	ed.enterInsertMode(app)
}

func mlOpYank(ed *Editor, app *tui.App, r Range) {
	yankRegister = ed.extractRange(r)
	ed.StatusLine = fmt.Sprintf("Yanked: %q", yankRegister)
	ed.updateDisplay()
}

// extractRange extracts text from a Range
func (ed *Editor) extractRange(r Range) string {
	startLine, startCol := r.Start.Line, r.Start.Col
	endLine, endCol := r.End.Line, r.End.Col

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

// deleteRange deletes text from a Range
func (ed *Editor) deleteRange(r Range) {
	startLine, startCol := r.Start.Line, r.Start.Col
	endLine, endCol := r.End.Line, r.End.Col

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

// extractBlock extracts a rectangular block of text (for visual block yank)
func (ed *Editor) extractBlock(startLine, endLine, startCol, endCol int) string {
	var lines []string
	for line := startLine; line <= endLine && line < len(ed.buf().Lines); line++ {
		text := ed.buf().Lines[line]
		// Extract the column range from this line
		sc := min(startCol, len(text))
		ec := min(endCol, len(text))
		if sc < ec {
			lines = append(lines, text[sc:ec])
		} else {
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n")
}

// deleteBlock deletes a rectangular block of text (for visual block delete)
func (ed *Editor) deleteBlock(startLine, endLine, startCol, endCol int) {
	for line := startLine; line <= endLine && line < len(ed.buf().Lines); line++ {
		text := ed.buf().Lines[line]
		// Delete the column range from this line
		sc := min(startCol, len(text))
		ec := min(endCol, len(text))
		if sc < len(text) {
			ed.buf().Lines[line] = text[:sc] + text[ec:]
		}
	}
}

// Multi-line text object functions

// toInnerParagraphML returns the range of the inner paragraph
func toInnerParagraphML(ed *Editor) Range {
	start, end := ed.findInnerParagraph()
	return Range{
		Start: Pos{Line: start, Col: 0},
		End:   Pos{Line: end, Col: len(ed.buf().Lines[end])},
	}
}

// toAParagraphML returns the range including trailing blank lines
func toAParagraphML(ed *Editor) Range {
	start, end := ed.findAParagraph()
	return Range{
		Start: Pos{Line: start, Col: 0},
		End:   Pos{Line: end, Col: len(ed.buf().Lines[end])},
	}
}

// toInnerSentenceML finds the current sentence boundaries across lines
func toInnerSentenceML(ed *Editor) Range {
	return ed.findSentenceBounds(false)
}

// toASentenceML finds the sentence including trailing whitespace
func toASentenceML(ed *Editor) Range {
	return ed.findSentenceBounds(true)
}

// findSentenceBounds finds sentence boundaries across lines
func (ed *Editor) findSentenceBounds(includeTrailing bool) Range {
	// Start from cursor position
	startLine := ed.win().Cursor
	startCol := ed.win().Col
	endLine := ed.win().Cursor
	endCol := ed.win().Col

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

	return Range{
		Start: Pos{Line: startLine, Col: startCol},
		End:   Pos{Line: endLine, Col: endCol},
	}
}

// Multi-line bracket/brace/paren text objects

// InvalidRange returns a Range with -1 values to indicate "not found"
var InvalidRange = Range{Start: Pos{Line: -1, Col: -1}, End: Pos{Line: -1, Col: -1}}

// findPairBoundsML finds matching bracket pairs across multiple lines.
// If cursor is inside a pair, uses that. Otherwise searches forward for next pair.
func (ed *Editor) findPairBoundsML(open, close byte, inner bool) Range {
	// Try to find a pair containing the cursor, or search forward for next pair
	r := ed.findPairContaining(open, close)
	if r.Start.Line < 0 {
		// Not inside a pair - search forward for next opening bracket
		r = ed.findNextPair(open, close)
	}
	if r.Start.Line < 0 {
		return InvalidRange
	}

	if inner {
		// Exclude the brackets themselves
		r.Start.Col++
		// If startCol goes past end of line, move to next line
		if r.Start.Col >= len(ed.buf().Lines[r.Start.Line]) && r.Start.Line < r.End.Line {
			r.Start.Line++
			r.Start.Col = 0
		}
		// endCol already points at closing bracket, so we don't include it
	} else {
		// Include both brackets
		r.End.Col++
	}

	return r
}

// findPairContaining searches backward for an opening bracket and forward for its match.
// Returns the bracket positions if cursor is inside a pair, or InvalidRange if not.
func (ed *Editor) findPairContaining(open, close byte) Range {
	// Search backward for opening bracket
	startLine := ed.win().Cursor
	startCol := ed.win().Col
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
						endLine, endCol := ed.findMatchingClose(open, close, startLine, startCol)
						if endLine >= 0 {
							return Range{
								Start: Pos{Line: startLine, Col: startCol},
								End:   Pos{Line: endLine, Col: endCol},
							}
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
			return InvalidRange
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
func (ed *Editor) findNextPair(open, close byte) Range {
	startLine := ed.win().Cursor
	startCol := ed.win().Col + 1 // Start after cursor

	for {
		line := ed.buf().Lines[startLine]
		for startCol < len(line) {
			if line[startCol] == open {
				// Found opening bracket - find its match
				endLine, endCol := ed.findMatchingClose(open, close, startLine, startCol)
				if endLine >= 0 {
					return Range{
						Start: Pos{Line: startLine, Col: startCol},
						End:   Pos{Line: endLine, Col: endCol},
					}
				}
			}
			startCol++
		}
		if startLine < len(ed.buf().Lines)-1 {
			startLine++
			startCol = 0
		} else {
			return InvalidRange
		}
	}
}

// Paren text objects
func toInnerParenML(ed *Editor) Range   { return ed.findPairBoundsML('(', ')', true) }
func toAParenML(ed *Editor) Range       { return ed.findPairBoundsML('(', ')', false) }

// Bracket text objects
func toInnerBracketML(ed *Editor) Range { return ed.findPairBoundsML('[', ']', true) }
func toABracketML(ed *Editor) Range     { return ed.findPairBoundsML('[', ']', false) }

// Brace text objects
func toInnerBraceML(ed *Editor) Range   { return ed.findPairBoundsML('{', '}', true) }
func toABraceML(ed *Editor) Range       { return ed.findPairBoundsML('{', '}', false) }

// Angle bracket text objects
func toInnerAngleML(ed *Editor) Range   { return ed.findPairBoundsML('<', '>', true) }
func toAAngleML(ed *Editor) Range       { return ed.findPairBoundsML('<', '>', false) }

// Word motion helper
func isWordChar(r byte) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// abs returns absolute value of an int
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
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
func (ed *Editor) enterVisualMode(app *tui.App, mode VisualMode) {
	ed.Mode = "VISUAL"
	ed.win().visualStart = ed.win().Cursor
	ed.win().visualStartCol = ed.win().Col
	ed.win().visualMode = mode
	switch mode {
	case VisualLine:
		ed.StatusLine = "-- VISUAL LINE --"
	case VisualBlock:
		ed.StatusLine = "-- VISUAL BLOCK --"
	default:
		ed.StatusLine = "-- VISUAL --"
	}
	ed.updateDisplay()

	visualRouter := riffkey.NewRouter().Name("visual")

	// Movement - same action methods as normal mode!
	// Middleware handles visual selection highlighting via refresh()
	visualRouter.Handle("j", func(m riffkey.Match) { ed.Down(m.Count) })
	visualRouter.Handle("k", func(m riffkey.Match) { ed.Up(m.Count) })
	visualRouter.Handle("h", func(m riffkey.Match) { ed.Left(m.Count) })
	visualRouter.Handle("l", func(m riffkey.Match) { ed.Right(m.Count) })
	visualRouter.Handle("gg", func(_ riffkey.Match) { ed.BufferStart() })
	visualRouter.Handle("G", func(_ riffkey.Match) { ed.BufferEnd() })
	visualRouter.Handle("0", func(_ riffkey.Match) { ed.LineStart() })
	visualRouter.Handle("$", func(_ riffkey.Match) { ed.LineEnd() })
	visualRouter.Handle("^", func(_ riffkey.Match) { ed.FirstNonBlank() })
	visualRouter.Handle("w", func(m riffkey.Match) { ed.NextWordStart(m.Count) })
	visualRouter.Handle("b", func(m riffkey.Match) { ed.PrevWordStart(m.Count) })
	visualRouter.Handle("e", func(m riffkey.Match) { ed.NextWordEnd(m.Count) })

	// Selection manipulation
	visualRouter.Handle("o", func(_ riffkey.Match) { ed.SwapVisualEnds() })
	visualRouter.Handle("O", func(_ riffkey.Match) { ed.SwapVisualEnds() })

	// Operators
	visualRouter.Handle("d", func(_ riffkey.Match) { ed.VisualDelete(app) })
	visualRouter.Handle("x", func(_ riffkey.Match) { ed.VisualDelete(app) })
	visualRouter.Handle("c", func(_ riffkey.Match) { ed.VisualChange(app) })
	visualRouter.Handle("y", func(_ riffkey.Match) { ed.VisualYank(app) })

	// Block insert/append (only meaningful in block mode, but available in all)
	visualRouter.Handle("I", func(_ riffkey.Match) { ed.VisualBlockInsert(app) })
	visualRouter.Handle("A", func(_ riffkey.Match) { ed.VisualBlockAppend(app) })

	// Switch visual modes
	visualRouter.Handle("v", func(_ riffkey.Match) {
		if ed.win().visualMode == VisualChar {
			ed.exitVisualMode(app)
		} else {
			ed.win().visualMode = VisualChar
			ed.StatusLine = "-- VISUAL --"
			ed.updateDisplay()
		}
	})
	visualRouter.Handle("V", func(_ riffkey.Match) {
		if ed.win().visualMode == VisualLine {
			ed.exitVisualMode(app)
		} else {
			ed.win().visualMode = VisualLine
			ed.StatusLine = "-- VISUAL LINE --"
			ed.updateDisplay()
		}
	})
	visualRouter.Handle("<C-v>", func(_ riffkey.Match) {
		if ed.win().visualMode == VisualBlock {
			ed.exitVisualMode(app)
		} else {
			ed.win().visualMode = VisualBlock
			ed.StatusLine = "-- VISUAL BLOCK --"
			ed.updateDisplay()
		}
	})

	// Text objects expand selection (uses shared definitions)
	for _, obj := range mlTextObjectDefs {
		objFn := obj.fn
		visualRouter.Handle(obj.key, func(_ riffkey.Match) {
			ed.VisualExpandToTextObject(objFn(ed))
		})
	}
	for _, obj := range wordTextObjectDefs {
		objFn := obj.fn
		visualRouter.Handle(obj.key, func(_ riffkey.Match) {
			line := ed.buf().Lines[ed.win().Cursor]
			ed.VisualExpandToWordObject(objFn(line, ed.win().Col))
		})
	}

	// Exit
	visualRouter.Handle("<Esc>", func(_ riffkey.Match) { ed.exitVisualMode(app) })

	// Visual mode hook: refresh() for selection highlighting instead of updateDisplay()
	visualRouter.AddOnAfter(func() {
		ed.refresh()
	})

	app.Push(visualRouter)
}

func (ed *Editor) exitVisualMode(app *tui.App) {
	ed.Mode = "NORMAL"
	ed.StatusLine = ""
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
	ed.updateCmdLineCursor()

	// Create command line router (NoCounts so digits work in commands)
	cmdRouter := riffkey.NewRouter().Name("cmdline").NoCounts()

	cmdRouter.Handle("<CR>", func(_ riffkey.Match) {
		cmd := ed.cmdLineInput
		ed.exitCommandMode(app)
		ed.executeCommand(app, ed.cmdLinePrompt, cmd)
	})
	cmdRouter.Handle("<Esc>", func(_ riffkey.Match) { ed.exitCommandMode(app) })
	cmdRouter.Handle("<BS>", func(_ riffkey.Match) { ed.CmdLineBackspace() })
	cmdRouter.Handle("<Space>", func(_ riffkey.Match) { ed.CmdLineAppend(' ') })

	// Tab completion for : commands
	cmdRouter.Handle("<Tab>", func(_ riffkey.Match) {
		if prompt == ":" {
			ed.completeNext()
			ed.updateDisplay()
		}
	})
	cmdRouter.Handle("<S-Tab>", func(_ riffkey.Match) {
		if prompt == ":" {
			ed.completePrev()
			ed.updateDisplay()
		}
	})

	cmdRouter.HandleUnmatched(func(k riffkey.Key) bool {
		if k.Rune != 0 && k.Mod == riffkey.ModNone {
			ed.CmdLineAppend(k.Rune)
			return true
		}
		return false
	})

	app.Push(cmdRouter)
}

// =============================================================================
// Command Line Actions
// =============================================================================

// CmdLineAppend adds a character to the command line input
func (ed *Editor) CmdLineAppend(ch rune) {
	ed.cmdLineInput += string(ch)
	ed.StatusLine = ed.cmdLinePrompt + ed.cmdLineInput
	// Update completions for : commands
	if ed.cmdLinePrompt == ":" {
		ed.updateCompletions()
	}
	ed.updateDisplay()
	ed.updateCmdLineCursor()
}

// CmdLineBackspace removes the last character from command line input
func (ed *Editor) CmdLineBackspace() {
	if len(ed.cmdLineInput) > 0 {
		ed.cmdLineInput = ed.cmdLineInput[:len(ed.cmdLineInput)-1]
		ed.StatusLine = ed.cmdLinePrompt + ed.cmdLineInput
		// Update completions for : commands
		if ed.cmdLinePrompt == ":" {
			ed.updateCompletions()
		}
		ed.updateDisplay()
		ed.updateCmdLineCursor()
	}
}

// updateCmdLineCursor positions the cursor at the end of command line input
func (ed *Editor) updateCmdLineCursor() {
	size := ed.app.Size()
	ed.app.SetCursor(1+len(ed.cmdLineInput), size.Height-1)
}

func (ed *Editor) exitCommandMode(app *tui.App) {
	ed.cmdLineActive = false
	ed.cmdCompletionActive = false
	ed.cmdMatches = nil
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
	// Store for @: repeat
	if cmd != "" {
		ed.lastColonCmd = cmd
	}

	// Handle quit commands separately (they exit, no display update needed)
	switch cmd {
	case "q", "quit":
		if !ed.root.IsLeaf() {
			ed.closeWindow()
		} else {
			app.Stop()
			return
		}
	case "qa", "qall":
		app.Stop()
		return
	}

	// All other commands update display and cursor at the end
	defer func() {
		ed.updateDisplay()
		ed.updateCursor()
	}()

	switch cmd {
	case "w", "write":
		ed.StatusLine = "E37: No write since last change (use :w! to override)"
	case "wq", "x":
		ed.StatusLine = "E37: No write since last change (use :wq! to override)"
	case "sp", "split":
		ed.splitHorizontal()
	case "vs", "vsplit":
		ed.splitVertical()
	case "Ex", "Explore":
		ed.openExplorer("")
	case "Vex", "Vexplore":
		ed.openExplorerSplit(true, "")
	case "Sex", "Sexplore":
		ed.openExplorerSplit(false, "")
	case "Files", "FZF":
		ed.openFuzzyFinder(app)
	case "close":
		ed.closeWindow()
	case "only", "on":
		ed.closeOtherWindows()
	case "noh", "nohlsearch":
		ed.searchPattern = ""
	case "set relativenumber", "set rnu":
		ed.relativeNumber = true
		ed.StatusLine = "relativenumber on"
	case "set norelativenumber", "set nornu":
		ed.relativeNumber = false
		ed.StatusLine = "relativenumber off"
	case "set number", "set nu":
		ed.relativeNumber = false
		ed.StatusLine = "number on (relativenumber off)"
	case "set cursorline", "set cul":
		ed.cursorLine = true
		ed.StatusLine = "cursorline on"
	case "set nocursorline", "set nocul":
		ed.cursorLine = false
		ed.StatusLine = "cursorline off"
	case "set signcolumn", "set scl":
		ed.showSignColumn = true
		ed.StatusLine = "signcolumn on"
	case "set nosigncolumn", "set noscl":
		ed.showSignColumn = false
		ed.StatusLine = "signcolumn off"
	case "gitsigns", "Gitsigns":
		ed.refreshGitSigns()
		ed.invalidateRenderedRange()
		ed.StatusLine = "Git signs refreshed"
	default:
		// Try to parse as line number first
		lineNum := 0
		isNumber := len(cmd) > 0
		for _, c := range cmd {
			if c >= '0' && c <= '9' {
				lineNum = lineNum*10 + int(c-'0')
			} else {
				isNumber = false
				break
			}
		}
		if isNumber && lineNum > 0 {
			lineNum = min(lineNum, len(ed.buf().Lines))
			ed.GotoLine(lineNum)
			return
		}

		// Try reflection-based command dispatch
		parts := strings.Fields(cmd)
		if len(parts) > 0 {
			cmdName := parts[0]
			args := parts[1:]
			if err := ed.ExecCommand(cmdName, args); err == nil {
				return
			}
		}

		ed.StatusLine = fmt.Sprintf("E492: Not an editor command: %s", cmd)
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
	app.Handle("f", func(_ riffkey.Match) { ed.promptForChar(app, ed.FindChar) })
	app.Handle("F", func(_ riffkey.Match) { ed.promptForChar(app, ed.FindCharBack) })
	app.Handle("t", func(_ riffkey.Match) { ed.promptForChar(app, ed.TillChar) })
	app.Handle("T", func(_ riffkey.Match) { ed.promptForChar(app, ed.TillCharBack) })
	app.Handle(";", func(_ riffkey.Match) { ed.RepeatFind() })
	app.Handle(",", func(_ riffkey.Match) { ed.RepeatFindReverse() })
}

// =============================================================================
// Find Char Actions - f/F/t/T and repeat with ;/,
// =============================================================================

// FindChar finds the next occurrence of ch on the line (f)
func (ed *Editor) FindChar(ch rune) {
	ed.lastFindChar = ch
	ed.lastFindDir = 1
	ed.lastFindTill = false
	ed.doFindChar(true, false, ch)
}

// FindCharBack finds the previous occurrence of ch on the line (F)
func (ed *Editor) FindCharBack(ch rune) {
	ed.lastFindChar = ch
	ed.lastFindDir = -1
	ed.lastFindTill = false
	ed.doFindChar(false, false, ch)
}

// TillChar moves to just before the next occurrence of ch (t)
func (ed *Editor) TillChar(ch rune) {
	ed.lastFindChar = ch
	ed.lastFindDir = 1
	ed.lastFindTill = true
	ed.doFindChar(true, true, ch)
}

// TillCharBack moves to just after the previous occurrence of ch (T)
func (ed *Editor) TillCharBack(ch rune) {
	ed.lastFindChar = ch
	ed.lastFindDir = -1
	ed.lastFindTill = true
	ed.doFindChar(false, true, ch)
}

// RepeatFind repeats the last f/F/t/T command (;)
func (ed *Editor) RepeatFind() {
	if ed.lastFindChar != 0 {
		ed.doFindChar(ed.lastFindDir == 1, ed.lastFindTill, ed.lastFindChar)
	}
}

// RepeatFindReverse repeats the last f/F/t/T in opposite direction (,)
func (ed *Editor) RepeatFindReverse() {
	if ed.lastFindChar != 0 {
		ed.doFindChar(ed.lastFindDir != 1, ed.lastFindTill, ed.lastFindChar)
	}
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

// ============================================================================
// File Tree / Netrw Implementation
// ============================================================================

// createNetrwBuffer creates a buffer for browsing a directory
func (ed *Editor) createNetrwBuffer(dirPath string) *Buffer {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		absPath = dirPath
	}

	ft := &FileTree{
		Path:       absPath,
		ShowHidden: false,
	}
	ft.refresh()

	buf := &Buffer{
		Lines:    ft.toLines(),
		FileName: absPath + "/", // trailing slash indicates directory
		marks:    make(map[rune]Pos),
		fileTree: ft,
	}
	return buf
}

// refresh reloads the directory entries
func (ft *FileTree) refresh() {
	ft.Entries = nil

	entries, err := os.ReadDir(ft.Path)
	if err != nil {
		ft.Entries = []DirEntry{{Name: "Error: " + err.Error(), IsDir: false}}
		return
	}

	// Sort: directories first, then files, alphabetically within each
	var dirs, files []DirEntry
	for _, e := range entries {
		name := e.Name()
		// Skip hidden files unless ShowHidden is true
		if !ft.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		de := DirEntry{Name: name, IsDir: e.IsDir()}
		if e.IsDir() {
			dirs = append(dirs, de)
		} else {
			files = append(files, de)
		}
	}

	// Sort each group alphabetically (case-insensitive)
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	// Add parent directory entry first
	ft.Entries = append(ft.Entries, DirEntry{Name: "..", IsDir: true})
	ft.Entries = append(ft.Entries, dirs...)
	ft.Entries = append(ft.Entries, files...)
}

// toLines converts file tree entries to displayable lines
func (ft *FileTree) toLines() []string {
	lines := make([]string, 0, len(ft.Entries)+2)

	// Header
	lines = append(lines, "\" Press ? for help")
	lines = append(lines, ft.Path+"/")
	lines = append(lines, "")

	for _, e := range ft.Entries {
		if e.IsDir {
			lines = append(lines, e.Name+"/")
		} else {
			lines = append(lines, e.Name)
		}
	}
	return lines
}

// getEntryAtLine returns the directory entry for a given line number
// Returns nil if the line is not an entry (header lines)
func (ft *FileTree) getEntryAtLine(lineIdx int) *DirEntry {
	// Skip header lines (help, path, blank)
	entryIdx := lineIdx - 3
	if entryIdx < 0 || entryIdx >= len(ft.Entries) {
		return nil
	}
	return &ft.Entries[entryIdx]
}

// openExplorer opens a file explorer in the current window
func (ed *Editor) openExplorer(dirPath string) {
	if dirPath == "" {
		// Default to directory of current file
		if ed.buf().FileName != "" && !ed.isNetrw() {
			dirPath = filepath.Dir(ed.buf().FileName)
		} else {
			dirPath = "."
		}
	}

	buf := ed.createNetrwBuffer(dirPath)
	ed.win().buffer = buf
	ed.win().Cursor = 3 // Start on first entry (after header)
	ed.win().Col = 0
	ed.win().topLine = 0
	ed.invalidateRenderedRange()
	ed.updateDisplay()
}

// openExplorerSplit opens a file explorer in a new split
func (ed *Editor) openExplorerSplit(vertical bool, dirPath string) {
	if dirPath == "" {
		if ed.buf().FileName != "" && !ed.isNetrw() {
			dirPath = filepath.Dir(ed.buf().FileName)
		} else {
			dirPath = "."
		}
	}

	buf := ed.createNetrwBuffer(dirPath)
	if vertical {
		ed.splitVerticalWithBuffer(buf)
	} else {
		ed.splitHorizontalWithBuffer(buf)
	}
	ed.win().Cursor = 3
	ed.win().Col = 0
	ed.invalidateRenderedRange()
	ed.updateDisplay()
}

// isNetrw returns true if the current buffer is a netrw browser
func (ed *Editor) isNetrw() bool {
	return ed.buf().fileTree != nil
}

// netrwEnter handles Enter key in netrw - opens file or enters directory
func (ed *Editor) netrwEnter(app *tui.App) {
	if !ed.isNetrw() {
		return
	}

	ft := ed.buf().fileTree
	entry := ft.getEntryAtLine(ed.win().Cursor)
	if entry == nil {
		return
	}

	fullPath := filepath.Join(ft.Path, entry.Name)

	if entry.IsDir {
		// Enter directory
		ft.Path = fullPath
		ft.refresh()
		ed.buf().Lines = ft.toLines()
		ed.buf().FileName = fullPath + "/"
		ed.win().Cursor = 3 // Reset to first entry
		ed.win().topLine = 0
		ed.invalidateRenderedRange()
		ed.updateDisplay()
	} else {
		// Open file
		lines := loadFile(fullPath)
		if lines == nil {
			ed.StatusLine = "Error: Could not open " + fullPath
			return
		}
		ed.buf().Lines = lines
		ed.buf().FileName = fullPath
		ed.buf().fileTree = nil // No longer a netrw buffer
		ed.buf().undoStack = nil
		ed.buf().redoStack = nil
		ed.win().Cursor = 0
		ed.win().Col = 0
		ed.win().topLine = 0
		ed.refreshGitSigns()
		ed.invalidateRenderedRange()
		ed.updateDisplay()
	}
}

// netrwUp goes up one directory level (like pressing - in vim netrw)
func (ed *Editor) netrwUp() {
	if !ed.isNetrw() {
		return
	}

	ft := ed.buf().fileTree
	parent := filepath.Dir(ft.Path)
	if parent == ft.Path {
		return // Already at root
	}

	ft.Path = parent
	ft.refresh()
	ed.buf().Lines = ft.toLines()
	ed.buf().FileName = parent + "/"
	ed.win().Cursor = 3
	ed.win().topLine = 0
	ed.invalidateRenderedRange()
	ed.updateDisplay()
}

// netrwToggleHidden toggles display of hidden files
func (ed *Editor) netrwToggleHidden() {
	if !ed.isNetrw() {
		return
	}

	ft := ed.buf().fileTree
	ft.ShowHidden = !ft.ShowHidden
	ft.refresh()
	ed.buf().Lines = ft.toLines()
	ed.win().Cursor = 3
	ed.win().topLine = 0
	ed.invalidateRenderedRange()
	ed.updateDisplay()
	if ft.ShowHidden {
		ed.StatusLine = "Showing hidden files"
	} else {
		ed.StatusLine = "Hiding hidden files"
	}
}

// ============================================================================
// Fuzzy Finder Implementation (Declarative)
// ============================================================================

// openFuzzyFinder opens a fuzzy file finder as a pushed view overlay
func (ed *Editor) openFuzzyFinder(app *tui.App) {
	// Save current buffer state for restoration on cancel
	ed.fuzzy.PrevBuffer = ed.buf()
	ed.fuzzy.PrevCursor = ed.win().Cursor

	// Get source directory
	sourceDir := "."
	if ed.fuzzy.PrevBuffer.FileName != "" && !ed.isNetrw() {
		sourceDir = filepath.Dir(ed.fuzzy.PrevBuffer.FileName)
	}

	// Collect all files recursively
	allFiles := ed.collectFilesRecursive(sourceDir, 1000)

	// Initialize fuzzy state
	ed.fuzzy.Active = true
	ed.fuzzy.Query = "> "
	ed.fuzzy.AllItems = allFiles
	ed.fuzzy.Matches = allFiles
	ed.fuzzy.Selected = 0
	ed.fuzzy.SourceDir = sourceDir

	ed.StatusLine = "Fuzzy finder: type to search, ↑↓/C-p/C-n to navigate, Enter to select, Esc to cancel"

	// Push the fuzzy view - this shows the declarative UI and activates its input handlers
	app.PushView("fuzzy")
}

// collectFilesRecursive collects all files recursively up to maxFiles
func (ed *Editor) collectFilesRecursive(dir string, maxFiles int) []string {
	var files []string

	// Use filepath.WalkDir for efficiency
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if len(files) >= maxFiles {
			return filepath.SkipAll
		}
		// Skip hidden directories
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		// Skip hidden files
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		// Only add files
		if !d.IsDir() {
			// Make path relative to dir
			relPath, err := filepath.Rel(dir, path)
			if err == nil {
				files = append(files, relPath)
			}
		}
		return nil
	})

	return files
}

// fuzzyMatch returns true if text matches the pattern fuzzily
func fuzzyMatch(text, pattern string) bool {
	ti, pi := 0, 0
	for ti < len(text) && pi < len(pattern) {
		if text[ti] == pattern[pi] {
			pi++
		}
		ti++
	}
	return pi == len(pattern)
}

// fuzzyFilterMatches updates the matches based on the query
func (ed *Editor) fuzzyFilterMatches() {
	query := ed.fuzzy.Query
	// Strip the "> " prompt prefix for matching
	if len(query) > 2 {
		query = query[2:]
	} else {
		query = ""
	}

	if query == "" {
		ed.fuzzy.Matches = ed.fuzzy.AllItems
		ed.fuzzy.Selected = 0
		return
	}

	query = strings.ToLower(query)
	var matches []string

	for _, item := range ed.fuzzy.AllItems {
		if fuzzyMatch(strings.ToLower(item), query) {
			matches = append(matches, item)
		}
	}

	ed.fuzzy.Matches = matches
	if ed.fuzzy.Selected >= len(ed.fuzzy.Matches) {
		ed.fuzzy.Selected = max(0, len(ed.fuzzy.Matches)-1)
	}
}

// fuzzyBackspace handles backspace in fuzzy finder
func (ed *Editor) fuzzyBackspace() {
	if !ed.fuzzy.Active {
		return
	}
	// Keep the "> " prefix, only delete after it
	if len(ed.fuzzy.Query) > 2 {
		ed.fuzzy.Query = ed.fuzzy.Query[:len(ed.fuzzy.Query)-1]
		ed.fuzzyFilterMatches()
	}
}

// fuzzyUp moves selection up
func (ed *Editor) fuzzyUp() {
	if !ed.fuzzy.Active || len(ed.fuzzy.Matches) == 0 {
		return
	}
	ed.fuzzy.Selected = max(0, ed.fuzzy.Selected-1)
}

// fuzzyDown moves selection down
func (ed *Editor) fuzzyDown() {
	if !ed.fuzzy.Active || len(ed.fuzzy.Matches) == 0 {
		return
	}
	ed.fuzzy.Selected = min(len(ed.fuzzy.Matches)-1, ed.fuzzy.Selected+1)
}

// fuzzySelect opens the selected file
func (ed *Editor) fuzzySelect(app *tui.App) {
	if !ed.fuzzy.Active || len(ed.fuzzy.Matches) == 0 {
		ed.fuzzyCancel(app)
		return
	}

	selectedFile := ed.fuzzy.Matches[ed.fuzzy.Selected]
	fullPath := filepath.Join(ed.fuzzy.SourceDir, selectedFile)

	lines := loadFile(fullPath)
	if lines == nil {
		ed.StatusLine = "Error: Could not open " + fullPath
		return
	}

	// Clear fuzzy state BEFORE popping (PopView triggers render)
	ed.fuzzy.Active = false
	ed.fuzzy.Query = ""
	ed.fuzzy.Matches = nil
	ed.fuzzy.AllItems = nil

	// Open the file in a new buffer BEFORE popping
	ed.buf().Lines = lines
	ed.buf().FileName = fullPath
	ed.buf().undoStack = nil
	ed.buf().redoStack = nil
	ed.win().Cursor = 0
	ed.win().Col = 0
	ed.win().topLine = 0
	ed.refreshGitSigns()
	ed.invalidateRenderedRange()
	ed.updateDisplay()
	ed.StatusLine = "Opened: " + fullPath

	// Pop fuzzy view last (this triggers the final render)
	app.PopView()
}

// fuzzyCancel cancels the fuzzy finder and restores previous state
func (ed *Editor) fuzzyCancel(app *tui.App) {
	if !ed.fuzzy.Active {
		return
	}

	// Clear fuzzy state BEFORE popping (PopView triggers render)
	ed.fuzzy.Active = false
	ed.fuzzy.Query = ""
	ed.fuzzy.Matches = nil
	ed.fuzzy.AllItems = nil

	// Restore cursor position BEFORE popping
	ed.win().Cursor = ed.fuzzy.PrevCursor
	ed.win().Col = 0
	ed.invalidateRenderedRange()
	ed.updateDisplay()
	ed.StatusLine = ""

	// Pop fuzzy view last (this triggers the final render)
	app.PopView()
}
