package glyph

import (
	"testing"
	"time"
)

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

	ui := VBox(
		Text(&stressData.Title),
		ForEach(&stressData.Items, func(item *StressItem) any {
			return HBox(
				Text(&item.Name),
				Progress(&item.CPU).Width(30),
			)
		}),
	)

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

	ui := VBox(
		Text("Wide Progress Bars"),
		ForEach(&items, func(item *StressItem) any {
			return HBox(
				Text(&item.Name),
				Progress(&item.CPU).Width(100),
			)
		}),
	)

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

	ui := VBox(
		Text("Dense Grid"),
		ForEach(&rows, func(row *[]StressItem) any {
			return HBox(
				ForEach(row, func(item *StressItem) any {
					return Progress(&item.CPU).Width(8)
				}),
			)
		}),
	)

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

	ui := VBox(
		Text(&stressData.Title),
		Text("═══════════════════════════════════════════════════════════════"),
		Text("CPU Cores"),
		ForEach(&cpuCores, func(item *StressItem) any {
			return HBox(
				Text(&item.Name),
				Progress(&item.CPU).Width(50),
			)
		}),
		Text(""),
		Text("Memory Banks"),
		ForEach(&memBanks, func(item *StressItem) any {
			return HBox(
				Text(&item.Name),
				Progress(&item.CPU).Width(50),
			)
		}),
		Text(""),
		Text("Process List"),
		ForEach(&procList, func(item *StressItem) any {
			return HBox(
				Text(&item.Name),
				Progress(&item.CPU).Width(40),
			)
		}),
	)

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

	ui := VBox(
		Text("Text Heavy Benchmark"),
		ForEach(&lines, func(line *string) any {
			return Text(line)
		}),
	)

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

	ui := VBox(
		Text(&stressData.Title),
		Text("═══════════════════════════════════════════════════════════════"),
		Text("CPU Cores"),
		ForEach(&cpuCores, func(item *StressItem) any {
			return HBox(
				Text(&item.Name),
				Progress(&item.CPU).Width(50),
			)
		}),
		Text(""),
		Text("Memory Banks"),
		ForEach(&memBanks, func(item *StressItem) any {
			return HBox(
				Text(&item.Name),
				Progress(&item.CPU).Width(50),
			)
		}),
		Text(""),
		Text("Process List"),
		ForEach(&procList, func(item *StressItem) any {
			return HBox(
				Text(&item.Name),
				Progress(&item.CPU).Width(40),
			)
		}),
	)

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

	ui := VBox(
		Text(&stressData.Title),
		Text("═══════════════════════════════════════════════════════════════"),
		Text("CPU Cores"),
		ForEach(&cpuCores, func(item *StressItem) any {
			return HBox(
				Text(&item.Name),
				Progress(&item.CPU).Width(50),
			)
		}),
		Text(""),
		Text("Memory Banks"),
		ForEach(&memBanks, func(item *StressItem) any {
			return HBox(
				Text(&item.Name),
				Progress(&item.CPU).Width(50),
			)
		}),
		Text(""),
		Text("Process List"),
		ForEach(&procList, func(item *StressItem) any {
			return HBox(
				Text(&item.Name),
				Progress(&item.CPU).Width(40),
			)
		}),
	)

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

	ui := VBox(
		Text(&stressData.Title),
		ForEach(&stressData.Items, func(item *StressItem) any {
			return HBox(
				Text(&item.Name),
				Progress(&item.CPU).Width(30),
			)
		}),
	)

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

	ui := VBox(
		Text(&stressData.Title),
		ForEach(&stressData.Items, func(item *StressItem) any {
			return HBox(
				Text(&item.Name),
				Progress(&item.CPU).Width(30),
			)
		}),
	)

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

	ui := VBox.Height(50)(
		Text("Header"),
		VBox.Grow(1)(
			Text(&content),
		),
		VBox.Grow(2)(
			Text(&content),
		),
		VBox.Grow(1)(
			Text(&content),
		),
		Text("Footer"),
	)

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

	ui := HBox(
		VBox.Grow(1)(Text(&content)),
		VBox.Grow(2)(Text(&content)),
		VBox.Grow(1)(Text(&content)),
	)

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

	ui := HBox(
		VBox.Border(BorderSingle).Title("STATUS").Grow(1)(Text(&val)),
		VBox.Border(BorderSingle).Title("SYSTEMS").Grow(1)(Text(&val)),
		VBox.Border(BorderSingle).Title("CAPACITY").Grow(1)(Text(&val)),
	)

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

	ui := VBox(
		Leader("CPU USAGE", &cpuVal),
		Leader("MEMORY", &memVal),
		Leader("DISK FREE", &dskVal),
		Leader("CPU USAGE", &cpuVal),
		Leader("MEMORY", &memVal),
		Leader("DISK FREE", &dskVal),
	)

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

