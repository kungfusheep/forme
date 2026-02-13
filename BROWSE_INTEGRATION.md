# Browse Integration - Component Proposal

This document outlines the components needed to integrate Browse (a terminal web browser) with the forme declarative framework, with a focus on creating generic, reusable components that can be contributed back to forme.

## Current State Comparison

| **forme has** | **Browse needs** |
|-------------|------------------|
| `Text`, `Progress`, `Row`, `Col` | Good foundation |
| `If`/`Else`, `ForEach` | Conditional rendering |
| Multi-view routing | Modal overlays |
| Diff-based screen updates | Performance |
| Double buffering | Performance |
| — | Scrollable viewport into large content |
| — | Scrollable list with selection |
| — | Label-based selection system |
| — | Text input field |
| — | Overlay/modal container (dims background) |
| — | Rich inline text (mixed styles per line) |

## Proposed Generic Components

These are all generic and suitable for contribution back to forme.

### 1. `Viewport` - Scrollable Content Window

```go
forme.Viewport{
    Content:   any,        // Child component tree
    ScrollY:   *int,       // Bound scroll position
    Height:    int,        // Visible height (0 = fill available)
    MaxScroll: *int,       // Output: computed max scroll value
}
```

**Use cases:** Any app with content taller than terminal - chat apps, log viewers, documentation browsers, pagers.

**Implementation notes:**
- Requires two-pass rendering: measure all content first, then render only visible slice
- Content inside viewport gets full measure, but render is culled to `[scrollY : scrollY+height]`
- Should expose computed `MaxScroll` so apps can clamp scroll position

### 2. `List` - Scrollable Selection List

```go
forme.List{
    Items:       *[]T,                    // Bound slice
    Selected:    *int,                    // Bound selection index (-1 for none)
    Render:      func(*T, bool) any,      // Item renderer (item, isSelected)
    Height:      int,                     // Visible rows (0 = show all)
    ShowLabels:  bool,                    // Generate a/b/c/... labels for quick-jump
    LabelInput:  *string,                 // Bound label input for filtering
}
```

**Use cases:** File pickers, menus, autocomplete dropdowns, buffer lists, bookmark lists.

**Features:**
- Auto-scroll to keep selection visible
- Optional label generation (a, b, c, ... aa, ab, ...) for keyboard quick-jump
- Scroll indicators when content overflows (arrows or scrollbar)
- Selected item highlighting via render function

**Label generation algorithm:**
```go
// Single chars first: a-z (26 items)
// Then doubles: aa, bb, cc, ... zz (26 more)
// Then triples if needed: aaa, bbb, ...
func GenerateLabels(count int) []string
```

### 3. `TextInput` - Single Line Input Field

```go
forme.TextInput{
    Value:       *string,      // Bound text value
    Placeholder: string,       // Shown when empty
    Cursor:      *int,         // Cursor position (optional, defaults to end)
    Width:       int,          // Field width (0 = fill available)
    Style:       DStyle,       // Normal style
    FocusStyle:  DStyle,       // Style when focused
    Mask:        rune,         // Password mask character (0 = none)
}
```

**Use cases:** Forms, search bars, command palettes, chat input, URL bars.

**Features:**
- Cursor rendering with blink or solid block
- Text overflow handling (horizontal scroll within field)
- Placeholder text when empty
- Optional password masking

### 4. `Overlay` - Modal Container

```go
forme.Overlay{
    Visible:  *bool,              // Show/hide binding
    DimBG:    bool,               // Dim underlying content (default true)
    Content:  any,                // What to render
    Position: OverlayPosition,    // Center, Bottom, Top, Custom{X, Y}
}

type OverlayPosition struct {
    Type   OverlayPositionType  // Center, Bottom, Top, Custom
    X, Y   int                  // For Custom positioning
}
```

**Use cases:** Modals, dialogs, tooltips, command palettes, confirmation prompts.

**Implementation notes:**
- Renders after base content in same frame
- `DimBG` applies semi-transparent dim to all cells rendered before overlay
- Content is measured and positioned according to `Position`

### 5. `Box` - Bordered Container

```go
forme.Box{
    Title:    any,              // string or *string for dynamic titles
    Border:   BorderStyle,      // Single, Rounded, Double, None
    Style:    DStyle,           // Border and title style
    Content:  any,              // Children
    Padding:  int,              // Inner padding (default 1)
    Width:    int,              // Fixed width (0 = fit content)
    Height:   int,              // Fixed height (0 = fit content)
}
```

**Use cases:** Grouping content, panels, dialog boxes, info cards.

**Border styles:**
```
Single:  ┌─┐│└┘
Rounded: ╭─╮│╰╯
Double:  ╔═╗║╚╝
```

### 6. `RichText` - Mixed Inline Styles

```go
forme.RichText{
    Spans: []Span,
}

type Span struct {
    Text  string
    Style DStyle
}

// Builder helpers
forme.Rich(
    "Normal text ",
    forme.Bold("bold"),
    " and ",
    forme.Colored("colored", forme.Cyan),
    " text",
)
```

**Use cases:** Formatted text, syntax highlighting, logs with levels, markdown rendering.

**Features:**
- Multiple spans with different styles on same line
- Word wrapping respects span boundaries
- Builder pattern for ergonomic construction

## Required Techniques

### 1. Two-Pass Measure for Viewport Culling

Current forme measures and renders in one pass. For viewport culling:

```
Pass 1: Measure all content → compute total height
Pass 2: Render only visible slice [scrollY : scrollY+viewportHeight]
```

