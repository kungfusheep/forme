package main

import (
	"fmt"
	"os"
	"time"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"
)

// demo showing flex layout with dynamic pointer bindings

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

	state.fuelABar = Bar(state.fuelA/10, 10)
	state.fuelBBar = Bar(state.fuelB/10, 10)
	state.fuelCBar = Bar(state.fuelC/10, 10)
	state.fuelAText = fmt.Sprintf(" %3d%%", state.fuelA)
	state.fuelBText = fmt.Sprintf(" %3d%%", state.fuelB)
	state.fuelCText = fmt.Sprintf(" %3d%%", state.fuelC)
	state.fuelWarning = ""
	state.rwrIndicator = LEDsBracket(state.rwr >= 1, state.rwr >= 2, state.rwr >= 3, state.rwr >= 4)

	ui := buildUI(state)

	app, err := NewApp()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	app.SetView(ui)
	app.Handle("q", func(_ riffkey.Match) {
		app.Stop()
	})

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			state.tick++
			state.status = fmt.Sprintf("TICK: %04d", state.tick)
			state.clock = time.Now().Format("15:04:05Z")

			if state.tick%10 == 0 {
				state.fuelC--
				if state.fuelC < 0 {
					state.fuelC = 100
				}
				state.fuelCBar = Bar(state.fuelC/10, 10)
				state.fuelCText = fmt.Sprintf(" %3d%%", state.fuelC)
				if state.fuelC < 50 {
					state.fuelWarning = "*** LOW FUEL WARNING ***"
				} else {
					state.fuelWarning = ""
				}
			}
			if state.tick%7 == 0 {
				state.rwr = (state.rwr + 1) % 5
				state.rwrIndicator = LEDsBracket(state.rwr >= 1, state.rwr >= 2, state.rwr >= 3, state.rwr >= 4)
			}

			app.RequestRender()
		}
	}()

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildUI(state *State) any {
	dim := RGB(0, 100, 0)

	return VBox(
		// top row: system status + elec subsys
		HBox(
			VBox.WidthPct(0.5).Height(10).Border(BorderSingle).BorderFG(dim).Title("SYSTEM STATUS")(
				Text(LeaderStr("RAM 00064K FRAM-HRC", "PASS", 32)),
				Text(LeaderStr("MPS 00016K RMU-INIT", "PASS", 32)),
				Text(LeaderStr("ECC 00004K FRAM-ERR", "PASS", 32)),
				Text(LeaderStr("I/O CTRL 8251A", "READY", 32)),
				Text(LeaderStr("NVRAM BATTERY 3.2V", "OK", 32)),
				Text(LeaderStr("CRYPTO KG-84A KEY", "LOADED", 32)),
			),

			VBox.WidthPct(0.5).Height(10).Border(BorderSingle).BorderFG(dim).Title("ELEC SUBSYS")(
				HBox(Text("GEN1 "), Text(Meter(142, 200, 12)), Text(" 142A")),
				HBox(Text("GEN2 "), Text(Meter(138, 200, 12)), Text(" 138A")),
				HBox(Text("BATT "), Text(Meter(92, 100, 12)), Text(" 24.8V")),
				Text("LOAD: NOMINAL 280A"),
				Text("INV-A 115VAC 400HZ OK"),
			),
		),

		// middle row: fuel status + subsystems
		HBox(
			VBox.WidthPct(0.5).Height(6).Border(BorderSingle).BorderFG(dim).Title("FUEL STATUS")(
				HBox(Text("RES A "), Text(&state.fuelABar), Text(&state.fuelAText)),
				HBox(Text("RES B "), Text(&state.fuelBBar), Text(&state.fuelBText)),
				HBox(Text("RES C "), Text(&state.fuelCBar), Text(&state.fuelCText)),
				Text(&state.fuelWarning).Bold(),
			),

			VBox.WidthPct(0.5).Height(6).Border(BorderSingle).BorderFG(dim).Title("SUBSYSTEMS")(
				HBox(Text(LED(true)), Text(" GEN1   "), Text(LED(true)), Text(" GEN2")),
				HBox(Text(LED(false)), Text(" BACKUP "), Text(LED(true)), Text(" COMM")),
				HBox(Text(LED(true)), Text(" RADAR  "), Text(LED(false)), Text(" WPNS")),
				HBox(Text("RWR: "), Text(&state.rwrIndicator)),
			),
		),

		// bottom: log (fills remaining space)
		VBox.Grow(1).Border(BorderSingle).BorderFG(dim).Title("LOG")(
			Text("21:14:32Z TACAN 22.1 ACQUIRED"),
			Text("21:14:35Z RAD CH9 482.160 TX 15.2W"),
			Text("21:14:37Z UHF 243.0 GRD ACTIVE"),
			Text("21:14:38Z TADIL BUS A ONLINE"),
			Text("21:14:40Z ENCR KEY 07A 1.2 SYNC"),
			Text("21:14:42Z ESM RWR STANDBY"),
		),

		// status bar
		HBox(
			Text(&state.status),
			Text(" | [Q]UIT | "),
			Text(&state.clock),
		),
	)
}

type State struct {
	fuelA, fuelB, fuelC int
	gen1, gen2, backup  bool
	commLink, radar     bool
	weapons             bool
	rwr                 int
	tick                int

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
