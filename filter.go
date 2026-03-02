package glyph

import (
	"strings"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// ============================================================================
// fzf query parser and scoring engine
// ============================================================================

// fzf query parser and scoring engine.
// uses junegunn/fzf's algo package for matching/scoring.
//
// query syntax:
//
//	"foo"     fuzzy subsequence match
//	"'foo"    exact substring match
//	"^foo"    prefix match
//	"foo$"    suffix match
//	"!foo"    negated fuzzy match
//	"!'foo"   negated exact match
//	"!^foo"   negated prefix match
//	"!foo$"   negated suffix match
//	"a b"     AND   all space-separated terms must match
//	"a | b"   OR    at least one pipe-separated term must match

func init() {
	algo.Init("default")
}

var fzfSlab = util.MakeSlab(100*1024, 2048)

// FzfQuery is a pre-parsed fzf query. parse once, score many.
type FzfQuery struct {
	groups []fzfGroup
}

type fzfGroup struct {
	terms []fzfTerm
}

type fzfTermKind int

const (
	termFuzzy fzfTermKind = iota
	termExact
	termPrefix
	termSuffix
)

type fzfTerm struct {
	pattern       string
	patRunes      []rune
	kind          fzfTermKind
	negated       bool
	caseSensitive bool
}

// ParseFzfQuery parses a raw query string into a reusable FzfQuery.
func ParseFzfQuery(raw string) FzfQuery {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return FzfQuery{}
	}

	var q FzfQuery

	orCount := 1
	for i := 0; i < len(raw)-2; i++ {
		if raw[i] == ' ' && raw[i+1] == '|' && raw[i+2] == ' ' {
			orCount++
		}
	}
	q.groups = make([]fzfGroup, 0, orCount)

	rest := raw
	for {
		idx := strings.Index(rest, " | ")
		var part string
		if idx < 0 {
			part = rest
		} else {
			part = rest[:idx]
		}

		part = strings.TrimSpace(part)
		if part != "" {
			g := parseGroup(part)
			if len(g.terms) > 0 {
				q.groups = append(q.groups, g)
			}
		}

		if idx < 0 {
			break
		}
		rest = rest[idx+3:]
	}
	return q
}

// Empty reports whether the query has no terms.
func (q *FzfQuery) Empty() bool {
	return len(q.groups) == 0
}

func parseGroup(part string) fzfGroup {
	tokenCount := 0
	inWord := false
	for i := 0; i < len(part); i++ {
		if part[i] == ' ' || part[i] == '\t' {
			inWord = false
		} else if !inWord {
			tokenCount++
			inWord = true
		}
	}

	g := fzfGroup{terms: make([]fzfTerm, 0, tokenCount)}

	start := -1
	for i := 0; i <= len(part); i++ {
		isSpace := i < len(part) && (part[i] == ' ' || part[i] == '\t')
		atEnd := i == len(part)
		if start < 0 {
			if !isSpace && !atEnd {
				start = i
			}
		} else if isSpace || atEnd {
			g.terms = append(g.terms, parseTerm(part[start:i]))
			start = -1
		}
	}
	return g
}

func parseTerm(tok string) fzfTerm {
	t := fzfTerm{kind: termFuzzy}

	if len(tok) > 1 && tok[0] == '!' {
		t.negated = true
		tok = tok[1:]
	}

	if len(tok) > 1 && tok[0] == '\'' {
		t.kind = termExact
		tok = tok[1:]
	} else if len(tok) > 1 && tok[0] == '^' {
		t.kind = termPrefix
		tok = tok[1:]
	} else if len(tok) > 1 && tok[len(tok)-1] == '$' {
		t.kind = termSuffix
		tok = tok[:len(tok)-1]
	}

	t.caseSensitive = hasUppercase(tok)
	if !t.caseSensitive {
		tok = strings.ToLower(tok)
	}

	t.pattern = tok
	t.patRunes = []rune(tok)
	return t
}

func hasUppercase(s string) bool {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if unicode.IsUpper(r) {
			return true
		}
		i += size
	}
	return false
}

// Score scores a single candidate against the parsed query.
// Returns (score, matched). Higher score = better match.
func (q *FzfQuery) Score(candidate string) (int, bool) {
	if len(q.groups) == 0 {
		return 0, true
	}

	bestScore := -1
	matched := false
	for i := range q.groups {
		score, ok := q.groups[i].score(candidate)
		if ok && score > bestScore {
			matched = true
			bestScore = score
		}
	}
	return bestScore, matched
}

func (g *fzfGroup) score(candidate string) (int, bool) {
	totalScore := 0
	for i := range g.terms {
		score, ok := g.terms[i].score(candidate)
		if !ok {
			return 0, false
		}
		totalScore += score
	}
	return totalScore, true
}

func (t *fzfTerm) score(candidate string) (int, bool) {
	// avoid []byte copy: algo functions only read from Chars, never mutate the backing slice
	chars := util.ToChars(unsafe.Slice(unsafe.StringData(candidate), len(candidate)))

	// direct dispatch: avoids function variable that prevents escape analysis
	// from proving &chars stays on the stack
	var result algo.Result
	switch t.kind {
	case termExact:
		result, _ = algo.ExactMatchNaive(t.caseSensitive, false, true, &chars, t.patRunes, false, fzfSlab)
	case termPrefix:
		result, _ = algo.PrefixMatch(t.caseSensitive, false, true, &chars, t.patRunes, false, fzfSlab)
	case termSuffix:
		result, _ = algo.SuffixMatch(t.caseSensitive, false, true, &chars, t.patRunes, false, fzfSlab)
	default:
		result, _ = algo.FuzzyMatchV2(t.caseSensitive, false, true, &chars, t.patRunes, false, fzfSlab)
	}
	matched := result.Start >= 0

	if t.negated {
		return 0, !matched
	}
	if !matched {
		return 0, false
	}
	return result.Score, true
}

// ============================================================================
// Filter: headless fzf-style filtering over any slice
// ============================================================================

// Filter provides fzf-style filtering mechanics for a slice of items.
// It handles query parsing, scoring, filtering and index mapping back to the
// original source slice. No UI opinions, bring your own rendering.
//
// usage:
//
//	f := NewFilter(&items, func(item *Item) string { return item.Name })
//	f.Update("query")           // re-filter when query changes
//	f.Items                     // filtered+ranked subset, point a ListC at &f.Items
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
// No-op if the query hasn't changed.
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
// Returns nil if the index is out of bounds.
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
// Returns -1 if the index is out of bounds.
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
