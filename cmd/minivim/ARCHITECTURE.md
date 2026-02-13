# minivim Action-Based Architecture

## Core Principle

Every editing operation is a **named function**. Handlers and commands are thin wiring that call these functions.

---

## Foundation Types

```go
// Position in a buffer
type Pos struct {
    Line int
    Col  int
}

// Range in a buffer (for text objects and selections)
type Range struct {
    Start Pos
    End   Pos
}

// Motion result - where we'd end up
type Motion struct {
    Pos       Pos
    Linewise  bool  // for operators: delete whole lines?
    Inclusive bool  // include the end character?
}
```

---

## Movement Actions

Methods on Editor that move the cursor. Each returns the new position (for composability) and updates cursor state.

```go
// Character movements
func (ed *Editor) Left(n int) Pos
func (ed *Editor) Right(n int) Pos
func (ed *Editor) Up(n int) Pos
func (ed *Editor) Down(n int) Pos

// Word movements
func (ed *Editor) NextWordStart(n int) Pos   // w
func (ed *Editor) NextWordEnd(n int) Pos     // e
func (ed *Editor) PrevWordStart(n int) Pos   // b
func (ed *Editor) NextWORDStart(n int) Pos   // W
func (ed *Editor) NextWORDEnd(n int) Pos     // E
func (ed *Editor) PrevWORDStart(n int) Pos   // B

// Line movements
func (ed *Editor) LineStart() Pos            // 0
func (ed *Editor) LineEnd() Pos              // $
func (ed *Editor) FirstNonBlank() Pos        // ^

// Buffer movements
func (ed *Editor) GotoLine(n int) Pos        // G, gg, :goto
func (ed *Editor) GotoCol(n int) Pos
func (ed *Editor) BufferStart() Pos          // gg
func (ed *Editor) BufferEnd() Pos            // G

// Search movements
func (ed *Editor) NextMatch() Pos            // n
func (ed *Editor) PrevMatch() Pos            // N
func (ed *Editor) FindChar(ch rune) Pos      // f
func (ed *Editor) FindCharBack(ch rune) Pos  // F
func (ed *Editor) TillChar(ch rune) Pos      // t
func (ed *Editor) TillCharBack(ch rune) Pos  // T

// Mark movements
func (ed *Editor) GotoMark(reg rune) Pos           // `a (exact)
func (ed *Editor) GotoMarkLine(reg rune) Pos       // 'a (line start)

// Scroll movements (move cursor + viewport)
func (ed *Editor) HalfPageDown() Pos         // C-d
func (ed *Editor) HalfPageUp() Pos           // C-u
func (ed *Editor) PageDown() Pos             // C-f
func (ed *Editor) PageUp() Pos               // C-b

// Jump list
func (ed *Editor) JumpBack() Pos             // C-o
func (ed *Editor) JumpForward() Pos          // C-i
```

---

## Text Objects

Functions that return a Range. Don't move cursor - just calculate bounds.

```go
// Word objects
func (ed *Editor) InnerWord() Range          // iw
func (ed *Editor) AWord() Range              // aw
func (ed *Editor) InnerWORD() Range          // iW
func (ed *Editor) AWORD() Range              // aW

// Quote objects
func (ed *Editor) InnerQuote(q rune) Range   // i", i'
func (ed *Editor) AQuote(q rune) Range       // a", a'

// Bracket objects
func (ed *Editor) InnerBracket(open, close rune) Range  // i(, i[, i{
func (ed *Editor) ABracket(open, close rune) Range      // a(, a[, a{

// Block objects
func (ed *Editor) InnerParagraph() Range     // ip
func (ed *Editor) AParagraph() Range         // ap
func (ed *Editor) InnerSentence() Range      // is
func (ed *Editor) ASentence() Range          // as
```

---

## Operators

Functions that operate on a Range. These do the actual editing.

```go
// Core operators
func (ed *Editor) Delete(r Range) string     // Returns deleted text (for yank register)
func (ed *Editor) Change(r Range)            // Delete + enter insert mode
func (ed *Editor) Yank(r Range)              // Copy to register

// Line operators
func (ed *Editor) DeleteLines(start, end int) string
func (ed *Editor) YankLines(start, end int)

// In-place operators
func (ed *Editor) ToggleCase(r Range)        // ~
func (ed *Editor) Uppercase(r Range)         // gU
func (ed *Editor) Lowercase(r Range)         // gu
func (ed *Editor) Indent(r Range)            // >
func (ed *Editor) Dedent(r Range)            // <
```

---

## Cursor Management

Low-level cursor control with automatic clamping.

```go
// Set cursor with bounds checking (the foundation everything uses)
func (ed *Editor) SetCursor(p Pos)           // Clamps, updates viewport, refreshes
func (ed *Editor) SetCursorQuiet(p Pos)      // Clamps only, no refresh (for batch ops)

// Get current state
func (ed *Editor) Cursor() Pos
func (ed *Editor) CurrentLine() string
func (ed *Editor) CurrentChar() rune

