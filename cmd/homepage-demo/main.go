package main

import (
	"os"
	"strings"

	. "github.com/kungfusheep/glyph"
)

func main() {
	dir, _ := os.ReadDir(".")
	files := make([]string, 0, len(dir))
	for _, f := range dir {
		if !strings.HasPrefix(f.Name(), ".") {
			files = append(files, f.Name())
		}
	}
	preview := ""
	showSidebar := true

	app, _ := NewApp()
	app.SetView(
		HBox(
			If(&showSidebar).Then(
				VBox.Width(25).Border(BorderRounded)(
					List(&files).OnSelect(func(f *string) {
						info, err := os.Stat(*f)
						if err != nil || info.IsDir() {
							preview = "Not a file"
							return
						}

						data, _ := os.ReadFile(*f)
						preview = string(data)
					}).BindVimNav(),
				),
			),
			VBox.Grow(2).Border(BorderRounded)(
				TextView(&preview).Grow(1).
					BindScroll("J", "K").BindPageScroll("<C-f>", "<C-b>"),
			),
		),
	).
		Handle("b", Toggle(&showSidebar)).
		Handle("q", app.Stop).
		Run()
}

func Toggle(b *bool) func() {
	return func() {
		*b = !*b
	}
}
