<!-- wed:{"title":"Components","theme":"default"} -->

# Components

## Text

```go
Text("Static text")
Text(&variable)  // Read at render time
Text("Styled").FG(Red).BG(White).Bold().Underline().Dim()
```

## Containers

### VBox

Vertical layout:

```go
VBox(
    Text("Top"),
    Text("Bottom"),
)

VBox.Gap(1)(children...)           // Gap between children
VBox.Border(BorderRounded)(...)    // With border
VBox.Width(40)(...)                // Fixed width
VBox.Height(10)(...)               // Fixed height
VBox.WidthPct(0.5)(...)            // 50% of parent width
VBox.Grow(1)(...)                  // Flex grow factor
VBox.Title("Panel")(...)           // Border title
VBox.BorderFG(Cyan)(...)           // Border color
VBox.Fill(Black)(...)              // Fill container area
VBox.CascadeStyle(&style)(...)     // Style inheritance for children
```

Chain multiple:

```go
VBox.Border(BorderSingle).Gap(1).Width(50)(
    Text("Title").Bold(),
    HRule(),
    Text("Content"),
)
```

### HBox

Horizontal layout. Same modifiers as VBox:

```go
HBox.Gap(2)(
    Text("Left"),
    Space(),
    Text("Right"),
)
```

## Spacing

```go
Space()       // Flexible, grows to fill available space
SpaceH(2)     // Fixed 2 lines vertical
SpaceW(10)    // Fixed 10 chars horizontal
Space().Grow(2)  // 2x grow factor
Space().Char('.')  // Dotted leader
```

## Dividers

```go
HRule()  // ────────
VRule()  // │

HRule().Char('=')
HRule().Style(Style{FG: BrightBlack})
```

## Progress

```go
Progress(75)           // Static 75%
Progress(&percent)     // Dynamic
Progress(75).Width(40) // Fixed width
```

## Spinner

```go
frame := 0
Spinner(&frame)
Spinner(&frame).Frames(SpinnerDots)
Spinner(&frame).Frames(SpinnerLine)
Spinner(&frame).Style(Style{FG: Cyan})
```

Increment `frame` in a goroutine for animation.

## Leader

Label with fill character:

```go
Leader("Name", "John")
Leader("Price", "$9.99").Fill('.').Width(30)
Leader("Status", "OK").Style(Style{FG: Green})
```

Output: `Name......John`

## Sparkline

Mini line chart:

```go
data := []float64{1, 4, 2, 8, 5, 7, 3}
Sparkline(&data)
Sparkline(&data).Width(20).Style(Style{FG: Green})
```

## List

Navigable list with selection:

```go
items := []string{"Apple", "Banana", "Cherry"}

List(&items).
    Marker("> ").
    MaxVisible(10).
    BindVimNav().
    Render(func(s *string) any { return Text(s) }).
    Style(Style{BG: PaletteColor(235)}).
    SelectedStyle(Style{BG: PaletteColor(238)})
```

Custom item rendering:

```go
type Item struct {
    Icon string
    Name string
}

items := []Item{...}

List(&items).
    BindNav("j", "k").
    Render(func(item *Item) any {
        return HBox(
            Text(&item.Icon),
            Space(),
            Text(&item.Name),
        )
    }).
    SelectedStyle(Style{BG: PaletteColor(236)})
```

Selection is managed internally. Use `.Selection(&idx)` to bind to an external index,
or `.Ref(func(l *ListC[Item]) { myList = l })` for a reference.

### Declarative Bindings

Bindings are declared on the component — no `*App` needed:

```go
List(&items).
    BindNav("j", "k").                    // up/down
    BindPageNav("<C-d>", "<C-u>").        // page up/down
    BindFirstLast("g", "G").              // first/last
    BindVimNav().                         // all of the above with vim defaults
    BindDelete("dd").                     // delete selected item
    Handle("<Enter>", func(item *Item) {  // action on selected item
        // item is a pointer to the selected element
    })
```

### Methods

| Method | Description |
|--------|-------------|
| `Render(fn func(*T) any)` | Custom item rendering |
| `Marker(s string)` | Selection marker (default `"> "`) |
| `MaxVisible(n int)` | Max visible items (0 = all) |
| `Selection(sel *int)` | Bind to external selection index |
| `Style(s Style)` | Default row style |
| `SelectedStyle(s Style)` | Selected row style |
| `MarkerStyle(s Style)` | Marker style |
| `BindNav(down, up string)` | Bind navigation keys |
| `BindVimNav()` | j/k, Ctrl-d/u, g/G |
| `BindDelete(key string)` | Bind delete key |
| `Handle(key, fn func(*T))` | Action on selected item |
| `Selected() *T` | Get selected item |
| `Index() int` | Get selected index |

## FilterList

Drop-in filterable list with fzf-style fuzzy matching. Composes an input,
filter and selection list into a single template node:

```go
FilterList(&items, func(s *string) string { return *s }).
    Placeholder("type to filter...").
    MaxVisible(20).
    Render(func(s *string) any { return Text(s) }).
    Handle("<Enter>", func(s *string) {
        fmt.Println("selected:", *s)
    }).
    HandleClear("<Esc>", app.Stop)
```

The second argument extracts searchable text from each item. For structs:

```go
type Profile struct {
    Name    string
    Service string
}

FilterList(&profiles, func(p *Profile) string {
    return p.Name + " " + p.Service
}).Render(func(p *Profile) any {
    return HBox(Text(&p.Name), Space(), Text(&p.Service).Dim())
})
```

