package main

import (
	"io"
	"log"
	"strings"
	"time"

	. "github.com/kungfusheep/glyph"
)

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	pr, pw := io.Pipe()

	app.SetView(VBox(
		Text("FilterLog Demo - fzf-style filtering").Bold(),
		Text("Type to filter | Ctrl-n/p: scroll | Ctrl-d/u: page | q: quit"),
		HRule().Style(Style{FG: Cyan}),
		FilterLog(pr).MaxLines(1000).Grow(1).Placeholder("filter..."),
	))

	app.Handle("q", func() { pw.Close(); app.Stop() })
	app.Handle("<C-c>", func() { pw.Close(); app.Stop() })

	// simulate streaming logs
	go func() {
		levels := []string{"INFO", "DEBUG", "WARN", "ERROR"}
		messages := []string{
			"Server started on :8080",
			"Connection accepted from 192.168.1.100",
			"Processing request /api/users",
			"Database query completed in 45ms",
			"Cache miss for key user:1234",
			"Retrying connection to redis",
			"Request completed successfully",
			"Memory usage: 128MB",
			"Garbage collection paused for 2ms",
			"New websocket connection established",
		}

		i := 0
		for {
			level := levels[i%len(levels)]
			msg := messages[i%len(messages)]
			timestamp := time.Now().Format("15:04:05.000")

			line := timestamp + " [" + level + "] " + strings.Repeat(" ", 5-len(level)) + msg + "\n"

			_, err := pw.Write([]byte(line))
			if err != nil {
				return
			}

			i++
			time.Sleep(200 * time.Millisecond)
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
