package glyph_test

import (
	. "github.com/kungfusheep/glyph"
	"os"
	"strings"
	"time"
)

// Vertical stack.
// Call VBox directly with children to stack them top to bottom.
func ExampleVBoxFn() {
	VBox(
		Text("First"),
		Text("Second"),
		Text("Third"),
	)
}

// Template syntax.
// Chain methods to configure, then call as a function with children. Configure once, render many.
func ExampleVBoxFn_chained() {
	VBox.Gap(1).Border(BorderRounded)(
		Text("First"),
		Text("Second"),
	)
}

// Flex growth.
// Grow fills available space. CascadeStyle passes a style to all descendants.
func ExampleVBoxFn_grow() {
	VBox.Grow(1).CascadeStyle(&Style{FG: White})(
		Text("header"),
		Text("content"),
		Text("footer"),
	)
}

// Horizontal layout.
// Arrange children side by side with a gap between them.
func ExampleHBoxFn() {
	HBox.Gap(2)(
		Text("left"),
		Text("right"),
	)
}

// Sidebar pattern.
// Combine WidthPct and Grow for a fixed sidebar with a flexible main area.
func ExampleHBoxFn_widths() {
	HBox(
		VBox.WidthPct(0.3)(Text("sidebar")),
		VBox.Grow(1)(Text("main content")),
	)
}

// Basic layering.
// Layer children on top of each other. The last child renders on top.
func ExampleOverlayFn() {
	Overlay(
		Text("base content"),
		Text("floating dialog"),
	)
}

// Modal dialog.
// Centered dialog with a dimmed backdrop. Size constrains the dialog; Backdrop dims everything behind it.
func ExampleOverlayFn_modal() {
	Overlay.Centered().Backdrop().BackdropFG(BrightBlack).Size(50, 15).BG(Black)(
		VBox.Grow(1)(
			Text("main content"),
		),
		VBox.Border(BorderRounded)(
			Text("confirm delete?"),
			HBox.Gap(2)(Text("[y]es"), Text("[n]o")),
		),
	)
}

// Static text.
// Text that never changes. Pass a string literal directly.
func ExampleText() {
	Text("hello world")
}

// Pointer binding.
// Pass a pointer so the rendered text reflects the current value. Mutate the string, then trigger a render to see it.
func ExampleText_pointer() {
	msg := "dynamic"
	Text(&msg)
}

// Inline styling.
// Chain style methods. Bold, Dim, Italic, Underline, and FG/BG are all available.
func ExampleText_styled() {
	msg := "styled"
	Text(&msg).Bold().FG(Cyan)
}

// Text input.
// A managed text input with placeholder text shown when empty.
func ExampleInput() {
	Input().Placeholder("Type here...")
}

// Password input.
// Mask hides input characters for sensitive fields.
func ExampleInput_mask() {
	Input().Placeholder("Password").Mask('*')
}

// Navigable list.
// A list over a pointer to a slice. The Render callback receives a pointer to each item for in-place updates.
func ExampleList() {
	items := []string{"Alpha", "Beta", "Gamma"}

	List(&items).Render(func(item *string) any {
		return Text(item)
	}).BindNav("j", "k")
}

// Toggleable list.
// Like List, but each item has a toggleable checkbox. The Done field is toggled in place through the pointer.
func ExampleCheckList() {
	type Task struct {
		Name string
		Done bool
	}
	tasks := []Task{{Name: "Ship it"}, {Name: "Test it"}}

	CheckList(&tasks).Render(func(t *Task) any {
		return Text(&t.Name)
	}).BindNav("j", "k").BindToggle(" ")
}

// Tab bar.
// Bound to an int pointer. The selected index is written when the user picks a tab.
func ExampleTabs() {
	var active int
	tabs := []string{"General", "Advanced", "About"}

	Tabs(tabs, &active)
}

// Custom tab styles.
// Customize appearance with Kind for the shape and per-state styles for active/inactive tabs.
func ExampleTabs_styled() {
	var active int
	tabs := []string{"Code", "Preview", "Settings"}

	Tabs(tabs, &active).
		Kind(TabsStyleBox).
		ActiveStyle(Style{FG: Cyan, Attr: AttrBold}).
		InactiveStyle(Style{FG: BrightBlack})
}

