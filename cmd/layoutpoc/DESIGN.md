# Layout POC - Design Document

Proof of concept for the new TUI layout architecture.

## Goals

- Clean phase separation: Template → Expand → Layout → Draw
- Fast-path layouts (inlined switch) for common cases
- Custom layout fallback (function pointer) for full control
- Pointer bindings for dynamic data
- ForEach expansion for volatile collections
- Viewport culling
- Architecture that allows future dirty tracking without rewrite

## Phases

```
Build (once)          → Template (stable data structure)
Expand (each frame)   → Expanded nodes (ForEach/If/Switch resolved)
Layout (each frame)   → Positioned nodes (X/Y/W/H computed)
Draw (each frame)     → Buffer writes (pure iteration)
```

## Checklist

### Core Data Structures
- [ ] TemplateNode - compiled node with layout hints
- [ ] Template - collection of nodes
- [ ] ExpandedNode - concrete node after ForEach expansion
- [ ] PositionedNode - node with computed geometry
- [ ] Rect - layout rectangle

### Build Phase
- [ ] Parse declarative UI into Template
- [ ] Store layout hints (Width, Height, PercentWidth, FlexGrow)
- [ ] Store content (static text, text pointers)
- [ ] Store children relationships
- [ ] ForEach compilation (slice pointer, element size, sub-template)
- [ ] If/Switch compilation
- [ ] Custom component compilation

### Expand Phase
- [ ] ForEach expansion (iterate slice, produce N nodes)
- [ ] If/Switch evaluation
- [ ] Conditional branch selection
- [ ] Element pointer tracking for bindings

### Layout Phase - Fast Paths
- [ ] LayoutRow - horizontal, inlined
- [ ] LayoutCol - vertical, inlined
- [ ] LayoutGrid - grid with N columns, inlined
- [ ] LayoutWrap - flow/wrap layout, inlined
- [ ] LayoutAbsolute - explicit X/Y positioning, inlined
- [ ] LayoutCustom - function pointer fallback

### Layout Phase - Features
- [ ] PercentWidth distribution
- [ ] FlexGrow distribution
- [ ] Gap handling
- [ ] Border accounting
- [ ] Explicit Width/Height respect

### Draw Phase
- [ ] Text rendering (static)
- [ ] Text rendering (pointer binding)
- [ ] Progress bar rendering
- [ ] Custom component rendering
- [ ] Container handling (borders, titles)

### Pointer Bindings
- [ ] String pointers (*string)
- [ ] Int pointers (*int)
- [ ] Bool pointers (*bool)
- [ ] Slice pointers for ForEach
- [ ] Offset calculation for ForEach element fields

### Viewport Culling
- [ ] Track visible region
- [ ] Skip layout for off-screen containers
- [ ] Skip draw for off-screen nodes

### Benchmarking
- [ ] Compare vs current SerialTemplate
- [ ] Measure fast-path vs custom layout overhead
- [ ] Measure ForEach expansion cost
- [ ] Measure with realistic UI complexity
- [ ] Profile memory allocations
- [ ] Verify zero allocations in hot path

### Inlining Analysis
- [ ] Check inlining with `go build -gcflags='-m'`
- [ ] Ensure Draw loop functions inline
- [ ] Ensure Layout switch cases inline
- [ ] Ensure buffer write methods inline
- [ ] Identify and fix any inlining blockers
- [ ] Compare inlining report vs current SerialTemplate

### Integration Path
- [ ] Define migration strategy from SerialTemplate
- [ ] Identify breaking changes
- [ ] Plan for minivim compatibility
- [ ] Plan for browser project compatibility

## Open Questions

1. Should Expand and Layout be combined into one phase?
2. ~~How do we handle nested ForEach?~~ **SOLVED: SerialOpForEachOffset**
3. What's the right granularity for dirty tracking hooks?
4. Should Custom components have access to child nodes or just geometry?

## Critical Bug Found & Fixed

**Nested ForEach was broken!** The old benchmark showed 268ns, but it was only rendering
1 text node because nested ForEach pointers weren't being rebound to actual element data.

