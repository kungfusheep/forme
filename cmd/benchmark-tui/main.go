package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"time"

	"riffkey"
	"tui"
)

const (
	numCPUs       = 16
	numLogLines   = 200
	logViewHeight = 15
	numWorkers    = 12
	targetFPS     = 60
)

// State holds all the dashboard data
type State struct {
	// Header
	Title   string
	FPSText string
	TimeText string

	// CPU panel - 16 cores in 2 columns
	CPUCores [numCPUs]CPUCore

	// Memory panel
	MemUsed     int
	MemTotal    int
	MemText     string
	MemProgress int

	// Swap
	SwapUsed     int
	SwapTotal    int
	SwapText     string
	SwapProgress int

	// Network panel
	NetRxRate string
	NetTxRate string
	NetPacketsIn  string
	NetPacketsOut string

	// Disk panels (multiple disks)
	Disk1Read   string
	Disk1Write  string
	Disk1Text   string
	Disk2Read   string
	Disk2Write  string
	Disk2Text   string

	// Process panel
	ProcessCount   string
	ThreadCount    string
	GoroutineCount string
	HandleCount    string

	// Worker status
	Workers [numWorkers]WorkerStatus

	// Temperature sensors
	CPUTemp    string
	GPUTemp    string
	SysTemp    string
	FanSpeed   string

	// Load average
	Load1  string
	Load5  string
	Load15 string

	// Uptime
	Uptime string

	// Log entries
	LogEntries []LogEntry

	// Stats
	FrameCount  int64
	StartTime   time.Time
	LastFPSCalc time.Time
	FramesSince int
}

type CPUCore struct {
	Label    string
	Usage    int
	UsageStr string
}

type WorkerStatus struct {
	ID       int
	Status   string
	Tasks    int
	Progress int
}

type LogEntry struct {
	Time    string
	Level   string
	Source  string
	Message string
	Style   tui.Style
}

var state = &State{
	Title:     "TUI Performance Benchmark - Full Screen Dashboard",
	MemTotal:  65536, // 64GB
	SwapTotal: 16384, // 16GB
	StartTime: time.Now(),
}

var logLayer = tui.NewLayer()

var logBuffer *tui.Buffer

func init() {
	// Initialize CPU cores
	for i := range state.CPUCores {
		state.CPUCores[i].Label = fmt.Sprintf("Core%-2d", i)
	}

	// Initialize workers
	for i := range state.Workers {
		state.Workers[i].ID = i
		state.Workers[i].Status = "idle"
	}

	// Initialize log buffer directly (no template rebuild needed)
	logBuffer = tui.NewBuffer(120, numLogLines)
	logLayer.SetBuffer(logBuffer)

	// Initialize log entries and write directly to buffer
	state.LogEntries = make([]LogEntry, 0, numLogLines)
	for i := 0; i < numLogLines; i++ {
		addLogEntry()
	}
	// Render initial entries to buffer
	for i, entry := range state.LogEntries {
		writeLogEntryToBuffer(i, entry)
	}
	logLayer.ScrollToEnd()
}

func addLogEntry() {
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG", "TRACE"}
	levelStyles := []tui.Style{
		{FG: tui.Green},
		{FG: tui.Yellow},
		{FG: tui.Red, Attr: tui.AttrBold},
		{FG: tui.Cyan},
		{FG: tui.BrightBlack},
	}

	sources := []string{"kernel", "network", "storage", "worker", "scheduler", "gc", "http", "db"}
	messages := []string{
		"Processing incoming request from remote client",
		"Connection pool expanded to handle load",
		"Cache invalidation triggered for stale entries",
		"Database query optimized, execution plan updated",
		"Worker thread completed batch processing task",
		"Memory allocation pool resized dynamically",
		"Socket buffer flushed to network interface",
		"Heartbeat acknowledged from cluster node",
		"Configuration hot-reload completed successfully",
		"Task queued for asynchronous processing",
		"Rate limiter threshold adjusted automatically",
		"Garbage collection cycle completed efficiently",
		"TLS handshake completed with remote peer",
		"Load balancer health check passed",
	}

	idx := rand.Intn(len(levels))
	entry := LogEntry{
		Time:    time.Now().Format("15:04:05.000"),
		Level:   levels[idx],
		Source:  sources[rand.Intn(len(sources))],
		Message: messages[rand.Intn(len(messages))],
		Style:   levelStyles[idx],
	}

	if len(state.LogEntries) >= numLogLines {
		copy(state.LogEntries, state.LogEntries[1:])
		state.LogEntries[numLogLines-1] = entry
	} else {
		state.LogEntries = append(state.LogEntries, entry)
	}
}

