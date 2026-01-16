# Layout Cycle - Current State & Known Gaps

## Naming

We should have ONE template system, not "GlintTemplate" vs "SerialTemplate" vs "FlexLayout".
Just "Template" - users shouldn't need to choose or understand the history.

The Glint methodology influenced the implementation, but that's an implementation detail.

---

## The Correct Model (from FlexLayout)

Three-phase layout system documented in `flexlayout.go`:

```
Update (top→down): Distribute available widths to children
Layout (bottom→up): Children calculate heights, parents position children
Draw (top→down): Render with viewport culling
```

**Why this ordering matters:**
1. Parent knows available width → tells children (constraints flow DOWN)
2. Children compute their height based on content → tell parent (sizes flow UP)
3. Parent positions children based on their reported sizes
4. Then we can render

This handles:
- Variable height content (text wrapping, nested ForEach with different item counts)
- Flex layouts where children expand to fill space
- Centering/alignment (need child size before positioning)

---

## What GlintTemplate Currently Does

**Single-pass compile-time layout:**

```go
Compile(ui) → positions baked into instructions
Layout()    → re-runs Compile() to refresh positions
Render()    → draws using baked positions
```

**Problems:**
1. No constraint passing (children don't know available width)
2. Heights are guessed/fixed (Text=1, Progress=1), not measured
3. ForEach item heights assumed uniform
4. No top-down/bottom-up separation

---

## The Gap

| Aspect | FlexLayout Model | GlintTemplate Current |
|--------|------------------|----------------------|
| Width distribution | Top-down from parent | None - fixed widths |
| Height calculation | Bottom-up from children | Fixed at compile (guess) |
| Variable heights | Supported | Broken (assumes uniform) |
| Flex/grow | Supported | Not implemented |
| Text wrapping | Width-aware | Not implemented |
| Layout timing | Per-frame, phased | Per-frame, single-pass |

---

## Concrete Example of the Problem

```go
ForEach(&rows, func(row *Row) {
  return Col{
    Text{&row.Name},
    ForEach(&row.Items, func(item *Item) {  // variable length!
      return Text{&item.Value}
    }),
  }
})
```

- Row 0 might have 3 items → height 4
- Row 1 might have 10 items → height 11
- To position Row 1, we need Row 0's height
- Row 0's height depends on laying out its children
- **This requires bottom-up height calculation**

GlintTemplate currently assumes all rows have the same height (from template compilation with dummy data).

---

## Frame Cycle (Not Yet Implemented)

The full render cycle should be:

```
1. Handle Events    (resize, input)
2. Update State     (data changes, scroll position)
3. Update Phase     (top→down: distribute widths)
4. Layout Phase     (bottom→up: calculate heights, set positions)
5. Render Phase     (top→down: draw with viewport culling)
6. Present          (flush to terminal)
```

Currently we have:
```
1. Handle Events    (manual)
2. Update State     (manual)
3. Layout()         (re-compiles, single-pass, no phases)
4. Render()         (draws)
5. Present          (manual)
```

---

## Open Questions

1. **Do we integrate FlexLayout into GlintTemplate?** Or keep them separate?

2. **How do we handle ForEach with variable-height items?**
   - Option A: Require uniform heights (current, limiting)
   - Option B: Measure all items every frame (correct, slower)
   - Option C: Cache heights, invalidate on structural change (complex)

3. **When does layout actually need to re-run?**
   - Viewport resize → yes
   - Scroll position change → no (just viewport culling)
   - Data value change → no (positions unchanged)
   - Slice length change → yes (affects heights/positions)
   - Structural change → yes (different UI tree)

4. **How do we track "dirty" efficiently?**
   - Dirty flags on nodes?
   - Generation counters?
   - Hash of structural shape?

5. **Can we cull layout for off-screen items?**
   - For uniform heights: yes (calculate position from index)
   - For variable heights: need cumulative heights (can't skip)

---

## Sub-Templates (Glint's Sub-Encoder Pattern)

Glint handles nested/complex types via sub-encoders. We can apply the same pattern:

```go
// ForEach has a sub-template for its elements
type ForEachInstruction struct {
    SlicePtr    unsafe.Pointer
    ElemSize    uintptr
    ElemTemplate *Template  // sub-template, compiled once
}
```

**Key insight**: The sub-template iteration can potentially be inlined into the main render loop:

```go
// Instead of calling out to a separate function:
for i := range slice {
    elemTemplate.Render(buf, elemPtr)  // function call overhead
}

// Could inline the sub-template instructions:
for i := range slice {
    // sub-template instructions expanded here
    // no function call, just continue the switch
}
```

This gives us:
- One template type (not multiple competing systems)
- Volatile data handled via sub-templates
- Potential for inlining to eliminate call overhead
- ForEach/nested structures as a natural part of the system, not a special case

---

## Next Steps (When We Return to This)

1. Unify naming - one "Template" type
2. Decide if variable-height ForEach is a requirement
3. If yes, implement proper two-phase layout (constraints down, sizes up)
4. Add dirty tracking to avoid full re-layout every frame
5. Integrate with viewport culling (only layout visible + buffer)
6. Explore sub-template inlining for ForEach
7. Benchmark the overhead vs current single-pass

---

## References

- `flexlayout.go` - Three-phase model implementation
- `glint_render.go` - Current single-pass implementation
- `DESIGN.md` - Original phase separation design (Build → Expand → Layout → Draw)
- `GLINT_METHODOLOGY.md` - Performance optimization approach
