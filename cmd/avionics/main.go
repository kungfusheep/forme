package main

import (
	"fmt"
	"log"
	"math"
	"time"

	. "github.com/kungfusheep/forme"
	"github.com/kungfusheep/riffkey"
)

// =============================================================================
// PRE-COMPOSED COMPONENT PATTERNS
// These helper functions return functional components
// =============================================================================

type StatusItem struct {
	Label  string
	Value  string
	Status Status
}

type Status uint8

const (
	StatusNormal Status = iota
	StatusWarning
	StatusCritical
	StatusInactive
)

func statusColor(s Status) Style {
	switch s {
	case StatusWarning:
		return Style{FG: Yellow}
	case StatusCritical:
		return Style{FG: Red}
	case StatusInactive:
		return Style{FG: BrightBlack}
	default:
		return Style{FG: Green}
	}
}

// StatusPanel builds a titled list of label...value items
func StatusPanel(title string, width int, items []StatusItem) VBoxC {
	children := make([]any, 0, len(items)+1)
	children = append(children, Text(title).FG(Green).Bold())

	for _, item := range items {
		children = append(children, Leader(item.Label, item.Value).Width(int16(width)).Fill('·').Style(statusColor(item.Status)))
	}

	return VBox(children...)
}

// Gauge builds a labeled progress bar with optional sparkline trend
func Gauge(label string, value, min, max float64, unit string, width int, history *[]float64) VBoxC {
	pct := (value - min) / (max - min)
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}

	valueStr := fmt.Sprintf("%.0f%s", value, unit)
	barWidth := width - len(label) - len(valueStr) - 4

	children := []any{
		HBox(
			Text(label).FG(Green),
			Text(" "),
			Progress(int(pct*100)).Width(int16(barWidth)),
			Text(" "),
			Text(valueStr).FG(BrightWhite),
		),
	}

	if history != nil && len(*history) > 0 {
		children = append(children, Sparkline(history).Width(int16(width)).Style(Style{FG: Green}))
	}

	return VBox(children...)
}

type Subsystem struct {
	Name   string
	Status Status
}

// SubsystemGrid builds a compact multi-column status grid
func SubsystemGrid(title string, columns int, systems []Subsystem) VBoxC {
	children := []any{Text(title).FG(Green).Bold()}

	var currentRow []any
	for i, sys := range systems {
		item := HBox(
			Text("● ").Style(statusColor(sys.Status)),
			Text(sys.Name).FG(Green),
		)
		currentRow = append(currentRow, item)

		if (i+1)%columns == 0 || i == len(systems)-1 {
			children = append(children, HBox.Gap(2)(currentRow...))
			currentRow = nil
		}
	}

	return VBox(children...)
}

type LogMessage struct {
	Time    time.Time
	Level   Status
	Message string
}

// MessageLog builds a scrollable timestamped message list
func MessageLog(title string, messages *[]LogMessage, maxVisible int) VBoxC {
	children := []any{Text(title).FG(Green).Bold()}

	msgs := *messages
	start := 0
	if len(msgs) > maxVisible {
		start = len(msgs) - maxVisible
	}

	for i := start; i < len(msgs); i++ {
		msg := msgs[i]
		children = append(children, HBox(
			Text(msg.Time.Format("15:04:05")+" ").FG(BrightBlack),
			Text(msg.Message).Style(statusColor(msg.Level)),
		))
	}

	return VBox(children...)
}

// =============================================================================
// DEMO
// =============================================================================