func writeLogEntryToBuffer(row int, entry LogEntry) {
	spans := []tui.Span{
		{Text: entry.Time + " ", Style: tui.Style{FG: tui.BrightBlack}},
		{Text: fmt.Sprintf("%-5s ", entry.Level), Style: entry.Style},
		{Text: fmt.Sprintf("[%-9s] ", entry.Source), Style: tui.Style{FG: tui.Blue}},
		{Text: entry.Message, Style: tui.Style{}},
	}
	logBuffer.WriteSpans(0, row, spans, 120)
}

func appendLogEntry() {
	// Shift buffer up by one line (scroll content)
	logBuffer.Blit(logBuffer, 0, 1, 0, 0, 120, numLogLines-1)

	// Shift entries array
	if len(state.LogEntries) >= numLogLines {
		copy(state.LogEntries, state.LogEntries[1:])
		state.LogEntries = state.LogEntries[:numLogLines-1]
	}

	// Add new entry
	entry := generateLogEntry()
	state.LogEntries = append(state.LogEntries, entry)

	// Write new entry to last row of buffer
	writeLogEntryToBuffer(numLogLines-1, entry)
}

func generateLogEntry() LogEntry {
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG", "TRACE"}
	levelStyles := []tui.Style{
		{FG: tui.Green},
		{FG: tui.Yellow},
		{FG: tui.Red, Attr: tui.AttrBold},
		{FG: tui.Cyan},
		{FG: tui.BrightBlack},
	}

	sources := []string{"kernel", "network", "storage", "worker", "scheduler", "gc", "http", "db"}
	messages := []string{
		"Processing incoming request from remote client",
		"Connection pool expanded to handle load",
		"Cache invalidation triggered for stale entries",
		"Database query optimized, execution plan updated",
		"Worker thread completed batch processing task",
		"Memory allocation pool resized dynamically",
		"Socket buffer flushed to network interface",
		"Heartbeat acknowledged from cluster node",
		"Configuration hot-reload completed successfully",
		"Task queued for asynchronous processing",
		"Rate limiter threshold adjusted automatically",
		"Garbage collection cycle completed efficiently",
		"TLS handshake completed with remote peer",
		"Load balancer health check passed",
	}

	idx := rand.Intn(len(levels))
	return LogEntry{
		Time:    time.Now().Format("15:04:05.000"),
		Level:   levels[idx],
		Source:  sources[rand.Intn(len(sources))],
		Message: messages[rand.Intn(len(messages))],
		Style:   levelStyles[idx],
	}
}