// Searchable list.
// The extract function tells the filter which string to match against for each item.
func ExampleFilterList() {
	items := []string{"Alpha", "Beta", "Gamma"}

	FilterList(&items, func(s *string) string { return *s }).
		Render(func(item *string) any {
			return Text(item)
		})
}

// Struct table.
// Generate a table from a struct slice. Column options control formatting.
func ExampleAutoTable() {
	type Row struct {
		Name string
		CPU  float64
	}
	rows := []Row{{Name: "api", CPU: 42.5}}

	AutoTable(&rows).Column("CPU", Number(1))
}

// Column formatters.
// Mix formatters: percentages, byte sizes, booleans, and colour-coded changes.
func ExampleAutoTable_columns() {
	type Service struct {
		Name   string
		CPU    float64
		Memory uint64
		Active bool
		Growth float64
	}
	rows := []Service{
		{Name: "api", CPU: 82.3, Memory: 1073741824, Active: true, Growth: 0.12},
	}

	AutoTable(&rows).
		Column("CPU", Percent(1)).
		Column("Memory", Bytes()).
		Column("Active", Bool("Yes", "No")).
		Column("Growth", PercentChange(1))
}

// Currency formatting.
// Format a float as a monetary value with the given symbol and decimal places.
func ExampleCurrency() {
	type Invoice struct {
		Item  string
		Price float64
	}
	rows := []Invoice{{Item: "Widget", Price: 42.50}}

	AutoTable(&rows).Column("Price", Currency("$", 2))
}

// Progress bar.
// Bound to a float64 pointer (0.0–1.0). Width sets the bar length in cells.
func ExampleProgress() {
	var pct int

	Progress(&pct).Width(30).FG(Green)
}

// Animated spinner.
// The frame pointer must be incremented by the caller (e.g. via a ticker goroutine).
func ExampleSpinner() {
	var frame int

	Spinner(&frame).Frames(SpinnerBraille)
}

// Radio group.
// Bound to an int pointer. The selected index updates when the user navigates and presses enter.
func ExampleRadio() {
	var selected int

	Radio(&selected, "Small", "Medium", "Large").BindNav("j", "k")
}

// Single checkbox.
// Bound to a bool pointer.
func ExampleCheckbox() {
	var agreed bool

	Checkbox(&agreed, "I agree to the terms")
}

// Leader line.
// Label on the left, value on the right, dots filling the gap. Both sides read from pointers each frame.
func ExampleLeader() {
	label := "Total"
	value := "$42.00"

	Leader(&label, &value)
}

// Mini chart.
// A line chart from a float64 slice pointer. Append values and trigger a render to update.
func ExampleSparkline() {
	values := []float64{1, 3, 5, 2, 8, 4}

	Sparkline(&values).Width(20).FG(Green)
}

// Flexible spacer.
// Pushes siblings apart. In a VBox, it pushes content to the top and bottom.
func ExampleSpace() {
	VBox(
		Text("header"),
		Space(),
		Text("footer"),
	)
}

// Fixed-height gap.
// Use in VBox to add exact spacing between elements.
func ExampleSpaceH() {
	VBox(
		Text("above"),
		SpaceH(2),
		Text("below"),
	)
}

// Fixed-width gap.
// Use in HBox to add exact spacing between elements.
func ExampleSpaceW() {
	HBox(
		Text("left"),
		SpaceW(4),
		Text("right"),
	)
}

// Horizontal divider.
// A line that fills the available width.
func ExampleHRule() {
	VBox(
		Text("above"),
		HRule(),
		Text("below"),
	)
}

// Vertical divider.
// A line that fills the available height.
func ExampleVRule() {
	HBox(
		Text("left"),
		VRule(),
		Text("right"),
	)
}

// Scrollbar.
// Parameters are total content height, visible viewport height, and a pointer to scroll position.
func ExampleScroll() {
	var pos int
	Scroll(100, 20, &pos)
}

// Conditional show.
// Show or hide content based on a bool pointer. The value is checked every frame.
func ExampleIf() {
	show := true
	If(&show).Then(Text("visible"))
}

