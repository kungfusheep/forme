package tui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"riffkey"
)

// Debug timing
var (
	DebugTiming      bool
	lastBuildTime    time.Duration
	lastLayoutTime   time.Duration
	lastRenderTime   time.Duration
	lastFlushTime    time.Duration
)

// App is a TUI application with integrated input handling via riffkey.
type App struct {
	screen *Screen

	// riffkey integration
	router *riffkey.Router
	input  *riffkey.Input
	reader *riffkey.Reader

	// SerialTemplate + BufferPool (for SetView single-view mode)
	template *SerialTemplate
	pool     *BufferPool

	// Multi-view routing
	viewTemplates map[string]*SerialTemplate
	viewRouters   map[string]*riffkey.Router
	currentView   string

	// State
	running    bool
	renderMu   sync.Mutex
	renderChan chan struct{}

	// Cursor state
	cursorX, cursorY int
	cursorVisible    bool
	cursorShape      CursorShape
}

// NewApp creates a new TUI application.
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
	}

	return app, nil
}

// SetView sets a declarative view for fast rendering.
// This uses SerialTemplate + BufferPool for maximum performance.
// Pointers in the view are captured at compile time - just mutate your state.
//
// Example:
//
//	state := &MyState{Title: "Hello", Progress: 50}
//	app.SetView(
//	    DCol{Children: []any{
//	        DText{Content: &state.Title},
//	        DProgress{Value: &state.Progress},
//	    }},
//	)
func (a *App) SetView(view any) *App {
	a.template = BuildSerial(view)
	// Create buffer pool for async clearing
	size := a.screen.Size()
	a.pool = NewBufferPool(size.Width, size.Height)
	return a
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
		a.viewTemplates = make(map[string]*SerialTemplate)
		a.viewRouters = make(map[string]*riffkey.Router)
	}

	// Create buffer pool if not exists (shared across all views)
	if a.pool == nil {
		size := a.screen.Size()
		a.pool = NewBufferPool(size.Width, size.Height)
	}

	// Compile template and create router for this view
	a.viewTemplates[name] = BuildSerial(view)
	router := riffkey.NewRouter()
	a.viewRouters[name] = router

	return &ViewBuilder{
		app:    a,
		name:   name,
		router: router,
	}
}

// Handle registers a key handler for this view.
func (vb *ViewBuilder) Handle(pattern string, handler func(riffkey.Match)) *ViewBuilder {
	vb.router.Handle(pattern, handler)
	return vb
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
// The modal's handlers take precedence until Pop() is called.
func (a *App) PushView(name string) {
	if router, ok := a.viewRouters[name]; ok {
		a.input.Push(router)
		a.RequestRender()
	}
}

// PopView removes the top modal overlay.
func (a *App) PopView() {
	a.input.Pop()
	a.RequestRender()
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
func (a *App) Handle(pattern string, handler func(riffkey.Match)) *App {
	a.router.Handle(pattern, handler)
	return a
}

// HandleNamed registers a named key binding (for rebinding support).
func (a *App) HandleNamed(name, pattern string, handler func(riffkey.Match)) *App {
	a.router.HandleNamed(name, pattern, handler)
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

// ShowCursor makes the cursor visible with the given shape.
func (a *App) ShowCursor(shape CursorShape) {
	a.cursorVisible = true
	a.cursorShape = shape
}

// HideCursor hides the cursor.
func (a *App) HideCursor() {
	a.cursorVisible = false
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

// render performs the actual render if needed.
func (a *App) render() {
	a.renderMu.Lock()
	defer a.renderMu.Unlock()

	var t0, t1 time.Time
	if DebugTiming {
		t0 = time.Now()
	}

	// Fast path: use SerialTemplate + BufferPool
	// Check for multi-view mode first, then single-view mode
	var tmpl *SerialTemplate
	if a.currentView != "" && a.viewTemplates != nil {
		tmpl = a.viewTemplates[a.currentView]
	} else if a.template != nil {
		tmpl = a.template
	}

	if tmpl == nil || a.pool == nil {
		return // No view set
	}

	size := a.screen.Size()
	buf := a.pool.Current()
	tmpl.Execute(buf, int16(size.Width), int16(size.Height))

	if DebugTiming {
		t1 = time.Now()
		lastBuildTime = 0
		lastLayoutTime = 0
		lastRenderTime = t1.Sub(t0)
	}

	// Copy to screen's back buffer for flush
	a.copyToScreen(buf)
	a.screen.Flush()
	a.pool.Swap() // Queue async clear

	// Position and show/hide cursor after render
	if a.cursorVisible {
		a.screen.SetCursorShape(a.cursorShape)
		a.screen.MoveCursor(a.cursorX, a.cursorY)
		a.screen.ShowCursor()
	} else {
		a.screen.HideCursor()
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
	return fmt.Sprintf("build:%v layout:%v render:%v flush:%v",
		lastBuildTime.Round(time.Microsecond),
		lastLayoutTime.Round(time.Microsecond),
		lastRenderTime.Round(time.Microsecond),
		lastFlushTime.Round(time.Microsecond))
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

	// Enter raw mode
	if err := a.screen.EnterRawMode(); err != nil {
		return err
	}
	defer a.screen.ExitRawMode()

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
	// Close stdin to unblock the reader
	os.Stdin.Close()
}

// handleResize watches for terminal resize events.
func (a *App) handleResize() {
	for range a.screen.ResizeChan() {
		a.RequestRender()
	}
}

// Size returns the current screen size.
func (a *App) Size() Size {
	return a.screen.Size()
}