func updateState() {
	t := float64(time.Since(state.StartTime).Milliseconds()) / 1000.0

	// Update time
	state.TimeText = time.Now().Format("2006-01-02 15:04:05")

	// Update CPU cores with varied sine-wave simulation
	for i := range state.CPUCores {
		phase := float64(i) * 0.5
		freq := 0.3 + float64(i%4)*0.15
		base := 20 + float64(i%8)*8
		usage := int(base + 45*math.Sin(t*freq+phase) + 15*math.Cos(t*freq*2+phase) + float64(rand.Intn(10)))
		if usage < 0 {
			usage = 0
		}
		if usage > 100 {
			usage = 100
		}
		state.CPUCores[i].Usage = usage
		state.CPUCores[i].UsageStr = fmt.Sprintf("%3d%%", usage)
	}

	// Update memory
	state.MemUsed = 40000 + int(15000*math.Sin(t*0.2)) + rand.Intn(2048)
	state.MemProgress = state.MemUsed * 100 / state.MemTotal
	state.MemText = fmt.Sprintf("%5d MB / %5d MB", state.MemUsed, state.MemTotal)

	// Update swap
	state.SwapUsed = 2048 + int(1024*math.Sin(t*0.15)) + rand.Intn(256)
	state.SwapProgress = state.SwapUsed * 100 / state.SwapTotal
	state.SwapText = fmt.Sprintf("%5d MB / %5d MB", state.SwapUsed, state.SwapTotal)

	// Update network
	rxRate := 850.5 + 300*math.Sin(t*0.8) + float64(rand.Intn(100))
	txRate := 420.2 + 150*math.Sin(t*0.6+1) + float64(rand.Intn(50))
	state.NetRxRate = fmt.Sprintf("RX: %7.1f MB/s", rxRate)
	state.NetTxRate = fmt.Sprintf("TX: %7.1f MB/s", txRate)
	state.NetPacketsIn = fmt.Sprintf("Packets In:  %d/s", 50000+rand.Intn(10000))
	state.NetPacketsOut = fmt.Sprintf("Packets Out: %d/s", 45000+rand.Intn(8000))

	// Update disks
	state.Disk1Read = fmt.Sprintf("Read:  %6.1f MB/s", 250+150*math.Sin(t*0.4)+float64(rand.Intn(30)))
	state.Disk1Write = fmt.Sprintf("Write: %6.1f MB/s", 180+100*math.Sin(t*0.5)+float64(rand.Intn(20)))
	state.Disk1Text = "nvme0n1 (SSD)"
	state.Disk2Read = fmt.Sprintf("Read:  %6.1f MB/s", 120+80*math.Sin(t*0.35)+float64(rand.Intn(15)))
	state.Disk2Write = fmt.Sprintf("Write: %6.1f MB/s", 90+60*math.Sin(t*0.45)+float64(rand.Intn(10)))
	state.Disk2Text = "sda (HDD)"

	// Update process counts
	state.ProcessCount = fmt.Sprintf("Processes:  %4d", 350+rand.Intn(50))
	state.ThreadCount = fmt.Sprintf("Threads:    %4d", 2800+rand.Intn(200))
	state.GoroutineCount = fmt.Sprintf("Goroutines: %4d", runtime.NumGoroutine())
	state.HandleCount = fmt.Sprintf("Handles:    %4d", 15000+rand.Intn(1000))

	// Update workers
	statuses := []string{"busy", "idle", "wait", "sync"}
	for i := range state.Workers {
		if rand.Intn(10) == 0 {
			state.Workers[i].Status = statuses[rand.Intn(len(statuses))]
		}
		state.Workers[i].Tasks = rand.Intn(100)
		state.Workers[i].Progress = rand.Intn(101)
	}

	// Update temperatures
	state.CPUTemp = fmt.Sprintf("CPU:  %2d°C", 45+rand.Intn(20))
	state.GPUTemp = fmt.Sprintf("GPU:  %2d°C", 50+rand.Intn(25))
	state.SysTemp = fmt.Sprintf("Sys:  %2d°C", 35+rand.Intn(10))
	state.FanSpeed = fmt.Sprintf("Fan: %4d RPM", 1200+rand.Intn(800))

	// Update load average
	load1 := 2.5 + 3*math.Sin(t*0.1) + rand.Float64()
	load5 := 2.0 + 2*math.Sin(t*0.08) + rand.Float64()
	load15 := 1.8 + 1.5*math.Sin(t*0.05) + rand.Float64()
	state.Load1 = fmt.Sprintf("1m: %.2f", load1)
	state.Load5 = fmt.Sprintf("5m: %.2f", load5)
	state.Load15 = fmt.Sprintf("15m: %.2f", load15)

	// Update uptime
	uptime := time.Since(state.StartTime)
	state.Uptime = fmt.Sprintf("Uptime: %s", uptime.Round(time.Second))

	// Add new log entries frequently (direct buffer write, no template rebuild)
	if rand.Intn(2) == 0 {
		appendLogEntry()
	}

	// Update FPS
	state.FrameCount++
	state.FramesSince++
	now := time.Now()
	elapsed := now.Sub(state.LastFPSCalc)
	if elapsed >= 500*time.Millisecond {
		fps := float64(state.FramesSince) / elapsed.Seconds()
		state.FPSText = fmt.Sprintf("FPS: %.1f | Frames: %d | Avg: %.1f", fps, state.FrameCount, float64(state.FrameCount)/time.Since(state.StartTime).Seconds())
		state.FramesSince = 0
		state.LastFPSCalc = now
	}
}

