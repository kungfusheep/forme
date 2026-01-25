# Styling

## Style Struct

```go
Style{
    FG:   Cyan,
    BG:   Black,
    Attr: AttrBold | AttrUnderline,
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
RGBColor(255, 128, 0)  // Orange
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
AttrUnderline
AttrBlink
AttrReverse
```

Combine with bitwise OR:

```go
Attr: AttrBold | AttrUnderline
```

## Container Styling

Background on container:

```go
VBox.Style(&Style{BG: PaletteColor(236)})(
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

### SelectionList

```go
&SelectionList{
    Style:         Style{BG: PaletteColor(235)},      // Normal items
    SelectedStyle: Style{BG: PaletteColor(238)},      // Selected item
    MarkerStyle:   Style{FG: Cyan},                   // Selection marker
}
```
