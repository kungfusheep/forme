package tui

import (
	"testing"
)

// BenchmarkBuildSimple measures compile time for a simple template.
func BenchmarkBuildSimple(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Build(VBoxNode{Children: []any{
			TextNode{Content: "Header"},
			TextNode{Content: "Body"},
			TextNode{Content: "Footer"},
		}})
	}
}

// BenchmarkBuildNested measures compile time for nested containers.
func BenchmarkBuildNested(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Build(VBoxNode{Children: []any{
			TextNode{Content: "Header"},
			HBoxNode{Children: []any{
				VBoxNode{Children: []any{
					TextNode{Content: "Left 1"},
					TextNode{Content: "Left 2"},
				}},
				VBoxNode{Children: []any{
					TextNode{Content: "Right 1"},
					TextNode{Content: "Right 2"},
				}},
			}},
			TextNode{Content: "Footer"},
		}})
	}
}

// BenchmarkBuildForEach measures compile time with ForEach.
func BenchmarkBuildForEach(b *testing.B) {
	type Item struct {
		Name string
	}
	items := make([]Item, 100)
	for i := range items {
		items[i].Name = "Item"
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Build(VBoxNode{Children: []any{
			TextNode{Content: "Header"},
			ForEachNode{
				Items: &items,
				Render: func(item *Item) any {
					return TextNode{Content: &item.Name}
				},
			},
		}})
	}
}

// BenchmarkV2ExecuteSimple measures execute time for a simple template.
func BenchmarkV2ExecuteSimple(b *testing.B) {
	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		TextNode{Content: "Body"},
		TextNode{Content: "Footer"},
	}})
	buf := NewBuffer(80, 24)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Clear()
		tmpl.Execute(buf, 80, 24)
	}
}

// BenchmarkV2ExecuteNested measures execute time for nested containers.
func BenchmarkV2ExecuteNested(b *testing.B) {
	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		HBoxNode{Children: []any{
			VBoxNode{Children: []any{
				TextNode{Content: "Left 1"},
				TextNode{Content: "Left 2"},
			}},
			VBoxNode{Children: []any{
				TextNode{Content: "Right 1"},
				TextNode{Content: "Right 2"},
			}},
		}},
		TextNode{Content: "Footer"},
	}})
	buf := NewBuffer(80, 24)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Clear()
		tmpl.Execute(buf, 80, 24)
	}
}

// BenchmarkV2ExecuteDynamic measures execute time with dynamic text.
func BenchmarkV2ExecuteDynamic(b *testing.B) {
	title := "Dynamic Title"
	status := "Running..."
	count := 42

	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: &title},
		TextNode{Content: &status},
		ProgressNode{Value: &count, BarWidth: 20},
	}})
	buf := NewBuffer(80, 24)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Clear()
		tmpl.Execute(buf, 80, 24)
	}
}

// BenchmarkV2ExecuteForEach10 measures execute time with 10 items.
func BenchmarkV2ExecuteForEach10(b *testing.B) {
	benchmarkV2ForEach(b, 10)
}

// BenchmarkV2ExecuteForEach100 measures execute time with 100 items.
func BenchmarkV2ExecuteForEach100(b *testing.B) {
	benchmarkV2ForEach(b, 100)
}

// BenchmarkV2ExecuteForEach1000 measures execute time with 1000 items.
func BenchmarkV2ExecuteForEach1000(b *testing.B) {
	benchmarkV2ForEach(b, 1000)
}

func benchmarkV2ForEach(b *testing.B, n int) {
	type Item struct {
		Name string
	}
	items := make([]Item, n)
	for i := range items {
		items[i].Name = "Item"
	}

	tmpl := Build(VBoxNode{Children: []any{
		ForEachNode{
			Items: &items,
			Render: func(item *Item) any {
				return TextNode{Content: &item.Name}
			},
		},
	}})
	buf := NewBuffer(80, n+10)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Clear()
		tmpl.Execute(buf, 80, int16(n)+10)
	}
}

// BenchmarkV2ExecuteIf measures execute time with conditional.
func BenchmarkV2ExecuteIf(b *testing.B) {
	show := true
	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		IfNode{
			Cond: &show,
			Then: VBoxNode{Children: []any{
				TextNode{Content: "Detail 1"},
				TextNode{Content: "Detail 2"},
				TextNode{Content: "Detail 3"},
			}},
		},
		TextNode{Content: "Footer"},
	}})
	buf := NewBuffer(80, 24)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		show = i%2 == 0 // Toggle condition
		buf.Clear()
		tmpl.Execute(buf, 80, 24)
	}
}

// BenchmarkV2ExecuteComplex measures a realistic complex layout.
func BenchmarkV2ExecuteComplex(b *testing.B) {
	type Task struct {
		Name   string
		Status string
	}
	tasks := []Task{
		{Name: "Build project", Status: "Done"},
		{Name: "Run tests", Status: "Running"},
		{Name: "Deploy", Status: "Pending"},
		{Name: "Monitor", Status: "Pending"},
		{Name: "Cleanup", Status: "Pending"},
	}
	title := "Task Manager"
	showCompleted := true
	progress := 40

	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: &title},
		HBoxNode{Gap: 2, Children: []any{
			TextNode{Content: "Status:"},
			ProgressNode{Value: &progress, BarWidth: 20},
		}},
		IfNode{
			Cond: &showCompleted,
			Then: TextNode{Content: "Showing all tasks"},
		},
		ForEachNode{
			Items: &tasks,
			Render: func(t *Task) any {
				return HBoxNode{Gap: 1, Children: []any{
					TextNode{Content: &t.Name},
					TextNode{Content: &t.Status},
				}}
			},
		},
	}})
	buf := NewBuffer(80, 24)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Clear()
		tmpl.Execute(buf, 80, 24)
	}
}

// BenchmarkV2WidthDistribution measures just the width phase.
func BenchmarkV2WidthDistribution(b *testing.B) {
	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		HBoxNode{Children: []any{
			VBoxNode{Children: []any{
				TextNode{Content: "Left 1"},
				TextNode{Content: "Left 2"},
			}},
			VBoxNode{Children: []any{
				TextNode{Content: "Right 1"},
				TextNode{Content: "Right 2"},
			}},
		}},
		TextNode{Content: "Footer"},
	}})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmpl.distributeWidths(80, nil)
	}
}

// BenchmarkV2Layout measures just the layout phase.
func BenchmarkV2Layout(b *testing.B) {
	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		HBoxNode{Children: []any{
			VBoxNode{Children: []any{
				TextNode{Content: "Left 1"},
				TextNode{Content: "Left 2"},
			}},
			VBoxNode{Children: []any{
				TextNode{Content: "Right 1"},
				TextNode{Content: "Right 2"},
			}},
		}},
		TextNode{Content: "Footer"},
	}})
	tmpl.distributeWidths(80, nil) // Need widths first

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmpl.layout(24)
	}
}

// BenchmarkV2Render measures just the render phase.
func BenchmarkV2Render(b *testing.B) {
	tmpl := Build(VBoxNode{Children: []any{
		TextNode{Content: "Header"},
		HBoxNode{Children: []any{
			VBoxNode{Children: []any{
				TextNode{Content: "Left 1"},
				TextNode{Content: "Left 2"},
			}},
			VBoxNode{Children: []any{
				TextNode{Content: "Right 1"},
				TextNode{Content: "Right 2"},
			}},
		}},
		TextNode{Content: "Footer"},
	}})
	tmpl.distributeWidths(80, nil)
	tmpl.layout(24)
	buf := NewBuffer(80, 24)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Clear()
		tmpl.render(buf, 0, 0, 80)
	}
}
