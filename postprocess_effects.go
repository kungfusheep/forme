package glyph

import "math"

// ---------------------------------------------------------------------------
// Subtle: one-liner polish for real apps
// ---------------------------------------------------------------------------

// PPDimAll applies the dim attribute to every cell on screen.
// Simple building block for focus patterns and background dimming.
func PPDimAll() PostProcess {
	return EachCell(func(_, _ int, c Cell, _ PostContext) Cell {
		c.Style.Attr = c.Style.Attr.With(AttrDim)
		return c
	})
}

// PPTint shifts all RGB colours toward a target colour.
// amount 0.0 = no change, 1.0 = fully tinted.
// Think colour grading: warm/cool/moody tones in one line.
func PPTint(target Color, amount float64) PostProcess {
	return EachCell(func(_, _ int, c Cell, ctx PostContext) Cell {
		c.Style.FG = lerpIfRGB(resolveFG(c.Style.FG, ctx), target, amount)
		c.Style.BG = lerpIfRGB(resolveBG(c.Style.BG, ctx), target, amount)
		return c
	})
}

// PPVignette darkens cells near the screen edges.
// strength 0.0 = no effect, 1.0 = full darkening at edges.
// Quadratic falloff for a natural cinematic feel.
func PPVignette(strength float64) PostProcess {
	black := Color{Mode: ColorRGB}
	return func(buf *Buffer, ctx PostContext) {
		cx := float64(ctx.Width) / 2
		cy := float64(ctx.Height) / 2
		// terminal cells are ~2:1 aspect ratio, compensate for circular vignette
		maxDist := math.Sqrt(cx*cx + cy*cy*4)

		for y := range ctx.Height {
			base := y * buf.width
			dy := (float64(y) - cy) * 2
			for x := range ctx.Width {
				dx := float64(x) - cx
				dist := math.Sqrt(dx*dx+dy*dy) / maxDist
				dim := dist * dist * strength
				if dim > 1 {
					dim = 1
				}
				idx := base + x
				c := &buf.cells[idx]
				c.Style.FG = lerpIfRGB(resolveFG(c.Style.FG, ctx), black, dim)
				c.Style.BG = lerpIfRGB(resolveBG(c.Style.BG, ctx), black, dim)
			}
		}
	}
}

// PPDesaturate removes colour saturation from all RGB cells.
// amount 0.0 = full colour, 1.0 = fully monochrome.
// Uses perceptual luminance weights (BT.601).
func PPDesaturate(amount float64) PostProcess {
	return EachCell(func(_, _ int, c Cell, ctx PostContext) Cell {
		c.Style.FG = desaturateColor(resolveFG(c.Style.FG, ctx), amount)
		c.Style.BG = desaturateColor(resolveBG(c.Style.BG, ctx), amount)
		return c
	})
}

// PPFadeRows fades the top and bottom N rows toward a colour.
// Creates a natural scroll-hint or edge-softening effect.
func PPFadeRows(topRows, bottomRows int, target Color) PostProcess {
	return func(buf *Buffer, ctx PostContext) {
		for y := range ctx.Height {
			var t float64
			if topRows > 0 && y < topRows {
				t = 1.0 - float64(y)/float64(topRows)
			} else if bottomRows > 0 && y >= ctx.Height-bottomRows {
				t = float64(y-(ctx.Height-bottomRows)+1) / float64(bottomRows)
			} else {
				continue
			}

			base := y * buf.width
			for x := range ctx.Width {
				idx := base + x
				c := &buf.cells[idx]
				c.Style.FG = lerpIfRGB(resolveFG(c.Style.FG, ctx), target, t)
				c.Style.BG = lerpIfRGB(resolveBG(c.Style.BG, ctx), target, t)
			}
		}
	}
}

// PPHighContrast boosts contrast by pushing colour channels toward extremes.
// amount 1.0 = noticeable punch, 3.0+ = stark black/white.
// Useful for accessibility or just making content pop.
func PPHighContrast(amount float64) PostProcess {
	return EachCell(func(_, _ int, c Cell, ctx PostContext) Cell {
		c.Style.FG = boostContrast(resolveFG(c.Style.FG, ctx), amount)
		c.Style.BG = boostContrast(resolveBG(c.Style.BG, ctx), amount)
		return c
	})
}

