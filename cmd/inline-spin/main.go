// inline spinner — like gum spin / ora
package main

import (
	"fmt"
	"log"
	"time"

	. "github.com/kungfusheep/glyph"
)

func main() {
	frame := 0
	status := "Installing dependencies..."
	done := false

	app, err := NewInlineApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(HBox(
		If(&done).
			Then(Text("✓ ").FG(Green).Bold()).
			Else(Spinner(&frame).FG(Cyan)),
		Text(&status),
	))

	go func() {
		steps := []string{
			"Resolving packages...",
			"Downloading dependencies...",
			"Linking binaries...",
			"Done!",
		}
		for i, s := range steps {
			time.Sleep(800 * time.Millisecond)
			status = s
			if i < len(steps)-1 {
				frame++
			}
			app.RequestRender()
		}
		done = true
		app.RequestRender()
		time.Sleep(400 * time.Millisecond)
		app.Stop()
	}()

	if err := app.RunNonInteractive(); err != nil {
		log.Fatal(err)
	}
	fmt.Println()
}
