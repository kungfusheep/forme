# API Reference

## App

```go
app, err := NewApp()
```

### Methods

| Method | Description |
|--------|-------------|
| `SetView(view any)` | Set the root view (single-view apps) |
| `View(name, view any)` | Register a named view |
| `Handle(pattern string, fn)` | Register key handler (`func()`, `func(any)`, `func(riffkey.Match)`) |
| `Run() error` | Start the app |
| `RunFrom(name string)` | Start from a named view |
| `Stop()` | Exit the app |
| `RequestRender()` | Request a render (safe from any goroutine) |
| `RenderNow()` | Force immediate render |
| `OnBeforeRender(fn func())` | Callback before each render |
| `OnAfterRender(fn func())` | Callback after each render |
| `OnResize(fn func(w, h int))` | Callback on terminal resize |
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

app.Handle("j", func() {
    count++
    // re-render happens automatically after input handlers
})
```

Rendering occurs:
- After input handlers complete
- When `RequestRender()` is called (safe from any goroutine)
- When `RenderNow()` is called (immediate, mutex-protected)

```go
// from a goroutine
app.RequestRender()
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
