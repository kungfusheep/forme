package forme

import (
	"context"
	"sync/atomic"
)

// BufferPool manages double-buffered rendering.
// Swap alternates between two buffers, clearing the inactive one
// synchronously before making it current.
type BufferPool struct {
	buffers [2]*Buffer
	current atomic.Uint32  // 0 or 1 - which buffer is active
	dirty   [2]atomic.Bool // track if each buffer needs clearing
}

// NewBufferPool creates a double-buffered pool.
func NewBufferPool(width, height int) *BufferPool {
	return &BufferPool{
		buffers: [2]*Buffer{
			NewBuffer(width, height),
			NewBuffer(width, height),
		},
	}
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

// Stop is a no-op kept for API compatibility.
func (p *BufferPool) Stop() {}

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
