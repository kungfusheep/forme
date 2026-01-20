package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"time"

	"tui"

	"golang.org/x/term"
)

var (
	duration  = flag.Duration("d", 5*time.Second, "benchmark duration")
	items     = flag.Int("items", 50, "number of items to render")
	asyncMode = flag.Bool("async", true, "use async buffer clearing")
	visual    = flag.Bool("visual", false, "show visual output")
	barWidth  = flag.Int("bar", 40, "progress bar width")
)

type Process struct {
	Name string
	CPU  float32
	Mem  float32
}

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	width, height := 120, 80
	if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		width, height = w, h
	}

	// Build data
	processes := make([]Process, *items)
	for i := range processes {
		processes[i] = Process{
			Name: fmt.Sprintf("proc-%03d", i),
			CPU:  rand.Float32(),
			Mem:  rand.Float32(),
		}
	}

	title := fmt.Sprintf("Benchmark: %d items, bar=%d, async=%v", *items, *barWidth, *asyncMode)

	// Build UI
	ui := tui.VBox{
		Children: []any{
			tui.Text{Content: &title},
			tui.ForEach(&processes, func(p *Process) any {
				return tui.HBox{Children: []any{
					tui.Text{Content: &p.Name},
					tui.Text{Content: " CPU:"},
					tui.Progress{Value: &p.CPU, BarWidth: int16(*barWidth)},
					tui.Text{Content: " MEM:"},
					tui.Progress{Value: &p.Mem, BarWidth: int16(*barWidth)},
				}}
			}),
		},
	}

	serial := tui.Build(ui)

	// Setup buffers
	var pool *tui.BufferPool
	var buf *tui.Buffer

	if *asyncMode {
		pool = tui.NewBufferPool(width, height)
		defer pool.Stop()
		buf = pool.Current()
	} else {
		buf = tui.NewBuffer(width, height)
	}

	// Collect frame times
	frameTimes := make([]time.Duration, 0, 100000)

	if *visual {
		fmt.Print("\033[?25l") // hide cursor
		fmt.Print("\033[2J")   // clear screen
		defer fmt.Print("\033[?25h")
	}

	fmt.Fprintf(os.Stderr, "Running benchmark for %v...\n", *duration)

	start := time.Now()
	frames := 0

	for time.Since(start) < *duration {
		frameStart := time.Now()

		// Simulate app logic - update some data
		for i := 0; i < len(processes)/10+1; i++ {
			idx := rand.Intn(len(processes))
			processes[idx].CPU += (rand.Float32() - 0.5) * 0.1
			if processes[idx].CPU < 0 {
				processes[idx].CPU = 0
			}
			if processes[idx].CPU > 1 {
				processes[idx].CPU = 1
			}
		}

		// Get buffer (swap if async)
		if *asyncMode {
			buf = pool.Swap()
		} else {
			buf.ClearDirty()
		}

		// Render
		serial.Execute(buf, int16(width), int16(height))

		frameTime := time.Since(frameStart)
		frameTimes = append(frameTimes, frameTime)
		frames++

		if *visual && frames%60 == 0 {
			fmt.Print("\033[H")
			printBuffer(buf)
		}
	}

	// Calculate stats
	sort.Slice(frameTimes, func(i, j int) bool {
		return frameTimes[i] < frameTimes[j]
	})

	var total time.Duration
	for _, ft := range frameTimes {
		total += ft
	}

	n := len(frameTimes)
	fmt.Fprintf(os.Stderr, "\n=== Results ===\n")
	fmt.Fprintf(os.Stderr, "Frames:     %d\n", frames)
	fmt.Fprintf(os.Stderr, "Duration:   %v\n", time.Since(start))
	fmt.Fprintf(os.Stderr, "Avg FPS:    %.0f\n", float64(frames)/time.Since(start).Seconds())
	fmt.Fprintf(os.Stderr, "\nFrame times:\n")
	fmt.Fprintf(os.Stderr, "  Min:      %v\n", frameTimes[0])
	fmt.Fprintf(os.Stderr, "  Max:      %v\n", frameTimes[n-1])
	fmt.Fprintf(os.Stderr, "  Avg:      %v\n", total/time.Duration(n))
	fmt.Fprintf(os.Stderr, "  P50:      %v\n", frameTimes[n*50/100])
	fmt.Fprintf(os.Stderr, "  P90:      %v\n", frameTimes[n*90/100])
	fmt.Fprintf(os.Stderr, "  P95:      %v\n", frameTimes[n*95/100])
	fmt.Fprintf(os.Stderr, "  P99:      %v\n", frameTimes[n*99/100])

	// Theoretical max FPS at each percentile
	fmt.Fprintf(os.Stderr, "\nTheoretical max FPS (framework only):\n")
	fmt.Fprintf(os.Stderr, "  At P50:   %.0f FPS\n", 1e9/float64(frameTimes[n*50/100].Nanoseconds()))
	fmt.Fprintf(os.Stderr, "  At P99:   %.0f FPS\n", 1e9/float64(frameTimes[n*99/100].Nanoseconds()))
}

func printBuffer(buf *tui.Buffer) {
	w, h := buf.Size()
	for y := 0; y < h && y < 40; y++ {
		for x := 0; x < w; x++ {
			cell := buf.Get(x, y)
			fmt.Print(string(cell.Rune))
		}
		fmt.Println()
	}
}
