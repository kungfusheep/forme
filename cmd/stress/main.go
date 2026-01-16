package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"tui"

	"golang.org/x/term"
)

// Live data that updates each frame
var liveData = struct {
	Title     string
	Time      string
	FrameTime string
	FPS       string
	Processes []Process
	CPUCores  []Core
}{
	Title: "Serial Template Stress Test",
}

type Process struct {
	Name string
	CPU  float32
	Mem  float32
}

type Core struct {
	Name string
	Load float32
}

func init() {
	// Initialize processes
	liveData.Processes = make([]Process, 30)
	for i := range liveData.Processes {
		liveData.Processes[i] = Process{
			Name: fmt.Sprintf("proc-%02d", i),
			CPU:  rand.Float32(),
			Mem:  rand.Float32(),
		}
	}

	// Initialize CPU cores
	liveData.CPUCores = make([]Core, 8)
	for i := range liveData.CPUCores {
		liveData.CPUCores[i] = Core{
			Name: fmt.Sprintf("Core %d", i),
			Load: rand.Float32(),
		}
	}
}

func updateData() {
	liveData.Time = time.Now().Format("15:04:05.000")

	// Randomly update some processes
	for i := range liveData.Processes {
		liveData.Processes[i].CPU += (rand.Float32() - 0.5) * 0.1
		if liveData.Processes[i].CPU < 0 {
			liveData.Processes[i].CPU = 0
		}
		if liveData.Processes[i].CPU > 1 {
			liveData.Processes[i].CPU = 1
		}

		liveData.Processes[i].Mem += (rand.Float32() - 0.5) * 0.05
		if liveData.Processes[i].Mem < 0 {
			liveData.Processes[i].Mem = 0
		}
		if liveData.Processes[i].Mem > 1 {
			liveData.Processes[i].Mem = 1
		}
	}

	// Update CPU cores
	for i := range liveData.CPUCores {
		liveData.CPUCores[i].Load += (rand.Float32() - 0.5) * 0.2
		if liveData.CPUCores[i].Load < 0 {
			liveData.CPUCores[i].Load = 0
		}
		if liveData.CPUCores[i].Load > 1 {
			liveData.CPUCores[i].Load = 1
		}
	}
}

func main() {
	// Get terminal size
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 120, 40
	}

	// Build the UI template once
	ui := tui.Col{
		Children: []any{
			tui.Text{Content: &liveData.Title},
			tui.Row{Children: []any{
				tui.Text{Content: "Time: "},
				tui.Text{Content: &liveData.Time},
				tui.Text{Content: "  Frame: "},
				tui.Text{Content: &liveData.FrameTime},
				tui.Text{Content: "  "},
				tui.Text{Content: &liveData.FPS},
			}},
			tui.Text{Content: ""},
			tui.Text{Content: "═══ CPU Cores ═══════════════════════════════════════════════════"},
			tui.ForEach(&liveData.CPUCores, func(core *Core) any {
				return tui.Row{Children: []any{
					tui.Text{Content: &core.Name},
					tui.Text{Content: ": "},
					tui.Progress{Value: &core.Load, BarWidth: 50},
				}}
			}),
			tui.Text{Content: ""},
			tui.Text{Content: "═══ Processes ═══════════════════════════════════════════════════"},
			tui.ForEach(&liveData.Processes, func(proc *Process) any {
				return tui.Row{Children: []any{
					tui.Text{Content: &proc.Name},
					tui.Text{Content: " CPU:"},
					tui.Progress{Value: &proc.CPU, BarWidth: 25},
					tui.Text{Content: " MEM:"},
					tui.Progress{Value: &proc.Mem, BarWidth: 25},
				}}
			}),
		},
	}

	serial := tui.BuildSerial(ui)
	buf := tui.NewBuffer(width, height)

	// Hide cursor
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	// Clear screen
	fmt.Print("\033[2J")

	frameCount := 0
	lastFPSUpdate := time.Now()
	var totalFrameTime time.Duration

	for {
		frameStart := time.Now()

		// Update data
		updateData()

		// Render
		buf.ClearDirty()
		serial.Execute(buf, int16(width), int16(height))

		// Output to terminal
		fmt.Print("\033[H") // Move to top-left
		output := renderBuffer(buf)
		fmt.Print(output)

		frameTime := time.Since(frameStart)
		totalFrameTime += frameTime
		frameCount++

		// Update FPS every second
		if time.Since(lastFPSUpdate) >= time.Second {
			avgFrame := totalFrameTime / time.Duration(frameCount)
			liveData.FrameTime = fmt.Sprintf("%v", avgFrame.Round(time.Microsecond))
			liveData.FPS = fmt.Sprintf("(%d FPS)", frameCount)
			frameCount = 0
			totalFrameTime = 0
			lastFPSUpdate = time.Now()
		}

		// Target ~60 FPS
		sleepTime := (16 * time.Millisecond) - frameTime
		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}
	}
}

func renderBuffer(buf *tui.Buffer) string {
	w, h := buf.Size()
	// Pre-allocate roughly enough space
	result := make([]byte, 0, w*h*4)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cell := buf.Get(x, y)
			// Simple render - just the rune
			result = append(result, string(cell.Rune)...)
		}
		result = append(result, '\n')
	}
	return string(result)
}
