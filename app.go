package forme

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kungfusheep/riffkey"
)

// Debug timing
var (
	DebugTiming     bool
	DebugFullRedraw bool // force full redraws instead of diff-based (set TUI_FULL_REDRAW=1 to enable)
	DebugFlush      bool // dump flush debug info (set TUI_DEBUG_FLUSH=1 to enable)
	lastBuildTime   time.Duration
	lastLayoutTime  time.Duration
	lastRenderTime  time.Duration
	lastFlushTime   time.Duration

	// fine-grained post-processing phase timings (only populated when DebugTiming=true)
	lastEffectTime time.Duration // resolveColor16 + all PostProcess passes
	lastDiffTime   time.Duration // Flush(): diff + escape-sequence building
	lastWriteTime  time.Duration // FlushBuffer(): Write() syscall to terminal
)

func init() {
	if os.Getenv("TUI_FULL_REDRAW") != "" {
		DebugFullRedraw = true
	}
	if os.Getenv("TUI_DEBUG_FLUSH") != "" {
		DebugFlush = true
	}
}

// App is a TUI application with integrated input handling via riffkey.
type App struct {
	screen *Screen

	// riffkey integration
	router *riffkey.Router
	input  *riffkey.Input
	reader *riffkey.Reader

	// Template + BufferPool (for SetView single-view mode)
	template *Template
	pool     *BufferPool

	// Multi-view routing
	viewTemplates map[string]*Template
	viewRouters   map[string]*riffkey.Router
	currentView   string
	viewStack     []string // pushed views (for modal overlays)

	// State
	running    bool
	renderMu   sync.Mutex
	renderChan chan struct{}

	// Cursor state
	cursorX, cursorY int
	cursorVisible    bool
	cursorShape      CursorShape
	cursorColor      Color
	cursorColorSet   bool

	// Resize callback
	onResize func(width, height int)

	// Before-render callback (for syncing state before layout)
	onBeforeRender func()

	// After-render callback (for cursor updates after layout is known)
	onAfterRender func()

	// Active layer for cursor (set during template render)
	activeLayer *Layer

	// Inline mode
	inline         bool
	clearOnExit    bool
	linesUsed      int
	viewHeight     int16 // Height of the view for inline mode
	nonInteractive bool  // True when running via RunNonInteractive

	// Jump labels
	jumpMode  *JumpMode
	jumpStyle JumpStyle

	// Post-processing pipeline
	postProcess   []PostProcess
	frameCount    uint64
	startTime     time.Time
	lastFrameTime time.Time
	defaultFG     Color // terminal's default FG (detected via OSC 10)
	defaultBG     Color // terminal's default BG (detected via OSC 11)

	// SetView limit (for catching anti-patterns)
	setViewCount int
	setViewLimit int // 0 = unlimited
}

// NewApp creates a new TUI application (fullscreen, alternate buffer).
func NewApp() (*App, error) {
	screen, err := NewScreen(nil)
	if err != nil {
		return nil, err
	}

	router := riffkey.NewRouter()
	input := riffkey.NewInput(router)
	reader := riffkey.NewReader(os.Stdin)

	app := &App{
		screen:     screen,
		router:     router,
		input:      input,
		reader:     reader,
		renderChan: make(chan struct{}, 1),
		jumpMode:   &JumpMode{},
		jumpStyle:  DefaultJumpStyle,
	}

	return app, nil
}

// NewInlineApp creates a new inline TUI application.
// Inline apps render at the current cursor position without taking over the screen.
// Use this for progress bars, selection menus, spinners, etc.
func NewInlineApp() (*App, error) {
	app, err := NewApp()
	if err != nil {
		return nil, err
	}
	app.inline = true
	return app, nil
}

func (a *App) Ref(f func(*App)) *App { f(a); return a }

// ClearOnExit sets whether the inline app should clear its content on exit.
// If true, the rendered content disappears when the app stops.
// If false (default), the content remains visible and cursor moves below it.
func (a *App) ClearOnExit(clear bool) *App {
	a.clearOnExit = clear
	return a
}

// IsInline returns true if this is an inline app.
func (a *App) IsInline() bool {
	return a.inline
}

// Height sets the height for inline apps.
// This determines how many lines the inline view will use.
// If not set, defaults to 1.
func (a *App) Height(h int16) *App {
	a.viewHeight = h
	return a
}

