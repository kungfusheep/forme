// streamdemo: items streamed into a FilterList over time
package main

import (
	"log"
	"math/rand"
	"time"

	. "github.com/kungfusheep/glyph"
)

type LogEntry struct {
	search  string // searchable text for filtering
	Display []Span // pre-styled spans for rendering
}

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	var entries []LogEntry
	var fl *FilterListC[LogEntry]

	app.View("main",
		VBox.Border(BorderRounded).Title(" stream demo ")(
			Text("Items stream in over time. Type to filter, ctrl-n/p nav, esc quit.").Dim(),
			FilterList(&entries, func(e *LogEntry) string { return e.search }).
				Placeholder("filter...").
				MaxVisible(20).
				Render(func(e *LogEntry) any {
					return RichTextNode{Spans: &e.Display}
				}).
				Ref(func(f *FilterListC[LogEntry]) { fl = f }).
				HandleClear("<Esc>", app.Stop),
		),
	)

	levels := []string{"INFO", "DEBUG", "WARN", "ERROR"}
	levelColors := map[string]Color{
		"INFO":  Green,
		"DEBUG": Cyan,
		"WARN":  Yellow,
		"ERROR": Red,
	}
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
		"TLS handshake completed",
		"Rate limiter triggered for 10.0.0.5",
		"Scheduled job cleanup_sessions started",
		"Webhook delivered to https://example.com/hook",
		"Config reloaded from /etc/app/config.yaml",
	}

	w := fl.Stream(app.RequestRender)
	go func() {
		defer w.Close()
		for i := 0; i < 20; i++ {
			time.Sleep(time.Duration(100+rand.Intn(400)) * time.Millisecond)

			ts := time.Now().Format("15:04:05.000")
			level := levels[rand.Intn(len(levels))]
			msg := messages[rand.Intn(len(messages))]

			w.Write(LogEntry{
				search: ts + " " + level + " " + msg,
				Display: []Span{
					{Text: ts, Style: Style{Attr: AttrDim}},
					{Text: " ["},
					{Text: level, Style: Style{FG: levelColors[level], Attr: AttrBold}},
					{Text: "] "},
					{Text: msg},
				},
			})
		}
	}()

	if err := app.RunFrom("main"); err != nil {
		log.Fatal(err)
	}
}
