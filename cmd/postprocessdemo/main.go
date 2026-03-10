package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"
)

// ---------------------------------------------------------------------------
// Custom effect examples — implement Effect to write your own.
// These live here rather than in the stdlib to show the pattern in action.
// ---------------------------------------------------------------------------

var matrixGlyphs = []rune{
	'ｦ', 'ｧ', 'ｨ', 'ｩ', 'ｪ', 'ｫ', 'ｬ', 'ｭ', 'ｮ', 'ｯ',
	'ｰ', 'ｱ', 'ｲ', 'ｳ', 'ｴ', 'ｵ', 'ｶ', 'ｷ', 'ｸ', 'ｹ',
	'ｺ', 'ｻ', 'ｼ', 'ｽ', 'ｾ', 'ｿ', 'ﾀ', 'ﾁ', 'ﾂ', 'ﾃ',
	'ﾄ', 'ﾅ', 'ﾆ', 'ﾇ', 'ﾈ', 'ﾉ', 'ﾊ', 'ﾋ', 'ﾌ', 'ﾍ',
	'ﾎ', 'ﾏ', 'ﾐ', 'ﾑ', 'ﾒ', 'ﾓ', 'ﾔ', 'ﾕ', 'ﾖ', 'ﾗ',
	'ﾘ', 'ﾙ', 'ﾚ', 'ﾛ', 'ﾜ', 'ﾝ',
	'0', '1', '2', '3', '4', '5', '7', '8', '9',
	':', '.', '"', '=', '*', '+', '-', '<', '>',
	'¦', '|',
}

type matrixEffect struct{ density int }

func (m matrixEffect) Apply(buf *Buffer, ctx PostContext) {
	w, h := ctx.Width, ctx.Height
	t := ctx.Time.Seconds()
	nGlyphs := uint64(len(matrixGlyphs))

	for x := range w {
		for drop := range m.density {
			hash := uint64(x*7+drop+1) * 2654435761
			speed := 3.0 + float64(hash%10)
			offset := float64(hash % uint64(h*3))
			tailLen := 5 + int(hash%12)
			headY := int(offset+t*speed) % (h + tailLen + 5)

			for dy := range tailLen {
				cy := headY - dy
				if cy < 0 || cy >= h {
					continue
				}
				c := buf.Get(x, cy)

				charHash := uint64(cy*w+x) * 2654435761
				flickerRate := uint64(t * 8)
				if dy == 0 || charHash%3 == 0 {
					flickerRate = uint64(t * 20)
				}
				c.Rune = matrixGlyphs[(charHash+flickerRate)%nGlyphs]

				fade := 1.0 - float64(dy)/float64(tailLen)
				switch {
				case dy == 0:
					c.Style.FG = RGB(200, 255, 200)
				case dy == 1:
					c.Style.FG = RGB(100, 255, 100)
				default:
					c.Style.FG = RGB(0, uint8(20+float64(200)*fade), 0)
				}
				c.Style.BG = Color{Mode: ColorRGB}
				buf.Set(x, cy, c)
			}
		}
	}
}

type dropShadowEffect struct {
	offsetX, offsetY int
	strength         float64
}

