// inline confirm prompt — like gum confirm / rm -i
package main

import (
	"fmt"
	"log"

	. "github.com/kungfusheep/glyph"
)

func main() {
	yes := false

	app, err := NewInlineApp()
	if err != nil {
		log.Fatal(err)
	}

	app.ClearOnExit(true).
		SetView(HBox(
			Text("Delete 3 files? ").Bold(),
			Text("(y/n) ").FG(BrightBlack),
		)).
		Handle("y", func() { yes = true; app.Stop() }).
		Handle("n", app.Stop).
		Handle("<Escape>", app.Stop).
		Run()

	if yes {
		fmt.Println("✓ Deleted 3 files")
	} else {
		fmt.Println("✗ Cancelled")
	}
}
