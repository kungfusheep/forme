package main

import (
	"log"
	"time"

	"riffkey"
	"tui"
)

func main() {
	// State
	spinnerFrame := 0
	scrollPos := 25
	selectedTab := 0
	sparkData := []float64{10, 25, 50, 75, 100, 80, 60, 40, 30, 50, 70, 90, 85, 65, 45}
	tableRows := [][]string{
		{"Leader", "Label...Value", "Done"},
		{"Table", "Tabular data", "Done"},
		{"Sparkline", "Mini charts", "Done"},
		{"HRule/VRule", "Dividers", "Done"},
		{"Spinner", "Loading anim", "Done"},
		{"Scrollbar", "Scroll indicator", "Done"},
		{"Tabs", "Tab headers", "Done"},
		{"TreeView", "Hierarchical", "Done"},
	}

	// TreeView data
	tree := &tui.TreeNode{
		Label:    "Components",
		Expanded: true,
		Children: []*tui.TreeNode{
			{
				Label:    "Layout",
				Expanded: true,
				Children: []*tui.TreeNode{
					{Label: "Row"},
					{Label: "Col"},
					{Label: "Box"},
				},
			},
			{
				Label:    "Display",
				Expanded: true,
				Children: []*tui.TreeNode{
					{Label: "Text"},
					{Label: "Progress"},
					{Label: "RichText"},
					{Label: "Leader"},
					{Label: "Table"},
					{Label: "Sparkline"},
				},
			},
			{
				Label:    "Widgets",
				Expanded: false,
				Children: []*tui.TreeNode{
					{Label: "Spinner"},
					{Label: "Scrollbar"},
					{Label: "Tabs"},
					{Label: "TreeView"},
				},
			},
		},
	}

	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Build the UI
	app.SetView(
		tui.VBox{
			Children: []any{
				// Title
				tui.HBox{
					Children: []any{
						tui.Text{Content: " TUI Component Showcase "},
					},
				}.Border(tui.BorderDouble),

				tui.Spacer{Height: 1},

				// Main content row
				tui.HBox{
					Gap: 2,
					Children: []any{
						// Left column - Leader, Sparkline, Spinner
						tui.VBox{
							Children: []any{
								tui.Text{Content: "Leader:", Style: tui.Style{Attr: tui.AttrBold}},
								tui.Leader{Label: "Status", Value: "Active", Width: 25, Fill: '.'},
								tui.Leader{Label: "Memory", Value: "1.2GB", Width: 25, Fill: '-'},
								tui.Leader{Label: "CPU", Value: "45%", Width: 25},

								tui.Spacer{Height: 1},
								tui.Text{Content: "Sparkline:", Style: tui.Style{Attr: tui.AttrBold}},
								tui.Sparkline{Values: &sparkData, Width: 25, Style: tui.Style{FG: tui.Cyan}},

								tui.Spacer{Height: 1},
								tui.Text{Content: "Spinners:", Style: tui.Style{Attr: tui.AttrBold}},
								tui.HBox{
									Children: []any{
										tui.Spinner{Frame: &spinnerFrame, Frames: tui.SpinnerBraille},
										tui.Text{Content: " Braille"},
									},
								},
								tui.HBox{
									Children: []any{
										tui.Spinner{Frame: &spinnerFrame, Frames: tui.SpinnerDots},
										tui.Text{Content: " Dots"},
									},
								},
								tui.HBox{
									Children: []any{
										tui.Spinner{Frame: &spinnerFrame, Frames: tui.SpinnerCircle},
										tui.Text{Content: " Circle"},
									},
								},
								tui.HBox{
									Children: []any{
										tui.Spinner{Frame: &spinnerFrame, Frames: tui.SpinnerLine},
										tui.Text{Content: " Line"},
									},
								},
							},
						}.WidthPct(0.30),

						// Vertical divider
						tui.VRule{Style: tui.Style{FG: tui.BrightBlack}},

						// Middle column - Table
						tui.VBox{
							Children: []any{
								tui.Text{Content: "Table:", Style: tui.Style{Attr: tui.AttrBold}},
								tui.Table{
									Columns: []tui.TableColumn{
										{Header: "Component", Width: 12, Align: tui.AlignLeft},
										{Header: "Description", Width: 16, Align: tui.AlignLeft},
										{Header: "Status", Width: 8, Align: tui.AlignCenter},
									},
									Rows:        &tableRows,
									ShowHeader:  true,
									HeaderStyle: tui.Style{FG: tui.Yellow, Attr: tui.AttrBold},
									RowStyle:    tui.Style{FG: tui.White},
									AltRowStyle: tui.Style{FG: tui.BrightBlack},
								},
							},
						}.WidthPct(0.35),

						// Vertical divider
						tui.VRule{Style: tui.Style{FG: tui.BrightBlack}},

						// Right column - TreeView
						tui.VBox{
							Children: []any{
								tui.Text{Content: "TreeView:", Style: tui.Style{Attr: tui.AttrBold}},
								tui.TreeView{
									Root:     tree,
									ShowRoot: true,
									Indent:   2,
									Style:    tui.Style{FG: tui.Green},
								},
							},
						},
					},
				},

				tui.Spacer{Height: 1},
				tui.HRule{Char: '─', Style: tui.Style{FG: tui.BrightBlack}},
				tui.Spacer{Height: 1},

				// Bottom section - Tabs and Scrollbar
				tui.HBox{
					Gap: 4,
					Children: []any{
						tui.VBox{
							Children: []any{
								tui.Text{Content: "Tabs (Underline):", Style: tui.Style{Attr: tui.AttrBold}},
								tui.Tabs{
									Labels:        []string{"Home", "Settings", "Help"},
									Selected:      &selectedTab,
									ActiveStyle:   tui.Style{FG: tui.Cyan},
									InactiveStyle: tui.Style{FG: tui.White},
								},
							},
						},
						tui.VBox{
							Children: []any{
								tui.Text{Content: "Tabs (Bracket):", Style: tui.Style{Attr: tui.AttrBold}},
								tui.Tabs{
									Labels:        []string{"Files", "Edit", "View"},
									Selected:      &selectedTab,
									Style:         tui.TabsStyleBracket,
									ActiveStyle:   tui.Style{FG: tui.Green},
									InactiveStyle: tui.Style{FG: tui.White},
								},
							},
						},
						tui.VBox{
							Children: []any{
								tui.Text{Content: "Scrollbar:", Style: tui.Style{Attr: tui.AttrBold}},
								tui.HBox{
									Children: []any{
										tui.Text{Content: "Pos: "},
										tui.Scrollbar{
											ContentSize: 100,
											ViewSize:    20,
											Position:    &scrollPos,
											Length:      10,
											ThumbStyle:  tui.Style{FG: tui.Cyan},
										},
									},
								},
							},
						},
					},
				},

				tui.Spacer{Height: 1},

				// Box style tabs
				tui.Text{Content: "Tabs (Box style):", Style: tui.Style{Attr: tui.AttrBold}},
				tui.Tabs{
					Labels:        []string{"Dashboard", "Analytics", "Reports"},
					Selected:      &selectedTab,
					Style:         tui.TabsStyleBox,
					ActiveStyle:   tui.Style{FG: tui.Magenta},
					InactiveStyle: tui.Style{FG: tui.BrightBlack},
				},

				tui.Spacer{Height: 1},
				tui.HRule{},
				tui.Text{Content: "Keys: Tab=cycle tabs | j/k=scroll | Space=toggle tree | q=quit"},
			},
		},
	).
		Handle("q", func(m riffkey.Match) {
			app.Stop()
		}).
		Handle("tab", func(m riffkey.Match) {
			selectedTab = (selectedTab + 1) % 3
		}).
		Handle("j", func(m riffkey.Match) {
			if scrollPos < 80 {
				scrollPos += 10
			}
		}).
		Handle("k", func(m riffkey.Match) {
			if scrollPos > 0 {
				scrollPos -= 10
			}
		}).
		Handle("<Space>", func(m riffkey.Match) {
			// Toggle "Widgets" expansion
			if len(tree.Children) > 2 {
				tree.Children[2].Expanded = !tree.Children[2].Expanded
			}
		})

	// Animation ticker
	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		frame := 0
		for range ticker.C {
			frame++
			spinnerFrame = frame

			// Rotate sparkline data
			first := sparkData[0]
			copy(sparkData, sparkData[1:])
			sparkData[len(sparkData)-1] = first

			app.RenderNow()
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
