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
//
// The prelude is replaceable, so this reads the spellings lowering itself
// knows (List, Set, Map, Option, Nullable) rather than anything about what
// the declarations mean. A model that redeclares `primitive string` in its
// own file reaches the same entry, because what matters is that the
// constructor is a primitive named `string`.
//
// This walk is the backend's own rather than [emit.Session.Resolve], which
// refuses a type parameter and a type argument outright. Go is the one
// target here with generics, so a parameter is a type it emits and not a
// type it skips, and the frame below is what substitutes one. Unifying the
// two belongs to whenever a second backend emits generics.
func (g *generator) goType(id *ir.ID) (string, error) {
	return g.typeIn(id, nil)
}

// typeIn is [generator.goType] inside a frame, where a parameter stands for
// the argument the frame binds it to rather than for itself.
func (g *generator) typeIn(id *ir.ID, fr *frame) (string, error) {
	t := g.Model.Type(id)
	if t == nil {
		return "", emit.Unsupported(nil, "type %s did not resolve", id.GetName())
	}
	pos := t.GetPosition()

	switch {
	case t.GetParam() != nil:
		ref := t.GetParam()
		if len(t.GetArgs()) > 0 {
			return "", emit.Unsupported(pos, "type parameter %s is applied to type arguments, and Go has no higher-kinded type parameters", ref.GetName())
		}
		if fr == nil {
			// Outside any frame a parameter is the rendered declaration's
			// own, and Go spells it as TDL does.
			return ref.GetName(), nil
		}
		if int(ref.GetIndex()) >= len(fr.args) {
			return "", emit.Unsupported(pos, "type parameter %s has no argument", ref.GetName())
		}
		return g.typeIn(fr.args[ref.GetIndex()], fr.outer)
	case t.GetUnit() != nil:
		return "", emit.Unsupported(pos, "a unit-typed field has no Go type yet")
	case t.GetExtern() != nil:
		ext := t.GetExtern()
		return "", emit.Unsupported(pos, "%s is declared in another package, and foreign types are not generated yet", ext.GetName())
	}

	decl := g.Model.Decl(t.GetCtor())
	if decl == nil {
		return "", emit.Unsupported(pos, "type %s did not resolve", t.GetCtor().GetName())
	}
	name := decl.GetMeta().GetName()

	// A foreign declaration is a type another package declares, so it is
	// imported and referred to rather than expanded or looked up here.
	if f, ok := g.foreign[decl]; ok {
		g.useAs(f.path, f.alias)
		args, err := g.typeArgsIn(t, fr)
		if err != nil {
			return "", err
		}
		return f.ref() + args, nil
	}

	// An alias is transparent, so it is expanded rather than referenced,
	// with its arguments standing for its parameters.
	if a := decl.GetAlias(); a != nil {
		if len(t.GetArgs()) != len(a.GetParams()) {
			return "", emit.Unsupported(pos, "%s takes %d type argument(s), and %d were given", name, len(a.GetParams()), len(t.GetArgs()))
		}
		return g.typeIn(a.GetTarget(), bind(t.GetArgs(), fr))
	}

	if decl.GetPrimitive() != nil {
		if goName, ok := primitives[name]; ok {
			// A primitive in the table takes no type parameters, so an
			// argument on one is a unit saying what the number measures.
			// Returning the Go type without reading it would make
			// `decimal<kg>` and `decimal<N>` the same type, so the
			// argument is visited, which is what reports it.
			if args := t.GetArgs(); len(args) > 0 {
				if _, err := g.typeIn(args[0], fr); err != nil {
					return "", err
				}
				return "", emit.Unsupported(pos,
					"%s is applied to a type argument, and Go has no type carrying one", name)
			}
			if strings.HasPrefix(goName, "time.") {
				g.use("time")
			}
			return goName, nil
		}
		if isCollection(name) {
			return g.collection(name, t, fr)
		}
		return "", emit.Unsupported(pos, "primitive %s has no Go type", name)
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
		return "", emit.Unsupported(pos, "%s is a class, and a class is not a Go type", name)
	}
	if decl.GetUnit() != nil {
		return "", emit.Unsupported(pos, "%s is a unit, and units are not generated yet", name)
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
		return "", emit.Unsupported(pos, "%s takes %d type argument(s), and %d were given", name, len(decl.Params()), len(args))
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
			return "", emit.Unsupported(pos, "%s needs %s to be comparable, and %s is not a comparable Go type", name, param, s)
		}
		if i < len(classes) {
			for _, c := range classes[i] {
				if !g.satisfies(a, fr, c) {
					return "", emit.Unsupported(pos, "%s needs %s to satisfy %s, and %s does not",
						name, param, g.Model.GetDecls()[c].GetMeta().GetName(), s)
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
			return "", emit.Unsupported(pos, "%s is missing a type argument", name)
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
			return "", emit.Unsupported(pos,
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
			return "", emit.Unsupported(pos,
				"a Map key becomes a Go map key, and %s is not a comparable Go type", k)
		}
		return "map[" + k + "]" + v, nil
	}
	return "", emit.Unsupported(pos, "primitive %s has no Go type", name)
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
	t := g.Model.Type(id)
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

	decl := g.Model.Decl(t.GetCtor())
	if decl == nil {
		return false
	}
	name := decl.GetMeta().GetName()

	// Whether a foreign type is a legal map key is decided by the package
	// that declares it, at the consumer's build. Refusing it here would
	// refuse a Set of a mapped uuid, which is the mapping working.
	if g.isForeign(decl) {
		return true
	}

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

// reserved is what go/build reads the last underscore-separated element of
// a file name as: the GOOS and GOARCH lists in internal/syslist, which name
// every past, present, and future target so a name is never reused, plus
// `test`, which leaves a file out of the package the rest of the time.
//
// The whole of both lists is here rather than the values this toolchain
// builds for, because a file name means the same thing to every toolchain
// that reads it and a generated package is read by more than one.
var reserved = map[string]bool{
	"test": true,

	"aix": true, "android": true, "darwin": true, "dragonfly": true,
	"freebsd": true, "hurd": true, "illumos": true, "ios": true,
	"js": true, "linux": true, "nacl": true, "netbsd": true,
	"openbsd": true, "plan9": true, "solaris": true, "wasip1": true,
	"windows": true, "zos": true,

	"386": true, "amd64": true, "amd64p32": true, "arm": true,
	"armbe": true, "arm64": true, "arm64be": true, "loong64": true,
	"mips": true, "mipsle": true, "mips64": true, "mips64le": true,
	"mips64p32": true, "mips64p32le": true, "ppc": true, "ppc64": true,
	"ppc64le": true, "riscv": true, "riscv64": true, "s390": true,
	"s390x": true, "sparc": true, "sparc64": true, "wasm": true,
}

// escape is appended to a file name Go would read something into, and is a
// name no GOOS or GOARCH will take, since the lists are Go's to extend.
const escape = "_tdl"

// fileName is the snake case file a declaration is written to.
//
// Go reads a meaning into how a file name ends: `foo_test.go` is a test
// file and `go build` leaves it out of the package, and `order_linux.go` or
// `report_arm64.go` build only on that OS or architecture. A declaration
// named FooTest or OrderLinux is neither of those things, so a name that
// would end that way gets [escape] appended and lands in the package on
// every platform.
//
// Two declarations can still collide, since FooTestTdl already spells what
// FooTest escapes to. A name deliberately spelled as another one's escape
// is worth less than the rule staying one a reader can predict.
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

	s := b.String()
	if readsAsSuffix(s) {
		s += escape
	}
	return s + ".go"
}

// readsAsSuffix reports whether go/build gives the end of a file name a
// meaning.
//
// Everything up to the first underscore is skipped the way go/build skips
// it: a prefix is required, so `linux.go` is an ordinary file and only
// `foo_linux.go` carries the constraint. The `_GOOS_GOARCH` pair form needs
// no case of its own, because its last element is a GOARCH.
func readsAsSuffix(name string) bool {
	i := strings.Index(name, "_")
	if i < 0 {
		return false
	}
	parts := strings.Split(name[i+1:], "_")
	return reserved[parts[len(parts)-1]]
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

// typeArgsIn renders a type's arguments as a Go type argument list, and ""
// for a type applied to none.
func (g *generator) typeArgsIn(t *ir.Type, fr *frame) (string, error) {
	if len(t.GetArgs()) == 0 {
		return "", nil
	}
	rendered := make([]string, len(t.GetArgs()))
	for i, a := range t.GetArgs() {
		s, err := g.typeIn(a, fr)
		if err != nil {
			return "", err
		}
		rendered[i] = s
	}
	return "[" + strings.Join(rendered, ", ") + "]", nil
}