// === Dynamic property benchmarks ===

// BenchmarkStaticHeight - VBox with static height (baseline)
func BenchmarkStaticHeight(b *testing.B) {
	buf := NewBuffer(80, 50)
	content := "content"

	ui := VBox.Height(50)(
		Text("Header"),
		VBox.Height(10).Border(BorderSingle).Title("SECTION A").Grow(1)(
			Text(&content),
		),
		VBox.Height(10).Border(BorderSingle).Title("SECTION B").Grow(2)(
			Text(&content),
		),
		VBox.Height(10).Border(BorderSingle).Title("SECTION C").Grow(1)(
			Text(&content),
		),
		Text("Footer"),
	)

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

// BenchmarkDynamicHeight - same layout but heights are *int16 pointers
func BenchmarkDynamicHeight(b *testing.B) {
	buf := NewBuffer(80, 50)
	content := "content"

	rootH := int16(50)
	secA := int16(10)
	secB := int16(10)
	secC := int16(10)

	ui := VBox.Height(&rootH)(
		Text("Header"),
		VBox.Height(&secA).Border(BorderSingle).Title("SECTION A").Grow(1)(
			Text(&content),
		),
		VBox.Height(&secB).Border(BorderSingle).Title("SECTION B").Grow(2)(
			Text(&content),
		),
		VBox.Height(&secC).Border(BorderSingle).Title("SECTION C").Grow(1)(
			Text(&content),
		),
		Text("Footer"),
	)

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

// BenchmarkStaticDashboard - full dashboard with all static properties
func BenchmarkStaticDashboard(b *testing.B) {
	buf := NewBuffer(120, 60)
	content := "status: ok"

	items := make([]StressItem, 20)
	for i := range items {
		items[i] = StressItem{
			Name: "svc-" + string(rune('A'+i)),
			CPU:  float32(i) / 20.0,
		}
	}

	ui := VBox.Height(60)(
		Text("Dashboard"),
		HBox.Height(20)(
			VBox.Width(40).Border(BorderSingle).Title("LEFT")(
				Text(&content),
				ForEach(&items, func(item *StressItem) any {
					return HBox(
						Text(&item.Name),
						Progress(&item.CPU).Width(20),
					)
				}),
			),
			VBox.Grow(1).Border(BorderSingle).Title("RIGHT")(
				Text(&content),
			),
		),
		VBox.Grow(1).Border(BorderSingle).Title("BOTTOM")(
			Text(&content),
		),
	)

	serial := Build(ui)
	buf.Clear()
	serial.Execute(buf, 120, 60)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		serial.Execute(buf, 120, 60)
	}
}

// BenchmarkDynamicDashboard - same dashboard but heights/widths are dynamic
func BenchmarkDynamicDashboard(b *testing.B) {
	buf := NewBuffer(120, 60)
	content := "status: ok"

	items := make([]StressItem, 20)
	for i := range items {
		items[i] = StressItem{
			Name: "svc-" + string(rune('A'+i)),
			CPU:  float32(i) / 20.0,
		}
	}

	rootH := int16(60)
	midH := int16(20)
	leftW := int16(40)

	ui := VBox.Height(&rootH)(
		Text("Dashboard"),
		HBox.Height(&midH)(
			VBox.Width(&leftW).Border(BorderSingle).Title("LEFT")(
				Text(&content),
				ForEach(&items, func(item *StressItem) any {
					return HBox(
						Text(&item.Name),
						Progress(&item.CPU).Width(20),
					)
				}),
			),
			VBox.Grow(1).Border(BorderSingle).Title("RIGHT")(
				Text(&content),
			),
		),
		VBox.Grow(1).Border(BorderSingle).Title("BOTTOM")(
			Text(&content),
		),
	)

	serial := Build(ui)
	buf.Clear()
	serial.Execute(buf, 120, 60)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.ClearDirty()
		serial.Execute(buf, 120, 60)
	}
}

