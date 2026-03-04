package glyph

import (
	"strings"
	"testing"
)

func TestSwitchInsideForEach(t *testing.T) {
	type item struct {
		Label  string
		Active bool
	}

	items := []item{
		{Label: "alpha", Active: false},
		{Label: "beta",  Active: true},
		{Label: "gamma", Active: false},
	}

	view := VBox(
		ForEach(&items, func(it *item) any {
			return If(&it.Active).
				Then(Text(&it.Label).FG(Green)).
				Else(Text(&it.Label).FG(BrightBlack))
		}),
	)

	tmpl := Build(view)
	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	output := buf.String()
	t.Logf("output:\n%s", output)

	// all three labels must appear
	for _, label := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(output, label) {
			t.Errorf("label %q missing from output", label)
		}
	}
}
