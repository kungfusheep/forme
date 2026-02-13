# TUI Design Principles

Internal notes on what "good defaults" look like for this library.

## Reference

Dense information displays - systems monitors, status panels, diagnostic screens. The kind of UI where every character earns its place.

## Principles

### 1. Density over whitespace

Margins are 1 character, not 4. Padding is optional. If there's empty space, ask whether it's serving a purpose.

### 2. Borders group related information

A box means "these things belong together". Nested boxes show hierarchy. No box means inline flow.

```
┌─ SECTION A ─────────┬─ SECTION B ────┐
│ Related items       │ Other items    │
│ grouped here        │ grouped here   │
└─────────────────────┴────────────────┘
```

### 3. Alignment through leaders

Dot-leaders connect labels to values. The eye can track across without losing the row.

```
RAM 00064K FRAM-HRC..............PASS
MPS 00016K RMU-INIT..............PASS
NVRAM BATTERY 3.2V...............OK
```

### 4. Symbols communicate state

Status lights are faster than words:
- `●` / `○` — on/off
- `▮` / `▯` — filled/empty
- `├──●──────┤` — position on a scale

### 5. Colour is optional

Single-colour UIs work. When colour is used, it should communicate something (alert, selection, hierarchy) not decorate.

### 6. No decoration

If an element isn't information, question whether it's needed.

## Patterns

### Status row with leader
```
LABEL.......................VALUE
```

### LED array
```
[●●○○]
```

### Segmented bar
```
▮▮▮▮▯▯▯▯
```

### Analog meter
```
├──●──────┤
```

### Panel with title
```
┌─ TITLE ──────────────┐
│ content              │
└──────────────────────┘
```

### Tabular data
```
ID  NAME      STATUS
01  ITEM-A    OK
02  ITEM-B    WARN
03  ITEM-C    FAIL
```

## What we're not

- Not trying to be cute or playful
- Not optimising for screenshots
- Not adding visual interest for its own sake

The goal is functional, dense, readable information display.