// RunNonInteractive runs an inline app without an input loop.
// Use this for progress bars, spinners, etc. that don't need keyboard input.
// Call Stop() when done to clean up and exit.
func (a *App) RunNonInteractive() error {
	if !a.inline {
		return fmt.Errorf("RunNonInteractive only works with inline apps")
	}

	a.running = true
	a.nonInteractive = true

	// Clean up buffer pool on exit
	if a.pool != nil {
		defer a.pool.Stop()
	}

	// Enter inline mode (raw mode without alternate buffer)
	if err := a.screen.EnterInlineMode(); err != nil {
		return err
	}

	// Initial render
	a.render()

	// Wait for Stop() to be called
	for a.running {
		select {
		case <-a.renderChan:
			a.render()
		case <-time.After(50 * time.Millisecond):
			// Check running flag periodically
		}
	}

	// Clean up
	a.screen.ExitInlineMode(a.linesUsed, a.clearOnExit)
	return nil
}

// SetViewLimit sets the maximum number of times SetView can be called.
// Panics if exceeded. Use this to catch anti-patterns where SetView is called
// repeatedly instead of using reactive updates via pointers.
//
// Example:
//
//	app.SetViewLimit(1) // Panic if SetView called more than once
//	app.SetView(myView) // OK
//	app.SetView(other)  // PANIC: SetView called 2 times, limit is 1
//
// Set to 0 (default) for unlimited calls.
func (a *App) SetViewLimit(n int) *App {
	a.setViewLimit = n
	return a
}

// SetView sets a declarative view for fast rendering.
// Pointers in the view are captured at compile time - just mutate your state.
//
// Example:
//
//	state := &MyState{Title: "Hello", Progress: 50}
//	app.SetView(
//	    Col{Children: []any{
//	        Text{Content: &state.Title},
//	        Progress{Value: &state.Progress},
//	    }},
//	)
func (a *App) SetView(view any) *App {
	a.setViewCount++
	if a.setViewLimit > 0 && a.setViewCount > a.setViewLimit {
		panic(fmt.Sprintf("SetView called %d times, limit is %d. Use reactive updates via pointers instead of calling SetView repeatedly.", a.setViewCount, a.setViewLimit))
	}

	a.template = Build(view)
	a.template.SetApp(a) // Link for jump mode support
	a.wireBindings(a.template, a.router)
	// Create buffer pool for async clearing (or reuse existing)
	size := a.screen.Size()
	if a.pool == nil {
		a.pool = NewBufferPool(size.Width, size.Height)
	} else if a.pool.Width() != size.Width || a.pool.Height() != size.Height {
		a.pool.Resize(size.Width, size.Height)
	}
	return a
}

// wireBindings registers all declarative component bindings on the given router.
func (a *App) wireBindings(tmpl *Template, router *riffkey.Router) {
	for _, b := range tmpl.pendingBindings {
		switch h := b.handler.(type) {
		case func(riffkey.Match):
			pattern := b.pattern
			router.Handle(pattern, func(m riffkey.Match) { h(m); a.RequestRender() })
		case func(any):
			pattern := b.pattern
			router.Handle(pattern, func(_ riffkey.Match) { h(nil); a.RequestRender() })
		case func():
			pattern := b.pattern
			router.Handle(pattern, func(_ riffkey.Match) { h(); a.RequestRender() })
		}
	}
	// focus manager takes precedence over single pendingTIB
	if tmpl.pendingFocusManager != nil {
		// wire focus manager bindings (Tab/Shift-Tab)
		for _, b := range tmpl.pendingFocusManager.bindings() {
			if h, ok := b.handler.(func(riffkey.Match)); ok {
				pattern := b.pattern
				router.Handle(pattern, func(m riffkey.Match) { h(m); a.RequestRender() })
			}
		}
		// route unmatched keys to focused component
		router.HandleUnmatched(tmpl.pendingFocusManager.HandleKey)
		router.NoCounts()
	} else if tmpl.pendingTIB != nil {
		th := riffkey.NewTextHandler(tmpl.pendingTIB.value, tmpl.pendingTIB.cursor)
		th.OnChange = tmpl.pendingTIB.onChange
		router.HandleUnmatched(th.HandleKey)
		router.NoCounts()
	}
	// wire Log invalidation
	for _, lv := range tmpl.pendingLogs {
		lv.onUpdate = a.RequestRender
	}
}

// ViewBuilder allows chaining Handle() calls after View().
type ViewBuilder struct {
	app    *App
	name   string
	router *riffkey.Router
}

