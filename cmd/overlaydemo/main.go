package main

import (
	"log"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"
)

// Rose Piné palette
var (
	rpBase    = Hex(0x191724)
	rpSurface = Hex(0x1f1d2e)
	rpOverlay = Hex(0x26233a)
	rpText    = Hex(0xe0def4)
	rpSubtle  = Hex(0x908caa)
	rpMuted   = Hex(0x6e6a86)
	rpLove    = Hex(0xeb6f92)
	rpGold    = Hex(0xf6c177)
	rpPine    = Hex(0x31748f)
	rpFoam    = Hex(0x9ccfd8)
	rpIris    = Hex(0xc4a7e7)
	rpRose    = Hex(0xebbcba)
)

func main() {
	showModal := false
	modalMessage := "  Modal opened! Press 'm' to close."

	var popupRef NodeRef

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(
		VBox.Fill(rpBase).CascadeStyle(&Style{FG: rpText}).Grow(1)(

			// header
			HBox.Fill(rpSurface).Gap(1)(
				Text("  Overlay Demo").Style(Style{FG: rpIris, Attr: AttrBold}),
			),

			SpaceH(1),

			// main content
			HBox.Grow(1).Gap(2).Margin(2)(

				VBox.Grow(1)(
					Text("Main Content").Style(Style{FG: rpRose, Attr: AttrBold}),
					SpaceH(1),
					Text("This is the background content behind the modal.").Style(Style{FG: rpSubtle}),
					Text("Press 'm' to open the overlay and see the vignette.").Style(Style{FG: rpSubtle}),
					SpaceH(1),
					HBox.Gap(2)(
						VBox.Grow(1).Border(BorderRounded).BorderFG(rpOverlay)(
							Text(" Services").Style(Style{FG: rpPine, Attr: AttrBold}),
							SpaceH(1),
							Text("  api-gateway    running").Style(Style{FG: rpFoam}),
							Text("  auth-service   running").Style(Style{FG: rpFoam}),
							Text("  data-pipeline  degraded").Style(Style{FG: rpGold}),
							Text("  cache-layer    stopped").Style(Style{FG: rpLove}),
						),
						VBox.Grow(1).Border(BorderRounded).BorderFG(rpOverlay)(
							Text(" Metrics").Style(Style{FG: rpPine, Attr: AttrBold}),
							SpaceH(1),
							Text("  CPU   ████████░░  78%").Style(Style{FG: rpFoam}),
							Text("  MEM   █████░░░░░  52%").Style(Style{FG: rpGold}),
							Text("  DISK  ███░░░░░░░  31%").Style(Style{FG: rpIris}),
						),
					),
				),
			),

			Space(),
			HRule().Style(Style{FG: rpOverlay}),
			Text("  Press 'm' to toggle modal | 'q' to quit").Style(Style{FG: rpMuted}),

			// overlay owns its vignette — no separate If() needed
			If(&showModal).Then(OverlayNode{
				Centered: true,
				Child: VBox.Gap(1).Width(52).Fill(rpSurface).NodeRef(&popupRef)(
					Space(),
					Text("  Confirm Action").Style(Style{FG: rpIris, Attr: AttrBold}),
					Text(&modalMessage).Style(Style{FG: rpText}),
					HRule().Style(Style{FG: rpOverlay}),
					Text("  Press 'm' to close  |  Esc to dismiss").Style(Style{FG: rpMuted}),

					// stdlib effect sampler — uncomment one at a time to preview
					ScreenEffect(
						// SEDesaturate().Dodge(&popupRef),
						SEVignette().Strength(0.17).Dodge(&popupRef).Smooth(), // darken edges
						// SEMonochrome(rpLove).Dodge(&popupRef),        // monochrome with "hole"bbbbbbbbbbb
						// SEMonochrome(rpFoam).Dodge(&popupRef), // tinted monochrome
						// SEMonochrome(Green).Dodge(&popupRef),
						SEDropShadow().Focus(&popupRef),
						// SEGlow().Focus(&popupRef),
					),
					//ScreenEffect(SEDesaturate().Strength(0.8)),                 // wash out colour
					//ScreenEffect(SETint(rpLove).Strength(0.15)),                // error / alert tint
					//ScreenEffect(SETint(rpGold).Strength(0.15)),                // warning tint
					//ScreenEffect(SETint(rpFoam).Strength(0.12)),                // success tint
					//ScreenEffect(SEBloom().Radius(2).Threshold(0.6).Strength(0.5)), // glow
					// ScreenEffect(SEHighContrast().Strength(1.8)),               // contrast boost
					// ScreenEffect(SEFocusDim(&popupRef)),                        // spotlight panel
				),
			}),
		),
	)

	app.Handle("m", func(_ riffkey.Match) {
		showModal = !showModal
	})
	app.Handle("q", func(_ riffkey.Match) {
		app.Stop()
	})
	app.Handle("<Escape>", func(_ riffkey.Match) {
		if showModal {
			showModal = false
		} else {
			app.Stop()
		}
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
