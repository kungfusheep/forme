package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"

	"github.com/kungfusheep/riffkey"
	. "github.com/kungfusheep/glyph"
)

// Grid returns a layout function that arranges children in a grid
// If cellH is 0, it uses each child's natural height
func Grid(cols, cellW, cellH int) LayoutFunc {
	return func(children []ChildSize, availW, availH int) []Rect {
		rects := make([]Rect, len(children))

		// Calculate row heights (max height of items in each row)
		numRows := (len(children) + cols - 1) / cols
		rowHeights := make([]int, numRows)
		for i := range children {
			row := i / cols
			h := cellH
			if h == 0 {
				h = children[i].MinH // Use natural height
			}
			if h > rowHeights[row] {
				rowHeights[row] = h
			}
		}

		// Calculate row Y offsets
		rowY := make([]int, numRows)
		y := 0
		for r := range rowY {
			rowY[r] = y
			y += rowHeights[r]
		}

		// Place items
		for i := range children {
			col := i % cols
			row := i / cols
			h := cellH
			if h == 0 {
				h = children[i].MinH
			}
			rects[i] = Rect{
				X: col * cellW,
				Y: rowY[row],
				W: cellW,
				H: h,
			}
		}
		return rects
	}
}

// MiniGraph is a custom renderer that draws a multi-row tall graph
type MiniGraph struct {
	Values *[]float64
	Width  int
	Height int
	Style  Style
}

func (g MiniGraph) MinSize() (width, height int) {
	h := g.Height
	if h < 1 {
		h = 8
	}
	return g.Width, h
}

func (g MiniGraph) Render(buf *Buffer, x, y, w, h int) {
	if g.Values == nil || len(*g.Values) == 0 {
		return
	}
	vals := *g.Values
	rows := h
	if rows < 1 {
		rows = 8
	}

	blocks := []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	for i := 0; i < w && i < len(vals); i++ {
		normalized := vals[i] / 100.0
		if normalized > 1 {
			normalized = 1
		}
		if normalized < 0 {
			normalized = 0
		}

		totalLevels := rows * 8
		level := int(normalized * float64(totalLevels))

		for row := 0; row < rows; row++ {
			rowY := y + (rows - 1 - row)
			rowLevel := level - (row * 8)

			var char rune
			if rowLevel >= 8 {
				char = '█'
			} else if rowLevel > 0 {
				char = blocks[rowLevel]
			} else {
				char = ' '
			}

			buf.Set(x+i, rowY, Cell{Rune: char, Style: g.Style})
		}
	}
}

// State holds all dashboard state
type State struct {
	// Display toggles
	ShowGraph    bool
	ShowProcs    bool
	ShowHelp     bool
	Paused       bool
	SelectedProc int

	// View mode for Switch/Case demo: "all", "compact", "graphs"
	ViewMode string

	// Stats
	Hostname  string
	Uptime    string
	Load      string
	CPUTotal  int
	MemTotal  int
	SwapTotal int

	// Quick stats
	Tasks    string
	Threads  string
	Running  string
	Sleeping string
	Stopped  string
	Zombie   string

	// Graph data
	CPUHistory    []float64
	RenderHistory []float64 // Render time in µs (scaled: 100 = 1000µs)
	FlushHistory  []float64 // Flush time in µs (scaled: 100 = 1000µs)

	// Process list
	Processes []Process

	// Help text
	HelpText string

	// Render stats
	Timing      string
	RenderLabel string
	FlushLabel  string
	RowStats    string // "dirty/changed" row counts
	FPSLabel    string // actual FPS

	// Animation state (not displayed directly)
	cpuTarget   float64
	memTarget   float64
	swapTarget  float64
	load1       float64
	load5       float64
	load15      float64
	procData    []processData
	startTime   time.Time
	frameCount  int64
	lastFPSTime time.Time
	fpsFrames   int
	currentFPS  float64
}

type Process struct {
	PID      string
	Name     string
	CPU      string
	Mem      string
	Status   string
	Selected bool
}

type processData struct {
	cpu float64
	mem float64
}