// Toggle views.
// Switch between two views based on state.
func ExampleIf_else() {
	loggedIn := false
	If(&loggedIn).Then(Text("dashboard")).Else(Text("login"))
}

// Value matching.
// Match against a specific value. Works with strings, ints, or any comparable type.
func ExampleCondition_eq() {
	status := "active"
	If(&status).Eq("active").Then(
		Text("online").FG(Green),
	).Else(
		Text("offline").FG(Red),
	)
}

// Multi-way branch.
// Branch on a value with named cases. Default catches unmatched values.
func ExampleSwitch() {
	mode := "edit"
	Switch(&mode).
		Case("edit", Text("editing")).
		Case("preview", Text("previewing")).
		Default(Text("idle"))
}

// Numeric comparison.
// IfOrd supports Gt, Lt, Gte, Lte for any ordered type (int, float64, string).
func ExampleOrdCondition() {
	count := 5
	IfOrd(&count).Gte(10).Then(Text("many")).Else(Text("few"))
}

// Slice iteration.
// Iterate a slice with per-item templates. The callback receives a pointer to each item.
func ExampleForEach() {
	type Todo struct {
		Title string
		Done  bool
	}
	items := []Todo{{Title: "Ship it"}, {Title: "Test it"}}

	ForEach(&items, func(item *Todo) any {
		return HBox.Gap(2)(
			Checkbox(&item.Done, ""),
			Text(&item.Title),
		)
	})
}

// Inline spans.
// Mix Bold, Dim, Italic, FG and plain strings in a single line.
func ExampleRich() {
	Rich(
		Bold("Important: "),
		Dim("supporting detail"),
		FG("error", Red),
	)
}

// Custom span styles.
// Styled creates a span with a full Style struct for fine-grained control.
func ExampleSpan() {
	Rich(
		Styled("custom", Style{FG: Hex(0xFF5500), Attr: AttrBold}),
		" mixed with ",
		Italic("emphasis"),
	)
}

// Full-screen app.
// SetView compiles the tree once, Handle binds keys, Run blocks until Stop is called.
func ExampleNewApp() {
	app, _ := NewApp()

	counter := 0
	app.SetView(
		VBox(
			Text(&counter),
			Text("press q to quit"),
		),
	)
	app.Handle("q", func() { app.Stop() })
	app.Run()
}

// Inline app.
// Renders within the terminal flow instead of taking over the screen. Height sets the visible rows.
func ExampleNewInlineApp() {
	app, _ := NewInlineApp()

	status := "loading..."
	app.SetView(Text(&status))
	app.Height(3).ClearOnExit(true)
	app.Run()
}

// Multi-view routing.
// Named views with per-view key bindings. Go switches between views; each view is compiled independently.
func ExampleApp_multiView() {
	app, _ := NewApp()

	app.View("home", VBox(
		Text("Welcome"),
		Text("press n for next"),
	)).Handle("n", func() { app.Go("detail") })

	app.View("detail", VBox(
		Text("Detail view"),
		Text("press b for back"),
	)).Handle("b", func() { app.Go("home") })

	app.RunFrom("home")
}

// Goroutine updates.
// Mutate the value behind the pointer from any goroutine, then call RequestRender to trigger a redraw.
func ExampleApp_goroutine() {
	app, _ := NewApp()

	status := "waiting..."
	app.SetView(Text(&status))
	app.Handle("q", func() { app.Stop() })

	time.AfterFunc(2*time.Second, func() {
		status = "done!"
		app.RequestRender()
	})

	app.Run()
}

// Views with text input.
// NoCounts disables vim-style count prefixes, preventing number keys from being swallowed before reaching an input.
func ExampleViewBuilder() {
	app, _ := NewApp()

	var name string
	app.View("editor",
		VBox(
			Text("Editor"),
			Input().Placeholder("type here..."),
		),
	).NoCounts().Handle("<C-s>", func() {
		name = name + " saved"
	}).Handle("escape", func() {
		app.Go("home")
	})
}

