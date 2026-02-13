# TUI Rendering Architecture

This document describes the rendering model for the TUI framework.

## Overview

The rendering pipeline follows a pattern inspired by jingo/glint:
- **Compile time**: Reflection, pointer offset calculation, op sequence building
- **Execute time**: Pure pointer arithmetic, no reflection, minimal allocation

## Compile: Static vs Dynamic Ops

Compile produces an **instruction stream** containing:

**Static ops** - unconditionally execute:
- `Text` (static string or pointer)
- `Progress` (static value or pointer)
- `ContainerStart` / `ContainerEnd`
- `Layer`, `Leader`, `Custom`, etc.

**Dynamic ops** - control flow with sub-templates:
- `If` - evaluate condition, execute ThenTmpl or ElseTmpl
- `ForEach` - iterate slice, execute IterTmpl for each element

```
Instruction stream example:
  [0] ContainerStart(Col)     ← static
  [1] Text("Header")          ← static
  [2] If(*showDetails)        ← dynamic: has ThenTmpl
  [3] ForEach(*items)         ← dynamic: has IterTmpl
  [4] Text("Footer")          ← static
  [5] ContainerEnd            ← static
```

Sub-templates (ThenTmpl, ElseTmpl, IterTmpl) are themselves compiled instruction streams.

## The Three Phases

```
┌────────────────────────────────────────────────────────┐
│ Phase 1: Width Distribution (top → down)               │
│ Output: W                                              │
└────────────────────────────────────────────────────────┘
                          │
                          ▼
┌────────────────────────────────────────────────────────┐
│ Phase 2: Layout (bottom → up)                          │
│ Output: H, localX, localY                              │
└────────────────────────────────────────────────────────┘
                          │
                          ▼
┌────────────────────────────────────────────────────────┐
│ Phase 3: Render (top → down)                           │
│ Output: pixels in buffer                               │
└────────────────────────────────────────────────────────┘
```

## Phase 1: Width Distribution (top → down)

Parent distributes width to children based on:
- Explicit width (`Width: 20`)
- Percentage width (`WidthPct: 0.5`)
- Flex grow (for remaining space)

```
                    Screen W=40
                         │
            ┌────────────┼────────────┐
            ▼            ▼            ▼
         Col[0]
         W=40
            │
    ┌───────┼───────┬────────────┐
    ▼       ▼       ▼            ▼
Text[1]  Row[2]  ForEach[6]
 W=40     W=40     W=40
            │
      ┌─────┴─────┐
      ▼           ▼
   Text[3]    Progress[4]
    W=30        W=10
```

**Output**: `geom[].W` for all ops

## Phase 2: Layout (bottom → up)

Layout is **owned by the parent**. When a parent does layout:
1. Children have already computed their heights (recursion unwinding)
2. Parent positions children in **local coordinates** (relative to parent origin)
3. Parent derives own height from children's local positions + heights
4. Parent returns height to grandparent

### Key Insight: Local vs Global Coordinates

Layout produces **local positions** - offsets within parent's coordinate space.
Layout does NOT compute global screen positions.

```
Col (parent)
├── Text      localX=0, localY=0, H=1
├── Row       localX=0, localY=1, H=1
│   ├── Text      localX=0,  localY=0, H=1  ← relative to Row
│   └── Progress  localX=30, localY=0, H=1  ← relative to Row
└── ForEach   localX=0, localY=2, H=7
    ├── item[0]   localX=0, localY=0, H=1   ← relative to ForEach
    ├── item[1]   localX=0, localY=1, H=4
    └── item[2]   localX=0, localY=5, H=2
```

### Layout Algorithm

```go
// Col layout (vertical stacking)
func (col *Col) layout() int16 {
    cursor := int16(0)
    for _, child := range col.children {
        child.localX = 0
        child.localY = cursor
        childH := child.layout()  // recurse - child computes its height
        cursor += childH + col.gap
    }
    col.H = cursor
    return col.H
}

// Row layout (horizontal stacking)
func (row *Row) layout() int16 {
    cursor := int16(0)
    maxH := int16(0)
    for _, child := range row.children {
        child.localX = cursor
        child.localY = 0
        childH := child.layout()  // recurse
        cursor += child.W + row.gap
        if childH > maxH {
            maxH = childH
        }
    }
    row.H = maxH
    return row.H
}
```

**Output**: `geom[].H`, `geom[].localX`, `geom[].localY` for all ops

## Phase 3: Render (top → down)

Walk tree from root, accumulating global position via simple addition.
Write to buffer at computed absolute coordinates.

```go
func render(node, globalX, globalY int16, buf *Buffer) {
    absX := globalX + node.localX
    absY := globalY + node.localY

    // Draw this node at absolute position
    node.draw(buf, absX, absY)

    // Recurse to children, passing our absolute position as their origin
    for _, child := range node.children {
        render(child, absX, absY, buf)
    }
}

// Start from root at screen origin
render(root, 0, 0, buf)
```