func buildView() any {
	// CPU bars in 2 columns (8 per column)
	cpuCol1 := make([]any, numCPUs/2)
	cpuCol2 := make([]any, numCPUs/2)
	for i := 0; i < numCPUs/2; i++ {
		cpuCol1[i] = tui.Row{Children: []any{
			tui.Text{Content: &state.CPUCores[i].Label},
			tui.Text{Content: " "},
			tui.Progress{Value: &state.CPUCores[i].Usage, BarWidth: 20},
			tui.Text{Content: " "},
			tui.Text{Content: &state.CPUCores[i].UsageStr},
		}}
		cpuCol2[i] = tui.Row{Children: []any{
			tui.Text{Content: &state.CPUCores[i+numCPUs/2].Label},
			tui.Text{Content: " "},
			tui.Progress{Value: &state.CPUCores[i+numCPUs/2].Usage, BarWidth: 20},
			tui.Text{Content: " "},
			tui.Text{Content: &state.CPUCores[i+numCPUs/2].UsageStr},
		}}
	}

	// Worker status rows
	workerRows := make([]any, numWorkers/2)
	for i := 0; i < numWorkers/2; i++ {
		w1 := &state.Workers[i]
		w2 := &state.Workers[i+numWorkers/2]
		workerRows[i] = tui.Row{Gap: 4, Children: []any{
			tui.Row{Children: []any{
				tui.Text{Content: fmt.Sprintf("W%02d ", w1.ID)},
				tui.Progress{Value: &w1.Progress, BarWidth: 12},
				tui.Text{Content: fmt.Sprintf(" %s", w1.Status)},
			}},
			tui.Row{Children: []any{
				tui.Text{Content: fmt.Sprintf("W%02d ", w2.ID)},
				tui.Progress{Value: &w2.Progress, BarWidth: 12},
				tui.Text{Content: fmt.Sprintf(" %s", w2.Status)},
			}},
		}}
	}

	return tui.Col{Children: []any{
		// Header
		tui.RichText{Spans: []tui.Span{
			{Text: "════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════", Style: tui.Style{FG: tui.Cyan}},
		}},
		tui.Row{Children: []any{
			tui.RichText{Spans: []tui.Span{
				{Text: "  ", Style: tui.Style{}},
				{Text: state.Title, Style: tui.Style{FG: tui.BrightWhite, Attr: tui.AttrBold}},
			}},
		}},
		tui.Row{Gap: 4, Children: []any{
			tui.Text{Content: &state.FPSText, Style: tui.Style{FG: tui.Green}},
			tui.Text{Content: &state.TimeText},
			tui.Text{Content: &state.Uptime},
		}},
		tui.RichText{Spans: []tui.Span{
			{Text: "════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════", Style: tui.Style{FG: tui.Cyan}},
		}},
		tui.Text{},

		// CPU Panel (2 columns)
		tui.RichText{Spans: []tui.Span{
			{Text: "┌─ ", Style: tui.Style{FG: tui.Blue}},
			{Text: "CPU Usage (16 Cores)", Style: tui.Style{FG: tui.BrightBlue, Attr: tui.AttrBold}},
			{Text: " ─────────────────────────────────────────────────────────────────────────────────────────────┐", Style: tui.Style{FG: tui.Blue}},
		}},
		tui.Row{Gap: 8, Children: []any{
			tui.Col{Children: cpuCol1},
			tui.Col{Children: cpuCol2},
			// Temps in the CPU panel
			tui.Col{Children: []any{
				tui.Text{Content: &state.CPUTemp, Style: tui.Style{FG: tui.Yellow}},
				tui.Text{Content: &state.GPUTemp, Style: tui.Style{FG: tui.Yellow}},
				tui.Text{Content: &state.SysTemp, Style: tui.Style{FG: tui.Yellow}},
				tui.Text{Content: &state.FanSpeed},
				tui.Text{},
				tui.Text{Content: "Load Average:", Style: tui.Style{Attr: tui.AttrBold}},
				tui.Text{Content: &state.Load1},
				tui.Text{Content: &state.Load5},
				tui.Text{Content: &state.Load15},
			}},
		}},
		tui.RichText{Spans: []tui.Span{
			{Text: "└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘", Style: tui.Style{FG: tui.Blue}},
		}},
		tui.Text{},

		// Memory & Process Info Row
		tui.Row{Gap: 4, Children: []any{
			// Memory Panel
			tui.Col{Children: []any{
				tui.RichText{Spans: []tui.Span{
					{Text: "┌─ ", Style: tui.Style{FG: tui.Magenta}},
					{Text: "Memory", Style: tui.Style{FG: tui.BrightMagenta, Attr: tui.AttrBold}},
					{Text: " ───────────────────────────────┐", Style: tui.Style{FG: tui.Magenta}},
				}},
				tui.Row{Children: []any{
					tui.Text{Content: "RAM:  "},
					tui.Progress{Value: &state.MemProgress, BarWidth: 25},
					tui.Text{Content: " "},
					tui.Text{Content: &state.MemText},
				}},
				tui.Row{Children: []any{
					tui.Text{Content: "Swap: "},
					tui.Progress{Value: &state.SwapProgress, BarWidth: 25},
					tui.Text{Content: " "},
					tui.Text{Content: &state.SwapText},
				}},
				tui.RichText{Spans: []tui.Span{
					{Text: "└────────────────────────────────────────────┘", Style: tui.Style{FG: tui.Magenta}},
				}},
			}},
			// Process Panel
			tui.Col{Children: []any{
				tui.RichText{Spans: []tui.Span{
					{Text: "┌─ ", Style: tui.Style{FG: tui.Green}},
					{Text: "Processes", Style: tui.Style{FG: tui.BrightGreen, Attr: tui.AttrBold}},
					{Text: " ────────────────┐", Style: tui.Style{FG: tui.Green}},
				}},
				tui.Text{Content: &state.ProcessCount},
				tui.Text{Content: &state.ThreadCount},
				tui.Text{Content: &state.GoroutineCount},
				tui.Text{Content: &state.HandleCount},
				tui.RichText{Spans: []tui.Span{
					{Text: "└────────────────────────────┘", Style: tui.Style{FG: tui.Green}},
				}},
			}},
		}},
		tui.Text{},

		// Network & Disk Row
		tui.Row{Gap: 4, Children: []any{
			// Network Panel
			tui.Col{Children: []any{
				tui.RichText{Spans: []tui.Span{
					{Text: "┌─ ", Style: tui.Style{FG: tui.Cyan}},
					{Text: "Network", Style: tui.Style{FG: tui.BrightCyan, Attr: tui.AttrBold}},
					{Text: " ──────────────────────┐", Style: tui.Style{FG: tui.Cyan}},
				}},
				tui.Text{Content: &state.NetRxRate},
				tui.Text{Content: &state.NetTxRate},
				tui.Text{Content: &state.NetPacketsIn},
				tui.Text{Content: &state.NetPacketsOut},
				tui.RichText{Spans: []tui.Span{
					{Text: "└───────────────────────────────┘", Style: tui.Style{FG: tui.Cyan}},
				}},
			}},
			// Disk 1 Panel
			tui.Col{Children: []any{
				tui.RichText{Spans: []tui.Span{
					{Text: "┌─ ", Style: tui.Style{FG: tui.Yellow}},
					{Text: "Disk: nvme0n1", Style: tui.Style{FG: tui.BrightYellow, Attr: tui.AttrBold}},
					{Text: " ────────┐", Style: tui.Style{FG: tui.Yellow}},
				}},
				tui.Text{Content: &state.Disk1Read},
				tui.Text{Content: &state.Disk1Write},
				tui.Text{},
				tui.Text{},
				tui.RichText{Spans: []tui.Span{
					{Text: "└──────────────────────────────┘", Style: tui.Style{FG: tui.Yellow}},
				}},
			}},
			// Disk 2 Panel
			tui.Col{Children: []any{
				tui.RichText{Spans: []tui.Span{
					{Text: "┌─ ", Style: tui.Style{FG: tui.Yellow}},
					{Text: "Disk: sda", Style: tui.Style{FG: tui.BrightYellow, Attr: tui.AttrBold}},
					{Text: " ────────────┐", Style: tui.Style{FG: tui.Yellow}},
				}},
				tui.Text{Content: &state.Disk2Read},
				tui.Text{Content: &state.Disk2Write},
				tui.Text{},
				tui.Text{},
				tui.RichText{Spans: []tui.Span{
					{Text: "└──────────────────────────────┘", Style: tui.Style{FG: tui.Yellow}},
				}},
			}},
		}},
		tui.Text{},

		// Workers Panel
		tui.RichText{Spans: []tui.Span{
			{Text: "┌─ ", Style: tui.Style{FG: tui.Red}},
			{Text: "Worker Pool Status", Style: tui.Style{FG: tui.BrightRed, Attr: tui.AttrBold}},
			{Text: " ──────────────────────────────────────────────────────────────────────────────────────────────┐", Style: tui.Style{FG: tui.Red}},
		}},
		tui.Col{Children: workerRows},
		tui.RichText{Spans: []tui.Span{
			{Text: "└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘", Style: tui.Style{FG: tui.Red}},
		}},
		tui.Text{},

		// Log Panel
		tui.RichText{Spans: []tui.Span{
			{Text: "┌─ ", Style: tui.Style{FG: tui.White}},
			{Text: "Live System Logs", Style: tui.Style{FG: tui.BrightWhite, Attr: tui.AttrBold}},
			{Text: " ────────────────────────────────────────────────────────────────────────────────────────────────┐", Style: tui.Style{FG: tui.White}},
		}},
		tui.LayerView{Layer: logLayer, ViewHeight: logViewHeight},
		tui.RichText{Spans: []tui.Span{
			{Text: "└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘", Style: tui.Style{FG: tui.White}},
		}},
		tui.Text{},
		tui.Text{Content: "Press 'q' to quit"},
	}}
}

