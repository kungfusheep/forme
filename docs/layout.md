# Layout

## Container Direction

`VBox` lays out children vertically (top to bottom).
`HBox` lays out children horizontally (left to right).

```go
VBox(A, B, C)    HBox(A, B, C)

┌───┐            ┌───┬───┬───┐
│ A │            │ A │ B │ C │
├───┤            └───┴───┴───┘
│ B │
├───┤
│ C │
└───┘
```

## Gap

Space between children:

```go
VBox.Gap(1)(A, B, C)

┌───┐
│ A │
│   │  ← 1 line gap
├───┤
│ B │
│   │  ← 1 line gap
├───┤
│ C │
└───┘
```

## Fixed Dimensions

```go
VBox.Width(40)(...)   // 40 characters wide
VBox.Height(10)(...)  // 10 lines tall
```

## Percentage Width

```go
HBox(
    VBox.WidthPct(0.3)(...),  // 30% of parent
    VBox.WidthPct(0.7)(...),  // 70% of parent
)
```

## Flex Grow

Distribute remaining space:

```go
HBox(
    Text("Fixed"),
    Space(),           // Takes remaining space (default grow=1)
    Text("Fixed"),
)
```

Custom grow factors:

```go
HBox(
    VBox.Grow(1)(...),  // 1 part
    VBox.Grow(2)(...),  // 2 parts (twice as wide)
    VBox.Grow(1)(...),  // 1 part
)
```

Spacers with grow:

```go
HBox(
    Text("A"),
    Space().Grow(1),    // 1x space
    Text("B"),
    Space().Grow(2),    // 2x space (wider)
    Text("C"),
)
```

## Borders

```go
VBox.Border(BorderSingle)(...)
VBox.Border(BorderDouble)(...)
VBox.Border(BorderRounded)(...)
VBox.Border(BorderThick)(...)
```

```
BorderSingle    BorderDouble    BorderRounded   BorderThick
┌─────────┐     ╔═════════╗     ╭─────────╮     ┏━━━━━━━━━┓
│ content │     ║ content ║     │ content │     ┃ content ┃
└─────────┘     ╚═════════╝     ╰─────────╯     ┗━━━━━━━━━┛
```

Border styling:

```go
VBox.Border(BorderRounded).BorderFG(Cyan).Title("Panel")(...)

╭─Panel────────╮
│ content      │
╰──────────────╯
```

## Nesting

Containers nest freely:

```go
VBox(
    HBox(
        VBox.Border(BorderSingle)(
            Text("Panel 1"),
        ),
        VBox.Border(BorderSingle)(
            Text("Panel 2"),
        ),
    ),
    HRule(),
    Text("Footer"),
)
```

## Common Patterns

### Sidebar + Content

```go
HBox(
    VBox.Width(25).Border(BorderSingle)(
        Text("Sidebar"),
    ),
    VBox.Grow(1)(
        Text("Main content"),
    ),
)
```

### Header + Content + Footer

```go
VBox(
    Text("Header").Bold(),
    HRule(),
    VBox.Grow(1)(
        Text("Content"),
    ),
    HRule(),
    Text("Footer"),
)
```

### Centered Content

```go
HBox(
    Space(),
    VBox(
        Text("Centered"),
    ),
    Space(),
)
```

### Right-Aligned

```go
HBox(
    Space(),
    Text("Right"),
)
```

### Space Between

```go
HBox(
    Text("Left"),
    Space(),
    Text("Right"),
)
```

### Even Distribution

```go
HBox(
    Text("A"),
    Space(),
    Text("B"),
    Space(),
    Text("C"),
)
```

## Custom Layouts

When VBox/HBox aren't sufficient, use `Arrange` with a custom `LayoutFunc`:

```go
// LayoutFunc receives child sizes and available space, returns positions
type LayoutFunc func(children []ChildSize, availW, availH int) []Rect

Arrange(myLayoutFunc)(children...)
```

### Grid Layout

```go
func Grid(cols, cellW, cellH int) LayoutFunc {
    return func(children []ChildSize, availW, availH int) []Rect {
        rects := make([]Rect, len(children))
        for i := range children {
            col := i % cols
            row := i / cols
            rects[i] = Rect{
                X: col * cellW,
                Y: row * cellH,
                W: cellW,
                H: cellH,
            }
        }
        return rects
    }
}

// Usage
Arrange(Grid(3, 20, 5))(  // 3 columns, 20 wide, 5 tall
    Text("A"), Text("B"), Text("C"),
    Text("D"), Text("E"), Text("F"),
)
```

### Adaptive Layout

```go
func Masonry(colWidth int) LayoutFunc {
    return func(children []ChildSize, availW, availH int) []Rect {
        cols := availW / colWidth
        if cols < 1 { cols = 1 }

        colHeights := make([]int, cols)
        rects := make([]Rect, len(children))

        for i, child := range children {
            // Find shortest column
            minCol := 0
            for c := 1; c < cols; c++ {
                if colHeights[c] < colHeights[minCol] {
                    minCol = c
                }
            }

            rects[i] = Rect{
                X: minCol * colWidth,
                Y: colHeights[minCol],
                W: colWidth,
                H: child.MinH,
            }
            colHeights[minCol] += child.MinH
        }
        return rects
    }
}
```

## Direct Drawing

For completely custom rendering (not layout), use Layer with direct buffer access:

```go
layer := NewLayer()

layer.Render = func() {
    buf := layer.Buffer()
    w, h := buf.Width(), buf.Height()

    // Absolute positioning
    buf.WriteString(0, 0, "Top-left", Style{})
    buf.WriteString(w-10, 0, "Top-right", Style{})

    // Draw shapes
    for x := 0; x < 20; x++ {
        buf.Set(x, 5, Cell{Rune: '─', Style: Style{FG: Yellow}})
    }
}

VBox(
    Text("Header"),
    LayerView(layer).ViewHeight(20),
)
```

### Scrollable Content

```go
layer := NewLayer()

buf := NewBuffer(80, 1000)
for i := 0; i < 1000; i++ {
    buf.WriteString(0, i, fmt.Sprintf("Line %d", i), Style{})
}
layer.SetBuffer(buf)

VBox(
    LayerView(layer).ViewHeight(20),
    Text("j/k to scroll"),
)

// Scroll handlers
app.Handle("j", func(_ riffkey.Match) { layer.ScrollDown(1) })
app.Handle("k", func(_ riffkey.Match) { layer.ScrollUp(1) })
```

See [api.md](api.md#buffer) for Buffer methods.
