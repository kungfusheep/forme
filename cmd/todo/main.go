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
				BindNav(app, "<C-n>", "<C-p>").
				BindToggle(app, "<tab>").
				BindDelete(app, "<C-d>"),
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
