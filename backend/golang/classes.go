package golang

import (
	"fmt"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// A class is a Go interface whose one method is unexported, and each
// declaration satisfying the class carries that method. Conformance in TDL is
// nominal and always declared, and an unexported method is what makes it so
// in Go: nothing outside the package can implement the interface, and nothing
// inside implements it without being generated to.
//
// A class's fields are not methods of the interface. The declarations
// satisfying it already declare them, Go refuses a field and a method with
// one name, and a getter under another name is API the model never asked
// for. Generated code never calls into a type argument's values, so a
// constraint's only job is to say which types may be arguments, and the
// marker says exactly that.

// planClasses decides which classes are generated and which declarations
// carry each marker, before anything is rendered: a use of a constrained
// declaration anywhere in the table asks whether its argument carries one.
func (g *generator) planClasses() {
	g.genClass = map[int32]bool{}
	g.marks = map[int32][]int32{}
	decls := g.Model.GetDecls()
	for i, d := range decls {
		if c := d.GetClass(); c != nil && emit.IsOwn(d) && len(c.GetParams()) == 0 && g.declName(d) != "" {
			g.genClass[int32(i)] = true
		}
	}

	// A class reaching itself is dropped before anything asks what it
	// requires: a Go interface cannot embed itself, directly or through
	// another, and the rest of the plan reads the classes that remain.
	var cyclic []int32
	for i := range decls {
		if g.genClass[int32(i)] && g.requiresCycle(int32(i)) {
			cyclic = append(cyclic, int32(i))
		}
	}
	g.classCycle = map[int32]bool{}
	for _, i := range cyclic {
		g.classCycle[i] = true
		delete(g.genClass, i)
	}

	for i, class := range decls {
		if !g.genClass[int32(i)] {
			continue
		}
		// A class the interface cannot embed is dropped by
		// [generator.supers], so it is reported once here rather than at
		// each of that function's callers.
		for _, ref := range class.GetClass().GetRequiresClasses() {
			if err := g.superProblem(class, ref); err != nil {
				g.Warn(err)
			}
		}
		for _, id := range g.Model.Satisfying(&ir.ID{Index: int32(i)}) {
			target := g.Model.Decl(id)
			if target == nil || !emit.IsOwn(target) {
				continue
			}
			if err := g.markerProblem(target, class); err != nil {
				g.Warn(err)
				continue
			}
			g.marks[id.GetIndex()] = append(g.marks[id.GetIndex()], int32(i))
		}
	}

	for _, inst := range g.Model.GetInstances() {
		if err := g.instanceProblem(inst); err != nil {
			g.Warn(err)
		}
	}
}

// markerProblem says why a declaration satisfying a class cannot carry its
// marker, or returns nil when it can.
func (g *generator) markerProblem(target, class *ir.Decl) error {
	pos := target.GetMeta().GetPosition()
	name, cname := target.GetMeta().GetName(), class.GetMeta().GetName()
	marker := "is" + g.declName(class)
	switch {
	case g.isForeign(target):
		return emit.Unsupported(pos, "%s satisfies %s, and it is a foreign type, which this package cannot add %s's method to", name, cname, cname)
	case target.GetStructure() == nil && target.GetEnumeration() == nil && target.GetNewtype() == nil:
		return emit.Unsupported(pos, "%s satisfies %s, and it declares no Go type to carry %s's method", name, cname, cname)
	case target.GetNewtype() != nil && g.pointerOrInterface(target.GetNewtype().GetBase(), map[int32]bool{}):
		return emit.Unsupported(pos, "%s satisfies %s, and it is a newtype over a pointer or an interface, which Go cannot add a method to", name, cname)
	case g.hasField(target, marker):
		return emit.Unsupported(pos, "%s satisfies %s, and its field %s would collide with the method marking it", name, cname, marker)
	}
	return nil
}

// instanceProblem says why an instance of a generated class does not become
// a marker, or returns nil when it does or is not this backend's to say.
func (g *generator) instanceProblem(inst *ir.Instance) error {
	ref := inst.GetClass()
	if ref.GetExtern() != nil || !g.genClass[ref.GetClass().GetIndex()] {
		return nil
	}
	pos := inst.GetMeta().GetPosition()
	cname := g.Model.Decl(ref.GetClass()).GetMeta().GetName()

	if len(inst.GetParams()) > 0 || len(inst.GetRequires()) > 0 {
		return emit.Unsupported(pos, "a conditional instance of %s holds for some type arguments, and Go cannot give a method to only some instantiations", cname)
	}
	if len(ref.GetArgs()) != 1 {
		return nil
	}
	t := g.Model.Type(ref.GetArgs()[0])
	if ext := t.GetExtern(); ext != nil {
		return emit.Unsupported(pos, "%s is declared in another package, and Go cannot add %s's method to it", ext.GetName(), cname)
	}
	if d := g.Model.Decl(t.GetCtor()); d != nil && !emit.IsOwn(d) {
		return emit.Unsupported(pos, "%s is declared by the prelude, and Go cannot add %s's method to it", d.GetMeta().GetName(), cname)
	}
	return nil
}

// pointerOrInterface reports whether a type's Go underlying type is a pointer
// or an interface, neither of which Go lets a declared type add methods to.
func (g *generator) pointerOrInterface(id *ir.ID, seen map[int32]bool) bool {
	t := g.Model.Type(id)
	d := g.Model.Decl(t.GetCtor())
	if d == nil || seen[t.GetCtor().GetIndex()] {
		return false
	}
	seen[t.GetCtor().GetIndex()] = true

	name := d.GetMeta().GetName()
	switch {
	case d.GetAlias() != nil:
		return g.pointerOrInterface(d.GetAlias().GetTarget(), seen)
	case d.GetNewtype() != nil:
		return g.pointerOrInterface(d.GetNewtype().GetBase(), seen)
	case d.GetEnumeration() != nil && (name == "Option" || name == "Nullable"):
		return true
	case d.GetEnumeration() != nil:
		return emit.Fielded(d.GetEnumeration())
	}
	return false
}

// hasField reports whether a declaration, or any variant of it, has a field
// with this Go name.
func (g *generator) hasField(d *ir.Decl, goName string) bool {
	fields := d.Fields()
	for _, v := range d.GetEnumeration().GetVariants() {
		fields = append(fields, v.GetFields()...)
	}
	for _, f := range fields {
		if g.fieldName(f) == goName {
			return true
		}
	}
	return false
}

// class renders a class as the interface its satisfying declarations
// implement, embedding the classes it requires.
func (g *generator) class(b *strings.Builder, decl *ir.Decl) error {
	pos, name := decl.GetMeta().GetPosition(), decl.GetMeta().GetName()
	if len(decl.GetClass().GetParams()) > 0 {
		return emit.Unsupported(pos, "%s takes type parameters, and a Go interface cannot state a relationship between types", name)
	}
	goName := g.declName(decl)
	if goName == "" {
		return emit.Unsupported(pos, "%s is named \"\" in this target, and its method is named after it", name)
	}
	if g.classCycle[g.cur] {
		return emit.Unsupported(pos, "%s requires itself, directly or through another class, and a Go interface cannot embed itself", name)
	}
	if len(decl.GetClass().GetAssocTypes()) > 0 {
		g.Warn(emit.Unsupported(pos, "%s requires associated types, and a Go interface cannot bind one, so it is generated without them", name))
	}

	g.doc(b, decl.GetMeta())
	fmt.Fprintf(b, "type %s interface {\n", goName)
	for _, s := range g.supers(decl) {
		fmt.Fprintf(b, "\t%s\n", g.declName(g.Model.GetDecls()[s]))
	}
	fmt.Fprintf(b, "\tis%s()\n}\n", goName)
	return nil
}

// requiresCycle reports whether a class's requires clause reaches the class
// itself, over the classes that are candidates to be generated.
func (g *generator) requiresCycle(start int32) bool {
	decls := g.Model.GetDecls()
	seen := map[int32]bool{}
	var reaches func(int32) bool
	reaches = func(i int32) bool {
		for _, ref := range decls[i].GetClass().GetRequiresClasses() {
			next := ref.GetClass()
			if next == nil || !g.genClass[next.GetIndex()] {
				continue
			}
			if next.GetIndex() == start {
				return true
			}
			if seen[next.GetIndex()] {
				continue
			}
			seen[next.GetIndex()] = true
			if reaches(next.GetIndex()) {
				return true
			}
		}
		return false
	}
	return reaches(start)
}

// supers is the generated classes a class requires.
//
// It says nothing about what it drops: [generator.planClasses] has already
// warned about each of those once, and this is asked again for every use of
// a constrained declaration.
func (g *generator) supers(class *ir.Decl) []int32 {
	var out []int32
	for _, ref := range class.GetClass().GetRequiresClasses() {
		if ref.GetClass() == nil {
			continue
		}
		if i := ref.GetClass().GetIndex(); g.genClass[i] {
			out = append(out, i)
		}
	}
	return out
}

// superProblem says why a class a class requires cannot be embedded in its
// Go interface, or returns nil when it can.
//
// Embedding is what makes the requirement hold, since a type satisfying the
// interface carries the method of everything it embeds. A requirement Go
// cannot embed is dropped and the interface is weaker than the model says,
// which is worth saying out loud.
func (g *generator) superProblem(class *ir.Decl, ref *ir.ClassRef) error {
	pos := ref.GetPosition()
	if pos == nil {
		pos = class.GetMeta().GetPosition()
	}
	into := g.declName(class)
	if ext := ref.GetExtern(); ext != nil {
		return emit.Unsupported(pos, "%s is declared in another package, so %s does not embed it", ext.GetName(), into)
	}
	super := g.Model.Decl(ref.GetClass())
	if super == nil {
		return emit.Unsupported(pos, "class %s did not resolve", ref.GetClass().GetName())
	}
	cname := super.GetMeta().GetName()
	switch {
	case !emit.IsOwn(super):
		return emit.Unsupported(pos, "%s is declared by the prelude, which is not generated, so %s does not embed it", cname, into)
	case !g.genClass[ref.GetClass().GetIndex()]:
		return emit.Unsupported(pos, "%s is not generated, so %s does not embed it", cname, into)
	}
	return nil
}

// writeMarkers writes the method marking the declaration being rendered as
// satisfying each class it carries.
func (g *generator) writeMarkers(b *strings.Builder, recv string) {
	for _, name := range g.markedClasses() {
		fmt.Fprintf(b, "\nfunc (%s) is%s() {}\n", recv, name)
	}
}

// markedClasses is the Go name of each class the declaration being rendered
// carries a marker for.
func (g *generator) markedClasses() []string {
	var names []string
	for _, c := range g.marks[g.cur] {
		names = append(names, g.declName(g.Model.GetDecls()[c]))
	}
	return names
}

// hasMarker reports whether a declaration carries a class's marker and the
// marker of every class that class requires, which is what implementing its
// interface takes.
func (g *generator) hasMarker(decl, class int32) bool {
	found := false
	for _, c := range g.marks[decl] {
		found = found || c == class
	}
	if !found {
		return false
	}
	for _, s := range g.supers(g.Model.GetDecls()[class]) {
		if !g.hasMarker(decl, s) {
			return false
		}
	}
	return true
}

// implies reports whether satisfying class c means satisfying want: they are
// one class, or c requires want, directly or through the classes it requires.
func (g *generator) implies(c, want int32, seen map[int32]bool) bool {
	if c == want {
		return true
	}
	if seen[c] {
		return false
	}
	seen[c] = true
	for _, s := range g.supers(g.Model.GetDecls()[c]) {
		if g.implies(s, want, seen) {
			return true
		}
	}
	return false
}

// paramClasses resolves a declaration's `requires` clause to the generated
// classes constraining each parameter. With report set, an entry Go cannot
// state is a warning; either way it is dropped, since the constraint is what
// is missing and not the type.
func (g *generator) paramClasses(decl *ir.Decl, report bool) [][]int32 {
	if len(decl.Params()) == 0 {
		return nil
	}
	out := make([][]int32, len(decl.Params()))
	for _, ref := range decl.Constraints() {
		i, err := g.constrains(decl, ref)
		if err != nil {
			if report {
				g.Warn(err)
			}
			continue
		}
		out[i] = append(out[i], ref.GetClass().GetIndex())
	}
	return out
}

// constrains resolves one `requires` entry to the parameter it constrains,
// or says why a Go constraint cannot state it.
func (g *generator) constrains(decl *ir.Decl, ref *ir.ClassRef) (int, error) {
	pos := ref.GetPosition()
	if ext := ref.GetExtern(); ext != nil {
		return 0, emit.Unsupported(pos, "%s is declared in another package, so the constraint naming it is not enforced", ext.GetName())
	}
	class := g.Model.Decl(ref.GetClass())
	if class == nil {
		return 0, emit.Unsupported(pos, "class %s did not resolve", ref.GetClass().GetName())
	}
	cname := class.GetMeta().GetName()
	switch {
	case !emit.IsOwn(class):
		return 0, emit.Unsupported(pos, "%s is declared by the prelude, which is not generated, so the constraint naming it is not enforced", cname)
	case !g.genClass[ref.GetClass().GetIndex()]:
		return 0, emit.Unsupported(pos, "%s is not generated, so the constraint naming it is not enforced", cname)
	}
	if len(ref.GetArgs()) == 1 {
		t := g.Model.Type(ref.GetArgs()[0])
		if p := t.GetParam(); p != nil && len(t.GetArgs()) == 0 && int(p.GetIndex()) < len(decl.Params()) {
			return int(p.GetIndex()), nil
		}
	}
	return 0, emit.Unsupported(pos, "requires %s constrains something other than one of %s's type parameters, which a Go constraint cannot say, so it is not enforced",
		cname, decl.GetMeta().GetName())
}

// satisfies reports whether a type argument satisfies a generated class: a
// declaration carrying its marker, or a parameter whose constraint implies
// it.
func (g *generator) satisfies(id *ir.ID, fr *frame, class int32) bool {
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
			return g.satisfies(fr.args[ref.GetIndex()], fr.outer, class)
		case int(ref.GetIndex()) >= len(g.curClasses):
			return false
		}
		for _, c := range g.curClasses[ref.GetIndex()] {
			if g.implies(c, class, map[int32]bool{}) {
				return true
			}
		}
		return false
	}

	d := g.Model.Decl(t.GetCtor())
	if d == nil {
		return false
	}
	if a := d.GetAlias(); a != nil {
		return g.satisfies(a.GetTarget(), bind(t.GetArgs(), fr), class)
	}
	return g.hasMarker(t.GetCtor().GetIndex(), class)
}