// Style chaining.
// Build a style by chaining methods on DefaultStyle. CascadeStyle on a container applies it to all descendants.
func ExampleDefaultStyle() {
	s := DefaultStyle().Bold().Foreground(Cyan)

	VBox.CascadeStyle(&s)(
		Text("all children inherit bold cyan"),
	)
}

// Struct literal.
// Construct a Style directly when you know the exact fields. Attr flags combine with bitwise OR.
func ExampleStyle() {
	highlight := Style{FG: Yellow, Attr: AttrBold | AttrUnderline}

	Text("warning").Style(highlight)
}

// Style margin.
// Margin is part of Style. Values are top, right, bottom, left (CSS order).
func ExampleStyle_margin() {
	s := DefaultStyle().MarginTRBL(1, 2, 1, 2)

	Text("margined text").Style(s)
}

// Hex colour.
// Takes a uint32, not a string. Use Go hex literals.
func ExampleHex() {
	Text("branded").FG(Hex(0xFF5500))
}

// RGB colour.
// Precise 24-bit colour from red, green, blue components.
func ExampleRGB() {
	Text("vivid").FG(RGB(255, 85, 0))
}

// Colour blending.
// LerpColor blends two colours. t=0 returns the first, t=1 returns the second, 0.5 is the midpoint.
func ExampleLerpColor() {
	pct := 0.75
	bar := LerpColor(Red, Green, pct)
	Text("75%").FG(bar)
}

// Terminal palette.
// BasicColor uses the terminal's 16-colour palette (0–15). These respect the user's terminal theme.
func ExampleBasicColor() {
	Text("theme-aware").FG(BasicColor(9))
}

// Extended palette.
// PaletteColor uses the 256-colour extended palette.
func ExamplePaletteColor() {
	Text("orange-ish").FG(PaletteColor(214))
}

// Custom rendering.
// Escape hatch for custom rendering. Measure reports size, Render draws directly into the cell buffer.
func ExampleWidget() {
	Widget(
		func(availW int16) (w, h int16) { return availW, 1 },
		func(buf *Buffer, x, y, w, h int16) {
			for i := int16(0); i < w; i++ {
				buf.Set(int(x+i), int(y), Cell{Rune: '='})
			}
		},
	)
}

// Streaming log.
// Stream lines from any io.Reader into a scrollable log view. AutoScroll keeps the view pinned to the bottom.
func ExampleLogC() {
	r := strings.NewReader("line 1\nline 2\nline 3\n")

	Log(r).MaxLines(1000).AutoScroll(true).BindVimNav()
}

// Stdin pipe.
// Pipe stdout from a subprocess or os.Stdin directly into a log view. The reader is consumed in a background goroutine.
func ExampleLogC_fromStdin() {
	Log(os.Stdin).MaxLines(5000).Grow(1)
}

// Filterable log.
// A log view with a built-in fuzzy search input. Lines are filtered as you type; matching is fzf-style.
func ExampleFilterLogC() {
	r := strings.NewReader("info: started\nerror: failed\ninfo: recovered\n")

	FilterLog(r).Placeholder("filter logs...").MaxLines(5000).BindVimNav()
}

// Scrollable canvas.
// LayerView displays a Layer, a virtual canvas that can be larger than the viewport.
func ExampleLayerViewC() {
	layer := NewLayer()

	LayerView(layer).Grow(1)
}

// Manual content.
// Write lines directly to a Layer for manual content control. The Render callback fires when the viewport resizes.
func ExampleLayer() {
	layer := NewLayer()
	layer.Render = func() {
		layer.SetLineString(0, "redrawn", DefaultStyle())
	}
	layer.SetLineString(0, "first line", DefaultStyle())
	layer.SetLineString(1, "second line", DefaultStyle().Bold())

	LayerView(layer).Grow(1)
}

// Jump target.
// Wrap any component as a jump target. When jump mode is active, labelled hints appear over each target.
func ExampleJumpC() {
	selected := "none"
	Jump(Text("clickable item"), func() {
		selected = "clicked"
		_ = selected
	})
}

