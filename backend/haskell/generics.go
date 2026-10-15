package haskell

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

var (
	kindType    = imported{"Data.Kind", "Type"}
	functor     = imported{"Prelude", "Functor"}
	foldable    = imported{"Prelude", "Foldable"}
	traversable = imported{"Prelude", "Traversable"}
)

// bound is a type reference and the frame its parameters resolve in.
type bound struct {
	id *ir.ID
	fr *frame
}

// frame binds a declaration's parameters to arguments while its body is
// walked. A nil frame is the declaration being rendered, whose parameters
// stay type variables.
type frame struct{ args []bound }

func bounds(ids []*ir.ID, fr *frame) []bound {
	out := make([]bound, len(ids))
	for i, id := range ids {
		out[i] = bound{id, fr}
	}
	return out
}

// application is a type reference with every bound parameter substituted:
// the type it applies, which is an unbound parameter, a declaration, an extern,
// or a unit, and the arguments it applies it to.
type application struct {
	t    *ir.Type
	args []bound
}

// resolve substitutes a reference's parameters. A parameter bound to a
// partial application, as `Collection<Map<string>, int>` binds `f`, takes
// the arguments it is applied to after those it was bound with.
func (g *generator) resolve(b bound) (application, error) {
	t := g.Model.Type(b.id)
	if t == nil {
		return application{}, emit.Unsupported(nil, "type %s did not resolve", b.id.GetName())
	}
	args := bounds(t.GetArgs(), b.fr)
	p := t.GetParam()
	if p == nil || b.fr == nil {
		return application{t, args}, nil
	}
	if int(p.GetIndex()) >= len(b.fr.args) {
		return application{}, emit.Unsupported(t.GetPosition(), "type parameter %s is not bound", p.GetName())
	}
	a, err := g.resolve(b.fr.args[p.GetIndex()])
	if err != nil {
		return application{}, err
	}
	a.args = append(slices.Clip(a.args), args...)
	return a, nil
}

// tyvar is a type parameter's Haskell type variable.
func tyvar(name string) string { return lowerFirst(name) }

// params is a declaration's type variables as its head writes them, each
// after a space, with a kind signature on a parameter whose kind is
// declared and is not `type`. It is an error when a parameter cannot be a
// type variable.
func (g *generator) params(d *ir.Decl) (string, error) {
	var b strings.Builder
	seen := map[string]string{}
	for _, p := range d.Params() {
		pos := p.GetPosition()
		if pos == nil {
			pos = d.GetMeta().GetPosition()
		}
		name, of := p.GetName(), d.GetMeta().GetName()
		v := tyvar(name)
		switch other, dup := seen[v]; {
		case unitKinded(p.GetKind()):
			return "", emit.Lost(emit.LossUnit, pos, "type parameter %s of %s is a unit, and units are not generated yet", name, of)
		case keywords[v]:
			return "", emit.Lost(emit.LossGeneric, pos, "type parameter %s of %s would be the type variable %s, which is a Haskell keyword", name, of, v)
		case !varid.MatchString(v):
			return "", emit.Lost(emit.LossGeneric, pos, "type parameter %s of %s would be the type variable %s, which Haskell refuses", name, of, v)
		case dup:
			return "", emit.Lost(emit.LossGeneric, pos, "type parameters %s and %s of %s are both the type variable %s", other, name, of, v)
		}
		seen[v] = name

		k := p.GetKind()
		if k == nil || k.GetArrow() == nil && k.GetParen() == nil && k.GetAtom() == ir.KindAtom_KIND_ATOM_TYPE {
			b.WriteString(" " + v)
			continue
		}
		fmt.Fprintf(&b, " (%s :: %s)", v, g.kind(k))
	}
	return b.String(), nil
}

// kind renders a kind, which holds no unit.
func (g *generator) kind(k *ir.Kind) string {
	base := g.use(kindType)
	if k.GetParen() != nil {
		base = "(" + g.kind(k.GetParen()) + ")"
	}
	if k.GetArrow() != nil {
		return base + " -> " + g.kind(k.GetArrow())
	}
	return base
}

