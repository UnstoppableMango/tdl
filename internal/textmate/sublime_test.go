package textmate_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/ebnf"
	"github.com/unstoppablemango/tdl/internal/textmate"
)

var sublimePath = filepath.Join("..", "..", "editors", "bat", "tdl.sublime-syntax")

// TestSublimeSyntax checks the committed sublime-syntax is up to date. Only
// tools/textmate writes it.
func TestSublimeSyntax(t *testing.T) {
	got := emitSublime(t)

	want, err := os.ReadFile(sublimePath)
	if err != nil {
		t.Fatalf("reading %s: %v (run `make textmate`)", sublimePath, err)
	}
	if got != string(want) {
		t.Errorf("%s is out of date, run `make textmate`", sublimePath)
	}
}

// TestSublimeContextsResolve checks every context a rule pushes or sets is
// defined, and every defined context is reached, since bat refuses the
// first and the second is a region that lost its begin.
func TestSublimeContextsResolve(t *testing.T) {
	out := emitSublime(t)

	defined := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^  ([a-z0-9-]+):$`).FindAllStringSubmatch(out, -1) {
		defined[m[1]] = true
	}
	if !defined["main"] {
		t.Fatal("no main context")
	}

	reached := map[string]bool{"main": true}
	for _, m := range regexp.MustCompile(`(?m)^      (?:push|set): (\S+)$|^    - include: (\S+)$`).FindAllStringSubmatch(out, -1) {
		name := m[1] + m[2]
		reached[name] = true
		if !defined[name] {
			t.Errorf("a rule names context %q, which is not defined", name)
		}
	}
	for name := range defined {
		if !reached[name] {
			t.Errorf("context %q is defined and nothing reaches it", name)
		}
	}
}

// TestSublimeKeepsTheRules checks the sublime-syntax carries every match
// the TextMate grammar does, so the two color alike.
func TestSublimeKeepsTheRules(t *testing.T) {
	out := emitSublime(t)

	var check func(rules []pattern)
	check = func(rules []pattern) {
		for _, rule := range rules {
			for _, re := range []string{rule.Match, rule.End} {
				if re != "" && !strings.Contains(out, quoteYAML(re)) {
					t.Errorf("the sublime-syntax has no rule matching %s", re)
				}
			}
			check(rule.Patterns)
		}
	}
	check(parse(t).Patterns)
}

func quoteYAML(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func emitSublime(t *testing.T) string {
	t.Helper()

	grammar, err := ebnf.ReadFile(grammarPath, ebnf.GrammarOptions)
	if err != nil {
		t.Fatal(err)
	}
	out, err := textmate.EmitSublime(grammar)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}
