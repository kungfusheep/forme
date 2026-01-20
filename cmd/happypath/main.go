package main

import (
	"log"
	"time"

	"riffkey"
	"tui"
)

type State struct {
	Counter  int
	Progress int
	Items    []Item
}

type Item struct {
	Name     string
	Progress int
}

func main() {
	state := &State{
		Counter:  0,
		Progress: 25,
		Items: []Item{
			{Name: "Task 1", Progress: 80},
			{Name: "Task 2", Progress: 45},
			{Name: "Task 3", Progress: 10},
		},
	}

	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	var boolFlag = true

	// Set the view - batteries included!
	app.SetView(
		tui.VBox{Children: []any{
			tui.Text{Content: "Happy Path.emo", Style: tui.Style{Attr: tui.AttrBold}},
			tui.Text{},
			tui.Text{Content: "Press j/k to change values, q to quit"},
			tui.Text{},
			tui.Progress{Value: &state.Progress, BarWidth: 40},
			tui.If(&boolFlag).Eq(true).Then(tui.Text{Content: "Counter would be here"}),
			tui.Text{},
			tui.HBox{Gap: 2, Children: []any{
				tui.Text{Content: "Counter:"},
				tui.ForEach(&state.Items, func(item *Item) any {
					return tui.Text{Content: &item.Name}
				}),
			}},
		}},
	).
		Handle("q", func(m riffkey.Match) {
			app.Stop()
		}).
		Handle("j", func(m riffkey.Match) {
			state.Counter++
			state.Progress = (state.Progress + 5) % 101
			for i := range state.Items {
				state.Items[i].Progress = (state.Items[i].Progress + 3) % 101
			}
		}).
		Handle("k", func(m riffkey.Match) {
			state.Counter--
			state.Progress = (state.Progress - 5 + 101) % 101
		})

	// Start a ticker to auto-update
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			state.Progress = (state.Progress + 1) % 101
			app.RequestRender()
		}
	}()

	// Run the app
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
