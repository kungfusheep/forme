package main

import (
	"log"

	. "github.com/kungfusheep/forme"
	"github.com/kungfusheep/riffkey"
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

			Space(),
			HRule().Style(Style{FG: BrightBlack}),
			Text("Press 'm' to toggle modal | 'q' to quit").FG(BrightBlack),

			// modal overlay + screen effect — both reactive to same bool
			If(&showModal).Then(OverlayNode{
				Backdrop: false,
				Centered: true,
				Child: VBox.Width(50).Border(BorderRounded).Fill(PaletteColor(236))(
					Text("Modal Dialog  ").FG(Cyan).Bold(),
					SpaceH(1),
					Text(&modalMessage).FG(White),
					SpaceH(1),
					Text("Press 'm' to close").FG(BrightBlack),
				),
			}),
			If(&showModal).Then(ScreenEffect(PPVignette(1.0))),
		),
	)

	app.Handle("m", func(_ riffkey.Match) {
		showModal = !showModal
		if showModal {
			modalMessage = "Modal opened! Press 'm' to close."
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
