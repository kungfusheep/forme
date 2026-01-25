package main

import (
	"fmt"
	"log"

	"riffkey"
	. "forme"
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
	color  Color
}{
	{"API", "Online", Green},
	{"Database", "Online", Green},
	{"Cache", "Warning", Yellow},
	{"Queue", "Online", Green},
	{"Storage", "Online", Green},
	{"Auth", "Online", Green},
}

var tabs = []string{"Overview", "Metrics", "Logs", "Alerts"}

var app *App

func main() {
	var err error
	app, err = NewApp()
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
		TextNode{Content: "MENU", Style: Style{FG: Cyan, Attr: AttrBold}},
		SpacerNode{Height: 1},
	}
	for i, item := range menuItems {
		idx := i
		label := item.label
		style := Style{FG: White}
		if item.action == activePanel {
			style = Style{FG: Cyan, Attr: AttrBold}
			label = "> " + label
		} else {
			label = "  " + label
		}
		menuChildren = append(menuChildren, JumpNode{
			Child: TextNode{Content: label, Style: style},
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
		actionChildren = append(actionChildren, JumpNode{
			Child: HBoxNode{
				Children: []any{
					TextNode{Content: fmt.Sprintf("[%s %s]", action.icon, action.label), Style: Style{FG: BrightWhite}},
				},
			},
			OnSelect: func() {
				status = fmt.Sprintf("Action: %s", quickActions[idx].label)
				rebuildView()
			},
		})
		if i < len(quickActions)-1 {
			actionChildren = append(actionChildren, TextNode{Content: "  "})
		}
	}

	// Build subsystem grid with jump targets
	subsystemChildren := []any{
		TextNode{Content: "SUBSYSTEMS", Style: Style{FG: Cyan, Attr: AttrBold}},
		SpacerNode{Height: 1},
	}
	for i, sys := range subsystems {
		idx := i
		indicator := "●"
		subsystemChildren = append(subsystemChildren, JumpNode{
			Child: HBoxNode{
				Children: []any{
					TextNode{Content: indicator + " ", Style: Style{FG: sys.color}},
					TextNode{Content: fmt.Sprintf("%-10s", sys.name), Style: Style{FG: White}},
					TextNode{Content: sys.status, Style: Style{FG: sys.color}},
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
		style := Style{FG: BrightBlack}
		content := fmt.Sprintf(" %s ", label)
		if i == selectedTab {
			style = Style{FG: Cyan, Attr: AttrBold}
			content = fmt.Sprintf("[%s]", label)
		}
		tabChildren = append(tabChildren, JumpNode{
			Child: TextNode{Content: content, Style: style},
			OnSelect: func() {
				selectedTab = idx
				status = fmt.Sprintf("Tab: %s", tabs[idx])
				rebuildView()
			},
		})
	}

	app.SetView(
		VBoxNode{
			Children: []any{
				// Header
				HBoxNode{
					Children: []any{
						TextNode{Content: "Dashboard", Style: Style{FG: Cyan, Attr: AttrBold}},
						SpacerNode{},
						TextNode{Content: status, Style: Style{FG: Yellow}},
					},
				},
				HRuleNode{Style: Style{FG: BrightBlack}},
				SpacerNode{Height: 1},

				// Main layout: sidebar + content
				HBoxNode{
					Gap: 2,
					Children: []any{
						// Sidebar
						VBoxNode{
							Children: menuChildren,
						}.WidthPct(0.15),

						VRuleNode{Style: Style{FG: BrightBlack}},

						// Main content area
						VBoxNode{
							Children: []any{
								// Quick actions bar
								HBoxNode{Children: actionChildren},
								SpacerNode{Height: 1},

								// Tabs
								HBoxNode{Children: tabChildren},
								HRuleNode{Style: Style{FG: BrightBlack}},
								SpacerNode{Height: 1},

								// Content based on selected tab
								buildTabContent(selectedTab),
							},
						},

						VRuleNode{Style: Style{FG: BrightBlack}},

						// Right sidebar - subsystems
						VBoxNode{
							Children: subsystemChildren,
						}.WidthPct(0.20),
					},
				},

				// Footer
				SpacerNode{Height: 1},
				HRuleNode{Style: Style{FG: BrightBlack}},
				TextNode{Content: "g:jump | tab:cycle tabs | q:quit", Style: Style{FG: BrightBlack}},
			},
		},
	)
}

// buildTabContent returns different content based on the selected tab
func buildTabContent(tab int) VBoxNode {
	switch tab {
	case 0: // Overview
		return VBoxNode{
			Children: []any{
				TextNode{Content: "System Overview", Style: Style{FG: White, Attr: AttrBold}},
				SpacerNode{Height: 1},
				// Stats row with jumpable cards
				HBoxNode{
					Gap: 2,
					Children: []any{
						buildJumpCard("Requests", "1.2M", "/day", Cyan),
						buildJumpCard("Errors", "0.02%", "rate", Green),
						buildJumpCard("Latency", "45ms", "p99", Yellow),
						buildJumpCard("Users", "8,432", "active", Magenta),
					},
				},
				SpacerNode{Height: 1},
				TextNode{Content: "Recent Activity", Style: Style{FG: White, Attr: AttrBold}},
				SpacerNode{Height: 1},
				buildActivityItem("User login", "john@example.com", "2m ago", Green),
				buildActivityItem("API call", "GET /users", "5m ago", Cyan),
				buildActivityItem("Cache miss", "session:abc123", "8m ago", Yellow),
			},
		}

	case 1: // Metrics
		return VBoxNode{
			Children: []any{
				TextNode{Content: "Performance Metrics", Style: Style{FG: White, Attr: AttrBold}},
				SpacerNode{Height: 1},
				HBoxNode{
					Gap: 2,
					Children: []any{
						buildJumpCard("CPU", "42%", "avg", Cyan),
						buildJumpCard("Memory", "2.1GB", "used", Yellow),
						buildJumpCard("Disk I/O", "120MB/s", "read", Green),
						buildJumpCard("Network", "450Mbps", "in", Magenta),
					},
				},
				SpacerNode{Height: 1},
				TextNode{Content: "Throughput (last hour)", Style: Style{FG: BrightBlack}},
				SparklineNode{Values: []float64{10, 25, 40, 35, 50, 45, 60, 55, 70, 65, 80, 75}, Style: Style{FG: Cyan}},
			},
		}

	case 2: // Logs
		return VBoxNode{
			Children: []any{
				TextNode{Content: "Recent Logs", Style: Style{FG: White, Attr: AttrBold}},
				SpacerNode{Height: 1},
				buildLogEntry("INFO", "Server started on port 8080", Green),
				buildLogEntry("DEBUG", "Processing request /api/users", Cyan),
				buildLogEntry("WARN", "Cache nearing capacity (85%)", Yellow),
				buildLogEntry("INFO", "Database connection established", Green),
				buildLogEntry("ERROR", "Failed to connect to Redis", Red),
				buildLogEntry("INFO", "Retry successful, Redis connected", Green),
			},
		}

	case 3: // Alerts
		return VBoxNode{
			Children: []any{
				TextNode{Content: "Active Alerts", Style: Style{FG: White, Attr: AttrBold}},
				SpacerNode{Height: 1},
				buildAlertItem("CRITICAL", "Database replica lag > 30s", Red),
				buildAlertItem("WARNING", "Memory usage above 80%", Yellow),
				buildAlertItem("WARNING", "SSL certificate expires in 7 days", Yellow),
				SpacerNode{Height: 1},
				TextNode{Content: "Resolved (last 24h)", Style: Style{FG: BrightBlack}},
				SpacerNode{Height: 1},
				buildAlertItem("OK", "API latency normalized", Green),
				buildAlertItem("OK", "Disk space freed", Green),
			},
		}

	default:
		return VBoxNode{
			Children: []any{
				TextNode{Content: "Unknown tab", Style: Style{FG: Red}},
			},
		}
	}
}

// buildActivityItem creates a jumpable activity entry
func buildActivityItem(action, detail, when string, color Color) JumpNode {
	return JumpNode{
		Child: HBoxNode{
			Children: []any{
				TextNode{Content: fmt.Sprintf("%-12s", action), Style: Style{FG: color}},
				TextNode{Content: fmt.Sprintf("%-20s", detail), Style: Style{FG: White}},
				TextNode{Content: when, Style: Style{FG: BrightBlack}},
			},
		},
		OnSelect: func() {
			status = fmt.Sprintf("Activity: %s - %s", action, detail)
			rebuildView()
		},
	}
}

// buildLogEntry creates a jumpable log entry
func buildLogEntry(level, message string, color Color) JumpNode {
	return JumpNode{
		Child: HBoxNode{
			Children: []any{
				TextNode{Content: fmt.Sprintf("[%-5s]", level), Style: Style{FG: color, Attr: AttrBold}},
				TextNode{Content: " " + message, Style: Style{FG: White}},
			},
		},
		OnSelect: func() {
			status = fmt.Sprintf("Log: [%s] %s", level, message)
			rebuildView()
		},
	}
}

// buildAlertItem creates a jumpable alert entry
func buildAlertItem(severity, message string, color Color) JumpNode {
	icon := "●"
	if severity == "CRITICAL" {
		icon = "◆"
	} else if severity == "OK" {
		icon = "✓"
	}
	return JumpNode{
		Child: HBoxNode{
			Children: []any{
				TextNode{Content: icon + " ", Style: Style{FG: color}},
				TextNode{Content: fmt.Sprintf("%-10s", severity), Style: Style{FG: color, Attr: AttrBold}},
				TextNode{Content: message, Style: Style{FG: White}},
			},
		},
		OnSelect: func() {
			status = fmt.Sprintf("Alert: [%s] %s", severity, message)
			rebuildView()
		},
	}
}

// buildJumpCard creates a jumpable stats card
func buildJumpCard(title, value, unit string, color Color) JumpNode {
	return JumpNode{
		Child: VBoxNode{
			Children: []any{
				TextNode{Content: title, Style: Style{FG: BrightBlack}},
				HBoxNode{
					Children: []any{
						TextNode{Content: value, Style: Style{FG: color, Attr: AttrBold}},
						TextNode{Content: " " + unit, Style: Style{FG: BrightBlack}},
					},
				},
			},
		}.Border(BorderSingle),
		OnSelect: func() {
			status = fmt.Sprintf("Card details: %s = %s %s", title, value, unit)
			rebuildView()
		},
	}
}