func main() {
	altitude := 32450.0
	heading := 274.0
	speed := 0.82
	fuel := 68.5
	throttle := 78.0

	altHistory := []float64{31200, 31800, 32100, 32300, 32400, 32450, 32450}
	fuelHistory := []float64{85, 82, 79, 76, 73, 70, 68}

	systems := []Subsystem{
		{Name: "ENG L", Status: StatusNormal},
		{Name: "ENG R", Status: StatusNormal},
		{Name: "HYD 1", Status: StatusNormal},
		{Name: "HYD 2", Status: StatusWarning},
		{Name: "ELEC", Status: StatusNormal},
		{Name: "FUEL", Status: StatusNormal},
		{Name: "NAV", Status: StatusNormal},
		{Name: "COMM", Status: StatusNormal},
	}

	messages := []LogMessage{
		{Time: time.Now().Add(-5 * time.Minute), Level: StatusNormal, Message: "NAV ALIGN COMPLETE"},
		{Time: time.Now().Add(-3 * time.Minute), Level: StatusNormal, Message: "WPT 3 PASSED"},
		{Time: time.Now().Add(-1 * time.Minute), Level: StatusWarning, Message: "HYD 2 PRESS LOW"},
		{Time: time.Now(), Level: StatusNormal, Message: "ALT HOLD ENGAGED"},
	}

	selectedMode := 0
	modes := []string{"NAV", "WPN", "DFNS"}
	frame := 0

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(
		VBox(
			// header
			HBox(
				Text("MFD-1").FG(Green).Bold(),
				Space(),
				Spinner(&frame).Frames(SpinnerLine).Style(Style{FG: Green}),
				Text(" SYS ACTIVE").FG(Green),
			),
			HRule().Char('─').Style(Style{FG: BrightBlack}),

			// mode selector
			HBox.Gap(1)(
				TabsNode{
					Labels:        modes,
					Selected:      &selectedMode,
					Style:         TabsStyleBracket,
					ActiveStyle:   Style{FG: Green, Attr: AttrBold},
					InactiveStyle: Style{FG: BrightBlack},
				},
			),

			SpaceH(1),

			// main content - two columns
			HBox.Gap(4)(
				// left column - flight data
				VBox.WidthPct(0.4)(
					StatusPanel("FLIGHT DATA", 24, []StatusItem{
						{Label: "ALT", Value: fmt.Sprintf("%.0f FT", altitude), Status: StatusNormal},
						{Label: "HDG", Value: fmt.Sprintf("%.0f°", heading), Status: StatusNormal},
						{Label: "MACH", Value: fmt.Sprintf("%.2f", speed), Status: StatusNormal},
						{Label: "GS", Value: "485 KT", Status: StatusNormal},
					}),

					SpaceH(1),
					Gauge("FUEL", fuel, 0, 100, "%", 24, &fuelHistory),

					SpaceH(1),
					Gauge("THRT", throttle, 0, 100, "%", 24, nil),

					SpaceH(1),
					Text("ALT TREND").FG(Green).Bold(),
					Sparkline(&altHistory).Width(24).Style(Style{FG: Green}),
				),

				VRule().Style(Style{FG: BrightBlack}),

				// right column - systems and messages
				VBox(
					SubsystemGrid("SUBSYSTEMS", 4, systems),

					SpaceH(1),
					HRule().Char('─').Style(Style{FG: BrightBlack}),
					SpaceH(1),

					MessageLog("MESSAGES", &messages, 5),
				),
			),

			// footer
			SpaceH(1),
			HRule().Char('─').Style(Style{FG: BrightBlack}),
			HBox(
				Text("N:NAV W:WPN D:DFNS TAB:CYCLE Q:EXIT").FG(BrightBlack),
			),
		),
	).
		Handle("q", func(m riffkey.Match) { app.Stop() }).
		Handle("n", func(m riffkey.Match) { selectedMode = 0 }).
		Handle("w", func(m riffkey.Match) { selectedMode = 1 }).
		Handle("d", func(m riffkey.Match) { selectedMode = 2 }).
		Handle("tab", func(m riffkey.Match) { selectedMode = (selectedMode + 1) % 3 })

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			frame++
			altitude += (float64(frame%10) - 5) * 2
			fuel -= 0.01
			if fuel < 0 {
				fuel = 68.5
			}

			copy(altHistory, altHistory[1:])
			altHistory[len(altHistory)-1] = altitude

			if frame%10 == 0 {
				copy(fuelHistory, fuelHistory[1:])
				fuelHistory[len(fuelHistory)-1] = math.Max(0, fuel)
			}

			app.RenderNow()
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
