package main

import (
	"fmt"

	. "github.com/kungfusheep/glyph"
)

func main() {
	cpu, mem := 72, 48
	online := true
	history := []float64{3, 5, 2, 7, 4, 6, 3, 5, 8, 4}

	requests := "1,247"
	latency := "42ms"
	errRate := "0.1%"
	uptime := "99.9%"
	status := "all systems operational"

	green := Style{FG: Green}
	cyan := Style{FG: Cyan}

	view := VBox.Border(BorderRounded).BorderFG(Green).Title("glyph")(
		HBox(
			VBox.Border(BorderDouble).BorderFG(Green).Title("system").Width(16).CascadeStyle(&green)(
				If(&online).
					Then(Text("● ONLINE")).
					Else(Text("● OFFLINE").FG(Red)),
				HRule(),
				Leader("CPU", &cpu),
				Leader("MEM", &mem),
				Sparkline(&history),
			),
			SpaceW(1),
			VBox.Grow(1).CascadeStyle(&cyan)(
				Text("metrics").Bold(),
				HRule(),
				Leader("requests", &requests),
				Leader("latency", &latency),
				Leader("errors", &errRate),
				Leader("uptime", &uptime),
			),
		),
		HRule(),
		Text(&status).FG(Green),
	)

	w, h := 50, 11
	tmpl := Build(view)
	buf := NewBuffer(w, h)
	tmpl.Execute(buf, int16(w), int16(h))

	// cursor home + clear screen so VHS captures from top-left
	fmt.Print("\033[H\033[2J")

	lastLine := 0
	for y := 0; y < buf.Height(); y++ {
		line := buf.GetLine(y)
		for _, r := range line {
			if r != ' ' && r != 0 {
				lastLine = y
				break
			}
		}
	}

	for y := 0; y <= lastLine; y++ {
		fmt.Println(buf.GetLineStyled(y))
	}
}
