package main

import (
	"fmt"
	"strings"

	. "github.com/kungfusheep/glyph"
)

func main() {
	fmt.Println("=== Layer/Blit Visual Demo ===")
	fmt.Println()

	// Create a small "layer" - imagine this is a webpage that's been rendered
	// The layer is TALLER than what fits on screen
	layer := NewBuffer(40, 20) // 40 wide, 20 tall

	// Fill the layer with distinct content so we can see scrolling
	for y := 0; y < 20; y++ {
		// Each line has a clear line number and pattern
		line := fmt.Sprintf("│ Line %02d: %s │", y, strings.Repeat("█", 25))
		layer.WriteStringFast(0, y, line, Style{}, 40)
	}

	fmt.Println("LAYER (the full pre-rendered content - 20 lines tall):")
	fmt.Println("This is like a webpage rendered to an off-screen buffer.")
	fmt.Println(strings.Repeat("─", 42))
	printBuffer(layer)
	fmt.Println(strings.Repeat("─", 42))

	// Create a "screen" - this is what the user actually sees
	// It's SHORTER than the layer (like a viewport)
	screen := NewBuffer(40, 8) // Only 8 lines tall!

	fmt.Println("\n\nSCREEN (viewport - only 8 lines tall):")
	fmt.Println("This is what the user actually sees on their terminal.")
	fmt.Println()

	// Demo: Blit different portions of the layer to the screen
	// This is like scrolling through the content

	fmt.Println(">>> screen.Blit(layer, srcY=0) - showing TOP of content")
	fmt.Println(strings.Repeat("─", 42))
	screen.Blit(layer, 0, 0, 0, 0, 40, 8) // srcY=0, show lines 0-7
	printBuffer(screen)
	fmt.Println(strings.Repeat("─", 42))

	fmt.Println("\n>>> screen.Blit(layer, srcY=6) - scrolled down 6 lines")
	fmt.Println(strings.Repeat("─", 42))
	screen.Blit(layer, 0, 6, 0, 0, 40, 8) // srcY=6, show lines 6-13
	printBuffer(screen)
	fmt.Println(strings.Repeat("─", 42))

	fmt.Println("\n>>> screen.Blit(layer, srcY=12) - scrolled to BOTTOM")
	fmt.Println(strings.Repeat("─", 42))
	screen.Blit(layer, 0, 12, 0, 0, 40, 8) // srcY=12, show lines 12-19
	printBuffer(screen)
	fmt.Println(strings.Repeat("─", 42))

	fmt.Println("\n\nKEY INSIGHT:")
	fmt.Println("• The LAYER is rendered ONCE (expensive)")
	fmt.Println("• Each scroll just BLITS a different portion (cheap - it's memcpy)")
	fmt.Println("• For Browse: render HTML→Layer once per page load")
	fmt.Println("• Scrolling = changing srcY, no re-rendering needed!")
}

func printBuffer(b *Buffer) {
	for y := 0; y < b.Height(); y++ {
		for x := 0; x < b.Width(); x++ {
			cell := b.Get(x, y)
			if cell.Rune == 0 {
				fmt.Print(" ")
			} else {
				fmt.Print(string(cell.Rune))
			}
		}
		fmt.Println()
	}
}
