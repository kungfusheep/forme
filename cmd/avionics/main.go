package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"riffkey"
	"tui"
)

// =============================================================================
// PRE-COMPOSED COMPONENT IDEAS
// These are "smart" components that just work when you throw data at them
// =============================================================================

// StatusPanel - dense label...value grid with optional status coloring
type StatusPanel struct {
	Title string
	Items []StatusItem
	Width int
}

type StatusItem struct {
	Label  string
	Value  string
	Status Status // determines color
}

type Status uint8

const (
	StatusNormal Status = iota
	StatusWarning
	StatusCritical
	StatusInactive
)

// Gauge - labeled value with bar, optional trend
type Gauge struct {
	Label   string
	Value   float64
	Min     float64
	Max     float64
	Unit    string
	History *[]float64 // optional trend data
	Width   int
}

// SubsystemGrid - compact status grid for multiple systems
type SubsystemGrid struct {
	Title    string
	Systems  []Subsystem
	Columns  int
}

type Subsystem struct {
	Name   string
	Status Status
	Brief  string // optional one-word status
}

// MessageLog - scrollable timestamped messages
type MessageLog struct {
	Messages   *[]LogMessage
	MaxVisible int
	Width      int
}

type LogMessage struct {
	Time    time.Time
	Level   Status
	Message string
}

// =============================================================================
// Helper to build StatusPanel as TUI components
// =============================================================================

func buildStatusPanel(sp StatusPanel) tui.Col {
	children := make([]any, 0, len(sp.Items)+1)

	// Title
	children = append(children, tui.Text{
		Content: sp.Title,
		Style:   tui.Style{FG: tui.Green, Attr: tui.AttrBold},
	})

	// Items as leaders
	for _, item := range sp.Items {
		style := statusColor(item.Status)
		children = append(children, tui.Leader{
			Label: item.Label,
			Value: item.Value,
			Width: int16(sp.Width),
			Fill:  '·',
			Style: style,
		})
	}

	return tui.Col{Children: children}
}

func buildGauge(g Gauge) tui.Col {
	pct := (g.Value - g.Min) / (g.Max - g.Min)
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}

	valueStr := fmt.Sprintf("%.0f%s", g.Value, g.Unit)
	barWidth := g.Width - len(g.Label) - len(valueStr) - 4

	children := []any{
		tui.Row{
			Children: []any{
				tui.Text{Content: g.Label, Style: tui.Style{FG: tui.Green}},
				tui.Text{Content: " "},
				tui.Progress{Value: int(pct * 100), BarWidth: int16(barWidth)},
				tui.Text{Content: " "},
				tui.Text{Content: valueStr, Style: tui.Style{FG: tui.BrightWhite}},
			},
		},
	}

	// Add sparkline if history provided
	if g.History != nil && len(*g.History) > 0 {
		children = append(children, tui.Sparkline{
			Values: g.History,
			Width:  int16(g.Width),
			Style:  tui.Style{FG: tui.Green},
		})
	}

	return tui.Col{Children: children}
}

func buildSubsystemGrid(sg SubsystemGrid) tui.Col {
	children := []any{
		tui.Text{Content: sg.Title, Style: tui.Style{FG: tui.Green, Attr: tui.AttrBold}},
	}

	// Build rows of systems
	var currentRow []any
	for i, sys := range sg.Systems {
		indicator := "●"
		style := statusColor(sys.Status)

		item := tui.Row{
			Children: []any{
				tui.Text{Content: indicator + " ", Style: style},
				tui.Text{Content: sys.Name, Style: tui.Style{FG: tui.Green}},
			},
		}

		currentRow = append(currentRow, item)

		if (i+1)%sg.Columns == 0 || i == len(sg.Systems)-1 {
			children = append(children, tui.Row{Gap: 2, Children: currentRow})
			currentRow = nil
		}
	}

	return tui.Col{Children: children}
}

func buildMessageLog(ml MessageLog) tui.Col {
	children := []any{
		tui.Text{Content: "MESSAGES", Style: tui.Style{FG: tui.Green, Attr: tui.AttrBold}},
	}

	msgs := *ml.Messages
	start := 0
	if len(msgs) > ml.MaxVisible {
		start = len(msgs) - ml.MaxVisible
	}

	for i := start; i < len(msgs); i++ {
		msg := msgs[i]
		timeStr := msg.Time.Format("15:04:05")
		style := statusColor(msg.Level)

		children = append(children, tui.Row{
			Children: []any{
				tui.Text{Content: timeStr + " ", Style: tui.Style{FG: tui.BrightBlack}},
				tui.Text{Content: msg.Message, Style: style},
			},
		})
	}

	return tui.Col{Children: children}
}

func statusColor(s Status) tui.Style {
	switch s {
	case StatusWarning:
		return tui.Style{FG: tui.Yellow}
	case StatusCritical:
		return tui.Style{FG: tui.Red}
	case StatusInactive:
		return tui.Style{FG: tui.BrightBlack}
	default:
		return tui.Style{FG: tui.Green}
	}
}

// =============================================================================
// DEMO
// =============================================================================

