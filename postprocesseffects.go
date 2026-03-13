package glyph

import "math"

// dynFloat64 bundles a static float64 with optional dynamic source (pointer,
// condition, or tween). replaces the repeated 3-field pattern across effects.
type dynFloat64 struct {
	val float64
	dyn any
	ptr *float64
}

func (d *dynFloat64) set(v any) {
	switch val := v.(type) {
	case float64:
		d.val = val
	case float32:
		d.val = float64(val)
	case int:
		d.val = float64(val)
	case *float64:
		d.dyn = val
	case conditionNode:
		d.dyn = val
	case tweenNode:
		d.dyn = val
	}
}

func (d *dynFloat64) compile(tmpl *Template) {
	if d.dyn != nil {
		d.ptr = tmpl.compileDynFloat64(d.dyn)
	}
}

func (d dynFloat64) resolve() float64 {
	if d.ptr != nil {
		return *d.ptr
	}
	return d.val
}

// dynInt bundles a static int with optional dynamic source.
type dynInt struct {
	val int
	dyn any
	ptr *int16
}

func (d *dynInt) set(v any) {
	switch val := v.(type) {
	case int:
		d.val = val
	case int16:
		d.val = int(val)
	case *int16:
		d.dyn = val
	case conditionNode:
		d.dyn = val
	case tweenNode:
		d.dyn = val
	}
}

func (d *dynInt) compile(tmpl *Template) {
	if d.dyn != nil {
		d.ptr = tmpl.compileDynInt16(d.dyn)
	}
}

func (d dynInt) resolve() int {
	if d.ptr != nil {
		return int(*d.ptr)
	}
	return d.val
}

// dynColor bundles a static Color with optional dynamic source.
type dynColor struct {
	val Color
	dyn any
	ptr *Color
}

func (d *dynColor) set(v any) {
	switch val := v.(type) {
	case Color:
		d.val = val
	case *Color:
		d.dyn = val
	case conditionNode:
		d.dyn = val
	case tweenNode:
		d.dyn = val
	}
}

func (d *dynColor) compile(tmpl *Template) {
	if d.dyn != nil {
		d.ptr = tmpl.compileDynColor(d.dyn)
	}
}

func (d dynColor) resolve() Color {
	if d.ptr != nil {
		return *d.ptr
	}
	return d.val
}

// ---------------------------------------------------------------------------
// Subtle: one-liner polish for real apps
// ---------------------------------------------------------------------------

// SEDimAll applies the terminal Dim attribute to every cell.
// The simplest possible effect — one attribute, whole screen.
func SEDimAll() Effect {
	return EachCell(func(_, _ int, c Cell, _ PostContext) Cell {
		c.Style.Attr = c.Style.Attr.With(AttrDim)
		return c
	})
}

// tintEffect shifts all RGB colours toward a target colour.
type tintEffect struct {
	target   Color
	strength dynFloat64
	dodge    *NodeRef
}

// SETint shifts all RGB colours toward a target colour.
// Think colour grading: warm/cool/moody tones in one line.
// Default strength 0.15 — tasteful tint out of the box.
func SETint(color Color) tintEffect {
	return tintEffect{target: color, strength: dynFloat64{val: 0.15}}
}

// Strength sets how strongly the tint blends in (0.0 = none, 1.0 = full).
func (t tintEffect) Strength(s any) tintEffect { t.strength.set(s); return t }

// Dodge exempts the given node from tinting — useful for preserving a focused panel.
func (t tintEffect) Dodge(ref *NodeRef) tintEffect { t.dodge = ref; return t }

func (t tintEffect) compileEffect(tmpl *Template) Effect {
	t.strength.compile(tmpl)
	return t
}

func (t tintEffect) Apply(buf *Buffer, ctx PostContext) {
	s := t.strength.resolve()
	EachCell(func(x, y int, c Cell, ectx PostContext) Cell {
		if t.dodge != nil && inRect(x, y, t.dodge) {
			return c
		}
		c.Style.FG = lerpIfRGB(resolveFG(c.Style.FG, ectx), t.target, s)
		c.Style.BG = lerpIfRGB(resolveBG(c.Style.BG, ectx), t.target, s)
		return c
	}).Apply(buf, ctx)
}

