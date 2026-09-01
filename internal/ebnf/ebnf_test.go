package ebnf_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/ebnf"
)

// notationOptions lint docs/notation.ebnf, which is a grammar of the
// notation rather than of TDL, so its terminals are its own and its
// quoted terminals are not TDL tokens.
var notationOptions = ebnf.Options{
	Terminals: []string{"nonterminal", "terminal", "quoted"},
}

func lint(t *testing.T, src string) []string {
	t.Helper()
	var out []string
	for _, d := range ebnf.Lint(src, ebnf.GrammarOptions) {
		out = append(out, d.String())
	}
	return out
}

func TestCleanGrammar(t *testing.T) {
	if got := lint(t, `File = { Decl } .
Decl = "package" identifier "." .
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
		{"undefined nonterminal", `File = Decl .`, "never defined"},
		{"defined twice", "File = Decl .\nDecl = \"package\" .\nDecl = \"import\" .", "defined again"},
		{"never used", "File = \"package\" .\nDecl = \"import\" .", "never used"},
		{"unknown terminal", `File = mystery .`, "unknown terminal"},
		{"invented spelling", `File = "notakeyword!" .`, "not text the lexer produces"},
		{"missing terminator", "File = \"package\"\nDecl = \"import\" .", "missing the closing"},
		{"unbalanced bracket", "File = [ \"package\" .\nDecl = \"import\" .", "unbalanced"},
		{"unterminated comment", "(* open\nFile = \"package\" .", "unterminated comment"},
		{"unterminated quote", "File = \"package .", "unterminated quoted"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := lint(t, c.src)
			for _, d := range got {
				if strings.Contains(d, c.want) {
					return
				}
			}
			t.Errorf("got %v, want one containing %q", got, c.want)
		})
	}
}

// A `"."` terminal is not the end of a production, and a comment holding
// an `=` is not the start of one.
func TestPunctuationInsideAProduction(t *testing.T) {
	if got := lint(t, `File = { "." } (* a = b *) .`); len(got) != 0 {
		t.Errorf("got %v, want none", got)
	}
}

func TestDocsAreClean(t *testing.T) {
	for _, c := range []struct {
		path string
		opts ebnf.Options
	}{
		{filepath.Join("..", "..", "docs", "grammar.ebnf"), ebnf.GrammarOptions},
		{filepath.Join("..", "..", "docs", "notation.ebnf"), notationOptions},
	} {
		b, err := os.ReadFile(c.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range ebnf.Lint(string(b), c.opts) {
			t.Errorf("%s:%s", filepath.Base(c.path), d)
		}
	}
}
