package golang

import (
	"strings"
	"unicode"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// primitives maps a prelude primitive to the Go type standing for it.
//
// Each entry is a decision rather than a translation, and
// docs/design/go-backend.md says why for the three that are placeholders:
// Go has no decimal, no UUID, and no date without a time, and choosing a
// dependency for every consumer is not this backend's call.
var primitives = map[string]string{
	"string":   "string",
	"int":      "int64",
	"bool":     "bool",
	"bytes":    "[]byte",
	"uuid":     "string",
	"instant":  "time.Time",
	"date":     "time.Time",
	"duration": "time.Duration",
	"decimal":  "string",
}

// goType returns the Go type expression for a type reference.
func (g *generator) goType(id *ir.ID) (string, error) {
	ref, err := g.Resolve(id)
	if err != nil {
		return "", err
	}
	return g.goRef(ref)
}

// goRef maps a resolved type reference to Go.
func (g *generator) goRef(r *emit.Ref) (string, error) {
	switch r.Form {
	case emit.Prim:
		goName, ok := primitives[r.Name]
		if !ok {
			return "", emit.Unsupported(r.Pos, "primitive %s has no Go type", r.Name)
		}
		if strings.HasPrefix(goName, "time.") {
			g.needTime = true
		}
		return goName, nil
	case emit.List:
		e, err := g.goRef(r.Elem)
		if err != nil {
			return "", err
		}
		return "[]" + e, nil
	case emit.Set:
		// Go has no set. A slice would silently permit the duplicates the
		// type exists to forbid, so the key set of a map is the closest
		// thing that keeps the guarantee.
		e, err := g.goRef(r.Elem)
		if err != nil {
			return "", err
		}
		if !g.comparable(r.Elem.ID) {
			return "", emit.Unsupported(r.Pos,
				"a Set becomes a Go map, and %s is not a comparable Go type", e)
		}
		return "map[" + e + "]struct{}", nil
	case emit.Map:
		k, err := g.goRef(r.Key)
		if err != nil {
			return "", err
		}
		v, err := g.goRef(r.Elem)
		if err != nil {
			return "", err
		}
		if !g.comparable(r.Key.ID) {
			return "", emit.Unsupported(r.Pos,
				"a Map key becomes a Go map key, and %s is not a comparable Go type", k)
		}
		return "map[" + k + "]" + v, nil
	case emit.Option, emit.Nullable:
		// Both mean "may be absent", and both become a pointer.
		inner, err := g.goRef(r.Elem)
		if err != nil {
			return "", err
		}
		return "*" + inner, nil
	}
	return g.declName(r.Decl), nil
}

// comparable reports whether the Go type standing for a type reference may
// be a map key.
//
// TDL says a Set holds distinct values and a Map is keyed, and says nothing
// about how a language decides two values are the same. Go does: a map key
// must be comparable, and `map[[]byte]struct{}` is not a weaker guarantee
// but a compile error. So a set or a key whose Go type cannot be one is
// reported as unsupported rather than emitted.
//
// An interface counts as comparable, which is Go's own rule: a sealed enum
// is legal as a key and panics only if a variant holding an incomparable
// field is used as one. Refusing it here would refuse the common case to
// prevent the rare one.
func (g *generator) comparable(id *ir.ID) bool {
	return g.comparableSeen(id, map[int32]bool{})
}

// comparableSeen carries the declarations on the path being walked, since a
// struct may reach itself and a cycle is not an answer.
func (g *generator) comparableSeen(id *ir.ID, seen map[int32]bool) bool {
	t := g.Model.Type(id)
	if t == nil {
		return false
	}
	// A parameter, a unit, and an extern each have no Go type at all, and
	// goType has already refused them by the time this is asked.
	if t.GetParam() != nil || t.GetUnit() != nil || t.GetExtern() != nil {
		return false
	}

	decl := g.Model.Decl(t.GetCtor())
	if decl == nil {
		return false
	}
	name := decl.GetMeta().GetName()

	if a := decl.GetAlias(); a != nil {
		return g.comparableSeen(a.GetTarget(), seen)
	}
	if decl.GetPrimitive() != nil {
		if goName, ok := primitives[name]; ok {
			return !strings.HasPrefix(goName, "[]")
		}
		// A List is a slice and a Set and a Map are maps; none of the three
		// is comparable.
		return false
	}

	// Option and Nullable are a pointer, and a pointer is comparable
	// whatever it points at.
	if decl.GetEnumeration() != nil && (name == "Option" || name == "Nullable") && len(t.GetArgs()) == 1 {
		return true
	}
	if decl.GetEnumeration() != nil {
		// Both enum shapes, a string type and a sealed interface, are legal
		// map keys.
		return true
	}

	// seen is the path being walked and not a memo of what has been
	// answered. A struct with two fields of the same struct type reaches it
	// twice on separate paths, and the second one is not a cycle.
	index := t.GetCtor().GetIndex()
	if seen[index] {
		return false
	}
	seen[index] = true
	defer delete(seen, index)

	if n := decl.GetNewtype(); n != nil {
		return g.comparableSeen(n.GetBase(), seen)
	}
	if decl.GetStructure() != nil {
		for _, f := range decl.Fields() {
			if !g.comparableSeen(f.GetType(), seen) {
				return false
			}
		}
		return true
	}
	// A class and a unit are not Go types, and goType refuses them.
	return false
}

// exported turns a TDL name into an exported Go identifier.
//
// Capitalizing the first rune is the whole rule. An initialism table would
// turn `id` into `ID` and be right most of the time, but a name a consumer
// cannot predict from the model is worse than an unidiomatic one, and
// `name("...")` is how a consumer says what they want instead.
func exported(name string) string {
	// A declaration's name may arrive qualified; the Go identifier is the
	// last segment.
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	if name == "" {
		return ""
	}
	r := []rune(name)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// fileName is the snake case file a declaration is written to.
func fileName(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}

	var b strings.Builder
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String() + ".go"
}

// packageClause is the Go package name for the output.
//
// The `package` directive is written as an import path, because that is
// what a consumer of the generated code writes, and the clause is the last
// segment of it. A TDL package name is dotted and gets the same treatment.
func packageClause(s string) string {
	if i := strings.LastIndexAny(s, "./"); i >= 0 {
		s = s[i+1:]
	}

	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || r == '_':
			b.WriteRune(unicode.ToLower(r))
		case unicode.IsDigit(r) && b.Len() > 0:
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "main"
	}
	return b.String()
}