// vignetteEffect darkens cells toward the screen edges.
type vignetteEffect struct {
	strength dynFloat64
	focus    *NodeRef
	dodge    *NodeRef
	quantize bool
}

// SEVignette darkens cells near the screen edges.
// Quadratic falloff for a natural cinematic feel. Default strength 0.8.
func SEVignette() vignetteEffect {
	return vignetteEffect{strength: dynFloat64{val: 0.8}, quantize: true}
}

// Strength sets edge darkening intensity (0.0 = no effect, 1.0 = full black at edges).
func (v vignetteEffect) Strength(s any) vignetteEffect { v.strength.set(s); return v }

func (v vignetteEffect) compileEffect(tmpl *Template) Effect {
	v.strength.compile(tmpl)
	return v
}

// Focus centres the vignette on the given node.
func (v vignetteEffect) Focus(ref *NodeRef) vignetteEffect { v.focus = ref; return v }

// Dodge exempts the given node from darkening.
func (v vignetteEffect) Dodge(ref *NodeRef) vignetteEffect { v.dodge = ref; return v }

// Smooth disables quantization for a continuous gradient (slightly more escape output).
func (v vignetteEffect) Smooth() vignetteEffect { v.quantize = false; return v }

func (v vignetteEffect) Apply(buf *Buffer, ctx PostContext) {
	black := Color{Mode: ColorRGB}
	var cx, cy float64
	if v.focus != nil {
		cx = float64(v.focus.X) + float64(v.focus.W)/2
		cy = float64(v.focus.Y) + float64(v.focus.H)/2
	} else {
		cx = float64(ctx.Width) / 2
		cy = float64(ctx.Height) / 2
	}
	// maxDist = distance from center to the farthest screen corner, aspect-compensated.
	// using max extents handles off-center focus nodes correctly.
	maxX := math.Max(cx, float64(ctx.Width)-cx)
	maxY := math.Max(cy, float64(ctx.Height)-cy) * 2
	maxDist := math.Sqrt(maxX*maxX + maxY*maxY)

	for y := range ctx.Height {
		base := y * buf.width
		dy := (float64(y) - cy) * 2
		for x := range ctx.Width {
			if v.dodge != nil && inRect(x, y, v.dodge) {
				continue
			}
			dx := float64(x) - cx
			dist := math.Sqrt(dx*dx+dy*dy) / maxDist
			dim := dist * dist * v.strength.resolve()
			if dim > 1 {
				dim = 1
			}
			// snap to 32 levels — imperceptible banding, collapses escape output
			if v.quantize {
				dim = math.Round(dim*32) / 32
			}
			idx := base + x
			c := &buf.cells[idx]
			c.Style.FG = lerpIfRGB(resolveFG(c.Style.FG, ctx), black, dim)
			c.Style.BG = lerpIfRGB(resolveBG(c.Style.BG, ctx), black, dim)
		}
	}
}

// desaturateEffect removes colour saturation from all RGB cells.
type desaturateEffect struct {
	strength dynFloat64
	dodge    *NodeRef
}

// SEDesaturate removes colour saturation from all RGB cells.
// Uses perceptual luminance weights (BT.601). Default strength 0.7.
func SEDesaturate() desaturateEffect { return desaturateEffect{strength: dynFloat64{val: 0.7}} }

// Strength sets how much to desaturate (0.0 = full colour, 1.0 = fully grey).
func (d desaturateEffect) Strength(s any) desaturateEffect { d.strength.set(s); return d }

func (d desaturateEffect) compileEffect(tmpl *Template) Effect {
	d.strength.compile(tmpl)
	return d
}

// Dodge exempts the given node — the classic "colour spotlight" on a grey world.
func (d desaturateEffect) Dodge(ref *NodeRef) desaturateEffect { d.dodge = ref; return d }

func (d desaturateEffect) Apply(buf *Buffer, ctx PostContext) {
	s := d.strength.resolve()
	EachCell(func(x, y int, c Cell, ectx PostContext) Cell {
		if d.dodge != nil && inRect(x, y, d.dodge) {
			return c
		}
		c.Style.FG = desaturateColor(resolveFG(c.Style.FG, ectx), s)
		c.Style.BG = desaturateColor(resolveBG(c.Style.BG, ectx), s)
		return c
	}).Apply(buf, ctx)
}

