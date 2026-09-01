package lex_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/lex"
)

func scan(src string) []lex.Token {
	l := lex.New("test.tdl", src)
	var toks []lex.Token
	for {
		tok := l.Next()
		if tok.Kind == lex.EOF {
			return append(toks, tok)
		}
		toks = append(toks, tok)
	}
}

func kinds(src string) []lex.Kind {
	toks := scan(src)
	ks := make([]lex.Kind, len(toks))
	for i, t := range toks {
		ks[i] = t.Kind
	}
	return ks
}

func want(t *testing.T, src string, expected ...lex.Kind) {
	t.Helper()
	expected = append(expected, lex.EOF)
	got := kinds(src)
	if len(got) != len(expected) {
		t.Fatalf("%q: got %v, want %v", src, got, expected)
	}
	for i := range got {
		if got[i] != expected[i] {
			t.Fatalf("%q: got %v, want %v", src, got, expected)
		}
	}
}

func TestKeywordsAndIdents(t *testing.T) {
	want(t, "package import as primitive unit alias type value entity enum class mixin instance target for requires where include null true false",
		lex.PACKAGE, lex.IMPORT, lex.AS, lex.PRIMITIVE, lex.UNIT, lex.ALIAS,
		lex.TYPE, lex.VALUE, lex.ENTITY, lex.ENUM, lex.CLASS, lex.MIXIN,
		lex.INSTANCE, lex.TARGET, lex.FOR, lex.REQUIRES, lex.WHERE,
		lex.INCLUDE, lex.NULL, lex.TRUE, lex.FALSE)

	// Modifiers and constraint names are contextual, not reserved. So is
	// `union`, which nothing in the language claims.
	want(t, "key owned deprecated min max length matches oneOf unique union",
		lex.IDENT, lex.IDENT, lex.IDENT, lex.IDENT, lex.IDENT,
		lex.IDENT, lex.IDENT, lex.IDENT, lex.IDENT, lex.IDENT)
}

func TestOperators(t *testing.T) {
	want(t, "{}()[]<>:,=?.|", lex.LBRACE, lex.RBRACE, lex.LPAREN, lex.RPAREN,
		lex.LBRACK, lex.RBRACK, lex.LT, lex.GT, lex.COLON, lex.COMMA,
		lex.EQUAL, lex.QUESTION, lex.DOT, lex.PIPE)
	want(t, "-> .. ^ * / =>", lex.ARROW, lex.RANGE, lex.CARET, lex.STAR, lex.SLASH, lex.FATARROW)
}

// `1..254` is an int, a range, and an int. `0.5` is one float.
func TestNumbersVersusRange(t *testing.T) {
	want(t, "1..254", lex.INT, lex.RANGE, lex.INT)
	want(t, "1..", lex.INT, lex.RANGE)
	want(t, "0.5", lex.FLOAT)
	want(t, "-3", lex.INT)

	if got := scan("0.5")[0].Text; got != "0.5" {
		t.Errorf("float text = %q", got)
	}
	if got := scan("-3")[0].Text; got != "-3" {
		t.Errorf("negative int text = %q", got)
	}
}

func TestComments(t *testing.T) {
	// An ordinary comment is skipped; a doc comment is a token.
	want(t, "// gone\nprimitive string", lex.PRIMITIVE, lex.IDENT)
	want(t, "/// kept\nprimitive string", lex.DOC, lex.PRIMITIVE, lex.IDENT)

	if got := scan("/// kept")[0].Text; got != "kept" {
		t.Errorf("doc text = %q, want %q", got, "kept")
	}
	if got := scan("///kept")[0].Text; got != "kept" {
		t.Errorf("doc text without a space = %q", got)
	}
}

// Whitespace is insignificant and produces no tokens.
func TestWhitespaceProducesNoTokens(t *testing.T) {
	want(t, "primitive\n\n\tstring", lex.PRIMITIVE, lex.IDENT)
	want(t, "primitive string primitive int",
		lex.PRIMITIVE, lex.IDENT, lex.PRIMITIVE, lex.IDENT)
}

func TestStrings(t *testing.T) {
	if got := scan(`"a\"b\n"`)[0].Text; got != "a\"b\n" {
		t.Errorf("decoded string = %q", got)
	}
	if got := scan(`"unterminated`)[0].Kind; got != lex.ILLEGAL {
		t.Errorf("unterminated string kind = %v", got)
	}
}

func TestPositions(t *testing.T) {
	toks := scan("primitive\n  string")
	if p := toks[1].Pos; p.Line != 2 || p.Col != 3 {
		t.Errorf("second token at %v, want 2:3", p)
	}
}

// A regex literal is scanned only when the parser asks, because `/` is also
// division in a unit expression.
func TestRescanRegex(t *testing.T) {
	src := `matches /^[a-z]+$/`
	l := lex.New("test.tdl", src)

	if got := l.Next().Kind; got != lex.IDENT {
		t.Fatalf("first token = %v, want IDENT", got)
	}
	slash := l.Next()
	if slash.Kind != lex.SLASH {
		t.Fatalf("second token = %v, want SLASH", slash.Kind)
	}

	tok := l.RescanRegexAt(slash.Pos)
	if tok.Kind != lex.REGEX || tok.Text != "^[a-z]+$" {
		t.Fatalf("regex = %v %q", tok.Kind, tok.Text)
	}
	if got := l.Next().Kind; got != lex.EOF {
		t.Errorf("after regex = %v, want EOF", got)
	}
}

func TestRescanRegexUnterminated(t *testing.T) {
	l := lex.New("test.tdl", "/nope\n")
	if got := l.RescanRegexAt(lex.Position{Filename: "test.tdl", Line: 1, Col: 1}); got.Kind != lex.ILLEGAL {
		t.Errorf("unterminated regex kind = %v", got.Kind)
	}
}

// A unit expression divides; it is never rescanned as a regex.
func TestUnitDivisionStaysSlash(t *testing.T) {
	want(t, "kg*m/s^2", lex.IDENT, lex.STAR, lex.IDENT, lex.SLASH, lex.IDENT, lex.CARET, lex.INT)
}
