package golang

import (
	"strings"
	"unicode"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// primitives maps a prelude primitive to its Go type. uuid, date, and
// decimal are placeholders; docs/design/go-backend.md says why.
var primitives = map[string]string{
	"string":   "string",
	"int":      "int64",
	"int32":    "int32",
	"uint32":   "uint32",
	"int64":    "int64",
	"uint64":   "uint64",
	"float32":  "float32",
	"float64":  "float64",
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
// It matches the prelude by name (a primitive named `string`, List, Option,
// and so on), so a replaced prelude maps the same way. It walks the type
// itself rather than through [emit.Session.Resolve], which refuses type
// parameters.
func (g *generator) goType(id *ir.ID) (string, error) {
	return g.typeIn(id, nil)
}

// typeIn is [generator.goType] inside a frame, where a parameter stands for
// the argument the frame binds it to.
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
			return "", emit.Lost(emit.LossGeneric, pos, "type parameter %s is applied to type arguments, and Go has no higher-kinded type parameters", ref.GetName())
		}
		if fr == nil {
			// Outside any frame a parameter belongs to the rendered
			// declaration.
			return ref.GetName(), nil
		}
		if int(ref.GetIndex()) >= len(fr.args) {
			return "", emit.Unsupported(pos, "type parameter %s has no argument", ref.GetName())
		}
		return g.typeIn(fr.args[ref.GetIndex()], fr.outer)
	case t.GetUnit() != nil:
		return "", emit.Lost(emit.LossUnit, pos, "a unit-typed field has no Go type yet")
	case t.GetExtern() != nil:
		ext := t.GetExtern()
		if f, ok := g.externForeign(t); ok {
			g.useForeign(f)
			args, err := g.typeArgsIn(t, fr)
			if err != nil {
				return "", err
			}
			return f.ref() + args, nil
		}
		return "", emit.Unsupported(pos, "%s is declared in another package, and foreign types are not generated yet", ext.GetName())
	}

	decl := g.Model.Decl(t.GetCtor())
	if decl == nil {
		return "", emit.Unsupported(pos, "type %s did not resolve", t.GetCtor().GetName())
	}
	name := decl.GetMeta().GetName()

	if f, ok := g.foreign[decl]; ok {
		g.useForeign(f)
		args, err := g.typeArgsIn(t, fr)
		if err != nil {
			return "", err
		}
		return f.ref() + args, nil
	}

	// An alias is expanded, its arguments bound to its parameters.
	if a := decl.GetAlias(); a != nil {
		if len(t.GetArgs()) != len(a.GetParams()) {
			return "", emit.Unsupported(pos, "%s takes %d type argument(s), and %d were given", name, len(a.GetParams()), len(t.GetArgs()))
		}
		return g.typeIn(a.GetTarget(), bind(t.GetArgs(), fr))
	}

	if decl.GetPrimitive() != nil {
		if goName, ok := primitives[name]; ok {
			// An argument here is a unit. Visiting it reports it, rather
			// than letting `decimal<kg>` and `decimal<N>` become one type.
			if args := t.GetArgs(); len(args) > 0 {
				if _, err := g.typeIn(args[0], fr); err != nil {
					return "", err
				}
				return "", emit.Lost(emit.LossUnit, pos,
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

	// Option and Nullable both become a pointer. SyntacticForm is
	// deliberately not read: the sugar has no semantic difference.
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
		return "", emit.Lost(emit.LossUnit, pos, "%s is a unit, and units are not generated yet", name)
	}

	return g.apply(decl, t, fr)
}

// apply names a generated declaration, instantiated with its type arguments
// when it takes parameters.
func (g *generator) apply(decl *ir.Decl, t *ir.Type, fr *frame) (string, error) {
	goName := g.declName(decl)
	// A fieldless enum's parameters are dropped at its declaration.
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
	// parameter, and Go does, so every use is checked here.
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

func isCollection(name string) bool {
	return name == "List" || name == "Set" || name == "Map"
}

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
		// A map's key set keeps the no-duplicates guarantee a slice would
		// lose.
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

// comparable reports whether the Go type for a type reference may be a map
// key. An interface counts, as in Go: a sealed enum is a legal key and
// panics only when a variant holding an incomparable field is used as one.
func (g *generator) comparable(id *ir.ID) bool {
	return g.comparableIn(id, nil, map[int32]bool{}, nil)
}

// comparableIn is [generator.comparable] inside a frame. seen holds the
// declarations on the current path, to stop at a cycle.
//
// A parameter of the declaration being rendered is comparable when
// [generator.inferComparable] decided so. That inference passes mark: a
// parameter reaching it is reported and counts as comparable.
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
	// A mapped extern is assumed comparable; goType refuses an unmapped
	// one.
	if t.GetExtern() != nil {
		_, ok := g.externForeign(t)
		return ok
	}
	if t.GetUnit() != nil {
		return false
	}

	decl := g.Model.Decl(t.GetCtor())
	if decl == nil {
		return false
	}
	name := decl.GetMeta().GetName()

	// The consumer's build decides whether a foreign type is a legal key.
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
		// List, Set, and Map are slices and maps.
		return false
	}

	// Option and Nullable are a pointer.
	if decl.GetEnumeration() != nil && (name == "Option" || name == "Nullable") && len(t.GetArgs()) == 1 {
		return true
	}
	if decl.GetEnumeration() != nil {
		// A string type or a sealed interface.
		return true
	}

	// seen is the current path, not a memo: two fields of one struct type
	// reach it twice without a cycle.
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
	return false
}

// exported turns a TDL name into an exported Go identifier by capitalizing
// the first rune of its last segment. There is deliberately no initialism
// table, so the name stays predictable; `name("...")` overrides it.
func exported(name string) string {
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

// reserved holds the file name suffixes go/build gives a meaning: `test`
// and every GOOS and GOARCH in internal/syslist, including ones the current
// toolchain does not build for.
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

// escape is appended to a file name ending in a [reserved] suffix.
const escape = "_tdl"

// fileName is the snake case file a declaration is written to. A name like
// FooTest or OrderLinux gets [escape] appended so go/build neither drops it
// as a test nor constrains it to one platform. FooTestTdl can still collide
// with FooTest's escape; the rule stays simple instead.
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
// meaning. As in go/build, the part before the first underscore is skipped,
// so `linux.go` is an ordinary file.
func readsAsSuffix(name string) bool {
	i := strings.Index(name, "_")
	if i < 0 {
		return false
	}
	parts := strings.Split(name[i+1:], "_")
	return reserved[parts[len(parts)-1]]
}

// packageClause is the Go package name: the last segment of an import path
// or a dotted TDL package name.
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

// typeArgsIn renders a type's arguments as a Go type argument list, or "".
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