// View registers a named view for multi-view routing.
// Returns a builder for chaining Handle() calls.
//
// Example:
//
//	app.View("home", homeView).
//	    Handle("j", moveDown).
//	    Handle("s", func(_ riffkey.Match) { app.Go("settings") })
func (a *App) View(name string, view any) *ViewBuilder {
	// Initialize maps if needed
	if a.viewTemplates == nil {
		a.viewTemplates = make(map[string]*Template)
	}
	if a.viewRouters == nil {
		a.viewRouters = make(map[string]*riffkey.Router)
	}

	// Create buffer pool if not exists (shared across all views)
	if a.pool == nil {
		size := a.screen.Size()
		a.pool = NewBufferPool(size.Width, size.Height)
	}

	// Compile template and create router for this view
	tmpl := Build(view)
	tmpl.SetApp(a) // Link for jump mode support
	router := riffkey.NewRouter()
	a.wireBindings(tmpl, router)
	a.viewTemplates[name] = tmpl
	a.viewRouters[name] = router

	return &ViewBuilder{
		app:    a,
		name:   name,
		router: router,
	}
}

func (vb *ViewBuilder) Ref(f func(*ViewBuilder)) *ViewBuilder { f(vb); return vb }

// NoCounts disables vim-style count prefixes (e.g., 5j) for this view.
// Use this when the view has text input so digits can be typed.
func (vb *ViewBuilder) NoCounts() *ViewBuilder {
	vb.router.NoCounts()
	return vb
}

// Handle registers a key handler for this view.
// Accepts func(riffkey.Match), func(any), or func() for convenience.
// Automatically requests a re-render after the handler runs.
func (vb *ViewBuilder) Handle(pattern string, handler any) *ViewBuilder {
	switch h := handler.(type) {
	case func(riffkey.Match):
		vb.router.Handle(pattern, func(m riffkey.Match) { h(m); vb.app.RequestRender() })
	case func(any):
		vb.router.Handle(pattern, func(_ riffkey.Match) { h(nil); vb.app.RequestRender() })
	case func():
		vb.router.Handle(pattern, func(_ riffkey.Match) { h(); vb.app.RequestRender() })
	}
	return vb
}

// UpdateView recompiles a view with a new view definition.
// Use this when the view's structure changes and needs re-compilation.
func (a *App) UpdateView(name string, view any) {
	if a.viewTemplates == nil {
		return
	}
	tmpl := Build(view)
	tmpl.SetApp(a) // Link for jump mode support
	if router, ok := a.viewRouters[name]; ok {
		a.wireBindings(tmpl, router)
	}
	a.viewTemplates[name] = tmpl
}

// Go switches to a different view.
// Swaps the template and input handlers.
func (a *App) Go(name string) {
	if _, ok := a.viewTemplates[name]; !ok {
		return // View doesn't exist
	}
	a.currentView = name
	a.input.SetRouter(a.viewRouters[name])
	a.RequestRender()
}

// Back returns to the previous view.
// Currently an alias for Pop() - may add history later.
func (a *App) Back() {
	a.input.Pop()
	a.RequestRender()
}

// PushView pushes a view as a modal overlay.
// The modal's handlers take precedence until PopView() is called.
// The pushed view becomes the active rendered view until popped.
func (a *App) PushView(name string) {
	if router, ok := a.viewRouters[name]; ok {
		a.viewStack = append(a.viewStack, name)
		a.input.Push(router)
		a.RequestRender()
	}
}

// PopView removes the top modal overlay.
// Returns to the previous view in the stack.
func (a *App) PopView() {
	if len(a.viewStack) > 0 {
		a.viewStack = a.viewStack[:len(a.viewStack)-1]
	}
	a.input.Pop()
	a.RequestRender()
}

// ViewRouter returns the router for a named view, if it exists.
// Useful for advanced configuration like HandleUnmatched.
func (a *App) ViewRouter(name string) (*riffkey.Router, bool) {
	if a.viewRouters == nil {
		return nil, false
	}
	router, ok := a.viewRouters[name]
	return router, ok
}

// Screen returns the screen.
func (a *App) Screen() *Screen {
	return a.screen
}

// Router returns the riffkey router for advanced configuration.
func (a *App) Router() *riffkey.Router {
	return a.router
}

// Input returns the riffkey input for modal handling (push/pop).
func (a *App) Input() *riffkey.Input {
	return a.input
}

