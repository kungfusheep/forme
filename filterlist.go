package glyph

import (
	"sync/atomic"
	"time"
)

// FilterListC is a drop-in filterable list. it composes an input, a
// filter and a list into a single template node. the caller owns the
// source slice; mutations happen through a [StreamWriter] obtained
// from [FilterListC.Stream].
//
// usage:
//
//	FilterList(&items, func(p *Profile) string { return p.Name }).
//	    Placeholder("filter...").
//	    Render(func(p *Profile) any { return Text(p.Name) }).
//	    MaxVisible(20).
//	    Handle("<Enter>", func(p *Profile) { ... })
//
// streaming:
//
//	w := fl.Stream(app.RequestRender)
//	go func() {
//	    defer w.Close()
//	    for p := range produce() {
//	        w.Write(p)
//	    }
//	}()
type FilterListC[T any] struct {
	input  *InputC
	list   *ListC[T]
	filter *Filter[T]

	placeholder string
	maxVisible  int
	border      BorderStyle
	title       string
	margin      [4]int16

	counterMatch int // filtered count, read by counter at render time
	counterTotal int // total count, read by counter at render time
	isStreaming  bool
	spinnerFrame int32 // accessed atomically by spinner goroutine and render path
}

// FilterList creates a filterable list backed by the caller's source slice.
func FilterList[T any](source *[]T, extract func(*T) string) *FilterListC[T] {
	f := NewFilter(source, extract)
	fl := &FilterListC[T]{
		input:  Input(),
		list:   List(&f.Items),
		filter: f,
	}
	// wire input changes to filter + clamp
	fl.input.declaredTIB = &textInputBinding{
		value:  &fl.input.field.Value,
		cursor: &fl.input.field.Cursor,
		onChange: func(string) {
			fl.sync()
		},
	}
	// default nav keys that don't conflict with text input
	fl.list.BindNav("<C-n>", "<C-p>").
		BindPageNav("<C-d>", "<C-u>")
	fl.updateCounter()
	return fl
}

// toTemplate returns the template tree for compilation.
func (fl *FilterListC[T]) toTemplate() any {
	fl.input.placeholder = fl.placeholder
	if fl.maxVisible > 0 {
		fl.list.maxVisible = fl.maxVisible
	}

	inputRow := HBox(
		Text("> ").Bold(),
		fl.input,
	)

	counter := newCounter(&fl.counterMatch, &fl.counterTotal).
		Prefix("  ").Dim().
		Streaming(&fl.isStreaming)
	counter.framePtr = &fl.spinnerFrame

	children := []any{inputRow, counter, fl.list}

	box := VBox
	if fl.border.Horizontal != 0 {
		box = box.Border(fl.border).Title(fl.title)
	}
	if fl.margin != [4]int16{} {
		box = box.MarginTRBL(fl.margin[0], fl.margin[1], fl.margin[2], fl.margin[3])
	}
	return box(children...)
}

// bindings returns declared bindings from the list (nav, handles, etc).
func (fl *FilterListC[T]) bindings() []binding {
	return fl.list.bindings()
}

// textBinding returns the text input binding for the input.
func (fl *FilterListC[T]) textBinding() *textInputBinding {
	return fl.input.textBinding()
}

func (fl *FilterListC[T]) sync() {
	fl.filter.Update(fl.input.Value())
	fl.list.ClampSelection()
	fl.updateCounter()
}

func (fl *FilterListC[T]) appended() {
	fl.filter.appended()
	fl.list.ClampSelection()
	fl.updateCounter()
}

func (fl *FilterListC[T]) refresh() {
	fl.filter.refresh()
	fl.list.ClampSelection()
	fl.updateCounter()
}

func (fl *FilterListC[T]) updateCounter() {
	fl.counterMatch = fl.filter.Len()
	fl.counterTotal = len(*fl.filter.source)
}

// streaming reports whether a stream is currently active.
func (fl *FilterListC[T]) streaming() bool {
	return fl.isStreaming
}

// ============================================================================
// StreamWriter — managed streaming
// ============================================================================

// StreamWriter provides a write interface for streaming items into a
// FilterList. Obtained from [FilterListC.Stream]. Each write appends
// to the source slice, updates the filter index, and signals a render.
// Call [StreamWriter.Close] when done to stop the spinner.
type StreamWriter[T any] struct {
	fl     *FilterListC[T]
	render func()
	done   chan struct{}
}

// Write appends a single item and triggers a filter + render update.
func (w *StreamWriter[T]) Write(item T) {
	*w.fl.filter.source = append(*w.fl.filter.source, item)
	w.fl.appended()
	w.render()
}

// WriteAll appends multiple items in one batch and triggers a single
// filter + render update.
func (w *StreamWriter[T]) WriteAll(items []T) {
	*w.fl.filter.source = append(*w.fl.filter.source, items...)
	w.fl.appended()
	w.render()
}

// Close ends the stream. The spinner disappears and a final render is
// signaled. Safe to call multiple times.
func (w *StreamWriter[T]) Close() {
	if !w.fl.isStreaming {
		return
	}
	close(w.done)
	w.fl.isStreaming = false
	w.render()
}

// Stream begins a streaming session: a spinner appears next to the
// counter and the returned [StreamWriter] can be used to append items.
// Call [StreamWriter.Close] when all items have been written.
//
// requestRender is called after each write and spinner tick to signal
// that the UI should redraw (typically app.RequestRender).
func (fl *FilterListC[T]) Stream(requestRender func()) *StreamWriter[T] {
	fl.isStreaming = true
	w := &StreamWriter[T]{
		fl:     fl,
		render: requestRender,
		done:   make(chan struct{}),
	}

	// spinner animation ticker — stops when Close is called
	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-w.done:
				return
			case <-ticker.C:
				atomic.AddInt32(&fl.spinnerFrame, 1)
				requestRender()
			}
		}
	}()

	return w
}