func unitKinded(k *ir.Kind) bool {
	if k == nil {
		return false
	}
	return k.GetAtom() == ir.KindAtom_KIND_ATOM_UNIT || unitKinded(k.GetParen()) || unitKinded(k.GetArrow())
}

// applyTo is a type applied to the declaration's own type variables, as an
// instance head writes it.
func applyTo(typ string, d *ir.Decl) hsType {
	if len(d.Params()) == 0 {
		return hsType{text: typ, atom: true}
	}
	text := typ
	for _, p := range d.Params() {
		text += " " + tyvar(p.GetName())
	}
	return hsType{text: text}
}

// bodyTypes is every type a declaration's values hold.
func bodyTypes(d *ir.Decl) []*ir.ID {
	if n := d.GetNewtype(); n != nil {
		return []*ir.ID{n.GetBase()}
	}
	var out []*ir.ID
	for _, f := range d.Fields() {
		out = append(out, f.GetType())
	}
	for _, v := range d.GetEnumeration().GetVariants() {
		for _, f := range v.GetFields() {
			out = append(out, f.GetType())
		}
	}
	return out
}

// generated reports whether a reference to a declaration names a type this
// backend declares, rather than one it maps or expands.
func (g *generator) generated(d *ir.Decl) bool {
	return emit.IsOwn(d) && g.foreign[d] == (imported{}) &&
		(d.GetStructure() != nil || d.GetEnumeration() != nil || d.GetNewtype() != nil)
}

// planFunctors decides, across the whole model, which declarations derive
// Functor, Foldable, and Traversable over their last parameter: those
// holding it only directly, in a list, a Maybe, a map's value, or as the
// last argument of another declaration deriving them. It starts from every
// candidate and drops one that fails until a pass drops nothing, so a
// recursive declaration derives them.
func (g *generator) planFunctors(own []*ir.Decl) {
	g.functors = map[*ir.Decl]bool{}
	for _, d := range own {
		ps := d.Params()
		if len(ps) == 0 || !g.generated(d) {
			continue
		}
		if k := ps[len(ps)-1].GetKind(); k != nil && (k.GetArrow() != nil || k.GetParen() != nil || k.GetAtom() != ir.KindAtom_KIND_ATOM_TYPE) {
			continue
		}
		g.functors[d] = true
	}
	for again := true; again; {
		again = false
		for d := range g.functors {
			last := int32(len(d.Params()) - 1)
			for _, id := range bodyTypes(d) {
				if !g.maps(bound{id, nil}, last, 0) {
					delete(g.functors, d)
					again = true
					break
				}
			}
		}
	}
}

// maxDepth bounds a walk through declarations' bodies, which a
// declaration recursing at a growing type would never end.
const maxDepth = 32

// maps reports whether Functor can map the parameter at index last of the
// declaration being rendered wherever b holds it.
func (g *generator) maps(b bound, last int32, depth int) bool {
	if depth > maxDepth {
		return false
	}
	a, err := g.resolve(b)
	if err != nil {
		return false
	}
	if p := a.t.GetParam(); p != nil {
		if p.GetIndex() == last {
			return len(a.args) == 0
		}
		return !g.mentions(a.args, last)
	}
	decl := g.Model.Decl(a.t.GetCtor())
	if decl != nil && decl.GetAlias() != nil && g.foreign[decl] == (imported{}) {
		n := len(decl.GetAlias().GetParams())
		if len(a.args) < n {
			return false
		}
		return g.maps(bound{decl.GetAlias().GetTarget(), &frame{a.args[:n]}}, last, depth+1) && !g.mentions(a.args[n:], last)
	}
	if !g.mentions(a.args, last) {
		return true
	}
	switch {
	case decl == nil || g.foreign[decl] != (imported{}):
		return false
	case decl.GetPrimitive() != nil && decl.GetMeta().GetName() == "List", optional(decl):
		return len(a.args) == 1 && g.maps(a.args[0], last, depth+1)
	case decl.GetPrimitive() != nil && decl.GetMeta().GetName() == "Map":
		return len(a.args) == 2 && !g.mentions(a.args[:1], last) && g.maps(a.args[1], last, depth+1)
	case g.functors[decl]:
		n := len(a.args)
		return n == len(decl.Params()) && !g.mentions(a.args[:n-1], last) && g.maps(a.args[n-1], last, depth+1)
	}
	return false
}

