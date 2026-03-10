package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/kungfusheep/riffkey"
	. "github.com/kungfusheep/glyph"
)

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	contentHeight := 100_000
	layer := NewLayer()
	buf := NewBuffer(80, contentHeight)

	colors := []Color{Red, Green, Yellow, Blue, Magenta, Cyan}
	for y := 0; y < contentHeight; y++ {
		color := colors[y%len(colors)]
		style := Style{FG: color}

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

	scrollInfo := fmt.Sprintf("Line 0/%d", contentHeight)

	app.SetView(VBox(
		Text("╔══════════════════════════════════════════════════════════════════════════════╗"),
		Text("║                    Layer Scrolling Demo - V2Template                         ║"),
		Text("╚══════════════════════════════════════════════════════════════════════════════╝"),
		Text(""),

		VBox.Border(BorderDouble).BorderFG(Cyan).Grow(1).Title("Scrollable Content")(
			LayerView(layer).Grow(1),
		),

		Text(""),
		HBox.Gap(2)(
			Text(&scrollInfo),
			Text("│ j/k:line  d/u:half  f/b:page  g/G:top/end  q:quit"),
		),
	))

	updateInfo := func() {
		scrollInfo = fmt.Sprintf("Line %d/%d (%.0f%%)",
			layer.ScrollY(),
			layer.MaxScroll(),
			float64(layer.ScrollY())/float64(max(1, layer.MaxScroll()))*100)
	}

	app.Handle("j", func(_ riffkey.Match) { layer.ScrollDown(1); updateInfo() })
	app.Handle("k", func(_ riffkey.Match) { layer.ScrollUp(1); updateInfo() })
	app.Handle("<Down>", func(_ riffkey.Match) { layer.ScrollDown(1); updateInfo() })
	app.Handle("<Up>", func(_ riffkey.Match) { layer.ScrollUp(1); updateInfo() })
	app.Handle("d", func(_ riffkey.Match) { layer.HalfPageDown(); updateInfo() })
	app.Handle("u", func(_ riffkey.Match) { layer.HalfPageUp(); updateInfo() })
	app.Handle("<C-d>", func(_ riffkey.Match) { layer.HalfPageDown(); updateInfo() })
	app.Handle("<C-u>", func(_ riffkey.Match) { layer.HalfPageUp(); updateInfo() })
	app.Handle("f", func(_ riffkey.Match) { layer.PageDown(); updateInfo() })
	app.Handle("b", func(_ riffkey.Match) { layer.PageUp(); updateInfo() })
	app.Handle("<Space>", func(_ riffkey.Match) { layer.PageDown(); updateInfo() })
	app.Handle("g", func(_ riffkey.Match) { layer.ScrollToTop(); updateInfo() })
	app.Handle("G", func(_ riffkey.Match) { layer.ScrollToEnd(); updateInfo() })
	app.Handle("q", func(_ riffkey.Match) { app.Stop() })
	app.Handle("<C-c>", func(_ riffkey.Match) { app.Stop() })

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
