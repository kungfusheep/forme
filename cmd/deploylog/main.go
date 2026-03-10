package main

import (
	"fmt"
	"io"
	"time"

	. "github.com/kungfusheep/glyph"
)

func main() {
	var (
		frame  int
		status = "deploying..."
		pct    int
		done   bool
		result string
	)

	pr, pw := io.Pipe()

	app, _ := NewInlineApp()
	app.ClearOnExit(true)
	app.SetView(
		VBox.FitContent()(
			HBox.Gap(2)(
				Spinner(&frame).FG(Cyan),
				Text(&status).Bold(),
				Progress(&pct).Width(20),
			),
			Log(pr).Grow(1).MaxLines(500),
			If(&done).Then(Text(&result).Bold().FG(Green)),
		),
	)

	steps := []string{
		"pulling image registry.io/app:v1.4.2",
		"verifying checksum sha256:a1b2c3d4...",
		"stopping old containers",
		"running database migrations",
		"starting new containers (3 replicas)",
		"waiting for health checks",
		"health check passed: replica 1/3",
		"health check passed: replica 2/3",
		"health check passed: replica 3/3",
		"updating load balancer",
	}

	// simulate deployment
	go func() {
		for i, step := range steps {
			fmt.Fprintf(pw, "[step %d/%d] %s\n", i+1, len(steps), step)
			pct = (i + 1) * 100 / len(steps)
			time.Sleep(800 * time.Millisecond)
		}
		done = true
		status = "complete"
		result = "deployed v1.4.2 in 8.0s"
		app.RequestRender()
	}()

	// animate spinner
	go func() {
		for range time.NewTicker(80 * time.Millisecond).C {
			frame++
			app.RequestRender()
		}
	}()

	app.Handle("q", app.Stop)
	app.Run()
}
