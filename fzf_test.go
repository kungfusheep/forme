package glyph

import (
	"testing"
)

func TestParseFzfQuery(t *testing.T) {
	t.Run("simple fuzzy", func(t *testing.T) {
		q := ParseFzfQuery("foo")
		if len(q.groups) != 1 {
			t.Fatalf("expected 1 group, got %d", len(q.groups))
		}
		if len(q.groups[0].terms) != 1 {
			t.Fatalf("expected 1 term, got %d", len(q.groups[0].terms))
		}
		term := q.groups[0].terms[0]
		if term.kind != termFuzzy {
			t.Errorf("expected fuzzy, got %d", term.kind)
		}
		if term.pattern != "foo" {
			t.Errorf("expected 'foo', got %q", term.pattern)
		}
		if term.negated {
			t.Error("should not be negated")
		}
		if term.caseSensitive {
			t.Error("lowercase should not be case-sensitive")
		}
	})

	t.Run("case sensitive when uppercase", func(t *testing.T) {
		q := ParseFzfQuery("Foo")
		term := q.groups[0].terms[0]
		if !term.caseSensitive {
			t.Error("uppercase pattern should be case-sensitive")
		}
	})

	t.Run("exact term", func(t *testing.T) {
		q := ParseFzfQuery("'exact")
		term := q.groups[0].terms[0]
		if term.kind != termExact {
			t.Errorf("expected exact, got %d", term.kind)
		}
		if term.pattern != "exact" {
			t.Errorf("expected 'exact', got %q", term.pattern)
		}
	})

	t.Run("prefix term", func(t *testing.T) {
		q := ParseFzfQuery("^prefix")
		term := q.groups[0].terms[0]
		if term.kind != termPrefix {
			t.Errorf("expected prefix, got %d", term.kind)
		}
		if term.pattern != "prefix" {
			t.Errorf("expected 'prefix', got %q", term.pattern)
		}
	})

	t.Run("suffix term", func(t *testing.T) {
		q := ParseFzfQuery("suffix$")
		term := q.groups[0].terms[0]
		if term.kind != termSuffix {
			t.Errorf("expected suffix, got %d", term.kind)
		}
		if term.pattern != "suffix" {
			t.Errorf("expected 'suffix', got %q", term.pattern)
		}
	})

	t.Run("negated term", func(t *testing.T) {
		q := ParseFzfQuery("!nope")
		term := q.groups[0].terms[0]
		if !term.negated {
			t.Error("should be negated")
		}
		if term.kind != termFuzzy {
			t.Errorf("expected fuzzy, got %d", term.kind)
		}
	})

	t.Run("negated exact", func(t *testing.T) {
		q := ParseFzfQuery("!'nope")
		term := q.groups[0].terms[0]
		if !term.negated || term.kind != termExact {
			t.Errorf("expected negated exact, got negated=%v kind=%d", term.negated, term.kind)
		}
	})

	t.Run("AND terms", func(t *testing.T) {
		q := ParseFzfQuery("foo bar baz")
		if len(q.groups) != 1 {
			t.Fatalf("expected 1 group, got %d", len(q.groups))
		}
		if len(q.groups[0].terms) != 3 {
			t.Fatalf("expected 3 terms, got %d", len(q.groups[0].terms))
		}
	})

	t.Run("OR groups", func(t *testing.T) {
		q := ParseFzfQuery("foo | bar")
		if len(q.groups) != 2 {
			t.Fatalf("expected 2 groups, got %d", len(q.groups))
		}
		if q.groups[0].terms[0].pattern != "foo" {
			t.Errorf("first group pattern = %q, want foo", q.groups[0].terms[0].pattern)
		}
		if q.groups[1].terms[0].pattern != "bar" {
			t.Errorf("second group pattern = %q, want bar", q.groups[1].terms[0].pattern)
		}
	})

	t.Run("complex query", func(t *testing.T) {
		q := ParseFzfQuery("^start 'mid !end$ | other")
		if len(q.groups) != 2 {
			t.Fatalf("expected 2 groups, got %d", len(q.groups))
		}

		g1 := q.groups[0]
		if len(g1.terms) != 3 {
			t.Fatalf("first group: expected 3 terms, got %d", len(g1.terms))
		}
		if g1.terms[0].kind != termPrefix {
			t.Errorf("term[0] should be prefix")
		}
		if g1.terms[1].kind != termExact {
			t.Errorf("term[1] should be exact")
		}
		if g1.terms[2].kind != termSuffix || !g1.terms[2].negated {
			t.Errorf("term[2] should be negated suffix")
		}

		g2 := q.groups[1]
		if len(g2.terms) != 1 {
			t.Fatalf("second group: expected 1 term, got %d", len(g2.terms))
		}
		if g2.terms[0].kind != termFuzzy {
			t.Errorf("term should be fuzzy")
		}
	})

	t.Run("bare pipe is not OR", func(t *testing.T) {
		q := ParseFzfQuery("foo|bar")
		if len(q.groups) != 1 {
			t.Fatalf("expected 1 group (bare pipe), got %d", len(q.groups))
		}
	})

	t.Run("empty query", func(t *testing.T) {
		q := ParseFzfQuery("")
		if len(q.groups) != 0 {
			t.Fatalf("expected 0 groups, got %d", len(q.groups))
		}
		if !q.Empty() {
			t.Error("empty query should report Empty()")
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		q := ParseFzfQuery("   ")
		if len(q.groups) != 0 {
			t.Fatalf("expected 0 groups, got %d", len(q.groups))
		}
	})
}

func TestFzfQueryScore(t *testing.T) {
	t.Run("empty query matches everything", func(t *testing.T) {
		q := ParseFzfQuery("")
		_, matched := q.Score("anything")
		if !matched {
			t.Error("empty query should match")
		}
	})

	t.Run("fuzzy match", func(t *testing.T) {
		q := ParseFzfQuery("abc")
		_, matched := q.Score("axbycz")
		if !matched {
			t.Error("should fuzzy match")
		}
	})

	t.Run("fuzzy no match", func(t *testing.T) {
		q := ParseFzfQuery("xyz")
		_, matched := q.Score("abcdef")
		if matched {
			t.Error("should not match")
		}
	})

	t.Run("AND requires all terms", func(t *testing.T) {
		q := ParseFzfQuery("quick fox")
		_, matched := q.Score("the quick brown fox")
		if !matched {
			t.Error("both terms present, should match")
		}
		_, matched = q.Score("the quick brown dog")
		if matched {
			t.Error("fox missing, should not match")
		}
	})

	t.Run("OR matches either", func(t *testing.T) {
		q := ParseFzfQuery("xyz | fox")
		_, matched := q.Score("the quick brown fox")
		if !matched {
			t.Error("second OR term matches, should match")
		}
	})

	t.Run("AND groups joined by OR", func(t *testing.T) {
		q := ParseFzfQuery("quick fox | slow cat | lazy dog")

		// first OR group matches: "quick" AND "fox" both present
		_, matched := q.Score("the quick brown fox")
		if !matched {
			t.Error("first AND group matches, should match")
		}

		// third OR group matches: "lazy" AND "dog" both present
		_, matched = q.Score("the lazy old dog")
		if !matched {
			t.Error("third AND group matches, should match")
		}

		// second OR group: "slow" present but "cat" missing, fails
		// other groups also fail
		_, matched = q.Score("the slow brown fox")
		if matched {
			t.Error("no AND group fully satisfied, should not match")
		}

		// none match
		_, matched = q.Score("hello world")
		if matched {
			t.Error("nothing matches, should not match")
		}
	})

	t.Run("negation", func(t *testing.T) {
		q := ParseFzfQuery("!xyz")
		_, matched := q.Score("abcdef")
		if !matched {
			t.Error("xyz not present, negation should match")
		}
		_, matched = q.Score("xyz")
		if matched {
			t.Error("xyz present, negation should not match")
		}
	})
}

func TestFzfAndOrPrecedence(t *testing.T) {
	type tc struct {
		name      string
		query     string
		candidate string
		want      bool
	}

	tests := []tc{
		// basic structure: ` | ` splits OR groups, space splits AND terms within
		// "a b | c d" = (a AND b) OR (c AND d)

		// --- single OR group with AND ---
		{"AND satisfied", "quick fox", "the quick brown fox", true},
		{"AND partial fail", "quick cat", "the quick brown fox", false},
		{"AND both missing", "slow cat", "the quick brown fox", false},

		// --- pure OR ---
		{"OR first matches", "fox | cat", "the quick brown fox", true},
		{"OR second matches", "fox | cat", "the lazy house cat", true},
		{"OR neither matches", "fox | cat", "the slow brown dog", false},

		// --- AND groups joined by OR ---
		// (quick AND fox) OR (lazy AND dog) OR (fast AND cat)
		{"OR-AND first group", "quick fox | lazy dog | fast cat", "the quick brown fox", true},
		{"OR-AND second group", "quick fox | lazy dog | fast cat", "the lazy old dog", true},
		{"OR-AND third group", "quick fox | lazy dog | fast cat", "the fast house cat", true},
		{"OR-AND no group satisfied", "quick fox | lazy dog | fast cat", "the slow brown fox", false},
		// fuzzy "cat" matches "the quick lazy fast animal" via subsequence c-a-t (quiCk lAzy fasT)
		// use exact terms to test strict AND isolation across OR groups
		{"OR-AND partial across groups fuzzy", "quick fox | lazy dog | fast cat", "the quick lazy fast animal", true},
		{"OR-AND partial across groups exact", "'quick 'fox | 'lazy 'dog | 'fast 'cat", "the quick lazy fast animal", false},

		// --- single term OR groups: a | b | c ---
		{"single-term OR first", "alpha | beta | gamma", "alpha testing", true},
		{"single-term OR middle", "alpha | beta | gamma", "beta release", true},
		{"single-term OR last", "alpha | beta | gamma", "gamma ray", true},
		{"single-term OR none", "alpha | beta | gamma", "delta force", false},

		// --- AND with negation inside OR groups ---
		// (!bad AND good) OR (nice)
		{"negation in AND group matches", "!bad good | nice", "good morning", true},
		{"negation in AND group blocked", "!bad good | nice", "bad good", false},
		{"negation in AND group falls to OR", "!bad good | nice", "nice day", true},

		// --- exact + prefix + suffix inside OR groups ---
		// (^start AND end$) OR ('middle)
		{"prefix+suffix AND group", "^the fox$ | 'brown", "the quick brown fox", true},
		{"exact match in OR fallback", "^the fox$ | 'brown", "a brown bear", true},
		{"prefix+suffix AND both needed", "^the fox$ | 'brown", "the quick red dog", false},

		// --- three AND terms in one group, OR'd with another ---
		// (a AND b AND c) OR (x)
		{"three-way AND satisfied", "quick brown fox | unicorn", "the quick brown fox", true},
		{"three-way AND one missing", "quick brown fox | unicorn", "the quick red fox", false},
		{"three-way AND fails, OR fallback", "quick brown fox | unicorn", "a magical unicorn", true},

		// --- edge: single term on each side ---
		{"single | single first", "foo | bar", "foo", true},
		{"single | single second", "foo | bar", "bar", true},
		{"single | single neither", "foo | bar", "baz", false},

		// --- verify AND binds tighter: "a b | c" is (a AND b) OR c, NOT a AND (b OR c) ---
		{"precedence: a b | c, AND group matches", "quick fox | zzz", "the quick brown fox", true},
		{"precedence: a b | c, OR fallback", "quick fox | zzz", "zzz", true},
		{"precedence: a b | c, only 'quick' not enough", "quick fox | zzz", "the quick brown dog", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := ParseFzfQuery(tt.query)
			_, matched := q.Score(tt.candidate)
			if matched != tt.want {
				t.Errorf("query=%q candidate=%q: got matched=%v, want %v", tt.query, tt.candidate, matched, tt.want)
			}
		})
	}
}

func BenchmarkFzfScore1000(b *testing.B) {
	candidates := make([]string, 1000)
	for i := range candidates {
		candidates[i] = "item_" + string(rune('a'+i%26)) + "_longer_suffix_text"
	}
	q := ParseFzfQuery("ist")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range candidates {
			q.Score(candidates[j])
		}
	}
}
