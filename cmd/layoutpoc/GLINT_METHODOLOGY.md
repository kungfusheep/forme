# Applying Glint's Methodology to TUI

## What Makes Glint Fast

Glint achieves **493ns encode / 775ns decode** with **zero allocations** by:

### 1. Pre-compiled Flat Instruction Array

```go
// Glint: Instructions built once at startup
type encodeInstruction struct {
    wire   WireType  // determines fast path in switch
    offset uintptr   // field offset for direct memory access
    fun    func()    // fallback for complex types only
}

// Runtime: Pure iteration, no tree traversal
for i := 0; i < len(e.instructions); i++ {
    switch e.instructions[i].wire {
    case WireInt32:
        b.AppendInt32(*(*int32)(unsafe.Add(p, e.instructions[i].offset)))
    // ... other fast paths
    default:
        e.instructions[i].fun(...)  // fallback only for complex types
    }
}
```

### 2. Direct Offset-Based Access

```go
// Glint: No reflection at runtime
*(*int32)(unsafe.Add(p, instruction.offset))

// vs typical reflection
reflect.ValueOf(v).Field(i).Int()  // ~100x slower
```

### 3. Type-Based Switch with Const Cases

```go
// Const cases enable jump table optimization
switch instruction.wire {
case WireBool:   // const
case WireInt:    // const
case WireInt8:   // const
// ... compiler generates O(1) jump table
}
```

### 4. Fast Paths for Common Types, Function Pointers for Complex

- **Fast path**: Inline code for scalars (bool, int, float, string)
- **Fallback**: Function pointer only for slices, maps, nested structs

### 5. Value Semantics to Avoid Heap Escapes

```go
// Glint: Reader passed by value, returned by value
func unmarshal(body Reader, ...) Reader {
    // body stays on stack
    return body
}
```

---

## Current TUI SerialTemplate Overhead

Profile of BenchmarkStressDenseGrid (100 progress bars, 3700ns):

| Cost | % | Cause |
|------|---|-------|
| **measureForEach** | 49% | Node creation, iteration, copying |
| **WriteProgressBar** | 23% | Actual buffer writes |
| **duffcopy/memmove** | 26% | Copying 88-byte SerialNode structs |

**The problem**: SerialTemplate creates intermediate `SerialNode` structs, then copies them, then iterates them to render. This is 3 passes over the data.

---

## How to Apply Glint's Methodology to TUI

### Key Insight: Eliminate Intermediate Nodes

Instead of:
```
Compile → [ops array]
Execute → [create nodes] → [position nodes] → [copy nodes] → [render nodes]
```

Do:
```
Compile → [render instructions with pre-computed positions]
Execute → [render directly to buffer]
```

### Proposed Architecture

#### 1. Pre-compiled Render Instructions

```go
type RenderInstruction struct {
    Kind     uint8    // determines fast path
    X, Y     int16    // pre-computed position (or offset from parent)
    Width    int16    // pre-computed width

    // For static content
    Text     string

    // For dynamic content (pointer bindings)
    StrOff   uintptr  // offset to string field
    FloatOff uintptr  // offset to float32 field

    // For containers
    Children []int16  // indices of child instructions

    // For ForEach (only complex type needing function)
    ForEachFn func(buf *Buffer, x, y int16, elemPtr unsafe.Pointer)
}
```

#### 2. Fast-Path Render Loop

```go
func (t *Template) Render(buf *Buffer, data unsafe.Pointer) {
    for i := 0; i < len(t.instructions); i++ {
        inst := &t.instructions[i]
        switch inst.Kind {
        case KindTextStatic:
            buf.WriteStringFast(int(inst.X), int(inst.Y), inst.Text, Style{})

        case KindTextOffset:
            text := *(*string)(unsafe.Add(data, inst.StrOff))
            buf.WriteStringFast(int(inst.X), int(inst.Y), text, Style{})

        case KindProgressOffset:
            ratio := *(*float32)(unsafe.Add(data, inst.FloatOff))
            buf.WriteProgressBar(int(inst.X), int(inst.Y), int(inst.Width), ratio, Style{})

        case KindForEach:
            // Only ForEach needs dynamic iteration
            inst.ForEachFn(buf, inst.X, inst.Y, data)
        }
    }
}
```

#### 3. Pre-compute Static Layouts at Compile Time

For static structures (no ForEach), compute X/Y positions during `Build`:

