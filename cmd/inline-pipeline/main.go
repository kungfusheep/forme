// inline deploy pipeline — like railway deploy / vercel
package main

import (
	"fmt"
	"log"
	"time"

	. "github.com/kungfusheep/glyph"
)

type step struct {
	icon   string
	name   string
	status string
}

func (s *step) run() {
	s.icon = "›"
	s.status = "running"
}

func (s *step) done() {
	s.icon = "✓"
	s.status = "done"
}

func main() {
	steps := []step{
		{"○", "Build", "pending"},
		{"○", "Test", "pending"},
		{"○", "Push image", "pending"},
		{"○", "Deploy", "pending"},
		{"○", "Health check", "pending"},
	}

	app, err := NewInlineApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(VBox(
		ForEach(&steps, func(s *step) any {
			return HBox.Gap(1)(
				Text(&s.icon).Width(1).Bold(),
				Text(&s.name).Width(15),
				Text(&s.status).FG(BrightBlack),
			)
		}),
	))

	go func() {
		for i := range steps {
			steps[i].run()
			app.RequestRender()
			time.Sleep(400 + time.Duration(i*200)*time.Millisecond)
			steps[i].done()
			app.RequestRender()
		}
		time.Sleep(300 * time.Millisecond)
		app.Stop()
	}()

	if err := app.RunNonInteractive(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Deployed to production!")
}
