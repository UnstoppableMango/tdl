package textmate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/ebnf"
	"github.com/unstoppablemango/tdl/internal/textmate"
	"github.com/unstoppablemango/tdl/lex"
)

var (
	grammarPath    = filepath.Join("..", "..", "docs", "grammar.ebnf")
	goldenPath     = filepath.Join("..", "..", "editors", "vscode", "syntaxes", "tdl.tmLanguage.json")
	treeSitterPath = filepath.Join("..", "..", "tree-sitter", "tree-sitter.json")
)

// TestTmLanguage checks the committed grammar is up to date. Only
// tools/textmate writes it.
func TestTmLanguage(t *testing.T) {
	got := emitDocs(t)

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading %s: %v (run `make textmate`)", goldenPath, err)
	}
	if got != string(want) {
		t.Errorf("%s is out of date, run `make textmate`", goldenPath)
	}
}

func TestEmitIsDeterministic(t *testing.T) {
	if first, second := emitDocs(t), emitDocs(t); first != second {
		t.Error("emitting the same grammar twice produced different bytes")
	}
}

// TestEveryKeywordIsColored checks each lex keyword lands in exactly one
// group and nothing else is colored as a keyword.
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

func TestModifiersAreContextual(t *testing.T) {
	mods := alternatives(t, ruleFor(t, "storage.modifier.tdl"))

	for _, want := range []string{"owned", "deprecated"} {
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

// TestDeclarationKeywordsComeFromTheGrammar checks every keyword followed
// by an identifier in docs/grammar.ebnf colors the declared name.
func TestDeclarationKeywordsComeFromTheGrammar(t *testing.T) {
	declares := alternatives(t, captureRuleFor(t, "entity.name.type.tdl"))

	for _, want := range []string{
		"alias", "class", "enum", "mixin",
		"primitive", "target", "type", "unit",
	} {
		if !declares[want] {
			t.Errorf("%q introduces a name in docs/grammar.ebnf and nothing colors it", want)
		}
	}

	for _, unwanted := range []string{"import", "instance", "package"} {
		if declares[unwanted] {
			t.Errorf("%q does not introduce a declaration name", unwanted)
		}
	}
	for text := range declares {
		if !lex.IsKeyword(text) {
			t.Errorf("%q introduces a name and lex does not reserve it", text)
		}
	}
}

// TestTypeReferenceKeywordsComeFromTheGrammar is the same check for the
// keywords followed by a type reference.
func TestTypeReferenceKeywordsComeFromTheGrammar(t *testing.T) {
	refers := alternatives(t, ruleCapturing(t, "keyword.control.tdl", "support.type.tdl"))

	for _, want := range []string{"include", "requires"} {
		if !refers[want] {
			t.Errorf("%q names a type in docs/grammar.ebnf and nothing colors it", want)
		}
	}

	for _, unwanted := range []string{"package", "import", "entity", "type"} {
		if refers[unwanted] {
			t.Errorf("%q does not name a type", unwanted)
		}
	}
}

// TestPatternsComeFromLex checks every lex pattern appears verbatim in the
// output.
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
		escaped, err := json.Marshal(pattern)
		if err != nil {
			t.Fatal(err)
		}
		// Trim one quote per end: StringPattern ends in an escaped quote.
		quoted := strings.TrimSuffix(strings.TrimPrefix(string(escaped), `"`), `"`)

		if !strings.Contains(out, quoted) {
			t.Errorf("lex.%s is not in the derived grammar", name)
		}
	}
}

// TestTargetPathsAreColored checks the path rule reads entries ending in
// both `=>` and `{`.
func TestTargetPathsAreColored(t *testing.T) {
	match := ruleFor(t, "entity.name.namespace.tdl")

	for _, want := range []string{"=>", "\\{"} {
		if !strings.Contains(match, want) {
			t.Errorf("the path rule does not read an entry ending in %s", want)
		}
	}
}

