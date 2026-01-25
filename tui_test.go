package tui

import "testing"

func TestAttribute(t *testing.T) {
	t.Run("Has", func(t *testing.T) {
		attr := AttrBold | AttrItalic
		if !attr.Has(AttrBold) {
			t.Error("expected attr to have Bold")
		}
		if !attr.Has(AttrItalic) {
			t.Error("expected attr to have Italic")
		}
		if attr.Has(AttrUnderline) {
			t.Error("expected attr to not have Underline")
		}
	})

	t.Run("With", func(t *testing.T) {
		attr := AttrBold
		attr = attr.With(AttrItalic)
		if !attr.Has(AttrBold) || !attr.Has(AttrItalic) {
			t.Error("expected attr to have both Bold and Italic")
		}
	})

	t.Run("Without", func(t *testing.T) {
		attr := AttrBold | AttrItalic
		attr = attr.Without(AttrBold)
		if attr.Has(AttrBold) {
			t.Error("expected attr to not have Bold")
		}
		if !attr.Has(AttrItalic) {
			t.Error("expected attr to still have Italic")
		}
	})
}

func TestColor(t *testing.T) {
	t.Run("DefaultColor", func(t *testing.T) {
		c := DefaultColor()
		if c.Mode != ColorDefault {
			t.Errorf("expected ColorDefault, got %v", c.Mode)
		}
	})

	t.Run("BasicColor", func(t *testing.T) {
		c := BasicColor(5)
		if c.Mode != Color16 || c.Index != 5 {
			t.Errorf("expected Color16 with index 5, got %v/%d", c.Mode, c.Index)
		}
	})

	t.Run("PaletteColor", func(t *testing.T) {
		c := PaletteColor(200)
		if c.Mode != Color256 || c.Index != 200 {
			t.Errorf("expected Color256 with index 200, got %v/%d", c.Mode, c.Index)
		}
	})

	t.Run("RGB", func(t *testing.T) {
		c := RGB(255, 128, 64)
		if c.Mode != ColorRGB || c.R != 255 || c.G != 128 || c.B != 64 {
			t.Errorf("expected RGB(255,128,64), got %+v", c)
		}
	})

	t.Run("Hex", func(t *testing.T) {
		c := Hex(0xFF8040)
		if c.Mode != ColorRGB || c.R != 255 || c.G != 128 || c.B != 64 {
			t.Errorf("expected RGB(255,128,64), got %+v", c)
		}
	})

	t.Run("Equal", func(t *testing.T) {
		c1 := RGB(100, 100, 100)
		c2 := RGB(100, 100, 100)
		c3 := RGB(100, 100, 101)

		if !c1.Equal(c2) {
			t.Error("expected c1 and c2 to be equal")
		}
		if c1.Equal(c3) {
			t.Error("expected c1 and c3 to not be equal")
		}
	})
}

func TestStyle(t *testing.T) {
	t.Run("DefaultStyle", func(t *testing.T) {
		s := DefaultStyle()
		if s.FG.Mode != ColorDefault || s.BG.Mode != ColorDefault {
			t.Error("expected default colors")
		}
		if s.Attr != AttrNone {
			t.Error("expected no attributes")
		}
	})

	t.Run("Chaining", func(t *testing.T) {
		s := DefaultStyle().
			Foreground(Red).
			Background(Blue).
			Bold().
			Italic()

		if !s.FG.Equal(Red) {
			t.Error("expected Red foreground")
		}
		if !s.BG.Equal(Blue) {
			t.Error("expected Blue background")
		}
		if !s.Attr.Has(AttrBold) || !s.Attr.Has(AttrItalic) {
			t.Error("expected Bold and Italic attributes")
		}
	})

	t.Run("Equal", func(t *testing.T) {
		s1 := DefaultStyle().Foreground(Red).Bold()
		s2 := DefaultStyle().Foreground(Red).Bold()
		s3 := DefaultStyle().Foreground(Red)

		if !s1.Equal(s2) {
			t.Error("expected s1 and s2 to be equal")
		}
		if s1.Equal(s3) {
			t.Error("expected s1 and s3 to not be equal")
		}
	})
}

func TestCell(t *testing.T) {
	t.Run("EmptyCell", func(t *testing.T) {
		c := EmptyCell()
		if c.Rune != ' ' {
			t.Errorf("expected space, got %q", c.Rune)
		}
	})

	t.Run("NewCell", func(t *testing.T) {
		style := DefaultStyle().Foreground(Red)
		c := NewCell('X', style)
		if c.Rune != 'X' || !c.Style.Equal(style) {
			t.Error("cell not created correctly")
		}
	})

	t.Run("Equal", func(t *testing.T) {
		c1 := NewCell('A', DefaultStyle().Foreground(Red))
		c2 := NewCell('A', DefaultStyle().Foreground(Red))
		c3 := NewCell('B', DefaultStyle().Foreground(Red))

		if !c1.Equal(c2) {
			t.Error("expected c1 and c2 to be equal")
		}
		if c1.Equal(c3) {
			t.Error("expected c1 and c3 to not be equal")
		}
	})
}

func TestSetViewLimit(t *testing.T) {
	// helper to simulate SetView counter logic without needing a real screen
	checkLimit := func(app *App) {
		app.setViewCount++
		if app.setViewLimit > 0 && app.setViewCount > app.setViewLimit {
			panic("SetView limit exceeded")
		}
	}

	t.Run("unlimited by default", func(t *testing.T) {
		app := &App{}
		// Should not panic - unlimited by default
		checkLimit(app)
		checkLimit(app)
		checkLimit(app)
		if app.setViewCount != 3 {
			t.Errorf("expected setViewCount=3, got %d", app.setViewCount)
		}
	})

	t.Run("limit of 1 allows single call", func(t *testing.T) {
		app := &App{}
		app.SetViewLimit(1)
		checkLimit(app)
		if app.setViewCount != 1 {
			t.Errorf("expected setViewCount=1, got %d", app.setViewCount)
		}
	})

	t.Run("limit of 1 panics on second call", func(t *testing.T) {
		app := &App{}
		app.SetViewLimit(1)
		checkLimit(app)

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on second SetView call")
			}
		}()
		checkLimit(app) // Should panic
	})

	t.Run("limit of 2 allows two calls", func(t *testing.T) {
		app := &App{}
		app.SetViewLimit(2)
		checkLimit(app)
		checkLimit(app)
		if app.setViewCount != 2 {
			t.Errorf("expected setViewCount=2, got %d", app.setViewCount)
		}
	})
}
