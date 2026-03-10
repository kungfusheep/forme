// inline multi-progress — like docker pull / cargo build
package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	. "github.com/kungfusheep/glyph"
)

type layer struct {
	name   string
	pct    int
	status string
}

func main() {
	layers := []layer{
		{"sha256:a1b2c3", 0, "pulling"},
		{"sha256:d4e5f6", 0, "pulling"},
		{"sha256:78abcd", 0, "pulling"},
		{"sha256:ef0123", 0, "pulling"},
	}

	app, err := NewInlineApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(VBox(
		ForEach(&layers, func(l *layer) any {
			return HBox.Gap(1)(
				Text(&l.name).FG(BrightBlack).Width(15),
				Progress(&l.pct).Width(30).FG(Cyan),
				Text(&l.status),
			)
		}),
	))

	go func() {
		for {
			allDone := true
			for i := range layers {
				if layers[i].pct < 100 {
					layers[i].pct += 1 + rand.Intn(4)
					if layers[i].pct > 100 {
						layers[i].pct = 100
					}
					if layers[i].pct >= 100 {
						layers[i].status = "✓ done"
					}
					allDone = false
				}
			}
			app.RequestRender()
			if allDone {
				time.Sleep(300 * time.Millisecond)
				app.Stop()
				return
			}
			time.Sleep(60 * time.Millisecond)
		}
	}()

	if err := app.RunNonInteractive(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Pull complete!")
}