// mentions reports whether any of args holds the parameter at index last
// of the declaration being rendered.
func (g *generator) mentions(args []bound, last int32) bool {
	for _, arg := range args {
		a, err := g.resolve(arg)
		if err != nil {
			return true
		}
		if p := a.t.GetParam(); p != nil && p.GetIndex() == last {
			return true
		}
		if g.mentions(a.args, last) {
			return true
		}
	}
	return false
}

// optional reports whether a declaration is the prelude's Option or
// Nullable, which are both Maybe.
func optional(d *ir.Decl) bool {
	name := d.GetMeta().GetName()
	return d.GetEnumeration() != nil && !emit.IsOwn(d) && (name == "Option" || name == "Nullable")
}

// constraint is one constraint an instance context places on a type.
type constraint struct {
	// arg is the constrained type, as an argument to the class.
	arg string
	// simple is whether arg is a type variable, which stock deriving
	// infers a constraint on.
	simple bool
	// size counts arg's constructors and variables, and vars each
	// variable's occurrences, for the Paterson conditions.
	size int
	vars map[string]int
}

// context is what an instance of Eq, Ord, or Show for a declaration
// requires of its type variables: the constraint each type it holds
// reduces to through the types it is built from. A declaration holding an
// applied type variable, `f t`, needs one GHC does not infer.
func (g *generator) context(d *ir.Decl) ([]constraint, error) {
	var out []constraint
	seen := map[string]bool{}
	visited := map[string]bool{}
	for _, id := range bodyTypes(d) {
		cs, err := g.reduce(bound{id, nil}, visited, 0)
		if err != nil {
			return nil, err
		}
		for _, c := range cs {
			if !seen[c.arg] {
				seen[c.arg] = true
				out = append(out, c)
			}
		}
	}
	return out, nil
}

// reduce is the constraints a class's instance for b needs, assuming the
// instances every collection, primitive, and mapped type has: one on each
// type argument.
func (g *generator) reduce(b bound, visited map[string]bool, depth int) ([]constraint, error) {
	if depth > maxDepth {
		return nil, emit.Unsupported(g.Model.Type(b.id).GetPosition(), "a declaration it holds recurses at a growing type, and its instances cannot be derived")
	}
	a, err := g.resolve(b)
	if err != nil {
		return nil, err
	}
	all := func(args []bound) ([]constraint, error) {
		var out []constraint
		for _, arg := range args {
			cs, err := g.reduce(arg, visited, depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, cs...)
		}
		return out, nil
	}

	if p := a.t.GetParam(); p != nil {
		if len(a.args) == 0 {
			v := tyvar(p.GetName())
			return []constraint{{arg: v, simple: true, size: 1, vars: map[string]int{v: 1}}}, nil
		}
		t, err := g.render(b)
		if err != nil {
			return nil, err
		}
		size, vars := g.measure(b, depth)
		return []constraint{{arg: t.arg(), size: size, vars: vars}}, nil
	}
	decl := g.Model.Decl(a.t.GetCtor())
	switch {
	case decl == nil || g.foreign[decl] != (imported{}):
		return all(a.args)
	case decl.GetAlias() != nil:
		n := len(decl.GetAlias().GetParams())
		if len(a.args) < n {
			return nil, nil
		}
		return g.reduce(bound{decl.GetAlias().GetTarget(), &frame{a.args[:n]}}, visited, depth+1)
	case g.generated(decl):
		sig := g.signature(b)
		if visited[sig] {
			return nil, nil
		}
		visited[sig] = true
		var out []constraint
		for _, id := range bodyTypes(decl) {
			cs, err := g.reduce(bound{id, &frame{a.args}}, visited, depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, cs...)
		}
		return out, nil
	}
	return all(a.args)
}

