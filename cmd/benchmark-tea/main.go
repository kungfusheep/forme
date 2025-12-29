package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	numCPUs       = 16
	numLogLines   = 200
	logViewHeight = 15
	numWorkers    = 12
	targetFPS     = 60
)

// Styles
var (
	cyan          = lipgloss.Color("6")
	blue          = lipgloss.Color("4")
	brightBlue    = lipgloss.Color("12")
	magenta       = lipgloss.Color("5")
	brightMagenta = lipgloss.Color("13")
	green         = lipgloss.Color("2")
	brightGreen   = lipgloss.Color("10")
	yellow        = lipgloss.Color("3")
	brightYellow  = lipgloss.Color("11")
	red           = lipgloss.Color("1")
	brightRed     = lipgloss.Color("9")
	white         = lipgloss.Color("7")
	brightWhite   = lipgloss.Color("15")
	brightBlack   = lipgloss.Color("8")

	titleStyle     = lipgloss.NewStyle().Foreground(brightWhite).Bold(true)
	fpsStyle       = lipgloss.NewStyle().Foreground(green)
	borderCyan     = lipgloss.NewStyle().Foreground(cyan)
	borderBlue     = lipgloss.NewStyle().Foreground(blue)
	headerBlue     = lipgloss.NewStyle().Foreground(brightBlue).Bold(true)
	borderMagenta  = lipgloss.NewStyle().Foreground(magenta)
	headerMagenta  = lipgloss.NewStyle().Foreground(brightMagenta).Bold(true)
	borderGreen    = lipgloss.NewStyle().Foreground(green)
	headerGreen    = lipgloss.NewStyle().Foreground(brightGreen).Bold(true)
	borderYellow   = lipgloss.NewStyle().Foreground(yellow)
	headerYellow   = lipgloss.NewStyle().Foreground(brightYellow).Bold(true)
	borderRed      = lipgloss.NewStyle().Foreground(red)
	headerRed      = lipgloss.NewStyle().Foreground(brightRed).Bold(true)
	borderWhite    = lipgloss.NewStyle().Foreground(white)
	headerWhite    = lipgloss.NewStyle().Foreground(brightWhite).Bold(true)
	dimStyle       = lipgloss.NewStyle().Foreground(brightBlack)
	greenStyle     = lipgloss.NewStyle().Foreground(green)
	yellowStyle    = lipgloss.NewStyle().Foreground(yellow)
	redBoldStyle   = lipgloss.NewStyle().Foreground(red).Bold(true)
	cyanStyle      = lipgloss.NewStyle().Foreground(cyan)
	blueStyle      = lipgloss.NewStyle().Foreground(blue)
	yellowTempStyle = lipgloss.NewStyle().Foreground(yellow)
)

type tickMsg time.Time

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
	Color   lipgloss.Color
	Bold    bool
}

type model struct {
	// Header
	Title    string
	FPSText  string
	TimeText string

	// CPU panel
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
	NetRxRate     string
	NetTxRate     string
	NetPacketsIn  string
	NetPacketsOut string

	// Disk panels
	Disk1Read  string
	Disk1Write string
	Disk2Read  string
	Disk2Write string

	// Process panel
	ProcessCount   string
	ThreadCount    string
	GoroutineCount string
	HandleCount    string

	// Worker status
	Workers [numWorkers]WorkerStatus

	// Temperature sensors
	CPUTemp  string
	GPUTemp  string
	SysTemp  string
	FanSpeed string

	// Load average
	Load1  string
	Load5  string
	Load15 string

	// Uptime
	Uptime string

	// Log entries
	LogEntries []LogEntry
	LogScroll  int

	// Stats
	FrameCount  int64
	StartTime   time.Time
	LastFPSCalc time.Time
	FramesSince int

	// Control
	Duration time.Duration
	Quitting bool
}

