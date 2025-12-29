package tui

import "testing"

func TestConditionEq(t *testing.T) {
	t.Run("If comparable Eq true", func(t *testing.T) {
		val := 5
		cond := If(&val).Eq(5)
		if !cond.evaluate() {
			t.Error("expected condition to be true when val == 5")
		}
	})

	t.Run("If comparable Eq false", func(t *testing.T) {
		val := 5
		cond := If(&val).Eq(10)
		if cond.evaluate() {
			t.Error("expected condition to be false when val != 10")
		}
	})

	t.Run("If comparable Ne", func(t *testing.T) {
		val := 5
		cond := If(&val).Ne(10)
		if !cond.evaluate() {
			t.Error("expected condition to be true when val != 10")
		}
	})
}

func TestOrdCondition(t *testing.T) {
	t.Run("Gt true", func(t *testing.T) {
		val := 10
		cond := IfOrd(&val).Gt(5)
		if !cond.evaluate() {
			t.Error("expected 10 > 5 to be true")
		}
	})

	t.Run("Gt false", func(t *testing.T) {
		val := 3
		cond := IfOrd(&val).Gt(5)
		if cond.evaluate() {
			t.Error("expected 3 > 5 to be false")
		}
	})

	t.Run("Lt true", func(t *testing.T) {
		val := 3
		cond := IfOrd(&val).Lt(5)
		if !cond.evaluate() {
			t.Error("expected 3 < 5 to be true")
		}
	})

	t.Run("Gte", func(t *testing.T) {
		val := 5
		cond := IfOrd(&val).Gte(5)
		if !cond.evaluate() {
			t.Error("expected 5 >= 5 to be true")
		}
	})

	t.Run("Lte", func(t *testing.T) {
		val := 5
		cond := IfOrd(&val).Lte(5)
		if !cond.evaluate() {
			t.Error("expected 5 <= 5 to be true")
		}
	})
}

func TestConditionThenElse(t *testing.T) {
	t.Run("Then branch accessible", func(t *testing.T) {
		val := true
		cond := If(&val).Eq(true).Then("yes").Else("no")
		if cond.getThen() != "yes" {
			t.Error("expected then to be 'yes'")
		}
		if cond.getElse() != "no" {
			t.Error("expected else to be 'no'")
		}
	})

	t.Run("Evaluates dynamically", func(t *testing.T) {
		val := 0
		cond := IfOrd(&val).Eq(0)

		if !cond.evaluate() {
			t.Error("expected true when val == 0")
		}

		val = 1
		if cond.evaluate() {
			t.Error("expected false when val == 1")
		}

		val = 0
		if !cond.evaluate() {
			t.Error("expected true again when val == 0")
		}
	})
}

func TestConditionInSerialTemplate(t *testing.T) {
	t.Run("If renders correct branch", func(t *testing.T) {
		activeLayer := 0

		view := Col{Children: []any{
			IfOrd(&activeLayer).Eq(0).Then(Text{Content: "LAYER0"}).Else(Text{Content: "OTHER"}),
			IfOrd(&activeLayer).Eq(1).Then(Text{Content: "LAYER1"}).Else(Text{Content: "OTHER"}),
		}}

		tmpl := BuildSerial(view)
		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		// Check first line has "LAYER0"
		line0 := extractLine(buf, 0, 10)
		if line0 != "LAYER0    " {
			t.Errorf("expected 'LAYER0    ', got %q", line0)
		}

		// Check second line has "OTHER" (since activeLayer != 1)
		line1 := extractLine(buf, 1, 10)
		if line1 != "OTHER     " {
			t.Errorf("expected 'OTHER     ', got %q", line1)
		}

		// Now change activeLayer and re-render
		activeLayer = 1
		buf.Clear()
		tmpl.Execute(buf, 20, 5)

		// Check first line now has "OTHER"
		line0 = extractLine(buf, 0, 10)
		if line0 != "OTHER     " {
			t.Errorf("after change: expected 'OTHER     ', got %q", line0)
		}

		// Check second line now has "LAYER1"
		line1 = extractLine(buf, 1, 10)
		if line1 != "LAYER1    " {
			t.Errorf("after change: expected 'LAYER1    ', got %q", line1)
		}
	})
}

// extractLine returns the text content from a buffer row
func extractLine(buf *Buffer, row, width int) string {
	result := make([]rune, width)
	for x := 0; x < width; x++ {
		result[x] = buf.Get(x, row).Rune
	}
	return string(result)
}

func TestRowLayout(t *testing.T) {
	t.Run("Row places children horizontally", func(t *testing.T) {
		view := Row{Children: []any{
			Text{Content: "AAA"},
			Text{Content: "BBB"},
			Text{Content: "CCC"},
		}}

		tmpl := BuildSerial(view)
		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		// All three texts should be on row 0, horizontally adjacent
		line := extractLine(buf, 0, 12)
		if line != "AAABBBCCC   " {
			t.Errorf("expected 'AAABBBCCC   ', got %q", line)
		}
	})

	t.Run("Row with gap", func(t *testing.T) {
		view := Row{Gap: 2, Children: []any{
			Text{Content: "AA"},
			Text{Content: "BB"},
		}}

		tmpl := BuildSerial(view)
		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		// "AA" then 2 spaces gap then "BB"
		line := extractLine(buf, 0, 10)
		if line != "AA  BB    " {
			t.Errorf("expected 'AA  BB    ', got %q", line)
		}
	})

	t.Run("Col places children vertically", func(t *testing.T) {
		view := Col{Children: []any{
			Text{Content: "AAA"},
			Text{Content: "BBB"},
		}}

		tmpl := BuildSerial(view)
		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		line0 := extractLine(buf, 0, 5)
		line1 := extractLine(buf, 1, 5)
		if line0 != "AAA  " {
			t.Errorf("row 0: expected 'AAA  ', got %q", line0)
		}
		if line1 != "BBB  " {
			t.Errorf("row 1: expected 'BBB  ', got %q", line1)
		}
	})

	t.Run("Nested Row in Col", func(t *testing.T) {
		view := Col{Children: []any{
			Row{Children: []any{
				Text{Content: "A"},
				Text{Content: "B"},
			}},
			Text{Content: "C"},
		}}

		tmpl := BuildSerial(view)
		buf := NewBuffer(20, 5)
		tmpl.Execute(buf, 20, 5)

		line0 := extractLine(buf, 0, 5)
		line1 := extractLine(buf, 1, 5)
		if line0 != "AB   " {
			t.Errorf("row 0: expected 'AB   ', got %q", line0)
		}
		if line1 != "C    " {
			t.Errorf("row 1: expected 'C    ', got %q", line1)
		}
	})
}
