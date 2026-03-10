package main

import (
	"log"
	"strings"

	. "github.com/kungfusheep/glyph"
)

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	focusStatus := "Focus: Name input"

	fm := NewFocusManager().OnChange(func(idx int) {
		focusStatus = "Focus: " + focusLabel(idx)
	})

	// fake log stream
	logReader := strings.NewReader(`[INFO] Application started
[DEBUG] Loading configuration...
[INFO] Database connected
[WARN] Cache miss for user_123
[ERROR] Failed to fetch resource: timeout
[INFO] Request completed in 42ms
[DEBUG] Cleaning up temp files
[INFO] User logged in: alice@example.com
[WARN] Rate limit approaching
[INFO] Processing batch job #456
`)

	app.SetView(VBox(
		Text("Focus Demo - Tab/Shift-Tab to switch").Bold(),
		Text(""),
		HBox(
			Text("Name:  "),
			Input().Placeholder("Your name").Width(30).ManagedBy(fm),
		),
		HBox(
			Text("Email: "),
			Input().Placeholder("your@email.com").Width(30).ManagedBy(fm),
		),

		Text("Logs (Ctrl-n/p to scroll):").Bold().MarginVH(1, 0),
		FilterLog(logReader).Placeholder("filter logs...").MaxLines(1000).ManagedBy(fm).Grow(1),

		Text(&focusStatus).FG(Cyan),
	))

	// Tab/Shift-Tab are auto-wired by FocusManager!
	app.Handle("q", func() { app.Stop() })
	app.Handle("<Escape>", func() { app.Stop() })

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func focusLabel(idx int) string {
	switch idx {
	case 0:
		return "Name input"
	case 1:
		return "Email input"
	case 2:
		return "Log filter"
	default:
		return "unknown"
	}
}
