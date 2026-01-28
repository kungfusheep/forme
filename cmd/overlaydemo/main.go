package main

import (
	"fmt"
	"log"

	"github.com/kungfusheep/riffkey"
	. "github.com/kungfusheep/forme"
)

func main() {
	showModal := false
	modalMessage := "This is a modal dialog!"

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(
		VBox(
			Text("Overlay Demo").FG(Cyan).Bold(),
			HRule().Style(Style{FG: BrightBlack}),
			SpaceH(1),

			Text("This is the main application content."),
			Text("The modal will appear centered over this."),
			SpaceH(1),

			HBox.Gap(2)(
				VBox.Border(BorderSingle)(
					Text("Panel 1").FG(Yellow).Bold(),
					Text("Some content here"),
					Text("More content"),
				),

				VBox.Border(BorderSingle)(
					Text("Panel 2").FG(Green).Bold(),
					Text("Different content"),
					Text("Even more content"),
				),
			),

			SpaceH(1),
			HRule().Style(Style{FG: BrightBlack}),
			Text("Press 'm' to toggle modal | 'q' to quit").FG(BrightBlack),

			// modal overlay
			If(&showModal).Then(OverlayNode{
				Backdrop: true,
				Centered: true,
				Child: VBox.Width(50).Border(BorderRounded).Fill(PaletteColor(236))(
					Text("Modal Dialog  ").FG(Cyan).Bold(),
					SpaceH(1),
					Text(&modalMessage).FG(White),
					SpaceH(1),
					Text("Press 'm' to close").FG(BrightBlack),
				),
			}),
		),
	)

	app.Handle("m", func(_ riffkey.Match) {
		showModal = !showModal
		if showModal {
			modalMessage = fmt.Sprintf("Modal opened! Press 'm' to close.")
		}
	})
	app.Handle("q", func(_ riffkey.Match) {
		app.Stop()
	})
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
