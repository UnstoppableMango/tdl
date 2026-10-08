package golang

import (
	"go/token"
	"go/types"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// A `foreign` directive maps a declaration to a type another package
// declares, which the generated code imports instead of declaring. It is how
// a model replaces the decimal, uuid, and date placeholders.
//
// Every foreign import gets an explicit alias, since a package's name is not
// always its path's last segment (`gopkg.in/yaml.v3` is package `yaml`).

// foreignType is a mapping and the alias its references are qualified by.
type foreignType struct {
	path  string
	name  string
	alias string
}

func (f foreignType) ref() string { return f.alias + "." + f.name }

// planForeign reads every mapping before rendering, so aliases are unique
// across the whole model.
func (g *generator) planForeign() {
	g.foreign = map[*ir.Decl]foreignType{}
	g.externs = map[*ir.Extern]foreignType{}
	g.aliases = map[string]bool{}

	byPath := map[string]string{}
	for _, d := range g.Model.GetDecls() {
		f, ok := g.mapping(d.GetDirectives(), d.GetMeta().GetName(), byPath)
		if !ok {
			continue
		}
		g.foreign[d] = f
		g.warnForeignConstraints(d)

		if key, ok := g.Find(d.GetDirectives(), "key"); ok {
			g.Warn(emit.Unsupported(key.GetPosition(),
				"%s is a foreign type, so its key is not generated: a method is declared beside the type",
				emit.LastSegment(d.GetMeta().GetName())))
		}
	}

	// A target path can also map a declaration another TDL package owns.
	for _, e := range g.Model.GetExterns() {
		if f, ok := g.mapping(e.GetDirectives(), e.GetPackage()+"."+e.GetName(), byPath); ok {
			g.externs[e] = f
		}
	}
}

// mapping reads the foreign directive among dirs, warning when it is
// unusable. byPath holds each import path's alias, so a path is imported
// under one alias across the model.
func (g *generator) mapping(dirs []*ir.Directive, of string, byPath map[string]string) (foreignType, bool) {
	dir, ok := g.Find(dirs, "foreign")
	if !ok {
		return foreignType{}, false
	}
	args := dir.GetArgs()
	if len(args) != 2 {
		g.Warn(emit.Unsupported(dir.GetPosition(),
			"foreign takes an import path and a type name, and %d argument(s) were given", len(args)))
		return foreignType{}, false
	}
	path, name := args[0].GetText(), args[1].GetText()
	if err := foreignProblem(dir.GetPosition(), of, path, name); err != nil {
		g.Warn(err)
		return foreignType{}, false
	}
	alias, ok := byPath[path]
	if !ok {
		alias = g.alias(path)
		byPath[path] = alias
		g.aliases[alias] = true
	}
	return foreignType{path: path, name: name, alias: alias}, true
}

// externForeign is the mapping of the extern a type refers to, if any.
func (g *generator) externForeign(t *ir.Type) (foreignType, bool) {
	i := int(t.GetExtern().GetIndex())
	if i < 0 || i >= len(g.Model.GetExterns()) {
		return foreignType{}, false
	}
	f, ok := g.externs[g.Model.GetExterns()[i]]
	return f, ok
}

// warnForeignConstraints warns at each constraint on a foreign declaration,
// since a foreign type cannot carry a Validate method.
func (g *generator) warnForeignConstraints(d *ir.Decl) {
	of := emit.LastSegment(d.GetMeta().GetName())
	cs := slices.Clone(d.GetNewtype().GetValueConstraints())
	fields := slices.Clone(d.Fields())
	for _, v := range d.GetEnumeration().GetVariants() {
		fields = append(fields, v.GetFields()...)
	}
	for _, f := range fields {
		cs = append(cs, f.GetConstraints()...)
	}
	for _, c := range cs {
		g.Warn(emit.Unsupported(c.GetPosition(),
			"%s is not checked: %s is a foreign type, whose values another package decides", emit.ConstraintText(c), of))
	}
}

// foreignProblem reports a mapping the generated code could not refer to.
func foreignProblem(pos *ir.Position, of, path, name string) error {
	switch {
	case path == "":
		return emit.Unsupported(pos, "the foreign mapping of %s has no import path", of)
	case !token.IsIdentifier(name):
		return emit.Unsupported(pos, "%s is mapped to %q, which is not a Go identifier", of, name)
	case !token.IsExported(name):
		return emit.Unsupported(pos, "%s is mapped to %s, which %s does not export", of, name, path)
	}
	return nil
}

// alias is the identifier an import of path is given, from its last one or
// two segments, or numbered when those are taken. A major version and a
// `go-` prefix are dropped, so `gopkg.in/yaml.v3`, `example.com/money/v2`,
// and `github.com/google/go-cmp` are `yaml`, `money`, and `cmp`.
func (g *generator) alias(path string) string {
	segments := strings.Split(major.ReplaceAllString(path, ""), "/")
	candidates := []string{identifier(last(segments, 1))}
	if len(segments) > 1 {
		candidates = append(candidates, identifier(last(segments, 2)))
	}
	for _, c := range candidates {
		if c != "" && !g.aliasTaken(c, path) {
			return c
		}
	}

	base := candidates[0]
	if base == "" {
		base = "pkg"
	}
	for n := 2; ; n++ {
		c := base + strconv.Itoa(n)
		if !g.aliasTaken(c, path) {
			return c
		}
	}
}

// aliasTaken reports whether an alias would shadow another import, a
// declaration, or a predeclared identifier. A mapping onto a package the
// backend imports itself is not a collision.
func (g *generator) aliasTaken(alias, path string) bool {
	if p, ok := importNames[alias]; ok && p != path {
		return true
	}
	return g.aliases[alias] ||
		types.Universe.Lookup(alias) != nil || token.IsKeyword(alias) || g.declares(alias)
}

// major matches a module path's major version suffix.
var major = regexp.MustCompile(`(/v[0-9]+|\.v[0-9]+)$`)

// identifier makes s a Go identifier, dropping a `go-` prefix and any
// character an identifier cannot hold.
func identifier(s string) string {
	s = strings.TrimPrefix(s, "go-")
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || r == '_':
			b.WriteRune(r)
		case unicode.IsDigit(r) && b.Len() > 0:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// last is the final n segments of a path, joined.
func last(segments []string, n int) string {
	if n > len(segments) {
		n = len(segments)
	}
	return strings.Join(segments[len(segments)-n:], "")
}

// isForeign reports whether a declaration is mapped to another package's
// type.
func (g *generator) isForeign(d *ir.Decl) bool {
	_, ok := g.foreign[d]
	return ok
}
