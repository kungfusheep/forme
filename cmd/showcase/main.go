package main

import (
	"log"
	"time"

	"riffkey"
	. "tui"
)

func main() {
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

	tree := &TreeNode{
		Label:    "Components",
		Expanded: true,
		Children: []*TreeNode{
			{
				Label:    "Layout",
				Expanded: true,
				Children: []*TreeNode{
					{Label: "VBox"},
					{Label: "HBox"},
					{Label: "Box"},
				},
			},
			{
				Label:    "Display",
				Expanded: true,
				Children: []*TreeNode{
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
				Children: []*TreeNode{
					{Label: "Spinner"},
					{Label: "Scrollbar"},
					{Label: "Tabs"},
					{Label: "TreeView"},
				},
			},
		},
	}

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(
		VBox(
			// title
			HBox.Border(BorderDouble)(Text(" TUI Component Showcase ")),

			SpaceH(1),

			// main content row
			HBox.Gap(2)(
				// left column
				VBox.WidthPct(0.30)(
					Text("Leader:").Bold(),
					Leader("Status", "Active").Width(25).Fill('.'),
					Leader("Memory", "1.2GB").Width(25).Fill('-'),
					Leader("CPU", "45%").Width(25),

					SpaceH(1),
					Text("Sparkline:").Bold(),
					Sparkline(&sparkData).Width(25).Style(Style{FG: Cyan}),

					SpaceH(1),
					Text("Spinners:").Bold(),
					HBox(Spinner(&spinnerFrame).Frames(SpinnerBraille), Text(" Braille")),
					HBox(Spinner(&spinnerFrame).Frames(SpinnerDots), Text(" Dots")),
					HBox(Spinner(&spinnerFrame).Frames(SpinnerCircle), Text(" Circle")),
					HBox(Spinner(&spinnerFrame).Frames(SpinnerLine), Text(" Line")),
				),

				VRule().Style(Style{FG: BrightBlack}),

				// middle column - table
				VBox.WidthPct(0.35)(
					Text("Table:").Bold(),
					Table{
						Columns: []TableColumn{
							{Header: "Component", Width: 12, Align: AlignLeft},
							{Header: "Description", Width: 16, Align: AlignLeft},
							{Header: "Status", Width: 8, Align: AlignCenter},
						},
						Rows:        &tableRows,
						ShowHeader:  true,
						HeaderStyle: Style{FG: Yellow, Attr: AttrBold},
						RowStyle:    Style{FG: White},
						AltRowStyle: Style{FG: BrightBlack},
					},
				),

				VRule().Style(Style{FG: BrightBlack}),

				// right column - tree
				VBox(
					Text("TreeView:").Bold(),
					TreeView{
						Root:     tree,
						ShowRoot: true,
						Indent:   2,
						Style:    Style{FG: Green},
					},
				),
			),

			SpaceH(1),
			HRule().Style(Style{FG: BrightBlack}),
			SpaceH(1),

			// bottom section - tabs and scrollbar
			HBox.Gap(4)(
				VBox(
					Text("Tabs (Underline):").Bold(),
					TabsNode{
						Labels:        []string{"Home", "Settings", "Help"},
						Selected:      &selectedTab,
						ActiveStyle:   Style{FG: Cyan},
						InactiveStyle: Style{FG: White},
					},
				),
				VBox(
					Text("Tabs (Bracket):").Bold(),
					TabsNode{
						Labels:        []string{"Files", "Edit", "View"},
						Selected:      &selectedTab,
						Style:         TabsStyleBracket,
						ActiveStyle:   Style{FG: Green},
						InactiveStyle: Style{FG: White},
					},
				),
				VBox(
					Text("Scrollbar:").Bold(),
					HBox(
						Text("Pos: "),
						Scroll(100, 20, &scrollPos).Length(10).ThumbStyle(Style{FG: Cyan}),
					),
				),
			),

			SpaceH(1),

			// box style tabs
			Text("Tabs (Box style):").Bold(),
			TabsNode{
				Labels:        []string{"Dashboard", "Analytics", "Reports"},
				Selected:      &selectedTab,
				Style:         TabsStyleBox,
				ActiveStyle:   Style{FG: Magenta},
				InactiveStyle: Style{FG: BrightBlack},
			},

			SpaceH(1),
			HRule(),
			Text("Keys: Tab=cycle tabs | j/k=scroll | Space=toggle tree | q=quit"),
		),
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
			if len(tree.Children) > 2 {
				tree.Children[2].Expanded = !tree.Children[2].Expanded
			}
		})

	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		frame := 0
		for range ticker.C {
			frame++
			spinnerFrame = frame
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
