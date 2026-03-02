package glyph_test

// compiler-verified examples matching the homepage code blocks.
// if these break, the homepage is lying.

import (
	"os/exec"
	"strings"
	"testing"

	. "github.com/kungfusheep/glyph"
)

type homepageCommit struct {
	Short   string
	Subject string
	Author  string
}

type homepageProc struct {
	PID     int
	Command string
	CPU     float64
	Mem     float64
}

type homepagePkg struct {
	Name string
	Desc string
}

// homepage: section 01, first app
func TestHomepage_firstApp(t *testing.T) {
	app, _ := NewApp()

	app.SetView(
		VBox(
			Text("Hello, glyph"),
			Text("Press q to quit"),
		),
	)

	app.Handle("q", app.Stop)
	_ = app
}

// homepage: section 02, compose real layouts
func TestHomepage_composeLayouts(t *testing.T) {
	var files []string
	preview := ""

	_ = HBox(
		VBox.Grow(1).Border(BorderRounded)(
			List(&files).BindVimNav().OnSelect(func(f *string) {
				preview = *f
			}),
		),
		VBox.Grow(2).Border(BorderRounded)(
			TextView(&preview).Grow(1),
		),
	)
}

// homepage: section 02, stream anything
func TestHomepage_streamAnything(t *testing.T) {
	app, _ := NewApp()

	cmd := exec.Command("go", "test", "./...")
	stdout, _ := cmd.StdoutPipe()
	_ = cmd.Start()

	app.SetView(
		VBox(
			Text("test output").Bold(),
			Log(stdout).Grow(1).MaxLines(1000),
		),
	)
	_ = app
}

// homepage: section 02, conditional rendering
func TestHomepage_conditionalRendering(t *testing.T) {
	connected := true
	uptime := "3h 42m"

	_ = If(&connected).Then(
		HBox.Gap(2)(
			Text("Status").Bold(),
			Text(&uptime).FG(Green),
		),
	).Else(
		Text("Disconnected").FG(Red),
	)
}

// homepage: section 02, render any slice
func TestHomepage_renderAnySlice(t *testing.T) {
	var commits []homepageCommit

	_ = List(&commits).Render(func(c *homepageCommit) any {
		return HBox.Gap(2)(
			Text(&c.Short).FG(Yellow),
			Text(&c.Subject),
			Text(&c.Author).FG(BrightBlack),
		)
	}).BindVimNav()
}

// homepage: section 02, process monitor
func TestHomepage_processMonitor(t *testing.T) {
	var procs []homepageProc
	cpuPct := 0.72
	memPct := 0.45

	_ = VBox(
		HBox.Gap(4)(
			Text("CPU"), Progress(&cpuPct).Width(30),
			Text("Mem"), Progress(&memPct).Width(30),
		),
		AutoTable(&procs).Sortable().Scrollable(20).BindVimNav(),
	)
}

// homepage: section 02, deploy log
func TestHomepage_deployLog(t *testing.T) {
	frame := 0
	status := "deploying..."
	pct := 0.0
	done := false
	result := "deployed"
	output := strings.NewReader("")

	_ = VBox.Border(BorderRounded).Title("deploy")(
		HBox.Gap(2)(
			Spinner(&frame).FG(Cyan),
			Text(&status).Bold(),
			Progress(&pct).Width(20),
		),
		Log(output).Grow(1).MaxLines(500),
		If(&done).Then(Text(&result).Bold().FG(Green)),
	)
}

// homepage: section 02, fuzzy finder
func TestHomepage_fuzzyFinder(t *testing.T) {
	var packages []homepagePkg

	_ = FilterList(&packages, func(p *homepagePkg) string { return p.Name }).
		Render(func(p *homepagePkg) any {
			return HBox.Gap(2)(
				Text(&p.Name).Bold(),
				Text(&p.Desc).FG(BrightBlack),
			)
		}).MaxVisible(15).Border(BorderRounded).Title("packages")
}

// homepage: section 02, live dashboard
func TestHomepage_liveDashboard(t *testing.T) {
	var reqData, latData []float64
	reqRate := "1,204 req/s"
	p99 := "12ms"
	events := strings.NewReader("")

	_ = VBox(
		HBox.Gap(1)(
			VBox.Grow(1).Border(BorderRounded).Title("requests/s")(
				Sparkline(&reqData).FG(Green),
				Text(&reqRate).FG(BrightBlack),
			),
			VBox.Grow(1).Border(BorderRounded).Title("p99 latency")(
				Sparkline(&latData).FG(Yellow),
				Text(&p99).FG(BrightBlack),
			),
		),
		Log(events).Grow(1).MaxLines(200),
	)
}

// homepage: section 02, forms with validation
func TestHomepage_formsWithValidation(t *testing.T) {
	var name, email string
	role := 0
	agree := false

	register := func() {}

	_ = Form.LabelBold().OnSubmit(register)(
		Field("Name", Input(&name).Validate(VRequired, VOnBlur)),
		Field("Email", Input(&email).Validate(VEmail, VOnBlur)),
		Field("Role", Radio(&role, "Admin", "User", "Guest")),
		Field("Terms", Checkbox(&agree, "I accept").Validate(VTrue, VOnSubmit)),
	)
}
