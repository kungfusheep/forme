package forme

// FilterListC is a drop-in filterable list. it composes an input, a
// filter and a list into a single template node.
//
// usage:
//
//	FilterList(&items, func(p *Profile) string { return p.Name }).
//	    Placeholder("filter...").
//	    Render(func(p *Profile) any { return Text(p.Name) }).
//	    MaxVisible(20).
//	    Handle("<Enter>", func(p *Profile) { ... })
type FilterListC[T any] struct {
	input  *InputC
	list   *ListC[T]
	filter *Filter[T]

	placeholder string
	maxVisible  int
	border      BorderStyle
	title       string
}

// FilterList creates a filterable list.
// extract returns the searchable text for each item.
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
	return fl
}

// toTemplate returns the template tree for compilation.
func (fl *FilterListC[T]) toTemplate() any {
	fl.input.placeholder = fl.placeholder
	if fl.maxVisible > 0 {
		fl.list.maxVisible = fl.maxVisible
	}

	children := []any{
		HBox(
			Text("> ").Bold(),
			fl.input,
		),
		fl.list,
	}

	if fl.border.Horizontal != 0 {
		return VBox.Border(fl.border).Title(fl.title)(children...)
	}
	return VBox(children...)
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
}

// Placeholder sets the input placeholder text.
func (fl *FilterListC[T]) Placeholder(p string) *FilterListC[T] {
	fl.placeholder = p
	return fl
}

// Render sets the render function for each list item.
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

// Handle registers a key binding that passes the currently selected
// original source item to the callback.
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
// to the current list selection. returns nil if nothing is selected.
func (fl *FilterListC[T]) Selected() *T {
	idx := fl.list.Index()
	return fl.filter.Original(idx)
}

// SelectedIndex returns the index into the original source slice.
// returns -1 if nothing is selected.
func (fl *FilterListC[T]) SelectedIndex() int {
	return fl.filter.OriginalIndex(fl.list.Index())
}

// Clear resets the filter and input.
func (fl *FilterListC[T]) Clear() {
	fl.input.Clear()
	fl.filter.Reset()
	fl.list.ClampSelection()
}

// Active reports whether a filter query is currently applied.
func (fl *FilterListC[T]) Active() bool {
	return fl.filter.Active()
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