// Placeholder sets the input placeholder text.
func (fl *FilterListC[T]) Placeholder(p string) *FilterListC[T] {
	fl.placeholder = p
	return fl
}

// Render customises how each item appears in the list.
// fn: func(item *T) any. return a component tree for the item row.
func (fl *FilterListC[T]) Render(fn func(*T) any) *FilterListC[T] {
	fl.list.Render(fn)
	return fl
}

// MaxVisible sets the maximum number of visible items.
func (fl *FilterListC[T]) MaxVisible(n int) *FilterListC[T] {
	fl.maxVisible = n
	return fl
}

// Border sets the border style.
func (fl *FilterListC[T]) Border(b BorderStyle) *FilterListC[T] {
	fl.border = b
	return fl
}

// Title sets the border title.
func (fl *FilterListC[T]) Title(t string) *FilterListC[T] {
	fl.title = t
	return fl
}

// Margin sets uniform margin on all sides.
func (fl *FilterListC[T]) Margin(all int16) *FilterListC[T] {
	fl.margin = [4]int16{all, all, all, all}
	return fl
}

// MarginVH sets vertical and horizontal margin.
func (fl *FilterListC[T]) MarginVH(v, h int16) *FilterListC[T] {
	fl.margin = [4]int16{v, h, v, h}
	return fl
}

// MarginTRBL sets individual margins for top, right, bottom, left.
func (fl *FilterListC[T]) MarginTRBL(t, r, b, l int16) *FilterListC[T] {
	fl.margin = [4]int16{t, r, b, l}
	return fl
}

// Handle registers a key binding that acts on the currently selected item.
// fn: func(item *T). receives a pointer to the selected source item (skipped if empty).
func (fl *FilterListC[T]) Handle(key string, fn func(*T)) *FilterListC[T] {
	fl.list.declaredBindings = append(fl.list.declaredBindings,
		binding{pattern: key, handler: func() {
			if item := fl.Selected(); item != nil {
				fn(item)
			}
		}},
	)
	return fl
}

// HandleClear registers a key that clears the filter when active,
// or calls the fallback when no filter is applied.
func (fl *FilterListC[T]) HandleClear(key string, fallback func()) *FilterListC[T] {
	fl.list.declaredBindings = append(fl.list.declaredBindings,
		binding{pattern: key, handler: func() {
			if fl.Active() {
				fl.Clear()
			} else if fallback != nil {
				fallback()
			}
		}},
	)
	return fl
}

// BindNav overrides the default navigation keys.
func (fl *FilterListC[T]) BindNav(down, up string) *FilterListC[T] {
	fl.list.BindNav(down, up)
	return fl
}

// Selected returns a pointer to the original source item corresponding
// to the current list selection. Returns nil if nothing is selected.
func (fl *FilterListC[T]) Selected() *T {
	idx := fl.list.Index()
	return fl.filter.Original(idx)
}

// SelectedIndex returns the index into the original source slice.
// Returns -1 if nothing is selected.
func (fl *FilterListC[T]) SelectedIndex() int {
	return fl.filter.OriginalIndex(fl.list.Index())
}

// Clear resets the filter and input.
func (fl *FilterListC[T]) Clear() {
	fl.input.Clear()
	fl.filter.Reset()
	fl.list.ClampSelection()
	fl.updateCounter()
}

// Active reports whether a filter query is currently applied.
func (fl *FilterListC[T]) Active() bool {
	return fl.filter.Active()
}

// SetQuery sets the filter query directly and syncs the list.
func (fl *FilterListC[T]) SetQuery(q string) {
	fl.input.SetValue(q)
	fl.sync()
}

// Query returns the current filter query string.
func (fl *FilterListC[T]) Query() string {
	return fl.input.Value()
}

// DeleteCharBefore removes the last character from the filter query.
func (fl *FilterListC[T]) DeleteCharBefore() {
	v := fl.input.Value()
	if len(v) > 0 {
		runes := []rune(v)
		fl.input.SetValue(string(runes[:len(runes)-1]))
		fl.sync()
	}
}

// SelectNext moves the list selection down by one.
func (fl *FilterListC[T]) SelectNext() {
	fl.list.Down(nil)
}

// SelectPrev moves the list selection up by one.
func (fl *FilterListC[T]) SelectPrev() {
	fl.list.Up(nil)
}

// PageDown moves the list selection down by a page.
func (fl *FilterListC[T]) PageDown() {
	fl.list.PageDown(nil)
}

// PageUp moves the list selection up by a page.
func (fl *FilterListC[T]) PageUp() {
	fl.list.PageUp(nil)
}

// Refresh re-evaluates the filter against the current source slice contents.
// Call this after mutating source items in-place (e.g. live data updates).
func (fl *FilterListC[T]) Refresh() {
	fl.refresh()
}

// Filter returns the underlying Filter for direct access.
func (fl *FilterListC[T]) Filter() *Filter[T] {
	return fl.filter
}

// Ref provides access to the FilterListC for external references.
func (fl *FilterListC[T]) Ref(f func(*FilterListC[T])) *FilterListC[T] {
	f(fl)
	return fl
}

// Marker sets the selection marker (default "> ").
func (fl *FilterListC[T]) Marker(m string) *FilterListC[T] {
	fl.list.Marker(m)
	return fl
}

// Style sets the default style for non-selected rows.
func (fl *FilterListC[T]) Style(s Style) *FilterListC[T] {
	fl.list.Style(s)
	return fl
}

// SelectedStyle sets the style for the selected row.
func (fl *FilterListC[T]) SelectedStyle(s Style) *FilterListC[T] {
	fl.list.SelectedStyle(s)
	return fl
}