// ---------------------------------------------------------------------------
// Medium: noticeable, purposeful
// ---------------------------------------------------------------------------

// PPFocusDim dims everything outside a rectangular focus area.
// Uses pointers for reactivity. Move the focus rect, next frame adapts.
func PPFocusDim(fx, fy, fw, fh *int) PostProcess {
	return func(buf *Buffer, ctx PostContext) {
		rx, ry := *fx, *fy
		rw, rh := *fw, *fh

		for y := range ctx.Height {
			base := y * buf.width
			inY := y >= ry && y < ry+rh
			for x := range ctx.Width {
				if inY && x >= rx && x < rx+rw {
					continue
				}
				buf.cells[base+x].Style.Attr = buf.cells[base+x].Style.Attr.With(AttrDim)
			}
		}
	}
}

// PPFrost replaces characters with shade blocks and tints toward a frosted
// white. Simulates frosted glass, ideal behind modals and overlays.
// opacity 0.0 = clear, 1.0 = fully frosted.
func PPFrost(opacity float64) PostProcess {
	shades := []rune{'░', '▒', '▓'}
	frost := RGB(180, 190, 210) // cool blue-white

	return func(buf *Buffer, ctx PostContext) {
		for y := range ctx.Height {
			base := y * buf.width
			for x := range ctx.Width {
				idx := base + x
				c := &buf.cells[idx]

				// replace visible runes with shade blocks for blur texture
				if c.Rune != ' ' && c.Rune != 0 {
					h := uint64(y*ctx.Width+x) * 2654435761
					c.Rune = shades[h%uint64(len(shades))]
				}

				c.Style.FG = desaturateColor(resolveFG(c.Style.FG, ctx), opacity*0.6)
				c.Style.FG = lerpIfRGB(c.Style.FG, frost, opacity*0.5)
				c.Style.BG = desaturateColor(resolveBG(c.Style.BG, ctx), opacity*0.4)
				c.Style.BG = lerpIfRGB(c.Style.BG, frost, opacity*0.3)
			}
		}
	}
}

// PPPulse oscillates screen brightness over time using a sine wave.
// speed controls frequency (1.0 = one cycle per second).
// amount controls how much it dims at the trough (0.3 = subtle, 0.8 = dramatic).
func PPPulse(speed, amount float64) PostProcess {
	black := Color{Mode: ColorRGB}
	return func(buf *Buffer, ctx PostContext) {
		// sine 0..1
		t := (math.Sin(ctx.Time.Seconds()*speed*math.Pi*2) + 1) * 0.5
		dim := t * amount

		for y := range ctx.Height {
			base := y * buf.width
			for x := range ctx.Width {
				idx := base + x
				c := &buf.cells[idx]
				c.Style.FG = lerpIfRGB(resolveFG(c.Style.FG, ctx), black, dim)
				c.Style.BG = lerpIfRGB(resolveBG(c.Style.BG, ctx), black, dim)
			}
		}
	}
}

// PPGradientMap remaps all colour luminance through a three-stop gradient.
// Computes perceptual luminance, then maps dark→mid→bright.
// Use for dramatic colour themes: cyberpunk, matrix, sunset, etc.
func PPGradientMap(dark, mid, bright Color) PostProcess {
	return EachCell(func(_, _ int, c Cell, ctx PostContext) Cell {
		c.Style.FG = gradientMap(resolveFG(c.Style.FG, ctx), dark, mid, bright)
		c.Style.BG = gradientMap(resolveBG(c.Style.BG, ctx), dark, mid, bright)
		return c
	})
}

// ---------------------------------------------------------------------------
// Visual flair
// ---------------------------------------------------------------------------

