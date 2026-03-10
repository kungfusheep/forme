package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/kungfusheep/glyph"
)

type Proc struct {
	PID     int
	Command string
	CPU     float64
	Mem     float64
	CWD     string
}

func main() {
	var (
		procs  []Proc
		status string

		cwdMap map[int]string
		cwdMu  sync.Mutex
	)

	home, _ := os.UserHomeDir()

	shortenPath := func(p string) string {
		if strings.HasPrefix(p, home) {
			return "~" + p[len(home):]
		}
		return p
	}

	refreshCWDs := func() {
		out, err := exec.Command("lsof", "-d", "cwd", "-Fpn").Output()
		if err != nil {
			return
		}
		m := make(map[int]string)
		var pid int
		for _, line := range strings.Split(string(out), "\n") {
			switch {
			case strings.HasPrefix(line, "p"):
				pid, _ = strconv.Atoi(line[1:])
			case strings.HasPrefix(line, "n"):
				m[pid] = line[1:]
			}
		}
		cwdMu.Lock()
		cwdMap = m
		cwdMu.Unlock()
	}

	filter := NewFilter(&procs, func(p *Proc) string {
		return fmt.Sprintf("%d %s %s", p.PID, p.Command, p.CWD)
	})

	refreshProcs := func() {
		procs = procs[:0]
		out, _ := exec.Command("ps", "-eo", "pid,pcpu,pmem,comm").Output()
		sc := bufio.NewScanner(strings.NewReader(string(out)))
		sc.Scan()

		cwdMu.Lock()
		cmap := cwdMap
		cwdMu.Unlock()

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

			cwd := ""
			if cmap != nil {
				if c, ok := cmap[pid]; ok {
					cwd = shortenPath(c)
				}
			}

			procs = append(procs, Proc{pid, cmd, cpu, mem, cwd})
		}

		status = fmt.Sprintf("%d processes", len(procs))

		// re-apply filter to fresh data
		q := filter.Query()
		filter.Reset()
		if q != "" {
			filter.Update(q)
		}
	}

	// initial load
	refreshCWDs()
	refreshProcs()

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	input := Input().Placeholder("filter processes...").Bind()

	app.JumpKey("<C-f>")
	app.SetView(
		VBox(
			HBox.Gap(2)(
				Text("glyph-top").Bold().FG(Cyan),
				Space(),
				Text(&status).FG(BrightBlack),
			),
			HBox(Text("> ").Bold(), input),
			AutoTable(&filter.Items).
				Column("CPU", Number(1)).
				Column("Mem", Number(1)).
				SortBy("CPU", false).
				Scrollable(30).
				BindNav("<C-n>", "<C-p>").
				BindPageNav("<C-d>", "<C-u>"),
			Text("type to filter  ctrl+n/p: navigate  esc: quit").FG(BrightBlack),
		),
	)

	app.Handle("<Esc>", func() { app.Stop() })

	// CWD refresh (slow, background, every 10s)
	go func() {
		for range time.NewTicker(10 * time.Second).C {
			refreshCWDs()
		}
	}()

	// process refresh + filter sync, every 2s
	go func() {
		for range time.NewTicker(2 * time.Second).C {
			filter.Update(input.Value())
			refreshProcs()
			app.RequestRender()
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
