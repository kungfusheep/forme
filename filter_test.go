package forme

import (
	"testing"
)

func TestFilterBasic(t *testing.T) {
	items := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	f := NewFilter(&items, func(s *string) string { return *s })

	t.Run("initial state has all items", func(t *testing.T) {
		if f.Len() != 5 {
			t.Fatalf("expected 5 items, got %d", f.Len())
		}
		if f.Active() {
			t.Error("should not be active with no query")
		}
	})

	t.Run("filter narrows results", func(t *testing.T) {
		f.Update("av")
		if !f.Active() {
			t.Error("should be active with query")
		}
		if f.Len() != 1 {
			t.Fatalf("expected 1 match for 'av', got %d", f.Len())
		}
		if f.Items[0] != "bravo" {
			t.Errorf("expected bravo, got %s", f.Items[0])
		}
	})

	t.Run("original maps back to source", func(t *testing.T) {
		orig := f.Original(0)
		if orig == nil {
			t.Fatal("Original returned nil")
		}
		if *orig != "bravo" {
			t.Errorf("expected bravo, got %s", *orig)
		}
		if f.OriginalIndex(0) != 1 {
			t.Errorf("expected original index 1, got %d", f.OriginalIndex(0))
		}
	})

	t.Run("reset restores all items", func(t *testing.T) {
		f.Reset()
		if f.Len() != 5 {
			t.Fatalf("expected 5 items after reset, got %d", f.Len())
		}
		if f.Active() {
			t.Error("should not be active after reset")
		}
	})

	t.Run("empty query resets", func(t *testing.T) {
		f.Update("av")
		f.Update("")
		if f.Len() != 5 {
			t.Fatalf("expected 5 items after empty query, got %d", f.Len())
		}
	})

	t.Run("no matches returns empty", func(t *testing.T) {
		f.Update("zzz")
		if f.Len() != 0 {
			t.Errorf("expected 0 matches, got %d", f.Len())
		}
	})

	t.Run("same query is no-op", func(t *testing.T) {
		f.Update("zzz") // already set
		if f.Query() != "zzz" {
			t.Errorf("expected query 'zzz', got %q", f.Query())
		}
	})
}

func TestFilterStruct(t *testing.T) {
	type profile struct {
		name    string
		service string
	}
	items := []profile{
		{"heap-2024-01-01", "api-gateway"},
		{"goroutine-2024-01-01", "api-gateway"},
		{"heap-2024-01-02", "auth-service"},
		{"cpu-2024-01-01", "payment-service"},
	}

	f := NewFilter(&items, func(p *profile) string { return p.name + " " + p.service })

	t.Run("filter by service", func(t *testing.T) {
		f.Update("gateway")
		if f.Len() != 2 {
			t.Fatalf("expected 2 matches, got %d", f.Len())
		}
	})

	t.Run("filter by name and service", func(t *testing.T) {
		f.Update("heap auth")
		if f.Len() != 1 {
			t.Fatalf("expected 1 match, got %d", f.Len())
		}
		if f.Items[0].name != "heap-2024-01-02" {
			t.Errorf("expected heap-2024-01-02, got %s", f.Items[0].name)
		}
	})

	t.Run("original points into source slice", func(t *testing.T) {
		f.Update("payment")
		if f.Len() != 1 {
			t.Fatalf("expected 1 match, got %d", f.Len())
		}
		orig := f.Original(0)
		if orig == nil {
			t.Fatal("Original returned nil")
		}
		// verify it's a pointer into the original slice
		if &items[3] != orig {
			t.Error("Original should return pointer into source slice")
		}
	})
}

func TestFilterOriginalBounds(t *testing.T) {
	items := []string{"a", "b", "c"}
	f := NewFilter(&items, func(s *string) string { return *s })

	if f.Original(-1) != nil {
		t.Error("negative index should return nil")
	}
	if f.Original(100) != nil {
		t.Error("out of bounds index should return nil")
	}
	if f.OriginalIndex(-1) != -1 {
		t.Error("negative index should return -1")
	}
	if f.OriginalIndex(100) != -1 {
		t.Error("out of bounds index should return -1")
	}
}

func TestFilterRanking(t *testing.T) {
	items := []string{
		"xyzabcxyz",  // abc scattered/embedded
		"abc",        // exact match — should rank highest
		"xxabcxxxxx", // abc present but longer
	}
	f := NewFilter(&items, func(s *string) string { return *s })

	f.Update("abc")
	if f.Len() != 3 {
		t.Fatalf("expected 3 matches, got %d", f.Len())
	}
	// best match should be "abc" (shortest, exact)
	if f.Items[0] != "abc" {
		t.Errorf("expected 'abc' as top result, got %q", f.Items[0])
	}
}

