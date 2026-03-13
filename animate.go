package glyph

import (
	"math"
	"time"
)

// tween describes an animation that interpolates toward a watched target value.
// created via Animate(), which returns a tweenNode for the property compiler.
type tween struct {
	target   any
	duration time.Duration
	ease     func(float64) float64
	from     any // initial value — if set, animation starts immediately
}

// AnimateFn configures and creates tweens. Methods return new AnimateFn values,
// so animation styles can be defined once and reused across properties:
//
//	smooth := Animate.Duration(200 * time.Millisecond).Ease(EaseOutCubic)
//	VBox.Height(smooth(&targetHeight))
//	VBox.Width(smooth(&targetWidth))
//	Sparkline(data).Height(smooth(If(&expanded).Then(int16(26)).Else(int16(1))))
type AnimateFn func(target any) *tween

// Animate creates a tween that watches a target value and interpolates toward it.
// target can be a pointer (*int16, *float32, etc.), a conditionNode, or any
// value the property compiler can resolve to a typed pointer.
//
//	// simple — uses defaults (200ms linear)
//	VBox.Height(Animate(&targetHeight))
//
//	// configured up-front, applied to target
//	Animate.Duration(300 * time.Millisecond).Ease(EaseOutCubic)(&targetHeight)
var Animate AnimateFn = func(target any) *tween {
	return &tween{
		target:   target,
		duration: 200 * time.Millisecond,
	}
}

// Duration sets the animation duration. Returns a new AnimateFn.
func (f AnimateFn) Duration(d time.Duration) AnimateFn {
	return func(target any) *tween {
		tw := f(target)
		tw.duration = d
		return tw
	}
}

// Ease sets the easing function. Receives t in [0,1], returns eased t in [0,1].
// Returns a new AnimateFn.
func (f AnimateFn) Ease(fn func(float64) float64) AnimateFn {
	return func(target any) *tween {
		tw := f(target)
		tw.ease = fn
		return tw
	}
}

// From sets the initial value before animation starts.
// The tween immediately begins interpolating from this value toward the target.
// Returns a new AnimateFn.
//
//	SEVignette().Strength(Animate.Duration(1*time.Second).From(0.0)(0.88))
func (f AnimateFn) From(v any) AnimateFn {
	return func(target any) *tween {
		tw := f(target)
		tw.from = v
		return tw
	}
}

// tweenNode interface for the compiler to detect tween nodes
type tweenNode interface {
	getTarget() any
	getTweenDuration() time.Duration
	getTweenEasing() func(float64) float64
	getTweenFrom() any
}

func (tw *tween) getTarget() any                        { return tw.target }
func (tw *tween) getTweenDuration() time.Duration       { return tw.duration }
func (tw *tween) getTweenEasing() func(float64) float64 { return tw.ease }
func (tw *tween) getTweenFrom() any                     { return tw.from }

var _ tweenNode = (*tween)(nil)

// --- color and style interpolation ---

func lerpColor(from, to Color, t float64) Color {
	r := uint8(float64(from.R) + t*float64(int16(to.R)-int16(from.R)))
	g := uint8(float64(from.G) + t*float64(int16(to.G)-int16(from.G)))
	b := uint8(float64(from.B) + t*float64(int16(to.B)-int16(from.B)))
	return RGB(r, g, b)
}

func lerpStyle(from, to Style, t float64) Style {
	s := to // snap non-interpolatable fields (attrs, transform, align) to target
	if from.FG.Mode != ColorDefault && to.FG.Mode != ColorDefault {
		s.FG = lerpColor(from.FG, to.FG, t)
	}
	if from.BG.Mode != ColorDefault && to.BG.Mode != ColorDefault {
		s.BG = lerpColor(from.BG, to.BG, t)
	}
	if from.Fill.Mode != ColorDefault && to.Fill.Mode != ColorDefault {
		s.Fill = lerpColor(from.Fill, to.Fill, t)
	}
	return s
}

// --- easing functions ---
// all take t in [0,1] and return eased value in [0,1]

func EaseInQuad(t float64) float64    { return t * t }
func EaseOutQuad(t float64) float64   { return t * (2 - t) }
func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

func EaseInCubic(t float64) float64    { return t * t * t }
func EaseOutCubic(t float64) float64   { t--; return 1 + t*t*t }
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	t--
	return 1 + 8*t*t*t
}

func EaseInQuart(t float64) float64    { return t * t * t * t }
func EaseOutQuart(t float64) float64   { t--; return 1 - t*t*t*t }
func EaseInOutQuart(t float64) float64 {
	if t < 0.5 {
		return 8 * t * t * t * t
	}
	t--
	return 1 - 8*t*t*t*t
}

func EaseInQuint(t float64) float64    { return t * t * t * t * t }
func EaseOutQuint(t float64) float64   { t--; return 1 + t*t*t*t*t }
func EaseInOutQuint(t float64) float64 {
	if t < 0.5 {
		return 16 * t * t * t * t * t
	}
	t--
	return 1 + 16*t*t*t*t*t
}

func EaseInSine(t float64) float64    { return 1 - math.Cos(t*math.Pi/2) }
func EaseOutSine(t float64) float64   { return math.Sin(t * math.Pi / 2) }
func EaseInOutSine(t float64) float64 { return -(math.Cos(math.Pi*t) - 1) / 2 }

func EaseInExpo(t float64) float64 {
	if t == 0 {
		return 0
	}
	return math.Pow(2, 10*(t-1))
}

func EaseOutExpo(t float64) float64 {
	if t == 1 {
		return 1
	}
	return 1 - math.Pow(2, -10*t)
}

func EaseInOutExpo(t float64) float64 {
	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}
	if t < 0.5 {
		return math.Pow(2, 20*t-10) / 2
	}
	return (2 - math.Pow(2, -20*t+10)) / 2
}

func EaseInCirc(t float64) float64    { return 1 - math.Sqrt(1-t*t) }
func EaseOutCirc(t float64) float64   { t--; return math.Sqrt(1 - t*t) }
func EaseInOutCirc(t float64) float64 {
	if t < 0.5 {
		return (1 - math.Sqrt(1-4*t*t)) / 2
	}
	t = 2*t - 2
	return (math.Sqrt(1-t*t) + 1) / 2
}

func EaseInBack(t float64) float64 {
	const s = 1.70158
	return t * t * ((s+1)*t - s)
}

func EaseOutBack(t float64) float64 {
	const s = 1.70158
	t--
	return 1 + t*t*((s+1)*t+s)
}

func EaseInOutBack(t float64) float64 {
	const s = 1.70158 * 1.525
	if t < 0.5 {
		return (4 * t * t * ((s+1)*2*t - s)) / 2
	}
	t = 2*t - 2
	return (t*t*((s+1)*t+s) + 2) / 2
}

func EaseOutBounce(t float64) float64 {
	switch {
	case t < 1.0/2.75:
		return 7.5625 * t * t
	case t < 2.0/2.75:
		t -= 1.5 / 2.75
		return 7.5625*t*t + 0.75
	case t < 2.5/2.75:
		t -= 2.25 / 2.75
		return 7.5625*t*t + 0.9375
	default:
		t -= 2.625 / 2.75
		return 7.5625*t*t + 0.984375
	}
}

func EaseInBounce(t float64) float64    { return 1 - EaseOutBounce(1-t) }
func EaseInOutBounce(t float64) float64 {
	if t < 0.5 {
		return (1 - EaseOutBounce(1-2*t)) / 2
	}
	return (1 + EaseOutBounce(2*t-1)) / 2
}
