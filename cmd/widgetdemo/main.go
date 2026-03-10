// widgetdemo: Demonstrates custom widgets with Widget()
package main

import (
	"log"
	"math"
	"time"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"
)

// partial blocks for sub-character precision (1/8th increments)
var partialBlocks = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// State
	progress := 0.65
	wavePhase := 0.0
	sparkData := []float64{0.2, 0.5, 0.8, 0.3, 0.6, 0.9, 0.4, 0.7, 0.5, 0.3, 0.8, 0.6, 0.4, 0.9, 0.7}

	// Colors for gradients
	gradientStart := RGB(0, 255, 128)  // cyan-green
	gradientEnd := RGB(255, 0, 128)    // magenta-red

	// Smooth gradient progress bar with sub-character precision
	gradientBar := Widget(
		func(availW int16) (w, h int16) {
			return availW, 1
		},
		func(buf *Buffer, x, y, w, h int16) {
			// calculate filled width with sub-character precision
			filledExact := float64(w) * progress
			fullBlocks := int(filledExact)
			partial := int((filledExact - float64(fullBlocks)) * 8) // 0-7 for partial block

			for i := int16(0); i < w; i++ {
				t := float64(i) / float64(w-1)
				color := LerpColor(gradientStart, gradientEnd, t)

				var ch rune
				var style Style

				if int(i) < fullBlocks {
					// full block with gradient color
					ch = '█'
					style.FG = color
				} else if int(i) == fullBlocks && partial > 0 {
					// partial block at the edge
					ch = partialBlocks[partial]
					style.FG = color
				} else {
					// empty portion
					ch = '░'
					style.FG = PaletteColor(238)
				}
				buf.Set(int(x+i), int(y), Cell{Rune: ch, Style: style})
			}
		},
	)

	// Animated sine wave using braille dots (2x4 sub-character resolution)
	sineWave := Widget(
		func(availW int16) (w, h int16) {
			return availW, 5
		},
		func(buf *Buffer, x, y, w, h int16) {
			waveTop := RGB(0, 255, 255)    // cyan
			waveBottom := RGB(255, 0, 255) // magenta

			// braille gives us 4 vertical dots per character
			subRows := int(h) * 4

			// first pass: collect all dot positions
			type dot struct {
				charX, charY int
				dotRow       int // 0-3 within the character
				dotCol       int // 0-1 within the character (left/right)
				value        float64
			}
			var dots []dot

			// sample at 2x horizontal resolution (braille has 2 columns per char)
			for i := 0; i < int(w)*2; i++ {
				angle := wavePhase + float64(i)*0.1
				value := (math.Sin(angle) + 1) / 2 // normalize to 0-1
				subY := int(float64(subRows-1) * (1 - value))

				charX := i / 2
				charY := subY / 4
				dotRow := subY % 4
				dotCol := i % 2

				dots = append(dots, dot{charX, charY, dotRow, dotCol, value})
			}

			// group dots by character position and build braille chars
			charDots := make(map[[2]int][]dot)
			for _, d := range dots {
				key := [2]int{d.charX, d.charY}
				charDots[key] = append(charDots[key], d)
			}

			// render each braille character
			for key, cellDots := range charDots {
				charX, charY := key[0], key[1]
				if charX >= int(w) || charY >= int(h) {
					continue
				}

				// build braille pattern
				// dot positions: [0,3,1,4,2,5,6,7] for bits 0-7
				// layout:  0 3
				//          1 4
				//          2 5
				//          6 7
				var pattern rune = 0x2800 // braille base
				var avgValue float64

				for _, d := range cellDots {
					avgValue += d.value
					bit := brailleBit(d.dotRow, d.dotCol)
					pattern |= (1 << bit)
				}
				avgValue /= float64(len(cellDots))

				color := LerpColor(waveBottom, waveTop, avgValue)
				buf.Set(int(x)+charX, int(y)+charY, Cell{Rune: pattern, Style: Style{FG: color}})
			}
		},
	)

	// Sparkline with gradient
	sparkline := Widget(
		func(availW int16) (w, h int16) {
			return int16(len(sparkData)), 1
		},
		func(buf *Buffer, x, y, w, h int16) {
			chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
			sparkLow := RGB(64, 128, 255)  // blue
			sparkHigh := RGB(255, 128, 64) // orange

			for i, v := range sparkData {
				idx := int(v * float64(len(chars)-1))
				if idx >= len(chars) {
					idx = len(chars) - 1
				}
				color := LerpColor(sparkLow, sparkHigh, v)
				buf.Set(int(x)+i, int(y), Cell{Rune: chars[idx], Style: Style{FG: color}})
			}
		},
	)

	// Animated rainbow bar
	rainbowPhase := 0.0
	rainbowBar := Widget(
		func(availW int16) (w, h int16) {
			return availW, 1
		},
		func(buf *Buffer, x, y, w, h int16) {
			for i := int16(0); i < w; i++ {
				// cycle through hue
				hue := math.Mod(rainbowPhase+float64(i)*0.05, 1.0)
				color := hueToRGB(hue)
				buf.Set(int(x+i), int(y), Cell{Rune: '█', Style: Style{FG: color}})
			}
		},
	)

	// Mini bar chart with gradient
	barData := []float64{0.3, 0.7, 0.5, 0.9, 0.4, 0.8, 0.6}
	barChart := Widget(
		func(availW int16) (w, h int16) {
			return int16(len(barData) * 3), 5 // 3 chars per bar, 5 tall
		},
		func(buf *Buffer, x, y, w, h int16) {
			barLow := RGB(50, 150, 50)
			barHigh := RGB(50, 255, 50)

			for i, v := range barData {
				barHeight := int(v * float64(h))
				barX := int(x) + i*3

				for row := 0; row < int(h); row++ {
					cellY := int(y) + int(h) - 1 - row
					if row < barHeight {
						t := float64(row) / float64(h-1)
						color := LerpColor(barLow, barHigh, t)
						buf.Set(barX, cellY, Cell{Rune: '█', Style: Style{FG: color}})
						buf.Set(barX+1, cellY, Cell{Rune: '█', Style: Style{FG: color}})
					}
				}
			}
		},
	)

	app.SetView(VBox.Gap(1)(
		Text("Widget Demo - Smooth Gradients").FG(Cyan).Bold(),
		HRule().Style(Style{FG: PaletteColor(238)}),

		Text("Smooth Gradient Progress (j/k to adjust):").FG(BrightWhite),
		gradientBar,

		Text("Animated Rainbow:").FG(BrightWhite),
		rainbowBar,

		Text("Sine Wave with Color Gradient:").FG(BrightWhite),
		sineWave,

		HBox.Gap(6)(
			VBox(
				Text("Sparkline:").FG(BrightWhite),
				sparkline,
			),
			VBox(
				Text("Bar Chart:").FG(BrightWhite),
				barChart,
			),
		),

		HRule().Style(Style{FG: PaletteColor(238)}),
		Text("j/k: progress | q: quit").FG(PaletteColor(245)),
	))

	// Animation ticker
	go func() {
		ticker := time.NewTicker(16 * time.Millisecond) // ~60fps
		defer ticker.Stop()
		for range ticker.C {
			wavePhase += 0.08
			rainbowPhase += 0.02
			app.RequestRender()
		}
	}()

	app.Handle("j", func(_ riffkey.Match) {
		progress -= 0.01 // finer control to show off sub-char precision
		if progress < 0 {
			progress = 0
		}
	})
	app.Handle("k", func(_ riffkey.Match) {
		progress += 0.01
		if progress > 1 {
			progress = 1
		}
	})
	app.Handle("q", func(_ riffkey.Match) { app.Stop() })
	app.Handle("<Escape>", func(_ riffkey.Match) { app.Stop() })

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// brailleBit returns the bit position for a dot in a braille character
// row: 0-3, col: 0-1 (left/right)
// braille layout:  0 3
//                  1 4
//                  2 5
//                  6 7
func brailleBit(row, col int) int {
	if col == 0 {
		if row < 3 {
			return row
		}
		return 6
	}
	if row < 3 {
		return row + 3
	}
	return 7
}

// hueToRGB converts a hue (0-1) to RGB color (saturation=1, lightness=0.5)
func hueToRGB(h float64) Color {
	h = math.Mod(h, 1.0)
	if h < 0 {
		h += 1
	}

	var r, g, b float64
	switch {
	case h < 1.0/6.0:
		r, g, b = 1, h*6, 0
	case h < 2.0/6.0:
		r, g, b = 1-(h-1.0/6.0)*6, 1, 0
	case h < 3.0/6.0:
		r, g, b = 0, 1, (h-2.0/6.0)*6
	case h < 4.0/6.0:
		r, g, b = 0, 1-(h-3.0/6.0)*6, 1
	case h < 5.0/6.0:
		r, g, b = (h-4.0/6.0)*6, 0, 1
	default:
		r, g, b = 1, 0, 1-(h-5.0/6.0)*6
	}

	return RGB(uint8(r*255), uint8(g*255), uint8(b*255))
}