// Handle registers a key binding with a vim-style pattern.
// Examples: "j", "gg", "<C-c>", "<C-w>j", "<Up>"
// Accepts func(riffkey.Match), func(any), or func() for convenience.
// Automatically requests a re-render after the handler runs.
func (a *App) Handle(pattern string, handler any) *App {
	switch h := handler.(type) {
	case func(riffkey.Match):
		a.router.Handle(pattern, func(m riffkey.Match) { h(m); a.RequestRender() })
	case func(any):
		a.router.Handle(pattern, func(_ riffkey.Match) { h(nil); a.RequestRender() })
	case func():
		a.router.Handle(pattern, func(_ riffkey.Match) { h(); a.RequestRender() })
	}
	return a
}

// HandleNamed registers a named key binding (for rebinding support).
// Automatically requests a re-render after the handler runs.
func (a *App) HandleNamed(name, pattern string, handler func(riffkey.Match)) *App {
	a.router.HandleNamed(name, pattern, func(m riffkey.Match) { handler(m); a.RequestRender() })
	return a
}

// BindField routes unmatched keys to a text input field.
func (a *App) BindField(f *InputState) *App {
	a.router.TextInput(&f.Value, &f.Cursor)
	return a
}

// UnbindField clears the text input field binding.
func (a *App) UnbindField() *App {
	a.router.HandleUnmatched(nil)
	return a
}

// Push pushes a new router onto the input stack (for modal input).
func (a *App) Push(r *riffkey.Router) {
	a.input.Push(r)
}

// Pop pops the current router from the input stack.
func (a *App) Pop() {
	a.input.Pop()
}

// SetCursor sets the cursor position (0-indexed screen coordinates).
// The cursor will be positioned here after each render.
func (a *App) SetCursor(x, y int) {
	a.cursorX = x
	a.cursorY = y
}

// SetCursorStyle sets the cursor visual style.
func (a *App) SetCursorStyle(style CursorShape) {
	a.cursorShape = style
}

// ShowCursor makes the cursor visible.
func (a *App) ShowCursor() {
	a.cursorVisible = true
}

// HideCursor hides the cursor.
func (a *App) HideCursor() {
	a.cursorVisible = false
}

// SetCursorColor sets the cursor color using OSC 12 escape sequence.
// This changes the actual cursor color in supporting terminals.
func (a *App) SetCursorColor(c Color) {
	a.cursorColor = c
	a.cursorColorSet = true
}

// Cursor returns the current cursor state.
func (a *App) Cursor() Cursor {
	return Cursor{
		X:       a.cursorX,
		Y:       a.cursorY,
		Style:   a.cursorShape,
		Visible: a.cursorVisible,
	}
}

// OnResize sets a callback to be called when the terminal is resized.
// The callback receives the new width and height.
// Use this to update viewport dimensions, reinitialize layers, etc.
func (a *App) OnResize(fn func(width, height int)) {
	a.onResize = fn
}

// OnBeforeRender sets a callback to be called before each render.
// Use this to sync state (e.g., filter updates) before layout runs.
func (a *App) OnBeforeRender(fn func()) {
	a.onBeforeRender = fn
}

// OnAfterRender sets a callback to be called after each render completes.
// Use this to update cursor position after layout is known.
func (a *App) OnAfterRender(fn func()) {
	a.onAfterRender = fn
}

// AddPostProcess appends a post-processing pass to the pipeline.
// Passes run in order after template rendering, before screen flush.
// Closures captured by the pass act as shader uniforms — mutate them to
// change behaviour next frame.
func (a *App) AddPostProcess(pp PostProcess) *App {
	a.postProcess = append(a.postProcess, pp)
	return a
}

// SetPostProcess replaces the entire post-processing pipeline.
// Call with no arguments to clear all passes.
func (a *App) SetPostProcess(passes ...PostProcess) *App {
	a.postProcess = passes
	return a
}

// Template returns the current template for debugging.
// Use with Template().DebugDump("") to inspect the op tree.
func (a *App) Template() *Template {
	return a.template
}

// RequestRender marks that a render is needed.
// Safe to call from any goroutine.
func (a *App) RequestRender() {
	select {
	case a.renderChan <- struct{}{}:
	default:
		// Already a render pending
	}
}

// RenderNow performs a render immediately without channel coordination.
// Use this from dedicated update goroutines to avoid scheduler overhead.
// The render is mutex-protected so it's safe to call concurrently.
func (a *App) RenderNow() {
	a.render()
}

