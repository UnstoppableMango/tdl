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
// declares: the generated package imports it and refers to it, and declares
// nothing for it. That is what makes the decimal, uuid, and date
// placeholders survivable, since the type a model means for them is a
// dependency this backend must not choose.
//
// A reference is qualified by an alias the import states, rather than by the
// path's last segment, because a package's name is not always its last
// segment (`gopkg.in/yaml.v3` is package `yaml`) and nothing in the model
// says which it is.

// foreignType is what a mapping states, with the alias its references are
// qualified by.
type foreignType struct {
	path  string
	name  string
	alias string
}

// ref is the Go expression naming the foreign type.
func (f foreignType) ref() string { return f.alias + "." + f.name }

// planForeign reads every mapping before anything is rendered, since a use
// of a foreign type anywhere asks for its alias, and two packages ending in
// one segment have to be told apart across the whole model.
func (g *generator) planForeign() {
	g.foreign = map[*ir.Decl]foreignType{}
	g.aliases = map[string]bool{}

	byPath := map[string]string{}
	for _, d := range g.Model.GetDecls() {
		dir, ok := g.Find(d.GetDirectives(), "foreign")
		if !ok {
			continue
		}
		args := dir.GetArgs()
		if len(args) != 2 {
			g.Warn(emit.Unsupported(dir.GetPosition(),
				"foreign takes an import path and a type name, and %d argument(s) were given", len(args)))
			continue
		}
		path, name := args[0].GetText(), args[1].GetText()
		if err := foreignProblem(dir.GetPosition(), d, path, name); err != nil {
			g.Warn(err)
			continue
		}
		alias, ok := byPath[path]
		if !ok {
			alias = g.alias(path)
			byPath[path] = alias
			g.aliases[alias] = true
		}
		g.foreign[d] = foreignType{path: path, name: name, alias: alias}
		g.warnForeignConstraints(d)

		// A key is a method, and a method is declared beside the type.
		if key, ok := g.Find(d.GetDirectives(), "key"); ok {
			g.Warn(emit.Unsupported(key.GetPosition(),
				"%s is a foreign type, so its key is not generated: a method is declared beside the type",
				emit.LastSegment(d.GetMeta().GetName())))
		}
	}
}

// warnForeignConstraints says at each constraint on a foreign declaration
// that nothing checks it: the values are another package's, and the methods
// a check would live on are declared beside the type rather than here.
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
			"%s is not checked: %s is a foreign type, whose values another package decides", constraintText(c), of))
	}
}

// foreignProblem reports a mapping the generated code could not refer to.
func foreignProblem(pos *ir.Position, d *ir.Decl, path, name string) error {
	of := d.GetMeta().GetName()
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

// alias is the identifier an import of path is given, derived from the
// segments that tell it apart from the imports already taken.
//
// A module's major version and a `go-` prefix are what a path carries and a
// package name does not, so `gopkg.in/yaml.v3`, `example.com/money/v2`, and
// `github.com/google/go-cmp` are `yaml`, `money`, and `cmp`. The alias is
// explicit either way, so this is about what a reader expects rather than
// about the reference resolving.
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

// aliasTaken reports whether an alias would shadow something the generated
// file already names: another import, a declaration, or a Go predeclared
// identifier.
//
// A mapping onto a package the backend imports itself is not a collision,
// since one import of one path is what the file writes either way.
func (g *generator) aliasTaken(alias, path string) bool {
	if p, ok := importNames[alias]; ok && p != path {
		return true
	}
	return g.aliases[alias] ||
		types.Universe.Lookup(alias) != nil || token.IsKeyword(alias) || g.declares(alias)
}

// major matches the major version a module path states, as its own segment
// or as a suffix of the last one.
var major = regexp.MustCompile(`(/v[0-9]+|\.v[0-9]+)$`)

// identifier is a path's trailing segments as one Go identifier, with the
// `go-` prefix a repository name often carries and whatever Go does not
// accept in an identifier dropped: `go-cmp` is `cmp`.
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
// type, which is what makes it something to import rather than to declare.
func (g *generator) isForeign(d *ir.Decl) bool {
	_, ok := g.foreign[d]
	return ok
}