// App-level jump mode.
// Press the jump key to enter jump mode, then type a target label to select it.
func ExampleJumpC_app() {
	app, _ := NewApp()

	items := []string{"Save", "Load", "Quit"}
	app.SetView(VBox(
		Jump(Text(&items[0]), func() {}),
		Jump(Text(&items[1]), func() {}),
		Jump(Text(&items[2]), func() { app.Stop() }),
	))

	app.JumpKey("f")
	app.Run()
}

// Compile a template.
// Build compiles a UI tree into a Template. This is the same step that SetView performs internally. Useful for rendering into a Layer or for headless testing.
func ExampleBuild() {
	tmpl := Build(VBox(
		Text("hello"),
		Text("world"),
	))

	buf := NewBuffer(80, 24)
	tmpl.Execute(buf, 80, 24)
}

// Predefined theme.
// CascadeStyle sets the base style for all descendants; individual elements can override with per-element styles.
func ExampleThemeEx() {
	theme := ThemeDark

	VBox.CascadeStyle(&theme.Base).Border(BorderRounded).BorderFG(theme.Border.FG)(
		Text("normal text"),
		Text("muted").Style(theme.Muted),
		Text("accent").Style(theme.Accent),
		Text("error!").Style(theme.Error),
	)
}

// Built-in borders.
// Choose from BorderSingle, BorderRounded, or BorderDouble.
func ExampleBorderStyle() {
	VBox.Border(BorderDouble).BorderFG(Cyan)(
		Text("double-bordered"),
	)
}

// Custom borders.
// Define custom borders with any rune for each edge and corner.
func ExampleBorderStyle_custom() {
	ascii := BorderStyle{
		Horizontal:  '-',
		Vertical:    '|',
		TopLeft:     '+',
		TopRight:    '+',
		BottomLeft:  '+',
		BottomRight: '+',
	}

	VBox.Border(ascii)(Text("ascii box"))
}

// Text alignment.
// Align is set via the Style struct. AlignLeft, AlignCenter, or AlignRight within a fixed width.
func ExampleAlign() {
	VBox(
		Text("left-aligned"),
		Text("centered").Style(Style{Align: AlignCenter}).Width(40),
		Text("right-aligned").Style(Style{Align: AlignRight}).Width(40),
	)
}

// Tab shapes.
// Three tab styles: TabsStyleUnderline (default), TabsStyleBox, and TabsStyleBracket.
func ExampleTabsStyle() {
	var active int
	tabs := []string{"Files", "Search", "Git"}

	Tabs(tabs, &active).Kind(TabsStyleBracket)
}

// Custom layout.
// Arrange lets you define a fully custom layout function. It receives child sizes and available space, returns rects.
func ExampleArrange() {
	grid := Arrange(func(children []ChildSize, w, h int) []Rect {
		cols := 3
		cellW := w / cols
		rects := make([]Rect, len(children))
		for i := range children {
			rects[i] = Rect{
				X: (i % cols) * cellW,
				Y: (i / cols) * 2,
				W: cellW,
				H: 2,
			}
		}
		return rects
	})

	grid(Text("a"), Text("b"), Text("c"))
}

// Scoped helpers.
// Define scopes local helper functions inside the view tree. The function runs at build time and returns a component.
func ExampleDefine() {
	a, b, c := true, false, true

	Define(func() any {
		dot := func(v *bool) any {
			return If(v).Then(Text("●").FG(Green)).Else(Text("○").FG(Red))
		}
		return HBox.Gap(1)(dot(&a), dot(&b), dot(&c))
	})
}

// Focus cycling.
// FocusManager coordinates focus between multiple inputs. Tab/Shift-Tab cycles through ManagedBy components.
func ExampleFocusManager() {
	fm := NewFocusManager()

	VBox(
		Input().Placeholder("Name").ManagedBy(fm),
		Input().Placeholder("Email").ManagedBy(fm),
	)
}

// Collapsible tree.
// TreeView renders a collapsible tree from TreeNode structs. Set Expanded to control which nodes start open.
func ExampleTreeView() {
	root := &TreeNode{
		Label: "src",
		Children: []*TreeNode{
			{Label: "main.go"},
			{Label: "lib", Children: []*TreeNode{
				{Label: "utils.go"},
				{Label: "types.go"},
			}},
		},
		Expanded: true,
	}

	_ = TreeView{Root: root, ShowRoot: true, Indent: 2, ShowLines: true}
}