// contrastEffect boosts contrast by pushing colour channels toward extremes.
type contrastEffect struct {
	strength dynFloat64
	dodge    *NodeRef
}

// SEContrast boosts contrast by pushing colour channels toward extremes.
// Default strength 1.5 — noticeable punch without going stark.
func SEContrast() contrastEffect { return contrastEffect{strength: dynFloat64{val: 1.5}} }

// Strength sets the contrast boost factor (1.0 = noticeable, 3.0+ = stark black/white).
func (h contrastEffect) Strength(s any) contrastEffect { h.strength.set(s); return h }

func (h contrastEffect) compileEffect(tmpl *Template) Effect {
	h.strength.compile(tmpl)
	return h
}

// Dodge exempts the given node from contrast adjustment.
func (h contrastEffect) Dodge(ref *NodeRef) contrastEffect { h.dodge = ref; return h }

func (h contrastEffect) Apply(buf *Buffer, ctx PostContext) {
	s := h.strength.resolve()
	EachCell(func(x, y int, c Cell, ectx PostContext) Cell {
		if h.dodge != nil && inRect(x, y, h.dodge) {
			return c
		}
		c.Style.FG = boostContrast(resolveFG(c.Style.FG, ectx), s)
		c.Style.BG = boostContrast(resolveBG(c.Style.BG, ectx), s)
		return c
	}).Apply(buf, ctx)
}

// ---------------------------------------------------------------------------
// Medium: noticeable, purposeful
// ---------------------------------------------------------------------------

// focusDimEffect dims everything outside the bounds of a NodeRef.
type focusDimEffect struct{ ref *NodeRef }

// SEFocusDim dims everything outside the bounds of a NodeRef.
// The ref is populated each frame after layout, so it tracks the node automatically.
func SEFocusDim(ref *NodeRef) focusDimEffect { return focusDimEffect{ref: ref} }