// Viewport management
func (ed *Editor) EnsureVisible()            // Scroll if cursor outside viewport
func (ed *Editor) CenterCursor()             // zz
func (ed *Editor) CursorToTop()              // zt
func (ed *Editor) CursorToBottom()           // zb
```

---

## Mode Management

```go
func (ed *Editor) EnterNormal()
func (ed *Editor) EnterInsert()
func (ed *Editor) EnterInsertAfter()         // a
func (ed *Editor) EnterInsertLineEnd()       // A
func (ed *Editor) EnterInsertLineStart()     // I
func (ed *Editor) EnterVisual()
func (ed *Editor) EnterVisualLine()
func (ed *Editor) EnterCommand(prompt string)
func (ed *Editor) OpenLineBelow()            // o
func (ed *Editor) OpenLineAbove()            // O
```

---

## Marks

```go
func (ed *Editor) SetMark(reg rune)
func (ed *Editor) GetMark(reg rune) (Pos, bool)
func (ed *Editor) ListMarks() []Mark
```

---

## Buffer Mutations

Low-level buffer operations.

```go
func (b *Buffer) InsertLine(at int, content string)
func (b *Buffer) DeleteLine(at int) string
func (b *Buffer) ReplaceLine(at int, content string)
func (b *Buffer) InsertText(p Pos, text string) Pos  // Returns end position
func (b *Buffer) DeleteText(r Range) string          // Returns deleted text
```

---

## How Handlers Wire It

Handlers become trivially simple:

```go
// Movement handlers
app.Handle("w", func(m riffkey.Match) { ed.NextWordStart(m.Count) })
app.Handle("b", func(m riffkey.Match) { ed.PrevWordStart(m.Count) })
app.Handle("e", func(m riffkey.Match) { ed.NextWordEnd(m.Count) })
app.Handle("0", func(_ riffkey.Match) { ed.LineStart() })
app.Handle("$", func(_ riffkey.Match) { ed.LineEnd() })
app.Handle("gg", func(_ riffkey.Match) { ed.BufferStart() })
app.Handle("G", func(m riffkey.Match) {
    if m.Count > 1 {
        ed.GotoLine(m.Count)
    } else {
        ed.BufferEnd()
    }
})

// Operator + text object handlers
app.Handle("diw", func(_ riffkey.Match) { ed.Delete(ed.InnerWord()) })
app.Handle("ciw", func(_ riffkey.Match) { ed.Change(ed.InnerWord()) })
app.Handle("yiw", func(_ riffkey.Match) { ed.Yank(ed.InnerWord()) })

app.Handle("di\"", func(_ riffkey.Match) { ed.Delete(ed.InnerQuote('"')) })
app.Handle("ci\"", func(_ riffkey.Match) { ed.Change(ed.InnerQuote('"')) })

// Compound operations stay clean
app.Handle("dd", func(m riffkey.Match) {
    ed.DeleteLines(ed.win().Cursor, ed.win().Cursor + m.Count - 1)
})

// Marks
app.Handle("m", func(_ riffkey.Match) {
    // ... push router to get register, then:
    ed.SetMark(reg)
})
app.Handle("'", func(_ riffkey.Match) {
    // ... push router to get register, then:
    ed.GotoMarkLine(reg)
})
```

---

## How Commands Wire It

Commands use the exact same actions:

```go
var commands = map[string]func(ed *Editor, args []string) error{
    "goto": func(ed *Editor, args []string) error {
        line, _ := strconv.Atoi(args[0])
        ed.GotoLine(line)
        return nil
    },
    "mark": func(ed *Editor, args []string) error {
        if len(args) == 0 {
            // List marks
            for reg, m := range ed.buf().marks {
                ed.StatusLine += fmt.Sprintf(" %c:%d", reg, m.Line)
            }
        } else {
            ed.SetMark(rune(args[0][0]))
        }
        return nil
    },
    "delmarks": func(ed *Editor, args []string) error {
        for _, arg := range args {
            delete(ed.buf().marks, rune(arg[0]))
        }
        return nil
    },
    "normal": func(ed *Editor, args []string) error {
        // Execute normal mode commands - uses same actions!
        // :normal gg=G would call BufferStart() then indent whole file
        return nil
    },
}
```

---

## Visual Mode Simplification

Visual mode handlers become almost identical to normal mode:

```go
// In visual mode, movements just update selection end
visualRouter.Handle("w", func(m riffkey.Match) {
    ed.NextWordStart(m.Count)  // Same action!
    ed.RefreshVisual()
})

// Operators work on visual selection
visualRouter.Handle("d", func(_ riffkey.Match) {
    ed.Delete(ed.VisualRange())
    ed.EnterNormal()
})

visualRouter.Handle("c", func(_ riffkey.Match) {
    ed.Change(ed.VisualRange())  // Enters insert mode
})
```

---

## Helper for Repetition

```go
func (ed *Editor) Repeat(n int, action func()) {
    for i := 0; i < n; i++ {
        action()
    }
}

// Usage:
app.Handle("w", func(m riffkey.Match) {
    ed.Repeat(m.Count, func() { ed.NextWordStart(1) })
})
```

Or actions take count directly (cleaner for most cases).

---

## Benefits

1. **Commands**: `:goto`, `:mark`, `:normal` all trivial to implement
2. **Testing**: Actions are pure-ish functions, easy to unit test
3. **Dot Repeat**: Record action calls, not keystrokes
4. **Discoverability**: List of actions = list of capabilities
5. **Composability**: Build complex operations from simple atoms
6. **Consistency**: One way to do each thing
7. **Future**: Lua scripting? Just expose these functions

---

## Migration Path

1. Add `Pos`, `Range` types
2. Add `SetCursor(p Pos)` with clamping - everything uses this
3. Extract movement actions one at a time, update handlers to call them
4. Extract text objects, update operator+object handlers
5. Consolidate visual mode to use same actions
6. Add command dispatch using actions

Each step is independently shippable and testable.
