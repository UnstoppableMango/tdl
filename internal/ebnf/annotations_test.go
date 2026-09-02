package ebnf_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/ebnf"
	"github.com/unstoppablemango/tdl/lex"
)

func read(t *testing.T, src string) (*ebnf.File, []string) {
	t.Helper()
	file, errs := ebnf.Read("test.ebnf", src, ebnf.GrammarOptions)
	var out []string
	for _, err := range errs {
		out = append(out, err.Error())
	}
	return file, out
}

// The whole of phase 2 has to be readable from Go, which is what makes
// the emitter possible. This asserts against the real file rather than a
// fixture, since drifting from it is the failure that matters.
func TestGrammarAnnotationsAreRead(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "grammar.ebnf"))
	if err != nil {
		t.Fatal(err)
	}
	file, errs := ebnf.Read("grammar.ebnf", string(b), ebnf.GrammarOptions)
	for _, e := range errs {
		t.Errorf("%v", e)
	}

	a := file.Annotations
	if a.Word != "identifier" {
		t.Errorf("word = %q, want identifier", a.Word)
	}
	if strings.Join(a.Extras, ",") != "doc_comment,line_comment" {
		t.Errorf("extras = %v", a.Extras)
	}
	if len(a.Conflicts) != 3 {
		t.Errorf("conflicts = %v, want three", a.Conflicts)
	}

	// A token binding resolves to the pattern, not to the symbol name, so
	// a caller never has to know lex to use one.
	if got := a.Tokens["doc_comment"]; got != lex.DocPattern {
		t.Errorf("doc_comment = %q, want lex.DocPattern", got)
	}
	if got := a.Prods["identifier"].Token; got != lex.IdentPattern {
		t.Errorf("identifier = %q, want lex.IdentPattern", got)
	}

	if !a.Prods["Decl"].Hidden {
		t.Error("Decl is not hidden")
	}
	if !a.Prods["FieldRel"].Inline {
		t.Error("FieldRel is not inline")
	}
	if !a.Prods["regex_lit"].External {
		t.Error("regex_lit is not external")
	}
	if p := a.Prods["UnitExpr"]; p.Assoc != "left" || p.Prec != 1 {
		t.Errorf("UnitExpr = %+v, want left 1", p)
	}
	if p := a.Prods["Kind"]; p.Assoc != "right" || p.Prec != 1 {
		t.Errorf("Kind = %+v, want right 1", p)
	}

	// Every production the file marks is a production the grammar has.
	for name := range a.Prods {
		if _, ok := file.Grammar[name]; !ok {
			t.Errorf("%s is annotated and is not a production", name)
		}
	}
}

func TestAnnotationDiagnostics(t *testing.T) {
	// Every case is a legal grammar carrying one bad annotation, so what
	// fails is the annotation and not the productions around it.
	const prelude = "File = identifier .\n/*@ token IdentPattern */\nidentifier = .\n"

	cases := []struct {
		name string
		src  string
		want string
	}{
		{"unknown keyword", "/*@ mystery */\n" + prelude, "unknown annotation"},
		{"empty", "/*@ */\n" + prelude, "empty annotation"},
		{"word names nothing", "/*@ word Nope */\n" + prelude, "not a production"},
		{"two words", "/*@ word identifier */\n/*@ word File */\n" + prelude, "already"},
		{"conflict names nothing", "/*@ conflict File Nope */\n" + prelude, "not a production"},
		{"conflict needs a name", "/*@ conflict */\n" + prelude, "at least one"},
		{"extra with no binding", "/*@ extra ghost */\n" + prelude, "no token annotation binds"},
		{"unknown lex symbol", "/*@ token doc = Nope */\n" + prelude, "not a lex symbol"},
		{"prec without a level", "File = identifier .\n/*@ prec.left */\n/*@ token IdentPattern */\nidentifier = .\n", "takes one level"},
		{"prec with a word", "File = identifier .\n/*@ prec.left high */\n/*@ token IdentPattern */\nidentifier = .\n", "takes a number"},
		{"hidden with an argument", "/*@ hidden yes */\n" + prelude, "takes no arguments"},
		{"hidden and inline", "/*@ hidden */\n/*@ inline */\n" + prelude, "both hidden and inline"},
		{"nothing below it", prelude + "/*@ hidden */\n", "no production below it"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, got := read(t, c.src)

			// Every one of these has to say where, which is the whole
			// difference between a diagnostic and a complaint.
			for _, d := range got {
				if !strings.HasPrefix(d, "test.ebnf:") {
					t.Errorf("%q carries no position", d)
				}
			}

			for _, d := range got {
				if strings.Contains(d, c.want) {
					return
				}
			}
			t.Errorf("got %v, want one containing %q", got, c.want)
		})
	}
}

// An annotation attaches to the production below it, so two in a row
// both describe the same one. regex_lit depends on it.
func TestStackedAnnotationsAttachTogether(t *testing.T) {
	file, errs := read(t, "File = identifier .\n/*@ token IdentPattern */\n/*@ external */\nidentifier = .\n")
	if len(errs) != 0 {
		t.Fatalf("got %v, want none", errs)
	}
	if p := file.Annotations.Prods["identifier"]; !p.External || p.Token != lex.IdentPattern {
		t.Errorf("identifier = %+v, want external with a token", p)
	}
}

// Ordinary comments are not annotations, and an annotation written
// inside a line comment or a string literal is not one either.
//
// The spelling check is off here: `/*@ word Nope */` is not text the
// lexer produces, which is the point of putting it in a terminal.
func TestOnlyAnnotationCommentsAreRead(t *testing.T) {
	opts := ebnf.Options{Start: "File", Annotated: true}
	src := "/* hidden */\n// /*@ word Nope */\nFile = \"/*@ word Nope */\" .\n"

	file, errs := ebnf.Read("test.ebnf", src, opts)
	if len(errs) != 0 {
		t.Fatalf("got %v, want none", errs)
	}
	if a := file.Annotations; a.Word != "" || len(a.Prods) != 0 {
		t.Errorf("read %+v, want nothing", a)
	}
}