### Render Trace Example

```
render(Col, globalX=0, globalY=0)
  absX=0+0=0, absY=0+0=0
  │
  ├─► render(Text, 0, 0)
  │   absX=0+0=0, absY=0+0=0  → write "Header" at (0,0)
  │
  ├─► render(Row, 0, 0)
  │   absX=0+0=0, absY=0+1=1
  │   │
  │   ├─► render(Text, 0, 1)
  │   │   absX=0+0=0, absY=1+0=1  → write "Status" at (0,1)
  │   │
  │   └─► render(Progress, 0, 1)
  │       absX=0+30=30, absY=1+0=1  → write bar at (30,1)
  │
  └─► render(ForEach, 0, 0)
      absX=0+0=0, absY=0+2=2
      │
      ├─► render(item[0], 0, 2)
      │   absX=0+0=0, absY=2+0=2  → write at (0,2)
      │
      ├─► render(item[1], 0, 2)
      │   absX=0+0=0, absY=2+1=3  → write at (0,3)
      │
      └─► render(item[2], 0, 2)
          absX=0+0=0, absY=2+5=7  → write at (0,7)
```

**Key characteristics of render phase**:
- No layout decisions
- No height calculations
- Simple arithmetic (addition) to compute global positions
- Just read values and write to buffer

## Control Flow (Dynamic Ops)

Control flow ops (If, ForEach) are **not containers** - they don't do layout. They are transparent directives that expand to 0, 1, or N children which the **parent container** lays out.

### Key Principle

```
Col                       ← container, owns layout
├── Text("Header")        ← static child
├── ForEach(*items)       ← control flow, transparent
│   └── IterTmpl
└── Text("Footer")        ← static child
```

When Col does layout, ForEach is not a positioned child. Col sees:
```
Col's children (after expansion):
├── Text("Header")        ← child 0
├── [item 0]              ← child 1  ┐
├── [item 1]              ← child 2  ├── expanded from ForEach
├── [item 2]              ← child 3  ┘
└── Text("Footer")        ← child 4
```

### Execution Model

Walking the instruction stream, dynamic ops delegate to sub-templates:

```go
for _, op := range ops {
    switch op.Kind {
    case Text, Progress, ...:
        // Static: just execute

    case If:
        // Evaluate condition, execute active branch
        if *op.condPtr {
            op.ThenTmpl.Execute(...)  // sub-template call
        } else if op.ElseTmpl != nil {
            op.ElseTmpl.Execute(...)
        }

    case ForEach:
        // Iterate, execute sub-template for each
        for i := 0; i < sliceLen; i++ {
            elemPtr := getElemPtr(op.slicePtr, i, op.elemSize)
            op.IterTmpl.ExecuteWithData(elemPtr, ...)  // sub-template call
        }
    }
}
```

Sub-template execution is like a function call - it runs all phases and returns geometry.

### Fast Path: Fixed Item Height

If a ForEach's sub-template has fixed height (no nested control flow), detectable at compile time:

```go
type SerialTemplate struct {
    // ...
    fixedHeight int16  // >0 if template always produces same height
}

// During layout, O(1) instead of O(n):
if op.IterTmpl.fixedHeight > 0 {
    totalH = int16(sliceLen) * op.IterTmpl.fixedHeight
}
```

## Data Structures

### Compile-Time (fixed after BuildSerial)

```go
type SerialTemplate struct {
    ops      []SerialOp      // flat operation sequence
    byLevel  [][]int16       // ops indexed by tree depth
}

type SerialOp struct {
    Kind     uint8
    Parent   int16

    // Value access (one used based on Kind)
    StaticStr string
    StrPtr    *string
    StrOff    uintptr
    // ... etc

    // Layout hints
    Width        int16
    PercentWidth float32
    FlexGrow     float32
    Gap          int8
    IsRow        bool

    // ForEach
    IterTmpl  *SerialTemplate
    SlicePtr  unsafe.Pointer
    ElemSize  uintptr
    itemGeoms []itemGeom      // runtime, but reused
}
```

### Runtime (filled each frame, pre-allocated)

```go
type opGeom struct {
    W, H           int16   // dimensions
    localX, localY int16   // position relative to parent
}

// Parallel to ops[], same indices
geom []opGeom
```

## Why This Architecture?

### Compile/Execute Split (jingo pattern)
- All reflection happens once at compile time
- Execute is pure pointer arithmetic
- Pre-allocated arrays, no per-frame allocation

### Local Coordinates
- Layout is decoupled from global positioning
- If parent moves, children don't need re-layout
- Enables future optimizations (dirty tracking, partial re-layout)

### Extensible Layout
- Layout logic owned by parent containers
- Custom layouts can be added without changing render
- Fast path optimizations for common cases (fixed heights)

### Simple Render
- Render phase is trivial: walk tree, add offsets, write bytes
- No decisions, no branching on layout
- Easy to reason about, easy to optimize