// PPBloom creates a coloured glow around bright cells.
// radius controls spread in cells (2-4 recommended).
// threshold 0.0-1.0 sets minimum brightness that blooms.
// intensity controls glow strength (0.3 = subtle, 1.0 = vivid).
// Bleeds bright colours into both FG and BG of surrounding cells.
func PPBloom(radius int, threshold, intensity float64) PostProcess {
	return func(buf *Buffer, ctx PostContext) {
		bw, bh := ctx.Width, ctx.Height
		// snapshot FG colours (avoids read-after-write)
		snap := make([]Color, bw*bh)
		for y := range bh {
			bufBase := y * buf.width
			snapBase := y * bw
			for x := range bw {
				snap[snapBase+x] = resolveFG(buf.cells[bufBase+x].Style.FG, ctx)
			}
		}

		thresh256 := threshold * 255
		maxDist := math.Sqrt(float64(radius*radius) + float64(radius*radius)*4)

		for y := range bh {
			base := y * buf.width
			for x := range bw {
				var sumR, sumG, sumB, sumWt float64

				for dy := -radius; dy <= radius; dy++ {
					ny := y + dy
					if ny < 0 || ny >= bh {
						continue
					}
					for dx := -radius; dx <= radius; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						nx := x + dx
						if nx < 0 || nx >= bw {
							continue
						}
						nc := snap[ny*bw+nx]
						lum := 0.299*float64(nc.R) + 0.587*float64(nc.G) + 0.114*float64(nc.B)
						if lum <= thresh256 {
							continue
						}

						// quadratic falloff, aspect-ratio compensated
						dist := math.Sqrt(float64(dx*dx) + float64(dy*dy)*4)
						falloff := 1.0 - dist/maxDist
						if falloff <= 0 {
							continue
						}
						falloff *= falloff

						excess := (lum - thresh256) / (255 - thresh256)
						wt := falloff * excess
						sumR += float64(nc.R) * wt
						sumG += float64(nc.G) * wt
						sumB += float64(nc.B) * wt
						sumWt += wt
					}
				}

				if sumWt > 0 {
					bloom := RGB(
						uint8(min(255, sumR/sumWt)),
						uint8(min(255, sumG/sumWt)),
						uint8(min(255, sumB/sumWt)),
					)
					blend := min(1.0, sumWt) * intensity
					c := &buf.cells[base+x]
					c.Style.FG = lerpIfRGB(resolveFG(c.Style.FG, ctx), bloom, blend)
					c.Style.BG = lerpIfRGB(resolveBG(c.Style.BG, ctx), bloom, blend*0.3)
				}
			}
		}
	}
}

// PPDropShadow projects a shadow behind non-empty cells.
// offsetX/offsetY control direction (1,1 = light from top-left).
// opacity controls shadow darkness (0.3 = subtle, 1.0 = solid).
func PPDropShadow(offsetX, offsetY int, opacity float64) PostProcess {
	shadow := Color{Mode: ColorRGB}
	return func(buf *Buffer, ctx PostContext) {
		w, h := ctx.Width, ctx.Height
		// snapshot which cells have visible content (non-space runes)
		occupied := make([]bool, w*h)
		for y := range h {
			base := y * buf.width
			for x := range w {
				r := buf.cells[base+x].Rune
				occupied[y*w+x] = r != ' ' && r != 0
			}
		}

		for y := range h {
			base := y * buf.width
			for x := range w {
				if occupied[y*w+x] {
					continue
				}
				srcX, srcY := x-offsetX, y-offsetY
				if srcX < 0 || srcX >= w || srcY < 0 || srcY >= h {
					continue
				}
				if !occupied[srcY*w+srcX] {
					continue
				}
				c := &buf.cells[base+x]
				c.Style.BG = LerpColor(resolveBG(c.Style.BG, ctx), shadow, opacity)
			}
		}
	}
}

// PPCRT simulates a CRT monitor with scanlines, vignette, and warm phosphor
// tint in a single pass.
func PPCRT() PostProcess {
	black := Color{Mode: ColorRGB}
	warm := RGB(255, 200, 150)
	return func(buf *Buffer, ctx PostContext) {
		cx := float64(ctx.Width) / 2
		cy := float64(ctx.Height) / 2
		maxDist := math.Sqrt(cx*cx + cy*cy*4)

		for y := range ctx.Height {
			base := y * buf.width
			dy := (float64(y) - cy) * 2

			scanDim := 0.0
			if y%2 == 1 {
				scanDim = 0.3
			}

			for x := range ctx.Width {
				idx := base + x
				c := &buf.cells[idx]

				// resolve defaults once, chain the rest
				fg := resolveFG(c.Style.FG, ctx)
				bg := resolveBG(c.Style.BG, ctx)

				if scanDim > 0 {
					fg = lerpIfRGB(fg, black, scanDim)
					bg = lerpIfRGB(bg, black, scanDim)
				}

				dx := float64(x) - cx
				dist := math.Sqrt(dx*dx+dy*dy) / maxDist
				dim := dist * dist * 0.6
				if dim > 1 {
					dim = 1
				}
				fg = lerpIfRGB(fg, black, dim)
				bg = lerpIfRGB(bg, black, dim)

				c.Style.FG = lerpIfRGB(fg, warm, 0.1)
				c.Style.BG = lerpIfRGB(bg, warm, 0.1)
			}
		}
	}
}

