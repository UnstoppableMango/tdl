package sema

import (
	"github.com/unstoppablemango/tdl/ir"
)

// accumulateConstraints copies a newtype's constraints down the chain it
// builds on.
//
// `type WorkEmail: Email where { matches(...) }` carries Email's
// constraints too. A newtype narrows its parent and never replaces it, so
// a value satisfying WorkEmail satisfies Email, and a backend reading one
// list gets the whole story rather than walking the chain itself.
//
// Each inherited constraint records which newtype it came from, so a
// backend can still explain where a rule started.
func (l *lowerer) accumulateConstraints() {
	done := map[int32]bool{}
	for i := range l.model.GetDecls() {
		l.accumulateInto(int32(i), done, map[int32]bool{})
	}
}

func (l *lowerer) accumulateInto(idx int32, done, onPath map[int32]bool) {
	if done[idx] || onPath[idx] {
		return // a cycle is already reported by the recursion check
	}

	decl := l.model.GetDecls()[idx]
	newtype := decl.GetNewtype()
	if newtype == nil {
		done[idx] = true
		return
	}

	onPath[idx] = true
	defer func() {
		delete(onPath, idx)
		done[idx] = true
	}()

	base := l.model.Type(newtype.GetBase())
	if base == nil || !base.GetCtor().Resolved() {
		return
	}

	parentIdx := base.GetCtor().GetIndex()
	l.accumulateInto(parentIdx, done, onPath)

	parent := l.model.GetDecls()[parentIdx].GetNewtype()
	if parent == nil {
		return
	}

	from := &ir.ID{Index: parentIdx, Name: l.model.GetDecls()[parentIdx].GetMeta().GetName()}
	for _, c := range parent.GetValueConstraints() {
		inherited := &ir.Constraint{
			Name:     c.GetName(),
			Args:     c.GetArgs(),
			Position: c.GetPosition(),
			From:     c.GetFrom(),
		}
		if inherited.From == nil {
			inherited.From = from
		}
		newtype.ValueConstraints = append(newtype.ValueConstraints, inherited)
	}
}

// checkDefaults resolves a field default written as a name.
//
// A name denotes an enum variant, and which enum it belongs to is the
// field's type, so this is the first point at which the check can be made:
// the parser records the name, and nothing before now knows the type.
func (l *lowerer) checkDefaults() {
	for _, decl := range l.model.GetDecls() {
		for _, f := range decl.Fields() {
			l.checkDefault(f)
		}
		for _, v := range decl.GetEnumeration().GetVariants() {
			for _, f := range v.GetFields() {
				l.checkDefault(f)
			}
		}
	}
}

func (l *lowerer) checkDefault(f *ir.Field) {
	def := f.GetDefaultValue()
	if def == nil || def.GetKind() != ir.LiteralKind_LITERAL_KIND_NAME {
		return
	}

	ty := l.model.Type(f.GetType())
	if ty == nil || ty.GetParam() != nil || ty.GetExtern() != nil {
		return // a parameter or a foreign type: nothing local to check against
	}

	// Look through the optionality sugar: `status: Status? = Draft` names a
	// variant of Status.
	for len(ty.GetArgs()) == 1 && isOptionLike(ty) {
		ty = l.model.Type(ty.GetArgs()[0])
		if ty == nil {
			return
		}
	}

	decl := l.model.Decl(ty.GetCtor())
	if decl == nil {
		return // already reported as undefined
	}

	enum := decl.GetEnumeration()
	if enum == nil {
		l.diags.add(positionOf(def.GetPosition()), "%s is not an enum, so %s is not a value for it",
			decl.GetMeta().GetName(), def.GetText())
		return
	}

	for i, v := range enum.GetVariants() {
		if v.GetMeta().GetName() == def.GetText() {
			def.Variant = &ir.ID{Index: int32(i), Name: v.GetMeta().GetName()}
			return
		}
	}
	l.diags.add(positionOf(def.GetPosition()), "%s has no variant %s",
		decl.GetMeta().GetName(), def.GetText())
}

// isOptionLike reports whether a type is the sugar for absence or null,
// which a default looks through.
func isOptionLike(ty *ir.Type) bool {
	switch ty.GetWrote() {
	case ir.SyntacticForm_SYNTACTIC_FORM_QUESTION, ir.SyntacticForm_SYNTACTIC_FORM_OR_NULL:
		return true
	}
	name := ty.GetCtor().GetName()
	return name == preludeOption || name == preludeNullable
}
