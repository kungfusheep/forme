package glyph

import "unicode/utf8"

type TextViewC struct {
	content          *string
	layer            *Layer
	grow             float32
	margin           [4]int16
	lastContent      string
	lastWidth        int
	declaredBindings []binding
}

// TextView creates a scrollable multi-line text display with word wrapping.
// Content is re-wrapped automatically when the text or viewport width changes.
func TextView(content *string) *TextViewC {
	tv := &TextViewC{
		content: content,
		layer:   NewLayer(),
	}
	tv.layer.AlwaysRender = true
	tv.layer.Render = tv.sync
	return tv
}

// Grow sets the flex grow factor so the view expands to fill available space.
func (tv *TextViewC) Grow(g float32) *TextViewC {
	tv.grow = g
	return tv
}

// Margin sets uniform margin on all sides.
func (tv *TextViewC) Margin(all int16) *TextViewC {
	tv.margin = [4]int16{all, all, all, all}
	return tv
}

// MarginVH sets vertical and horizontal margin.
func (tv *TextViewC) MarginVH(v, h int16) *TextViewC {
	tv.margin = [4]int16{v, h, v, h}
	return tv
}

// MarginTRBL sets individual margins for top, right, bottom, left.
func (tv *TextViewC) MarginTRBL(t, r, b, l int16) *TextViewC {
	tv.margin = [4]int16{t, r, b, l}
	return tv
}

// Layer returns the underlying layer for external scroll wiring.
func (tv *TextViewC) Layer() *Layer { return tv.layer }

// BindScroll registers keys for line-by-line scrolling.
func (tv *TextViewC) BindScroll(down, up string) *TextViewC {
	tv.declaredBindings = append(tv.declaredBindings,
		binding{pattern: down, handler: func() { tv.layer.ScrollDown(1) }},
		binding{pattern: up, handler: func() { tv.layer.ScrollUp(1) }},
	)
	return tv
}

// BindPageScroll registers keys for half-page scrolling.
func (tv *TextViewC) BindPageScroll(down, up string) *TextViewC {
	tv.declaredBindings = append(tv.declaredBindings,
		binding{pattern: down, handler: func() { tv.layer.HalfPageDown() }},
		binding{pattern: up, handler: func() { tv.layer.HalfPageUp() }},
	)
	return tv
}

func (tv *TextViewC) bindings() []binding { return tv.declaredBindings }

func (tv *TextViewC) sync() {
	c := *tv.content
	w := tv.layer.ViewportWidth()
	if w <= 0 {
		return
	}
	if c == tv.lastContent && w == tv.lastWidth {
		return
	}
	tv.lastContent = c
	tv.lastWidth = w

	lines := wrapText(c, w)
	if len(lines) == 0 {
		lines = []string{""}
	}
	h := max(len(lines), tv.layer.ViewportHeight())
	buf := NewBuffer(w, h)
	for i, line := range lines {
		buf.WriteStringFast(0, i, line, Style{}, w)
	}
	tv.layer.SetBuffer(buf)
}

func (t *Template) compileTextViewC(v *TextViewC, parent int16, depth int) int16 {
	layerView := LayerView(v.layer).Grow(v.grow)
	if v.margin != [4]int16{} {
		layerView = layerView.MarginTRBL(v.margin[0], v.margin[1], v.margin[2], v.margin[3])
	}
	return t.compileLayerViewC(layerView, parent, depth)
}

// wrapText splits on newlines, expands tabs, then character-wraps lines exceeding width.
func wrapText(s string, width int) []string {
	if width <= 0 {
		return nil
	}
	const tabWidth = 4
	var out []string
	line := make([]byte, 0, width)
	col := 0

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size

		if r == '\n' {
			out = append(out, string(line))
			line = line[:0]
			col = 0
			continue
		}

		if r == '\t' {
			spaces := tabWidth - (col % tabWidth)
			for j := 0; j < spaces; j++ {
				if col >= width {
					out = append(out, string(line))
					line = line[:0]
					col = 0
				}
				line = append(line, ' ')
				col++
			}
			continue
		}

		if col >= width {
			out = append(out, string(line))
			line = line[:0]
			col = 0
		}

		line = utf8.AppendRune(line, r)
		col++
	}
	out = append(out, string(line))
	return out
}
