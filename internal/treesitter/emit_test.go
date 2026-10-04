package treesitter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/ebnf"
	"github.com/unstoppablemango/tdl/internal/treesitter"
)

var (
	grammarPath = filepath.Join("..", "..", "docs", "grammar.ebnf")
	goldenPath  = filepath.Join("..", "..", "tree-sitter", "grammar.js")
)

// TestGrammarJS checks the committed grammar.js against docs/grammar.ebnf.
// tools/treesitter writes the file; this only reads it.
func TestGrammarJS(t *testing.T) {
	got := emitDocs(t)

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading %s: %v (run `make treesitter`)", goldenPath, err)
	}
	if got != string(want) {
		t.Errorf("%s is out of date, run `make treesitter`", goldenPath)
	}
}

func TestEmitIsDeterministic(t *testing.T) {
	if first, second := emitDocs(t), emitDocs(t); first != second {
		t.Error("emitting the same grammar twice produced different bytes")
	}
}

func TestEmit(t *testing.T) {
	// Each case asserts on one rule of a whole grammar.
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
			"colliding names",
			"File = TypeArgs type_args .\nTypeArgs = \"package\" .\n/*@ token IdentPattern */\ntype_args = .\n",
			"TypeArgs and type_args are both type_args",
		},
		{
			"self inlining",
			"File = Other .\n/*@ inline */\nOther = identifier Other .\n" + ident,
			"Other is inline and refers to itself",
		},
		{
			"inlining a production with no expression",
			"File = identifier .\n/*@ token IdentPattern */\n/*@ inline */\nidentifier = .\n",
			"identifier is inline and has no expression",
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

// ident is the lexical production every test grammar needs.
const ident = "/*@ token IdentPattern */\nidentifier = .\n"

// testOptions are GrammarOptions without the lex spelling check.
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

	file, err := ebnf.ReadFile(grammarPath, ebnf.GrammarOptions)
	if err != nil {
		t.Fatal(err)
	}
	js, err := treesitter.Emit(file)
	if err != nil {
		t.Fatal(err)
	}
	return string(js)
}
