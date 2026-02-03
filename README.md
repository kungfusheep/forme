<!-- ![logo](./logo.svg) -->

# forme

Declarative terminal UI for Go.

![hero](./hero.gif)

```go
VBox.Border(BorderDouble).Title("SYS").FitContent()(
    If(&online).
        Then(Text("● ONLINE")).
        Else(Text("● OFFLINE").FG(Red)),
    HRule(),
    Leader("CPU", &cpu),
    Leader("MEM", &mem),
    Sparkline(&history),
),
```

- **Declarative** — describe your UI as a tree, not a sequence of draw calls
- **Fast** — compiled templates, layered rendering, blitting for smooth scrolling
- **Flexbox layout** — VBox/HBox with Grow, Gap, Width, borders, cascading styles
- **Batteries included** — sparklines, tables, leaders, progress bars, lists, text input, vim-style jump labels

## Install

```bash
go get github.com/kungfusheep/forme
```

## Quick Start

```go
package main

import . "github.com/kungfusheep/forme"

func main() {
    app, _ := NewApp()
    app.SetView(Text("Hello, terminal!")).Run()
}
```

---

## Dynamic Content

Pass a pointer to read the current value on each render:

```go
name := "world"
Text(&name)  // reads *name each frame
```

## Layout

```go
VBox(                          // vertical stack
    Text("Header"),
    HBox(                      // horizontal stack
        Text("Left"),
        Space(),               // flexible spacer
        Text("Right"),
    ),
    Text("Footer"),
)

VBox.Gap(2)(...)               // spacing between children
VBox.Border(BorderRounded)(...)// bordered container
HBox.Width(41)(...)            // fixed width
```

## Styling

```go
Text("hello").Bold().FG(Red).BG(Black)

// Colors: Red, Green, Blue, Cyan, Magenta, Yellow, White, Black
//         BrightRed, BrightGreen, ...
//         PaletteColor(202)     // 256 palette
//         RGB(255, 128, 64)     // true color
//         Hex(0xFF8041)
```

### CascadeStyle

Apply a style that cascades down to all children:

```go
green := Style{FG: Green}

VBox.CascadeStyle(&green)(
    Text("I'm green"),
    Text("Me too"),
    Text("Override").FG(Red),  // still works
)
```

## Iteration

```go
items := []Item{{Name: "Apple"}, {Name: "Banana"}}

ForEach(&items, func(item *Item) any {
    return Text(&item.Name)
})
```

## Conditionals

```go
If(&selected).Then(
    Text("> active"),
).Else(
    Text("  inactive"),
)
```

## AutoTable

Struct slice to table:

```go
people := []Person{{"Alice", 30}, {"Bob", 25}}

AutoTable(people)                    // all fields auto-detected
AutoTable(people).Columns("Name")    // select columns
```

## Custom Widgets

```go
Widget(
    func(availW int16) (w, h int16) { return availW, 1 },
    func(buf *Buffer, x, y, w, h int16) {
        // direct buffer access for custom rendering
    },
)
```

## Input Handling

Key bindings can be declared on components or on the app/view:

```go
// declarative — on the component
List(&items).BindVimNav().Handle("<Enter>", func(item *Item) { ... })

// view-level
app.View("main", myUI).
    Handle("q", app.Stop).
    Handle("<C-c>", app.Stop)
```

See [docs/input.md](docs/input.md) for full key pattern syntax.

## Text Input

```go
Input().Placeholder("Search...").Bind()
```

`.Bind()` routes unmatched keys to the input. Arrow keys, backspace,
Ctrl-a/e/k/u all work out of the box.

## FilterList

Drop-in fuzzy-filterable list:

```go
FilterList(&items, func(s *string) string { return *s }).
    Placeholder("type to filter...").
    Render(func(s *string) any { return Text(s) }).
    Handle("<Enter>", func(s *string) { ... }).
    HandleClear("<Esc>", app.Stop)
```

Supports fzf query syntax: `foo` (fuzzy), `'exact`, `^prefix`, `suffix$`,
`!negation`, `a b` (AND), `a | b` (OR).

## Components

| Component | Description |
|-----------|-------------|
| `Text(s)` | Text (string or *string) |
| `VBox` / `HBox` | Vertical / horizontal layout |
| `Space()` | Flexible spacer |
| `HRule()` / `VRule()` | Horizontal / vertical line |
| `Leader` | Label with dot leader |
| `Progress` | Progress bar |
| `Sparkline` | Mini line chart |
| `Spinner` | Animated spinner |
| `ForEach` | Slice iteration |
| `If().Then().Else()` | Conditional |
| `Switch().Case()` | Multi-branch conditional |
| `AutoTable` | Struct slice to table |
| `List` | Navigable selection list |
| `CheckList` | List with checkboxes |
| `FilterList` | Fuzzy-filterable list with input |
| `Input` | Text input with declarative binding |
| `Radio` | Radio button group |
| `Tabs` | Tab bar |
| `Widget` | Custom render function |
| `Overlay` | Modal / popup |
| `Jump` | Vim-easymotion labels |

## Full Example

A simple todo app:

```go
package main

import . "github.com/kungfusheep/forme"

type Todo struct {
	Text string `forme:"render"`
	Done bool   `forme:"checked"`
}

func main() {
	todos := []Todo{{"Learn forme", true}, {"Build something", false}}
	var input Field

	app, _ := NewApp()
	app.SetView(
		VBox.Border(BorderRounded).Title("Todo").FitContent().Gap(1)(
			CheckList(&todos).
				BindNav("<C-n>", "<C-p>").
				BindToggle("<tab>").
				BindDelete("<C-d>"),
			HBox.Gap(1)(
				Text("Add:"),
				TextInput{Field: &input, Width: 30},
			),
		)).
		Handle("<Enter>", func() {
			if input.Value != "" {
				todos = append(todos, Todo{Text: input.Value})
				input.Clear()
			}
		}).
		Handle("<C-c>", app.Stop).
		BindField(&input).
		Run()
}
```

A fuzzy finder:

```go
package main

import (
	"fmt"
	. "github.com/kungfusheep/forme"
)

func main() {
	languages := []string{"Go", "Rust", "Python", "JavaScript", "TypeScript"}
	status := "type to filter"

	app, _ := NewApp()
	app.View("main",
		VBox.Border(BorderRounded).Title("fuzzy finder")(
			FilterList(&languages, func(s *string) string { return *s }).
				Placeholder("type to filter...").
				Render(func(s *string) any { return Text(s) }).
				Handle("<Enter>", func(s *string) {
					status = fmt.Sprintf("selected: %s", *s)
				}).
				HandleClear("<Esc>", app.Stop),
			Space(),
			HRule(),
			Text(&status).Dim(),
		),
	).Handle("q", app.Stop)

	app.RunFrom("main")
}
```

## Demos

| Demo | Description |
|------|-------------|
| `go run ./cmd/hero` | The hero screenshot above |
| `go run ./cmd/todo` | Todo app with checklist |
| `go run ./cmd/forme-fzf` | Fuzzy finder with FilterList |
| `go run ./cmd/happypath` | Basic layout patterns |
| `go run ./cmd/tabledemo` | AutoTable showcase |
| `go run ./cmd/widgetdemo` | Custom widget examples |
| `go run ./cmd/jumpdemo` | Vim-style jump labels |
| `go run ./cmd/routing` | Multi-view navigation |
| `go run ./cmd/minivim` | Full text editor |

## License

Apache-2.0 License. See [LICENSE](./LICENSE) for details.
