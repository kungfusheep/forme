package glyph_test

// compiler-verified examples matching the start.html code blocks.
// if these break, the getting started page is lying.

import (
	"fmt"
	"testing"

	. "github.com/kungfusheep/glyph"
)

type startTask struct {
	Title    string
	Done     bool
	Priority int
}

// start.html: section 02, first app (counter)
func TestStart_firstApp(t *testing.T) {
	app, _ := NewApp()
	count := 0

	app.SetView(
		VBox(
			Text(&count),
			Text("up/down to count, q to quit"),
		),
	)

	app.Handle("up", func() { count++ })
	app.Handle("down", func() { count-- })
	app.Handle("q", app.Stop)
	_ = app
}

// start.html: section 03, lists with render
func TestStart_listRender(t *testing.T) {
	app, _ := NewApp()

	tasks := []startTask{
		{"Write tutorial", false, 0},
		{"Add tests", false, 0},
		{"Ship it", false, 0},
	}

	app.SetView(
		VBox(
			Text("Tasks").Bold(),
			List(&tasks).Render(func(t *startTask) any {
				return HBox.Gap(1)(
					If(&t.Done).Then(Text("[x]")).Else(Text("[ ]")),
					Text(&t.Title),
				)
			}).BindVimNav(),
		),
	)

	app.Handle("q", app.Stop)
	_ = app
}

// start.html: section 03, filter list
func TestStart_filterList(t *testing.T) {
	tasks := []startTask{
		{"Write tutorial", false, 0},
	}

	_ = FilterList(&tasks, func(t *startTask) string { return t.Title }).
		Render(func(t *startTask) any {
			return Text(&t.Title)
		}).MaxVisible(10).Border(BorderRounded)
}

// start.html: section 04, layout split pane
func TestStart_layoutSplitPane(t *testing.T) {
	_ = HBox(
		VBox.Grow(1).Border(BorderRounded).Title("left")(
			Text("panel one"),
		),
		VBox.Grow(2).Border(BorderRounded).Title("right")(
			Text("panel two takes 2/3 width"),
		),
	)
}

// start.html: section 04, status bar with gaps
func TestStart_statusBar(t *testing.T) {
	_ = HBox.Gap(2)(
		Text("ready").FG(Green),
		Text("3 tasks").Bold(),
		Space(),
		Text("q: quit").FG(BrightBlack),
	)
}

// start.html: section 04, full split-pane app
func TestStart_fullSplitApp(t *testing.T) {
	app, _ := NewApp()

	tasks := []startTask{
		{"Write tutorial", false, 1},
		{"Add tests", true, 2},
		{"Ship it", false, 3},
	}
	detail := "select a task"

	app.SetView(
		VBox(
			Text("Task Manager").Bold(),
			HBox(
				VBox.Grow(1).Border(BorderRounded).Title("tasks")(
					List(&tasks).Render(func(t *startTask) any {
						return HBox.Gap(1)(
							If(&t.Done).Then(Text("[x]")).Else(Text("[ ]")),
							Text(&t.Title),
						)
					}).OnSelect(func(t *startTask) {
						detail = fmt.Sprintf("%s (priority: %d)", t.Title, t.Priority)
					}).BindVimNav(),
				),
				VBox.Grow(1).Border(BorderRounded).Title("detail")(
					Text(&detail),
				),
			),
			HBox.Gap(2)(
				Text("j/k: navigate").FG(BrightBlack),
				Text("q: quit").FG(BrightBlack),
			),
		),
	)

	app.Handle("q", app.Stop)
	_ = app
}

// start.html: section 05, keyboard list handle
func TestStart_keyboardListHandle(t *testing.T) {
	tasks := []startTask{
		{"Write tutorial", false, 0},
	}

	_ = List(&tasks).Render(func(t *startTask) any {
		return HBox.Gap(1)(
			If(&t.Done).Then(Text("[x]")).Else(Text("[ ]")),
			Text(&t.Title),
		)
	}).Handle("<Space>", func(t *startTask) {
		t.Done = !t.Done
	}).BindVimNav()
}

// start.html: section 05, autotable bind nav
func TestStart_autoTableBindNav(t *testing.T) {
	type proc struct {
		PID int
		CPU float64
	}
	var procs []proc

	_ = AutoTable(&procs).Scrollable(20).
		BindNav("<C-n>", "<C-p>").
		BindPageNav("<C-d>", "<C-u>")
}

// start.html: section 06, inline styling
func TestStart_inlineStyling(t *testing.T) {
	_ = Text("error").FG(Red).Bold()
	_ = Text("muted").FG(BrightBlack).Dim()
	_ = Text("success").FG(Green)
	_ = Text("warning").FG(Yellow).Bold()
}

// start.html: section 06, hex/rgb colours
func TestStart_hexRgb(t *testing.T) {
	_ = Text("branded").FG(Hex(0xc44040))
	_ = Text("custom").BG(RGB(20, 20, 40))
}

// start.html: section 06, cascade style theme
func TestStart_cascadeTheme(t *testing.T) {
	theme := ThemeDark

	_ = VBox.CascadeStyle(&theme.Base).Border(BorderRounded).BorderFG(theme.Border.FG)(
		Text("normal text"),
		Text("muted").Style(theme.Muted),
		Text("accent").Style(theme.Accent),
		Text("error!").Style(theme.Error),
	)
}

// start.html: section 06, custom theme
func TestStart_customTheme(t *testing.T) {
	myTheme := ThemeEx{
		Base:   Style{FG: White},
		Muted:  Style{FG: BrightBlack},
		Accent: Style{FG: Cyan, Attr: AttrBold},
		Error:  Style{FG: Red},
		Border: Style{FG: BrightBlack},
	}
	_ = myTheme
}

// start.html: section 06, border colours
func TestStart_borderColours(t *testing.T) {
	_ = VBox.Border(BorderRounded).BorderFG(Cyan).Title("status")(
		Text("all systems go").FG(Green),
	)
}
