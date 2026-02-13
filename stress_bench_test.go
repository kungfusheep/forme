package forme

import "testing"

// Stress test data
var stressData = struct {
	Title     string
	Items     []StressItem
	BigGrid   [][]int
	WideRatio float32
}{
	Title:     "Stress Test Dashboard",
	WideRatio: 0.67,
}

type StressItem struct {
	Name  string
	Value int
	CPU   float32
}

func init() {
	// 100 items for ForEach stress
	stressData.Items = make([]StressItem, 100)
	for i := range stressData.Items {
		stressData.Items[i] = StressItem{
			Name:  "process-" + string(rune('A'+i%26)) + string(rune('0'+i%10)),
			Value: i * 7 % 100,
			CPU:   float32(i%100) / 100.0,
		}
	}

	// 20x20 grid for grid stress
	stressData.BigGrid = make([][]int, 20)
	for i := range stressData.BigGrid {
		stressData.BigGrid[i] = make([]int, 20)
		for j := range stressData.BigGrid[i] {
			stressData.BigGrid[i][j] = (i + j) % 100
		}
	}
}

// BenchmarkStress100Items - 100 ForEach items
func BenchmarkStress100Items(b *testing.B) {
	buf := NewBuffer(80, 120)

	ui := VBoxNode{
		Children: []any{
			TextNode{Content: &stressData.Title},
			ForEach(&stressData.Items, func(item *StressItem) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &item.Name},
					ProgressNode{Value: &item.CPU, BarWidth: 30},
				}}
			}),
		},
	}

	serial := Build(ui)

	buf.Clear()
	serial.Execute(buf, 80, 120)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		serial.Execute(buf, 80, 120)
	}
}

// BenchmarkStressWideProgress - 100-char wide progress bars
func BenchmarkStressWideProgress(b *testing.B) {
	buf := NewBuffer(120, 30)

	items := make([]StressItem, 20)
	for i := range items {
		items[i] = StressItem{
			Name: "wide-" + string(rune('A'+i)),
			CPU:  float32(i) / 20.0,
		}
	}

	ui := VBoxNode{
		Children: []any{
			TextNode{Content: "Wide Progress Bars"},
			ForEach(&items, func(item *StressItem) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &item.Name},
					ProgressNode{Value: &item.CPU, BarWidth: 100},
				}}
			}),
		},
	}

	serial := Build(ui)

	buf.Clear()
	serial.Execute(buf, 120, 30)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		serial.Execute(buf, 120, 30)
	}
}

// BenchmarkStressDenseGrid - Many small items in a grid pattern
func BenchmarkStressDenseGrid(b *testing.B) {
	buf := NewBuffer(100, 50)

	// 10x10 grid of progress bars
	rows := make([][]StressItem, 10)
	for i := range rows {
		rows[i] = make([]StressItem, 10)
		for j := range rows[i] {
			rows[i][j] = StressItem{
				CPU: float32((i*10+j)%100) / 100.0,
			}
		}
	}

	ui := VBoxNode{
		Children: []any{
			TextNode{Content: "Dense Grid"},
			ForEach(&rows, func(row *[]StressItem) any {
				return HBoxNode{Children: []any{
					ForEach(row, func(item *StressItem) any {
						return ProgressNode{Value: &item.CPU, BarWidth: 8}
					}),
				}}
			}),
		},
	}

	serial := Build(ui)

	buf.Clear()
	serial.Execute(buf, 100, 50)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		serial.Execute(buf, 100, 50)
	}
}

// BenchmarkStressHeavyDashboard - Everything combined
func BenchmarkStressHeavyDashboard(b *testing.B) {
	buf := NewBuffer(120, 80)

	// Multiple sections with different content
	cpuCores := stressData.Items[:8]
	memBanks := stressData.Items[8:16]
	procList := stressData.Items[16:50]

	ui := VBoxNode{
		Children: []any{
			TextNode{Content: &stressData.Title},
			TextNode{Content: "═══════════════════════════════════════════════════════════════"},
			TextNode{Content: "CPU Cores"},
			ForEach(&cpuCores, func(item *StressItem) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &item.Name},
					ProgressNode{Value: &item.CPU, BarWidth: 50},
				}}
			}),
			TextNode{Content: ""},
			TextNode{Content: "Memory Banks"},
			ForEach(&memBanks, func(item *StressItem) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &item.Name},
					ProgressNode{Value: &item.CPU, BarWidth: 50},
				}}
			}),
			TextNode{Content: ""},
			TextNode{Content: "Process List"},
			ForEach(&procList, func(item *StressItem) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &item.Name},
					ProgressNode{Value: &item.CPU, BarWidth: 40},
				}}
			}),
		},
	}

	serial := Build(ui)

	buf.Clear()
	serial.Execute(buf, 120, 80)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		serial.Execute(buf, 120, 80)
	}
}