// Case transform.
// Transform text case via the Style struct. Transforms run at render time without modifying the source string.
func ExampleTextTransform() {
	Text("hello world").Style(Style{Transform: TransformUppercase})
}

// Registration form.
// Form arranges labeled fields with aligned labels and automatic focus management. Mix inputs, checkboxes and validators.
func ExampleFormFn() {
	var name, email, pass string
	var terms bool

	Form.LabelBold().Gap(1)(
		Field("Name", Input(&name).Placeholder("Jane Doe")),
		Field("Email", Input(&email).Placeholder("jane@example.com")),
		Field("Password", Input(&pass).Placeholder("min 8 chars").Mask('*')),
		Field("Terms", Checkbox(&terms, "I agree")),
	)
}

// Submit handler.
// Forward-declare the form variable so OnSubmit can reference the same instance.
func ExampleFormFn_onSubmit() {
	var name, email, pass string
	var terms bool
	var form *FormC

	form = Form.LabelBold().OnSubmit(func() {
		_ = form.ValidateAll()
	})(
		Field("Name", Input(&name).Validate(VRequired)),
		Field("Email", Input(&email).Validate(VEmail)),
		Field("Password", Input(&pass).Mask('*').Validate(VMinLen(8))),
		Field("Terms", Checkbox(&terms, "I agree").Validate(VTrue)),
	)
	_ = form
}

// Mixed control types.
// Inputs, checkboxes and radio groups all work as form fields with automatic focus cycling.
func ExampleFormFn_mixedControls() {
	var name string
	var notify bool
	var role int

	Form.LabelBold().Gap(1)(
		Field("Name", Input(&name).Placeholder("you")),
		Field("Notify", Checkbox(&notify, "Send me updates")),
		Field("Role", Radio(&role, "Viewer", "Editor", "Admin")),
	)
}

// ---------------------------------------------------------------------------
// Screen effects
// ---------------------------------------------------------------------------

// Single post-processing effect.
// Place ScreenEffect anywhere in the view tree — it applies to the whole screen.
func ExampleScreenEffect() {
	ScreenEffect(SEVignette())
}

// Stacked effects.
// Effects apply in order, left to right — each sees the output of the previous.
func ExampleScreenEffect_multiple() {
	ScreenEffect(
		SEDesaturate().Strength(0.5),
		SETint(Hex(0xFF6600)).Strength(0.1),
	)
}

// Conditional effect.
// Wrap in If() to toggle reactively based on state.
func ExampleScreenEffect_conditional() {
	dimmed := false
	If(&dimmed).Then(ScreenEffect(SEDimAll()))
}

// Per-cell transform.
// The fragment shader equivalent — define per-cell logic, iteration is handled.
func ExampleEachCell() {
	EachCell(func(x, y int, c Cell, ctx PostContext) Cell {
		if y%2 == 0 {
			c.Style.Attr = c.Style.Attr.With(AttrDim)
		}
		return c
	})
}

// Blend mode wrapper.
// Snapshots the buffer, runs the effect, then blends the result back using a Photoshop-style mode.
func ExampleWithBlend() {
	ScreenEffect(WithBlend(BlendScreen, SEFire()))
}

// Plasma through overlay blend.
// Combine blend modes with quantization for efficient animated effects.
func ExampleWithBlend_quantized() {
	ScreenEffect(WithQuantize(32, WithBlend(BlendOverlay, SEPlasma())))
}

// Quantize wrapper.
// Snap colours to step-size buckets after an effect runs. Use step=32 to cut animation bytes ~40%.
func ExampleWithQuantize() {
	ScreenEffect(WithQuantize(32, SEPlasma()))
}

// Manual colour blend.
// BlendColor combines two colours with a Photoshop-style mode — useful inside custom effects.
func ExampleBlendColor() {
	base := RGB(100, 150, 200)
	top := RGB(255, 100, 50)
	_ = BlendColor(base, top, BlendScreen)
}

// Dim entire screen uniformly.
func ExampleSEDimAll() {
	ScreenEffect(SEDimAll())
}

