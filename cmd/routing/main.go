package main

import (
	"log"

	"riffkey"
	"tui"
)

// Home view state
var home = struct {
	Title   string
	Counter int
}{
	Title:   "Home Screen",
	Counter: 0,
}

// Settings view state
var settings = struct {
	Title  string
	Volume int
}{
	Title:  "Settings",
	Volume: 50,
}

// Help modal state
var help = struct {
	Title string
	Text  string
}{
	Title: "Help",
	Text:  "Press Esc to close",
}

func main() {
	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Global handler (works on all views)
	app.Handle("q", func(_ riffkey.Match) {
		app.Stop()
	})

	// Home view
	app.View("home", homeView()).
		Handle("j", func(_ riffkey.Match) {
			home.Counter++
		}).
		Handle("k", func(_ riffkey.Match) {
			home.Counter--
		}).
		Handle("s", func(_ riffkey.Match) {
			app.Go("settings")
		}).
		Handle("?", func(_ riffkey.Match) {
			app.PushView("help")
		})

	// Settings view
	app.View("settings", settingsView()).
		Handle("j", func(_ riffkey.Match) {
			if settings.Volume > 0 {
				settings.Volume--
			}
		}).
		Handle("k", func(_ riffkey.Match) {
			if settings.Volume < 100 {
				settings.Volume++
			}
		}).
		Handle("<Esc>", func(_ riffkey.Match) {
			app.Go("home")
		}).
		Handle("?", func(_ riffkey.Match) {
			app.PushView("help")
		})

	// Help modal
	app.View("help", helpView()).
		Handle("<Esc>", func(_ riffkey.Match) {
			app.PopView()
		})

	// Start on home
	if err := app.RunFrom("home"); err != nil {
		log.Fatal(err)
	}
}

func homeView() any {
	return tui.VBox{Children: []any{
		tui.Text{Content: &home.Title, Style: tui.Style{Attr: tui.AttrBold}},
		tui.Text{},
		tui.Text{Content: "j/k: change counter"},
		tui.Text{Content: "s: go to settings"},
		tui.Text{Content: "?: help"},
		tui.Text{Content: "q: quit"},
		tui.Text{},
		tui.Progress{Value: &home.Counter, BarWidth: 30},
	}}
}

func settingsView() any {
	return tui.VBox{Children: []any{
		tui.Text{Content: &settings.Title, Style: tui.Style{Attr: tui.AttrBold}},
		tui.Text{},
		tui.Text{Content: "j/k: adjust volume"},
		tui.Text{Content: "Esc: back to home"},
		tui.Text{Content: "?: help"},
		tui.Text{},
		tui.Text{Content: "Volume:"},
		tui.Progress{Value: &settings.Volume, BarWidth: 30},
	}}
}

func helpView() any {
	return tui.VBox{Children: []any{
		tui.Text{Content: &help.Title, Style: tui.Style{Attr: tui.AttrBold}},
		tui.Text{},
		tui.Text{Content: &help.Text},
	}}
}
