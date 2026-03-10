package glyph

import "strings"

// Display helpers for common UI patterns.

// LeaderStr creates a dot-leader string: "LABEL...........VALUE"
// Width is the total width including label and value.
// Deprecated: Use the Leader component for pointer binding support.
func LeaderStr(label, value string, width int) string {
	dots := width - len(label) - len(value)
	if dots < 1 {
		dots = 1
	}
	return label + strings.Repeat(".", dots) + value
}

// LeaderDash creates a dash-leader string: "LABEL-----------VALUE"
func LeaderDash(label, value string, width int) string {
	dashes := width - len(label) - len(value)
	if dashes < 1 {
		dashes = 1
	}
	return label + strings.Repeat("-", dashes) + value
}

// LED returns a single LED indicator: ● (on) or ○ (off)
func LED(on bool) string {
	if on {
		return "●"
	}
	return "○"
}

// LEDs returns multiple LED indicators: ●●○○
func LEDs(states ...bool) string {
	var b strings.Builder
	for _, on := range states {
		if on {
			b.WriteRune('●')
		} else {
			b.WriteRune('○')
		}
	}
	return b.String()
}

// LEDsBracket returns bracketed LED indicators: [●●○○]
func LEDsBracket(states ...bool) string {
	return "[" + LEDs(states...) + "]"
}

// Bar returns a segmented bar: ▮▮▮▯▯
func Bar(filled, total int) string {
	var b strings.Builder
	for i := range total {
		if i < filled {
			b.WriteRune('▮')
		} else {
			b.WriteRune('▯')
		}
	}
	return b.String()
}

// BarBracket returns a bracketed bar: [▮▮▮▯▯]
func BarBracket(filled, total int) string {
	return "[" + Bar(filled, total) + "]"
}

// Meter returns an analog-style meter: ├──●──────┤
func Meter(value, max, width int) string {
	if width < 3 {
		width = 3
	}
	inner := width - 2 // Account for ├ and ┤
	pos := 0
	if max > 0 {
		pos = (value * (inner - 1)) / max
	}
	if pos >= inner {
		pos = inner - 1
	}
	if pos < 0 {
		pos = 0
	}

	var b strings.Builder
	b.WriteRune('├')
	for i := range inner {
		if i == pos {
			b.WriteRune('●')
		} else {
			b.WriteRune('─')
		}
	}
	b.WriteRune('┤')
	return b.String()
}

// DrawPanel draws a bordered panel with title and returns the interior region.
// Title appears in the top border: ┌─ TITLE ─────┐
func (b *Buffer) DrawPanel(x, y, w, h int, title string, style Style) *Region {
	b.DrawBorder(x, y, w, h, BorderSingle, style)

	if title != "" {
		titleStr := string(BorderSingle.Horizontal) + " " + title + " "
		b.WriteString(x+1, y, titleStr, style)
	}

	return b.Region(x+1, y+1, w-2, h-2)
}

// DrawPanelEx draws a panel with custom border style.
func (b *Buffer) DrawPanelEx(x, y, w, h int, title string, border BorderStyle, style Style) *Region {
	b.DrawBorder(x, y, w, h, border, style)

	if title != "" {
		titleStr := string(border.Horizontal) + " " + title + " "
		b.WriteString(x+1, y, titleStr, style)
	}

	return b.Region(x+1, y+1, w-2, h-2)
}