// render performs the actual render if needed.
func (a *App) render() {
	a.renderMu.Lock()
	defer a.renderMu.Unlock()

	var t0, t1 time.Time
	if DebugTiming {
		t0 = time.Now()
	}

	if a.pool == nil {
		return // No pool
	}

	// sync state before layout (e.g., filter updates)
	if a.onBeforeRender != nil {
		a.onBeforeRender()
	}

	// clear active layer before render (will be set if a layer has visible cursor)
	a.activeLayer = nil

	size := a.screen.Size()
	buf := a.pool.Current()

	// For inline mode, use view height instead of terminal height
	renderHeight := int16(size.Height)
	if a.inline && a.viewHeight > 0 {
		renderHeight = a.viewHeight
	} else if a.inline {
		// auto-size: give layout full terminal height, then trim to content
		renderHeight = int16(size.Height)
	}

	// Priority: pushed views > current view > base template
	var activeTmpl *Template
	if len(a.viewStack) > 0 {
		topView := a.viewStack[len(a.viewStack)-1]
		if a.viewTemplates != nil {
			if tmpl, ok := a.viewTemplates[topView]; ok {
				activeTmpl = tmpl
			}
		}
	}
	if activeTmpl == nil {
		if a.currentView != "" && a.viewTemplates != nil {
			if tmpl, ok := a.viewTemplates[a.currentView]; ok {
				activeTmpl = tmpl
			} else {
				return // View not found
			}
		} else if a.template != nil {
			activeTmpl = a.template
		} else {
			return // No view set
		}
	}
	activeTmpl.Execute(buf, int16(size.Width), renderHeight)

	// for inline auto-size, use content height instead of full terminal height
	if a.inline && a.viewHeight == 0 {
		if h := buf.ContentHeight(); h > 0 {
			renderHeight = int16(h)
		}
	}

	// apply layer cursor if one was set during template render
	if a.activeLayer != nil {
		if x, y, visible := a.activeLayer.ScreenCursor(); visible {
			a.cursorX = x
			a.cursorY = y
			a.cursorVisible = true
			a.cursorShape = a.activeLayer.cursor.Style
		}
	}

	// call after-render callback (e.g., for additional cursor customization)
	if a.onAfterRender != nil {
		a.onAfterRender()
	}

	if DebugTiming {
		t1 = time.Now()
		lastBuildTime = 0
		lastLayoutTime = 0
		lastRenderTime = t1.Sub(t0)
	}

	// post-processing pipeline: tree-declared ScreenEffects first, then imperative
	treeEffects := activeTmpl.ScreenEffects()
	a.screen.forceRGB = len(treeEffects) > 0 || len(a.postProcess) > 0
	if a.screen.forceRGB {
		var tEffect time.Time
		if DebugTiming {
			tEffect = time.Now()
		}

		// resolve Color16 cells to detected palette RGB before effects run
		resolveColor16(buf, size.Width, int(renderHeight))

		now := time.Now()
		if a.startTime.IsZero() {
			a.startTime = now
		}
		var delta time.Duration
		if !a.lastFrameTime.IsZero() {
			delta = now.Sub(a.lastFrameTime)
		}
		a.lastFrameTime = now

		ppCtx := PostContext{
			Width:     size.Width,
			Height:    int(renderHeight),
			Frame:     a.frameCount,
			Delta:     delta,
			Time:      now.Sub(a.startTime),
			DefaultFG: a.defaultFG,
			DefaultBG: a.defaultBG,
		}
		for _, pp := range treeEffects {
			pp(buf, ppCtx)
		}
		for _, pp := range a.postProcess {
			pp(buf, ppCtx)
		}
		a.frameCount++

		if DebugTiming {
			lastEffectTime = time.Since(tEffect)
		}
	}

	// Copy to screen's back buffer for flush
	a.copyToScreen(buf)

	if a.inline {
		// Inline mode: render at cursor position
		a.linesUsed = a.screen.FlushInline(int(renderHeight))
		a.pool.Swap() // Queue async clear
	} else {
		// Fullscreen mode
		var tDiff time.Time
		if DebugTiming {
			tDiff = time.Now()
		}
		if DebugFullRedraw {
			a.screen.FlushFull()
		} else {
			a.screen.Flush() // diff + escape-sequence building
		}
		if DebugTiming {
			lastDiffTime = time.Since(tDiff)
		}
		a.pool.Swap()

		// Add cursor ops to same buffer - one syscall for everything
		if a.cursorColorSet {
			a.screen.BufferCursorColor(a.cursorColor)
		}
		a.screen.BufferCursor(a.cursorX, a.cursorY, a.cursorVisible, a.cursorShape)

		var tWrite time.Time
		if DebugTiming {
			tWrite = time.Now()
		}
		a.screen.FlushBuffer() // single Write() syscall to terminal
		if DebugTiming {
			lastWriteTime = time.Since(tWrite)
		}
	}

	if DebugTiming {
		lastFlushTime = time.Since(t1)
	}
}

