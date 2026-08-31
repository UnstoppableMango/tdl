package lex_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/unstoppablemango/tdl/lex"
)

// Every fixed spelling the tables report must lex back to the kind they
// report it for. A generator reads the tables and the parser reads the
// lexer, so a disagreement between them is a rule that can never match.
func TestFixedSpellingsLexBack(t *testing.T) {
	var all []string
	all = append(all, lex.Keywords()...)
	all = append(all, lex.Punctuation()...)

	if len(all) == 0 {
		t.Fatal("no fixed spellings")
	}

	for _, text := range all {
		kind, ok := lex.Lookup(text)
		if !ok {
			t.Errorf("Lookup(%q): not found", text)
			continue
		}
		if got := lex.Spelling(kind); got != text {
			t.Errorf("Spelling(%v) = %q, want %q", kind, got, text)
		}
		if got := lex.Pattern(kind); got != "" {
			t.Errorf("Pattern(%v) = %q, want \"\" for a fixed spelling", kind, got)
		}

		toks := scan(text)
		if len(toks) != 2 || toks[0].Kind != kind {
			t.Errorf("scan(%q) = %v, want [%v EOF]", text, kinds(text), kind)
		}
	}
}

// A class scanned by shape has a pattern and no spelling, and the two
// answers do not overlap.
func TestShapedKindsHavePatterns(t *testing.T) {
	shaped := []lex.Kind{lex.IDENT, lex.INT, lex.FLOAT, lex.STRING, lex.DOC, lex.REGEX}

	for _, k := range shaped {
		pattern := lex.Pattern(k)
		if pattern == "" {
			t.Errorf("Pattern(%v) is empty", k)
			continue
		}
		if _, err := regexp.Compile(pattern); err != nil {
			t.Errorf("Pattern(%v) does not compile: %v", k, err)
		}
		if got := lex.Spelling(k); got != "" {
			t.Errorf("Spelling(%v) = %q, want \"\"", k, got)
		}
		if _, ok := lex.Lookup(pattern); ok {
			t.Errorf("Lookup(%q): a pattern is not a spelling", pattern)
		}
	}
}

func anchored(t *testing.T, k lex.Kind) *regexp.Regexp {
	t.Helper()
	return regexp.MustCompile(`^(?:` + lex.Pattern(k) + `)`)
}

// The patterns say what the lexer accepts, so each one must match exactly
// the text the lexer took for that kind, and reject what it would not.
func TestPatternsMatchTheLexer(t *testing.T) {
	cases := []struct {
		kind   lex.Kind
		accept []string
		reject []string
	}{
		{lex.IDENT, []string{"a", "_x", "Order", "snake_case", "a1"}, []string{"1a", "-a", ""}},
		{lex.INT, []string{"0", "123", "-5"}, []string{"a", "-", "1.5"}},
		{lex.FLOAT, []string{"1.23", "-0.5"}, []string{"1", "1.", ".5"}},
		{lex.STRING, []string{`""`, `"abc"`, `"a\nb"`, `"a\"b"`}, []string{`"`, `"a`, `"a\qb"`}},
		{lex.DOC, []string{"///", "/// text"}, []string{"// text", "/"}},
		{lex.REGEX, []string{`//`, `/^[^@]+$/`, `/a\/b/`}, []string{`/`, `/a`}},
	}

	for _, c := range cases {
		re := anchored(t, c.kind)

		for _, src := range c.accept {
			match := re.FindString(src)
			if match != src {
				t.Errorf("%v: match of %q = %q, want the whole input", c.kind, src, match)
				continue
			}
			if got := lexOne(t, c.kind, src); got != c.kind {
				t.Errorf("%v: lexing %q gave %v", c.kind, src, got)
			}
		}

		for _, src := range c.reject {
			if match := re.FindString(src); match == src && src != "" {
				t.Errorf("%v: %q matched but should not", c.kind, src)
			}
		}
	}
}

// lexOne returns the kind the lexer gives the whole of src. REGEX is never
// produced by Next, so it is asked for the way the parser asks.
func lexOne(t *testing.T, want lex.Kind, src string) lex.Kind {
	t.Helper()
	l := lex.New("test.tdl", src)
	if want == lex.REGEX {
		return l.RescanRegexAt(lex.Position{Filename: "test.tdl", Line: 1, Col: 1}).Kind
	}
	return l.Next().Kind
}

// The table-driven cases are what a reader can hold; the corpus is what
// catches a pattern that is right about them and wrong about real source.
func TestPatternsAgreeWithTheLexerOverTheCorpus(t *testing.T) {
	sources, err := filepath.Glob(filepath.Join("..", "testdata", "conformance", "*", "*.tdl"))
	if err != nil {
		t.Fatal(err)
	}
	prelude, err := filepath.Glob(filepath.Join("..", "prelude", "*.tdl"))
	if err != nil {
		t.Fatal(err)
	}
	sources = append(sources, prelude...)
	if len(sources) == 0 {
		t.Fatal("no corpus sources")
	}

	seen := map[lex.Kind]int{}
	for _, path := range sources {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		src := string(b)

		for _, tok := range scan(src) {
			pattern := lex.Pattern(tok.Kind)
			if pattern == "" {
				continue
			}
			seen[tok.Kind]++

			match := regexp.MustCompile(`^(?:` + pattern + `)`).FindString(src[tok.Pos.Offset:])
			if match == "" {
				t.Errorf("%s:%s: %v pattern matched nothing", path, tok.Pos, tok.Kind)
				continue
			}
			if got := lexOne(t, tok.Kind, match); got != tok.Kind {
				t.Errorf("%s:%s: pattern took %q, which lexes as %v not %v",
					path, tok.Pos, match, got, tok.Kind)
			}
		}
	}

	// A pattern no corpus token exercised is a pattern this test did not
	// check, and saying so beats a green run that proved less than it looks.
	for _, k := range []lex.Kind{lex.IDENT, lex.STRING, lex.DOC} {
		if seen[k] == 0 {
			t.Errorf("no %v token in the corpus", k)
		}
	}
}
