# API Reference

## App

```go
app, err := NewApp()
```

### Methods

| Method | Description |
|--------|-------------|
| `SetView(view any)` | Set the root view |
| `Handle(pattern string, fn func(riffkey.Match))` | Register key handler |
| `Run() error` | Start the app |
| `Stop()` | Exit the app |
| `RenderNow()` | Force immediate render |
| `EnterJumpMode()` | Activate jump label mode |
| `ExitJumpMode()` | Deactivate jump label mode |

### Multi-View (Router)

```go
app.View("home", homeUI).
    Handle("1", func(_ riffkey.Match) { app.Go("settings") })

app.View("settings", settingsUI).
    Handle("q", func(_ riffkey.Match) { app.Back() })

app.RunFrom("home")
```

| Method | Description |
|--------|-------------|
| `View(name string, view any)` | Register a named view |
| `Go(name string)` | Navigate to view |
| `Back()` | Return to previous view |
| `RunFrom(name string)` | Start from named view |

## Dynamic Values

Pass pointers so values are read at render time:

```go
count := 0
app.SetView(Text(&count))

app.Handle("j", func(_ riffkey.Match) {
    count++
    // Re-render happens automatically after input handlers
})
```

Rendering occurs:
- After input handlers complete
- When `RenderNow()` is called explicitly

```go
// Force immediate render (e.g., from a goroutine)
app.RenderNow()
```

## Layers

Scrollable content areas:

```go
layer := NewLayer()
buf := NewBuffer(80, 1000)
// ... write to buf ...
layer.SetBuffer(buf)
```

| Method | Description |
|--------|-------------|
| `SetBuffer(buf *Buffer)` | Set content buffer |
| `ScrollDown(n int)` | Scroll down n lines |
| `ScrollUp(n int)` | Scroll up n lines |
| `PageDown()` | Scroll one page down |
| `PageUp()` | Scroll one page up |
| `HalfPageDown()` | Scroll half page down |
| `HalfPageUp()` | Scroll half page up |
| `ScrollToTop()` | Jump to top |
| `ScrollToEnd()` | Jump to bottom |
| `ScrollY() int` | Current scroll position |
| `MaxScroll() int` | Maximum scroll position |

## Buffer

Low-level drawing:

```go
buf := NewBuffer(80, 24)
buf.WriteString(x, y, "text", style)
buf.WriteStringFast(x, y, "text", style, maxWidth)
buf.Set(x, y, Cell{Rune: 'X', Style: style})
buf.Get(x, y) Cell
buf.Clear()
```
