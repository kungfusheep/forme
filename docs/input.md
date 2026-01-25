# Input Handling

Input handling uses [riffkey](https://github.com/...) for vim-style key patterns.

## Basic Handlers

```go
app.Handle("q", func(_ riffkey.Match) {
    app.Stop()
})
```

## Key Patterns

### Single Keys

```go
app.Handle("q", handler)
app.Handle("j", handler)
app.Handle("1", handler)
```

### Sequences

```go
app.Handle("gg", handler)      // g then g
app.Handle("dd", handler)      // d then d
```

### Modifiers

```go
app.Handle("<C-c>", handler)   // Ctrl+C
app.Handle("<C-w>", handler)   // Ctrl+W
app.Handle("<S-Tab>", handler) // Shift+Tab
app.Handle("<C-S-a>", handler) // Ctrl+Shift+A
```

### Special Keys

```go
app.Handle("<Enter>", handler)
app.Handle("<Escape>", handler)
app.Handle("<Tab>", handler)
app.Handle("<Space>", handler)
app.Handle("<Backspace>", handler)
app.Handle("<Delete>", handler)
app.Handle("<Up>", handler)
app.Handle("<Down>", handler)
app.Handle("<Left>", handler)
app.Handle("<Right>", handler)
app.Handle("<Home>", handler)
app.Handle("<End>", handler)
app.Handle("<PageUp>", handler)
app.Handle("<PageDown>", handler)
app.Handle("<F1>", handler)    // F1-F12
```

### Multi-Key Sequences

```go
app.Handle("<C-w>j", handler)  // Ctrl+W then j
app.Handle("<C-w>k", handler)  // Ctrl+W then k
```

## Unmatched Input

Handle keys that don't match any pattern:

```go
app.Router().HandleUnmatched(func(k riffkey.Key) bool {
    // Return true if handled, false to ignore
    return false
})
```

Useful for text input:

```go
handler := riffkey.NewTextHandler(&field.Value, &field.Cursor)
app.Router().HandleUnmatched(func(k riffkey.Key) bool {
    return handler.HandleKey(k)
})
```

## View-Specific Handlers

Handlers can be set on views:

```go
app.SetView(
    VBox(...),
).
    Handle("j", down).
    Handle("k", up).
    Handle("q", quit)
```

Or separately:

```go
app.SetView(VBox(...))
app.Handle("j", down)
app.Handle("k", up)
```

## Handler Function

```go
func(m riffkey.Match) {
    // m.Key - the key that was pressed
    // m.Pattern - the pattern that matched
}
```

## Common Patterns

### Navigation

```go
app.Handle("j", func(_ riffkey.Match) { moveDown() })
app.Handle("k", func(_ riffkey.Match) { moveUp() })
app.Handle("<Down>", func(_ riffkey.Match) { moveDown() })
app.Handle("<Up>", func(_ riffkey.Match) { moveUp() })
app.Handle("g", func(_ riffkey.Match) { goToTop() })
app.Handle("G", func(_ riffkey.Match) { goToBottom() })
```

### Scrolling

```go
app.Handle("j", func(_ riffkey.Match) { layer.ScrollDown(1) })
app.Handle("k", func(_ riffkey.Match) { layer.ScrollUp(1) })
app.Handle("<C-d>", func(_ riffkey.Match) { layer.HalfPageDown() })
app.Handle("<C-u>", func(_ riffkey.Match) { layer.HalfPageUp() })
app.Handle("<Space>", func(_ riffkey.Match) { layer.PageDown() })
```

### Modal Toggle

```go
showModal := false

app.Handle("m", func(_ riffkey.Match) {
    showModal = !showModal
})

app.Handle("<Escape>", func(_ riffkey.Match) {
    if showModal {
        showModal = false
    } else {
        app.Stop()
    }
})
```

### Tab Cycling

```go
selected := 0
count := 3

app.Handle("<Tab>", func(_ riffkey.Match) {
    selected = (selected + 1) % count
})

app.Handle("<S-Tab>", func(_ riffkey.Match) {
    selected = (selected + count - 1) % count
})
```