func main() {
	// Flight data
	altitude := 32450.0
	heading := 274.0
	speed := 0.82
	fuel := 68.5
	throttle := 78.0

	// Trend history
	altHistory := []float64{31200, 31800, 32100, 32300, 32400, 32450, 32450}
	fuelHistory := []float64{85, 82, 79, 76, 73, 70, 68}

	// Systems status
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

	// Message log
	messages := []LogMessage{
		{Time: time.Now().Add(-5 * time.Minute), Level: StatusNormal, Message: "NAV ALIGN COMPLETE"},
		{Time: time.Now().Add(-3 * time.Minute), Level: StatusNormal, Message: "WPT 3 PASSED"},
		{Time: time.Now().Add(-1 * time.Minute), Level: StatusWarning, Message: "HYD 2 PRESS LOW"},
		{Time: time.Now(), Level: StatusNormal, Message: "ALT HOLD ENGAGED"},
	}

	// Mode selection
	selectedMode := 0
	modes := []string{"NAV", "WPN", "DFNS"}

	// Animation frame
	frame := 0

	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Build the avionics display
	app.SetView(
		tui.Col{
			Children: []any{
				// Header bar - minimal
				tui.Row{
					Children: []any{
						tui.Text{Content: "MFD-1", Style: tui.Style{FG: tui.Green, Attr: tui.AttrBold}},
						tui.Spacer{},
						tui.Spinner{Frame: &frame, Frames: tui.SpinnerLine, Style: tui.Style{FG: tui.Green}},
						tui.Text{Content: " SYS ACTIVE", Style: tui.Style{FG: tui.Green}},
					},
				},
				tui.HRule{Char: '─', Style: tui.Style{FG: tui.BrightBlack}},

				// Mode selector - functional, not decorative
				tui.Row{
					Gap: 1,
					Children: []any{
						tui.Tabs{
							Labels:        modes,
							Selected:      &selectedMode,
							Style:         tui.TabsStyleBracket,
							ActiveStyle:   tui.Style{FG: tui.Green, Attr: tui.AttrBold},
							InactiveStyle: tui.Style{FG: tui.BrightBlack},
						},
					},
				},

				tui.Spacer{Height: 1},

				// Main content - two columns
				tui.Row{
					Gap: 4,
					Children: []any{
						// Left column - Primary flight data
						tui.Col{
							Children: []any{
								buildStatusPanel(StatusPanel{
									Title: "FLIGHT DATA",
									Width: 24,
									Items: []StatusItem{
										{Label: "ALT", Value: fmt.Sprintf("%.0f FT", altitude), Status: StatusNormal},
										{Label: "HDG", Value: fmt.Sprintf("%.0f°", heading), Status: StatusNormal},
										{Label: "MACH", Value: fmt.Sprintf("%.2f", speed), Status: StatusNormal},
										{Label: "GS", Value: "485 KT", Status: StatusNormal},
									},
								}),

								tui.Spacer{Height: 1},

								// Fuel gauge with trend
								buildGauge(Gauge{
									Label:   "FUEL",
									Value:   fuel,
									Min:     0,
									Max:     100,
									Unit:    "%",
									Width:   24,
									History: &fuelHistory,
								}),

								tui.Spacer{Height: 1},

								// Throttle gauge (no trend)
								buildGauge(Gauge{
									Label: "THRT",
									Value: throttle,
									Min:   0,
									Max:   100,
									Unit:  "%",
									Width: 24,
								}),

								tui.Spacer{Height: 1},

								// Altitude trend
								tui.Text{Content: "ALT TREND", Style: tui.Style{FG: tui.Green, Attr: tui.AttrBold}},
								tui.Sparkline{Values: &altHistory, Width: 24, Style: tui.Style{FG: tui.Green}},
							},
						}.WidthPct(0.4),

						tui.VRule{Style: tui.Style{FG: tui.BrightBlack}},

						// Right column - Systems and messages
						tui.Col{
							Children: []any{
								buildSubsystemGrid(SubsystemGrid{
									Title:   "SUBSYSTEMS",
									Columns: 4,
									Systems: systems,
								}),

								tui.Spacer{Height: 1},
								tui.HRule{Char: '─', Style: tui.Style{FG: tui.BrightBlack}},
								tui.Spacer{Height: 1},

								buildMessageLog(MessageLog{
									Messages:   &messages,
									MaxVisible: 5,
									Width:      40,
								}),
							},
						},
					},
				},

				// Footer
				tui.Spacer{Height: 1},
				tui.HRule{Char: '─', Style: tui.Style{FG: tui.BrightBlack}},
				tui.Row{
					Children: []any{
						tui.Text{Content: "N:NAV W:WPN D:DFNS TAB:CYCLE Q:EXIT", Style: tui.Style{FG: tui.BrightBlack}},
					},
				},
			},
		},
	).
		Handle("q", func(m riffkey.Match) { app.Stop() }).
		Handle("n", func(m riffkey.Match) { selectedMode = 0 }). // NAV
		Handle("w", func(m riffkey.Match) { selectedMode = 1 }). // WPN
		Handle("d", func(m riffkey.Match) { selectedMode = 2 }). // DFNS
		Handle("tab", func(m riffkey.Match) { selectedMode = (selectedMode + 1) % 3 })

	// Subtle animation - just the spinner
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			frame++

			// Slowly vary some values for realism
			altitude += (float64(frame%10) - 5) * 2
			fuel -= 0.01
			if fuel < 0 {
				fuel = 68.5
			}

			// Rotate history
			copy(altHistory, altHistory[1:])
			altHistory[len(altHistory)-1] = altitude

			// Update fuel history occasionally
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
