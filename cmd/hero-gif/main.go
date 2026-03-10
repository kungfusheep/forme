package main

import (
	"log"
	"time"

	. "github.com/kungfusheep/glyph"
)

func main() {
	cpu, mem := 72, 48
	online := true
	history := []float64{3, 5, 2, 7, 4, 6, 3, 5, 8, 4}
	accent := Style{FG: Cyan}
	tick := 0

	app, err := NewInlineApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(
		VBox.Border(BorderDouble).BorderFG(Cyan).Title("SYS").FitContent().CascadeStyle(&accent)(
			If(&online).
				Then(Text("● ONLINE")).
				Else(Text("● OFFLINE").FG(Red)),
			HRule(),
			Leader("CPU", &cpu),
			Leader("MEM", &mem),
			Sparkline(&history),
		),
	).Handle("q", app.Stop)

	go func() {
		for range time.Tick(300 * time.Millisecond) {
			tick++
			cpu = 50 + (tick*17)%50
			mem = 30 + (tick*13)%40
			copy(history, history[1:])
			history[len(history)-1] = float64(cpu / 10)
			online = (tick/8)%2 == 0
			app.RequestRender()
		}
	}()

	app.Run()
}
