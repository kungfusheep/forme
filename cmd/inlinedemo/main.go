package main

import (
	"fmt"
	"log"
	"time"

	"riffkey"
	"tui"
)

func main() {
	fmt.Println("=== Inline Mode Demo ===")
	fmt.Println()

	// Test 1: Non-interactive progress (uses RunNonInteractive)
	fmt.Println("Test 1: Progress bar")
	testProgress()
	fmt.Println("Test 1 complete")
	fmt.Println()

	// Test 2: Interactive single-line (uses Run with input)
	fmt.Println("Test 2: Press 'q' to continue")
	testInteractiveSingleLine()
	fmt.Println("Test 2 complete")
	fmt.Println()

	// Test 3: Interactive multi-line menu
	fmt.Println("Test 3: Menu (j/k to move, enter to select)")
	choice := testMenu()
	fmt.Printf("You selected: %s\n", choice)
	fmt.Println("Test 3 complete")
	fmt.Println()

	fmt.Println("=== All tests done ===")
}

func testProgress() {
	progress := 0

	app, err := tui.NewInlineApp()
	if err != nil {
		log.Fatal(err)
	}

	app.ClearOnExit(true).Height(1).SetView(
		tui.HBox{
			Children: []any{
				tui.Text{Content: "["},
				tui.Progress{Value: &progress, BarWidth: 20},
				tui.Text{Content: "]"},
			},
		},
	)

	go func() {
		for progress < 100 {
			time.Sleep(15 * time.Millisecond)
			progress++
			app.RequestRender()
		}
		time.Sleep(100 * time.Millisecond)
		app.Stop()
	}()

	app.RunNonInteractive()
}

func testInteractiveSingleLine() {
	msg := "Waiting for 'q'..."

	app, err := tui.NewInlineApp()
	if err != nil {
		log.Fatal(err)
	}

	app.ClearOnExit(true).Height(1).SetView(
		tui.Text{Content: &msg},
	).Handle("q", func(m riffkey.Match) {
		msg = "Got it!"
		app.Stop()
	})

	app.Run()
}

func testMenu() string {
	selected := 0
	options := []string{"Apple", "Banana", "Cherry"}

	line1 := "> Apple"
	line2 := "  Banana"
	line3 := "  Cherry"

	updateLines := func() {
		switch selected {
		case 0:
			line1, line2, line3 = "> Apple", "  Banana", "  Cherry"
		case 1:
			line1, line2, line3 = "  Apple", "> Banana", "  Cherry"
		case 2:
			line1, line2, line3 = "  Apple", "  Banana", "> Cherry"
		}
	}

	app, err := tui.NewInlineApp()
	if err != nil {
		log.Fatal(err)
	}

	app.ClearOnExit(true).Height(3).SetView(
		tui.VBox{
			Children: []any{
				tui.Text{Content: &line1},
				tui.Text{Content: &line2},
				tui.Text{Content: &line3},
			},
		},
	).
		Handle("j", func(m riffkey.Match) {
			if selected < 2 {
				selected++
				updateLines()
			}
		}).
		Handle("k", func(m riffkey.Match) {
			if selected > 0 {
				selected--
				updateLines()
			}
		}).
		Handle("<CR>", func(m riffkey.Match) {
			app.Stop()
		}).
		Handle("q", func(m riffkey.Match) {
			selected = -1
			app.Stop()
		})

	app.Run()

	if selected >= 0 {
		return options[selected]
	}
	return "(cancelled)"
}