func main() {
	// CPU profiling - writes to cpu.prof on exit
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer func() {
		pprof.StopCPUProfile()
		f.Close()
		fmt.Println("CPU profile written to cpu.prof")
		fmt.Println("Run: go tool pprof -http=:8080 cpu.prof")
	}()

	// Enable debug timing
	DebugTiming = true

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize state
	state := &State{
		ShowGraph:    true,
		ShowProcs:    true,
		ShowHelp:     false,
		Paused:       false,
		SelectedProc: 0,
		ViewMode:     "all", // Switch/Case demo: "all", "compact", "graphs"

		Hostname:  "v2-dashboard",
		Uptime:    "0:00:00",
		Load:      "0.00, 0.00, 0.00",
		CPUTotal:  25,
		MemTotal:  35,
		SwapTotal: 8,

		Tasks:    "Tasks:    142",
		Threads:  "Threads:  891",
		Running:  "Running:    3",
		Sleeping: "Sleeping: 139",
		Stopped:  "Stopped:    0",
		Zombie:   "Zombie:     0",

		CPUHistory:    make([]float64, 60),
		RenderHistory: make([]float64, 60),
		FlushHistory:  make([]float64, 60),

		HelpText:    "[q]uit [g]raph [p]rocs [h]elp [space]pause [j/k]select",
		RenderLabel: "Render:    0µs",
		FlushLabel:  "Flush:     0µs",

		cpuTarget:   25 * 4,
		memTarget:   35,
		swapTarget:  8,
		load1:       1.5,
		load5:       1.2,
		load15:      0.8,
		startTime:   time.Now(),
		lastFPSTime: time.Now(),
		FPSLabel:    "FPS: --",
	}

	// Initialize processes
	names := []string{"systemd", "kworker", "nginx", "postgres", "redis", "node", "go", "python", "java", "docker"}
	statuses := []string{"Running", "Sleeping", "Waiting", "Zombie"}

	state.Processes = make([]Process, 15)
	state.procData = make([]processData, 15)

	for i := range state.Processes {
		state.procData[i].cpu = rand.Float64() * 5
		state.procData[i].mem = rand.Float64() * 3
		state.Processes[i] = Process{
			PID:      fmt.Sprintf("%5d", 1000+i),
			Name:     fmt.Sprintf("%-12s", names[i%len(names)]),
			CPU:      fmt.Sprintf("%5.1f%%", state.procData[i].cpu),
			Mem:      fmt.Sprintf("%5.1f%%", state.procData[i].mem),
			Status:   statuses[i%len(statuses)],
			Selected: i == 0, // First item selected by default
		}
	}

	// Initialize history
	for i := range state.CPUHistory {
		state.CPUHistory[i] = float64(state.CPUTotal)
	}

	cpuStyle := Style{FG: Color{Mode: ColorRGB, R: 80, G: 200, B: 120}}
	renderStyle := Style{FG: Color{Mode: ColorRGB, R: 255, G: 180, B: 80}} // Orange
	flushStyle := Style{FG: Color{Mode: ColorRGB, R: 80, G: 180, B: 255}}  // Blue
	warnStyle := Style{FG: Yellow}
	critStyle := Style{FG: Red, Attr: AttrBold}

	// Build UI with conditionals (using V2 template for Box, custom Renderer support)
	// Layout uses Grow() to push footer to bottom of screen
	app.SetView(VBox(
		// header
		HBox.Gap(2)(
			Text(&state.Hostname),
			Text("Uptime:"),
			Text(&state.Uptime),
			Space(),
			Text("Mode:"),
			Switch(&state.ViewMode).
				Case("all", Text("[ALL]")).
				Case("compact", Text("[COMPACT]")).
				Case("graphs", Text("[GRAPHS]")).
				Default(Text("[?]")),
		),

		// resource bars with threshold indicators
		HBox.Gap(1)(
			Text("CPU:"),
			Progress(&state.CPUTotal).Width(25),
			IfOrd(&state.CPUTotal).Gt(50).Then(
				IfOrd(&state.CPUTotal).Gt(80).Then(
					Text("CRIT").FG(critStyle.FG).Bold(),
				).Else(
					Text("WARN").FG(warnStyle.FG),
				),
			).Else(
				Text("    "),
			),
		),
		HBox.Gap(1)(
			Text("MEM:"),
			Progress(&state.MemTotal).Width(25),
			IfOrd(&state.MemTotal).Gte(60).Then(
				Text("HIGH").FG(warnStyle.FG),
			).Else(
				Text("    "),
			),
		),
		HBox.Gap(1)(
			Text("SWP:"),
			Progress(&state.SwapTotal).Width(25),
			IfOrd(&state.SwapTotal).Lt(20).Then(
				Text(" OK "),
			).Else(
				Text("USED").FG(warnStyle.FG),
			),
		),

		// main content - two columns with weighted widths (1:2)
		HBox.Gap(1)(
			// left panel
			VBox.Grow(1)(
				VBox.Border(BorderSingle).Title("Stats").BorderFG(Cyan)(
					Box{
						Layout:   Grid(2, 15, 0),
						Children: []any{Text(&state.Tasks), Text(&state.Running), Text(&state.Sleeping), Text(&state.Stopped)},
					},
				),
				VBox.Border(BorderRounded).Title("Load").BorderFG(Green)(
					Text(&state.Load),
				),
			),

			// right panel
			VBox.Grow(2)(
				Switch(&state.ViewMode).
					Case("all", VBox.Border(BorderSingle).Title("All Stats").BorderFG(Magenta)(
						Box{
							Layout:   Grid(3, 15, 1),
							Children: []any{Text(&state.Tasks), Text(&state.Threads), Text(&state.Running), Text(&state.Sleeping), Text(&state.Stopped), Text(&state.Zombie)},
						},
					)).
					Case("compact", HBox.Gap(2)(Text(&state.Tasks), Text(&state.Running), Text("Load:"), Text(&state.Load))).
					Case("graphs", Text("─── Graphs Mode ───")).
					Default(Text("Unknown view mode")),

				If(&state.ShowGraph).Eq(true).Then(
					VBox.Border(BorderRounded).Title("CPU History").BorderFG(cpuStyle.FG)(
						MiniGraph{Values: &state.CPUHistory, Width: 60, Height: 4, Style: cpuStyle},
					),
				),
			),
		),

		// middle section - vertical grow with weighted children
		VBox.Grow(1)(
			VBox.Border(BorderDouble).Title("Timing").BorderFG(Yellow).Grow(1)(
				HBox.Gap(1)(Text(&state.RenderLabel), MiniGraph{Values: &state.RenderHistory, Width: 60, Height: 2, Style: renderStyle}),
				HBox.Gap(1)(Text(&state.FlushLabel), MiniGraph{Values: &state.FlushHistory, Width: 60, Height: 2, Style: flushStyle}),
				HBox.Gap(2)(Text(&state.RowStats), Text(&state.FPSLabel)),
			),

			If(&state.ShowProcs).Eq(true).Then(
				VBox.Border(BorderSingle).Title("Processes").BorderFG(BrightBlue).Grow(2)(
					If(&state.Paused).Ne(false).Then(
						Text("=== PAUSED ===").FG(warnStyle.FG),
					),
					HBox.Gap(2)(Text(" "), Text("  PID"), Text("NAME        "), Text("  CPU"), Text("  MEM"), Text("STATUS  ")),
					ForEach(&state.Processes, func(p *Process) any {
						return HBox.Gap(2)(
							If(&p.Selected).Eq(true).Then(Text(">")).Else(Text(" ")),
							Text(&p.PID),
							Text(&p.Name),
							Text(&p.CPU),
							Text(&p.Mem),
							Text(&p.Status),
						)
					}),
				),
			),
		),

		// footer
		If(&state.ShowHelp).Eq(true).Then(Text(&state.HelpText)),
		Text(&state.Timing),
	)).
		// Key handlers
		Handle("q", func(_ riffkey.Match) {
			app.Stop()
		}).
		Handle("g", func(_ riffkey.Match) {
			state.ShowGraph = !state.ShowGraph
		}).
		Handle("p", func(_ riffkey.Match) {
			state.ShowProcs = !state.ShowProcs
		}).
		Handle("h", func(_ riffkey.Match) {
			state.ShowHelp = !state.ShowHelp
		}).
		Handle("<Space>", func(_ riffkey.Match) {
			state.Paused = !state.Paused
		}).
		Handle("j", func(_ riffkey.Match) {
			if state.SelectedProc < len(state.Processes)-1 {
				state.Processes[state.SelectedProc].Selected = false
				state.SelectedProc++
				state.Processes[state.SelectedProc].Selected = true
			}
		}).
		Handle("k", func(_ riffkey.Match) {
			if state.SelectedProc > 0 {
				state.Processes[state.SelectedProc].Selected = false
				state.SelectedProc--
				state.Processes[state.SelectedProc].Selected = true
			}
		}).
		// View mode cycling (Switch/Case demo)
		Handle("m", func(_ riffkey.Match) {
			switch state.ViewMode {
			case "all":
				state.ViewMode = "compact"
			case "compact":
				state.ViewMode = "graphs"
			case "graphs":
				state.ViewMode = "all"
			default:
				state.ViewMode = "all"
			}
		})

	// Animation ticker
	go func() {
		ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
		defer ticker.Stop()
		for range ticker.C {
			if state.Paused {
				continue
			}

			state.frameCount++

			// Update uptime
			elapsed := time.Since(state.startTime)
			hours := int(elapsed.Hours())
			mins := int(elapsed.Minutes()) % 60
			secs := int(elapsed.Seconds()) % 60
			state.Uptime = fmt.Sprintf("%d:%02d:%02d", hours, mins, secs)

			// Smooth load drift
			state.load1 += (rand.Float64() - 0.5) * 0.1
			state.load5 += (rand.Float64() - 0.5) * 0.05
			state.load15 += (rand.Float64() - 0.5) * 0.02
			state.load1 = math.Max(0.1, math.Min(4.0, state.load1))
			state.load5 = math.Max(0.1, math.Min(3.0, state.load5))
			state.load15 = math.Max(0.1, math.Min(2.0, state.load15))
			state.Load = fmt.Sprintf("%.2f, %.2f, %.2f", state.load1, state.load5, state.load15)

			// Shift targets occasionally
			if state.frameCount%30 == 0 {
				state.cpuTarget = 15 + rand.Float64()*70
				state.memTarget = 30 + rand.Float64()*20
				state.swapTarget = 5 + rand.Float64()*15
			}

			// Smooth interpolation
			state.CPUTotal = int(float64(state.CPUTotal) + (state.cpuTarget-float64(state.CPUTotal))*0.05)
			state.MemTotal = int(float64(state.MemTotal) + (state.memTarget-float64(state.MemTotal))*0.03)
			state.SwapTotal = int(float64(state.SwapTotal) + (state.swapTarget-float64(state.SwapTotal))*0.02)

			// Update history
			copy(state.CPUHistory, state.CPUHistory[1:])
			state.CPUHistory[len(state.CPUHistory)-1] = float64(state.CPUTotal)

			// Update processes
			for i := range state.procData {
				state.procData[i].cpu += (rand.Float64() - 0.5) * 0.3
				state.procData[i].mem += (rand.Float64() - 0.5) * 0.1
				state.procData[i].cpu = math.Max(0.1, math.Min(25.0, state.procData[i].cpu))
				state.procData[i].mem = math.Max(0.1, math.Min(15.0, state.procData[i].mem))
				state.Processes[i].CPU = fmt.Sprintf("%5.1f%%", state.procData[i].cpu)
				state.Processes[i].Mem = fmt.Sprintf("%5.1f%%", state.procData[i].mem)
			}

			// Update quick stats with some variation
			running := 2 + rand.Intn(3)
			sleeping := 139 + rand.Intn(5) - 2
			state.Running = fmt.Sprintf("Running:  %3d", running)
			state.Sleeping = fmt.Sprintf("Sleeping: %3d", sleeping)

			// Update timing stats
			state.Timing = TimingString()

			// Capture timing history for graphs
			timings := GetTimings()
			// Scale: 100 = 1000µs (1ms), so divide by 10 to get percentage
			copy(state.RenderHistory, state.RenderHistory[1:])
			copy(state.FlushHistory, state.FlushHistory[1:])
			state.RenderHistory[len(state.RenderHistory)-1] = timings.RenderUs / 10.0 // 1ms = 100%
			state.FlushHistory[len(state.FlushHistory)-1] = timings.FlushUs / 10.0

			// Update labels with current values
			state.RenderLabel = fmt.Sprintf("Render: %5.0fµs", timings.RenderUs)
			state.FlushLabel = fmt.Sprintf("Flush:  %5.0fµs", timings.FlushUs)

			// Get row stats from flush
			flushStats := GetFlushStats()
			state.RowStats = fmt.Sprintf("Rows: %d dirty, %d changed", flushStats.DirtyRows, flushStats.ChangedRows)

			// Track actual FPS
			state.fpsFrames++
			if time.Since(state.lastFPSTime) >= time.Second {
				state.currentFPS = float64(state.fpsFrames) / time.Since(state.lastFPSTime).Seconds()
				state.fpsFrames = 0
				state.lastFPSTime = time.Now()
			}
			state.FPSLabel = fmt.Sprintf("FPS: %.1f", state.currentFPS)

			// RenderNow() avoids channel coordination overhead
			app.RenderNow()
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