This should be opt-in, triggered by `Viewport` component. Content inside viewport gets full measure pass, but render pass is culled.

**Performance consideration:** For very large content (thousands of items), consider virtualization - only measure visible items plus buffer. This is an optimization that can come later.

### 2. Layered Rendering for Overlays

Three options:

**A) Declaration order (simplest, recommended)**
- Overlays declared last in tree render last
- Works with current forme model
- Browse can use `If` to conditionally include overlays

**B) Explicit Z-index**
```go
forme.Layer{Z: 1, Content: overlay}
```
- More complex, probably overkill

**C) Post-render hook**
- Special overlay phase after main render
- Most complex

**Recommendation:** Option A is sufficient. Overlays are conditionally shown, so they can be declared at end of component tree with `If(&showOverlay, overlayContent)`.

### 3. Focus Management

For `TextInput` to receive keystrokes:

```go
// Option A: Explicit focus
app.SetFocus(myInputRef)

// Option B: Auto-focus visible inputs
// Single visible TextInput auto-receives input

// Option C: Focus as state
forme.TextInput{
    Focused: *bool,
    // ...
}
```

**Recommendation:** Start with Option C (focus as state). App controls which input has focus via boolean binding. Input handlers only fire when focused.

### 4. Background Dimming

For overlay dim effect:

```go
// During overlay render, apply dim to all previously-rendered cells
func (b *Buffer) DimAll() {
    for i := range b.cells {
        b.cells[i].Style.Attr |= AttrDim
    }
}
```

Or more sophisticated: dim only non-overlay region.

## Browse-Specific Components

These are too specialized for forme core but would use forme primitives:

1. **HTML Document Renderer** - Walks DOM tree, produces styled cells with links/headings
2. **Link Label Overlay** - Labels positioned at link locations within document content
3. **Find Highlighting** - Match positions highlighted in rendered document
4. **Define Mode** - Word extraction from rendered content with multi-position labels

These would be implemented in Browse using forme's `Buffer` directly or as custom render functions.

## Migration Strategy

### Phase 1: Overlays (High Value, Low Risk)

Migrate existing modal UIs to forme:

| Browse Mode | forme Components |
|-------------|----------------|
| Theme picker | `Overlay` + `Box` + `List` |
| Buffer list | `Overlay` + `Box` + `List` |
| Favourites | `Overlay` + `Box` + `List` |
| TOC | `Overlay` + `Box` + `List` |
| Nav mode | `Overlay` + `Box` + `List` |

**Benefit:** Immediate code reduction, consistent styling, better maintainability.

### Phase 2: Input Components

| Browse Mode | forme Components |
|-------------|----------------|
| Omnibox | `Overlay` + `Box` + `TextInput` |
| Find bar | `TextInput` (bottom-positioned) |
| Form input | `TextInput` |

**Benefit:** Unified input handling, cursor management, consistent UX.

### Phase 3: Main Viewport

| Browse Area | forme Components |
|-------------|----------------|
| Document content | `Viewport` with custom HTML renderer |
| Status bar | `Row` positioned at bottom |
| Scroll indicator | Built into `Viewport` or custom |

**Benefit:** Viewport culling, unified scroll handling.

### Phase 4: Complex Modes

| Browse Mode | Approach |
|-------------|----------|
| Jump mode | Custom - labels on document positions |
| Preview mode | Custom - positioned tooltip near link |
| Define mode | Custom - word extraction + multi-labels |

**Benefit:** These may stay mostly custom but can use forme primitives for rendering.

## Implementation Priority

Recommended order for implementing in forme:

1. **`List`** - Highest value, used by 5+ Browse modes
2. **`Overlay`** - Required for modal pattern
3. **`Box`** - Simple, enables nice modal styling
4. **`TextInput`** - Enables Omnibox, Find, Form input
5. **`Viewport`** - Main content area (most complex)
6. **`RichText`** - Nice to have, Browse can work around

## Example: Buffer List in forme

Current Browse implementation: ~80 lines of manual rendering.

With forme components:

```go
// State
type BufferListState struct {
    Visible   bool
    Selected  int
    Input     string
    Buffers   []BufferInfo
}

// View
func bufferListView(state *BufferListState) any {
    return forme.Overlay{
        Visible: &state.Visible,
        DimBG:   true,
        Content: forme.Box{
            Title:  "Buffers",
            Border: forme.BorderRounded,
            Content: forme.List{
                Items:      &state.Buffers,
                Selected:   &state.Selected,
                ShowLabels: true,
                LabelInput: &state.Input,
                Height:     10,
                Render: func(b *BufferInfo, selected bool) any {
                    style := forme.DStyle{}
                    if selected {
                        style.Inverse = true
                    }
                    return forme.Text{
                        Content: b.Title + " - " + b.URL,
                        Style:   style,
                    }
                },
            },
        },
    }
}
```

**Result:** Declarative, bound to state, ~30 lines, reusable patterns.

## Open Questions

1. **Viewport virtualization** - For very long content, should Viewport support virtual scrolling (only measure visible + buffer)?

2. **Focus system** - How sophisticated should focus management be? Tab order? Focus trapping in modals?

3. **Animation** - Should forme support animated transitions? Fade in/out for overlays?

4. **Scroll indicators** - Arrows (↑/↓) vs scrollbar track? Configurable?

5. **Label customization** - Should List allow custom label sets (numbers, custom chars)?

---

*Document created for forme/Browse integration planning. Components designed to be generic and reusable across any terminal application.*
