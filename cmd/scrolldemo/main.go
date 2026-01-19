package main

import (
	"fmt"
	"log"
	"strings"

	"riffkey"
	"tui"
)

func main() {
	// Create the app
	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Create a layer with lots of content (simulating a long document)
	contentHeight := 100_000
	layer := tui.NewLayer()
	buf := tui.NewBuffer(80, contentHeight)

	// Fill with interesting content
	colors := []tui.Color{tui.Red, tui.Green, tui.Yellow, tui.Blue, tui.Magenta, tui.Cyan}
	for y := 0; y < contentHeight; y++ {
		color := colors[y%len(colors)]
		style := tui.Style{FG: color}

		// Create varied content
		var line string
		switch y % 10 {
		case 0:
			line = fmt.Sprintf("═══════════════════ Section %d ═══════════════════", y/10+1)
		case 1:
			line = fmt.Sprintf("  Line %03d: %s", y, strings.Repeat("▓", 40))
		case 2:
			line = fmt.Sprintf("  Line %03d: %s", y, strings.Repeat("▒", 40))
		case 3:
			line = fmt.Sprintf("  Line %03d: %s", y, strings.Repeat("░", 40))
		case 4:
			line = fmt.Sprintf("  Line %03d: Lorem ipsum dolor sit amet", y)
		case 5:
			line = fmt.Sprintf("  Line %03d: The quick brown fox jumps over", y)
		case 6:
			line = fmt.Sprintf("  Line %03d: %s", y, strings.Repeat("█", y%30+10))
		case 7:
			line = fmt.Sprintf("  Line %03d: ════════════════════════════", y)
		case 8:
			line = fmt.Sprintf("  Line %03d: ◆◆◆ Important content here ◆◆◆", y)
		case 9:
			line = fmt.Sprintf("  Line %03d: ────────────────────────────", y)
		}
		buf.WriteStringFast(0, y, line, style, 80)
	}
	layer.SetBuffer(buf)

	// State for display
	scrollInfo := fmt.Sprintf("Line 0/%d", contentHeight)

	// Build the view
	view := tui.Col{Children: []any{
		tui.Text{Content: "╔══════════════════════════════════════════════════════════════════════════════╗"},
		tui.Text{Content: "║                    Layer Scrolling Demo - V2Template                         ║"},
		tui.Text{Content: "╚══════════════════════════════════════════════════════════════════════════════╝"},
		tui.Text{Content: ""},

		// Main content area with layer - layer grows to fill container
		tui.Col{
			Title: "Scrollable Content",
			Children: []any{
				tui.LayerView{Layer: layer}.Grow(1),
			},
		}.Border(tui.BorderDouble).BorderFG(tui.Cyan).Grow(1),

		tui.Text{Content: ""},
		tui.Row{Gap: 2, Children: []any{
			tui.Text{Content: &scrollInfo},
			tui.Text{Content: "│ j/k:line  d/u:half  f/b:page  g/G:top/end  q:quit"},
		}},
	}}

	app.SetView(view)

	// Update scroll info helper
	updateInfo := func() {
		scrollInfo = fmt.Sprintf("Line %d/%d (%.0f%%)",
			layer.ScrollY(),
			layer.MaxScroll(),
			float64(layer.ScrollY())/float64(max(1, layer.MaxScroll()))*100)
	}

	// Key bindings
	app.Handle("j", func(_ riffkey.Match) {
		layer.ScrollDown(1)
		updateInfo()
	})
	app.Handle("k", func(_ riffkey.Match) {
		layer.ScrollUp(1)
		updateInfo()
	})
	app.Handle("<Down>", func(_ riffkey.Match) {
		layer.ScrollDown(1)
		updateInfo()
	})
	app.Handle("<Up>", func(_ riffkey.Match) {
		layer.ScrollUp(1)
		updateInfo()
	})
	app.Handle("d", func(_ riffkey.Match) {
		layer.HalfPageDown()
		updateInfo()
	})
	app.Handle("u", func(_ riffkey.Match) {
		layer.HalfPageUp()
		updateInfo()
	})
	app.Handle("<C-d>", func(_ riffkey.Match) {
		layer.HalfPageDown()
		updateInfo()
	})
	app.Handle("<C-u>", func(_ riffkey.Match) {
		layer.HalfPageUp()
		updateInfo()
	})
	app.Handle("f", func(_ riffkey.Match) {
		layer.PageDown()
		updateInfo()
	})
	app.Handle("b", func(_ riffkey.Match) {
		layer.PageUp()
		updateInfo()
	})
	app.Handle("<Space>", func(_ riffkey.Match) {
		layer.PageDown()
		updateInfo()
	})
	app.Handle("g", func(_ riffkey.Match) {
		layer.ScrollToTop()
		updateInfo()
	})
	app.Handle("G", func(_ riffkey.Match) {
		layer.ScrollToEnd()
		updateInfo()
	})
	app.Handle("q", func(_ riffkey.Match) {
		app.Stop()
	})
	app.Handle("<C-c>", func(_ riffkey.Match) {
		app.Stop()
	})

	// Run!
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
