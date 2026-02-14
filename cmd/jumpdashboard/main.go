// jumpdashboard: Jump-mode dashboard with reactive ForEach, Switch, and If
// Single SetView call — no rebuild on interaction, all state is reactive.
package main

import (
	"fmt"
	"log"

	"github.com/kungfusheep/riffkey"
	. "github.com/kungfusheep/forme"
)

type menuItem struct {
	Label  string
	Action string
}

type quickAction struct {
	Label string
	Icon  string
}

type subsystem struct {
	Name   string
	Status string
	Color  Color
}

func main() {
	activePanel := "dashboard"
	selectedTab := 0
	status := "Press 'g' for jump mode | q to quit"

	menuItems := []menuItem{
		{"Dashboard", "dashboard"},
		{"Analytics", "analytics"},
		{"Settings", "settings"},
		{"Users", "users"},
		{"Reports", "reports"},
	}

	quickActions := []quickAction{
		{"New Report", "+"},
		{"Export", "↓"},
		{"Refresh", "⟳"},
		{"Filter", "⚙"},
	}

	subsystems := []subsystem{
		{"API", "Online", Green},
		{"Database", "Online", Green},
		{"Cache", "Warning", Yellow},
		{"Queue", "Online", Green},
		{"Storage", "Online", Green},
		{"Auth", "Online", Green},
	}

	tabs := []string{"Overview", "Metrics", "Logs", "Alerts"}

	// pre-build sidebar menu with reactive If for active highlight
	menuChildren := []any{Text("MENU").FG(Cyan).Bold(), SpaceH(1)}
	for i, item := range menuItems {
		idx := i
		menuChildren = append(menuChildren, Jump(
			If(&activePanel).Eq(item.Action).Then(
				Text("> "+item.Label).FG(Cyan).Bold(),
			).Else(
				Text("  "+item.Label).FG(White),
			),
			func() {
				activePanel = menuItems[idx].Action
				status = fmt.Sprintf("Switched to: %s", menuItems[idx].Label)
			},
		))
	}

	// pre-build quick actions
	actionChildren := []any{}
	for i, action := range quickActions {
		idx := i
		if i > 0 {
			actionChildren = append(actionChildren, Text("  "))
		}
		actionChildren = append(actionChildren, Jump(
			Text(fmt.Sprintf("[%s %s]", action.Icon, action.Label)).FG(BrightWhite),
			func() {
				status = fmt.Sprintf("Action: %s", quickActions[idx].Label)
			},
		))
	}

	// pre-build subsystem grid
	subsystemChildren := []any{Text("SUBSYSTEMS").FG(Cyan).Bold(), SpaceH(1)}
	for i, sys := range subsystems {
		idx := i
		subsystemChildren = append(subsystemChildren, Jump(
			HBox(
				Text("● ").FG(sys.Color),
				Text(fmt.Sprintf("%-10s", sys.Name)).FG(White),
				Text(sys.Status).FG(sys.Color),
			),
			func() {
				status = fmt.Sprintf("Subsystem: %s (%s)", subsystems[idx].Name, subsystems[idx].Status)
			},
		))
	}

	// pre-build tab row with reactive If for selected highlight
	tabChildren := []any{}
	for i, label := range tabs {
		idx := i
		tabChildren = append(tabChildren, Jump(
			If(&selectedTab).Eq(i).Then(
				Text(fmt.Sprintf("[%s]", label)).FG(Cyan).Bold(),
			).Else(
				Text(fmt.Sprintf(" %s ", label)).FG(BrightBlack),
			),
			func() {
				selectedTab = idx
				status = fmt.Sprintf("Tab: %s", tabs[idx])
			},
		))
	}

	// helper: jumpable card
	jumpCard := func(title, value, unit string, color Color) JumpC {
		return Jump(
			VBox.Border(BorderSingle)(
				Text(title).FG(BrightBlack),
				HBox(Text(value).FG(color).Bold(), Text(" "+unit).FG(BrightBlack)),
			),
			func() { status = fmt.Sprintf("Card: %s = %s %s", title, value, unit) },
		)
	}

	// helper: jumpable log/activity/alert entries
	jumpRow := func(label string, content any) JumpC {
		return Jump(content, func() { status = label })
	}

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(VBox(
		HBox(Text("Dashboard").FG(Cyan).Bold(), Space(), Text(&status).FG(Yellow)),
		HRule().FG(BrightBlack),
		SpaceH(1),

		HBox.Gap(2)(
			VBox.WidthPct(0.15)(menuChildren...),
			VRule().FG(BrightBlack),
			VBox(
				HBox(actionChildren...),
				SpaceH(1),
				HBox(tabChildren...),
				HRule().FG(BrightBlack),
				SpaceH(1),

				// tab content — reactive Switch, compiled once
				Switch(&selectedTab).
					Case(0, VBox(
						Text("System Overview").FG(White).Bold(),
						SpaceH(1),
						HBox.Gap(2)(
							jumpCard("Requests", "1.2M", "/day", Cyan),
							jumpCard("Errors", "0.02%", "rate", Green),
							jumpCard("Latency", "45ms", "p99", Yellow),
							jumpCard("Users", "8,432", "active", Magenta),
						),
						SpaceH(1),
						Text("Recent Activity").FG(White).Bold(),
						SpaceH(1),
						jumpRow("User login - john@example.com", HBox(Text("User login  ").FG(Green), Text("john@example.com    ").FG(White), Text("2m ago").FG(BrightBlack))),
						jumpRow("API call - GET /users", HBox(Text("API call    ").FG(Cyan), Text("GET /users          ").FG(White), Text("5m ago").FG(BrightBlack))),
						jumpRow("Cache miss - session:abc123", HBox(Text("Cache miss  ").FG(Yellow), Text("session:abc123      ").FG(White), Text("8m ago").FG(BrightBlack))),
					)).
					Case(1, VBox(
						Text("Performance Metrics").FG(White).Bold(),
						SpaceH(1),
						HBox.Gap(2)(
							jumpCard("CPU", "42%", "avg", Cyan),
							jumpCard("Memory", "2.1GB", "used", Yellow),
							jumpCard("Disk I/O", "120MB/s", "read", Green),
							jumpCard("Network", "450Mbps", "in", Magenta),
						),
						SpaceH(1),
						Text("Throughput (last hour)").FG(BrightBlack),
						Sparkline([]float64{10, 25, 40, 35, 50, 45, 60, 55, 70, 65, 80, 75}).FG(Cyan),
					)).
					Case(2, VBox(
						Text("Recent Logs").FG(White).Bold(),
						SpaceH(1),
						jumpRow("[INFO ] Server started on port 8080", HBox(Text("[INFO ]").FG(Green).Bold(), Text(" Server started on port 8080").FG(White))),
						jumpRow("[DEBUG] Processing request /api/users", HBox(Text("[DEBUG]").FG(Cyan).Bold(), Text(" Processing request /api/users").FG(White))),
						jumpRow("[WARN ] Cache nearing capacity (85%)", HBox(Text("[WARN ]").FG(Yellow).Bold(), Text(" Cache nearing capacity (85%)").FG(White))),
						jumpRow("[INFO ] Database connection established", HBox(Text("[INFO ]").FG(Green).Bold(), Text(" Database connection established").FG(White))),
						jumpRow("[ERROR] Failed to connect to Redis", HBox(Text("[ERROR]").FG(Red).Bold(), Text(" Failed to connect to Redis").FG(White))),
						jumpRow("[INFO ] Retry successful, Redis connected", HBox(Text("[INFO ]").FG(Green).Bold(), Text(" Retry successful, Redis connected").FG(White))),
					)).
					Case(3, VBox(
						Text("Active Alerts").FG(White).Bold(),
						SpaceH(1),
						jumpRow("[CRITICAL] Database replica lag > 30s", HBox(Text("◆ ").FG(Red), Text("CRITICAL  ").FG(Red).Bold(), Text("Database replica lag > 30s").FG(White))),
						jumpRow("[WARNING] Memory usage above 80%", HBox(Text("● ").FG(Yellow), Text("WARNING   ").FG(Yellow).Bold(), Text("Memory usage above 80%").FG(White))),
						jumpRow("[WARNING] SSL certificate expires in 7 days", HBox(Text("● ").FG(Yellow), Text("WARNING   ").FG(Yellow).Bold(), Text("SSL certificate expires in 7 days").FG(White))),
						SpaceH(1),
						Text("Resolved (last 24h)").FG(BrightBlack),
						SpaceH(1),
						jumpRow("[OK] API latency normalized", HBox(Text("✓ ").FG(Green), Text("OK        ").FG(Green).Bold(), Text("API latency normalized").FG(White))),
						jumpRow("[OK] Disk space freed", HBox(Text("✓ ").FG(Green), Text("OK        ").FG(Green).Bold(), Text("Disk space freed").FG(White))),
					)).
					Default(Text("Unknown tab").FG(Red)),
			),
			VRule().FG(BrightBlack),
			VBox.WidthPct(0.20)(subsystemChildren...),
		),

		SpaceH(1),
		HRule().FG(BrightBlack),
		Text("g:jump | tab:cycle tabs | q:quit").FG(BrightBlack),
	))

	app.JumpKey("g").
		Handle("q", func(_ riffkey.Match) { app.Stop() }).
		Handle("tab", func(_ riffkey.Match) {
			selectedTab = (selectedTab + 1) % len(tabs)
			status = fmt.Sprintf("Tab: %s", tabs[selectedTab])
		})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
