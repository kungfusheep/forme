package main

import (
	"fmt"
	"log"

	"riffkey"
	"tui"
)

func main() {
	selected := -1
	items := []string{"Apple", "Banana", "Cherry", "Date", "Elderberry", "Fig", "Grape"}
	status := "Press 'g' to enter jump mode, 'q' to quit"

	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Build the UI with Jump-wrapped items
	children := make([]any, 0, len(items)+3)

	// Title
	children = append(children, tui.Text{
		Content: "Jump Labels Demo",
		Style:   tui.Style{FG: tui.Cyan, Attr: tui.AttrBold},
	})
	children = append(children, tui.Spacer{Height: 1})

	// Items wrapped in Jump
	for i, item := range items {
		idx := i // capture for closure
		children = append(children, tui.Jump{
			Child: tui.Text{Content: fmt.Sprintf("  %s", item)},
			OnSelect: func() {
				selected = idx
				status = fmt.Sprintf("Selected: %s (index %d)", items[idx], idx)
			},
		})
	}

	// Status line
	children = append(children, tui.Spacer{Height: 1})
	children = append(children, tui.Text{Content: &status, Style: tui.Style{FG: tui.Yellow}})

	app.SetView(tui.Col{Children: children}).
		JumpKey("g"). // Register 'g' as jump trigger
		Handle("q", func(_ riffkey.Match) {
			app.Stop()
		}).
		Handle("r", func(_ riffkey.Match) {
			selected = -1
			status = "Press 'g' to enter jump mode, 'q' to quit"
		})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

	if selected >= 0 {
		fmt.Printf("Final selection: %s\n", items[selected])
	} else {
		fmt.Println("No selection made")
	}
}
