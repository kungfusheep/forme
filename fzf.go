package forme

import (
	"strings"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// fzf query parser and scoring engine.
// uses junegunn/fzf's algo package for matching/scoring.
//
// query syntax:
//   "foo"     fuzzy subsequence match
//   "'foo"    exact substring match
//   "^foo"    prefix match
//   "foo$"    suffix match
//   "!foo"    negated fuzzy match
//   "!'foo"   negated exact match
//   "!^foo"   negated prefix match
//   "!foo$"   negated suffix match
//   "a b"     AND — all space-separated terms must match
//   "a | b"   OR  — at least one pipe-separated term must match

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
// returns (score, matched). higher score = better match.
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