// copyToScreen copies pool buffer to screen's back buffer.
func (a *App) copyToScreen(src *Buffer) {
	dst := a.screen.Buffer()
	dst.CopyFrom(src) // Fast bulk copy
}

// TimingString returns a formatted timing string.
func TimingString() string {
	return fmt.Sprintf("render:%v effect:%v diff:%v write:%v",
		lastRenderTime.Round(time.Microsecond),
		lastEffectTime.Round(time.Microsecond),
		lastDiffTime.Round(time.Microsecond),
		lastWriteTime.Round(time.Microsecond))
}

// Timings holds timing data for the last frame.
type Timings struct {
	BuildUs  float64 // Build time in microseconds
	LayoutUs float64 // Layout time in microseconds
	RenderUs float64 // Render time in microseconds
	FlushUs  float64 // Flush time in microseconds

	// fine-grained post-processing breakdown (only valid when effects are active)
	EffectUs float64 // resolveColor16 + all PostProcess passes (pure Go)
	DiffUs   float64 // Flush(): diff comparison + escape-sequence building (pure Go)
	WriteUs  float64 // FlushBuffer(): Write() syscall — time spent waiting on terminal
}

// GetTimings returns the timing data for the last frame.
func GetTimings() Timings {
	return Timings{
		BuildUs:  float64(lastBuildTime.Microseconds()),
		LayoutUs: float64(lastLayoutTime.Microseconds()),
		RenderUs: float64(lastRenderTime.Microseconds()),
		FlushUs:  float64(lastFlushTime.Microseconds()),
		EffectUs: float64(lastEffectTime.Microseconds()),
		DiffUs:   float64(lastDiffTime.Microseconds()),
		WriteUs:  float64(lastWriteTime.Microseconds()),
	}
}

// Run starts the application. Blocks until Stop is called.
// For multi-view apps, use RunFrom(startView) instead.
func (a *App) Run() error {
	return a.run("")
}

// RunFrom starts the application on the specified view.
// Use this for multi-view apps.
func (a *App) RunFrom(startView string) error {
	return a.run(startView)
}

func (a *App) run(startView string) error {
	a.running = true

	// Set up starting view if specified
	if startView != "" && a.viewTemplates != nil {
		a.currentView = startView
		if router, ok := a.viewRouters[startView]; ok {
			a.input.SetRouter(router)
		}
	}

	// Clean up buffer pool on exit if using fast path
	if a.pool != nil {
		defer a.pool.Stop()
	}

	// Enter raw mode (inline or fullscreen)
	if a.inline {
		if err := a.screen.EnterInlineMode(); err != nil {
			return err
		}
		// Use closure so linesUsed is read at defer time, not now (when it's 0)
		defer func() { a.screen.ExitInlineMode(a.linesUsed, a.clearOnExit) }()
	} else {
		if err := a.screen.EnterRawMode(); err != nil {
			return err
		}
		defer a.screen.ExitRawMode()
	}

	// detect terminal's default colours for post-processing
	a.defaultFG, a.defaultBG = a.screen.QueryDefaultColors()

	// Handle resize
	go a.handleResize()

	// Handle async render requests (from timers, data updates, etc)
	go a.handleRenderRequests()

	// Initial render
	a.render()

	// Run riffkey input loop
	// afterDispatch is called after every key - perfect for rendering
	err := a.input.Run(a.reader, func(handled bool) {
		if !a.running {
			return
		}
		// Always render after input (state may have changed)
		a.render()
	})

	// Normal termination via Stop() causes reader to return error
	if !a.running {
		// Reopen stdin for inline apps so subsequent apps can use it
		if a.inline {
			reopenStdin()
		}
		return nil
	}
	return err
}

