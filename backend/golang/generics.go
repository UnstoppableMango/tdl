package golang

import (
	"fmt"
	"go/token"
	"go/types"
	"slices"
	"strings"
	"unicode"

	"github.com/unstoppablemango/tdl/ir"
)

// frame binds a parameterized declaration's parameters to the arguments it
// was applied to, while its body is walked: an alias being expanded, or a
// struct or newtype being checked for comparability.
//
// An argument is written in the context that applied it, which is the frame
// outside this one, or the declaration being rendered when there is none.
type frame struct {
	args  []*ir.ID
	outer *frame
}

// bind is the frame a declaration's body is walked in when it is applied to
// args, and nil when it takes none, since its body then names nothing from
// outside.
func bind(args []*ir.ID, outer *frame) *frame {
	if len(args) == 0 {
		return nil
	}
	return &frame{args: args, outer: outer}
}

// typeParams is the Go type parameter list of the declaration being
// rendered, "[K comparable, V any]", or "" when it takes none.
func (g *generator) typeParams(decl *ir.Decl) string {
	names := paramNames(decl)
	if len(names) == 0 || phantom(decl) {
		return ""
	}
	parts := make([]string, len(names))
	for i, n := range names {
		parts[i] = n + " " + g.constraint(int32(i))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// typeArgs is how the declaration's own methods name it, "[K, V]".
func typeArgs(decl *ir.Decl) string {
	names := paramNames(decl)
	if len(names) == 0 || phantom(decl) {
		return ""
	}
	return "[" + strings.Join(names, ", ") + "]"
}

func paramNames(decl *ir.Decl) []string {
	var names []string
	for _, p := range decl.Params() {
		names = append(names, p.GetName())
	}
	return names
}

// constraint is the Go constraint on a parameter of the declaration being
// rendered: the classes its `requires` clause names, and comparable when Go
// needs it to be.
func (g *generator) constraint(i int32) string {
	var names []string
	if int(i) < len(g.curClasses) {
		for _, c := range g.curClasses[i] {
			names = append(names, g.declName(g.model.GetDecls()[c]))
		}
	}
	comparable := g.paramComparable(i)
	switch {
	case len(names) == 0 && comparable:
		return "comparable"
	case len(names) == 0:
		return "any"
	case len(names) == 1 && !comparable:
		return names[0]
	}
	if comparable {
		names = append([]string{"comparable"}, names...)
	}
	return "interface{ " + strings.Join(names, "; ") + " }"
}

// paramComparable reports whether Go was told a parameter of the declaration
// being rendered is comparable.
func (g *generator) paramComparable(i int32) bool {
	flags := g.needsComparable[g.cur]
	return int(i) < len(flags) && flags[i]
}

// carries reports whether any variant of an enum carries fields, which is
// what decides its Go shape.
func carries(e *ir.Enum) bool {
	for _, v := range e.GetVariants() {
		if len(v.GetFields()) > 0 {
			return true
		}
	}
	return false
}

// phantom reports whether a declaration's parameters are dropped: a fieldless
// enum is constants, a Go constant cannot be generic, and nothing in it names
// a parameter.
func phantom(decl *ir.Decl) bool {
	e := decl.GetEnumeration()
	return e != nil && len(e.GetParams()) > 0 && !carries(e)
}

// paramProblem says why a declaration's type parameters cannot be Go type
// parameters, or returns nil when they can.
func (g *generator) paramProblem(decl *ir.Decl) error {
	for _, p := range decl.Params() {
		pos := p.GetPosition()
		if pos == nil {
			pos = decl.GetMeta().GetPosition()
		}
		name, of := p.GetName(), decl.GetMeta().GetName()
		switch {
		case higherKinded(p.GetKind()):
			return unsupported(pos, "type parameter %s of %s takes type arguments, and Go has no higher-kinded type parameters", name, of)
		case unitKinded(p.GetKind()):
			return unsupported(pos, "type parameter %s of %s is a unit, and units are not generated yet", name, of)
		case token.IsKeyword(name):
			return unsupported(pos, "type parameter %s of %s is a Go keyword", name, of)
		// A parameter would shadow what the generated code names, and TDL
		// never saw the clash: `int` is Go's int64, so a parameter named
		// int64 is legal TDL and captures the field typed `int`.
		case types.Universe.Lookup(name) != nil, name == "time":
			return unsupported(pos, "type parameter %s of %s would shadow Go's %s", name, of, name)
		case g.declares(name):
			return unsupported(pos, "type parameter %s of %s would shadow the generated declaration %s", name, of, name)
		}
	}
	return nil
}

func higherKinded(k *ir.Kind) bool {
	return k != nil && (k.GetArrow() != nil || higherKinded(k.GetParen()))
}

func unitKinded(k *ir.Kind) bool {
	if k == nil || k.GetArrow() != nil {
		return false
	}
	return k.GetAtom() == ir.KindAtom_KIND_ATOM_UNIT || unitKinded(k.GetParen())
}

// declares reports whether the package declares a type with this Go name.
func (g *generator) declares(goName string) bool {
	for _, d := range g.own() {
		switch d.GetNode().(type) {
		case *ir.Decl_Structure, *ir.Decl_Enumeration, *ir.Decl_Newtype, *ir.Decl_Class:
			if g.declName(d) == goName {
				return true
			}
		}
	}
	return false
}

// receiver names a method's receiver after its type, unless a type parameter
// is already spelled that way: the two share a scope, and Go refuses the
// redeclaration.
func receiver(decl *ir.Decl, goName string) string {
	taken := paramNames(decl)
	if r := string(unicode.ToLower([]rune(goName)[0])); !slices.Contains(taken, r) {
		return r
	}
	for i := 0; ; i++ {
		r := "r"
		if i > 0 {
			r = fmt.Sprintf("r%d", i)
		}
		if !slices.Contains(taken, r) {
			return r
		}
	}
}

// inferComparable decides which type parameters Go needs to be comparable:
// one reaching a set element, a map key, or a key field, directly or by
// being handed to a parameter that already has to be.
//
// TDL has no way to say a parameter is comparable, so this is the only way
// `Bag<T> { items: {T} }` compiles. A declaration may hand its parameter to
// one later in the table, so this repeats until a pass marks nothing new.
func (g *generator) inferComparable() {
	g.needsComparable = map[int32][]bool{}
	decls := g.model.GetDecls()
	for i, d := range decls {
		if isOwn(d) && len(d.Params()) > 0 {
			g.needsComparable[int32(i)] = make([]bool, len(d.Params()))
		}
	}

	for again := true; again; {
		again = false
		for i, d := range decls {
			flags := g.needsComparable[int32(i)]
			if flags == nil {
				continue
			}
			mark := func(p int32) {
				if int(p) < len(flags) && !flags[p] {
					flags[p] = true
					again = true
				}
			}
			for _, id := range bodyTypes(d) {
				g.positions(id, nil, mark)
			}
			for _, id := range g.keyTypes(d) {
				g.comparableIn(id, nil, map[int32]bool{}, mark)
			}
		}
	}
}

// positions walks a type reference for the places Go compares a type, and
// asks comparableIn about what reaches one, which marks the parameters that
// do.
func (g *generator) positions(id *ir.ID, fr *frame, mark func(int32)) {
	t := g.model.Type(id)
	if t == nil {
		return
	}
	if ref := t.GetParam(); ref != nil {
		// A parameter bound by an alias being expanded stands for its
		// argument, which is walked where it was written.
		if fr != nil && int(ref.GetIndex()) < len(fr.args) {
			g.positions(fr.args[ref.GetIndex()], fr.outer, mark)
		}
		return
	}

	decl := g.model.Decl(t.GetCtor())
	if decl == nil {
		return
	}
	args := t.GetArgs()
	if a := decl.GetAlias(); a != nil {
		g.positions(a.GetTarget(), bind(args, fr), mark)
		return
	}

	name := decl.GetMeta().GetName()
	switch {
	case decl.GetPrimitive() != nil && (name == "Set" || name == "Map"):
		if len(args) > 0 {
			g.comparableIn(args[0], fr, map[int32]bool{}, mark)
		}
	case !phantom(decl):
		flags := g.needsComparable[t.GetCtor().GetIndex()]
		for i, a := range args {
			if i < len(flags) && flags[i] {
				g.comparableIn(a, fr, map[int32]bool{}, mark)
			}
		}
	}
	for _, a := range args {
		g.positions(a, fr, mark)
	}
}

// bodyTypes is every type a declaration's body names directly.
func bodyTypes(d *ir.Decl) []*ir.ID {
	var out []*ir.ID
	for _, f := range d.Fields() {
		out = append(out, f.GetType())
	}
	for _, v := range d.GetEnumeration().GetVariants() {
		for _, f := range v.GetFields() {
			out = append(out, f.GetType())
		}
	}
	if n := d.GetNewtype(); n != nil {
		out = append(out, n.GetBase())
	}
	return out
}

// keyTypes is the type of every field a `key` directive names.
func (g *generator) keyTypes(d *ir.Decl) []*ir.ID {
	dir, ok := g.find(d.GetDirectives(), "key")
	if !ok {
		return nil
	}
	var out []*ir.ID
	for _, arg := range dir.GetArgs() {
		for _, f := range d.Fields() {
			if f.GetMeta().GetName() == arg.GetText() {
				out = append(out, f.GetType())
			}
		}
	}
	return out
}
