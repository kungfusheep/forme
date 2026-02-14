// forme-htop: System monitor demo using real-time reactive bindings
package main

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	. "github.com/kungfusheep/forme"
	"github.com/kungfusheep/riffkey"
)

type Process struct {
	PID     int
	Command string
	CPU     float64
	Mem     float64
	Display string // formatted display string
}

type State struct {
	Processes   []Process
	SelectedIdx int
	SortBy      string // "cpu" or "mem"
	CPUPercent  string
	MemPercent  string
	MemUsed     string
	MemTotal    string
	Uptime      string
	StatusLine  string
}

func main() {
	state := &State{
		SortBy:     "cpu",
		StatusLine: "↑/↓ navigate, c=sort CPU, m=sort Mem, q=quit",
	}
	refreshData(state)

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(
		VBox(
			Text("System Monitor").Bold(),
			Text(""),
			HBox(Text("CPU: "), Text(&state.CPUPercent), Text("%")),
			HBox(Text("Mem: "), Text(&state.MemUsed), Text(" / "), Text(&state.MemTotal), Text(" ("), Text(&state.MemPercent), Text("%)")),
			HBox(Text("Uptime: "), Text(&state.Uptime)),
			Text(""),
			Text("   PID    CPU%   MEM%  COMMAND"),
			Text("  ─────  ─────  ─────  ────────────────────"),
			List(&state.Processes).
				Selection(&state.SelectedIdx).
				MaxVisible(15).
				Render(func(p *Process) any {
					return Text(&p.Display)
				}).
				BindNav("j", "k").
				BindPageNav("<PageDown>", "<PageUp>").
				BindFirstLast("<Home>", "<End>"),
			Text(""),
			Text(&state.StatusLine),
		),
	)

	// Start refresh ticker
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for range ticker.C {
			refreshData(state)
			app.RequestRender()
		}
	}()

	app.Handle("c", func(_ riffkey.Match) {
		state.SortBy = "cpu"
		sortProcesses(state)
	})

	app.Handle("m", func(_ riffkey.Match) {
		state.SortBy = "mem"
		sortProcesses(state)
	})

	app.Handle("q", func(_ riffkey.Match) {
		ticker.Stop()
		app.Stop()
	})

	app.Handle("<Esc>", func(_ riffkey.Match) {
		ticker.Stop()
		app.Stop()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func refreshData(state *State) {
	// Get processes
	state.Processes = getProcesses()
	sortProcesses(state)

	// Update display string for each process
	for i := range state.Processes {
		p := &state.Processes[i]
		p.Display = fmt.Sprintf("%5d  %5.1f%%  %5.1f%%  %-20s",
			p.PID, p.CPU, p.Mem, truncate(p.Command, 20))
	}

	// Get system stats
	cpuPct, memPct, memUsed, memTotal := getSystemStats()
	state.CPUPercent = fmt.Sprintf("%d", cpuPct)
	state.MemPercent = fmt.Sprintf("%d", memPct)
	state.MemUsed = memUsed
	state.MemTotal = memTotal
	state.Uptime = getUptime()
}

func getProcesses() []Process {
	var processes []Process

	// Use ps to get process info (works on macOS and Linux)
	cmd := exec.Command("ps", "-eo", "pid,pcpu,pmem,comm")
	output, err := cmd.Output()
	if err != nil {
		return processes
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	scanner.Scan() // Skip header

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		cpu, _ := strconv.ParseFloat(fields[1], 64)
		mem, _ := strconv.ParseFloat(fields[2], 64)
		command := fields[3]
		// Get just the command name, not full path
		if idx := strings.LastIndex(command, "/"); idx >= 0 {
			command = command[idx+1:]
		}

		processes = append(processes, Process{
			PID:     pid,
			Command: command,
			CPU:     cpu,
			Mem:     mem,
		})
	}

	return processes
}

func sortProcesses(state *State) {
	switch state.SortBy {
	case "cpu":
		sort.Slice(state.Processes, func(i, j int) bool {
			return state.Processes[i].CPU > state.Processes[j].CPU
		})
	case "mem":
		sort.Slice(state.Processes, func(i, j int) bool {
			return state.Processes[i].Mem > state.Processes[j].Mem
		})
	}
}

func getSystemStats() (cpuPct, memPct int, memUsed, memTotal string) {
	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// For demo purposes, use runtime stats
	// Real htop would use system calls
	if runtime.GOOS == "darwin" {
		// macOS: Use vm_stat
		cmd := exec.Command("vm_stat")
		output, err := cmd.Output()
		if err == nil {
			memPct, memUsed, memTotal = parseVMStat(string(output))
		}

		// CPU: Parse top output briefly
		cmd = exec.Command("top", "-l", "1", "-n", "0")
		output, err = cmd.Output()
		if err == nil {
			cpuPct = parseCPUFromTop(string(output))
		}
	}

	return
}

func parseVMStat(output string) (pct int, used, total string) {
	var pagesFree, pagesActive, pagesInactive, pagesSpeculative, pagesWired uint64
	pageSize := uint64(4096)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Pages free:") {
			pagesFree = parseVMStatValue(line)
		} else if strings.HasPrefix(line, "Pages active:") {
			pagesActive = parseVMStatValue(line)
		} else if strings.HasPrefix(line, "Pages inactive:") {
			pagesInactive = parseVMStatValue(line)
		} else if strings.HasPrefix(line, "Pages speculative:") {
			pagesSpeculative = parseVMStatValue(line)
		} else if strings.HasPrefix(line, "Pages wired down:") {
			pagesWired = parseVMStatValue(line)
		}
	}

	totalPages := pagesFree + pagesActive + pagesInactive + pagesSpeculative + pagesWired
	usedPages := pagesActive + pagesWired

	if totalPages > 0 {
		pct = int(usedPages * 100 / totalPages)
	}

	totalBytes := totalPages * pageSize
	usedBytes := usedPages * pageSize

	used = formatBytes(usedBytes)
	total = formatBytes(totalBytes)

	return
}

func parseVMStatValue(line string) uint64 {
	// "Pages free:                              123456."
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return 0
	}
	valStr := strings.TrimSpace(strings.TrimSuffix(parts[1], "."))
	val, _ := strconv.ParseUint(valStr, 10, 64)
	return val
}

func parseCPUFromTop(output string) int {
	// Look for "CPU usage:" line
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "CPU usage:") {
			// "CPU usage: 12.34% user, 5.67% sys, 81.99% idle"
			parts := strings.Split(line, ",")
			for _, part := range parts {
				if strings.Contains(part, "idle") {
					// Extract idle percentage
					fields := strings.Fields(part)
					if len(fields) >= 1 {
						idleStr := strings.TrimSuffix(fields[0], "%")
						idle, _ := strconv.ParseFloat(idleStr, 64)
						return int(100 - idle)
					}
				}
			}
		}
	}
	return 0
}

func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/MB)
	default:
		return fmt.Sprintf("%.1fK", float64(bytes)/KB)
	}
}

func getUptime() string {
	cmd := exec.Command("uptime")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	// Extract just the uptime part
	s := strings.TrimSpace(string(output))
	if idx := strings.Index(s, "up "); idx >= 0 {
		s = s[idx+3:]
		if idx := strings.Index(s, ","); idx >= 0 {
			// Find second comma (after user count)
			rest := s[idx+1:]
			if idx2 := strings.Index(rest, ","); idx2 >= 0 {
				s = s[:idx+1+idx2]
			} else {
				s = s[:idx]
			}
		}
	}
	return strings.TrimSpace(s)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "~"
}
