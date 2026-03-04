package glyph

import (
	"fmt"
	"testing"
)

// TestSwitchInHBoxForEachLayout isolates the exact scenario: Switch as an HBox
// child inside ForEach, where different elements match different cases with
// different widths. We print geom state after layout and after render to see
// exactly what's happening.
func TestSwitchInHBoxForEachLayout(t *testing.T) {
	type item struct {
		Name   string
		Status string
	}

	items := []item{
		{Name: "alpha", Status: "ok"},   // Default: "  ok" = 4 chars
		{Name: "beta", Status: "warn"},  // Case:    "! warn" = 6 chars
		{Name: "gamma", Status: "ok"},   // Default: "  ok" = 4 chars
	}

	view := HBox(
		ForEach(&items, func(it *item) any {
			return HBox.Gap(1)(
				Text(&it.Name),
				Switch(&it.Status).
					Case("warn", Text("! warn")).
					Default(Text("  ok")),
			)
		}),
	)

	tmpl := Build(view)
	buf := NewBuffer(60, 5)
	tmpl.Execute(buf, 60, 5)

	// Find the ForEach op and its iter template
	var forEachOp *Op
	for i := range tmpl.ops {
		if tmpl.ops[i].Kind == OpForEach {
			forEachOp = &tmpl.ops[i]
			break
		}
	}
	if forEachOp == nil {
		t.Fatal("no ForEach op found")
	}

	iterTmpl := forEachOp.IterTmpl

	fmt.Println("=== iter template geom after layout (last-element state) ===")
	for i, op := range iterTmpl.ops {
		g := iterTmpl.geom[i]
		fmt.Printf("  op[%d] kind=%-12d parent=%2d  LocalX=%d LocalY=%d W=%d H=%d\n",
			i, op.Kind, op.Parent, g.LocalX, g.LocalY, g.W, g.H)
	}

	fmt.Println("\n=== iterGeoms (per-element Y positions) ===")
	for i, g := range forEachOp.iterGeoms {
		fmt.Printf("  item[%d] %q  LocalX=%d LocalY=%d W=%d H=%d\n",
			i, items[i].Name, g.LocalX, g.LocalY, g.W, g.H)
	}

	fmt.Println("\n=== Switch case geoms (case[0]=warn, def=ok) ===")
	for i, op := range iterTmpl.ops {
		if op.Kind == OpSwitch {
			fmt.Printf("  Switch op[%d] geom W=%d\n", i, iterTmpl.geom[i].W)
			for j, ct := range op.SwitchCases {
				if ct != nil && len(ct.geom) > 0 {
					fmt.Printf("    case[%d] template geom[0] W=%d\n", j, ct.geom[0].W)
				}
			}
			if op.SwitchDef != nil && len(op.SwitchDef.geom) > 0 {
				fmt.Printf("    default template geom[0] W=%d\n", op.SwitchDef.geom[0].W)
			}
		}
	}

	fmt.Printf("\n=== rendered output ===\n%s\n", buf.String())

	// assert both status strings appear
	output := buf.String()
	if !felhContains(output, "! warn") {
		t.Errorf("expected '! warn' in output, got:\n%s", output)
	}
	if !felhContains(output, "  ok") {
		t.Errorf("expected '  ok' in output, got:\n%s", output)
	}
}

func felhContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