// TestEnumBodyColorsItsVariants checks the enum region colors both bare
// variants and variants carrying fields.
func TestEnumBodyColorsItsVariants(t *testing.T) {
	var regions []pattern
	for _, rule := range parse(t).Patterns {
		if rule.BeginCaptures["2"].Name == "entity.name.type.tdl" {
			regions = append(regions, rule)
		}
	}
	if len(regions) != 1 {
		t.Fatalf("%d regions name a declaration, want 1", len(regions))
	}

	enum := regions[0]
	if !strings.Contains(enum.Begin, "(enum)") {
		t.Errorf("the region begins with %q, which does not read an enum", enum.Begin)
	}

	const variant = "variable.other.enummember.tdl"
	var bare, withBody bool
	for _, inner := range enum.Patterns {
		switch {
		case inner.Name == variant:
			bare = true
		case inner.BeginCaptures["1"].Name == variant:
			withBody = true
		}
	}
	if !bare {
		t.Errorf("nothing in the enum body colors a bare variant %s", variant)
	}
	if !withBody {
		t.Errorf("nothing in the enum body colors a variant carrying fields")
	}
}

// TestEveryOpenBraceIsClosedByItsRegion checks a rule reading `{` also
// reads its `}` and includes $self, so a nested brace cannot end a region
// early.
func TestEveryOpenBraceIsClosedByItsRegion(t *testing.T) {
	var check func(rules []pattern)
	check = func(rules []pattern) {
		for _, rule := range rules {
			if !strings.Contains(rule.Begin, `\{`) {
				check(rule.Patterns)
				continue
			}
			if rule.End != `\}` {
				t.Errorf("a rule beginning %q ends %q, so its brace is not its own", rule.Begin, rule.End)
			}
			if !includesSelf(rule.Patterns) {
				t.Errorf("a rule beginning %q does not include $self, so a brace nested in it ends the region", rule.Begin)
			}
			check(rule.Patterns)
		}
	}
	check(parse(t).Patterns)
}

func includesSelf(rules []pattern) bool {
	for _, rule := range rules {
		if rule.Include == "$self" {
			return true
		}
	}
	return false
}

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

// A file is the emitted grammar as the tests read it back.
type file struct {
	ScopeName string    `json:"scopeName"`
	Patterns  []pattern `json:"patterns"`
}

type pattern struct {
	Name          string             `json:"name"`
	Match         string             `json:"match"`
	Begin         string             `json:"begin"`
	End           string             `json:"end"`
	Include       string             `json:"include"`
	Captures      map[string]capture `json:"captures"`
	BeginCaptures map[string]capture `json:"beginCaptures"`
	Patterns      []pattern          `json:"patterns"`
}

type capture struct {
	Name string `json:"name"`
}

func emitDocs(t *testing.T) string {
	t.Helper()

	grammar, err := ebnf.ReadFile(grammarPath, ebnf.GrammarOptions)
	if err != nil {
		t.Fatal(err)
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

// captureRuleFor is the match of the one rule capturing a scope.
func captureRuleFor(t *testing.T, scope string) string {
	t.Helper()

	var found []string
	for _, rule := range parse(t).Patterns {
		for _, capture := range rule.Captures {
			if capture.Name == scope {
				found = append(found, rule.Match)
			}
		}
	}
	if len(found) != 1 {
		t.Fatalf("%d rules capture %s, want 1", len(found), scope)
	}
	return found[0]
}

// ruleCapturing is the match of the one rule capturing first and second
// as groups 1 and 2.
func ruleCapturing(t *testing.T, first, second string) string {
	t.Helper()

	var found []string
	for _, rule := range parse(t).Patterns {
		if rule.Captures["1"].Name == first && rule.Captures["2"].Name == second {
			found = append(found, rule.Match)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%d rules capture %s then %s, want 1", len(found), first, second)
	}
	return found[0]
}

// alternatives are the escaped spellings in a rule's first group. Parsed by
// hand, since Go's regexp cannot compile the lookaround.
func alternatives(t *testing.T, match string) map[string]bool {
	t.Helper()

	start := strings.Index(match, "(")
	if start < 0 {
		t.Fatalf("%q is not an alternation", match)
	}
	start++
	if strings.HasPrefix(match[start:], "?:") {
		start += len("?:")
	}

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

// split breaks an alternation on unescaped `|`.
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