// PPMonochrome converts all colours to a single-tint monochrome.
// Pass RGB(0, 255, 0) for green phosphor, RGB(255, 180, 0) for amber.
func PPMonochrome(tint Color) PostProcess {
	return EachCell(func(_, _ int, c Cell, ctx PostContext) Cell {
		c.Style.FG = monochromeColor(resolveFG(c.Style.FG, ctx), tint)
		c.Style.BG = monochromeColor(resolveBG(c.Style.BG, ctx), tint)
		return c
	})
}

// ---------------------------------------------------------------------------
// Demoscene: animated effects that use ctx.Time
// ---------------------------------------------------------------------------

// PPPlasma overlays a classic 4-oscillator sine plasma on cell colours.
// intensity 0.0 = no effect, 1.0 = full plasma replacement.
func PPPlasma(intensity float64) PostProcess {
	bgIntensity := intensity * 0.4
	return func(buf *Buffer, ctx PostContext) {
		t := ctx.Time.Seconds()
		EachCell(func(x, y int, c Cell, ectx PostContext) Cell {
			fx := float64(x)
			fy := float64(y) * 2 // aspect ratio compensation
			v := math.Sin(fx/8 + t)
			v += math.Sin(fy/4 + t*1.3)
			v += math.Sin((fx+fy)/10 + t*0.7)
			v += math.Sin(math.Sqrt(fx*fx+fy*fy)/6 + t*1.5)
			v = (v + 4) / 8 // normalize 0..1

			r := uint8(128 + 127*math.Sin(v*math.Pi*2))
			g := uint8(128 + 127*math.Sin(v*math.Pi*2+2.094))
			b := uint8(128 + 127*math.Sin(v*math.Pi*2+4.189))
			plasma := RGB(r, g, b)

			c.Style.FG = lerpIfRGB(resolveFG(c.Style.FG, ectx), plasma, intensity)
			c.Style.BG = lerpIfRGB(resolveBG(c.Style.BG, ectx), plasma, bgIntensity)
			return c
		})(buf, ctx)
	}
}

// matrixGlyphs contains half-width Katakana, select Kanji, digits, and Latin
// fragments, the character mix seen in the Matrix films.
var matrixGlyphs = []rune{
	// half-width katakana (U+FF66 – U+FF9D)
	'ｦ', 'ｧ', 'ｨ', 'ｩ', 'ｪ', 'ｫ', 'ｬ', 'ｭ', 'ｮ', 'ｯ',
	'ｰ', 'ｱ', 'ｲ', 'ｳ', 'ｴ', 'ｵ', 'ｶ', 'ｷ', 'ｸ', 'ｹ',
	'ｺ', 'ｻ', 'ｼ', 'ｽ', 'ｾ', 'ｿ', 'ﾀ', 'ﾁ', 'ﾂ', 'ﾃ',
	'ﾄ', 'ﾅ', 'ﾆ', 'ﾇ', 'ﾈ', 'ﾉ', 'ﾊ', 'ﾋ', 'ﾌ', 'ﾍ',
	'ﾎ', 'ﾏ', 'ﾐ', 'ﾑ', 'ﾒ', 'ﾓ', 'ﾔ', 'ﾕ', 'ﾖ', 'ﾗ',
	'ﾘ', 'ﾙ', 'ﾚ', 'ﾛ', 'ﾜ', 'ﾝ',
	// digits and symbols (mirrored/stylised as in the films)
	'0', '1', '2', '3', '4', '5', '7', '8', '9',
	':', '.', '"', '=', '*', '+', '-', '<', '>',
	'¦', '|',
}

