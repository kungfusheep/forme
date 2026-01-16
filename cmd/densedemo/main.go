package main

import (
	"fmt"
	"os"
	"time"

	"riffkey"
	"tui"
)

// Demo showing dense information display patterns.

func main() {
	screen, err := tui.NewScreen(nil)
	if err != nil {
		panic(err)
	}

	state := &State{
		fuelA:    92,
		fuelB:    87,
		fuelC:    34,
		gen1:     true,
		gen2:     true,
		backup:   false,
		commLink: true,
		radar:    true,
		weapons:  false,
		rwr:      3,
		tick:     0,
		running:  true,
	}

	// Input handling
	router := riffkey.NewRouter()
	router.Handle("q", func(_ riffkey.Match) {
		state.running = false
	})
	input := riffkey.NewInput(router)
	reader := riffkey.NewReader(os.Stdin)

	if err := screen.EnterRawMode(); err != nil {
		panic(err)
	}
	defer screen.ExitRawMode()

	// Style - single colour, two tones
	normal := tui.DefaultStyle().Foreground(tui.Green)
	dim := tui.DefaultStyle().Foreground(tui.RGB(0, 100, 0))
	bright := tui.DefaultStyle().Foreground(tui.BrightGreen).Bold()

	render := func() {
		buf := screen.Buffer()
		buf.Clear()
		size := screen.Size()
		w, h := size.Width, size.Height

		// ─────────────────────────────────────────────────────────────
		// SYSTEM STATUS
		// ─────────────────────────────────────────────────────────────
		leftW := w / 2
		if leftW < 35 {
			leftW = 35
		}
		region := buf.DrawPanel(0, 0, leftW, 10, "SYSTEM STATUS", dim)

		items := []struct{ label, value string }{
			{"RAM 00064K FRAM-HRC", "PASS"},
			{"MPS 00016K RMU-INIT", "PASS"},
			{"ECC 00004K FRAM-ERR", "PASS"},
			{"I/O CTRL 8251A", "READY"},
			{"NVRAM BATTERY 3.2V", "OK"},
			{"CRYPTO KG-84A KEY", "LOADED"},
		}
		for y, item := range items {
			region.WriteString(0, y, tui.LeaderStr(item.label, item.value, leftW-4), normal)
		}

		// ─────────────────────────────────────────────────────────────
		// ELECTRICAL
		// ─────────────────────────────────────────────────────────────
		rightX := leftW
		rightW := w - leftW
		if rightW < 25 {
			rightW = 25
		}
		region = buf.DrawPanel(rightX, 0, rightW, 10, "ELEC SUBSYS", dim)

		region.WriteString(0, 0, "GEN1 "+tui.Meter(142, 200, 12)+" 142A", normal)
		region.WriteString(0, 1, "GEN2 "+tui.Meter(138, 200, 12)+" 138A", normal)
		region.WriteString(0, 2, "BATT "+tui.Meter(state.fuelA, 100, 12)+" 24.8V", normal)
		region.WriteString(0, 4, "LOAD: NOMINAL 280A", normal)
		region.WriteString(0, 5, "INV-A 115VAC 400HZ OK", normal)

		// ─────────────────────────────────────────────────────────────
		// FUEL STATUS
		// ─────────────────────────────────────────────────────────────
		region = buf.DrawPanel(0, 10, leftW, 6, "FUEL STATUS", dim)

		region.WriteString(0, 0, fmt.Sprintf("RES A %s %3d%%", tui.Bar(state.fuelA/10, 10), state.fuelA), normal)
		region.WriteString(0, 1, fmt.Sprintf("RES B %s %3d%%", tui.Bar(state.fuelB/10, 10), state.fuelB), normal)
		region.WriteString(0, 2, fmt.Sprintf("RES C %s %3d%%", tui.Bar(state.fuelC/10, 10), state.fuelC), normal)
		if state.fuelC < 50 {
			region.WriteString(0, 3, "*** LOW FUEL WARNING ***", bright)
		}

		// ─────────────────────────────────────────────────────────────
		// SUBSYSTEMS
		// ─────────────────────────────────────────────────────────────
		region = buf.DrawPanel(rightX, 10, rightW, 6, "SUBSYSTEMS", dim)

		region.WriteString(0, 0, tui.LED(state.gen1)+" GEN1   "+tui.LED(state.gen2)+" GEN2", normal)
		region.WriteString(0, 1, tui.LED(state.backup)+" BACKUP "+tui.LED(state.commLink)+" COMM", normal)
		region.WriteString(0, 2, tui.LED(state.radar)+" RADAR  "+tui.LED(state.weapons)+" WPNS", normal)
		region.WriteString(0, 3, "RWR: "+tui.LEDsBracket(state.rwr >= 1, state.rwr >= 2, state.rwr >= 3, state.rwr >= 4), normal)

		// ─────────────────────────────────────────────────────────────
		// LOG
		// ─────────────────────────────────────────────────────────────
		logH := h - 16 - 1
		if logH < 3 {
			logH = 3
		}
		region = buf.DrawPanel(0, 16, w, logH, "LOG", dim)

		logs := []string{
			"21:14:32Z TACAN 22.1 ACQUIRED",
			"21:14:35Z RAD CH9 482.160 TX 15.2W",
			"21:14:37Z UHF 243.0 GRD ACTIVE",
			"21:14:38Z TADIL BUS A ONLINE",
			"21:14:40Z ENCR KEY 07A 1.2 SYNC",
			"21:14:42Z ESM RWR STANDBY",
		}
		for i, log := range logs {
			if i >= logH-2 {
				break
			}
			region.WriteString(0, i, log, normal)
		}

		// ─────────────────────────────────────────────────────────────
		// STATUS BAR
		// ─────────────────────────────────────────────────────────────
		statusY := h - 1
		buf.HLine(0, statusY, w, '─', dim)
		buf.WriteString(1, statusY, fmt.Sprintf(" TICK: %04d | [Q]UIT ", state.tick), normal)
		buf.WriteString(w-10, statusY, time.Now().Format("15:04:05Z"), dim)

		screen.Flush()
	}

	render()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	go func() {
		input.Run(reader, func(_ bool) {
			if state.running {
				render()
			}
		})
	}()

	for state.running {
		<-ticker.C
		state.tick++
		if state.tick%10 == 0 {
			state.fuelC--
			if state.fuelC < 0 {
				state.fuelC = 100
			}
		}
		if state.tick%7 == 0 {
			state.rwr = (state.rwr + 1) % 5
		}
		render()
	}

	os.Stdin.Close()
}

type State struct {
	fuelA, fuelB, fuelC int
	gen1, gen2, backup  bool
	commLink, radar     bool
	weapons             bool
	rwr                 int
	tick                int
	running             bool
}