```go
// At compile time, not runtime:
func (t *Template) compileCol(children []any, x, y int16) {
    currentY := y
    for _, child := range children {
        switch c := child.(type) {
        case Text:
            t.instructions = append(t.instructions, RenderInstruction{
                Kind: KindTextStatic,
                X:    x,
                Y:    currentY,  // computed at compile time!
                Text: c.Content,
            })
            currentY++
        }
    }
}
```

#### 4. ForEach: The Only Dynamic Case

ForEach is the only case that truly needs runtime iteration:

```go
func compileForEach(slice any, render func(elem any) any) {
    // Pre-compile the element template
    elemTemplate := compileTemplate(render(dummyElem))

    // Store a function that:
    // 1. Iterates the slice at runtime
    // 2. Renders each element using pre-compiled positions with Y offset
    forEachFn := func(buf *Buffer, baseX, baseY int16, slicePtr unsafe.Pointer) {
        hdr := *(*sliceHeader)(slicePtr)
        for i := 0; i < hdr.Len; i++ {
            elemPtr := unsafe.Add(hdr.Data, uintptr(i)*elemSize)
            elemTemplate.RenderWithOffset(buf, baseX, baseY + int16(i), elemPtr)
        }
    }
}
```

---

## Expected Performance Gains

| Current | Proposed | Savings |
|---------|----------|---------|
| Create 100 SerialNode structs | No intermediate nodes | ~1200ns |
| Copy 88 bytes × 100 | No copying | ~400ns |
| 3 passes (create, position, render) | 1 pass (render) | ~500ns |
| measureForEach context stack | Pre-computed positions | ~200ns |

**Estimated new time**: 3700ns - 2300ns = **~1400ns** for 100 progress bars

vs direct WriteProgressBar: 1030ns → **only 1.4x overhead** instead of 3.6x

---

## Implementation Checklist

### Phase 1: Static Layout (no ForEach)
- [ ] Define RenderInstruction struct
- [ ] Implement compile-time position calculation
- [ ] Implement fast-path render loop with switch
- [ ] Benchmark vs SerialTemplate for static layouts

### Phase 2: Pointer Bindings
- [ ] Add offset-based text access (KindTextOffset)
- [ ] Add offset-based progress access (KindProgressOffset)
- [ ] Verify zero allocations

### Phase 3: ForEach
- [ ] Implement ForEach with pre-compiled element template
- [ ] Handle nested ForEach via SerialOpForEachOffset pattern
- [ ] Benchmark 10x10 grid

### Phase 4: Containers
- [ ] Handle Row/Col with gap
- [ ] Handle borders
- [ ] Handle flex layouts (may need runtime width distribution)

---

## Key Principles from Glint

1. **Reflection happens ONCE at compile time** - never in the hot path
2. **Flat instruction array** - sequential iteration, cache-friendly
3. **Direct memory access** - `unsafe.Add(p, offset)` instead of intermediate structs
4. **Const-case switch** - enables jump table optimization
5. **Fast paths for common cases** - function pointers only for complex types
6. **Value semantics** - avoid heap escapes in hot path
7. **No intermediate allocations** - write directly to final destination

---

## Worked Example: `top`-like Volatile Data

### Data Model

```go
type TopData struct {
    Load    float32        // changes every frame
    MemPct  float32        // changes every frame
    Uptime  string         // changes slowly
    Procs   []ProcessInfo  // volatile: items appear/disappear/reorder
}

type ProcessInfo struct {
    PID     int
    CPU     float32
    Mem     float32
    Command string
}
```

### UI Definition

```go
ui := Col{Children: []any{
    // Header row - static labels + volatile values
    Row{Children: []any{
        Text{Content: "LOAD: "},
        Text{Content: &data.Load},     // pointer binding
        Text{Content: "  MEM: "},
        Progress{Value: &data.MemPct, BarWidth: 10},
    }},

    // Column headers - completely static
    Text{Content: "  PID  %CPU  %MEM  COMMAND"},

    // Process list - volatile collection
    ForEach(&data.Procs, func(p *ProcessInfo) any {
        return Row{Children: []any{
            Text{Content: &p.PID},      // offset from ProcessInfo base
            Text{Content: &p.CPU},
            Text{Content: &p.Mem},
            Text{Content: &p.Command},
        }}
    }),
}}
```

### What Happens at Compile Time