// PPMatrix overlays falling green code rain using half-width Katakana and
// symbols to replicate the look of the Matrix films.
// density controls how many drops per column (1 = sparse, 3 = dense).
// Uses sparse column iteration, only touches cells within drop trails.
func PPMatrix(density int) PostProcess {
	if density < 1 {
		density = 1
	}
	nGlyphs := uint64(len(matrixGlyphs))
	return func(buf *Buffer, ctx PostContext) {
		w, h := ctx.Width, ctx.Height
		t := ctx.Time.Seconds()

		for x := range w {
			for drop := range density {
				hash := uint64(x*7+drop+1) * 2654435761
				speed := 3.0 + float64(hash%10)
				offset := float64(hash % uint64(h*3))
				tailLen := 5 + int(hash%12)

				headY := int(offset+t*speed) % (h + tailLen + 5)

				for dy := range tailLen {
					cy := headY - dy
					if cy < 0 || cy >= h {
						continue
					}

					fade := 1.0 - float64(dy)/float64(tailLen)
					base := cy * buf.width
					c := &buf.cells[base+x]

					// deterministic per-cell glyph with time-based mutation;
					// head cell and every ~3rd tail cell flicker faster
					charHash := uint64(cy*w+x) * 2654435761
					flickerRate := uint64(t * 8)
					if dy == 0 || charHash%3 == 0 {
						flickerRate = uint64(t * 20)
					}
					c.Rune = matrixGlyphs[(charHash+flickerRate)%nGlyphs]

					switch {
					case dy == 0:
						c.Style.FG = RGB(200, 255, 200)
					case dy == 1:
						c.Style.FG = RGB(100, 255, 100)
					default:
						c.Style.FG = RGB(0, uint8(20+float64(200)*fade), 0)
					}
					c.Style.BG = Color{Mode: ColorRGB}
				}
			}
		}
	}
}

