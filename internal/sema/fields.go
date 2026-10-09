package sema

import (
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/ir"
)

// checkClassFields reports a type that conforms to a class without the
// fields the class, or a class it requires, declares. A class's fields
// bind every satisfying type, so a backend may read them through the class.
// It runs after expandIncludes, so a field a mixin supplies counts.
//
// A type conforms where it is declared, as in `type Order: Auditable`, or
// through an instance, conditional or not; an instance's head is checked
// as the constructor it applies. An instance for an extern is checked
// against the dependency declaring it, lowered on demand.
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
		if ext := ty.GetExtern(); ext.Resolved() {
			l.checkExternFields(ext, inst.GetClass())
			continue
		}
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

// checkExternFields checks an instance for a declaration in another
// package. Its fields are in the dependency's model, so field types are
// compared by what they name rather than by index.
func (l *lowerer) checkExternFields(ext *ir.ID, ref *ir.ClassRef) {
	extern := l.model.GetExterns()[ext.GetIndex()]
	dep := l.dependency(extern.GetPackage())
	if dep == nil {
		return
	}
	b, ok := dep.file.names[extern.GetName()]
	if !ok || b.kind != bindDecl {
		return // the dependency's own diagnostics report it
	}
	subject := dep.model.Decl(b.id)
	name := extern.GetPackage() + "." + extern.GetName()

	have := map[string]*ir.Field{}
	for _, f := range subject.GetStructure().GetFields() {
		have[f.GetMeta().GetName()] = f
	}
	for _, req := range l.requiredFields(ref.GetClass(), map[int32]bool{}) {
		field := req.field.GetMeta().GetName()
		got, ok := have[field]
		if !ok {
			l.diags.add(positionOf(ref.GetPosition()), "%s has no field %s, required by %s", name, field, req.class)
			continue
		}
		want, wok := l.typeKey(req.field.GetType())
		is, iok := dep.typeKey(got.GetType())
		if wok && iok && want != is {
			l.diags.add(positionOf(ref.GetPosition()), "%s.%s is %s, but %s requires %s",
				name, field, got.GetType().GetName(), req.class, req.field.GetType().GetName())
		}
	}
}

// dependency lowers the root import declaring pkg, once. A dependency is
// lowered without checking its own imports' types, so a cycle ends, and
// its diagnostics are its own.
func (l *lowerer) dependency(pkg string) *lowerer {
	if dep, ok := l.deps[pkg]; ok {
		return dep
	}
	var dep *lowerer
	if file, ok := l.depFiles[pkg]; ok && l.cfg.checkDeps {
		cfg := l.cfg
		cfg.checkDeps, cfg.refs = false, nil
		dep = lowerFile(file, cfg)
	}
	l.deps[pkg] = dep
	return dep
}

// typeKey names a type by what it refers to, so two models can compare
// one: a declaration by package and name, with the prelude's apart. It
// reports false for a type mentioning a parameter or failing to resolve,
// which is not compared, as in [lowerer.sameType].
func (l *lowerer) typeKey(id *ir.ID) (string, bool) {
	t := l.expandAlias(l.model.Type(id))
	if t == nil || t.GetParam() != nil {
		return "", false
	}

	var b strings.Builder
	switch {
	case t.GetCtor() != nil:
		if !t.GetCtor().Resolved() {
			return "", false
		}
		b.WriteString(l.declKey(t.GetCtor().GetIndex()))
	case t.GetExtern() != nil:
		ext := l.model.GetExterns()[t.GetExtern().GetIndex()]
		if key, ok := l.externUnitKey(ext); ok {
			b.WriteString(key)
			break
		}
		b.WriteString(ext.GetPackage() + "." + ext.GetName())
	case t.GetUnit() != nil:
		b.WriteString(l.unitKey(l.model.GetUnits()[t.GetUnit().GetIndex()]))
	}

	if len(t.GetArgs()) > 0 {
		b.WriteString("<")
		for i, arg := range t.GetArgs() {
			key, ok := l.typeKey(arg)
			if !ok {
				return "", false
			}
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(key)
		}
		b.WriteString(">")
	}
	return b.String(), true
}

// unitKey names a unit by its base dimensions, which is how either model
// spells it after reduction.
func (l *lowerer) unitKey(u *ir.Unit) string {
	var b strings.Builder
	for i, d := range u.GetDims() {
		if i > 0 {
			b.WriteString("*")
		}
		b.WriteString(l.declKey(d.GetBase().GetIndex()))
		if d.GetExponent() != 1 {
			b.WriteString("^" + strconv.Itoa(int(d.GetExponent())))
		}
	}
	return b.String()
}

// externUnitKey is the [lowerer.unitKey] of an extern naming a unit, read
// from its dependency, since a type argument naming one is recorded as a
// type reference.
func (l *lowerer) externUnitKey(ext *ir.Extern) (string, bool) {
	dep := l.dependency(ext.GetPackage())
	if dep == nil {
		return "", false
	}
	b, ok := dep.file.names[ext.GetName()]
	if !ok || b.kind != bindDecl {
		return "", false
	}
	def := dep.model.Decl(b.id).GetUnit()
	if !def.GetUnit().Resolved() {
		return "", false
	}
	return dep.unitKey(dep.model.GetUnits()[def.GetUnit().GetIndex()]), true
}

// declKey is a declaration's package-qualified name. A prelude
// declaration has no package, so it is the same in every model.
func (l *lowerer) declKey(idx int32) string {
	name := l.model.GetDecls()[idx].GetMeta().GetName()
	if int(idx) < l.preludeDecls {
		return "." + name
	}
	return l.model.GetPackage() + "." + name
}