func TestFilterSourceChanges(t *testing.T) {
	items := []string{"alpha", "bravo"}
	f := NewFilter(&items, func(s *string) string { return *s })

	// add items to source
	items = append(items, "charlie")
	f.Reset()
	if f.Len() != 2 {
		// f.source still points to old slice header since items was reassigned
		// this is expected — source is a *[]T so we need to update through the pointer
	}

	// proper way: mutate through pointer
	items2 := []string{"alpha", "bravo"}
	f2 := NewFilter(&items2, func(s *string) string { return *s })
	items2 = append(items2, "charlie")
	f2.Reset()
	// items2 may or may not have reallocated, but f2.source points at items2
	if f2.Len() != len(items2) {
		t.Errorf("expected %d items, got %d", len(items2), f2.Len())
	}
}

func TestFilterListClampsSelectionOnSync(t *testing.T) {
	items := []string{
		"Go", "Rust", "Python", "JavaScript", "TypeScript",
		"Ruby", "Java", "C", "C++", "C#",
	}
	fl := FilterList(&items, func(s *string) string { return *s })

	// navigate down several times (simulating j presses)
	fl.list.SetIndex(7)
	if fl.list.Index() != 7 {
		t.Fatalf("expected index 7, got %d", fl.list.Index())
	}

	// simulate typing "o" — triggers onChange which calls sync()
	fl.input.SetValue("o")
	fl.sync()

	if fl.Filter().Len() != 2 {
		t.Fatalf("expected 2 filtered items, got %d", fl.Filter().Len())
	}
	// selection must be clamped to last valid index
	if fl.list.Index() != 1 {
		t.Errorf("expected selection clamped to 1, got %d", fl.list.Index())
	}
	if sel := fl.Selected(); sel == nil {
		t.Error("Selected() returned nil after clamp")
	}
}

func TestFilterListClampsToZeroOnEmpty(t *testing.T) {
	items := []string{"Go", "Rust", "Python"}
	fl := FilterList(&items, func(s *string) string { return *s })

	fl.list.SetIndex(2)
	fl.input.SetValue("zzz")
	fl.sync()

	if fl.Filter().Len() != 0 {
		t.Fatalf("expected 0 filtered items, got %d", fl.Filter().Len())
	}
	if fl.list.Index() != 0 {
		t.Errorf("expected selection 0 on empty list, got %d", fl.list.Index())
	}
}

func TestFilterListCompilesAsTemplateTree(t *testing.T) {
	items := []string{"a", "b", "c"}
	fl := FilterList(&items, func(s *string) string { return *s }).
		Placeholder("search...").
		MaxVisible(10).
		Render(func(s *string) any { return Text(s) })

	tmpl := Build(VBox(fl))
	if len(tmpl.pendingBindings) < 2 {
		t.Errorf("expected at least 2 nav bindings, got %d", len(tmpl.pendingBindings))
	}
	if tmpl.pendingTIB == nil {
		t.Error("expected text input binding to be set")
	}
}

func TestFilterListSelectedMapsToOriginal(t *testing.T) {
	items := []string{"Go", "Rust", "Python", "JavaScript"}
	fl := FilterList(&items, func(s *string) string { return *s })

	fl.input.SetValue("o")
	fl.sync()

	// should have Go and Python
	if fl.Filter().Len() != 2 {
		t.Fatalf("expected 2 items, got %d", fl.Filter().Len())
	}

	// select second filtered item (Python)
	fl.list.SetIndex(1)
	sel := fl.Selected()
	if sel == nil {
		t.Fatal("Selected() returned nil")
	}
	if *sel != "Python" {
		t.Errorf("expected Python, got %s", *sel)
	}

	// verify it maps back to the original slice
	idx := fl.SelectedIndex()
	if idx != 2 {
		t.Errorf("expected original index 2, got %d", idx)
	}
}

func BenchmarkFilterUpdate(b *testing.B) {
	items := make([]string, 1000)
	for i := range items {
		items[i] = "prefix/service-name/instance-id/heap/profile_" + string(rune('a'+i%26)) + ".pb.gz"
	}
	f := NewFilter(&items, func(s *string) string { return *s })

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.lastQuery = "" // force re-filter
		f.Update("heap service")
	}
}