func (d dropShadowEffect) Apply(buf *Buffer, ctx PostContext) {
	shadow := Color{Mode: ColorRGB}
	w, h := ctx.Width, ctx.Height
	occupied := make([]bool, w*h)
	for y := range h {
		for x := range w {
			r := buf.Get(x, y).Rune
			occupied[y*w+x] = r != ' ' && r != 0
		}
	}
	for y := range h {
		for x := range w {
			if occupied[y*w+x] {
				continue
			}
			srcX, srcY := x-d.offsetX, y-d.offsetY
			if srcX < 0 || srcX >= w || srcY < 0 || srcY >= h {
				continue
			}
			if !occupied[srcY*w+srcX] {
				continue
			}
			c := buf.Get(x, y)
			bg := c.Style.BG
			if bg.Mode == ColorDefault && ctx.DefaultBG.Mode != ColorDefault {
				bg = ctx.DefaultBG
			}
			c.Style.BG = LerpColor(bg, shadow, d.strength)
			buf.Set(x, y, c)
		}
	}
}

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
		effect   Effect
	}

	var focusRef NodeRef

	effects := []*fx{
		// static effects
		{"Vignette", false, false, SEVignette()},
		{"Warm Tint", false, false, SETint(Hex(0xFF6600)).Strength(0.2)},
		{"Desaturate", false, false, SEDesaturate()},
		{"Cool Tint", false, false, SETint(Hex(0x0066FF)).Strength(0.2)},
		{"Focus Dim", false, false, SEFocusDim(&focusRef)},
		{"Frost", false, false, SEFrost()},
		{"Hi-Contrast", false, false, SEContrast().Strength(2.0)},
		{"Gradient Map", false, false, SEGradientMap(RGB(0, 0, 50), RGB(0, 128, 128), RGB(200, 255, 200))},
		{"Fade Edges", false, false, SEFadeRows(10, 13, Color{Mode: ColorRGB})},
		{"Bloom", false, false, SEBloom().Threshold(0.6).Strength(0.4)},
		{"CRT", false, false, SECRT()},
		{"Monochrome", false, false, SEMonochrome(RGB(0, 255, 80))},
		// animated effects (WithQuantize(32) halves bytes/frame for smoother playback)
		{"Plasma", false, true, WithQuantize(32, SEPlasma())},
		{"Fire", false, true, SEFire()},
		// blend mode variants
		{"Plsm×Normal", false, true, WithQuantize(32, WithBlend(BlendNormal, SEPlasma().Strength(0.8)))},
		{"Plsm×Mult", false, true, WithQuantize(32, WithBlend(BlendMultiply, SEPlasma().Strength(0.8)))},
		{"Plsm×Screen", false, true, WithQuantize(32, WithBlend(BlendScreen, SEPlasma().Strength(0.8)))},
		{"Plsm×Ovrlay", false, true, WithQuantize(32, WithBlend(BlendOverlay, SEPlasma().Strength(0.8)))},
		{"Plsm×Add", false, true, WithQuantize(32, WithBlend(BlendAdd, SEPlasma().Strength(0.8)))},
		{"Plsm×Soft", false, true, WithQuantize(32, WithBlend(BlendSoftLight, SEPlasma().Strength(0.8)))},
		{"Plsm×Dodge", false, true, WithQuantize(32, WithBlend(BlendColorDodge, SEPlasma().Strength(0.8)))},
		{"Plsm×Burn", false, true, WithQuantize(32, WithBlend(BlendColorBurn, SEPlasma().Strength(0.8)))},
		{"Fire×Screen", false, true, WithBlend(BlendScreen, SEFire())},
		// custom effects — local implementations showing how to build your own
		{"Matrix", false, true, matrixEffect{density: 2}},
		{"Drop Shadow", false, false, dropShadowEffect{offsetX: 1, offsetY: 1, strength: 0.6}},
		{"Mtrx×Ovrlay", false, true, WithBlend(BlendOverlay, matrixEffect{density: 2})},
		// focus/dodge tests — left panel is the ref node
		{"Vgnt×Dodge", false, false, SEVignette().Dodge(&focusRef)},
		{"DropShadow", false, false, SEDropShadow().Focus(&focusRef)},
		{"Glow", false, false, SEGlow().Focus(&focusRef)},
		{"Tnt×Dodge", false, false, SETint(Hex(0xFF6600)).Strength(0.5).Dodge(&focusRef)},
		{"Dsat×Dodge", false, false, SEDesaturate().Dodge(&focusRef)},
		{"Mono×Dodge", false, false, SEMonochrome(RGB(0, 200, 80)).Dodge(&focusRef)},
		{"Cntr×Dodge", false, false, SEContrast().Strength(2.0).Dodge(&focusRef)},
		{"Frst×Dodge", false, false, SEFrost().Dodge(&focusRef)},
		{"Frst×Focus", false, false, SEFrost().Focus(&focusRef)},
		{"Blom×Focus", false, false, SEBloom().Strength(0.8).Focus(&focusRef)},
	}

	var labels [36]string
	var activeStatus string
	keys := []string{"a", "s", "d", "f", "g", "h", "j", "k", "l", "z", "x", "c", "v", "b", "n", "m", "w", "e", "r", "t", "y", "u", "i", "o", "p", ";", "1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}

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

	for i, k := range keys[:len(effects)] {
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
		// focus/dodge test entries
		Hex(0xFF9944), Hex(0x44DDFF), Hex(0x88FF44), Hex(0xFF44AA), Hex(0x44FFFF),
		Hex(0xFFFF44), Hex(0xAA88FF), Hex(0xFF6688), Hex(0x88AAFF), Hex(0xFFAA88),
	}

	effectLabels := make([]any, 0, len(effects))
	for i := range effects {
		effectLabels = append(effectLabels, Text(&labels[i]).Style(Style{FG: labelColors[i]}))
	}

	// declarative screen effects, reactive via If
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

				// left: colourful content filling the space — NodeRef tracked for focus/dodge effects
				VBox.Grow(1).Margin(1).NodeRef(&focusRef)(

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
					Text(" Try: m (fire!)  o+n (matrix glow)  j+k+l (plasma blends)  f (focus dim)").FG(Hex(0x555555)),
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
