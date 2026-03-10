package main

import (
	"fmt"

	. "github.com/kungfusheep/glyph"
)

type Pkg struct {
	Name string
	Desc string
}

func main() {
	packages := []Pkg{
		{"glyph", "declarative TUI framework"},
		{"chi", "lightweight HTTP router"},
		{"cobra", "CLI application framework"},
		{"viper", "configuration management"},
		{"zap", "structured logging"},
		{"sqlc", "type-safe SQL"},
		{"ent", "entity framework for Go"},
		{"fx", "dependency injection"},
		{"golangci-lint", "linter aggregator"},
		{"air", "live reload for Go apps"},
		{"templ", "HTML templating"},
		{"bubbletea", "TUI framework"},
		{"huh", "terminal forms"},
		{"gum", "shell script components"},
		{"task", "task runner"},
		{"gotestsum", "test output formatter"},
		{"delve", "Go debugger"},
		{"gore", "Go REPL"},
		{"gofumpt", "strict gofmt"},
		{"mockgen", "mock generator"},
	}

	selected := ""

	app, _ := NewApp()
	app.SetView(
		VBox(
			FilterList(&packages, func(p *Pkg) string { return p.Name }).
				Render(func(p *Pkg) any {
					return HBox.Gap(2)(
						Text(&p.Name).Bold(),
						Text(&p.Desc).FG(BrightBlack),
					)
				}).
				MaxVisible(15).
				Border(BorderRounded).
				Title("packages").
				Handle("<Enter>", func(p *Pkg) {
					selected = fmt.Sprintf("selected: %s — %s", p.Name, p.Desc)
				}).
				HandleClear("<Esc>", app.Stop),
			Text(&selected).FG(Green),
		),
	)

	app.Run()
}