func initialModel(duration time.Duration) model {
	m := model{
		Title:     "Bubbletea Performance Benchmark - Full Screen Dashboard",
		MemTotal:  65536,
		SwapTotal: 16384,
		StartTime: time.Now(),
		Duration:  duration,
	}

	// Initialize CPU cores
	for i := range m.CPUCores {
		m.CPUCores[i].Label = fmt.Sprintf("Core%-2d", i)
	}

	// Initialize workers
	for i := range m.Workers {
		m.Workers[i].ID = i
		m.Workers[i].Status = "idle"
	}

	// Initialize log entries
	m.LogEntries = make([]LogEntry, 0, numLogLines)
	for i := 0; i < numLogLines; i++ {
		m.LogEntries = append(m.LogEntries, generateLogEntry())
	}

	return m
}

func generateLogEntry() LogEntry {
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG", "TRACE"}
	colors := []lipgloss.Color{green, yellow, red, cyan, brightBlack}
	bolds := []bool{false, false, true, false, false}

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
		Color:   colors[idx],
		Bold:    bolds[idx],
	}
}

func (m model) Init() tea.Cmd {
	return tea.Tick(time.Second/targetFPS, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.Quitting = true
			return m, tea.Quit
		}

	case tickMsg:
		// Check duration
		if time.Since(m.StartTime) >= m.Duration {
			m.Quitting = true
			return m, tea.Quit
		}

		m = m.updateState()
		return m, tea.Tick(time.Second/targetFPS, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}

	return m, nil
}