// BenchmarkStressTextHeavy - Lots of text, minimal progress bars
func BenchmarkStressTextHeavy(b *testing.B) {
	buf := NewBuffer(100, 60)

	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "This is line number " + string(rune('0'+i/10)) + string(rune('0'+i%10)) + " with some extra text to make it longer and stress the text rendering path"
	}

	ui := VBoxNode{
		Children: []any{
			TextNode{Content: "Text Heavy Benchmark"},
			ForEach(&lines, func(line *string) any {
				return TextNode{Content: line}
			}),
		},
	}

	serial := Build(ui)

	buf.Clear()
	serial.Execute(buf, 100, 60)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		serial.Execute(buf, 100, 60)
	}
}

// === Async Clear Benchmarks ===

// BenchmarkAsyncClearHeavy - Heavy dashboard with async buffer pool
func BenchmarkAsyncClearHeavy(b *testing.B) {
	pool := NewBufferPool(120, 80)
	defer pool.Stop()

	cpuCores := stressData.Items[:8]
	memBanks := stressData.Items[8:16]
	procList := stressData.Items[16:50]

	ui := VBoxNode{
		Children: []any{
			TextNode{Content: &stressData.Title},
			TextNode{Content: "═══════════════════════════════════════════════════════════════"},
			TextNode{Content: "CPU Cores"},
			ForEach(&cpuCores, func(item *StressItem) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &item.Name},
					ProgressNode{Value: &item.CPU, BarWidth: 50},
				}}
			}),
			TextNode{Content: ""},
			TextNode{Content: "Memory Banks"},
			ForEach(&memBanks, func(item *StressItem) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &item.Name},
					ProgressNode{Value: &item.CPU, BarWidth: 50},
				}}
			}),
			TextNode{Content: ""},
			TextNode{Content: "Process List"},
			ForEach(&procList, func(item *StressItem) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &item.Name},
					ProgressNode{Value: &item.CPU, BarWidth: 40},
				}}
			}),
		},
	}

	serial := Build(ui)

	// Warm up
	buf := pool.Current()
	serial.Execute(buf, 120, 80)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := pool.Swap()  // async clear of old buffer
		serial.Execute(buf, 120, 80)
	}
}

// BenchmarkSyncClearHeavy - Same but with sync clear for comparison
func BenchmarkSyncClearHeavy(b *testing.B) {
	buf := NewBuffer(120, 80)

	cpuCores := stressData.Items[:8]
	memBanks := stressData.Items[8:16]
	procList := stressData.Items[16:50]

	ui := VBoxNode{
		Children: []any{
			TextNode{Content: &stressData.Title},
			TextNode{Content: "═══════════════════════════════════════════════════════════════"},
			TextNode{Content: "CPU Cores"},
			ForEach(&cpuCores, func(item *StressItem) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &item.Name},
					ProgressNode{Value: &item.CPU, BarWidth: 50},
				}}
			}),
			TextNode{Content: ""},
			TextNode{Content: "Memory Banks"},
			ForEach(&memBanks, func(item *StressItem) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &item.Name},
					ProgressNode{Value: &item.CPU, BarWidth: 50},
				}}
			}),
			TextNode{Content: ""},
			TextNode{Content: "Process List"},
			ForEach(&procList, func(item *StressItem) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &item.Name},
					ProgressNode{Value: &item.CPU, BarWidth: 40},
				}}
			}),
		},
	}

	serial := Build(ui)

	// Warm up
	serial.Execute(buf, 120, 80)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()  // sync clear
		serial.Execute(buf, 120, 80)
	}
}

// BenchmarkAsyncClear100Items - 100 items with async clear
func BenchmarkAsyncClear100Items(b *testing.B) {
	pool := NewBufferPool(80, 120)
	defer pool.Stop()

	ui := VBoxNode{
		Children: []any{
			TextNode{Content: &stressData.Title},
			ForEach(&stressData.Items, func(item *StressItem) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &item.Name},
					ProgressNode{Value: &item.CPU, BarWidth: 30},
				}}
			}),
		},
	}

	serial := Build(ui)

	// Warm up
	buf := pool.Current()
	serial.Execute(buf, 80, 120)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := pool.Swap()
		serial.Execute(buf, 80, 120)
	}
}

// BenchmarkSyncClear100Items - 100 items with sync clear for comparison
func BenchmarkSyncClear100Items(b *testing.B) {
	buf := NewBuffer(80, 120)

	ui := VBoxNode{
		Children: []any{
			TextNode{Content: &stressData.Title},
			ForEach(&stressData.Items, func(item *StressItem) any {
				return HBoxNode{Children: []any{
					TextNode{Content: &item.Name},
					ProgressNode{Value: &item.CPU, BarWidth: 30},
				}}
			}),
		},
	}

	serial := Build(ui)

	// Warm up
	serial.Execute(buf, 80, 120)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		serial.Execute(buf, 80, 120)
	}
}

