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
	tick := 0

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(
		VBox.Margin(10).Border(BorderDouble).Title("SYS").FitContent()(
			If(&online).
				Then(Text("● ONLINE")).
				Else(Text("● OFFLINE").FG(Red)),
			HRule(),
			Leader("CPU", &cpu),
			Leader("MEM", &mem).Margin(1),
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
			online = (tick/8)%2 == 0 // toggle every ~2.4s
			app.RequestRender()
		}
	}()

	app.Run()
}