func (m model) updateState() model {
	t := float64(time.Since(m.StartTime).Milliseconds()) / 1000.0

	// Update time
	m.TimeText = time.Now().Format("2006-01-02 15:04:05")

	// Update CPU cores
	for i := range m.CPUCores {
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
		m.CPUCores[i].Usage = usage
		m.CPUCores[i].UsageStr = fmt.Sprintf("%3d%%", usage)
	}

	// Update memory
	m.MemUsed = 40000 + int(15000*math.Sin(t*0.2)) + rand.Intn(2048)
	m.MemProgress = m.MemUsed * 100 / m.MemTotal
	m.MemText = fmt.Sprintf("%5d MB / %5d MB", m.MemUsed, m.MemTotal)

	// Update swap
	m.SwapUsed = 2048 + int(1024*math.Sin(t*0.15)) + rand.Intn(256)
	m.SwapProgress = m.SwapUsed * 100 / m.SwapTotal
	m.SwapText = fmt.Sprintf("%5d MB / %5d MB", m.SwapUsed, m.SwapTotal)

	// Update network
	rxRate := 850.5 + 300*math.Sin(t*0.8) + float64(rand.Intn(100))
	txRate := 420.2 + 150*math.Sin(t*0.6+1) + float64(rand.Intn(50))
	m.NetRxRate = fmt.Sprintf("RX: %7.1f MB/s", rxRate)
	m.NetTxRate = fmt.Sprintf("TX: %7.1f MB/s", txRate)
	m.NetPacketsIn = fmt.Sprintf("Packets In:  %d/s", 50000+rand.Intn(10000))
	m.NetPacketsOut = fmt.Sprintf("Packets Out: %d/s", 45000+rand.Intn(8000))

	// Update disks
	m.Disk1Read = fmt.Sprintf("Read:  %6.1f MB/s", 250+150*math.Sin(t*0.4)+float64(rand.Intn(30)))
	m.Disk1Write = fmt.Sprintf("Write: %6.1f MB/s", 180+100*math.Sin(t*0.5)+float64(rand.Intn(20)))
	m.Disk2Read = fmt.Sprintf("Read:  %6.1f MB/s", 120+80*math.Sin(t*0.35)+float64(rand.Intn(15)))
	m.Disk2Write = fmt.Sprintf("Write: %6.1f MB/s", 90+60*math.Sin(t*0.45)+float64(rand.Intn(10)))

	// Update process counts
	m.ProcessCount = fmt.Sprintf("Processes:  %4d", 350+rand.Intn(50))
	m.ThreadCount = fmt.Sprintf("Threads:    %4d", 2800+rand.Intn(200))
	m.GoroutineCount = fmt.Sprintf("Goroutines: %4d", runtime.NumGoroutine())
	m.HandleCount = fmt.Sprintf("Handles:    %4d", 15000+rand.Intn(1000))

	// Update workers
	statuses := []string{"busy", "idle", "wait", "sync"}
	for i := range m.Workers {
		if rand.Intn(10) == 0 {
			m.Workers[i].Status = statuses[rand.Intn(len(statuses))]
		}
		m.Workers[i].Tasks = rand.Intn(100)
		m.Workers[i].Progress = rand.Intn(101)
	}

	// Update temperatures
	m.CPUTemp = fmt.Sprintf("CPU:  %2d°C", 45+rand.Intn(20))
	m.GPUTemp = fmt.Sprintf("GPU:  %2d°C", 50+rand.Intn(25))
	m.SysTemp = fmt.Sprintf("Sys:  %2d°C", 35+rand.Intn(10))
	m.FanSpeed = fmt.Sprintf("Fan: %4d RPM", 1200+rand.Intn(800))

	// Update load average
	load1 := 2.5 + 3*math.Sin(t*0.1) + rand.Float64()
	load5 := 2.0 + 2*math.Sin(t*0.08) + rand.Float64()
	load15 := 1.8 + 1.5*math.Sin(t*0.05) + rand.Float64()
	m.Load1 = fmt.Sprintf("1m: %.2f", load1)
	m.Load5 = fmt.Sprintf("5m: %.2f", load5)
	m.Load15 = fmt.Sprintf("15m: %.2f", load15)

	// Update uptime
	uptime := time.Since(m.StartTime)
	m.Uptime = fmt.Sprintf("Uptime: %s", uptime.Round(time.Second))

	// Add new log entries
	if rand.Intn(2) == 0 {
		if len(m.LogEntries) >= numLogLines {
			m.LogEntries = append(m.LogEntries[1:], generateLogEntry())
		} else {
			m.LogEntries = append(m.LogEntries, generateLogEntry())
		}
	}

	// Update FPS
	m.FrameCount++
	m.FramesSince++
	now := time.Now()
	elapsed := now.Sub(m.LastFPSCalc)
	if elapsed >= 500*time.Millisecond {
		fps := float64(m.FramesSince) / elapsed.Seconds()
		m.FPSText = fmt.Sprintf("FPS: %.1f | Frames: %d | Avg: %.1f", fps, m.FrameCount, float64(m.FrameCount)/time.Since(m.StartTime).Seconds())
		m.FramesSince = 0
		m.LastFPSCalc = now
	}

	return m
}

func renderProgressBar(percent int, width int) string {
	filled := width * percent / 100
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "]"
}

