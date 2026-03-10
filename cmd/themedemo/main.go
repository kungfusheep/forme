package main

import (
	"fmt"
	"log"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"
)

func main() {
	// Available themes with background fill colors
	themes := []struct {
		name  string
		theme ThemeEx
		fill  Color // background fill for the whole app
	}{
		{"Dark", ThemeDark, Hex(0x1E1E2E)},              // dark purple-gray
		{"Light", ThemeLight, Hex(0xF5F5DC)},            // beige
		{"Monochrome", ThemeMonochrome, DefaultColor()}, // terminal default
		{"Custom", ThemeEx{
			Base:   Style{FG: Hex(0xE0E0E0)},
			Muted:  Style{FG: Hex(0x808080)},
			Accent: Style{FG: Hex(0x00BFFF), Attr: AttrBold},
			Error:  Style{FG: Hex(0xFF6B6B)},
			Border: Style{FG: Hex(0x404040)},
		}, Hex(0x2D2D3A)}, // custom dark blue
	}

	// Current theme index and active root style (includes Fill)
	themeIdx := 0
	currentTheme := themes[themeIdx].theme
	themeName := themes[themeIdx].name
	rootStyle := Style{FG: currentTheme.Base.FG, Fill: themes[themeIdx].fill}

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Build the UI using the functional API
	app.SetView(
		// VBox.CascadeStyle() at the root propagates to all children (including Fill)
		VBox.CascadeStyle(&rootStyle)(
			// Header
			HBox.Border(BorderRounded).BorderFG(currentTheme.Border.FG)(
				Text(" Theme Demo "),
				Text(" - Press 't' to cycle themes, 'q' to quit"),
			),

			SpaceH(1),

			// Theme info
			HBox(
				Text("Current theme: "),
				Text(&themeName).Style(currentTheme.Accent),
			),

			SpaceH(1),
			HRule().Style(currentTheme.Border),
			SpaceH(1),

			// Demo content - all inherits Base style automatically
			Text("This text inherits the Base style from the parent VBox."),
			Text("No explicit Style needed - it just works!"),

			SpaceH(1),

			// Nested container with different inherited style
			VBox.CascadeStyle(&currentTheme.Muted)(
				Text("This section uses Muted style (nested CascadeStyle)"),
				Text("All children in here are muted too."),
				HBox(
					Text("Even "),
					Text("deeply "),
					Text("nested "),
					Text("text!"),
				),
			),

			SpaceH(1),

			// Explicit overrides
			HBox(
				Text("Accent: ").Style(currentTheme.Accent),
				Text("Error: ").Style(currentTheme.Error),
				Text("Back to inherited"),
			),

			SpaceH(1),
			HRule().Style(currentTheme.Border),
			SpaceH(1),

			// Another section
			VBox.CascadeStyle(&currentTheme.Accent)(
				Text("This entire section uses Accent style"),
				Text("Great for highlighting important content"),
			),

			SpaceH(1),
			HRule().Style(currentTheme.Border),
			SpaceH(1),

			// Fill demo - containers with background colors
			Text("Container Fill Demo (background colors):"),
			SpaceH(1),

			HBox.Gap(2)(
				VBox.CascadeStyle(&Style{FG: White, Fill: Blue}).Size(12, 3)(
					Text(" Blue Fill "),
					Text(" Panel    "),
				),

				VBox.CascadeStyle(&Style{FG: Black, Fill: Yellow}).Size(12, 3)(
					Text(" Yellow   "),
					Text(" Fill     "),
				),

				VBox.CascadeStyle(&Style{FG: White, Fill: Red}).Size(12, 3)(
					Text(" Red Fill "),
					Text(" Warning  "),
				),
			),

			SpaceH(1),
			HRule().Style(currentTheme.Border),
			SpaceH(1),

			// Text Transform Demo
			Text("Text Transform Demo (inherited styles):"),
			SpaceH(1),

			HBox.Gap(2)(
				VBox.CascadeStyle(&Style{FG: Cyan, Transform: TransformUppercase})(
					Text("uppercase section"),
					Text("all text here"),
					Text("is capitalized"),
				),
				VBox.CascadeStyle(&Style{FG: Magenta, Attr: AttrItalic})(
					Text("Italic Section"),
					Text("All text here"),
					Text("is italicized"),
				),
				VBox.CascadeStyle(&Style{FG: Yellow, Attr: AttrBold | AttrUnderline})(
					Text("Bold+Underline"),
					Text("Combined attrs"),
					Text("also inherit"),
				),
			),

			SpaceH(1),

			// Footer
			Text("The magic: change theme, and everything updates!").Style(currentTheme.Muted),
		),
	).Handle("q", func(m riffkey.Match) {
		app.Stop()
	}).Handle("t", func(m riffkey.Match) {
		// Cycle to next theme
		themeIdx = (themeIdx + 1) % len(themes)

		// Update the theme struct in place - pointers pick up changes
		currentTheme.Base = themes[themeIdx].theme.Base
		currentTheme.Muted = themes[themeIdx].theme.Muted
		currentTheme.Accent = themes[themeIdx].theme.Accent
		currentTheme.Error = themes[themeIdx].theme.Error
		currentTheme.Border = themes[themeIdx].theme.Border
		themeName = themes[themeIdx].name

		// Update root style (FG + Fill for background)
		rootStyle.FG = currentTheme.Base.FG
		rootStyle.Fill = themes[themeIdx].fill

		fmt.Printf("\x1b]0;Theme: %s\x07", themeName) // Update terminal title
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
