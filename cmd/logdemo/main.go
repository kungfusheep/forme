package main

import (
	"fmt"
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

	// create a pipe - we'll write to pw, Log reads from pr
	pr, pw := io.Pipe()

	var lv *LogC
	status := ""

	app.SetView(VBox(
		Text("Log Demo - streaming logs").Bold(),
		Text("j/k: scroll | Ctrl-d/u: page | g/G: top/bottom | q: quit"),
		HRule().Style(Style{FG: Cyan}),
		Log(pr).MaxLines(1000).Grow(1).BindVimNav().Ref(func(l *LogC) { lv = l }),
		If(&status).Then(Text(&status).FG(Yellow)),
	))

	// update status indicator
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			if lv == nil {
				continue
			}
			n := lv.NewLines()
			if n > 0 {
				status = fmt.Sprintf("  ↓ %d new lines", n)
			} else {
				status = ""
			}
		}
	}()

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
				return // pipe closed
			}

			i++
			time.Sleep(200 * time.Millisecond)
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
