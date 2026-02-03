# Styling

## Style Struct

```go
Style{
    FG:        Cyan,
    BG:        Black,
    Fill:      Blue,                // fill empty cells in this container
    Attr:      AttrBold | AttrUnderline,
    Transform: Uppercase,          // text transform (Uppercase, Lowercase, Capitalize)
}
```

## Colors

### Named Colors (16-color)

```go
Black, Red, Green, Yellow, Blue, Magenta, Cyan, White
BrightBlack, BrightRed, BrightGreen, BrightYellow
BrightBlue, BrightMagenta, BrightCyan, BrightWhite
```

### 256-Color Palette

```go
PaletteColor(0)    // Black
PaletteColor(196)  // Bright red
PaletteColor(236)  // Dark gray (good for modal backgrounds)
```

### RGB (True Color)

Terminal support varies.

```go
RGB(255, 128, 0)  // Orange
```

### Default

```go
DefaultColor()  // Terminal default
```

## Text Styling

Method chaining:

```go
Text("Hello").FG(Red)
Text("Hello").BG(Blue)
Text("Hello").Bold()
Text("Hello").Dim()
Text("Hello").Underline()
Text("Hello").FG(Cyan).Bold().Underline()
```

Using Style struct:

```go
Text("Hello").Style(Style{
    FG:   Yellow,
    BG:   Black,
    Attr: AttrBold,
})
```

## Attributes

```go
AttrBold
AttrDim
AttrItalic
AttrUnderline
AttrBlink
AttrInverse
AttrStrikethrough
```

Combine with bitwise OR:

```go
Attr: AttrBold | AttrUnderline
```

## Container Styling

Fill container area:

```go
VBox.Fill(PaletteColor(236))(
    Text("Content"),
)
```

Border color:

```go
VBox.Border(BorderRounded).BorderFG(Cyan)(...)
VBox.Border(BorderRounded).BorderBG(Black)(...)
```

## Component-Specific Styling

### HRule / VRule

```go
HRule().Style(Style{FG: BrightBlack})
HRule().Char('=')
```

### Space (fill character)

```go
Space().Char('.').Style(Style{FG: BrightBlack})
```

### Leader

```go
Leader("Key", "Value").Style(Style{FG: Cyan})
Leader("Key", "Value").Fill('.')
```

### Progress

```go
Progress(75).Style(Style{FG: Green})
```

### Spinner

```go
Spinner(&frame).Style(Style{FG: Cyan})
```

### List / FilterList

```go
List(&items).
    Style(Style{BG: PaletteColor(235)}).          // Normal items
    SelectedStyle(Style{BG: PaletteColor(238)}).  // Selected item
    MarkerStyle(Style{FG: Cyan})                  // Selection marker

FilterList(&items, extract).
    Style(Style{BG: PaletteColor(235)}).
    SelectedStyle(Style{BG: PaletteColor(238)})
```

## Container Fill

Fill a container's entire area with a color:

```go
VBox.Fill(PaletteColor(236))(
    Text("Content"),  // empty space around text is also filled
)

HBox.Fill(Blue)(
    Text("Left"),
    Text("Right"),
)
```

Fill does NOT cascade to children — it only fills the container itself.

## Style Inheritance

Containers can pass styles to their children using `CascadeStyle`:

```go
theme := Style{FG: Cyan, BG: Black, Attr: AttrBold}

VBox.CascadeStyle(&theme)(
    Text("Inherits Cyan FG, Black BG, Bold"),
    Text("Same here"),
)
```

### Inheritance Rules

**Empty style inherits everything:**

```go
VBox.CascadeStyle(&Style{FG: Red, Attr: AttrBold})(
    Text("Hello"),  // Red + Bold
)
```

**Partial style merges selectively:**

When a child has an explicit style, only certain properties merge:

```go
VBox.CascadeStyle(&Style{FG: Red, Attr: AttrBold})(
    Text("Override").FG(Green),  // Green FG, Bold (attr merged)
)
```

- `Attr` — merged with bitwise OR (parent + child combined)
- `Transform` — inherited if child doesn't set one
- `FG`, `BG` — NOT inherited when child sets any style property

### Container Fill vs Style Fill

Two different mechanisms:

**`.Fill()` on VBox/HBox** — fills the container's area with a color. Does not cascade.

```go
VBox.Fill(PaletteColor(236))(...)  // only this container is filled
```

**`Fill` field on `Style`** — when set via `CascadeStyle`, cascades to nested containers:

```go
VBox.CascadeStyle(&Style{FG: White, Fill: Blue})(
    VBox.CascadeStyle(&Style{FG: Yellow})(  // no Fill specified
        Text("Yellow on Blue"),              // inherits Blue fill
    ),
)
```

### Scoped Inheritance

Nested containers create new scopes:

```go
VBox.CascadeStyle(&Style{FG: Red})(
    Text("Red"),
    VBox.CascadeStyle(&Style{FG: Green})(
        Text("Green"),
    ),
    Text("Red again"),  // parent style restored
)
```

Works through conditionals and loops:

```go
VBox.CascadeStyle(&baseStyle)(
    If(&showDetails).Then(
        Text("Inherits baseStyle"),
    ),
    ForEach(&items, func(item *Item) any {
        return Text(item.Name)  // each item inherits baseStyle
    }),
)
```

### Dynamic Themes

Because `CascadeStyle` uses a pointer, you can change themes at runtime:

```go
var theme = Style{FG: Cyan}

app.SetView(
    VBox.CascadeStyle(&theme)(
        Text("Themed content"),
    ),
)

// later...
theme = Style{FG: Magenta}  // takes effect on next render
```

See `cmd/themedemo` for a working example with theme switching.
