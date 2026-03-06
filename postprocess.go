package forme

import (
	"math"
	"sync"
	"time"
)

// resolveColor16 updates Color16 cells in the buffer to use the detected
// terminal palette RGB values. Called once before effects run so all
// colour math operates on the terminal's actual colours.
func resolveColor16(buf *Buffer, w, h int) {
	for y := range h {
		base := y * buf.width
		for x := range w {
			c := &buf.cells[base+x]
			if c.Style.FG.Mode == Color16 {
				rgb := basic16RGB[c.Style.FG.Index&0xF]
				c.Style.FG.R, c.Style.FG.G, c.Style.FG.B = rgb[0], rgb[1], rgb[2]
			}
			if c.Style.BG.Mode == Color16 {
				rgb := basic16RGB[c.Style.BG.Index&0xF]
				c.Style.BG.R, c.Style.BG.G, c.Style.BG.B = rgb[0], rgb[1], rgb[2]
			}
		}
	}
}

// PostProcess transforms the buffer after rendering, before flush.
// Like a GPU shader pass — receives the full cell buffer and frame context.
// Mutate cells in-place. Chain multiple passes for layered effects.
type PostProcess func(buf *Buffer, ctx PostContext)

// ScreenEffectNode is a declarative node that registers a full-screen
// post-processing effect. Place it anywhere in the view tree — it takes
// zero layout space and applies to the entire screen. Works with If()
// for reactive toggling.
type ScreenEffectNode struct {
	Effect PostProcess
}

// ScreenEffect creates a declarative full-screen post-processing node.
func ScreenEffect(pp PostProcess) ScreenEffectNode {
	return ScreenEffectNode{Effect: pp}
}

// PostContext provides frame metadata to post-processing passes.
type PostContext struct {
	Width  int
	Height int
	Frame  uint64
	Delta  time.Duration // time since last frame
	Time   time.Duration // total elapsed since first render

	// terminal's default FG/BG detected via OSC 10/11 at startup.
	// Mode == ColorRGB when detected, ColorDefault when unknown.
	DefaultFG Color
	DefaultBG Color
}

// EachCell wraps a per-cell transform into a PostProcess.
// The fragment shader equivalent — you define the per-cell logic,
// iteration is handled for you. Splits work across four quadrants
// using a persistent worker pool — zero allocations in the hot path.
func EachCell(fn func(x, y int, cell Cell, ctx PostContext) Cell) PostProcess {
	return func(buf *Buffer, ctx PostContext) {
		qp := getCellPool()
		midX, midY := ctx.Width/2, ctx.Height/2

		// set shared work (safe: written before wakeup send, read after recv)
		qp.fn = fn
		qp.buf = buf
		qp.ctx = ctx
		qp.quads = [4][4]int{
			{0, midX, 0, midY},
			{midX, ctx.Width, 0, midY},
			{0, midX, midY, ctx.Height},
			{midX, ctx.Width, midY, ctx.Height},
		}

		for i := range 3 {
			qp.wakeup[i] <- struct{}{}
		}

		// run 4th quadrant on caller
		q := qp.quads[3]
		for y := q[2]; y < q[3]; y++ {
			base := y * buf.width
			for x := q[0]; x < q[1]; x++ {
				idx := base + x
				buf.cells[idx] = fn(x, y, buf.cells[idx], ctx)
			}
		}

		for i := range 3 {
			<-qp.done[i]
		}
	}
}

type cellPool struct {
	fn     func(x, y int, cell Cell, ctx PostContext) Cell
	buf    *Buffer
	ctx    PostContext
	quads  [4][4]int
	wakeup [3]chan struct{}
	done   [3]chan struct{}
}

var cellPoolInstance struct {
	once sync.Once
	pool *cellPool
}

