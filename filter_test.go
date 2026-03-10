package glyph

import (
	"fmt"
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
		"abc",        // exact match, should rank highest
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
		// this is expected, source is a *[]T so we need to update through the pointer
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

	// simulate typing "o", triggers onChange which calls sync()
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

func TestStreamWriterWrite(t *testing.T) {
	var items []string
	fl := FilterList(&items, func(s *string) string { return *s }).
		Render(func(s *string) any { return Text(s) })

	renders := 0
	w := fl.Stream(func() { renders++ })

	w.Write("alpha")
	w.Write("bravo")
	w.WriteAll([]string{"charlie", "delta"})
	w.Close()

	if fl.Filter().Len() != 4 {
		t.Fatalf("expected 4 items, got %d", fl.Filter().Len())
	}
	if renders < 3 {
		t.Errorf("expected at least 3 render calls, got %d", renders)
	}
}

func TestStreamWriterWithFilter(t *testing.T) {
	var items []string
	fl := FilterList(&items, func(s *string) string { return *s }).
		Render(func(s *string) any { return Text(s) })

	// apply filter before streaming
	fl.input.SetValue("o")
	fl.sync()

	w := fl.Stream(func() {})
	w.Write("Go")     // matches "o"
	w.Write("Rust")   // no match
	w.Write("Python") // matches "o"
	w.Write("Odin")   // matches "o"
	w.Close()

	if fl.Filter().Len() != 3 {
		t.Fatalf("expected 3 filtered items, got %d", fl.Filter().Len())
	}
	if len(items) != 4 {
		t.Fatalf("expected 4 total items in source, got %d", len(items))
	}
}

func TestStreamLifecycle(t *testing.T) {
	var items []string
	fl := FilterList(&items, func(s *string) string { return *s }).
		Render(func(s *string) any { return Text(s) })

	if fl.streaming() {
		t.Fatal("should not be streaming before Stream called")
	}

	w := fl.Stream(func() {})

	if !fl.streaming() {
		t.Fatal("expected streaming=true after Stream called")
	}

	w.Write("item")
	w.Close()

	if fl.streaming() {
		t.Error("expected streaming=false after Close")
	}

	// Close is idempotent
	w.Close()
	if fl.streaming() {
		t.Error("expected streaming=false after second Close")
	}
}

func TestStreamCounterUpdates(t *testing.T) {
	var items []string
	fl := FilterList(&items, func(s *string) string { return *s }).
		Render(func(s *string) any { return Text(s) })

	fl.input.SetValue("a")
	fl.sync()

	w := fl.Stream(func() {})

	w.Write("alpha") // matches "a"
	w.Write("bravo") // matches "a"
	w.Write("echo")  // no match

	if fl.counterMatch != 2 {
		t.Errorf("expected counterMatch=2, got %d", fl.counterMatch)
	}
	if fl.counterTotal != 3 {
		t.Errorf("expected counterTotal=3, got %d", fl.counterTotal)
	}

	w.Close()
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

func BenchmarkStreamWriter(b *testing.B) {
	for _, size := range []int{100, 1000, 5000, 10000} {
		prebuilt := make([]string, size)
		for i := range prebuilt {
			prebuilt[i] = fmt.Sprintf("item-%d-service-name", i)
		}

		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			var items []string
			fl := FilterList(&items, func(s *string) string { return *s }).
				Render(func(s *string) any { return Text(s) })

			fl.input.SetValue("a")
			fl.sync()

			// warm internal slices to steady-state capacity
			w := fl.Stream(func() {})
			w.WriteAll(prebuilt)
			w.Close()

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				// reset source to baseline
				items = items[:0]
				fl.refresh()

				w := fl.Stream(func() {})
				for j := 0; j < size; j++ {
					w.Write(prebuilt[j])
				}
				w.Close()
			}
		})
	}
}