// Warm colour grade.
func ExampleSETint() {
	ScreenEffect(SETint(Hex(0xFF6600)).Strength(0.15))
}

// Tint that spares a focused panel.
func ExampleSETint_dodge() {
	var panel NodeRef
	ScreenEffect(SETint(Hex(0x0066FF)).Strength(0.3).Dodge(&panel))
}

// Cinematic edge darkening.
func ExampleSEVignette() {
	ScreenEffect(SEVignette().Strength(0.7))
}

// Vignette centred on a panel.
func ExampleSEVignette_focus() {
	var panel NodeRef
	ScreenEffect(SEVignette().Focus(&panel))
}

// Vignette that skips a panel.
func ExampleSEVignette_dodge() {
	var panel NodeRef
	ScreenEffect(SEVignette().Dodge(&panel))
}

// Smooth vignette without quantization.
func ExampleSEVignette_smooth() {
	ScreenEffect(SEVignette().Smooth().Strength(0.6))
}

// Wash out colour.
func ExampleSEDesaturate() {
	ScreenEffect(SEDesaturate().Strength(0.8))
}

// Colour spotlight — grey world, one panel in colour.
func ExampleSEDesaturate_dodge() {
	var panel NodeRef
	ScreenEffect(SEDesaturate().Dodge(&panel))
}

// Punch up contrast.
func ExampleSEContrast() {
	ScreenEffect(SEContrast().Strength(2.0))
}

// Contrast that spares a panel.
func ExampleSEContrast_dodge() {
	var panel NodeRef
	ScreenEffect(SEContrast().Dodge(&panel))
}

// Dim everything outside a node.
func ExampleSEFocusDim() {
	var panel NodeRef
	ScreenEffect(SEFocusDim(&panel))
}

// Frosted glass behind a modal.
func ExampleSEFrost() {
	var modal NodeRef
	ScreenEffect(SEFrost().Dodge(&modal))
}

// Frost only a specific region.
func ExampleSEFrost_focus() {
	var sidebar NodeRef
	ScreenEffect(SEFrost().Focus(&sidebar))
}

// Remap colours through a three-stop gradient.
func ExampleSEGradientMap() {
	ScreenEffect(SEGradientMap(
		RGB(0, 0, 50),      // shadows → deep blue
		RGB(0, 128, 128),   // midtones → teal
		RGB(200, 255, 200), // highlights → mint
	))
}

// Directional drop shadow behind a panel.
func ExampleSEDropShadow() {
	var panel NodeRef
	ScreenEffect(SEDropShadow().Focus(&panel))
}

// Symmetric glow — offset(0,0) centres the shadow source on the panel.
func ExampleSEDropShadow_glow() {
	var panel NodeRef
	ScreenEffect(SEDropShadow().Focus(&panel).Offset(0, 0).Strength(0.4).Radius(12))
}

// Colour-sampling glow that reads the panel's edge colours and spills them outward.
func ExampleSEGlow() {
	var panel NodeRef
	ScreenEffect(SEGlow().Focus(&panel))
}

// Bloom around bright cells.
func ExampleSEBloom() {
	ScreenEffect(SEBloom().Threshold(0.6).Strength(0.3))
}

// Bloom constrained to a panel.
func ExampleSEBloom_focus() {
	var panel NodeRef
	ScreenEffect(SEBloom().Focus(&panel))
}

// CRT scanlines + vignette + phosphor warmth.
func ExampleSECRT() {
	ScreenEffect(SECRT())
}

// Green phosphor monochrome.
func ExampleSEMonochrome() {
	ScreenEffect(SEMonochrome(RGB(0, 255, 80)))
}

// Monochrome with a colour spotlight.
func ExampleSEMonochrome_dodge() {
	var panel NodeRef
	ScreenEffect(SEMonochrome(RGB(0, 255, 80)).Dodge(&panel))
}

// Fade top and bottom rows toward black.
func ExampleSEFadeRows() {
	ScreenEffect(SEFadeRows(5, 5, Color{Mode: ColorRGB}))
}

// Snap colours to 32-level steps.
// Reduces escape output for animated effects with negligible visible banding.
func ExampleSEQuantize() {
	ScreenEffect(SEQuantize(32))
}