func getCellPool() *cellPool {
	cellPoolInstance.once.Do(func() {
		p := &cellPool{}
		for i := range 3 {
			p.wakeup[i] = make(chan struct{}, 1)
			p.done[i] = make(chan struct{}, 1)
			go func(idx int) {
				for range p.wakeup[idx] {
					q := p.quads[idx]
					buf, fn, ctx := p.buf, p.fn, p.ctx
					for y := q[2]; y < q[3]; y++ {
						base := y * buf.width
						for x := q[0]; x < q[1]; x++ {
							i := base + x
							buf.cells[i] = fn(x, y, buf.cells[i], ctx)
						}
					}
					p.done[idx] <- struct{}{}
				}
			}(i)
		}
		cellPoolInstance.pool = p
	})
	return cellPoolInstance.pool
}

// ---------------------------------------------------------------------------
// Blend modes
// ---------------------------------------------------------------------------

type BlendMode int

const (
	BlendNormal   BlendMode = iota
	BlendMultiply           // darkens: a * b / 255
	BlendScreen             // lightens: 255 - (255-a)(255-b)/255
	BlendOverlay            // multiply if dark, screen if light
	BlendAdd                // clipped addition
	BlendSoftLight          // gentle contrast
	BlendColorDodge         // dramatic brighten
	BlendColorBurn          // dramatic darken
)

// BlendColor combines two RGB colours using the specified blend mode.
func BlendColor(base, top Color, mode BlendMode) Color {
	if base.Mode == ColorDefault || top.Mode == ColorDefault {
		return top
	}
	return RGB(
		blendChannel(base.R, top.R, mode),
		blendChannel(base.G, top.G, mode),
		blendChannel(base.B, top.B, mode),
	)
}

func blendChannel(a, b uint8, mode BlendMode) uint8 {
	fa, fb := float64(a)/255, float64(b)/255
	var r float64
	switch mode {
	case BlendMultiply:
		r = fa * fb
	case BlendScreen:
		r = 1 - (1-fa)*(1-fb)
	case BlendOverlay:
		if fa < 0.5 {
			r = 2 * fa * fb
		} else {
			r = 1 - 2*(1-fa)*(1-fb)
		}
	case BlendAdd:
		r = fa + fb
	case BlendSoftLight:
		if fb < 0.5 {
			r = fa - (1-2*fb)*fa*(1-fa)
		} else {
			r = fa + (2*fb-1)*(softLightG(fa)-fa)
		}
	case BlendColorDodge:
		if fb >= 1 {
			r = 1
		} else {
			r = fa / (1 - fb)
		}
	case BlendColorBurn:
		if fb <= 0 {
			r = 0
		} else {
			r = 1 - (1-fa)/fb
		}
	default:
		r = fb
	}
	if r < 0 {
		r = 0
	} else if r > 1 {
		r = 1
	}
	return uint8(r * 255)
}

func softLightG(a float64) float64 {
	if a <= 0.25 {
		return ((16*a-12)*a + 4) * a
	}
	return math.Sqrt(a)
}

// WithBlend wraps any PostProcess with a blend mode. Snapshots the buffer
// before the effect runs, then blends the effect's output with the original
// using the specified mode. Works with any effect — plasma through multiply,
// fire through screen, etc.
func WithBlend(mode BlendMode, pp PostProcess) PostProcess {
	return func(buf *Buffer, ctx PostContext) {
		w, h := ctx.Width, ctx.Height

		// snapshot original FG+BG
		type colorPair struct{ fg, bg Color }
		snap := make([]colorPair, w*h)
		for y := range h {
			bufBase := y * buf.width
			snapBase := y * w
			for x := range w {
				c := buf.cells[bufBase+x]
				snap[snapBase+x] = colorPair{c.Style.FG, c.Style.BG}
			}
		}

		// run the effect
		pp(buf, ctx)

		// blend result with snapshot
		for y := range h {
			bufBase := y * buf.width
			snapBase := y * w
			for x := range w {
				c := &buf.cells[bufBase+x]
				orig := snap[snapBase+x]
				c.Style.FG = BlendColor(orig.fg, c.Style.FG, mode)
				c.Style.BG = BlendColor(orig.bg, c.Style.BG, mode)
			}
		}
	}
}
