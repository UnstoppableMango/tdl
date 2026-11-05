package zod

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// A `matches` constraint is checked only under the `patterns` directive on
// the target block, since it runs the author's pattern on a backtracking
// engine. The pattern is printed from RE2's parse by [emit.ECMAScript], so
// it means what RE2 means, and declared as a module constant before the
// schema checking it, named for that schema. docs/architecture/
// zod-backend.md's Patterns is the design.

// pattern is a module constant holding a printed pattern.
type pattern struct {
	name, source, literal string
}

// checkable is the pattern a `matches` constraint is checked against, or
// why it is not checked: without the patterns directive, a pattern RE2
// refuses, one naming a surrogate, or one whose shape a backtracking
// engine takes exponential time on.
func (g *generator) checkable(c *ir.Constraint) (*regexp.Regexp, string) {
	if !g.checkPatterns {
		return nil, "Zod checks a pattern only under the patterns directive"
	}
	re, err := emit.Pattern(c.GetArgs()[0])
	if err != nil {
		return nil, err.Error()
	}
	if emit.Surrogates(re) {
		return nil, "it names a surrogate code point, which RE2 never sees and JavaScript can match"
	}
	if emit.Backtracks(re) {
		return nil, "JavaScript's engine backtracks, and can take exponential time on this pattern's shape"
	}
	return regexp.MustCompile(c.GetArgs()[0].GetText()), ""
}

// pattern is a `matches` check's method call, declaring its constant for
// the schema being written, or why it has none.
func (g *generator) pattern(c *ir.Constraint, msg string) (string, string) {
	if _, why := g.checkable(c); why != "" {
		return "", why
	}
	if !g.onString {
		return "", "Zod checks a pattern only on a string schema"
	}
	re, _ := emit.Pattern(c.GetArgs()[0])
	name := g.prefix + "Pattern"
	if n := len(g.patterns); n > 0 {
		name = fmt.Sprintf("%s%d", name, n+1)
	}
	g.patterns = append(g.patterns, pattern{name: name, source: emit.ConstraintText(c), literal: "/" + emit.ECMAScript(re) + "/u"})
	return ".regex(" + name + ", " + msg + ")", ""
}

// lineTerminators escapes what would end a // comment early.
var lineTerminators = strings.NewReplacer("\n", `\n`, "\r", `\r`, "\u2028", `\u2028`, "\u2029", `\u2029`)

// declarePatterns writes the constants the schema being written checks,
// each under the constraint it prints, and returns their names.
func (g *generator) declarePatterns(b *strings.Builder) []string {
	var names []string
	for _, p := range g.patterns {
		fmt.Fprintf(b, "// %s\nconst %s = %s;\n", lineTerminators.Replace(p.source), p.name, p.literal)
		names = append(names, p.name)
	}
	g.patterns = nil
	return names
}