// handleRenderRequests processes async render requests.
func (a *App) handleRenderRequests() {
	for {
		select {
		case <-a.renderChan:
			if !a.running {
				return
			}
			a.render()
		}
	}
}

// Stop signals the application to stop.
func (a *App) Stop() {
	a.running = false
	// Close stdin to unblock the input reader (not needed for non-interactive)
	if !a.nonInteractive {
		os.Stdin.Close()
	}
}

// reopenStdin reopens stdin from /dev/tty after it was closed.
// This allows running multiple inline apps in sequence.
func reopenStdin() {
	f, err := os.Open("/dev/tty")
	if err == nil {
		os.Stdin = f
	}
}

// handleResize watches for terminal resize events.
func (a *App) handleResize() {
	for size := range a.screen.ResizeChan() {
		// Resize the buffer pool to match new terminal dimensions
		if a.pool != nil {
			a.pool.Resize(size.Width, size.Height)
		}
		// Notify application of resize
		if a.onResize != nil {
			a.onResize(size.Width, size.Height)
		}
		a.RequestRender()
	}
}

// Size returns the current screen size.
func (a *App) Size() Size {
	return a.screen.Size()
}

// =============================================================================
// Jump Labels
// =============================================================================

// JumpKey registers a key pattern to trigger jump mode.
// This is a convenience method that calls EnterJumpMode when the key is pressed.
func (a *App) JumpKey(pattern string) *App {
	a.router.Handle(pattern, func(_ riffkey.Match) {
		a.EnterJumpMode()
	})
	return a
}

// SetJumpStyle sets the global style for jump labels.
func (a *App) SetJumpStyle(style JumpStyle) *App {
	a.jumpStyle = style
	return a
}

// JumpStyle returns the current jump style.
func (a *App) JumpStyle() JumpStyle {
	return a.jumpStyle
}

// JumpModeActive returns true if jump mode is currently active.
func (a *App) JumpModeActive() bool {
	return a.jumpMode.Active
}

// JumpMode returns the jump mode state for use during rendering.
func (a *App) JumpMode() *JumpMode {
	return a.jumpMode
}

// EnterJumpMode activates jump label mode.
// A render is triggered to collect jump targets, then a temporary router
// is pushed to handle label input.
func (a *App) EnterJumpMode() {
	if a.jumpMode.Active {
		return // Already in jump mode
	}

	a.jumpMode.Active = true
	a.jumpMode.ClearJumpTargets()

	// Render to collect targets (they register during render)
	a.render()

	// Assign labels after collecting targets
	a.jumpMode.AssignLabels()

	if len(a.jumpMode.Targets) == 0 {
		// No targets, exit immediately
		a.jumpMode.Active = false
		return
	}

	// Create temporary router for jump input
	jumpRouter := riffkey.NewRouter().NoCounts()

	// Build label lookup
	for _, target := range a.jumpMode.Targets {
		target := target // capture for closure
		jumpRouter.Handle(target.Label, func(_ riffkey.Match) {
			if target.OnSelect != nil {
				target.OnSelect()
			}
			a.ExitJumpMode()
		})
	}

	// Escape cancels
	jumpRouter.Handle("<Esc>", func(_ riffkey.Match) {
		a.ExitJumpMode()
	})

	// Any unmatched key cancels (unless it's a partial match for multi-char labels)
	jumpRouter.HandleUnmatched(func(k riffkey.Key) bool {
		// For multi-char labels, accumulate input
		if k.Rune != 0 && k.Mod == riffkey.ModNone {
			a.jumpMode.Input += string(k.Rune)
			// Check if any label starts with this prefix
			if a.jumpMode.HasPartialMatch(a.jumpMode.Input) {
				return true // Keep waiting for more input
			}
		}
		// No match, cancel
		a.ExitJumpMode()
		return true
	})

	a.input.Push(jumpRouter)

	// Re-render to show labels
	a.RequestRender()
}

// ExitJumpMode deactivates jump label mode.
func (a *App) ExitJumpMode() {
	if !a.jumpMode.Active {
		return
	}

	a.jumpMode.Active = false
	a.jumpMode.ClearJumpTargets()
	a.input.Pop()
	a.RequestRender()
}

// AddJumpTarget registers a jump target during rendering.
// Called by Jump components when jump mode is active.
func (a *App) AddJumpTarget(x, y int16, onSelect func(), style Style) {
	if a.jumpMode.Active {
		a.jumpMode.AddTarget(x, y, onSelect, style)
	}
}
