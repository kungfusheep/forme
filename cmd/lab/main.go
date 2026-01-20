package main

import (
	"fmt"
	"log"

	"riffkey"
	"tui"
)

const contentLines = 500 // Lines per layer

// Test state
var state = struct {
	Message     string
	ActiveLayer int // 0, 1, or 2
	Status1     string
	Status2     string
	Status3     string
}{
	Message:     "Layer Stress Test - [/]: switch layer | j/k: scroll | d/u: half page | g/G: top/bottom | q: quit",
	ActiveLayer: 0,
}

func updateStatus() {
	// No longer need to track arrows here - using If conditionals in the view
	state.Status1 = fmt.Sprintf("RED: %d/%d", layer1.ScrollY(), layer1.MaxScroll())
	state.Status2 = fmt.Sprintf("GRN: %d/%d", layer2.ScrollY(), layer2.MaxScroll())
	state.Status3 = fmt.Sprintf("BLU: %d/%d", layer3.ScrollY(), layer3.MaxScroll())
}

// Three independent layers
var (
	layer1 = tui.NewLayer()
	layer2 = tui.NewLayer()
	layer3 = tui.NewLayer()
)

func init() {
	// Build and render each layer with different content
	buildLayer(layer1, "RED", tui.Red, contentLines)
	buildLayer(layer2, "GREEN", tui.Green, contentLines)
	buildLayer(layer3, "BLUE", tui.Blue, contentLines)
	updateStatus()
}

func buildLayer(layer *tui.Layer, name string, color tui.Color, lines int) {
	children := make([]any, 0, lines+4)

	// Header
	children = append(children,
		tui.RichText{Spans: []tui.Span{
			{Text: fmt.Sprintf("══════ %s LAYER ══════", name), Style: tui.Style{FG: color, Attr: tui.AttrBold}},
		}},
	)

	// Generate content lines
	for i := 1; i <= lines; i++ {
		var spans []tui.Span
		switch {
		case i%50 == 0:
			// Major section headers
			spans = []tui.Span{
				{Text: fmt.Sprintf("━━━ %s Section %d ", name, i/50), Style: tui.Style{FG: color, Attr: tui.AttrBold}},
				{Text: "━━━━━━━━━━━━━━━━━", Style: tui.Style{FG: color}},
			}
		case i%10 == 0:
			// Minor section
			spans = []tui.Span{
				{Text: fmt.Sprintf("── %s-%04d ", name, i), Style: tui.Style{FG: color}},
				{Text: "────────────", Style: tui.Style{FG: tui.BrightBlack}},
			}
		default:
			// Normal lines
			spans = []tui.Span{
				{Text: fmt.Sprintf("  %s ", name[:1]), Style: tui.Style{FG: color}},
				{Text: fmt.Sprintf("%04d: ", i), Style: tui.Style{FG: tui.BrightBlack}},
				{Text: "The quick brown fox jumps", Style: tui.Style{}},
			}
		}
		children = append(children, tui.RichText{Spans: spans})
	}

	// Footer
	children = append(children,
		tui.RichText{Spans: []tui.Span{
			{Text: fmt.Sprintf("══════ END %s ══════", name), Style: tui.Style{FG: color, Attr: tui.AttrBold}},
		}},
	)

	pageContent := tui.VBox{Children: children}
	pageTemplate := tui.Build(pageContent)

	// Render to layer with correct height
	layer.SetContent(pageTemplate, 80, len(children))
}

func activeLayer() *tui.Layer {
	switch state.ActiveLayer {
	case 0:
		return layer1
	case 1:
		return layer2
	default:
		return layer3
	}
}

func main() {
	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(labView()).
		Handle("q", func(_ riffkey.Match) {
			app.Stop()
		}).
		Handle("<Tab>", func(_ riffkey.Match) {
			state.ActiveLayer = (state.ActiveLayer + 1) % 3
			updateStatus()
		}).
		Handle("[", func(_ riffkey.Match) {
			state.ActiveLayer = (state.ActiveLayer + 2) % 3 // prev
			updateStatus()
		}).
		Handle("]", func(_ riffkey.Match) {
			state.ActiveLayer = (state.ActiveLayer + 1) % 3 // next
			updateStatus()
		}).
		Handle("j", func(_ riffkey.Match) {
			activeLayer().ScrollDown(1)
			updateStatus()
		}).
		Handle("k", func(_ riffkey.Match) {
			activeLayer().ScrollUp(1)
			updateStatus()
		}).
		Handle("d", func(_ riffkey.Match) {
			activeLayer().HalfPageDown()
			updateStatus()
		}).
		Handle("u", func(_ riffkey.Match) {
			activeLayer().HalfPageUp()
			updateStatus()
		}).
		Handle("g", func(_ riffkey.Match) {
			activeLayer().ScrollToTop()
			updateStatus()
		}).
		Handle("G", func(_ riffkey.Match) {
			activeLayer().ScrollToEnd()
			updateStatus()
		})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func labView() any {
	return tui.VBox{Children: []any{
		// Header
		tui.Text{Content: &state.Message, Style: tui.Style{Attr: tui.AttrBold}},
		tui.Text{},

		// Status bar with declarative If conditions for active indicator
		tui.HBox{Children: []any{
			// Layer 1 indicator - shows ">" when active
			tui.IfOrd(&state.ActiveLayer).Eq(0).Then(tui.Text{Content: ">"}).Else(tui.Text{Content: " "}),
			tui.Text{Content: &state.Status1},
			tui.Text{Content: " | "},
			// Layer 2 indicator
			tui.IfOrd(&state.ActiveLayer).Eq(1).Then(tui.Text{Content: ">"}).Else(tui.Text{Content: " "}),
			tui.Text{Content: &state.Status2},
			tui.Text{Content: " | "},
			// Layer 3 indicator
			tui.IfOrd(&state.ActiveLayer).Eq(2).Then(tui.Text{Content: ">"}).Else(tui.Text{Content: " "}),
			tui.Text{Content: &state.Status3},
		}},
		tui.Text{},

		// Layer 1
		tui.Text{Content: "┌─ RED LAYER ─────────────────────────────"},
		tui.LayerView{Layer: layer1, ViewHeight: 8},
		tui.Text{Content: "└─────────────────────────────────────────"},
		tui.Text{},

		// Layer 2
		tui.Text{Content: "┌─ GREEN LAYER ───────────────────────────"},
		tui.LayerView{Layer: layer2, ViewHeight: 8},
		tui.Text{Content: "└─────────────────────────────────────────"},
		tui.Text{},

		// Layer 3
		tui.Text{Content: "┌─ BLUE LAYER ────────────────────────────"},
		tui.LayerView{Layer: layer3, ViewHeight: 8},
		tui.Text{Content: "└─────────────────────────────────────────"},
	}}
}
