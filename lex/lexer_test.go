package lex_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/lex"
)

func collect(src string) []lex.Token {
	l := lex.New("test.tdl", src)
	var toks []lex.Token
	for {
		tok := l.Next()
		toks = append(toks, tok)
		if tok.Kind == lex.EOF {
			return toks
		}
	}
}

func kinds(toks []lex.Token) []lex.Kind {
	ks := make([]lex.Kind, len(toks))
	for i, t := range toks {
		ks[i] = t.Kind
	}
	return ks
}

func assertKinds(t *testing.T, src string, want ...lex.Kind) {
	t.Helper()
	got := kinds(collect(src))
	want = append(want, lex.EOF)
	if len(got) != len(want) {
		t.Fatalf("%q: got %v, want %v", src, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%q: token %d: got %v, want %v", src, i, got[i], want[i])
		}
	}
}

func TestKeywordsAndIdents(t *testing.T) {
	assertKinds(t, "package import as type enum union true false",
		lex.PACKAGE, lex.IMPORT, lex.AS, lex.TYPE, lex.ENUM, lex.UNION, lex.TRUE, lex.FALSE)
	assertKinds(t, "User list map", lex.IDENT, lex.IDENT, lex.IDENT)
}

func TestPunctuation(t *testing.T) {
	assertKinds(t, `{}()[]<>:,=?.@`,
		lex.LBRACE, lex.RBRACE, lex.LPAREN, lex.RPAREN, lex.LBRACK, lex.RBRACK,
		lex.LT, lex.GT, lex.COLON, lex.COMMA, lex.EQUAL,
		lex.QUESTION, lex.DOT, lex.AT)
}

func TestLiterals(t *testing.T) {
	assertKinds(t, `"hello" 123 1.5`, lex.STRING, lex.INT, lex.FLOAT)

	toks := collect(`"a\nb"`)
	if toks[0].Text != "a\nb" {
		t.Fatalf("got %q, want %q", toks[0].Text, "a\nb")
	}
}

func TestComments(t *testing.T) {
	assertKinds(t, "type // a comment\nUser", lex.TYPE, lex.IDENT)
}

func TestUnterminatedString(t *testing.T) {
	toks := collect(`"abc`)
	if toks[0].Kind != lex.ILLEGAL {
		t.Fatalf("got %v, want ILLEGAL", toks[0].Kind)
	}
}

func TestPositions(t *testing.T) {
	toks := collect("type\nUser")
	if toks[0].Pos.Line != 1 || toks[0].Pos.Col != 1 {
		t.Fatalf("got %+v, want line 1 col 1", toks[0].Pos)
	}
	if toks[1].Pos.Line != 2 || toks[1].Pos.Col != 1 {
		t.Fatalf("got %+v, want line 2 col 1", toks[1].Pos)
	}
}

func TestFullDeclaration(t *testing.T) {
	src := `package example.v1

type User {
  id: string
  name: string?
  tags: list<string>
  role: Role = "member"
}`
	toks := collect(src)
	if toks[len(toks)-1].Kind != lex.EOF {
		t.Fatalf("last token should be EOF, got %v", toks[len(toks)-1].Kind)
	}
	for _, tok := range toks {
		if tok.Kind == lex.ILLEGAL {
			t.Fatalf("unexpected ILLEGAL token: %+v", tok)
		}
	}
}
