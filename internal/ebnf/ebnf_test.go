package ebnf_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/ebnf"
)

func lint(src string) []string {
	var out []string
	for _, err := range ebnf.Lint("test.ebnf", src, ebnf.GrammarOptions) {
		out = append(out, err.Error())
	}
	return out
}

func TestCleanGrammar(t *testing.T) {
	if got := lint(`File = { Decl } .
Decl = "package" identifier "." .
identifier = .
`); len(got) != 0 {
		t.Errorf("got %v, want none", got)
	}
}

func TestDiagnostics(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"undefined nonterminal", `File = Decl .`, "missing production"},
		{"never reached", "File = \"package\" .\nDecl = \"import\" .", "unreachable"},
		{"missing terminator", `File = "package"`, "expected"},
		{"unbalanced bracket", `File = [ "package" .`, "expected"},
		{"stray close", `File = ] .`, "expected"},
		{"unterminated comment", "/* open\nFile = \"package\" .", "not terminated"},
		{"unterminated quote", "File = \"package .", "string not terminated"},

		// The check the library does not make.
		{"invented spelling", `File = "notakeyword!" .`, "not text the lexer produces"},
		{"spelling with trailing text", `File = "package x" .`, "not text the lexer produces"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := lint(c.src)
			for _, d := range got {
				if strings.Contains(d, c.want) {
					return
				}
			}
			t.Errorf("got %v, want one containing %q", got, c.want)
		})
	}
}

// `_` is a terminal the lexer scans as an identifier rather than as a
// fixed spelling, so lex.Lookup alone would reject a grammar that is
// right. ImportDecl depends on it.
func TestUnderscoreIsALegalTerminal(t *testing.T) {
	if got := lint(`File = "import" ( identifier | "_" ) .
identifier = .
`); len(got) != 0 {
		t.Errorf("got %v, want none", got)
	}
}

// A lexical production carries no expression: the name is the lexer's and
// the grammar only says it exists.
func TestLexicalProductionsNeedNoBody(t *testing.T) {
	if got := lint("File = identifier .\nidentifier = .\n"); len(got) != 0 {
		t.Errorf("got %v, want none", got)
	}
}

func TestDocsAreClean(t *testing.T) {
	for _, c := range []struct {
		name string
		opts ebnf.Options
	}{
		{"grammar.ebnf", ebnf.GrammarOptions},
		{"notation.ebnf", ebnf.NotationOptions},
	} {
		path := filepath.Join("..", "..", "docs", c.name)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range ebnf.Lint(c.name, string(b), c.opts) {
			t.Errorf("%v", d)
		}
	}
}
