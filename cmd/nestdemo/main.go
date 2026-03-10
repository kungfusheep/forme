// nestdemo: Stress test for complex nested layouts
// Tests deeply nested VBox/HBox, borders, spacers, ForEach, SelectionList, and more.
// Uses fully reactive template with Switch - SetView called only once!
package main

import (
	"fmt"
	"log"

	. "github.com/kungfusheep/glyph"
)

type MenuItem struct {
	Icon     string
	Label    string
	Shortcut string
}

type KeyValue struct {
	Key   string
	Value string
}

func main() {
	currentDemo := 0
	demoNames := []string{
		"1. Nested VBox in HBox",
		"2. HBox in VBox in HBox",
		"3. Bordered + Nested Content",
		"4. Nested Borders",
		"5. Multiple Spacers",
		"6. ForEach + Nested HBox",
		"7. SelectionList + Complex Render",
		"8. Grow Factors",
		"9. Styled Text in Borders",
		"10. HRule Inside Border",
		"11. Deep Nesting (5 levels)",
		"12. HBox + Sidebar + LayerView",
		"13. All Combined Stress Test",
	}
	demoName := demoNames[0]

	menuItems := []MenuItem{
		{Icon: "*", Label: "New File", Shortcut: "Ctrl+N"},
		{Icon: "#", Label: "Open", Shortcut: "Ctrl+O"},
		{Icon: "!", Label: "Save", Shortcut: "Ctrl+S"},
		{Icon: "X", Label: "Close", Shortcut: "Ctrl+W"},
		{Icon: "?", Label: "Help", Shortcut: "F1"},
	}
	menuSel := 0

	kvPairs := []KeyValue{
		{Key: "alpha", Value: "1"},
		{Key: "beta", Value: "2"},
		{Key: "gamma", Value: "3"},
		{Key: "delta", Value: "4"},
	}
	kvSel := 0

	menuList := List(&menuItems).
		Selection(&menuSel).
		MaxVisible(10).
		Style(Style{BG: PaletteColor(235)}).
		SelectedStyle(Style{BG: PaletteColor(240)}).
		MarkerStyle(Style{FG: Cyan}).
		Render(func(item *MenuItem) any {
			return HBox.Gap(1)(
				Text(&item.Icon).FG(Yellow),
				Text(&item.Label),
				Space().Char('_'),
				Text(&item.Shortcut).FG(BrightBlack),
			)
		})

	kvList := List(&kvPairs).
		Selection(&kvSel).
		MaxVisible(4).
		Style(Style{BG: PaletteColor(234)}).
		SelectedStyle(Style{BG: PaletteColor(238)}).
		Render(func(kv *KeyValue) any {
			return HBox(
				Text(&kv.Key).FG(Cyan),
				Space().Char('-').Style(Style{FG: BrightBlack}),
				Text(&kv.Value).FG(Yellow),
			)
		})

	editorLayer := NewLayer()
	editorLayer.EnsureSize(80, 20)
	for y := 0; y < 20; y++ {
		lineText := fmt.Sprintf("%3d | // This is line %d of the editor content", y+1, y+1)
		editorLayer.SetLineString(y, lineText, Style{FG: DefaultColor()})
	}

	type FileItem struct {
		Name    string
		IsDir   bool
		Display string
	}
	sidebarItems := []FileItem{
		{Name: "src", IsDir: true, Display: "+ src/"},
		{Name: "main.go", IsDir: false, Display: "  main.go"},
		{Name: "utils.go", IsDir: false, Display: "  utils.go"},
		{Name: "test", IsDir: true, Display: "+ test/"},
		{Name: "README.md", IsDir: false, Display: "  README.md"},
	}
	sidebarVisible := true

	sidebarList := List(&sidebarItems).
		MaxVisible(10).
		Style(Style{BG: PaletteColor(235)}).
		SelectedStyle(Style{BG: PaletteColor(238)}).
		Render(func(item *FileItem) any {
			style := Style{FG: White}
			if item.IsDir {
				style.FG = Cyan
			}
			return Text(&item.Display).Style(style)
		})

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(VBox(
		// Header
		HBox.Gap(1)(
			Text("Complex Nested Layouts Demo").FG(Cyan).Bold(),
			Space(),
			Text("j/k: switch demo | h/l: adjust selection | q: quit").FG(BrightBlack),
		),
		HRule(),

		// Demo selector
		HBox.Gap(2)(
			Text("Demo:"),
			Text(&demoName).FG(Yellow),
		),
		SpaceH(1),

		// Demo content - reactive Switch
		Switch(&currentDemo).
			Case(0, // Nested VBox in HBox
				VBox(
					Text("HBox containing multiple VBoxes:"),
					SpaceH(1),
					HBox.Gap(3)(
						VBox.Border(BorderSingle)(
							Text("Column A").FG(Red),
							Text("A1"),
							Text("A2"),
							Text("A3"),
						),
						VBox.Border(BorderSingle)(
							Text("Column B").FG(Green),
							Text("B1"),
							Text("B2"),
						),
						VBox.Border(BorderSingle)(
							Text("Column C").FG(Blue),
							Text("C1"),
						),
					),
				)).
			Case(1, // HBox in VBox in HBox
				VBox(
					Text("3 levels of nesting:"),
					SpaceH(1),
					HBox.Gap(1)(
						Text("[").FG(Yellow),
						VBox.Border(BorderRounded)(
							HBox(Text("X").FG(Red), Text(" "), Text("Y").FG(Green)),
							HBox(Text("1").FG(Blue), Text(" "), Text("2").FG(Magenta)),
						),
						Text("]").FG(Yellow),
					),
				)).
			Case(2, // Bordered + Nested Content
				VBox(
					Text("Bordered container with nested HBox and Spacer:"),
					SpaceH(1),
					VBox.Border(BorderRounded)(
						HBox.Gap(1)(Text("Name:").FG(Cyan), Text("John Doe")),
						HBox.Gap(1)(Text("Email:").FG(Cyan), Text("john@example.com")),
						HRule(),
						HBox.Gap(1)(Text("Status:").FG(Cyan), Space(), Text("Active").FG(Green)),
					),
				)).
			Case(3, // Nested Borders
				VBox(
					Text("Nested borders (double > single > rounded):"),
					SpaceH(1),
					VBox.Border(BorderDouble)(
						Text("Outer content"),
						VBox.Border(BorderSingle)(
							Text("Inner content").FG(Yellow),
							VBox.Border(BorderRounded)(
								Text("Deepest!").FG(Red),
							),
						),
					),
				)).
			Case(4, // Multiple Spacers
				VBox(
					Text("Multiple spacers distribute space evenly:"),
					HBox.Border(BorderSingle)(
						Text("LEFT").FG(Red),
						Space(),
						Text("CENTER").FG(Green),
						Space(),
						Text("RIGHT").FG(Blue),
					),
					SpaceH(1),
					Text("With dotted leaders:"),
					HBox.Border(BorderSingle)(
						Text("Item"),
						Space().Char('.').Style(Style{FG: BrightBlack}),
						Text("$9.99"),
					),
				)).
			Case(5, // ForEach + Nested HBox
				VBox(
					Text("ForEach with nested HBox and dotted leaders:"),
					SpaceH(1),
					VBox.Border(BorderRounded)(
						ForEach(&kvPairs, func(kv *KeyValue) any {
							return HBox.Gap(1)(
								Text(&kv.Key).FG(Cyan),
								Space().Char('.').Style(Style{FG: BrightBlack}),
								Text(&kv.Value).FG(Yellow),
							)
						}),
					),
				)).
			Case(6, // SelectionList + Complex Render
				VBox(
					Text("SelectionList with complex HBox render (use h/l to navigate):"),
					SpaceH(1),
					VBox.Border(BorderRounded)(menuList),
				)).
			Case(7, // Grow Factors
				VBox(
					Text("Spacers with different grow factors (1:2:1):"),
					SpaceH(1),
					HBox.Border(BorderSingle)(
						Text("A").FG(Red),
						Space().Grow(1),
						Text("B").FG(Green),
						Space().Grow(2),
						Text("C").FG(Blue),
						Space().Grow(1),
						Text("D").FG(Magenta),
					),
					SpaceH(1),
					Text("Notice B-C gap is 2x wider than A-B and C-D gaps"),
				)).
			Case(8, // Styled Text in Borders
				VBox(
					Text("Styled text inside bordered containers:"),
					SpaceH(1),
					VBox.Border(BorderRounded)(
						HBox.Gap(2)(
							Text("Red").FG(Red),
							Text("Green").FG(Green),
							Text("Blue").FG(Blue),
						),
						HBox.Gap(1)(
							Text("Bold").Bold(),
							Text("Dim").Dim(),
							Text("Underline").Underline(),
						),
					),
				)).
			Case(9, // HRule Inside Border
				VBox(
					Text("HRule inside bordered VBox (should not bleed):"),
					SpaceH(1),
					VBox.Border(BorderSingle)(
						Text("Header Section").FG(Yellow).Bold(),
						HRule(),
						Text("Body content goes here"),
						Text("More content..."),
						HRule(),
						Text("Footer").FG(BrightBlack),
					),
					)).
			Case(10, // Deep Nesting
				VBox(
					Text("5 levels of nesting:"),
					SpaceH(1),
					VBox.Border(BorderDouble)(
						Text("Level 1").FG(Red),
						HBox(
							Text("Level 2").FG(Yellow),
							SpaceW(1),
							VBox.Border(BorderSingle)(
								Text("Level 3").FG(Green),
								HBox(
									Text("Level 4").FG(Cyan),
									SpaceW(1),
									VBox.Border(BorderRounded)(
										Text("Level 5 - DEEP!").FG(Magenta).Bold(),
									),
								),
							),
						),
					),
					)).
			Case(11, // HBox + Sidebar + LayerView
				VBox(
					Text("HBox with sidebar (SelectionList) + LayerView - replicates editor layout:"),
					SpaceH(1),
					HBox.Gap(1)(
						If(&sidebarVisible).Then(
							VBox.Border(BorderSingle).Width(25)(
								Text("Files").FG(Yellow).Bold(),
								HRule(),
								sidebarList,
							),
						),
						VBox.Border(BorderRounded).Grow(1)(
							Text("Editor").FG(Cyan).Bold(),
							HRule(),
							LayerView(editorLayer).ViewHeight(10),
						),
					),
					SpaceH(1),
					Text("b: toggle sidebar | h/l: navigate sidebar | s/S: scroll editor").FG(BrightBlack),
					)).
			Case(12, // All Combined Stress Test
				VBox(
					Text("Combined stress test (use h/l to navigate list):"),
					SpaceH(1),
					HBox.Gap(2)(
						VBox.Border(BorderDouble)(
							Text("Left Panel").FG(Yellow).Bold(),
							HRule(),
							kvList,
						),
						VBox.Border(BorderSingle)(
							Text("Right Panel").FG(Cyan).Bold(),
							HRule(),
							VBox.Border(BorderRounded)(
								ForEach(&menuItems, func(item *MenuItem) any {
									return HBox.Gap(1)(
										Text(&item.Icon),
										Text(&item.Label),
										Space().Char('.').Style(Style{FG: BrightBlack}),
										Text(&item.Shortcut).FG(BrightBlack),
									)
								}),
							),
						),
					),
				)).
			Default(Text("Unknown demo")),
	))

	nextDemo := func() {
		if currentDemo < len(demoNames)-1 {
			currentDemo++
			demoName = demoNames[currentDemo]
		}
	}
	prevDemo := func() {
		if currentDemo > 0 {
			currentDemo--
			demoName = demoNames[currentDemo]
		}
	}
	selDown := func() {
		switch currentDemo {
		case 6:
			menuList.Down(nil)
		case 11:
			sidebarList.Down(nil)
		case 12:
			kvList.Down(nil)
		}
	}
	selUp := func() {
		switch currentDemo {
		case 6:
			menuList.Up(nil)
		case 11:
			sidebarList.Up(nil)
		case 12:
			kvList.Up(nil)
		}
	}

	app.Handle("j", nextDemo)
	app.Handle("k", prevDemo)
	app.Handle("<Down>", nextDemo)
	app.Handle("<Up>", prevDemo)
	app.Handle("l", selDown)
	app.Handle("h", selUp)
	app.Handle("<Right>", selDown)
	app.Handle("<Left>", selUp)

	app.Handle("s", func() {
		if currentDemo == 11 {
			editorLayer.ScrollDown(1)
		}
	})
	app.Handle("S", func() {
		if currentDemo == 11 {
			editorLayer.ScrollUp(1)
		}
	})
	app.Handle("b", func() {
		if currentDemo == 11 {
			sidebarVisible = !sidebarVisible
		}
	})

	app.Handle("q", app.Stop)
	app.Handle("<Escape>", app.Stop)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Demo completed!")
}
