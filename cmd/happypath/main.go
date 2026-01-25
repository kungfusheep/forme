package main

import (
	"log"
	"time"

	"riffkey"
	. "forme"
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

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	var showCounter = true

	app.SetView(
		VBox(
			Text("Happy Path Demo").Bold(),
			SpaceH(1),
			Text("Press j/k to change values, q to quit"),
			SpaceH(1),
			Progress(&state.Progress).Width(40),
			If(&showCounter).Then(Text("Counter is visible!")),
			SpaceH(1),
			HBox.Gap(2)(
				Text("Tasks:"),
				ForEach(&state.Items, func(item *Item) any {
					return Text(&item.Name)
				}),
			),
		),
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

	// auto-update ticker
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			state.Progress = (state.Progress + 1) % 101
			app.RequestRender()
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
