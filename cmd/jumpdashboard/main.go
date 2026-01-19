package main

import (
	"fmt"
	"log"

	"riffkey"
	"tui"
)

// State - using structs that can be referenced
var (
	activePanel = "dashboard"
	selectedTab = 0
	status      = "Press 'g' for jump mode | q to quit"
)

// Menu items
var menuItems = []struct {
	label  string
	action string
}{
	{"Dashboard", "dashboard"},
	{"Analytics", "analytics"},
	{"Settings", "settings"},
	{"Users", "users"},
	{"Reports", "reports"},
}

// Quick actions
var quickActions = []struct {
	label string
	icon  string
}{
	{"New Report", "+"},
	{"Export", "↓"},
	{"Refresh", "⟳"},
	{"Filter", "⚙"},
}

// Subsystem statuses
var subsystems = []struct {
	name   string
	status string
	color  tui.Color
}{
	{"API", "Online", tui.Green},
	{"Database", "Online", tui.Green},
	{"Cache", "Warning", tui.Yellow},
	{"Queue", "Online", tui.Green},
	{"Storage", "Online", tui.Green},
	{"Auth", "Online", tui.Green},
}

var tabs = []string{"Overview", "Metrics", "Logs", "Alerts"}

var app *tui.App

func main() {
	var err error
	app, err = tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Build initial view
	rebuildView()

	// Set up keybindings
	app.JumpKey("g").
		Handle("q", func(_ riffkey.Match) { app.Stop() }).
		Handle("tab", func(_ riffkey.Match) {
			selectedTab = (selectedTab + 1) % len(tabs)
			status = fmt.Sprintf("Tab: %s", tabs[selectedTab])
			rebuildView()
		})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// rebuildView reconstructs the entire view with current state
func rebuildView() {
	// Build sidebar menu with jump targets
	menuChildren := []any{
		tui.Text{Content: "MENU", Style: tui.Style{FG: tui.Cyan, Attr: tui.AttrBold}},
		tui.Spacer{Height: 1},
	}
	for i, item := range menuItems {
		idx := i
		label := item.label
		style := tui.Style{FG: tui.White}
		if item.action == activePanel {
			style = tui.Style{FG: tui.Cyan, Attr: tui.AttrBold}
			label = "> " + label
		} else {
			label = "  " + label
		}
		menuChildren = append(menuChildren, tui.Jump{
			Child: tui.Text{Content: label, Style: style},
			OnSelect: func() {
				activePanel = menuItems[idx].action
				status = fmt.Sprintf("Switched to: %s", menuItems[idx].label)
				rebuildView()
			},
		})
	}

	// Build quick actions row with jump targets
	actionChildren := []any{}
	for i, action := range quickActions {
		idx := i
		actionChildren = append(actionChildren, tui.Jump{
			Child: tui.Row{
				Children: []any{
					tui.Text{Content: fmt.Sprintf("[%s %s]", action.icon, action.label), Style: tui.Style{FG: tui.BrightWhite}},
				},
			},
			OnSelect: func() {
				status = fmt.Sprintf("Action: %s", quickActions[idx].label)
				rebuildView()
			},
		})
		if i < len(quickActions)-1 {
			actionChildren = append(actionChildren, tui.Text{Content: "  "})
		}
	}

	// Build subsystem grid with jump targets
	subsystemChildren := []any{
		tui.Text{Content: "SUBSYSTEMS", Style: tui.Style{FG: tui.Cyan, Attr: tui.AttrBold}},
		tui.Spacer{Height: 1},
	}
	for i, sys := range subsystems {
		idx := i
		indicator := "●"
		subsystemChildren = append(subsystemChildren, tui.Jump{
			Child: tui.Row{
				Children: []any{
					tui.Text{Content: indicator + " ", Style: tui.Style{FG: sys.color}},
					tui.Text{Content: fmt.Sprintf("%-10s", sys.name), Style: tui.Style{FG: tui.White}},
					tui.Text{Content: sys.status, Style: tui.Style{FG: sys.color}},
				},
			},
			OnSelect: func() {
				status = fmt.Sprintf("Subsystem details: %s (%s)", subsystems[idx].name, subsystems[idx].status)
				rebuildView()
			},
		})
	}

	// Build tab row
	tabChildren := []any{}
	for i, label := range tabs {
		idx := i
		style := tui.Style{FG: tui.BrightBlack}
		content := fmt.Sprintf(" %s ", label)
		if i == selectedTab {
			style = tui.Style{FG: tui.Cyan, Attr: tui.AttrBold}
			content = fmt.Sprintf("[%s]", label)
		}
		tabChildren = append(tabChildren, tui.Jump{
			Child: tui.Text{Content: content, Style: style},
			OnSelect: func() {
				selectedTab = idx
				status = fmt.Sprintf("Tab: %s", tabs[idx])
				rebuildView()
			},
		})
	}

	app.SetView(
		tui.Col{
			Children: []any{
				// Header
				tui.Row{
					Children: []any{
						tui.Text{Content: "Dashboard", Style: tui.Style{FG: tui.Cyan, Attr: tui.AttrBold}},
						tui.Spacer{},
						tui.Text{Content: status, Style: tui.Style{FG: tui.Yellow}},
					},
				},
				tui.HRule{Style: tui.Style{FG: tui.BrightBlack}},
				tui.Spacer{Height: 1},

				// Main layout: sidebar + content
				tui.Row{
					Gap: 2,
					Children: []any{
						// Sidebar
						tui.Col{
							Children: menuChildren,
						}.WidthPct(0.15),

						tui.VRule{Style: tui.Style{FG: tui.BrightBlack}},

						// Main content area
						tui.Col{
							Children: []any{
								// Quick actions bar
								tui.Row{Children: actionChildren},
								tui.Spacer{Height: 1},

								// Tabs
								tui.Row{Children: tabChildren},
								tui.HRule{Style: tui.Style{FG: tui.BrightBlack}},
								tui.Spacer{Height: 1},

								// Content based on selected tab
								buildTabContent(selectedTab),
							},
						},

						tui.VRule{Style: tui.Style{FG: tui.BrightBlack}},

						// Right sidebar - subsystems
						tui.Col{
							Children: subsystemChildren,
						}.WidthPct(0.20),
					},
				},

				// Footer
				tui.Spacer{Height: 1},
				tui.HRule{Style: tui.Style{FG: tui.BrightBlack}},
				tui.Text{Content: "g:jump | tab:cycle tabs | q:quit", Style: tui.Style{FG: tui.BrightBlack}},
			},
		},
	)
}

// buildTabContent returns different content based on the selected tab
func buildTabContent(tab int) tui.Col {
	switch tab {
	case 0: // Overview
		return tui.Col{
			Children: []any{
				tui.Text{Content: "System Overview", Style: tui.Style{FG: tui.White, Attr: tui.AttrBold}},
				tui.Spacer{Height: 1},
				// Stats row with jumpable cards
				tui.Row{
					Gap: 2,
					Children: []any{
						buildJumpCard("Requests", "1.2M", "/day", tui.Cyan),
						buildJumpCard("Errors", "0.02%", "rate", tui.Green),
						buildJumpCard("Latency", "45ms", "p99", tui.Yellow),
						buildJumpCard("Users", "8,432", "active", tui.Magenta),
					},
				},
				tui.Spacer{Height: 1},
				tui.Text{Content: "Recent Activity", Style: tui.Style{FG: tui.White, Attr: tui.AttrBold}},
				tui.Spacer{Height: 1},
				buildActivityItem("User login", "john@example.com", "2m ago", tui.Green),
				buildActivityItem("API call", "GET /users", "5m ago", tui.Cyan),
				buildActivityItem("Cache miss", "session:abc123", "8m ago", tui.Yellow),
			},
		}

	case 1: // Metrics
		return tui.Col{
			Children: []any{
				tui.Text{Content: "Performance Metrics", Style: tui.Style{FG: tui.White, Attr: tui.AttrBold}},
				tui.Spacer{Height: 1},
				tui.Row{
					Gap: 2,
					Children: []any{
						buildJumpCard("CPU", "42%", "avg", tui.Cyan),
						buildJumpCard("Memory", "2.1GB", "used", tui.Yellow),
						buildJumpCard("Disk I/O", "120MB/s", "read", tui.Green),
						buildJumpCard("Network", "450Mbps", "in", tui.Magenta),
					},
				},
				tui.Spacer{Height: 1},
				tui.Text{Content: "Throughput (last hour)", Style: tui.Style{FG: tui.BrightBlack}},
				tui.Sparkline{Values: []float64{10, 25, 40, 35, 50, 45, 60, 55, 70, 65, 80, 75}, Style: tui.Style{FG: tui.Cyan}},
			},
		}

	case 2: // Logs
		return tui.Col{
			Children: []any{
				tui.Text{Content: "Recent Logs", Style: tui.Style{FG: tui.White, Attr: tui.AttrBold}},
				tui.Spacer{Height: 1},
				buildLogEntry("INFO", "Server started on port 8080", tui.Green),
				buildLogEntry("DEBUG", "Processing request /api/users", tui.Cyan),
				buildLogEntry("WARN", "Cache nearing capacity (85%)", tui.Yellow),
				buildLogEntry("INFO", "Database connection established", tui.Green),
				buildLogEntry("ERROR", "Failed to connect to Redis", tui.Red),
				buildLogEntry("INFO", "Retry successful, Redis connected", tui.Green),
			},
		}

	case 3: // Alerts
		return tui.Col{
			Children: []any{
				tui.Text{Content: "Active Alerts", Style: tui.Style{FG: tui.White, Attr: tui.AttrBold}},
				tui.Spacer{Height: 1},
				buildAlertItem("CRITICAL", "Database replica lag > 30s", tui.Red),
				buildAlertItem("WARNING", "Memory usage above 80%", tui.Yellow),
				buildAlertItem("WARNING", "SSL certificate expires in 7 days", tui.Yellow),
				tui.Spacer{Height: 1},
				tui.Text{Content: "Resolved (last 24h)", Style: tui.Style{FG: tui.BrightBlack}},
				tui.Spacer{Height: 1},
				buildAlertItem("OK", "API latency normalized", tui.Green),
				buildAlertItem("OK", "Disk space freed", tui.Green),
			},
		}

	default:
		return tui.Col{
			Children: []any{
				tui.Text{Content: "Unknown tab", Style: tui.Style{FG: tui.Red}},
			},
		}
	}
}

// buildActivityItem creates a jumpable activity entry
func buildActivityItem(action, detail, when string, color tui.Color) tui.Jump {
	return tui.Jump{
		Child: tui.Row{
			Children: []any{
				tui.Text{Content: fmt.Sprintf("%-12s", action), Style: tui.Style{FG: color}},
				tui.Text{Content: fmt.Sprintf("%-20s", detail), Style: tui.Style{FG: tui.White}},
				tui.Text{Content: when, Style: tui.Style{FG: tui.BrightBlack}},
			},
		},
		OnSelect: func() {
			status = fmt.Sprintf("Activity: %s - %s", action, detail)
			rebuildView()
		},
	}
}

// buildLogEntry creates a jumpable log entry
func buildLogEntry(level, message string, color tui.Color) tui.Jump {
	return tui.Jump{
		Child: tui.Row{
			Children: []any{
				tui.Text{Content: fmt.Sprintf("[%-5s]", level), Style: tui.Style{FG: color, Attr: tui.AttrBold}},
				tui.Text{Content: " " + message, Style: tui.Style{FG: tui.White}},
			},
		},
		OnSelect: func() {
			status = fmt.Sprintf("Log: [%s] %s", level, message)
			rebuildView()
		},
	}
}

// buildAlertItem creates a jumpable alert entry
func buildAlertItem(severity, message string, color tui.Color) tui.Jump {
	icon := "●"
	if severity == "CRITICAL" {
		icon = "◆"
	} else if severity == "OK" {
		icon = "✓"
	}
	return tui.Jump{
		Child: tui.Row{
			Children: []any{
				tui.Text{Content: icon + " ", Style: tui.Style{FG: color}},
				tui.Text{Content: fmt.Sprintf("%-10s", severity), Style: tui.Style{FG: color, Attr: tui.AttrBold}},
				tui.Text{Content: message, Style: tui.Style{FG: tui.White}},
			},
		},
		OnSelect: func() {
			status = fmt.Sprintf("Alert: [%s] %s", severity, message)
			rebuildView()
		},
	}
}

// buildJumpCard creates a jumpable stats card
func buildJumpCard(title, value, unit string, color tui.Color) tui.Jump {
	return tui.Jump{
		Child: tui.Col{
			Children: []any{
				tui.Text{Content: title, Style: tui.Style{FG: tui.BrightBlack}},
				tui.Row{
					Children: []any{
						tui.Text{Content: value, Style: tui.Style{FG: color, Attr: tui.AttrBold}},
						tui.Text{Content: " " + unit, Style: tui.Style{FG: tui.BrightBlack}},
					},
				},
			},
		}.Border(tui.BorderSingle),
		OnSelect: func() {
			status = fmt.Sprintf("Card details: %s = %s %s", title, value, unit)
			rebuildView()
		},
	}
}
