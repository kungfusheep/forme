package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"riffkey"
	"tui"
)

// Process represents a running process
type Process struct {
	PID  int
	Name string
	CPU  int // 0-100
	Mem  int // 0-100
}

// State holds all dashboard data
type State struct {
	AllProcesses     []Process // Full list
	VisibleProcesses []Process // Slice for viewport
	CPUCores         []int     // 0-100
	Title            string
	Status           string
	Frame            int
	ViewportY        int
	VisibleRows      int
	NextPID          int
}

func main() {
	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize state
	state := &State{
		Title:       "=== System Monitor (Template POC) ===",
		Status:      "j/k=scroll g/G=top/bottom a=add x=del q=quit",
		VisibleRows: 10, // Will be adjusted based on terminal size
		NextPID:     1100,
	}

	// Create processes
	state.AllProcesses = make([]Process, 100)
	for i := range state.AllProcesses {
		state.AllProcesses[i] = Process{
			PID:  1000 + i,
			Name: fmt.Sprintf("proc-%02d", i),
			CPU:  i % 100,
			Mem:  (i * 7) % 100,
		}
	}

	// Create CPU cores
	state.CPUCores = make([]int, 8)
	for i := range state.CPUCores {
		state.CPUCores[i] = i*10 + 20
	}

	// Initialize visible slice
	state.VisibleProcesses = state.AllProcesses[:state.VisibleRows]

	// Build UI declaratively
	ui := tui.Col{Children: []any{
		tui.Text{Content: &state.Title},
		tui.Text{Content: ""},

		// CPU cores - horizontal row (manual since ForEach in Row not supported yet)
		tui.Text{Content: "CPU Cores:"},
		tui.Col{Gap: 1, Children: []any{
			tui.Progress{Value: &state.CPUCores[0], BarWidth: 8},
			tui.Progress{Value: &state.CPUCores[1], BarWidth: 8},
			tui.Progress{Value: &state.CPUCores[2], BarWidth: 8},
			tui.Progress{Value: &state.CPUCores[3], BarWidth: 8},
			tui.Progress{Value: &state.CPUCores[4], BarWidth: 8},
			tui.Progress{Value: &state.CPUCores[5], BarWidth: 8},
			tui.Progress{Value: &state.CPUCores[6], BarWidth: 8},
			tui.Progress{Value: &state.CPUCores[7], BarWidth: 8},
		}},

		tui.Text{Content: ""},
		tui.Text{Content: "Processes:"},

		// Process list - vertical ForEach with Row for each item
		tui.ForEach(&state.VisibleProcesses, func(p *Process) any {
			return tui.Row{Gap: 1, Children: []any{
				tui.Text{Content: &p.Name},
				tui.Progress{Value: &p.CPU, BarWidth: 15},
				tui.Progress{Value: &p.Mem, BarWidth: 15},
			}}
		}),

		tui.Text{Content: ""},
		tui.Text{Content: &state.Status},
	}}

	// Compile template ONCE
	tmpl := Compile(ui)
	_ = tmpl // TODO: integrate with app rendering

	// Set view and handlers
	app.SetView(ui).
		Handle("q", func(m riffkey.Match) {
			app.Stop()
		}).
		Handle("j", func(m riffkey.Match) {
			// Scroll down
			maxY := len(state.AllProcesses) - state.VisibleRows
			if maxY < 0 {
				maxY = 0
			}
			state.ViewportY++
			if state.ViewportY > maxY {
				state.ViewportY = maxY
			}
			updateVisibleSlice(state)
		}).
		Handle("k", func(m riffkey.Match) {
			// Scroll up
			state.ViewportY--
			if state.ViewportY < 0 {
				state.ViewportY = 0
			}
			updateVisibleSlice(state)
		}).
		Handle("g", func(m riffkey.Match) {
			// Go to top
			state.ViewportY = 0
			updateVisibleSlice(state)
		}).
		Handle("G", func(m riffkey.Match) {
			// Go to bottom
			maxY := len(state.AllProcesses) - state.VisibleRows
			if maxY < 0 {
				maxY = 0
			}
			state.ViewportY = maxY
			updateVisibleSlice(state)
		}).
		Handle("a", func(m riffkey.Match) {
			// Add process
			state.AllProcesses = append(state.AllProcesses, Process{
				PID:  state.NextPID,
				Name: fmt.Sprintf("new-%04d", state.NextPID),
				CPU:  50,
				Mem:  50,
			})
			state.NextPID++
			updateVisibleSlice(state)
		}).
		Handle("x", func(m riffkey.Match) {
			// Delete process (from end)
			if len(state.AllProcesses) > 0 {
				state.AllProcesses = state.AllProcesses[:len(state.AllProcesses)-1]
				// Adjust viewport if needed
				maxY := len(state.AllProcesses) - state.VisibleRows
				if maxY < 0 {
					maxY = 0
				}
				if state.ViewportY > maxY {
					state.ViewportY = maxY
				}
				updateVisibleSlice(state)
			}
		})

	// Animation ticker
	go func() {
		ticker := time.NewTicker(33 * time.Millisecond) // ~30 FPS
		defer ticker.Stop()
		for range ticker.C {
			state.Frame++

			// Animate CPU cores (sine waves) - values 0-100
			for i := range state.CPUCores {
				phase := float64(state.Frame)/20.0 + float64(i)*0.5
				state.CPUCores[i] = int(30 + 40*math.Sin(phase))
			}

			// Animate ALL process CPU/Mem - values 0-100
			for i := range state.AllProcesses {
				phase := float64(state.Frame)/30.0 + float64(i)*0.2
				state.AllProcesses[i].CPU = int(20 + 60*math.Sin(phase))
				state.AllProcesses[i].Mem = int(30 + 50*math.Cos(phase*0.7))
			}

			// Update status
			endY := state.ViewportY + state.VisibleRows
			if endY > len(state.AllProcesses) {
				endY = len(state.AllProcesses)
			}
			state.Status = fmt.Sprintf("Row %d-%d of %d | a=add x=del j/k/g/G q",
				state.ViewportY+1, endY, len(state.AllProcesses))

			app.RequestRender()
		}
	}()

	// Run the app
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func updateVisibleSlice(state *State) {
	totalProcs := len(state.AllProcesses)
	if state.ViewportY > totalProcs-state.VisibleRows {
		state.ViewportY = totalProcs - state.VisibleRows
	}
	if state.ViewportY < 0 {
		state.ViewportY = 0
	}
	endY := state.ViewportY + state.VisibleRows
	if endY > totalProcs {
		endY = totalProcs
	}
	if totalProcs > 0 {
		state.VisibleProcesses = state.AllProcesses[state.ViewportY:endY]
	} else {
		state.VisibleProcesses = nil
	}
}
