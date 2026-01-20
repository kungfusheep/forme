package main

import (
	"fmt"
	"log"

	"riffkey"
	"tui"
)

func main() {
	// Modal state - pointer for reactivity
	showModal := false
	modalMessage := "This is a modal dialog!"

	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Build view once - modal visibility controlled by pointer
	app.SetView(
		tui.VBox{
			Children: []any{
				// Main content
				tui.Text{Content: "Overlay Demo", Style: tui.Style{FG: tui.Cyan, Attr: tui.AttrBold}},
				tui.HRule{Style: tui.Style{FG: tui.BrightBlack}},
				tui.Spacer{Height: 1},

				tui.Text{Content: "This is the main application content."},
				tui.Text{Content: "The modal will appear centered over this."},
				tui.Spacer{Height: 1},

				tui.HBox{
					Gap: 2,
					Children: []any{
						tui.VBox{Children: []any{
							tui.Text{Content: "Panel 1", Style: tui.Style{FG: tui.Yellow, Attr: tui.AttrBold}},
							tui.Text{Content: "Some content here"},
							tui.Text{Content: "More content"},
						}}.Border(tui.BorderSingle),

						tui.VBox{Children: []any{
							tui.Text{Content: "Panel 2", Style: tui.Style{FG: tui.Green, Attr: tui.AttrBold}},
							tui.Text{Content: "Different content"},
							tui.Text{Content: "Even more content"},
						}}.Border(tui.BorderSingle),
					},
				},

				tui.Spacer{Height: 1},
				tui.HRule{Style: tui.Style{FG: tui.BrightBlack}},
				tui.Text{Content: "Press 'm' to toggle modal | 'q' to quit", Style: tui.Style{FG: tui.BrightBlack}},

				// Modal overlay - controlled by tui.If
				tui.If(&showModal).Eq(true).Then(tui.Overlay{
					Backdrop: true,
					Centered: true,
					Child: tui.VBox{
						Children: []any{
							tui.Text{Content: "Modal Dialog  ", Style: tui.Style{FG: tui.Cyan, Attr: tui.AttrBold}},
							tui.Spacer{Height: 1},
							tui.Text{Content: &modalMessage, Style: tui.Style{FG: tui.White}},
							tui.Spacer{Height: 1},
							tui.Text{Content: "Press 'm' to close", Style: tui.Style{FG: tui.BrightBlack}},
						},
					}.Width(50).Border(tui.BorderRounded),
				}),
			},
		},
	)

	// Toggle modal with 'm'
	app.Handle("m", func(_ riffkey.Match) {
		showModal = !showModal
		if showModal {
			modalMessage = fmt.Sprintf("Modal opened! Press 'm' to close.")
		}
	})

	// Quit with 'q'
	app.Handle("q", func(_ riffkey.Match) {
		app.Stop()
	})

	// Also allow Escape to close modal or quit
	app.Handle("<Escape>", func(_ riffkey.Match) {
		if showModal {
			showModal = false
		} else {
			app.Stop()
		}
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
