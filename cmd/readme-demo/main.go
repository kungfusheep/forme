package main

import (
	"log"
	"time"

	. "github.com/kungfusheep/forme"
)

func main() {
	alt, spd, hdg := 32450, 485, 274
	fuel, throttle := 68, 82
	history := []float64{28, 31, 29, 32, 30, 33, 31, 34, 32, 35}
	frame := 0
	hydWarn := true

	green := Style{FG: Green}

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(
		VBox.Border(BorderDouble).BorderFG(Green).Title("MFD-1").FitContent().CascadeStyle(&green)(
			HBox(
				Text("SYS").Bold(),
				Space(),
				Spinner(&frame).Frames(SpinnerLine),
				Text(" NOMINAL"),
			),
			HRule(),

			Leader("ALT", &alt),
			Leader("SPD", &spd),
			Leader("HDG", &hdg),
			SpaceH(1),

			Define(func() any {
				pgres := func(label string, val *int) any {
					return HBox(Text(label), Progress(val).Width(16), Text(" "), Text("%").Dim())
				}

				return VBox(
					pgres("FUEL  ", &fuel),
					pgres("THROT ", &throttle),
				)
			}),

			HBox(
				Text("FLAPS "),
				IfOrd(&throttle).Gt(4).Then(
					Text("DOWN").FG(Red).Bold(),
				).Else(
					Text("UP").FG(Green).Bold(),
				),
			),
			SpaceH(1),

			Text("ALT TREND").Dim(),
			Sparkline(&history),
			HRule(),

			Define(func() any {
				dot := func(label string, warn *bool) any {
					if warn == nil {
						return HBox(Text("●"), Text(" "+label))
					}
					return HBox(
						If(warn).Then(Text("●").FG(Yellow)).Else(Text("●")),
						Text(" "+label),
					)
				}

				return HBox.Gap(1)(
					dot("ENG", nil),
					dot("NAV", nil),
					dot("HYD", &hydWarn),
					dot("COM", nil),
				)
			}),
		),
	).
		Handle("q", app.Stop).
		Handle("<C-c>", app.Stop)

	go func() {
		for range time.Tick(100 * time.Millisecond) {
			frame++
			if frame%3 == 0 {
				alt += (frame % 7) - 3
				copy(history, history[1:])
				history[len(history)-1] = float64(alt / 1000)
			}
			if frame%5 == 0 {
				hydWarn = !hydWarn
			}
			app.RequestRender()
		}
	}()

	app.Run()
}
