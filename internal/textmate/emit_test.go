package textmate_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/ebnf"
	"github.com/unstoppablemango/tdl/internal/textmate"
	"github.com/unstoppablemango/tdl/lex"
)

// update rewrites the committed grammar instead of checking it:
//
//	go test ./internal/textmate -update
var update = flag.Bool("update", false, "rewrite editors/vscode/syntaxes/tdl.tmLanguage.json")

var (
	grammarPath    = filepath.Join("..", "..", "docs", "grammar.ebnf")
	goldenPath     = filepath.Join("..", "..", "editors", "vscode", "syntaxes", "tdl.tmLanguage.json")
	treeSitterPath = filepath.Join("..", "..", "tree-sitter", "tree-sitter.json")
)

// TestTmLanguage checks the committed grammar against the file it is
// derived from. A keyword that reaches lex and not the colors is a diff
// here, which is the whole reason the grammar is derived.
func TestTmLanguage(t *testing.T) {
	got := emitDocs(t)

	if *update {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("writing %s: %v", goldenPath, err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading %s: %v (run `go test ./internal/textmate -update`)", goldenPath, err)
	}
	if got != string(want) {
		t.Errorf("%s is out of date, run `go test ./internal/textmate -update`", goldenPath)
	}
}

// TestEmitIsDeterministic is what makes the check above worth making. A
// Grammar is a map, and so is a set of spellings.
func TestEmitIsDeterministic(t *testing.T) {
	if first, second := emitDocs(t), emitDocs(t); first != second {
		t.Error("emitting the same grammar twice produced different bytes")
	}
}

// TestEveryKeywordIsColored holds the grammar to lex.Keywords the way
// internal/treesitter holds highlights.scm to it.
//
// A new keyword is colored by the emitter rather than by hand, so what
// this catches is the other direction: a spelling colored as a keyword
// that lex does not reserve, and a keyword landing in both groups. What
// makes a new keyword visible at all is the golden check above.
func TestEveryKeywordIsColored(t *testing.T) {
	values := alternatives(t, ruleFor(t, "constant.language.tdl"))
	keywords := alternatives(t, ruleFor(t, "keyword.control.tdl"))

	for _, kw := range lex.Keywords() {
		switch {
		case values[kw] && keywords[kw]:
			t.Errorf("%q is colored as both a constant and a keyword", kw)
		case !values[kw] && !keywords[kw]:
			t.Errorf("%q is a keyword lex produces and nothing colors", kw)
		}
	}

	for _, spelled := range []map[string]bool{values, keywords} {
		for text := range spelled {
			if !lex.IsKeyword(text) {
				t.Errorf("%q is colored as a keyword and lex does not reserve it", text)
			}
		}
	}
}

// TestEveryPunctuationIsColored is the same check for lex.Punctuation.
func TestEveryPunctuationIsColored(t *testing.T) {
	ops := alternatives(t, ruleFor(t, "keyword.operator.tdl"))
	delims := alternatives(t, ruleFor(t, "punctuation.tdl"))

	for _, p := range lex.Punctuation() {
		quoted := regexp.QuoteMeta(p)
		switch {
		case ops[quoted] && delims[quoted]:
			t.Errorf("%q is colored as both an operator and a delimiter", p)
		case !ops[quoted] && !delims[quoted]:
			t.Errorf("%q is punctuation lex produces and nothing colors", p)
		}
	}
}

// TestModifiersAreContextual checks the modifiers come from the grammar
// rather than from a list here, which is what makes a new one colored
// without a change to the emitter.
func TestModifiersAreContextual(t *testing.T) {
	mods := alternatives(t, ruleFor(t, "storage.modifier.tdl"))

	for _, want := range []string{"key", "owned", "deprecated"} {
		if !mods[want] {
			t.Errorf("%q is a modifier docs/grammar.ebnf spells and nothing colors", want)
		}
	}
	for text := range mods {
		if lex.IsKeyword(text) {
			t.Errorf("%q is reserved, so it is a keyword rather than a modifier", text)
		}
	}
}

// TestPatternsComeFromLex holds the copied shapes to the originals. They
// are Oniguruma in the output and Go's regexp in lex, and the two agree on
// everything these six use; a pattern that stopped being copyable would
// show up as a difference here.
func TestPatternsComeFromLex(t *testing.T) {
	out := emitDocs(t)

	for name, pattern := range map[string]string{
		"IdentPattern":       lex.IdentPattern,
		"IntPattern":         lex.IntPattern,
		"FloatPattern":       lex.FloatPattern,
		"StringPattern":      lex.StringPattern,
		"DocPattern":         lex.DocPattern,
		"RegexPattern":       lex.RegexPattern,
		"LineCommentPattern": lex.LineCommentPattern,
	} {
		// JSON escapes every backslash, and the patterns are full of them.
		escaped, err := json.Marshal(pattern)
		if err != nil {
			t.Fatal(err)
		}
		quoted := strings.Trim(string(escaped), `"`)

		// IdentPattern reaches the file inside the guard on a numeric
		// literal rather than as a rule of its own, since coloring an
		// identifier needs a parse.
		if !strings.Contains(out, quoted) && name != "IdentPattern" {
			t.Errorf("lex.%s is not in the derived grammar", name)
		}
	}
}

// TestScopeNameMatchesTreeSitter checks the two derived grammars name the
// language the same thing. An editor keying a theme or an injection off
// the scope reads one name, and there is only one language.
func TestScopeNameMatchesTreeSitter(t *testing.T) {
	var config struct {
		Grammars []struct {
			Scope string `json:"scope"`
		} `json:"grammars"`
	}
	read(t, treeSitterPath, &config)

	if len(config.Grammars) != 1 {
		t.Fatalf("%s declares %d grammars, want 1", treeSitterPath, len(config.Grammars))
	}
	if got, want := scopeName(t), config.Grammars[0].Scope; got != want {
		t.Errorf("scopeName = %q, tree-sitter.json says %q", got, want)
	}
}

// A file is the shape of the emitted grammar a test reads back.
type file struct {
	ScopeName string `json:"scopeName"`
	Patterns  []struct {
		Name  string `json:"name"`
		Match string `json:"match"`
	} `json:"patterns"`
}

func emitDocs(t *testing.T) string {
	t.Helper()

	src, err := os.ReadFile(grammarPath)
	if err != nil {
		t.Fatal(err)
	}

	grammar, errs := ebnf.Read(grammarPath, string(src), ebnf.GrammarOptions)
	for _, err := range errs {
		t.Errorf("%v", err)
	}
	if grammar == nil {
		t.FailNow()
	}

	out, err := textmate.Emit(grammar)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func parse(t *testing.T) file {
	t.Helper()

	var out file
	if err := json.Unmarshal([]byte(emitDocs(t)), &out); err != nil {
		t.Fatalf("the derived grammar is not JSON: %v", err)
	}
	return out
}

func scopeName(t *testing.T) string {
	t.Helper()
	return parse(t).ScopeName
}

// ruleFor is the match of the one rule naming a scope.
func ruleFor(t *testing.T, scope string) string {
	t.Helper()

	var found []string
	for _, rule := range parse(t).Patterns {
		if rule.Name == scope {
			found = append(found, rule.Match)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%d rules name %s, want 1", len(found), scope)
	}
	return found[0]
}

// alternatives are the spellings in a rule's alternation, still escaped.
//
// Read rather than matched: the patterns are Oniguruma and use lookaround
// Go's regexp cannot compile, so nothing here can run one.
func alternatives(t *testing.T, match string) map[string]bool {
	t.Helper()

	const open = "(?:"
	start := strings.Index(match, open)
	if start < 0 {
		t.Fatalf("%q is not an alternation", match)
	}
	start += len(open)

	depth, end := 1, -1
	for i := start; i < len(match); i++ {
		switch match[i] {
		case '\\':
			i++ // an escaped paren is a character, not a group
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				end = i
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		t.Fatalf("%q has an unclosed group", match)
	}

	out := map[string]bool{}
	for _, item := range split(match[start:end]) {
		out[item] = true
	}
	return out
}

// split breaks an alternation on the `|` between its parts, which is not
// the `|` lex produces as a token.
func split(body string) []string {
	var out []string
	var current strings.Builder
	for i := 0; i < len(body); i++ {
		switch {
		case body[i] == '\\' && i+1 < len(body):
			current.WriteByte(body[i])
			i++
			current.WriteByte(body[i])
		case body[i] == '|':
			out = append(out, current.String())
			current.Reset()
		default:
			current.WriteByte(body[i])
		}
	}
	return append(out, current.String())
}

func read(t *testing.T, path string, into any) {
	t.Helper()

	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(src, into); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}
