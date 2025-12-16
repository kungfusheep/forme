package tui

import (
	"io"
	"os"
	"sync"
	"time"
)

// Loop is the main event loop for a TUI application.
// It coordinates input, deferred callbacks, and rendering.
type Loop struct {
	screen *Screen
	reader io.Reader

	// Callbacks
	onInput  func([]byte) // Raw input handler
	onResize func(Size)   // Resize handler
	onRender func(*Buffer) // Render callback

	// CallLater queue
	callLaterMu    sync.Mutex
	callLaterQueue []func()

	// Control channels
	inputChan  chan []byte
	frameChan  chan struct{}
	stopChan   chan struct{}

	// State
	running bool
	mu      sync.Mutex
}

// NewLoop creates a new event loop with the given screen.
func NewLoop(screen *Screen) *Loop {
	return &Loop{
		screen:    screen,
		reader:    os.Stdin,
		inputChan: make(chan []byte, 16),
		frameChan: make(chan struct{}, 1),
		stopChan:  make(chan struct{}),
	}
}

// Screen returns the screen associated with this loop.
func (l *Loop) Screen() *Screen {
	return l.screen
}

// OnInput sets the input handler callback.
// The callback receives raw input bytes.
func (l *Loop) OnInput(fn func([]byte)) {
	l.onInput = fn
}

// OnResize sets the resize handler callback.
func (l *Loop) OnResize(fn func(Size)) {
	l.onResize = fn
}

// OnRender sets the render callback.
// This is called during the render phase with the back buffer.
func (l *Loop) OnRender(fn func(*Buffer)) {
	l.onRender = fn
}

// CallLater schedules a function to be called on the next frame.
// This is the integration point for animations and deferred work.
// Safe to call from any goroutine.
func (l *Loop) CallLater(fn func()) {
	l.callLaterMu.Lock()
	l.callLaterQueue = append(l.callLaterQueue, fn)
	l.callLaterMu.Unlock()

	l.requestFrame()
}

// requestFrame signals that a frame should be rendered.
// Non-blocking - if a frame is already requested, this does nothing.
func (l *Loop) requestFrame() {
	select {
	case l.frameChan <- struct{}{}:
	default:
	}
}

// RequestRender requests that the screen be re-rendered.
// This triggers a frame on the next loop iteration.
func (l *Loop) RequestRender() {
	l.requestFrame()
}

// Run starts the event loop. This blocks until Stop is called.
func (l *Loop) Run() error {
	l.mu.Lock()
	if l.running {
		l.mu.Unlock()
		return nil
	}
	l.running = true
	l.mu.Unlock()

	// Enter raw mode
	if err := l.screen.EnterRawMode(); err != nil {
		return err
	}
	defer l.screen.ExitRawMode()

	// Start input reader goroutine
	go l.readInput()

	// Initial render
	l.processFrame()

	// Main loop
	for {
		select {
		case input := <-l.inputChan:
			l.handleInput(input)
			l.processFrameNow()

		case <-l.frameChan:
			l.processFrameNow()

		case size := <-l.screen.ResizeChan():
			l.handleResize(size)
			l.processFrameNow()

		case <-l.stopChan:
			return nil
		}
	}
}

// Stop signals the event loop to stop.
func (l *Loop) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.running {
		return
	}
	l.running = false
	close(l.stopChan)
}

// readInput reads from stdin in a separate goroutine.
func (l *Loop) readInput() {
	buf := make([]byte, 256)
	for {
		n, err := l.reader.Read(buf)
		if err != nil {
			return
		}
		if n > 0 {
			// Make a copy of the input
			input := make([]byte, n)
			copy(input, buf[:n])

			select {
			case l.inputChan <- input:
			case <-l.stopChan:
				return
			}
		}
	}
}

// handleInput processes raw input bytes.
func (l *Loop) handleInput(input []byte) {
	if l.onInput != nil {
		l.onInput(input)
	}
}

// handleResize processes a resize event.
func (l *Loop) handleResize(size Size) {
	if l.onResize != nil {
		l.onResize(size)
	}
}

// processFrameNow processes a frame immediately.
// This drains any pending frame requests first.
func (l *Loop) processFrameNow() {
	// Drain any pending frame requests
	select {
	case <-l.frameChan:
	default:
	}
	l.processFrame()
}

// processFrame executes one frame: callLater queue, then render.
func (l *Loop) processFrame() {
	// Phase 1: Process callLater queue
	// Loop until queue is empty (callbacks may add more callbacks)
	for {
		l.callLaterMu.Lock()
		if len(l.callLaterQueue) == 0 {
			l.callLaterMu.Unlock()
			break
		}
		// Swap out the queue
		queue := l.callLaterQueue
		l.callLaterQueue = nil
		l.callLaterMu.Unlock()

		// Execute all callbacks
		for _, fn := range queue {
			fn()
		}
	}

	// Phase 2: Render
	if l.onRender != nil {
		l.screen.Clear()
		l.onRender(l.screen.Buffer())
	}

	// Phase 3: Flush to terminal
	l.screen.Flush()
}

// RunFor runs the loop for a specified duration, then stops.
// Useful for testing and demos.
func (l *Loop) RunFor(d time.Duration) error {
	go func() {
		time.Sleep(d)
		l.Stop()
	}()
	return l.Run()
}