// === Conditional property benchmarks ===

// BenchmarkConditionalHeight - height driven by If condition
func BenchmarkConditionalHeight(b *testing.B) {
	buf := NewBuffer(80, 50)
	content := "content"
	expanded := true

	ui := VBox.Height(If(&expanded).Then(int16(50)).Else(int16(20)))(
		Text("Header"),
		VBox.Height(If(&expanded).Then(int16(10)).Else(int16(5))).Border(BorderSingle).Title("A").Grow(1)(
			Text(&content),
		),
		VBox.Height(If(&expanded).Then(int16(10)).Else(int16(5))).Border(BorderSingle).Title("B").Grow(2)(
			Text(&content),
		),
		VBox.Height(If(&expanded).Then(int16(10)).Else(int16(5))).Border(BorderSingle).Title("C").Grow(1)(
			Text(&content),
		),
		Text("Footer"),
	)

	serial := Build(ui)
	buf.Clear()
	serial.Execute(buf, 80, 50)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		expanded = i%2 == 0
		buf.ClearDirty()
		serial.Execute(buf, 80, 50)
	}
}

// BenchmarkConditionalDashboard - dashboard with multiple conditional properties
func BenchmarkConditionalDashboard(b *testing.B) {
	buf := NewBuffer(120, 60)
	content := "status: ok"
	detailed := true

	items := make([]StressItem, 20)
	for i := range items {
		items[i] = StressItem{
			Name: "svc-" + string(rune('A'+i)),
			CPU:  float32(i) / 20.0,
		}
	}

	ui := VBox.Height(60)(
		Text("Dashboard"),
		HBox.Height(If(&detailed).Then(int16(30)).Else(int16(10)))(
			VBox.Width(If(&detailed).Then(int16(50)).Else(int16(30))).Border(BorderSingle).Title("LEFT")(
				Text(&content),
				ForEach(&items, func(item *StressItem) any {
					return HBox(
						Text(&item.Name),
						Progress(&item.CPU).Width(If(&detailed).Then(int16(30)).Else(int16(15))),
					)
				}),
			),
			VBox.Grow(1).Border(BorderSingle).Title("RIGHT")(
				Text(&content),
			),
		),
		VBox.Grow(1).Border(BorderSingle).Title("BOTTOM")(
			Text(&content),
		),
	)

	serial := Build(ui)
	buf.Clear()
	serial.Execute(buf, 120, 60)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		detailed = i%2 == 0
		buf.ClearDirty()
		serial.Execute(buf, 120, 60)
	}
}