**Fix:** Added `SerialOpForEachOffset` to handle nested ForEach where the inner slice
pointer is within the parent element range. At compile time, we detect this and store
an offset instead of an absolute pointer. At runtime, we compute the actual slice
address using `elemBase + offset`.

**Tests added:**
- `TestSimpleForEach` - verifies single-level ForEach (11 nodes)
- `TestNestedForEach` - verifies nested ForEach (101 nodes for 10x10 grid)

## Performance Targets

**North Star: ~400ns per frame** (for simple viewports)

- Layout + Draw < 1000ns for typical visible viewport (revised based on real measurements)
- Fast-path layouts should be within 10% of hand-written loops
- Custom layout overhead acceptable (one function call per custom container)
- Zero allocations in hot path after initial setup

## CORRECTED SerialTemplate Baseline (Apple M3 Pro)

**OLD (WRONG - nested ForEach was broken):**
```
BenchmarkStressDenseGrid     4368015     272 ns/op    0 B/op    0 allocs/op  # WRONG!
```

**NEW (CORRECT - with nested ForEach fix):**
```
BenchmarkStressDenseGrid       271305    3707 ns/op    0 B/op    0 allocs/op
```

## Raw Performance Comparison

| Benchmark | Time | Allocs | Overhead | Description |
|-----------|------|--------|----------|-------------|
| Direct WriteProgressBar x100 | **1055ns** | 0 | 1.0x | Pre-computed positions, direct calls |
| **GlintTemplate 10x10** | **1710ns** | 0 | **1.62x** | Glint-style direct rendering |
| SerialTemplate DenseGrid 10x10 | **3674ns** | 0 | 3.48x | 100 progress bars via nested ForEach |

## GlintTemplate POC Results

**🎉 GlintTemplate achieves PARITY with hand-written code!**

| Benchmark | Time | vs Hand-written |
|-----------|------|-----------------|
| Hand-written nested loops | **1163ns** | baseline |
| **GlintTemplate (with grid opt)** | **1165ns** | **1.00x** |
| SerialTemplate | **3670ns** | 3.15x slower |

**GlintTemplate is 3.15x faster than SerialTemplate!**

| Metric | SerialTemplate | GlintTemplate | Improvement |
|--------|----------------|---------------|-------------|
| Time (10x10 grid) | 3670ns | 1165ns | **3.15x faster** |
| Overhead vs Direct | 3.15x | **~0%** | **Eliminated** |
| Allocations | 0 | 0 | Same |

**Key optimization: Compile-time pattern detection**

For nested ForEach grids (very common for tables, lists, dashboards), the compiler detects
the pattern and stores pre-computed parameters. At render time, it's just two nested loops
with no function call overhead:

```go
// Detected pattern: ForEach(rows, Row(ForEach(items, Progress)))
// Render becomes:
func renderGridFast(buf *Buffer) {
    for i := 0; i < outerLen; i++ {
        for j := 0; j < innerLen; j++ {
            ratio := *(*float32)(ptr + offset)
            buf.WriteProgressBar(x, y, width, ratio, Style{})
        }
    }
}
```

**What makes it fast:**
1. Pattern detection at compile time
2. All offsets and sizes pre-computed
3. Single `renderGridFast` function - no call chain
4. Direct unsafe pointer arithmetic
5. Zero intermediate data structures

## Performance Journey

| Version | Time | Notes |
|---------|------|-------|
| SerialTemplate | 3670ns | Original implementation |
| Initial GlintTemplate | 1710ns | 2.15x faster |
| + Single-instruction fast path | 1490ns | Eliminated some recursion |
| + Nested ForEach inlining | 1470ns | More inlining |
| + Specialized progress loop | 1470ns | Removed switch from inner loop |
| **+ Grid pattern detection** | **1165ns** | **Parity with hand-written!** |

## Updated Observations

- **Hand-written nested loops**: ~1163ns for 100 progress bars
- **GlintTemplate (grid opt)**: ~1165ns - essentially identical
- **SerialTemplate**: ~3670ns - 3.15x slower
- Zero allocations maintained
- The key insight: detect common patterns at compile time and generate specialized code paths

POC successfully demonstrates that declarative UI can match hand-written performance.
