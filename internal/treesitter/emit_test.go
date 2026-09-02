package treesitter_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/ebnf"
	"github.com/unstoppablemango/tdl/internal/treesitter"
)

// update rewrites tree-sitter/grammar.js instead of checking it:
//
//	go test ./internal/treesitter -update
var update = flag.Bool("update", false, "rewrite tree-sitter/grammar.js")

var (
	grammarPath = filepath.Join("..", "..", "docs", "grammar.ebnf")
	goldenPath  = filepath.Join("..", "..", "tree-sitter", "grammar.js")
)

// TestGrammarJS checks the committed grammar.js against the grammar it is
// derived from. It is the check the corpus cannot make: a production that
// reaches docs/grammar.ebnf and not the derived parser is a diff here.
func TestGrammarJS(t *testing.T) {
	got := emitDocs(t)

	if *update {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("writing %s: %v", goldenPath, err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading %s: %v (run `go test ./internal/treesitter -update`)", goldenPath, err)
	}
	if got != string(want) {
		t.Errorf("%s is out of date, run `go test ./internal/treesitter -update`", goldenPath)
	}
}

// TestEmitIsDeterministic is what makes the check above worth making. A
// Grammar is a map, so an emitter that iterated one would produce a
// different file every run and the diff would say nothing.
func TestEmitIsDeterministic(t *testing.T) {
	if first, second := emitDocs(t), emitDocs(t); first != second {
		t.Error("emitting the same grammar twice produced different bytes")
	}
}

func TestEmit(t *testing.T) {
	// Every case is a whole grammar, since Emit takes one, and asserts on
	// the one rule the case is about.
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			"sequence",
			"File = \"package\" identifier .\n" + ident,
			"file: $ => seq('package', $.identifier),",
		},
		{
			"alternation",
			"File = \"package\" | identifier .\n" + ident,
			"file: $ => choice('package', $.identifier),",
		},
		{
			"option",
			"File = [ identifier ] .\n" + ident,
			"file: $ => optional($.identifier),",
		},
		{
			"repetition",
			"File = { identifier } .\n" + ident,
			"file: $ => repeat($.identifier),",
		},
		{
			"group",
			"File = ( identifier ) \"package\" .\n" + ident,
			"file: $ => seq($.identifier, 'package'),",
		},
		{
			"name",
			"File = Other .\nOther = identifier .\n" + ident,
			"other: $ => $.identifier,",
		},
		{
			"snake case",
			"File = TypeArgs .\nTypeArgs = identifier .\n" + ident,
			"type_args: $ => $.identifier,",
		},
		{
			"hidden",
			"File = Other .\n/*@ hidden */\nOther = identifier .\n" + ident,
			"_other: $ => $.identifier,",
		},
		{
			"hidden reference",
			"File = Other .\n/*@ hidden */\nOther = identifier .\n" + ident,
			"file: $ => $._other,",
		},
		{
			// tree-sitter refuses a token in its own inline array, so the
			// substitution happens here instead.
			"inline",
			"File = Other identifier .\n/*@ inline */\nOther = \"package\" .\n" + ident,
			"file: $ => seq('package', $.identifier),",
		},
		{
			"prec",
			"/*@ prec 2 */\nFile = identifier .\n" + ident,
			"file: $ => prec(2, $.identifier),",
		},
		{
			"left associative",
			"/*@ prec.left 1 */\nFile = identifier .\n" + ident,
			"file: $ => prec.left(1, $.identifier),",
		},
		{
			"right associative",
			"/*@ prec.right 3 */\nFile = identifier .\n" + ident,
			"file: $ => prec.right(3, $.identifier),",
		},
		{
			"token production",
			"File = identifier .\n" + ident,
			"identifier: $ => /[_A-Za-z][_A-Za-z0-9]*/,",
		},
		{
			// A pattern is written for Go's regexp, which needs no
			// delimiter, so a bare '/' has to be escaped for JavaScript.
			"pattern with a slash",
			"/*@ extra line_comment */\n/*@ token line_comment = LineCommentPattern */\nFile = identifier .\n" + ident,
			`line_comment: $ => /\/\/[^\n]*/,`,
		},
		{
			"extras",
			"/*@ extra line_comment */\n/*@ token line_comment = LineCommentPattern */\nFile = identifier .\n" + ident,
			`extras: $ => [/\s/, $.line_comment],`,
		},
		{
			"word",
			"/*@ word identifier */\nFile = identifier .\n" + ident,
			"word: $ => $.identifier,",
		},
		{
			// The externals array names it; nothing emits a rule for it,
			// because scanner.c is what produces it.
			"external",
			"File = identifier regex_lit .\n" + ident + "/*@ token RegexPattern */\n/*@ external */\nregex_lit = .\n",
			"externals: $ => [$.regex_lit],",
		},
		{
			"conflict",
			"/*@ conflict File Other */\nFile = Other .\nOther = identifier .\n" + ident,
			"[$.file, $.other],",
		},
		{
			// One name is a rule that cannot be decided against its own
			// other readings, which is a thing tree-sitter accepts.
			"conflict with one production",
			"/*@ conflict File */\nFile = identifier .\n" + ident,
			"[$.file],",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := emit(t, c.src)
			if !strings.Contains(got, c.want) {
				t.Errorf("want %s in\n%s", c.want, got)
			}
		})
	}
}

func TestEmitDiagnostics(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			// Two spellings of one rule name would silently drop a rule.
			"colliding names",
			"File = TypeArgs type_args .\nTypeArgs = \"package\" .\n/*@ token IdentPattern */\ntype_args = .\n",
			"TypeArgs and type_args are both type_args",
		},
		{
			"self inlining",
			"File = Other .\n/*@ inline */\nOther = identifier Other .\n" + ident,
			"Other is inline and refers to itself",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			file := read(t, c.src)
			_, err := treesitter.Emit(file)
			if err == nil {
				t.Fatalf("want %s, got no error", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want %s", err, c.want)
			}
		})
	}
}

// ident is the one production every test grammar needs, since File has to
// reach something the lexer defines.
const ident = "/*@ token IdentPattern */\nidentifier = .\n"

// testOptions are GrammarOptions without the check that every terminal is
// TDL, so a case can quote whatever makes its point.
var testOptions = ebnf.Options{Start: "File", Annotated: true}

func read(t *testing.T, src string) *ebnf.File {
	t.Helper()

	file, errs := ebnf.Read("test.ebnf", src, testOptions)
	for _, err := range errs {
		t.Errorf("%v", err)
	}
	if file == nil {
		t.FailNow()
	}
	return file
}

func emit(t *testing.T, src string) string {
	t.Helper()

	js, err := treesitter.Emit(read(t, src))
	if err != nil {
		t.Fatal(err)
	}
	return string(js)
}

func emitDocs(t *testing.T) string {
	t.Helper()

	src, err := os.ReadFile(grammarPath)
	if err != nil {
		t.Fatal(err)
	}

	file, errs := ebnf.Read(grammarPath, string(src), ebnf.GrammarOptions)
	for _, err := range errs {
		t.Errorf("%v", err)
	}
	if file == nil {
		t.FailNow()
	}

	js, err := treesitter.Emit(file)
	if err != nil {
		t.Fatal(err)
	}
	return string(js)
}
