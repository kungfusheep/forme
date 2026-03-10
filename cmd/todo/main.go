package main

import . "github.com/kungfusheep/glyph"

type Todo struct {
	Text string `glyph:"render"`
	Done bool   `glyph:"checked"`
}

func main() {

	todos := []Todo{{"Learn glyph", true}, {"Build something", false}}
	var input InputState

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
		Handle("<enter>", func() {
			if input.Value != "" {
				todos = append(todos, Todo{Text: input.Value})
				input.Clear()
			}
		}).
		Handle("<C-c>", app.Stop).
		BindField(&input).
		Run()
}