func (f focusDimEffect) Apply(buf *Buffer, ctx PostContext) {
	rx, ry := f.ref.X, f.ref.Y
	rw, rh := f.ref.W, f.ref.H

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

type pulseEffect struct {
	speed    dynFloat64
	strength dynFloat64
}

func SEPulse() pulseEffect {
	return pulseEffect{speed: dynFloat64{val: 1.0}, strength: dynFloat64{val: 0.3}}
}

// Speed sets oscillation frequency in cycles per second.
func (p pulseEffect) Speed(s any) pulseEffect { p.speed.set(s); return p }

// Strength sets how much brightness dims at the trough (0.3 = subtle, 0.8 = dramatic).
func (p pulseEffect) Strength(s any) pulseEffect { p.strength.set(s); return p }

func (p pulseEffect) compileEffect(tmpl *Template) Effect {
	p.speed.compile(tmpl)
	p.strength.compile(tmpl)
	return p
}

func (p pulseEffect) Apply(buf *Buffer, ctx PostContext) {
	black := Color{Mode: ColorRGB}
	t := (math.Sin(ctx.Time.Seconds()*p.speed.resolve()*math.Pi*2) + 1) * 0.5
	dim := t * p.strength.resolve()

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

// gradientMapEffect remaps all colour luminance through a three-stop gradient.
type gradientMapEffect struct{ dark, mid, bright Color }

// SEGradientMap remaps all colour luminance through a three-stop gradient.
// Dark shades map to the first colour, midtones to the second, highlights to the third.
func SEGradientMap(dark, mid, bright Color) gradientMapEffect {
	return gradientMapEffect{dark: dark, mid: mid, bright: bright}
}

func (g gradientMapEffect) Apply(buf *Buffer, ctx PostContext) {
	EachCell(func(_, _ int, c Cell, ectx PostContext) Cell {
		c.Style.FG = gradientMap(resolveFG(c.Style.FG, ectx), g.dark, g.mid, g.bright)
		c.Style.BG = gradientMap(resolveBG(c.Style.BG, ectx), g.dark, g.mid, g.bright)
		return c
	}).Apply(buf, ctx)
}

// ---------------------------------------------------------------------------
// Visual flair
// ---------------------------------------------------------------------------

// dropShadowEffect is a glow/drop-shadow — the inverse of vignette.
// Where vignette darkens from the screen edges inward, this darkens outward
// from a focus node's perimeter. At offset (0,0) it's a symmetric glow.
// Any offset displaces the shadow source, giving a directional drop shadow.
type dropShadowEffect struct {
	strength dynFloat64
	radius   dynInt
	offsetX  int
	offsetY  int
	tint     dynColor
	focus    *NodeRef
}

// SEDropShadow creates a radial glow/shadow emanating outward from a focus node.
// Default: radius 8, strength 0.2, offset (-1,-1) for a subtle directional shadow.
// Chain .Focus(&ref) to set the source node, .Offset(x,y) to adjust direction.
func SEDropShadow() dropShadowEffect {
	return dropShadowEffect{
		strength: dynFloat64{val: 0.2},
		radius:   dynInt{val: 8},
		offsetX:  -1, offsetY: -1,
		tint: dynColor{val: Color{Mode: ColorRGB}},
	}
}

// Strength sets shadow darkness (0.0 = none, 1.0 = full black at source edge).
func (d dropShadowEffect) Strength(s any) dropShadowEffect { d.strength.set(s); return d }

func (d dropShadowEffect) compileEffect(tmpl *Template) Effect {
	d.strength.compile(tmpl)
	d.radius.compile(tmpl)
	d.tint.compile(tmpl)
	return d
}

// Radius sets how far the shadow spreads in cells.
func (d dropShadowEffect) Radius(r any) dropShadowEffect { d.radius.set(r); return d }

// Offset displaces the shadow source — turns the symmetric glow into a directional drop shadow.
func (d dropShadowEffect) Offset(x, y int) dropShadowEffect { d.offsetX = x; d.offsetY = y; return d }

// Tint sets the shadow colour (default black).
func (d dropShadowEffect) Tint(c any) dropShadowEffect { d.tint.set(c); return d }

// Focus sets the node the shadow emanates from.
func (d dropShadowEffect) Focus(ref *NodeRef) dropShadowEffect { d.focus = ref; return d }

func (d dropShadowEffect) Apply(buf *Buffer, ctx PostContext) {
	if d.focus == nil {
		return
	}

	ref := d.focus
	radius := float64(d.radius.resolve())
	sx, sy := ref.X+d.offsetX, ref.Y+d.offsetY

	for y := range ctx.Height {
		base := y * buf.width
		for x := range ctx.Width {
			if inRect(x, y, ref) {
				continue
			}

			cx := max(sx, min(x, sx+ref.W-1))
			cy := max(sy, min(y, sy+ref.H-1))
			dx := float64(x - cx)
			dy := float64(y-cy) * 2
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist >= radius {
				continue
			}

			t := 1.0 - dist/radius
			dim := t * t * d.strength.resolve()

			tintColor := d.tint.resolve()
			c := &buf.cells[base+x]
			c.Style.FG = lerpIfRGB(resolveFG(c.Style.FG, ctx), tintColor, dim)
			c.Style.BG = lerpIfRGB(resolveBG(c.Style.BG, ctx), tintColor, dim)
		}
	}
}

// glowEffect emanates light outward from a focus node, sampling the node's
// edge colours and boosting them — the glow takes on the colour of the content.
type glowEffect struct {
	strength   dynFloat64
	radius     dynInt
	brightness dynFloat64
	focus      *NodeRef
}

// SEGlow creates a colour-sampling glow that reads the focus node's edge pixels
// and spills a brightened version of those colours into the surrounding area.
// Default: radius 8, strength 0.5, brightness 1.4.
func SEGlow() glowEffect {
	return glowEffect{
		strength:   dynFloat64{val: 0.5},
		radius:     dynInt{val: 8},
		brightness: dynFloat64{val: 1.4},
	}
}

// Strength sets how strongly the glow blends into surrounding cells.
func (g glowEffect) Strength(s any) glowEffect { g.strength.set(s); return g }

func (g glowEffect) compileEffect(tmpl *Template) Effect {
	g.strength.compile(tmpl)
	g.radius.compile(tmpl)
	g.brightness.compile(tmpl)
	return g
}

// Radius sets how far the glow spreads in cells.
func (g glowEffect) Radius(r any) glowEffect { g.radius.set(r); return g }

// Brightness sets the boost applied to sampled edge colours (1.0 = no boost).
func (g glowEffect) Brightness(b any) glowEffect { g.brightness.set(b); return g }

// Focus sets the node the glow emanates from.
func (g glowEffect) Focus(ref *NodeRef) glowEffect { g.focus = ref; return g }

func (g glowEffect) Apply(buf *Buffer, ctx PostContext) {
	if g.focus == nil {
		return
	}

	ref := g.focus
	radius := float64(g.radius.resolve())

	for y := range ctx.Height {
		base := y * buf.width
		for x := range ctx.Width {
			if inRect(x, y, ref) {
				continue
			}

			ex := max(ref.X, min(x, ref.X+ref.W-1))
			ey := max(ref.Y, min(y, ref.Y+ref.H-1))

			dx := float64(x - ex)
			dy := float64(y-ey) * 2
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist >= radius {
				continue
			}

			edge := buf.Get(ex, ey)
			sample := resolveBG(edge.Style.BG, ctx)
			if sample.Mode != ColorRGB {
				continue
			}

			bright := g.brightness.resolve()
			boosted := Color{
				Mode: ColorRGB,
				R:    uint8(min(int(float64(sample.R)*bright), 255)),
				G:    uint8(min(int(float64(sample.G)*bright), 255)),
				B:    uint8(min(int(float64(sample.B)*bright), 255)),
			}

			t := 1.0 - dist/radius
			blend := t * t * g.strength.resolve()

			c := &buf.cells[base+x]
			c.Style.FG = lerpIfRGB(resolveFG(c.Style.FG, ctx), boosted, blend)
			c.Style.BG = lerpIfRGB(resolveBG(c.Style.BG, ctx), boosted, blend)
		}
	}
}

// bloomEffect creates a coloured glow around bright cells.
type bloomEffect struct {
	radius    dynInt
	threshold dynFloat64
	strength  dynFloat64
	focus     *NodeRef
}

// SEBloom creates a coloured glow around bright cells.
// Bleeds bright colours into both FG and BG of surrounding cells.
// Default: radius 2, threshold 0.6, strength 0.3.
func SEBloom() bloomEffect {
	return bloomEffect{
		radius:    dynInt{val: 2},
		threshold: dynFloat64{val: 0.6},
		strength:  dynFloat64{val: 0.3},
	}
}

// Radius sets the spread in cells (2-4 recommended).
func (b bloomEffect) Radius(r any) bloomEffect { b.radius.set(r); return b }

// Threshold sets the minimum brightness that blooms (0.0–1.0).
func (b bloomEffect) Threshold(t any) bloomEffect { b.threshold.set(t); return b }

// Strength sets glow intensity (0.3 = subtle, 1.0 = vivid).
func (b bloomEffect) Strength(s any) bloomEffect { b.strength.set(s); return b }

func (b bloomEffect) compileEffect(tmpl *Template) Effect {
	b.radius.compile(tmpl)
	b.threshold.compile(tmpl)
	b.strength.compile(tmpl)
	return b
}

// Focus constrains bloom output to the given node — only cells within the rect receive glow.
func (b bloomEffect) Focus(ref *NodeRef) bloomEffect { b.focus = ref; return b }

func (b bloomEffect) Apply(buf *Buffer, ctx PostContext) {
	bw, bh := ctx.Width, ctx.Height
	// snapshot raw FG colours — do NOT resolve ColorDefault to terminal FG here.
	// only cells with explicit colours (ColorRGB, Color16) should act as sources.
	snap := make([]Color, bw*bh)
	for y := range bh {
		bufBase := y * buf.width
		snapBase := y * bw
		for x := range bw {
			snap[snapBase+x] = buf.cells[bufBase+x].Style.FG
		}
	}

	radius := b.radius.resolve()
	thresh := b.threshold.resolve()
	thresh256 := thresh * 255
	maxDist := math.Sqrt(float64(radius*radius) + float64(radius*radius)*4)

	// constrain output to focus rect if set
	x0, y0, x1, y1 := 0, 0, bw, bh
	if b.focus != nil {
		x0 = max(0, b.focus.X)
		y0 = max(0, b.focus.Y)
		x1 = min(bw, b.focus.X+b.focus.W)
		y1 = min(bh, b.focus.Y+b.focus.H)
	}

	for y := y0; y < y1; y++ {
		base := y * buf.width
		for x := x0; x < x1; x++ {
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
				blend := (sumWt / (sumWt + 1)) * b.strength.resolve()
				c := &buf.cells[base+x]
				c.Style.FG = lerpIfRGB(resolveFG(c.Style.FG, ctx), bloom, blend)
				c.Style.BG = lerpIfRGB(resolveBG(c.Style.BG, ctx), bloom, blend*0.3)
			}
		}
	}
}

// monochromeEffect converts all colours to a single-tint monochrome.
type monochromeEffect struct {
	tint  Color
	dodge *NodeRef
}

// SEMonochrome converts all colours to a single-tint monochrome.
// Pass RGB(0, 255, 0) for green phosphor, RGB(255, 180, 0) for amber.
func SEMonochrome(tint Color) monochromeEffect { return monochromeEffect{tint: tint} }

// Dodge exempts the given node — keep one panel in colour while the world goes mono.
func (m monochromeEffect) Dodge(ref *NodeRef) monochromeEffect { m.dodge = ref; return m }

func (m monochromeEffect) Apply(buf *Buffer, ctx PostContext) {
	EachCell(func(x, y int, c Cell, ectx PostContext) Cell {
		if m.dodge != nil && inRect(x, y, m.dodge) {
			return c
		}
		c.Style.FG = monochromeColor(resolveFG(c.Style.FG, ectx), m.tint)
		c.Style.BG = monochromeColor(resolveBG(c.Style.BG, ectx), m.tint)
		return c
	}).Apply(buf, ctx)
}

// ---------------------------------------------------------------------------
// Transitions & kinetic effects (require animation system for best results)
// ---------------------------------------------------------------------------

// dissolveEffect randomly hides cells based on progress.
type dissolveEffect struct{ progress *float64 }

func SEDissolve(progress *float64) dissolveEffect { return dissolveEffect{progress: progress} }

func (d dissolveEffect) Apply(buf *Buffer, ctx PostContext) {
	p := *d.progress
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

// screenShakeEffect displaces the entire buffer horizontally with a sine wave.
type screenShakeEffect struct{ amplitude float64 }

func SEScreenShake(amplitude float64) screenShakeEffect {
	return screenShakeEffect{amplitude: amplitude}
}

func (s screenShakeEffect) Apply(buf *Buffer, ctx PostContext) {
	offset := int(math.Round(math.Sin(float64(ctx.Frame)*1.5) * s.amplitude))
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

// ---------------------------------------------------------------------------
// Quantize (output optimisation, works standalone or via WithQuantize)
// ---------------------------------------------------------------------------

// quantizeEffect snaps all RGB colour channels to the nearest multiple of step.
type quantizeEffect struct{ step uint8 }

// SEQuantize snaps all RGB channels to the nearest multiple of step.
// Use step=32 to cut animation bytes per frame by ~40% with negligible banding.
// Prefer WithQuantize to wrap another effect rather than using this standalone.
func SEQuantize(step uint8) quantizeEffect { return quantizeEffect{step: step} }

func (q quantizeEffect) Apply(buf *Buffer, ctx PostContext) {
	EachCell(func(_, _ int, c Cell, _ PostContext) Cell {
		if c.Style.FG.Mode == ColorRGB {
			c.Style.FG.R = quantizeUint8(c.Style.FG.R, q.step)
			c.Style.FG.G = quantizeUint8(c.Style.FG.G, q.step)
			c.Style.FG.B = quantizeUint8(c.Style.FG.B, q.step)
		}
		if c.Style.BG.Mode == ColorRGB {
			c.Style.BG.R = quantizeUint8(c.Style.BG.R, q.step)
			c.Style.BG.G = quantizeUint8(c.Style.BG.G, q.step)
			c.Style.BG.B = quantizeUint8(c.Style.BG.B, q.step)
		}
		return c
	}).Apply(buf, ctx)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func resolveFG(c Color, ctx PostContext) Color {
	if c.Mode == ColorDefault && ctx.DefaultFG.Mode != ColorDefault {
		return ctx.DefaultFG
	}
	return c
}

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

func quantizeUint8(v, step uint8) uint8 {
	if step <= 1 {
		return v
	}
	rounded := int(math.Round(float64(v)/float64(step))) * int(step)
	if rounded > 255 {
		return 255
	}
	return uint8(rounded)
}

func inRect(x, y int, ref *NodeRef) bool {
	return x >= ref.X && x < ref.X+ref.W && y >= ref.Y && y < ref.Y+ref.H
}
