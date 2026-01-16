package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"

	"riffkey"
	"tui"
)

// Grid returns a layout function that arranges children in a grid
// If cellH is 0, it uses each child's natural height
func Grid(cols, cellW, cellH int) tui.LayoutFunc {
	return func(children []tui.ChildSize, availW, availH int) []tui.Rect {
		rects := make([]tui.Rect, len(children))

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
			rects[i] = tui.Rect{
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
	Style  tui.Style
}

func (g MiniGraph) MinSize() (width, height int) {
	h := g.Height
	if h < 1 {
		h = 8
	}
	return g.Width, h
}

func (g MiniGraph) Render(buf *tui.Buffer, x, y, w, h int) {
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

			buf.Set(x+i, rowY, tui.Cell{Rune: char, Style: g.Style})
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
	tui.DebugTiming = true

	app, err := tui.NewApp()
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

	cpuStyle := tui.Style{FG: tui.Color{Mode: tui.ColorRGB, R: 80, G: 200, B: 120}}
	renderStyle := tui.Style{FG: tui.Color{Mode: tui.ColorRGB, R: 255, G: 180, B: 80}} // Orange
	flushStyle := tui.Style{FG: tui.Color{Mode: tui.ColorRGB, R: 80, G: 180, B: 255}}  // Blue
	warnStyle := tui.Style{FG: tui.Yellow}
	critStyle := tui.Style{FG: tui.Red, Attr: tui.AttrBold}

	// Build UI with conditionals (using V2 template for Box, custom Renderer support)
	// Layout uses Grow() to push footer to bottom of screen
	app.SetV2View(tui.Col{Children: []any{
		// ══════════════════════════════════════════════════════════════
		// HEADER SECTION (fixed height)
		// ══════════════════════════════════════════════════════════════
		tui.Row{Gap: 2, Children: []any{
			tui.Text{Content: &state.Hostname},
			tui.Text{Content: "Uptime:"},
			tui.Text{Content: &state.Uptime},
			// HORIZONTAL GROW: Spacer pushes mode indicator to right side
			tui.Col{}.Grow(1),
			tui.Text{Content: "Mode:"},
			// SWITCH/CASE: Display different text based on ViewMode
			tui.Switch(&state.ViewMode).
				Case("all", tui.Text{Content: "[ALL]"}).
				Case("compact", tui.Text{Content: "[COMPACT]"}).
				Case("graphs", tui.Text{Content: "[GRAPHS]"}).
				Default(tui.Text{Content: "[?]"}),
		}},

		// Resource bars with threshold indicators
		tui.Row{Gap: 1, Children: []any{
			tui.Text{Content: "CPU:"},
			tui.Progress{Value: &state.CPUTotal, BarWidth: 25},
			// IfOrd.Gt: Show warning indicator when CPU > 50%
			tui.IfOrd(&state.CPUTotal).Gt(50).Then(
				tui.IfOrd(&state.CPUTotal).Gt(80).Then(
					tui.Text{Content: "CRIT", Style: tui.Style{FG: critStyle.FG, Attr: tui.AttrBold}},
				).Else(
					tui.Text{Content: "WARN", Style: tui.Style{FG: warnStyle.FG}},
				),
			).Else(
				tui.Text{Content: "    "},
			),
		}},
		tui.Row{Gap: 1, Children: []any{
			tui.Text{Content: "MEM:"},
			tui.Progress{Value: &state.MemTotal, BarWidth: 25},
			// IfOrd.Gte: Show warning at >= 60%
			tui.IfOrd(&state.MemTotal).Gte(60).Then(
				tui.Text{Content: "HIGH", Style: tui.Style{FG: warnStyle.FG}},
			).Else(
				tui.Text{Content: "    "},
			),
		}},
		tui.Row{Gap: 1, Children: []any{
			tui.Text{Content: "SWP:"},
			tui.Progress{Value: &state.SwapTotal, BarWidth: 25},
			// IfOrd.Lt: Show "LOW" when swap < 20% (inverse logic demo)
			tui.IfOrd(&state.SwapTotal).Lt(20).Then(
				tui.Text{Content: " OK "},
			).Else(
				tui.Text{Content: "USED", Style: tui.Style{FG: warnStyle.FG}},
			),
		}},

		// ══════════════════════════════════════════════════════════════
		// MAIN CONTENT - HORIZONTAL GROW: Two columns with weighted widths
		// Left panel Grow(1), Right panel Grow(2) = 1:2 width ratio
		// ══════════════════════════════════════════════════════════════
		tui.Row{Gap: 1, Children: []any{
			// LEFT PANEL: Grow(1) - gets 1/3 of width
			tui.Col{Children: []any{
				// BORDERED PANEL: Stats with single border
				tui.Col{
					Title: "Stats",
					Children: []any{
						tui.Box{
							Layout: Grid(2, 15, 0), // Wider cells to fit "Sleeping: 139"
							Children: []any{
								tui.Text{Content: &state.Tasks},
								tui.Text{Content: &state.Running},
								tui.Text{Content: &state.Sleeping},
								tui.Text{Content: &state.Stopped},
							},
						},
					},
				}.Border(tui.BorderSingle).BorderFG(tui.Cyan),

				// BORDERED PANEL: Load with rounded border
				tui.Col{
					Title: "Load",
					Children: []any{
						tui.Text{Content: &state.Load},
					},
				}.Border(tui.BorderRounded).BorderFG(tui.Green),
			}}.Grow(1),

			// RIGHT PANEL: Grow(2) - gets 2/3 of width
			tui.Col{Children: []any{
				// SWITCH/CASE: Show different content based on ViewMode
				tui.Switch(&state.ViewMode).
					Case("all", tui.Col{
						Title: "All Stats",
						Children: []any{
							tui.Box{
								Layout: Grid(3, 15, 1),
								Children: []any{
									tui.Text{Content: &state.Tasks},
									tui.Text{Content: &state.Threads},
									tui.Text{Content: &state.Running},
									tui.Text{Content: &state.Sleeping},
									tui.Text{Content: &state.Stopped},
									tui.Text{Content: &state.Zombie},
								},
							},
						},
					}.Border(tui.BorderSingle).BorderFG(tui.Magenta)).
					Case("compact", tui.Row{Gap: 2, Children: []any{
						tui.Text{Content: &state.Tasks},
						tui.Text{Content: &state.Running},
						tui.Text{Content: "Load:"},
						tui.Text{Content: &state.Load},
					}}).
					Case("graphs", tui.Text{Content: "─── Graphs Mode ───"}).
					Default(tui.Text{Content: "Unknown view mode"}),

				// Conditional: CPU Graph (If.Eq demo)
				tui.If(&state.ShowGraph).Eq(true).Then(
					tui.Col{
						Title: "CPU History",
						Children: []any{
							MiniGraph{Values: &state.CPUHistory, Width: 60, Height: 4, Style: cpuStyle},
						},
					}.Border(tui.BorderRounded).BorderFG(cpuStyle.FG),
				),
			}}.Grow(2),
		}},

		// ══════════════════════════════════════════════════════════════
		// MIDDLE SECTION - VERTICAL GROW with weighted children
		// Graphs Grow(1), Process list Grow(2) = 1:2 height ratio
		// ══════════════════════════════════════════════════════════════
		tui.Col{Children: []any{
			// GRAPHS SECTION: Grow(1) - gets 1/3 of remaining height
			tui.Col{
				Title: "Timing",
				Children: []any{
					tui.Row{Gap: 1, Children: []any{
						tui.Text{Content: &state.RenderLabel},
						MiniGraph{Values: &state.RenderHistory, Width: 60, Height: 2, Style: renderStyle},
					}},
					tui.Row{Gap: 1, Children: []any{
						tui.Text{Content: &state.FlushLabel},
						MiniGraph{Values: &state.FlushHistory, Width: 60, Height: 2, Style: flushStyle},
					}},
					tui.Row{Gap: 2, Children: []any{
						tui.Text{Content: &state.RowStats},
						tui.Text{Content: &state.FPSLabel},
					}},
				},
			}.Border(tui.BorderDouble).BorderFG(tui.Yellow).Grow(1),

			// PROCESS LIST: Grow(2) - gets 2/3 of remaining height
			tui.If(&state.ShowProcs).Eq(true).Then(tui.Col{
				Title: "Processes",
				Children: []any{
					// Show "PAUSED" header when paused (If.Ne demo)
					tui.If(&state.Paused).Ne(false).Then(
						tui.Text{Content: "=== PAUSED ===", Style: tui.Style{FG: warnStyle.FG}},
					),
					tui.Row{Gap: 2, Children: []any{
						tui.Text{Content: " "},
						tui.Text{Content: "  PID"},
						tui.Text{Content: "NAME        "},
						tui.Text{Content: "  CPU"},
						tui.Text{Content: "  MEM"},
						tui.Text{Content: "STATUS  "},
					}},
					// ForEach demo with nested conditionals
					tui.ForEach(&state.Processes, func(p *Process) any {
						return tui.Row{Gap: 2, Children: []any{
							tui.If(&p.Selected).Eq(true).Then(
								tui.Text{Content: ">"},
							).Else(
								tui.Text{Content: " "},
							),
							tui.Text{Content: &p.PID},
							tui.Text{Content: &p.Name},
							tui.Text{Content: &p.CPU},
							tui.Text{Content: &p.Mem},
							tui.Text{Content: &p.Status},
						}}
					}),
				},
			}.Border(tui.BorderSingle).BorderFG(tui.BrightBlue).Grow(2)),
		}}.Grow(1), // <-- OUTER GROW: This whole section expands vertically

		// ══════════════════════════════════════════════════════════════
		// FOOTER SECTION (fixed height, stays at bottom)
		// ══════════════════════════════════════════════════════════════
		// Conditional: Help bar
		tui.If(&state.ShowHelp).Eq(true).Then(
			tui.Text{Content: &state.HelpText},
		),

		// Render stats (always at bottom)
		tui.Text{Content: &state.Timing},
	}}).
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
			state.Timing = tui.TimingString()

			// Capture timing history for graphs
			timings := tui.GetTimings()
			// Scale: 100 = 1000µs (1ms), so divide by 10 to get percentage
			copy(state.RenderHistory, state.RenderHistory[1:])
			copy(state.FlushHistory, state.FlushHistory[1:])
			state.RenderHistory[len(state.RenderHistory)-1] = timings.RenderUs / 10.0 // 1ms = 100%
			state.FlushHistory[len(state.FlushHistory)-1] = timings.FlushUs / 10.0

			// Update labels with current values
			state.RenderLabel = fmt.Sprintf("Render: %5.0fµs", timings.RenderUs)
			state.FlushLabel = fmt.Sprintf("Flush:  %5.0fµs", timings.FlushUs)

			// Get row stats from flush
			flushStats := tui.GetFlushStats()
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
