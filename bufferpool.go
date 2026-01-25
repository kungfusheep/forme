package forme

import (
	"context"
	"sync"
	"sync/atomic"
)

// BufferPool manages double-buffered rendering with async clearing.
// The clear happens in a background goroutine during app logic time,
// making it invisible to the render critical path.
type BufferPool struct {
	buffers [2]*Buffer
	current atomic.Uint32 // 0 or 1 - which buffer is active
	dirty   [2]atomic.Bool // track if each buffer needs clearing

	mu            sync.Mutex
	cond          *sync.Cond
	pendingClear  *Buffer
	pendingIdx    int // which buffer index is pending clear
	clearerActive bool
}

// NewBufferPool creates a double-buffered pool with async clearing.
func NewBufferPool(width, height int) *BufferPool {
	p := &BufferPool{
		buffers: [2]*Buffer{
			NewBuffer(width, height),
			NewBuffer(width, height),
		},
	}
	p.cond = sync.NewCond(&p.mu)
	p.startClearer()
	return p
}

// Current returns the current buffer for rendering.
func (p *BufferPool) Current() *Buffer {
	return p.buffers[p.current.Load()]
}

// Swap switches to the other buffer.
// Returns the new current buffer (cleared and ready to use).
func (p *BufferPool) Swap() *Buffer {
	old := p.current.Load()
	next := 1 - old

	// Mark old buffer as needing clear
	p.dirty[old].Store(true)

	// Only clear if needed (skip if already clean)
	if p.dirty[next].Load() {
		p.buffers[next].ClearDirty()
		p.dirty[next].Store(false)
	}

	p.current.Store(next)
	return p.buffers[next]
}

// startClearer launches the background clearing goroutine.
func (p *BufferPool) startClearer() {
	p.clearerActive = true
	go func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		for p.clearerActive {
			// Wait for work
			for p.pendingClear == nil && p.clearerActive {
				p.cond.Wait()
			}

			if !p.clearerActive {
				return
			}

			// Grab the buffer to clear and its index
			buf := p.pendingClear
			idx := p.pendingIdx
			p.pendingClear = nil
			p.mu.Unlock()

			// Clear outside the lock
			buf.ClearDirty()
			// Mark as clean
			p.dirty[idx].Store(false)

			p.mu.Lock()
		}
	}()
}

// Stop shuts down the clearer goroutine.
func (p *BufferPool) Stop() {
	p.mu.Lock()
	p.clearerActive = false
	p.cond.Signal()
	p.mu.Unlock()
}

// Width returns the buffer width.
func (p *BufferPool) Width() int {
	return p.buffers[0].Width()
}

// Height returns the buffer height.
func (p *BufferPool) Height() int {
	return p.buffers[0].Height()
}

// Resize resizes both buffers in the pool to new dimensions.
// Call this when the terminal is resized.
func (p *BufferPool) Resize(width, height int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := 0; i < 2; i++ {
		p.buffers[i].Resize(width, height)
		p.dirty[i].Store(false) // Mark as clean after resize (Resize clears)
	}
}

// Run executes a render loop until ctx is cancelled.
// Each frame the callback receives a pre-cleared buffer - do whatever you need with it.
func (p *BufferPool) Run(ctx context.Context, frame func(buf *Buffer)) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		buf := p.Current()
		frame(buf)
		p.Swap()
	}
}
