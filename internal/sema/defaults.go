package sema

import (
	"github.com/unstoppablemango/tdl/ir"
)

// accumulateConstraints copies a newtype's constraints down the chain it
// builds on, so `type WorkEmail: Email where { ... }` carries Email's
// constraints too, each recording the newtype it came from.
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

// resolveNames resolves a default or a constraint argument written as a
// name, on a field or a newtype, to a variant of the enum it constrains.
func (l *lowerer) resolveNames() {
	for _, decl := range l.model.GetDecls() {
		for _, f := range decl.Fields() {
			l.resolveFieldNames(f)
		}
		for _, v := range decl.GetEnumeration().GetVariants() {
			for _, f := range v.GetFields() {
				l.resolveFieldNames(f)
			}
		}
		if nt := decl.GetNewtype(); nt != nil {
			l.resolveConstraintArgs(nt.GetBase(), nt.GetValueConstraints())
		}
	}
}

func (l *lowerer) resolveFieldNames(f *ir.Field) {
	l.checkDefault(f)
	l.resolveConstraintArgs(f.GetType(), f.GetConstraints())
}

func (l *lowerer) checkDefault(f *ir.Field) {
	def := f.GetDefaultValue()
	if def == nil || def.GetKind() != ir.LiteralKind_LITERAL_KIND_NAME {
		return
	}

	decl := l.valueDecl(f.GetType())
	if decl == nil {
		return // a parameter, a foreign type, or already reported as undefined
	}

	enum := decl.GetEnumeration()
	if enum == nil {
		l.diags.add(positionOf(def.GetPosition()), "%s is not an enum, so %s is not a value for it",
			decl.GetMeta().GetName(), def.GetText())
		return
	}
	l.resolveVariant(decl, enum, def)
}

// resolveConstraintArgs resolves each constraint argument written as a
// name to a variant of the type's enum, skipping one a newtype inherited,
// which is resolved where it was written. On a type that is not an enum
// the name is left for the backend.
func (l *lowerer) resolveConstraintArgs(id *ir.ID, cs []*ir.Constraint) {
	decl := l.valueDecl(id)
	enum := decl.GetEnumeration()
	if enum == nil {
		return
	}
	for _, c := range cs {
		if c.GetFrom() != nil {
			continue
		}
		for _, arg := range c.GetArgs() {
			if arg.GetKind() == ir.LiteralKind_LITERAL_KIND_NAME {
				l.resolveVariant(decl, enum, arg)
			}
		}
	}
}

// resolveVariant points a name at the variant it denotes, and reports one
// that denotes none.
func (l *lowerer) resolveVariant(decl *ir.Decl, enum *ir.Enum, lit *ir.Literal) {
	for i, v := range enum.GetVariants() {
		if v.GetMeta().GetName() == lit.GetText() {
			lit.Variant = &ir.ID{Index: int32(i), Name: v.GetMeta().GetName()}
			return
		}
	}
	l.diags.add(positionOf(lit.GetPosition()), "%s has no variant %s",
		decl.GetMeta().GetName(), lit.GetText())
}

// valueDecl is the declaration whose values a type holds, looking through
// optionality, aliases, and newtypes, or nil when there is nothing local to
// resolve a name against.
func (l *lowerer) valueDecl(id *ir.ID) *ir.Decl {
	seen := map[int32]bool{}
	for {
		ty := l.model.Type(id)
		if ty == nil || ty.GetParam() != nil || ty.GetExtern() != nil {
			return nil // a parameter or a foreign type: nothing local to check against
		}
		if len(ty.GetArgs()) == 1 && isOptionLike(ty) {
			id = ty.GetArgs()[0]
			continue
		}
		decl := l.model.Decl(ty.GetCtor()) // nil when already reported as undefined
		if decl == nil || seen[ty.GetCtor().GetIndex()] {
			return decl // a cycle is already reported by the recursion check
		}
		seen[ty.GetCtor().GetIndex()] = true
		switch {
		case decl.GetAlias() != nil:
			id = decl.GetAlias().GetTarget()
		case decl.GetNewtype() != nil:
			id = decl.GetNewtype().GetBase()
		default:
			return decl
		}
	}
}

// isOptionLike reports whether a type is Option or Nullable.
func isOptionLike(ty *ir.Type) bool {
	switch ty.GetWrote() {
	case ir.SyntacticForm_SYNTACTIC_FORM_QUESTION, ir.SyntacticForm_SYNTACTIC_FORM_OR_NULL:
		return true
	}
	name := ty.GetCtor().GetName()
	return name == preludeOption || name == preludeNullable
}
