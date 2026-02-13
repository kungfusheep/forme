package forme

// Filter provides fzf-style filtering mechanics for a slice of items.
// it handles query parsing, scoring, filtering and index mapping back to the
// original source slice. no UI opinions — bring your own rendering.
//
// usage:
//
//	f := NewFilter(&items, func(item *Item) string { return item.Name })
//	f.Update("query")           // re-filter when query changes
//	f.Items                     // filtered+ranked subset — point a ListC at &f.Items
//	f.Original(selectedIndex)   // map filtered index back to source item
type Filter[T any] struct {
	Items []T // filtered+ranked subset, safe to point a ListC at &f.Items

	source    *[]T
	extract   func(*T) string
	lastQuery string
	query     FzfQuery
	indices   []int    // indices[i] = index into *source for Items[i]
	matches   []scored // reusable scratch for scoring
}

type scored struct {
	index int
	score int
}

// NewFilter creates a filter over a source slice.
// extract returns the searchable text for each item.
func NewFilter[T any](source *[]T, extract func(*T) string) *Filter[T] {
	f := &Filter[T]{
		source:  source,
		extract: extract,
	}
	f.Reset()
	return f
}

// Update re-filters the source slice with a new query string.
// no-op if the query hasn't changed.
func (f *Filter[T]) Update(query string) {
	if query == f.lastQuery {
		return
	}
	f.lastQuery = query
	f.query = ParseFzfQuery(query)

	if f.query.Empty() {
		f.Reset()
		return
	}

	// score all source items, collect matches (reuse scratch slice)
	src := *f.source
	matches := f.matches[:0]
	if cap(matches) < len(src) {
		matches = make([]scored, 0, len(src))
	}
	for i := range src {
		text := f.extract(&src[i])
		score, ok := f.query.Score(text)
		if ok {
			matches = append(matches, scored{index: i, score: score})
		}
	}

	// sort by score descending, then by original index ascending
	for i := 1; i < len(matches); i++ {
		j := i
		for j > 0 && scoredLess(matches[j], matches[j-1]) {
			matches[j], matches[j-1] = matches[j-1], matches[j]
			j--
		}
	}

	f.matches = matches // save for reuse next call

	// rebuild Items and indices
	f.Items = f.Items[:0]
	f.indices = f.indices[:0]
	for _, m := range matches {
		f.Items = append(f.Items, src[m.index])
		f.indices = append(f.indices, m.index)
	}
}

// Reset clears the filter, restoring all source items in original order.
func (f *Filter[T]) Reset() {
	f.lastQuery = ""
	f.query = FzfQuery{}

	src := *f.source
	if cap(f.Items) < len(src) {
		f.Items = make([]T, len(src))
		f.indices = make([]int, len(src))
	} else {
		f.Items = f.Items[:len(src)]
		f.indices = f.indices[:len(src)]
	}
	copy(f.Items, src)
	for i := range f.indices {
		f.indices[i] = i
	}
}

// Original maps a filtered index back to a pointer into the source slice.
// returns nil if the index is out of bounds.
func (f *Filter[T]) Original(filteredIndex int) *T {
	if filteredIndex < 0 || filteredIndex >= len(f.indices) {
		return nil
	}
	src := *f.source
	origIdx := f.indices[filteredIndex]
	if origIdx < 0 || origIdx >= len(src) {
		return nil
	}
	return &src[origIdx]
}

// OriginalIndex maps a filtered index back to the index in the source slice.
// returns -1 if the index is out of bounds.
func (f *Filter[T]) OriginalIndex(filteredIndex int) int {
	if filteredIndex < 0 || filteredIndex >= len(f.indices) {
		return -1
	}
	return f.indices[filteredIndex]
}

// Active reports whether a filter query is currently applied.
func (f *Filter[T]) Active() bool {
	return !f.query.Empty()
}

// Query returns the current raw query string.
func (f *Filter[T]) Query() string {
	return f.lastQuery
}

// Len returns the number of currently visible (filtered) items.
func (f *Filter[T]) Len() int {
	return len(f.Items)
}

func scoredLess(a, b struct {
	index int
	score int
}) bool {
	if a.score != b.score {
		return a.score > b.score
	}
	return a.index < b.index
}
