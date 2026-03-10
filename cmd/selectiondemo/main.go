// selectiondemo: Demonstrates SelectionList with complex layouts in a modal
package main

import (
	"fmt"
	"log"

	"github.com/kungfusheep/riffkey"
	. "github.com/kungfusheep/glyph"
)

type Command struct {
	Icon     string
	Name     string
	Shortcut string
}

func main() {
	commands := []Command{
		{Icon: "[O]", Name: "Open File", Shortcut: "Ctrl+O"},
		{Icon: "[S]", Name: "Save", Shortcut: "Ctrl+S"},
		{Icon: "[A]", Name: "Save As", Shortcut: "Ctrl+Shift+S"},
		{Icon: "[F]", Name: "Find", Shortcut: "Ctrl+F"},
		{Icon: "[R]", Name: "Replace", Shortcut: "Ctrl+H"},
		{Icon: "[G]", Name: "Go to Line", Shortcut: "Ctrl+G"},
		{Icon: "[U]", Name: "Undo", Shortcut: "Ctrl+Z"},
		{Icon: "[Y]", Name: "Redo", Shortcut: "Ctrl+Y"},
		{Icon: "[X]", Name: "Cut", Shortcut: "Ctrl+X"},
		{Icon: "[C]", Name: "Copy", Shortcut: "Ctrl+C"},
		{Icon: "[V]", Name: "Paste", Shortcut: "Ctrl+V"},
		{Icon: "[*]", Name: "Select All", Shortcut: "Ctrl+A"},
	}

	selected := 0
	showModal := true
	status := "Press 'm' to toggle modal"

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	list := List(&commands).
		Selection(&selected).
		Marker("> ").
		MaxVisible(8).
		SelectedStyle(Style{BG: PaletteColor(236)}).
		Render(func(cmd *Command) any {
			return HBox.Gap(2)(
				Text(&cmd.Icon),
				Text(&cmd.Name),
				Space().Char('.').Style(Style{FG: BrightBlack}),
				Text(&cmd.Shortcut).FG(BrightBlack),
			)
		})

	app.SetView(VBox(
		Text("Selection List Demo").FG(Cyan).Bold(),
		HRule().Style(Style{FG: BrightBlack}),
		SpaceH(1),
		Text("This is the main application content."),
		Text("The command palette appears as a modal overlay."),
		SpaceH(1),
		Text(&status).FG(BrightBlack),

		// modal overlay
		If(&showModal).Then(Overlay.Centered().Backdrop()(
			VBox.Border(BorderRounded).Width(45)(
				Text(" Command Palette ").FG(Cyan).Bold(),
				HRule().Style(Style{FG: BrightBlack}),
				list,
				HRule().Style(Style{FG: BrightBlack}),
				Text("j/k:navigate  Enter:select  Esc:close").FG(BrightBlack),
			),
		)),
	))

	app.Handle("j", func(_ riffkey.Match) {
		if showModal {
			list.Down(nil)
		}
	})
	app.Handle("k", func(_ riffkey.Match) {
		if showModal {
			list.Up(nil)
		}
	})
	app.Handle("<Down>", func(_ riffkey.Match) {
		if showModal {
			list.Down(nil)
		}
	})
	app.Handle("<Up>", func(_ riffkey.Match) {
		if showModal {
			list.Up(nil)
		}
	})
	app.Handle("<Enter>", func(_ riffkey.Match) {
		if showModal && selected < len(commands) {
			cmd := commands[selected]
			status = fmt.Sprintf("Selected: %s", cmd.Name)
			showModal = false
		}
	})
	app.Handle("m", func(_ riffkey.Match) {
		showModal = !showModal
	})
	app.Handle("<Escape>", func(_ riffkey.Match) {
		if showModal {
			showModal = false
		} else {
			app.Stop()
		}
	})
	app.Handle("q", func(_ riffkey.Match) {
		if !showModal {
			app.Stop()
		}
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Final selection: %s\n", commands[selected].Name)
}
