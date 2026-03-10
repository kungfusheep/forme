# Screen Effects — Post-Processing API

Glyph's post-processing system applies GPU-shader-style passes to the terminal cell buffer after rendering, before the frame is flushed. Effects are declared inline in the view tree with `ScreenEffect()` and toggled reactively like any other node.

## How it works

Each frame, the renderer walks the view tree, collects all active `ScreenEffect` nodes, and runs their effects in order against the full cell buffer. Effects operate on `Color` values (RGB, 16-colour, or terminal default) and receive a `PostContext` with frame timing and terminal colour detection data.

Effects are pure values — structs with a chain API — so they're safe to declare once at the top of a view function and reuse across frames.

## Quick start

```go
// single effect
ScreenEffect(SEVignette())

// toggle reactively
dimmed := false
If(&dimmed).Then(ScreenEffect(SEDesaturate()))

// stack multiple effects (applied left to right)
ScreenEffect(
    SEVignette().Strength(0.5),
    SETint(Hex(0xFF6600)).Strength(0.1),
)
```

## Focus & Dodge

Many effects accept a `*NodeRef` to target specific panels. Capture a node's layout rect by calling `.NodeRef(&ref)` on any container node.

- **`.Focus(&ref)`** — centres or sources the effect from a node (vignette, drop shadow, glow)
- **`.Dodge(&ref)`** — exempts a node from the effect (all applicable effects)

```go
var modal NodeRef
VBox.NodeRef(&modal)(...)

ScreenEffect(
    SEVignette().Dodge(&modal),    // darken edges, spare the modal
    SEDropShadow().Focus(&modal),  // shadow radiates outward from the modal
)
```

## Effect reference

### Subtle — always-on polish

| Effect | What it does | Key knobs |
|---|---|---|
| `SEDimAll()` | Applies terminal Dim attribute to every cell | — |
| `SETint(color)` | Colour-grades toward a target hue | `.Strength()` `.Dodge()` |
| `SEVignette()` | Darkens screen edges with quadratic falloff | `.Strength()` `.Focus()` `.Dodge()` `.Smooth()` |
| `SEDesaturate()` | Removes colour saturation (perceptual luminance) | `.Strength()` `.Dodge()` |
| `SEContrast()` | Pushes colour channels toward extremes | `.Strength()` `.Dodge()` |
| `SEFadeRows(top, bottom, color)` | Fades N rows at each edge toward a colour | — |

### Purposeful — modal and focus patterns

| Effect | What it does | Key knobs |
|---|---|---|
| `SEFocusDim(&ref)` | Dims everything outside a node | — |
| `SEFrost()` | Frosted glass — replaces chars with shade blocks and desaturates | `.Strength()` `.Focus()` `.Dodge()` |
| `SEDropShadow()` | Radial darkening outward from a focus node | `.Focus()` `.Strength()` `.Radius()` `.Offset()` `.Tint()` |
| `SEGlow()` | Colour-sampling glow — reads edge pixels and spills a brightened version outward | `.Focus()` `.Strength()` `.Radius()` `.Brightness()` |

`SEDropShadow` and `SEGlow` are two sides of the same coin: drop shadow darkens outward (fixed tint), glow brightens outward (sampled colour). At `Offset(0, 0)` drop shadow becomes a symmetric glow around the panel.

### Visual flair

| Effect | What it does | Key knobs |
|---|---|---|
| `SEBloom()` | Bleeds bright cell colours into their neighbours | `.Threshold()` `.Strength()` `.Radius()` `.Focus()` |
| `SECRT()` | CRT scanlines + vignette + warm phosphor tint | — |
| `SEMonochrome(tint)` | Single-tint monochrome | `.Dodge()` |
| `SEGradientMap(dark, mid, bright)` | Remaps all luminance through a three-stop gradient | — |

### Utilities

| Effect | What it does |
|---|---|
| `SEQuantize(step)` | Snaps RGB channels to step-size buckets. Use step=32 to cut animated effect bytes ~40%. |
| `WithBlend(mode, effect)` | Wraps any effect with a Photoshop-style blend mode |
| `WithQuantize(step, effect)` | Wraps any effect with post-quantization |
| `EachCell(fn)` | Per-cell fragment shader — define per-cell logic, iteration and parallelism handled for you |

## Blend modes

`WithBlend` supports: `BlendNormal`, `BlendMultiply`, `BlendScreen`, `BlendOverlay`, `BlendAdd`, `BlendSoftLight`, `BlendColorDodge`, `BlendColorBurn`.

```go
// fire through screen blend — additive bright
ScreenEffect(WithBlend(BlendScreen, SEFire()))

// plasma through overlay with quantization — efficient animated wash
ScreenEffect(WithQuantize(32, WithBlend(BlendOverlay, SEPlasma())))
```

## Custom effects

Implement the `Effect` interface to write your own:

```go
type Effect interface {
    Apply(buf *Buffer, ctx PostContext)
}
```

`EachCell` handles iteration and splits work across a worker pool for you — use it for any per-cell transform:

```go
type invertEffect struct{}

func (e invertEffect) Apply(buf *Buffer, ctx PostContext) {
    EachCell(func(x, y int, c Cell, ctx PostContext) Cell {
        if c.Style.FG.Mode == ColorRGB {
            c.Style.FG = RGB(255-c.Style.FG.R, 255-c.Style.FG.G, 255-c.Style.FG.B)
        }
        return c
    }).Apply(buf, ctx)
}
```

For effects that need spatial context (neighbouring cells, distance from a point), implement `Apply` directly and iterate `buf.cells` manually — see `SEVignette` or `SEDropShadow` in `postprocesseffects.go` for reference patterns.

## PostContext fields

```go
type PostContext struct {
    Width, Height int           // current terminal dimensions in cells
    Frame         uint64        // monotonically increasing frame counter
    Delta         time.Duration // time since last frame
    Time          time.Duration // total elapsed since first render
    DefaultFG     Color         // terminal's detected foreground (OSC 10)
    DefaultBG     Color         // terminal's detected background (OSC 11)
}
```

`DefaultFG` and `DefaultBG` are populated at startup via OSC 10/11 queries. Their `Mode` is `ColorRGB` when detected, `ColorDefault` when the terminal didn't respond. Effects that lerp toward or away from the background colour use these to work correctly against any terminal theme.