func main() {
	// Parse duration from args
	duration := 10 * time.Second
	profile := false
	for _, arg := range os.Args[1:] {
		if arg == "-profile" {
			profile = true
		} else if d, err := strconv.Atoi(arg); err == nil {
			duration = time.Duration(d) * time.Second
		}
	}

	// CPU profiling
	if profile {
		f, err := os.Create("cpu.prof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	state.LastFPSCalc = time.Now()

	app.SetView(buildView()).
		Handle("q", func(_ riffkey.Match) {
			app.Stop()
		})

	// Main update loop
	go func() {
		ticker := time.NewTicker(time.Second / targetFPS)
		defer ticker.Stop()

		timeout := time.After(duration)
		for {
			select {
			case <-ticker.C:
				updateState()
				app.RequestRender()
			case <-timeout:
				app.Stop()
				return
			}
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

	// Print final stats
	elapsed := time.Since(state.StartTime)
	avgFPS := float64(state.FrameCount) / elapsed.Seconds()
	fmt.Printf("\n=== TUI Benchmark Results ===\n")
	fmt.Printf("Duration: %.2fs\n", elapsed.Seconds())
	fmt.Printf("Total Frames: %d\n", state.FrameCount)
	fmt.Printf("Average FPS: %.2f\n", avgFPS)
	fmt.Printf("Target FPS: %d\n", targetFPS)
	fmt.Printf("Efficiency: %.1f%%\n", (avgFPS/float64(targetFPS))*100)
}