Navigation uses Ctrl-n/Ctrl-p by default (no conflict with text input).
All printable keys go to the filter input.

### Query Syntax

Inherits fzf query syntax:

| Pattern | Meaning |
|---------|---------|
| `foo` | Fuzzy match |
| `'foo` | Exact substring |
| `^foo` | Prefix match |
| `foo$` | Suffix match |
| `!foo` | Negated match |
| `a b` | AND (both must match) |
| `a \| b` | OR (either matches) |

### Methods

| Method | Description |
|--------|-------------|
| `Placeholder(s string)` | Input placeholder text |
| `Render(fn func(*T) any)` | Custom item rendering |
| `MaxVisible(n int)` | Max visible items |
| `Handle(key, fn func(*T))` | Action on selected original item |
| `HandleClear(key, fallback)` | Clear filter if active, else fallback |
| `BindNav(down, up string)` | Override nav keys |
| `Selected() *T` | Selected item in original slice |
| `SelectedIndex() int` | Index in original slice |
| `Clear()` | Reset filter and input |
| `Active() bool` | Whether a filter is applied |
| `Border(b BorderStyle)` | Border style |
| `Title(s string)` | Border title |
| `Style(s Style)` | Default row style |
| `SelectedStyle(s Style)` | Selected row style |
| `Marker(s string)` | Selection marker |

## Input

Text input with declarative binding:

```go
Input().
    Placeholder("Enter text...").
    Width(30).
    Bind()
```

`.Bind()` routes unmatched keys to the input automatically — arrow keys,
backspace, Ctrl-a/e/k/u all work. No manual `HandleUnmatched` needed.

Access the value:

```go
var myInput *InputC

Input().
    Placeholder("Search...").
    Bind().
    Ref(func(i *InputC) { myInput = i })

// later
myInput.Value()     // current text
myInput.SetValue(s) // set text
myInput.Clear()     // reset
```

For password fields:

```go
Input().Placeholder("Password").Mask('*').Bind()
```

### Raw TextInput

The lower-level `TextInput` struct is still available:

```go
field := Field{}

TextInput{
    Field:       &field,
    Placeholder: "Enter text",
    Width:       30,
    Mask:        '*',
}
```

## LayerView

Display scrollable Layer content:

```go
layer := NewLayer()
layer.SetBuffer(contentBuffer)

LayerView(layer)
LayerView(layer).Grow(1)        // Fill available space
LayerView(layer).ViewHeight(20) // Fixed viewport height
```

## Overlay

Modal/popup:

```go
If(&showModal).Then(
    Overlay.Centered().Backdrop()(
        VBox.Width(50).Border(BorderRounded).Fill(PaletteColor(236))(
            Text("Title").Bold(),
            SpaceH(1),
            Text("Content"),
        ),
    ),
)
```

Modifiers:

```go
Overlay(children...)                    // Basic overlay
Overlay.Centered()(...)                 // Centered on screen
Overlay.Backdrop()(...)                 // Dim background
Overlay.At(10, 5)(...)                  // Position at x=10, y=5
Overlay.Size(60, 20)(...)               // Fixed size
Overlay.BG(PaletteColor(236))(...)      // Background color
Overlay.Centered().Backdrop().BG(c)(...) // Chain modifiers
```

## Jump

Vim-easymotion style labels:

```go
Jump(Text("Click me"), func() {
    // handle selection
})

Jump(Text("Styled"), onSelect).Style(Style{FG: Magenta})
```

Activate with `app.EnterJumpMode()`.

## Tabs

```go
selected := 0
modes := []string{"NAV", "EDIT", "HELP"}

Tabs(modes, &selected).
    Style(TabsStyleBracket).
    Gap(2).
    ActiveStyle(Style{FG: Cyan, Attr: AttrBold}).
    InactiveStyle(Style{FG: BrightBlack})
```

Tab styles:

```go
TabsStyleBracket    // [NAV] [EDIT] [HELP]
TabsStyleUnderline  // NAV  EDIT  HELP  (active item underlined)
TabsStyleBox        // boxed tab style
```

## Widget

Fully custom components when you need complete control:

```go
Widget(
    // Measure: return natural size given available width
    func(availW int16) (w, h int16) {
        return 20, 3
    },
    // Render: draw directly to buffer
    func(buf *Buffer, x, y, w, h int16) {
        buf.WriteString(int(x), int(y), "Custom content", Style{})
    },
)
```

### Gradient Progress Bar

```go
progress := 0.65

Widget(
    func(availW int16) (w, h int16) { return availW, 1 },
    func(buf *Buffer, x, y, w, h int16) {
        filled := int(float64(w) * progress)
        for i := int16(0); i < w; i++ {
            if int(i) < filled {
                buf.Set(int(x+i), int(y), Cell{Rune: '█', Style: Style{FG: Green}})
            } else {
                buf.Set(int(x+i), int(y), Cell{Rune: '░', Style: Style{FG: BrightBlack}})
            }
        }
    },
)
```

### Sparkline

```go
data := []float64{0.2, 0.5, 0.8, 0.3, 0.9}

Widget(
    func(availW int16) (w, h int16) { return int16(len(data)), 1 },
    func(buf *Buffer, x, y, w, h int16) {
        chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
        for i, v := range data {
            idx := int(v * float64(len(chars)-1))
            buf.Set(int(x)+i, int(y), Cell{Rune: chars[idx], Style: Style{FG: Cyan}})
        }
    },
)
```

Use Widget when built-in components don't fit your needs. You handle all measurement and rendering.

