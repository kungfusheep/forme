# glyph

Declarative terminal UI framework for Go. **[useglyph.sh](https://useglyph.sh)**

![hero](./assets/hero.gif)

```go
VBox.Border(BorderDouble).Title("SYS").FitContent()(
    If(&online).
        Then(Text("● ONLINE")).
        Else(Text("● OFFLINE").FG(Red)),
    HRule(),
    Leader("CPU", &cpu),
    Leader("MEM", &mem),
    Sparkline(&history),
)
```

- **Declarative.** UI is a tree of typed values; glyph compiles the layout and renders each frame.
- **Flex layout.** VBox/HBox with grow, gap, borders, and cascading styles.
- **Components.** Lists, tables, inputs, fuzzy filtering, sparklines, tabs, tree views, overlays, vim bindings, theming.
- **Your data, directly.** Pass a pointer; glyph reads the current value every update.

## Requirements

Go 1.25 or later. macOS and Linux only; Windows support is on the roadmap.

## Install

```bash
go get github.com/kungfusheep/glyph
```

## Quick start

```go
package main

import (
    "log"
    . "github.com/kungfusheep/glyph"
)

func main() {
    app, err := NewApp()
    if err != nil {
        log.Fatal(err)
    }
    app.SetView(Text("Hello, terminal!")).Run()
}
```

## State

The core is stable and has been used to build real things. The API covers the full range of what a TUI framework needs. Pre-1.0 because the surface is still under evaluation; parts of it can be better still, and 1.0 means committing to it.

## Demos

Run any of the included demos to see the framework in action:

| Demo | Description |
|------|-------------|
| `go run ./cmd/hero` | The hero screenshot above |
| `go run ./cmd/todo` | Todo app with checklist |
| `go run ./cmd/glyph-fzf` | Fuzzy finder with FilterList |
| `go run ./cmd/happypath` | Basic layout patterns |
| `go run ./cmd/tabledemo` | AutoTable showcase |
| `go run ./cmd/widgetdemo` | Custom widget examples |
| `go run ./cmd/jumpdemo` | Vim-style jump labels |
| `go run ./cmd/routing` | Multi-view navigation |
| `go run ./cmd/minivim` | Full text editor |

## Docs

Full documentation, API reference, and getting started guide at **[useglyph.sh](https://useglyph.sh)**.

## License

Apache-2.0. See [LICENSE](./LICENSE) for details.
