package golang

import (
	"fmt"
	"strings"
	"unicode"

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

// unsupportedError reports a type this phase cannot express. It reaches the
// user as a warning rather than stopping the run, so a model that is mostly
// generatable generates.
type unsupportedError struct {
	what     string
	position *ir.Position
}

func (e *unsupportedError) Error() string { return e.what }

func unsupported(pos *ir.Position, format string, args ...any) error {
	return &unsupportedError{what: fmt.Sprintf(format, args...), position: pos}
}

// goType returns the Go type expression for a type reference.
//
// The prelude is replaceable, so this reads the spellings lowering itself
// knows (List, Set, Map, Option, Nullable) rather than anything about what
// the declarations mean. A model that redeclares `primitive string` in its
// own file reaches the same entry, because what matters is that the
// constructor is a primitive named `string`.
func (g *generator) goType(id *ir.ID) (string, error) {
	return g.typeIn(id, nil)
}

// typeIn is [generator.goType] inside a frame, where a parameter stands for
// the argument the frame binds it to rather than for itself.
func (g *generator) typeIn(id *ir.ID, fr *frame) (string, error) {
	t := g.model.Type(id)
	if t == nil {
		return "", unsupported(nil, "type %s did not resolve", id.GetName())
	}
	pos := t.GetPosition()

	switch {
	case t.GetParam() != nil:
		ref := t.GetParam()
		if len(t.GetArgs()) > 0 {
			return "", unsupported(pos, "type parameter %s is applied to type arguments, and Go has no higher-kinded type parameters", ref.GetName())
		}
		if fr == nil {
			// Outside any frame a parameter is the rendered declaration's
			// own, and Go spells it as TDL does.
			return ref.GetName(), nil
		}
		if int(ref.GetIndex()) >= len(fr.args) {
			return "", unsupported(pos, "type parameter %s has no argument", ref.GetName())
		}
		return g.typeIn(fr.args[ref.GetIndex()], fr.outer)
	case t.GetUnit() != nil:
		return "", unsupported(pos, "a unit-typed field has no Go type yet")
	case t.GetExtern() != nil:
		ext := t.GetExtern()
		return "", unsupported(pos, "%s is declared in another package, and foreign types are not generated yet", ext.GetName())
	}

	decl := g.model.Decl(t.GetCtor())
	if decl == nil {
		return "", unsupported(pos, "type %s did not resolve", t.GetCtor().GetName())
	}
	name := decl.GetMeta().GetName()

	// An alias is transparent, so it is expanded rather than referenced,
	// with its arguments standing for its parameters.
	if a := decl.GetAlias(); a != nil {
		if len(t.GetArgs()) != len(a.GetParams()) {
			return "", unsupported(pos, "%s takes %d type argument(s), and %d were given", name, len(a.GetParams()), len(t.GetArgs()))
		}
		return g.typeIn(a.GetTarget(), bind(t.GetArgs(), fr))
	}

	if decl.GetPrimitive() != nil {
		if goName, ok := primitives[name]; ok {
			if strings.HasPrefix(goName, "time.") {
				g.use("time")
			}
			return goName, nil
		}
		if isCollection(name) {
			return g.collection(name, t, fr)
		}
		return "", unsupported(pos, "primitive %s has no Go type", name)
	}

	// Option and Nullable both mean "may be absent", and both become a
	// pointer. SyntacticForm records which spelling was written and is
	// deliberately not read: lowering is authoritative, and generating two
	// shapes would make the sugar semantic after the compiler decided it
	// was not.
	if decl.GetEnumeration() != nil && (name == "Option" || name == "Nullable") && len(t.GetArgs()) == 1 {
		inner, err := g.typeIn(t.GetArgs()[0], fr)
		if err != nil {
			return "", err
		}
		return "*" + inner, nil
	}

	if decl.GetClass() != nil {
		return "", unsupported(pos, "%s is a class, and a class is not a Go type", name)
	}
	if decl.GetUnit() != nil {
		return "", unsupported(pos, "%s is a unit, and units are not generated yet", name)
	}

	if g.skipped[t.GetCtor().GetIndex()] != nil {
		return "", unsupported(pos, "%s is not generated, so nothing generated can name it", name)
	}
	return g.apply(decl, t, fr)
}

// apply names a generated declaration, instantiated with its type arguments
// when it takes parameters.
func (g *generator) apply(decl *ir.Decl, t *ir.Type, fr *frame) (string, error) {
	goName := g.declName(decl)
	// A fieldless enum's parameters are dropped where it is declared, so
	// they are dropped where it is used.
	if phantom(decl) {
		return goName, nil
	}

	name, pos, args := decl.GetMeta().GetName(), t.GetPosition(), t.GetArgs()
	if len(args) != len(decl.Params()) {
		return "", unsupported(pos, "%s takes %d type argument(s), and %d were given", name, len(decl.Params()), len(args))
	}
	if len(args) == 0 {
		return goName, nil
	}

	// The compiler does not check a constraint whose argument is a
	// parameter, and Go does, so every use is checked here and one Go would
	// refuse is not generated.
	flags := g.needsComparable[t.GetCtor().GetIndex()]
	classes := g.paramClasses(decl, false)
	rendered := make([]string, len(args))
	for i, a := range args {
		s, err := g.typeIn(a, fr)
		if err != nil {
			return "", err
		}
		param := decl.Params()[i].GetName()
		if i < len(flags) && flags[i] && !g.comparableIn(a, fr, map[int32]bool{}, nil) {
			return "", unsupported(pos, "%s needs %s to be comparable, and %s is not a comparable Go type", name, param, s)
		}
		if i < len(classes) {
			for _, c := range classes[i] {
				if !g.satisfies(a, fr, c) {
					return "", unsupported(pos, "%s needs %s to satisfy %s, and %s does not",
						name, param, g.model.GetDecls()[c].GetMeta().GetName(), s)
				}
			}
		}
		rendered[i] = s
	}
	return goName + "[" + strings.Join(rendered, ", ") + "]", nil
}

// isCollection reports whether a prelude primitive is one of the three
// collections, which are separate from [primitives] because each reads its
// type arguments.
func isCollection(name string) bool {
	return name == "List" || name == "Set" || name == "Map"
}

// collection maps the three prelude collections.
func (g *generator) collection(name string, t *ir.Type, fr *frame) (string, error) {
	args := t.GetArgs()
	pos := t.GetPosition()

	elem := func(i int) (string, error) {
		if i >= len(args) {
			return "", unsupported(pos, "%s is missing a type argument", name)
		}
		return g.typeIn(args[i], fr)
	}

	switch name {
	case "List":
		e, err := elem(0)
		if err != nil {
			return "", err
		}
		return "[]" + e, nil
	case "Set":
		// Go has no set. A slice would silently permit the duplicates the
		// type exists to forbid, so the key set of a map is the closest
		// thing that keeps the guarantee.
		e, err := elem(0)
		if err != nil {
			return "", err
		}
		if !g.comparableIn(args[0], fr, map[int32]bool{}, nil) {
			return "", unsupported(pos,
				"a Set becomes a Go map, and %s is not a comparable Go type", e)
		}
		return "map[" + e + "]struct{}", nil
	case "Map":
		k, err := elem(0)
		if err != nil {
			return "", err
		}
		v, err := elem(1)
		if err != nil {
			return "", err
		}
		if !g.comparableIn(args[0], fr, map[int32]bool{}, nil) {
			return "", unsupported(pos,
				"a Map key becomes a Go map key, and %s is not a comparable Go type", k)
		}
		return "map[" + k + "]" + v, nil
	}
	return "", unsupported(pos, "primitive %s has no Go type", name)
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
	return g.comparableIn(id, nil, map[int32]bool{}, nil)
}

// comparableIn is [generator.comparable] inside a frame. seen carries the
// declarations on the path being walked, since a struct may reach itself and
// a cycle is not an answer.
//
// A parameter of the declaration being rendered is comparable when Go was
// told it is, which [generator.inferComparable] decided. mark is how that
// inference asks: a parameter reaching this question is reported to it and
// counts as comparable, since saying so is what makes it one.
func (g *generator) comparableIn(id *ir.ID, fr *frame, seen map[int32]bool, mark func(int32)) bool {
	t := g.model.Type(id)
	if t == nil {
		return false
	}
	if ref := t.GetParam(); ref != nil {
		switch {
		case len(t.GetArgs()) > 0:
			return false
		case fr != nil:
			if int(ref.GetIndex()) >= len(fr.args) {
				return false
			}
			return g.comparableIn(fr.args[ref.GetIndex()], fr.outer, seen, mark)
		case mark != nil:
			mark(ref.GetIndex())
			return true
		}
		return g.paramComparable(ref.GetIndex())
	}
	// A unit and an extern have no Go type at all, and goType has already
	// refused them by the time this is asked.
	if t.GetUnit() != nil || t.GetExtern() != nil {
		return false
	}

	decl := g.model.Decl(t.GetCtor())
	if decl == nil {
		return false
	}
	name := decl.GetMeta().GetName()

	if a := decl.GetAlias(); a != nil {
		return g.comparableIn(a.GetTarget(), bind(t.GetArgs(), fr), seen, mark)
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
		return g.comparableIn(n.GetBase(), bind(t.GetArgs(), fr), seen, mark)
	}
	if decl.GetStructure() != nil {
		inner := bind(t.GetArgs(), fr)
		for _, f := range decl.Fields() {
			if !g.comparableIn(f.GetType(), inner, seen, mark) {
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