func (m model) View() string {
	if m.Quitting {
		return ""
	}

	var b strings.Builder

	// Header
	b.WriteString(borderCyan.Render("════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════"))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(titleStyle.Render(m.Title))
	b.WriteString("\n")
	b.WriteString(fpsStyle.Render(m.FPSText))
	b.WriteString("    ")
	b.WriteString(m.TimeText)
	b.WriteString("    ")
	b.WriteString(m.Uptime)
	b.WriteString("\n")
	b.WriteString(borderCyan.Render("════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════"))
	b.WriteString("\n\n")

	// CPU Panel
	b.WriteString(borderBlue.Render("┌─ "))
	b.WriteString(headerBlue.Render("CPU Usage (16 Cores)"))
	b.WriteString(borderBlue.Render(" ─────────────────────────────────────────────────────────────────────────────────────────────┐"))
	b.WriteString("\n")

	// CPU in 2 columns with temps
	for i := 0; i < numCPUs/2; i++ {
		// Column 1
		b.WriteString(m.CPUCores[i].Label)
		b.WriteString(" ")
		b.WriteString(renderProgressBar(m.CPUCores[i].Usage, 20))
		b.WriteString(" ")
		b.WriteString(m.CPUCores[i].UsageStr)
		b.WriteString("        ")

		// Column 2
		b.WriteString(m.CPUCores[i+numCPUs/2].Label)
		b.WriteString(" ")
		b.WriteString(renderProgressBar(m.CPUCores[i+numCPUs/2].Usage, 20))
		b.WriteString(" ")
		b.WriteString(m.CPUCores[i+numCPUs/2].UsageStr)
		b.WriteString("        ")

		// Temps column (only first few rows)
		switch i {
		case 0:
			b.WriteString(yellowTempStyle.Render(m.CPUTemp))
		case 1:
			b.WriteString(yellowTempStyle.Render(m.GPUTemp))
		case 2:
			b.WriteString(yellowTempStyle.Render(m.SysTemp))
		case 3:
			b.WriteString(m.FanSpeed)
		case 5:
			b.WriteString(lipgloss.NewStyle().Bold(true).Render("Load Average:"))
		case 6:
			b.WriteString(m.Load1)
		case 7:
			b.WriteString(m.Load5)
		}
		b.WriteString("\n")
	}
	b.WriteString(borderBlue.Render("└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘"))
	b.WriteString("\n\n")

	// Memory Panel
	b.WriteString(borderMagenta.Render("┌─ "))
	b.WriteString(headerMagenta.Render("Memory"))
	b.WriteString(borderMagenta.Render(" ───────────────────────────────┐"))
	b.WriteString("    ")
	b.WriteString(borderGreen.Render("┌─ "))
	b.WriteString(headerGreen.Render("Processes"))
	b.WriteString(borderGreen.Render(" ────────────────┐"))
	b.WriteString("\n")

	b.WriteString("RAM:  ")
	b.WriteString(renderProgressBar(m.MemProgress, 25))
	b.WriteString(" ")
	b.WriteString(m.MemText)
	b.WriteString("    ")
	b.WriteString(m.ProcessCount)
	b.WriteString("\n")

	b.WriteString("Swap: ")
	b.WriteString(renderProgressBar(m.SwapProgress, 25))
	b.WriteString(" ")
	b.WriteString(m.SwapText)
	b.WriteString("    ")
	b.WriteString(m.ThreadCount)
	b.WriteString("\n")

	b.WriteString(borderMagenta.Render("└────────────────────────────────────────────┘"))
	b.WriteString("    ")
	b.WriteString(m.GoroutineCount)
	b.WriteString("\n")

	b.WriteString("                                                  ")
	b.WriteString(m.HandleCount)
	b.WriteString("\n")

	b.WriteString("                                                  ")
	b.WriteString(borderGreen.Render("└────────────────────────────┘"))
	b.WriteString("\n\n")

	// Network & Disk Row
	b.WriteString(borderCyan.Render("┌─ "))
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render("Network"))
	b.WriteString(borderCyan.Render(" ──────────────────────┐"))
	b.WriteString("    ")
	b.WriteString(borderYellow.Render("┌─ "))
	b.WriteString(headerYellow.Render("Disk: nvme0n1"))
	b.WriteString(borderYellow.Render(" ────────┐"))
	b.WriteString("    ")
	b.WriteString(borderYellow.Render("┌─ "))
	b.WriteString(headerYellow.Render("Disk: sda"))
	b.WriteString(borderYellow.Render(" ────────────┐"))
	b.WriteString("\n")

	b.WriteString(m.NetRxRate)
	b.WriteString("                        ")
	b.WriteString(m.Disk1Read)
	b.WriteString("               ")
	b.WriteString(m.Disk2Read)
	b.WriteString("\n")

	b.WriteString(m.NetTxRate)
	b.WriteString("                        ")
	b.WriteString(m.Disk1Write)
	b.WriteString("               ")
	b.WriteString(m.Disk2Write)
	b.WriteString("\n")

	b.WriteString(m.NetPacketsIn)
	b.WriteString("\n")
	b.WriteString(m.NetPacketsOut)
	b.WriteString("\n")

	b.WriteString(borderCyan.Render("└───────────────────────────────┘"))
	b.WriteString("    ")
	b.WriteString(borderYellow.Render("└──────────────────────────────┘"))
	b.WriteString("    ")
	b.WriteString(borderYellow.Render("└──────────────────────────────┘"))
	b.WriteString("\n\n")

	// Workers Panel
	b.WriteString(borderRed.Render("┌─ "))
	b.WriteString(headerRed.Render("Worker Pool Status"))
	b.WriteString(borderRed.Render(" ──────────────────────────────────────────────────────────────────────────────────────────────┐"))
	b.WriteString("\n")

	for i := 0; i < numWorkers/2; i++ {
		w1 := &m.Workers[i]
		w2 := &m.Workers[i+numWorkers/2]
		b.WriteString(fmt.Sprintf("W%02d ", w1.ID))
		b.WriteString(renderProgressBar(w1.Progress, 12))
		b.WriteString(fmt.Sprintf(" %s", w1.Status))
		b.WriteString("    ")
		b.WriteString(fmt.Sprintf("W%02d ", w2.ID))
		b.WriteString(renderProgressBar(w2.Progress, 12))
		b.WriteString(fmt.Sprintf(" %s", w2.Status))
		b.WriteString("\n")
	}

	b.WriteString(borderRed.Render("└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘"))
	b.WriteString("\n\n")

	// Log Panel
	b.WriteString(borderWhite.Render("┌─ "))
	b.WriteString(headerWhite.Render("Live System Logs"))
	b.WriteString(borderWhite.Render(" ────────────────────────────────────────────────────────────────────────────────────────────────┐"))
	b.WriteString("\n")

	// Show last N log entries
	start := len(m.LogEntries) - logViewHeight
	if start < 0 {
		start = 0
	}
	for i := start; i < len(m.LogEntries); i++ {
		entry := m.LogEntries[i]
		b.WriteString(dimStyle.Render(entry.Time + " "))
		levelStyle := lipgloss.NewStyle().Foreground(entry.Color)
		if entry.Bold {
			levelStyle = levelStyle.Bold(true)
		}
		b.WriteString(levelStyle.Render(fmt.Sprintf("%-5s ", entry.Level)))
		b.WriteString(blueStyle.Render(fmt.Sprintf("[%-9s] ", entry.Source)))
		b.WriteString(entry.Message)
		b.WriteString("\n")
	}

	b.WriteString(borderWhite.Render("└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘"))
	b.WriteString("\n\n")

	b.WriteString("Press 'q' to quit")

	return b.String()
}

func main() {
	// Parse duration from args
	duration := 10 * time.Second
	if len(os.Args) > 1 {
		if d, err := strconv.Atoi(os.Args[1]); err == nil {
			duration = time.Duration(d) * time.Second
		}
	}

	m := initialModel(duration)
	m.LastFPSCalc = time.Now()

	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Print final stats
	fm := finalModel.(model)
	elapsed := time.Since(fm.StartTime)
	avgFPS := float64(fm.FrameCount) / elapsed.Seconds()
	fmt.Printf("\n=== Bubbletea Benchmark Results ===\n")
	fmt.Printf("Duration: %.2fs\n", elapsed.Seconds())
	fmt.Printf("Total Frames: %d\n", fm.FrameCount)
	fmt.Printf("Average FPS: %.2f\n", avgFPS)
	fmt.Printf("Target FPS: %d\n", targetFPS)
	fmt.Printf("Efficiency: %.1f%%\n", (avgFPS/float64(targetFPS))*100)
}