// measure is a type's size and its variables' occurrences, with synonyms
// expanded, as GHC counts them.
func (g *generator) measure(b bound, depth int) (int, map[string]int) {
	vars := map[string]int{}
	var walk func(bound, int) int
	walk = func(b bound, depth int) int {
		a, err := g.resolve(b)
		if err != nil || depth > maxDepth {
			return 1
		}
		if decl := g.Model.Decl(a.t.GetCtor()); decl != nil && decl.GetAlias() != nil && g.foreign[decl] == (imported{}) {
			if n := len(decl.GetAlias().GetParams()); len(a.args) >= n {
				size := walk(bound{decl.GetAlias().GetTarget(), &frame{a.args[:n]}}, depth+1)
				for _, arg := range a.args[n:] {
					size += walk(arg, depth+1)
				}
				return size
			}
		}
		if p := a.t.GetParam(); p != nil {
			vars[tyvar(p.GetName())]++
		}
		size := 1
		for _, arg := range a.args {
			size += walk(arg, depth+1)
		}
		return size
	}
	return walk(b, depth), vars
}

// signature identifies a type with its parameters substituted.
func (g *generator) signature(b bound) string {
	a, err := g.resolve(b)
	if err != nil {
		return "?"
	}
	var s string
	switch t := a.t; {
	case t.GetParam() != nil:
		s = "p" + t.GetParam().GetName()
	case t.GetExtern() != nil:
		s = "e" + strconv.Itoa(int(t.GetExtern().GetIndex()))
	case t.GetUnit() != nil:
		s = "u" + strconv.Itoa(int(t.GetUnit().GetIndex()))
	default:
		s = "d" + strconv.Itoa(int(t.GetCtor().GetIndex()))
	}
	if len(a.args) == 0 {
		return s
	}
	parts := make([]string, len(a.args))
	for i, arg := range a.args {
		parts[i] = g.signature(arg)
	}
	return s + "(" + strings.Join(parts, ",") + ")"
}

// derivings writes a declaration's instances: the stock classes given and
// Generic, then Functor, Foldable, and Traversable where they map its last
// parameter, then the classes `derive` adds. When the declaration holds a
// type GHC infers no context for, `f t`, each class but Generic and the
// Functor family is derived standalone with its context written out.
func (g *generator) derivings(b *strings.Builder, d *ir.Decl, typ string, classes ...imported) error {
	ctx, err := g.context(d)
	if err != nil {
		return err
	}
	var fft []imported
	if g.functors[d] {
		fft = []imported{functor, foldable, traversable}
	}
	derived := g.derived(d)
	if !slices.ContainsFunc(ctx, func(c constraint) bool { return !c.simple }) {
		b.WriteString(g.deriving(append(append(classes, generic), fft...)...))
		b.WriteString(g.anyclass(derived))
		return nil
	}

	b.WriteString(g.deriving(append([]imported{generic}, fft...)...))
	head := applyTo(typ, d)
	size := 1 + len(d.Params())
	for _, c := range ctx {
		bigger := c.size >= size
		for _, n := range c.vars {
			bigger = bigger || n > 1
		}
		if bigger {
			g.exts["UndecidableInstances"] = true
		}
	}
	standalone := func(strategy string, c imported) string {
		name := g.use(c)
		parts := make([]string, len(ctx))
		for i, k := range ctx {
			parts[i] = name + " " + k.arg
		}
		return fmt.Sprintf("deriving %s instance (%s) => %s %s\n", strategy, strings.Join(parts, ", "), name, head.arg())
	}
	b.WriteString("\n")
	for _, c := range classes {
		b.WriteString(standalone("stock", c))
	}
	if len(derived) > 0 {
		g.exts["DeriveAnyClass"] = true
		for _, c := range derived {
			b.WriteString(standalone("anyclass", c))
		}
	}
	return nil
}
