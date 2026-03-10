package main

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
	"time"

	. "github.com/kungfusheep/glyph"
)

type Proc struct {
	PID     int
	Command string
	CPU     float64
	Mem     float64
}

func main() {
	var (
		procs  []Proc
		cpuPct int
		memPct int
	)

	refresh := func() {
		procs = procs[:0]
		out, _ := exec.Command("ps", "-eo", "pid,pcpu,pmem,comm").Output()
		sc := bufio.NewScanner(strings.NewReader(string(out)))
		sc.Scan()
		for sc.Scan() {
			f := strings.Fields(sc.Text())
			if len(f) < 4 {
				continue
			}
			pid, _ := strconv.Atoi(f[0])
			cpu, _ := strconv.ParseFloat(f[1], 64)
			mem, _ := strconv.ParseFloat(f[2], 64)
			cmd := f[3]
			if idx := strings.LastIndex(cmd, "/"); idx >= 0 {
				cmd = cmd[idx+1:]
			}
			procs = append(procs, Proc{pid, cmd, cpu, mem})
		}

		out2, _ := exec.Command("top", "-l", "1", "-n", "0").Output()
		sc2 := bufio.NewScanner(strings.NewReader(string(out2)))
		for sc2.Scan() {
			line := sc2.Text()
			if strings.Contains(line, "CPU usage:") {
				for _, part := range strings.Split(line, ",") {
					if strings.Contains(part, "idle") {
						idle, _ := strconv.ParseFloat(strings.TrimSuffix(strings.Fields(part)[0], "%"), 64)
						cpuPct = int(100 - idle)
					}
				}
			}
			if strings.Contains(line, "PhysMem:") {
				for _, part := range strings.Split(line, ",") {
					if strings.Contains(part, "used") {
						s := strings.TrimSpace(part)
						s = strings.TrimSuffix(s, " used")
						if strings.HasSuffix(s, "G") {
							val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "G"), 64)
							memPct = int(val / 16 * 100) // rough estimate
						}
					}
				}
			}
		}
	}
	refresh()

	app, _ := NewApp()
	app.JumpKey("g")
	app.SetView(
		VBox(
			HBox.Gap(4)(
				Text("CPU"), Progress(&cpuPct).Width(30),
				Text("Mem"), Progress(&memPct).Width(30),
			),
			AutoTable(&procs).Sortable().Scrollable(20).BindVimNav(),
		),
	)

	go func() {
		for range time.NewTicker(2 * time.Second).C {
			refresh()
			app.RequestRender()
		}
	}()

	app.Handle("q", app.Stop)
	app.Run()
}
