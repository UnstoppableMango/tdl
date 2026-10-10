package golang

import (
	"fmt"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// A class is a Go interface with one unexported marker method, which each
// satisfying declaration carries, keeping conformance nominal as in TDL.
// A class's fields are not interface methods: Go refuses a field and a
// method with one name, and generated code never calls into a type
// argument, so the marker is all a constraint needs.

// planClasses decides which classes are generated and which declarations
// carry each marker.
func (g *generator) planClasses() {
	plan := g.PlanInterfaces(emit.InterfaceRules{
		// A Go interface states no relationship between types.
		Generates: func(d *ir.Decl) bool { return len(d.GetClass().GetParams()) == 0 && g.declName(d) != "" },
		Carry:     g.markerProblem,
	})
	g.classes = plan
	g.genClass, g.classCycle, g.marks = plan.Classes, plan.Cyclic, plan.Implements

	decls := g.Model.GetDecls()
	for i, class := range decls {
		if !g.genClass[int32(i)] {
			continue
		}
		// Reported once here; [generator.supers] drops these silently.
		for _, ref := range class.GetClass().GetRequiresClasses() {
			if err := g.superProblem(class, ref); err != nil {
				g.lose(emit.LossClass, err)
			}
		}
	}

	for _, inst := range g.Model.GetInstances() {
		if err := g.instanceProblem(inst); err != nil {
			g.lose(emit.LossClass, err)
		}
	}
}

// markerProblem says why a declaration cannot carry a class's marker, or
// returns nil.
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
// a marker, or returns nil.
func (g *generator) instanceProblem(inst *ir.Instance) error {
	refusal, typ := g.classes.Instance(g.Model, inst)
	if refusal == emit.Implemented {
		return nil
	}
	pos := inst.GetMeta().GetPosition()
	cname := g.Model.Decl(inst.GetClass().GetClass()).GetMeta().GetName()
	switch refusal {
	case emit.Conditional:
		return emit.Unsupported(pos, "a conditional instance of %s holds for some type arguments, and Go cannot give a method to only some instantiations", cname)
	case emit.ForeignType:
		return emit.Unsupported(pos, "%s is declared in another package, and Go cannot add %s's method to it", typ, cname)
	default:
		return emit.Unsupported(pos, "%s is declared by the prelude, and Go cannot add %s's method to it", typ, cname)
	}
}

// pointerOrInterface reports whether a type's Go underlying type is a pointer
// or an interface, which cannot carry methods.
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

// class renders a class as an interface embedding the classes it requires.
func (g *generator) class(b *strings.Builder, decl *ir.Decl) error {
	pos, name := decl.GetMeta().GetPosition(), decl.GetMeta().GetName()
	if len(decl.GetClass().GetParams()) > 0 {
		return emit.Lost(emit.LossClass, pos, "%s takes type parameters, and a Go interface cannot state a relationship between types", name)
	}
	goName := g.declName(decl)
	if goName == "" {
		return emit.Unsupported(pos, "%s is named \"\" in this target, and its method is named after it", name)
	}
	if g.classCycle[g.cur] {
		return emit.Unsupported(pos, "%s requires itself, directly or through another class, and a Go interface cannot embed itself", name)
	}
	if len(decl.GetClass().GetAssocTypes()) > 0 {
		g.Lose(emit.LossClass, pos, "%s requires associated types, and a Go interface cannot bind one, so it is generated without them", name)
	}

	g.doc(b, decl.GetMeta())
	g.annotate(b, decl)
	fmt.Fprintf(b, "type %s interface {\n", goName)
	for _, s := range g.supers(decl) {
		fmt.Fprintf(b, "\t%s\n", g.declName(g.Model.GetDecls()[s]))
	}
	fmt.Fprintf(b, "\tis%s()\n}\n", goName)
	return nil
}

// supers is the generated classes a class requires. [generator.planClasses]
// warns about the rest.
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

// superProblem says why a required class cannot be embedded in the class's
// interface, or returns nil. A dropped requirement weakens the interface.
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

// writeMarkers writes a marker method for each class the declaration being
// rendered satisfies.
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

// hasMarker reports whether a declaration implements a class's interface:
// it carries the class's marker and those of every class it requires.
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

// implies reports whether c is want or requires it, directly or not.
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
// classes constraining each parameter. An entry Go cannot state is dropped,
// with a warning when report is set.
func (g *generator) paramClasses(decl *ir.Decl, report bool) [][]int32 {
	if len(decl.Params()) == 0 {
		return nil
	}
	out := make([][]int32, len(decl.Params()))
	for _, ref := range decl.Constraints() {
		i, err := g.constrains(decl, ref)
		if err != nil {
			if report {
				g.lose(emit.LossGeneric, err)
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

// satisfies reports whether a type argument carries a generated class's
// marker, or is a parameter whose constraint implies it.
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