```go
// Compiled instructions (simplified):
instructions := []RenderInstruction{
    // Row 0: Header
    {Kind: TextStatic,    X: 0,  Y: 0, Text: "LOAD: "},
    {Kind: TextFloat,     X: 6,  Y: 0, FloatOff: offsetof(TopData.Load)},
    {Kind: TextStatic,    X: 12, Y: 0, Text: "  MEM: "},
    {Kind: ProgressFloat, X: 19, Y: 0, Width: 10, FloatOff: offsetof(TopData.MemPct)},

    // Row 1: Column headers (static)
    {Kind: TextStatic,    X: 0,  Y: 1, Text: "  PID  %CPU  %MEM  COMMAND"},

    // Row 2+: ForEach (dynamic)
    {Kind: ForEach,       X: 0,  Y: 2,
        SliceOff:  offsetof(TopData.Procs),
        ElemSize:  sizeof(ProcessInfo),
        RowHeight: 1,
        ElemInstructions: []RenderInstruction{
            // These X positions are relative to row start
            // These offsets are relative to ProcessInfo base
            {Kind: TextInt,    X: 0,  Width: 5,  IntOff: offsetof(ProcessInfo.PID)},
            {Kind: TextFloat,  X: 6,  Width: 5,  FloatOff: offsetof(ProcessInfo.CPU)},
            {Kind: TextFloat,  X: 12, Width: 5,  FloatOff: offsetof(ProcessInfo.Mem)},
            {Kind: TextString, X: 18, Width: 20, StrOff: offsetof(ProcessInfo.Command)},
        },
    },
}
```

### What Happens at Render Time (Every Frame)

```go
func (t *Template) Render(buf *Buffer, dataPtr unsafe.Pointer) {
    for i := range t.instructions {
        inst := &t.instructions[i]

        switch inst.Kind {
        case TextStatic:
            // Static text - just write it
            buf.WriteString(inst.X, inst.Y, inst.Text)

        case TextFloat:
            // Read current value via offset, format, write
            val := *(*float32)(unsafe.Add(dataPtr, inst.FloatOff))
            buf.WriteFloat(inst.X, inst.Y, val, inst.Width)

        case ProgressFloat:
            val := *(*float32)(unsafe.Add(dataPtr, inst.FloatOff))
            buf.WriteProgressBar(inst.X, inst.Y, inst.Width, val)

        case ForEach:
            // Get slice header via offset
            slicePtr := unsafe.Add(dataPtr, inst.SliceOff)
            hdr := *(*sliceHeader)(slicePtr)

            // Render each element
            for j := 0; j < hdr.Len; j++ {
                elemPtr := unsafe.Add(hdr.Data, uintptr(j) * inst.ElemSize)
                y := inst.Y + int16(j) * inst.RowHeight

                // Render element using pre-compiled sub-instructions
                for k := range inst.ElemInstructions {
                    sub := &inst.ElemInstructions[k]
                    switch sub.Kind {
                    case TextInt:
                        val := *(*int)(unsafe.Add(elemPtr, sub.IntOff))
                        buf.WriteInt(inst.X + sub.X, y, val, sub.Width)
                    case TextFloat:
                        val := *(*float32)(unsafe.Add(elemPtr, sub.FloatOff))
                        buf.WriteFloat(inst.X + sub.X, y, val, sub.Width)
                    case TextString:
                        val := *(*string)(unsafe.Add(elemPtr, sub.StrOff))
                        buf.WriteString(inst.X + sub.X, y, val)
                    }
                }
            }
        }
    }
}
```

### Why This Is Fast

1. **Static parts**: Pre-computed X/Y, just write bytes to buffer
2. **Volatile scalars**: Pre-computed X/Y, one pointer dereference to get value
3. **Volatile collections**:
   - Slice header read (1 deref)
   - Per-element: pointer arithmetic + value reads
   - **No intermediate nodes created**
   - **No struct copying**

### What We DON'T Do

```go
// BAD: Current SerialTemplate approach
for each ForEach iteration:
    create SerialNode{Kind, X, Y, W, H, Text, Ratio, ...}  // 88 bytes!
    append to nodes slice  // copy!

then iterate nodes again to render  // second pass!
```

### Performance Estimate for 50-Process `top`

| Operation | Current | Glint-style |
|-----------|---------|-------------|
| Static header | ~100ns | ~50ns (no node creation) |
| 50 process rows | ~1850ns | ~500ns (direct writes) |
| Node creation | ~600ns | **0ns** |
| Node copying | ~200ns | **0ns** |
| **Total** | ~2750ns | **~550ns** |

### Handling Collection Size Changes

When `data.Procs` grows from 50 to 60 items:
- **No recompilation needed** - ForEach reads slice length at runtime
- **Just renders 10 more rows** - same pre-compiled element template
- **Dirty tracking opportunity**: Only clear/redraw rows 50-59

When `data.Procs` shrinks from 50 to 40 items:
- **Renders 40 rows** - same as above
- **Need to clear rows 40-49** - either explicit clear or track dirty region
