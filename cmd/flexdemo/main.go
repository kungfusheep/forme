package main

import (
	"fmt"
	"os"
	"time"

	"riffkey"
	"tui"
)

// Demo showing flex layout using the standard App pattern.
// The UI is compiled once, pointer bindings update dynamically.

func main() {
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
		status:   "TICK: 0000",
		clock:    time.Now().Format("15:04:05Z"),
	}

	// Precomputed display strings (pointer bindings)
	state.fuelABar = tui.Bar(state.fuelA/10, 10)
	state.fuelBBar = tui.Bar(state.fuelB/10, 10)
	state.fuelCBar = tui.Bar(state.fuelC/10, 10)
	state.fuelAText = fmt.Sprintf(" %3d%%", state.fuelA)
	state.fuelBText = fmt.Sprintf(" %3d%%", state.fuelB)
	state.fuelCText = fmt.Sprintf(" %3d%%", state.fuelC)
	state.fuelWarning = ""
	state.rwrIndicator = tui.LEDsBracket(state.rwr >= 1, state.rwr >= 2, state.rwr >= 3, state.rwr >= 4)

	// Build UI with pointer bindings - compiled once!
	ui := buildUI(state)

	app, err := tui.NewApp()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	app.SetView(ui)
	app.Handle("q", func(_ riffkey.Match) {
		app.Stop()
	})

	// Update state periodically
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				state.tick++
				state.status = fmt.Sprintf("TICK: %04d", state.tick)
				state.clock = time.Now().Format("15:04:05Z")

				if state.tick%10 == 0 {
					state.fuelC--
					if state.fuelC < 0 {
						state.fuelC = 100
					}
					state.fuelCBar = tui.Bar(state.fuelC/10, 10)
					state.fuelCText = fmt.Sprintf(" %3d%%", state.fuelC)
					if state.fuelC < 50 {
						state.fuelWarning = "*** LOW FUEL WARNING ***"
					} else {
						state.fuelWarning = ""
					}
				}
				if state.tick%7 == 0 {
					state.rwr = (state.rwr + 1) % 5
					state.rwrIndicator = tui.LEDsBracket(state.rwr >= 1, state.rwr >= 2, state.rwr >= 3, state.rwr >= 4)
				}

				app.RequestRender()
			}
		}
	}()

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildUI(state *State) any {
	dim := tui.RGB(0, 100, 0)

	return tui.VBox{
		Children: []any{
			// TOP ROW: SYSTEM STATUS + ELEC SUBSYS
			tui.HBox{
				Children: []any{
					tui.VBox{
						Title: "SYSTEM STATUS",
						Children: []any{
							tui.Text{Content: tui.LeaderStr("RAM 00064K FRAM-HRC", "PASS", 32)},
							tui.Text{Content: tui.LeaderStr("MPS 00016K RMU-INIT", "PASS", 32)},
							tui.Text{Content: tui.LeaderStr("ECC 00004K FRAM-ERR", "PASS", 32)},
							tui.Text{Content: tui.LeaderStr("I/O CTRL 8251A", "READY", 32)},
							tui.Text{Content: tui.LeaderStr("NVRAM BATTERY 3.2V", "OK", 32)},
							tui.Text{Content: tui.LeaderStr("CRYPTO KG-84A KEY", "LOADED", 32)},
						},
					}.WidthPct(0.5).Height(10).Border(tui.BorderSingle).BorderFG(dim),

					tui.VBox{
						Title: "ELEC SUBSYS",
						Children: []any{
							tui.HBox{Children: []any{
								tui.Text{Content: "GEN1 "},
								tui.Text{Content: tui.Meter(142, 200, 12)},
								tui.Text{Content: " 142A"},
							}},
							tui.HBox{Children: []any{
								tui.Text{Content: "GEN2 "},
								tui.Text{Content: tui.Meter(138, 200, 12)},
								tui.Text{Content: " 138A"},
							}},
							tui.HBox{Children: []any{
								tui.Text{Content: "BATT "},
								tui.Text{Content: tui.Meter(92, 100, 12)},
								tui.Text{Content: " 24.8V"},
							}},
							tui.Text{Content: ""},
							tui.Text{Content: "LOAD: NOMINAL 280A"},
							tui.Text{Content: "INV-A 115VAC 400HZ OK"},
						},
					}.WidthPct(0.5).Height(10).Border(tui.BorderSingle).BorderFG(dim),
				},
			},

			// MIDDLE ROW: FUEL STATUS + SUBSYSTEMS
			tui.HBox{
				Children: []any{
					tui.VBox{
						Title: "FUEL STATUS",
						Children: []any{
							tui.HBox{Children: []any{
								tui.Text{Content: "RES A "},
								tui.Text{Content: &state.fuelABar},
								tui.Text{Content: &state.fuelAText},
							}},
							tui.HBox{Children: []any{
								tui.Text{Content: "RES B "},
								tui.Text{Content: &state.fuelBBar},
								tui.Text{Content: &state.fuelBText},
							}},
							tui.HBox{Children: []any{
								tui.Text{Content: "RES C "},
								tui.Text{Content: &state.fuelCBar},
								tui.Text{Content: &state.fuelCText},
							}},
							tui.Text{Content: &state.fuelWarning, Style: tui.Style{Attr: tui.AttrBold}},
						},
					}.WidthPct(0.5).Height(6).Border(tui.BorderSingle).BorderFG(dim),

					tui.VBox{
						Title: "SUBSYSTEMS",
						Children: []any{
							tui.HBox{Children: []any{
								tui.Text{Content: tui.LED(true)},
								tui.Text{Content: " GEN1   "},
								tui.Text{Content: tui.LED(true)},
								tui.Text{Content: " GEN2"},
							}},
							tui.HBox{Children: []any{
								tui.Text{Content: tui.LED(false)},
								tui.Text{Content: " BACKUP "},
								tui.Text{Content: tui.LED(true)},
								tui.Text{Content: " COMM"},
							}},
							tui.HBox{Children: []any{
								tui.Text{Content: tui.LED(true)},
								tui.Text{Content: " RADAR  "},
								tui.Text{Content: tui.LED(false)},
								tui.Text{Content: " WPNS"},
							}},
							tui.HBox{Children: []any{
								tui.Text{Content: "RWR: "},
								tui.Text{Content: &state.rwrIndicator},
							}},
						},
					}.WidthPct(0.5).Height(6).Border(tui.BorderSingle).BorderFG(dim),
				},
			},

			// BOTTOM: LOG (fills remaining space)
			tui.VBox{
				Title: "LOG",
				Children: []any{
					tui.Text{Content: "21:14:32Z TACAN 22.1 ACQUIRED"},
					tui.Text{Content: "21:14:35Z RAD CH9 482.160 TX 15.2W"},
					tui.Text{Content: "21:14:37Z UHF 243.0 GRD ACTIVE"},
					tui.Text{Content: "21:14:38Z TADIL BUS A ONLINE"},
					tui.Text{Content: "21:14:40Z ENCR KEY 07A 1.2 SYNC"},
					tui.Text{Content: "21:14:42Z ESM RWR STANDBY"},
				},
			}.Grow(1).Border(tui.BorderSingle).BorderFG(dim),

			// Status bar
			tui.HBox{
				Children: []any{
					tui.Text{Content: &state.status},
					tui.Text{Content: " | [Q]UIT | "},
					tui.Text{Content: &state.clock},
				},
			},
		},
	}
}

type State struct {
	fuelA, fuelB, fuelC int
	gen1, gen2, backup  bool
	commLink, radar     bool
	weapons             bool
	rwr                 int
	tick                int

	// Display strings (bound via pointers)
	status       string
	clock        string
	fuelABar     string
	fuelBBar     string
	fuelCBar     string
	fuelAText    string
	fuelBText    string
	fuelCText    string
	fuelWarning  string
	rwrIndicator string
}