// === Targeted benchmarks for optimized paths ===

// BenchmarkFlexVBox - VBox with flex children (tests scratch slice reuse)
func BenchmarkFlexVBox(b *testing.B) {
	buf := NewBuffer(80, 50)
	content := "flex content"

	ui := VBoxNode{
		Children: []any{
			TextNode{Content: "Header"},
			VBoxNode{Children: []any{
				TextNode{Content: &content},
			}}.Grow(1),
			VBoxNode{Children: []any{
				TextNode{Content: &content},
			}}.Grow(2),
			VBoxNode{Children: []any{
				TextNode{Content: &content},
			}}.Grow(1),
			TextNode{Content: "Footer"},
		},
	}.Height(50)

	serial := Build(ui)
	buf.Clear()
	serial.Execute(buf, 80, 50)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		serial.Execute(buf, 80, 50)
	}
}

// BenchmarkFlexHBox - HBox with flex children (tests scratch slice reuse)
func BenchmarkFlexHBox(b *testing.B) {
	buf := NewBuffer(120, 30)
	content := "flex content"

	ui := HBoxNode{
		Children: []any{
			VBoxNode{Children: []any{TextNode{Content: &content}}}.Grow(1),
			VBoxNode{Children: []any{TextNode{Content: &content}}}.Grow(2),
			VBoxNode{Children: []any{TextNode{Content: &content}}}.Grow(1),
		},
	}

	serial := Build(ui)
	buf.Clear()
	serial.Execute(buf, 120, 30)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		serial.Execute(buf, 120, 30)
	}
}

// BenchmarkBorderedTitles - containers with border titles (tests zero-alloc title render)
func BenchmarkBorderedTitles(b *testing.B) {
	buf := NewBuffer(80, 30)
	val := "100%"

	ui := HBoxNode{
		Children: []any{
			VBoxNode{
				Title:    "STATUS",
				Children: []any{TextNode{Content: &val}},
			}.Border(BorderSingle).Grow(1),
			VBoxNode{
				Title:    "SYSTEMS",
				Children: []any{TextNode{Content: &val}},
			}.Border(BorderSingle).Grow(1),
			VBoxNode{
				Title:    "CAPACITY",
				Children: []any{TextNode{Content: &val}},
			}.Border(BorderSingle).Grow(1),
		},
	}

	serial := Build(ui)
	buf.Clear()
	serial.Execute(buf, 80, 30)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		serial.Execute(buf, 80, 30)
	}
}

// BenchmarkLeaderIntFloat - leader lines with int/float pointers (tests strconv path)
func BenchmarkLeaderIntFloat(b *testing.B) {
	buf := NewBuffer(60, 20)
	cpuVal := 78
	memVal := 4.2
	dskVal := 120

	ui := VBoxNode{
		Children: []any{
			LeaderNode{Label: "CPU USAGE", Value: &cpuVal},
			LeaderNode{Label: "MEMORY", Value: &memVal},
			LeaderNode{Label: "DISK FREE", Value: &dskVal},
			LeaderNode{Label: "CPU USAGE", Value: &cpuVal},
			LeaderNode{Label: "MEMORY", Value: &memVal},
			LeaderNode{Label: "DISK FREE", Value: &dskVal},
		},
	}

	serial := Build(ui)
	buf.Clear()
	serial.Execute(buf, 60, 20)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		serial.Execute(buf, 60, 20)
	}
}

// BenchmarkTreeView - tree rendering (tests scratch prefix reuse)
func BenchmarkTreeView(b *testing.B) {
	buf := NewBuffer(60, 40)

	root := &TreeNode{
		Label:    "root",
		Expanded: true,
		Children: make([]*TreeNode, 5),
	}
	for i := range root.Children {
		child := &TreeNode{
			Label:    "child-" + string(rune('A'+i)),
			Expanded: true,
			Children: make([]*TreeNode, 3),
		}
		for j := range child.Children {
			child.Children[j] = &TreeNode{
				Label:    "leaf-" + string(rune('a'+j)),
				Expanded: false,
			}
		}
		root.Children[i] = child
	}

	ui := TreeView{
		Root:          root,
		ShowRoot:      true,
		Indent:        2,
		ShowLines:     true,
		ExpandedChar:  '▼',
		CollapsedChar: '▶',
		LeafChar:      '·',
	}

	serial := Build(ui)
	buf.Clear()
	serial.Execute(buf, 60, 40)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		serial.Execute(buf, 60, 40)
	}
}

// BenchmarkFillRect - tests optimized FillRect (no border merge for spaces)
func BenchmarkFillRect(b *testing.B) {
	buf := NewBuffer(120, 50)
	cell := Cell{Rune: ' ', Style: Style{BG: Color{Mode: ColorRGB, R: 30, G: 30, B: 60}}}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.FillRect(0, 0, 120, 50, cell)
	}
}