// PPFire simulates a rising fire effect using cellular automaton heat propagation.
// The fire burns from the bottom of the screen upward, overlaying content with
// fire colours and shade block characters.
func PPFire() PostProcess {
	var heatBuf []float64
	prevW, prevH := 0, 0
	fireChars := []rune{'░', '▒', '▓', '█'}

	return func(buf *Buffer, ctx PostContext) {
		w, h := ctx.Width, ctx.Height

		if w != prevW || h != prevH {
			heatBuf = make([]float64, w*h)
			prevW, prevH = w, h
		}

		// seed bottom row
		seed := uint64(ctx.Time.Seconds() * 60)
		for x := range w {
			hash := (uint64(x) + seed) * 2654435761
			if hash%3 == 0 {
				heatBuf[(h-1)*w+x] = 0.8 + float64(hash%200)/1000
			} else {
				heatBuf[(h-1)*w+x] *= 0.3
			}
		}

		// propagate upward (top-to-bottom reads old values from below)
		for y := range h - 1 {
			for x := range w {
				below := heatBuf[(y+1)*w+x]
				left := below
				right := below
				if x > 0 {
					left = heatBuf[(y+1)*w+x-1]
				}
				if x < w-1 {
					right = heatBuf[(y+1)*w+x+1]
				}
				var below2 float64
				if y+2 < h {
					below2 = heatBuf[(y+2)*w+x]
				}

				avg := (below + left + right + below2) / 4.0
				cool := 0.04 + 0.02*float64(y)/float64(h)
				heatBuf[y*w+x] = max(0, avg-cool)
			}
		}

		// render fire overlay
		for y := range h {
			base := y * buf.width
			for x := range w {
				heat := heatBuf[y*w+x]
				if heat < 0.01 {
					continue
				}

				var col Color
				switch {
				case heat < 0.25:
					f := heat / 0.25
					col = RGB(uint8(200*f), 0, 0)
				case heat < 0.5:
					f := (heat - 0.25) / 0.25
					col = RGB(200+uint8(55*f), uint8(150*f), 0)
				case heat < 0.75:
					f := (heat - 0.5) / 0.25
					col = RGB(255, 150+uint8(105*f), uint8(80*f))
				default:
					f := (heat - 0.75) / 0.25
					col = RGB(255, 255, 80+uint8(175*f))
				}

				c := &buf.cells[base+x]
				ci := int(heat * float64(len(fireChars)-1))
				if ci >= len(fireChars) {
					ci = len(fireChars) - 1
				}
				c.Rune = fireChars[ci]
				c.Style.FG = col
				c.Style.BG = lerpIfRGB(resolveBG(c.Style.BG, ctx), col, heat*0.4)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Transitions & kinetic effects (require animation system for best results)
// ---------------------------------------------------------------------------

// PPDissolve randomly hides cells based on progress.
// progress 0.0 = fully visible, 1.0 = fully dissolved.
// Deterministic per-cell, same cells always dissolve at the same progress point.
func PPDissolve(progress *float64) PostProcess {
	return func(buf *Buffer, ctx PostContext) {
		p := *progress
		if p <= 0 {
			return
		}
		empty := EmptyCell()
		for y := range ctx.Height {
			base := y * buf.width
			for x := range ctx.Width {
				cellHash := uint64(y*ctx.Width+x) * 2654435761
				threshold := float64(cellHash%1000) / 1000.0
				if threshold < p {
					buf.cells[base+x] = empty
				}
			}
		}
	}
}

// PPScreenShake displaces the entire buffer horizontally with a sine wave.
// amplitude controls max displacement in cells.
// Oscillates each frame for a continuous shake. Set amplitude via pointer
// and decay it yourself for a triggered one-shot shake.
func PPScreenShake(amplitude float64) PostProcess {
	return func(buf *Buffer, ctx PostContext) {
		offset := int(math.Round(math.Sin(float64(ctx.Frame)*1.5) * amplitude))
		if offset == 0 {
			return
		}

		empty := EmptyCell()
		for y := range ctx.Height {
			base := y * buf.width
			if offset > 0 {
				for x := ctx.Width - 1; x >= 0; x-- {
					if srcX := x - offset; srcX >= 0 {
						buf.cells[base+x] = buf.cells[base+srcX]
					} else {
						buf.cells[base+x] = empty
					}
				}
			} else {
				for x := range ctx.Width {
					if srcX := x - offset; srcX < ctx.Width {
						buf.cells[base+x] = buf.cells[base+srcX]
					} else {
						buf.cells[base+x] = empty
					}
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// resolveFG returns a colour with RGB values populated. For ColorDefault,
// uses the terminal's detected default FG if available.
func resolveFG(c Color, ctx PostContext) Color {
	if c.Mode == ColorDefault && ctx.DefaultFG.Mode != ColorDefault {
		return ctx.DefaultFG
	}
	return c
}

// resolveBG returns a colour with RGB values populated. For ColorDefault,
// uses the terminal's detected default BG if available.
func resolveBG(c Color, ctx PostContext) Color {
	if c.Mode == ColorDefault && ctx.DefaultBG.Mode != ColorDefault {
		return ctx.DefaultBG
	}
	return c
}

func lerpIfRGB(c, target Color, t float64) Color {
	if c.Mode == ColorDefault {
		return c
	}
	return LerpColor(c, target, t)
}

func desaturateColor(c Color, amount float64) Color {
	if c.Mode == ColorDefault {
		return c
	}
	gray := uint8(0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B))
	return LerpColor(c, RGB(gray, gray, gray), amount)
}

func boostContrast(c Color, amount float64) Color {
	if c.Mode == ColorDefault {
		return c
	}
	return RGB(
		contrastChannel(c.R, amount),
		contrastChannel(c.G, amount),
		contrastChannel(c.B, amount),
	)
}

func contrastChannel(v uint8, amount float64) uint8 {
	f := (float64(v)/255.0-0.5)*(1.0+amount) + 0.5
	if f < 0 {
		f = 0
	} else if f > 1 {
		f = 1
	}
	return uint8(f * 255)
}

func monochromeColor(c, tint Color) Color {
	if c.Mode == ColorDefault {
		return c
	}
	lum := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
	return RGB(
		uint8(lum*float64(tint.R)/255),
		uint8(lum*float64(tint.G)/255),
		uint8(lum*float64(tint.B)/255),
	)
}

func gradientMap(c, dark, mid, bright Color) Color {
	if c.Mode == ColorDefault {
		return c
	}
	lum := (0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)) / 255.0
	if lum < 0.5 {
		return LerpColor(dark, mid, lum*2)
	}
	return LerpColor(mid, bright, (lum-0.5)*2)
}
