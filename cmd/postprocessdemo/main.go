package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	. "github.com/kungfusheep/forme"
	"github.com/kungfusheep/riffkey"
)

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	progress := 65
	progress2 := 38
	progress3 := 82

	type fx struct {
		name     string
		active   bool
		animated bool
		effect   PostProcess
	}

	focusX, focusY, focusW, focusH := 2, 5, 50, 14

	effects := []*fx{
		// static effects
		{"Vignette", false, false, PPVignette(0.8)},
		{"Warm Tint", false, false, PPTint(Hex(0xFF6600), 0.2)},
		{"Desaturate", false, false, PPDesaturate(0.7)},
		{"Cool Tint", false, false, PPTint(Hex(0x0066FF), 0.2)},
		{"Focus Dim", false, false, PPFocusDim(&focusX, &focusY, &focusW, &focusH)},
		{"Frost", false, false, PPFrost(0.8)},
		{"Hi-Contrast", false, false, PPHighContrast(2.0)},
		{"Gradient Map", false, false, PPGradientMap(RGB(0, 0, 50), RGB(0, 128, 128), RGB(200, 255, 200))},
		{"Fade Edges", false, false, PPFadeRows(10, 13, Color{Mode: ColorRGB})},
		{"Bloom", false, false, PPBloom(2, 0.6, 0.4)},
		{"Drop Shadow", false, false, PPDropShadow(1, 1, 0.6)},
		{"CRT", false, false, PPCRT()},
		{"Monochrome", false, false, PPMonochrome(RGB(0, 255, 80))},
		// animated effects (WithQuantize(32) halves bytes/frame for smoother playback)
		{"Plasma", false, true, WithQuantize(32, PPPlasma(0.6))},
		{"Matrix", false, true, PPMatrix(2)},
		{"Fire", false, true, PPFire()},
		// blend mode variants
		{"Plsm×Normal", false, true, WithQuantize(32, WithBlend(BlendNormal, PPPlasma(0.8)))},
		{"Plsm×Mult", false, true, WithQuantize(32, WithBlend(BlendMultiply, PPPlasma(0.8)))},
		{"Plsm×Screen", false, true, WithQuantize(32, WithBlend(BlendScreen, PPPlasma(0.8)))},
		{"Plsm×Ovrlay", false, true, WithQuantize(32, WithBlend(BlendOverlay, PPPlasma(0.8)))},
		{"Plsm×Add", false, true, WithQuantize(32, WithBlend(BlendAdd, PPPlasma(0.8)))},
		{"Plsm×Soft", false, true, WithQuantize(32, WithBlend(BlendSoftLight, PPPlasma(0.8)))},
		{"Plsm×Dodge", false, true, WithQuantize(32, WithBlend(BlendColorDodge, PPPlasma(0.8)))},
		{"Plsm×Burn", false, true, WithQuantize(32, WithBlend(BlendColorBurn, PPPlasma(0.8)))},
		{"Fire×Screen", false, true, WithBlend(BlendScreen, PPFire())},
		{"Mtrx×Ovrlay", false, true, WithBlend(BlendOverlay, PPMatrix(2))},
	}

	var labels [26]string
	var activeStatus string
	keys := []string{"a", "s", "d", "f", "g", "h", "j", "k", "l", "z", "x", "c", "v", "b", "n", "m", "w", "e", "r", "t", "y", "u", "i", "o", "p", ";"}

	updateLabels := func() {
		var active []string
		for i, e := range effects {
			marker := " "
			if e.active {
				marker = "●"
				active = append(active, e.name)
			}
			labels[i] = fmt.Sprintf("%s [%s] %s", marker, keys[i], e.name)
		}
		if len(active) > 0 {
			activeStatus = " " + strings.Join(active, " + ")
		} else {
			activeStatus = " none"
		}
	}
	updateLabels()

	var tickerStop chan struct{}

	manageTicker := func() {
		needsTicker := false
		for _, e := range effects {
			if e.active && e.animated {
				needsTicker = true
				break
			}
		}
		if needsTicker && tickerStop == nil {
			tickerStop = make(chan struct{})
			go func(stop chan struct{}) {
				for {
					select {
					case <-stop:
						return
					default:
						time.Sleep(33 * time.Millisecond)
						app.RequestRender()
					}
				}
			}(tickerStop)
		} else if !needsTicker && tickerStop != nil {
			close(tickerStop)
			tickerStop = nil
		}
	}

	for i, k := range keys {
		app.Handle(k, func(_ riffkey.Match) {
			effects[i].active = !effects[i].active
			updateLabels()
			manageTicker()
		})
	}

	labelColors := []Color{
		Hex(0xAAFFAA), Hex(0xFFBB88), Hex(0xCCCCCC), Hex(0x88AAFF), Hex(0xFFFF88),
		Hex(0xAAEEFF), Hex(0xFF6666), Hex(0xCC88FF), Hex(0x88FFCC), Hex(0xFFCC44),
		Hex(0x66FFAA), Hex(0xFF44FF), Hex(0x44FF88), Hex(0xFF88FF), Hex(0x00FF66),
		Hex(0xFF4400), Hex(0xDD88FF), Hex(0x88DDFF), Hex(0xFFAA44), Hex(0x44FFAA),
		Hex(0xFFDD88), Hex(0x88FF88), Hex(0xFF8888), Hex(0x88FFFF), Hex(0xFFFF66),
		Hex(0xDD66FF),
	}

	effectLabels := make([]any, 0, len(effects))
	for i := range effects {
		effectLabels = append(effectLabels, Text(&labels[i]).Style(Style{FG: labelColors[i]}))
	}

	// declarative screen effects — reactive via If
	screenEffects := make([]any, len(effects))
	for i := range effects {
		screenEffects[i] = If(&effects[i].active).Then(ScreenEffect(effects[i].effect))
	}

	rightPanel := append([]any{
		Text("Effects").Style(Style{FG: Hex(0xCCCCCC), Attr: AttrBold}),
		SpaceH(1),
	}, effectLabels...)

	bgStyle := Style{FG: Hex(0xCCCCCC), Fill: Hex(0x1A1A2E)}

	// wide gradient row builder
	gradientRow := func(startR, startG, startB, endR, endG, endB uint8, cols int) []any {
		children := make([]any, cols)
		for i := range cols {
			t := float64(i) / float64(cols-1)
			r := uint8(float64(startR) + t*(float64(endR)-float64(startR)))
			g := uint8(float64(startG) + t*(float64(endG)-float64(startG)))
			b := uint8(float64(startB) + t*(float64(endB)-float64(startB)))
			children[i] = Text("█").Style(Style{FG: RGB(r, g, b)})
		}
		return children
	}

	// build the view tree with screen effects declared inline
	viewChildren := []any{
		VBox.CascadeStyle(&bgStyle).Grow(1)(
			// header
			HBox.Fill(Hex(0x16213E))(
				Text("  Post-Processing Demo").Style(Style{FG: Hex(0x5599FF), Attr: AttrBold}),
				Space(),
				Text("q=quit  ").Style(Style{FG: Hex(0x555555)}),
			),

			// status bar
			HBox.Fill(Hex(0x0F3460))(
				Text(" Active:").Style(Style{FG: Hex(0x888888)}),
				Text(&activeStatus).Style(Style{FG: Hex(0xFFFF44), Attr: AttrBold}),
			),

			SpaceH(1),

			// main content area
			HBox.Grow(1).Gap(2)(

				// left: colourful content filling the space
				VBox.Grow(1).Margin(1)(

					// colour swatches row
					Text(" Colour Palette").Style(Style{FG: Hex(0x999999)}),
					SpaceH(1),
					HBox.Gap(1)(
						VBox.Fill(Hex(0xFF4444)).Size(8, 3)(Text("  RED  ").FG(Hex(0xFFFFFF)).Bold()),
						VBox.Fill(Hex(0xFF8844)).Size(8, 3)(Text(" ORANGE").FG(Hex(0xFFFFFF)).Bold()),
						VBox.Fill(Hex(0xFFDD44)).Size(8, 3)(Text(" YELLOW").FG(Hex(0x000000)).Bold()),
						VBox.Fill(Hex(0x44DD44)).Size(8, 3)(Text(" GREEN ").FG(Hex(0xFFFFFF)).Bold()),
						VBox.Fill(Hex(0x4488FF)).Size(8, 3)(Text("  BLUE ").FG(Hex(0xFFFFFF)).Bold()),
						VBox.Fill(Hex(0xDD44DD)).Size(8, 3)(Text(" PURPLE").FG(Hex(0xFFFFFF)).Bold()),
					),

					SpaceH(1),

					// gradients
					Text(" Gradients").Style(Style{FG: Hex(0x999999)}),
					SpaceH(1),
					HBox(gradientRow(0, 0, 80, 255, 100, 255, 60)...),
					HBox(gradientRow(255, 0, 0, 255, 255, 0, 60)...),
					HBox(gradientRow(0, 80, 0, 0, 255, 200, 60)...),

					SpaceH(1),

					// progress bars
					Text(" System Metrics").Style(Style{FG: Hex(0x999999)}),
					SpaceH(1),
					HBox.Gap(2)(
						Text(" CPU  ").FG(Hex(0x888888)),
						Progress(&progress).Width(40).FG(Hex(0x44DDAA)),
					),
					HBox.Gap(2)(
						Text(" MEM  ").FG(Hex(0x888888)),
						Progress(&progress2).Width(40).FG(Hex(0xFF8844)),
					),
					HBox.Gap(2)(
						Text(" DISK ").FG(Hex(0x888888)),
						Progress(&progress3).Width(40).FG(Hex(0x4488FF)),
					),

					SpaceH(1),

					// info panels
					HBox.Gap(2)(
						VBox.Grow(1).Border(BorderRounded).BorderFG(Hex(0x334466))(
							Text(" Network Activity").FG(Hex(0x44AAFF)).Bold(),
							SpaceH(1),
							Text("   192.168.1.10  ████████░░  80%").FG(Hex(0x66DD88)),
							Text("   192.168.1.22  ██████░░░░  60%").FG(Hex(0xDDAA44)),
							Text("   192.168.1.35  ███░░░░░░░  30%").FG(Hex(0xDD6644)),
							Text("   192.168.1.41  █░░░░░░░░░  10%").FG(Hex(0x886666)),
						),
						VBox.Grow(1).Border(BorderRounded).BorderFG(Hex(0x334466))(
							Text(" Services").FG(Hex(0x44AAFF)).Bold(),
							SpaceH(1),
							Text("   api-gateway      running").FG(Hex(0x66DD88)),
							Text("   auth-service      running").FG(Hex(0x66DD88)),
							Text("   data-pipeline     degraded").FG(Hex(0xDDAA44)),
							Text("   cache-layer       stopped").FG(Hex(0xDD6644)),
						),
					),

					// filler
					Space(),

					// footer hint
					Text(" Try: b+v (plasma mono)  n (matrix)  m (fire!)  c+a (CRT glow)  n+d (faded matrix)").FG(Hex(0x555555)),
				),

				// right: effect toggles
				VBox.Width(32).CascadeStyle(&Style{Fill: Hex(0x16213E)}).Margin(1)(rightPanel...),
			),
		),
	}
	viewChildren = append(viewChildren, screenEffects...)

	app.SetView(VBox.Grow(1)(viewChildren...)).Handle("q", func(_ riffkey.Match) {
		app.Stop()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
