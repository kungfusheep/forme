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
VBox.Style(&Style{BG: Black})(...) // Background style
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
selected := 0

list := List(&items, &selected).
    Marker("> ").
    MaxVisible(10).
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
selected := 0

list := List(&items, &selected).
    Render(func(item *Item) any {
        return HBox(
            Text(&item.Icon),
            Space(),
            Text(&item.Name),
        )
    }).
    SelectedStyle(Style{BG: PaletteColor(236)})
```

Navigation methods:

```go
app.Handle("j", func(_ riffkey.Match) { list.Down(nil) })
app.Handle("k", func(_ riffkey.Match) { list.Up(nil) })
app.Handle("g", func(_ riffkey.Match) { list.First(nil) })
app.Handle("G", func(_ riffkey.Match) { list.Last(nil) })
```

## TextInput

```go
field := Field{}
focus := FocusGroup{}

TextInput{
    Field:       &field,
    FocusGroup:  &focus,
    FocusIndex:  0,
    Placeholder: "Enter text",
    Width:       30,
    Mask:        '*',  // For passwords
}
```

Handle text input in router:

```go
handler := riffkey.NewTextHandler(&field.Value, &field.Cursor)
app.Router().HandleUnmatched(func(k riffkey.Key) bool {
    return handler.HandleKey(k)
})
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
        VBox.Width(50).Border(BorderRounded).Style(&Style{BG: PaletteColor(236)})(
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
TabsStyleBracket  // [NAV] [EDIT] [HELP]
TabsStylePipe     // NAV | EDIT | HELP
TabsStyleNone     // NAV  EDIT  HELP
```

