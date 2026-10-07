package sema

import (
	"github.com/unstoppablemango/tdl/ir"
)

// checkClassFields reports a type that conforms to a class without the
// fields the class, or a class it requires, declares. A class's fields
// bind every satisfying type, so a backend may read them through the class.
// It runs after expandIncludes, so a field a mixin supplies counts.
//
// A type conforms where it is declared, as in `type Order: Auditable`, or
// through an instance, conditional or not; an instance's head is checked
// as the constructor it applies. An extern's fields are not in the model,
// so an instance for one is not checked.
func (l *lowerer) checkClassFields() {
	for i, decl := range l.model.GetDecls() {
		for _, ref := range conformsOf(decl) {
			l.checkFieldsOf(int32(i), ref)
		}
	}

	for _, inst := range l.model.GetInstances() {
		args := inst.GetClass().GetArgs()
		if len(args) != 1 {
			continue // a class with several parameters declares no fields
		}
		ty := l.model.Type(args[0])
		if ty == nil || !ty.GetCtor().Resolved() {
			continue
		}
		l.checkFieldsOf(ty.GetCtor().GetIndex(), inst.GetClass())
	}
}

// checkFieldsOf checks one declaration against one conformance.
func (l *lowerer) checkFieldsOf(decl int32, ref *ir.ClassRef) {
	subject := l.model.GetDecls()[decl]
	have := map[string]*ir.Field{}
	for _, f := range subject.GetStructure().GetFields() {
		have[f.GetMeta().GetName()] = f
	}

	for _, req := range l.requiredFields(ref.GetClass(), map[int32]bool{}) {
		name := req.field.GetMeta().GetName()
		got, ok := have[name]
		if !ok {
			l.diags.add(positionOf(ref.GetPosition()), "%s has no field %s, required by %s",
				subject.GetMeta().GetName(), name, req.class)
			continue
		}
		if !l.sameType(got.GetType(), req.field.GetType()) {
			l.diags.add(positionOf(got.GetMeta().GetPosition()), "%s.%s is %s, but %s requires %s",
				subject.GetMeta().GetName(), name, got.GetType().GetName(), req.class, req.field.GetType().GetName())
		}
	}
}

type requiredField struct {
	field *ir.Field
	class string // the class declaring it
}

// requiredFields returns a class's fields and those of every class it
// requires, transitively.
func (l *lowerer) requiredFields(class *ir.ID, seen map[int32]bool) []requiredField {
	if !class.Resolved() || seen[class.GetIndex()] {
		return nil
	}
	seen[class.GetIndex()] = true

	decl := l.model.Decl(class)
	var out []requiredField
	for _, f := range decl.GetClass().GetFields() {
		out = append(out, requiredField{field: f, class: decl.GetMeta().GetName()})
	}
	for _, sup := range decl.GetClass().GetRequiresClasses() {
		out = append(out, l.requiredFields(sup.GetClass(), seen)...)
	}
	return out
}

// sameType compares two types structurally, ignoring how each was spelled
// and seeing through an alias without parameters. A type that mentions a
// parameter or failed to resolve is not compared, since the field is
// present and the parameter is the instantiation's to decide.
func (l *lowerer) sameType(a, b *ir.ID) bool {
	ta, tb := l.expandAlias(l.model.Type(a)), l.expandAlias(l.model.Type(b))
	if ta == nil || tb == nil || ta.GetParam() != nil || tb.GetParam() != nil {
		return true
	}

	switch {
	case ta.GetCtor() != nil || tb.GetCtor() != nil:
		if !ta.GetCtor().Resolved() || !tb.GetCtor().Resolved() {
			return true
		}
		if ta.GetCtor().GetIndex() != tb.GetCtor().GetIndex() {
			return false
		}
	case ta.GetExtern() != nil || tb.GetExtern() != nil:
		if ta.GetExtern().GetIndex() != tb.GetExtern().GetIndex() {
			return false
		}
	}
	if ta.GetUnit().GetIndex() != tb.GetUnit().GetIndex() || len(ta.GetArgs()) != len(tb.GetArgs()) {
		return false
	}
	for i := range ta.GetArgs() {
		if !l.sameType(ta.GetArgs()[i], tb.GetArgs()[i]) {
			return false
		}
	}
	return true
}

// expandAlias follows aliases without parameters to the type they name.
func (l *lowerer) expandAlias(t *ir.Type) *ir.Type {
	for range searchDepth {
		alias := l.model.Decl(t.GetCtor()).GetAlias()
		if alias == nil || len(alias.GetParams()) > 0 || len(t.GetArgs()) > 0 {
			return t
		}
		t = l.model.Type(alias.GetTarget())
	}
	return t
}