// BenchmarkAnimateIdle - animation evaluators present but not actively animating
func BenchmarkAnimateIdle(b *testing.B) {
	buf := NewBuffer(80, 50)
	content := "content"
	target := int16(20)

	ui := VBox.Height(50)(
		Text("Header"),
		VBox.Height(Animate.Duration(200 * time.Millisecond)(&target)).Border(BorderSingle).Title("A").Grow(1)(
			Text(&content),
		),
		VBox.Height(Animate.Duration(200 * time.Millisecond)(&target)).Border(BorderSingle).Title("B").Grow(2)(
			Text(&content),
		),
		VBox.Height(Animate.Duration(200 * time.Millisecond)(&target)).Border(BorderSingle).Title("C").Grow(1)(
			Text(&content),
		),
		Text("Footer"),
	)

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

// BenchmarkAnimateActive - animation evaluators actively interpolating each frame
func BenchmarkAnimateActive(b *testing.B) {
	buf := NewBuffer(80, 50)
	content := "content"
	target := int16(20)

	ui := VBox.Height(50)(
		Text("Header"),
		VBox.Height(Animate.Duration(200 * time.Millisecond)(&target)).Border(BorderSingle).Title("A").Grow(1)(
			Text(&content),
		),
		VBox.Height(Animate.Duration(200 * time.Millisecond)(&target)).Border(BorderSingle).Title("B").Grow(2)(
			Text(&content),
		),
		VBox.Height(Animate.Duration(200 * time.Millisecond)(&target)).Border(BorderSingle).Title("C").Grow(1)(
			Text(&content),
		),
		Text("Footer"),
	)

	serial := Build(ui)
	buf.Clear()
	serial.Execute(buf, 80, 50)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// flip target each iteration to keep animations active
		if i%2 == 0 {
			target = 30
		} else {
			target = 20
		}
		buf.ClearDirty()
		serial.Execute(buf, 80, 50)
	}
}

// BenchmarkAnimateComposed - Animate wrapping If conditions
func BenchmarkAnimateComposed(b *testing.B) {
	buf := NewBuffer(80, 50)
	content := "content"
	expanded := true

	smooth := Animate.Duration(200 * time.Millisecond).Ease(EaseOutCubic)

	ui := VBox.Height(50)(
		Text("Header"),
		VBox.Height(smooth(If(&expanded).Then(int16(30)).Else(int16(10)))).Border(BorderSingle).Title("A").Grow(1)(
			Text(&content),
		),
		VBox.Height(smooth(If(&expanded).Then(int16(30)).Else(int16(10)))).Border(BorderSingle).Title("B").Grow(2)(
			Text(&content),
		),
		VBox.Height(smooth(If(&expanded).Then(int16(30)).Else(int16(10)))).Border(BorderSingle).Title("C").Grow(1)(
			Text(&content),
		),
		Text("Footer"),
	)

	serial := Build(ui)
	buf.Clear()
	serial.Execute(buf, 80, 50)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		expanded = i%2 == 0
		buf.ClearDirty()
		serial.Execute(buf, 80, 50)
	}
}

// BenchmarkAnimateDashboard - full dashboard with multiple animated properties
func BenchmarkAnimateDashboard(b *testing.B) {
	buf := NewBuffer(120, 60)
	content := "status: ok"
	detailed := true

	items := make([]StressItem, 20)
	for i := range items {
		items[i] = StressItem{
			Name: "svc-" + string(rune('A'+i)),
			CPU:  float32(i) / 20.0,
		}
	}

	smooth := Animate.Duration(200 * time.Millisecond).Ease(EaseOutCubic)

	ui := VBox.Height(60)(
		Text("Dashboard"),
		HBox.Height(smooth(If(&detailed).Then(int16(30)).Else(int16(10))))(
			VBox.Width(smooth(If(&detailed).Then(int16(50)).Else(int16(30)))).Border(BorderSingle).Title("LEFT")(
				Text(&content),
				ForEach(&items, func(item *StressItem) any {
					return HBox(
						Text(&item.Name),
						Progress(&item.CPU).Width(smooth(If(&detailed).Then(int16(30)).Else(int16(15)))),
					)
				}),
			),
			VBox.Grow(1).Border(BorderSingle).Title("RIGHT")(
				Text(&content),
			),
		),
		VBox.Grow(1).Border(BorderSingle).Title("BOTTOM")(
			Text(&content),
		),
	)

	serial := Build(ui)
	buf.Clear()
	serial.Execute(buf, 120, 60)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		detailed = i%2 == 0
		buf.ClearDirty()
		serial.Execute(buf, 120, 60)
	}
}
