package sema

import (
	"github.com/unstoppablemango/tdl/ir"
)

// searchDepth bounds the conditional instance search.
//
// The spec's two rules on an instance are what make the search finite, and
// they are checked at the declaration. This is a backstop for the case
// where those checks and the search disagree: it turns a hang into a
// diagnostic, and reaching it is a bug rather than a user error.
const searchDepth = 32

// validateInstances enforces the spec's two rules on a conditional
// instance, at the point of declaration rather than at use.
//
// An instance head must be a type constructor applied to distinct
// parameters, and every constraint in the `requires` clause must be
// structurally smaller than the head. Without the first, matching a head
// is not a decidable pattern match; without the second, discharging a
// condition can produce a goal no smaller than the one it came from, and
// the search does not terminate.
func (l *lowerer) validateInstances() {
	for _, inst := range l.model.GetInstances() {
		if len(inst.GetParams()) == 0 {
			continue // a ground instance matches itself and asks nothing
		}

		headSize := 0
		for _, arg := range inst.GetClass().GetArgs() {
			headSize += l.typeSize(arg)
			l.checkHeadShape(inst, arg)
		}

		for _, req := range inst.GetRequires() {
			size := 0
			for _, arg := range req.GetArgs() {
				size += l.typeSize(arg)
			}
			if size >= headSize {
				l.diags.add(positionOf(req.GetPosition()),
					"%s is not smaller than the instance head, so resolving it would not terminate",
					req.GetClass().GetName())
			}
		}
	}
}

// checkHeadShape reports a head argument that is not a constructor applied
// to distinct parameters.
func (l *lowerer) checkHeadShape(inst *ir.Instance, arg *ir.ID) {
	ty := l.model.Type(arg)
	if ty == nil {
		return
	}

	// A bare parameter as the whole head would match every type.
	if ty.GetParam() != nil {
		l.diags.add(positionOf(inst.GetClass().GetPosition()),
			"an instance head must be a type constructor, not the bare parameter %s",
			ty.GetParam().GetName())
		return
	}

	seen := map[string]bool{}
	for _, sub := range ty.GetArgs() {
		subTy := l.model.Type(sub)
		if subTy == nil {
			continue
		}
		ref := subTy.GetParam()
		if ref == nil {
			l.diags.add(positionOf(inst.GetClass().GetPosition()),
				"an instance head must be applied to parameters, not to %s", sub.GetName())
			continue
		}
		if seen[ref.GetName()] {
			l.diags.add(positionOf(inst.GetClass().GetPosition()),
				"the instance head repeats the parameter %s", ref.GetName())
		}
		seen[ref.GetName()] = true
	}
}

// ground reports whether a type mentions no parameters.
func (l *lowerer) ground(id *ir.ID) bool {
	ty := l.model.Type(id)
	if ty == nil {
		return false
	}
	if ty.GetParam() != nil {
		return false
	}
	for _, arg := range ty.GetArgs() {
		if !l.ground(arg) {
			return false
		}
	}
	return true
}

// typeSize counts the constructors and parameters in a type, which is the
// measure "structurally smaller" refers to.
func (l *lowerer) typeSize(id *ir.ID) int {
	ty := l.model.Type(id)
	if ty == nil {
		return 0
	}

	size := 1
	for _, arg := range ty.GetArgs() {
		size += l.typeSize(arg)
	}
	return size
}

// searchSatisfaction fills in the types that satisfy each class through a
// conditional instance.
//
// Every type in the table is tried against every class. The table holds
// only types the model actually mentions, so this asks the question that
// can come up rather than enumerating a space.
func (l *lowerer) searchSatisfaction() {
	if len(l.model.GetInstances()) == 0 {
		return
	}

	for _, sat := range l.model.GetSatisfies() {
		for i, ty := range l.model.GetTypes() {
			if len(ty.GetArgs()) == 0 {
				continue // a bare name is a declaration question, already answered
			}

			id := &ir.ID{Index: int32(i), Name: typeName(ty)}
			if !l.ground(id) {
				// An open type like `Page<T>` is the instance head itself, or
				// a use inside a generic declaration. Whether it satisfies
				// anything depends on what T becomes, so it is not an answer.
				continue
			}
			if l.satisfiesType(sat.GetClass(), id, 0) {
				sat.Types = append(sat.Types, id)
			}
		}
	}
}

// satisfiesType reports whether a type satisfies a class.
//
// A type whose constructor is a declaration answers from the ground index.
// Anything else is matched against the class's conditional instances: bind
// the head's parameters to the type's arguments, then discharge each
// condition under that binding.
func (l *lowerer) satisfiesType(class, id *ir.ID, depth int) bool {
	if depth > searchDepth {
		l.diags.add(positionOf(l.model.Type(id).GetPosition()),
			"gave up resolving whether %s satisfies %s", id.GetName(), class.GetName())
		return false
	}

	ty := l.model.Type(id)
	if ty == nil {
		return false
	}

	if len(ty.GetArgs()) == 0 && l.satisfies(class, ty.GetCtor()) {
		return true
	}

	for _, inst := range l.model.GetInstances() {
		if inst.GetClass().GetClass().GetIndex() != class.GetIndex() {
			continue
		}
		if len(inst.GetClass().GetArgs()) != 1 {
			continue // multi-parameter classes are matched, not searched
		}

		binding, ok := l.match(inst.GetClass().GetArgs()[0], id)
		if !ok {
			continue
		}
		if l.discharge(inst, binding, depth) {
			return true
		}
	}
	return false
}

// match unifies an instance head against a type, returning the parameter
// binding it implies. The head is a constructor applied to distinct
// parameters, so this is a shallow match rather than full unification.
func (l *lowerer) match(head, subject *ir.ID) (map[string]*ir.ID, bool) {
	h, s := l.model.Type(head), l.model.Type(subject)
	if h == nil || s == nil {
		return nil, false
	}
	if h.GetCtor().GetIndex() != s.GetCtor().GetIndex() || !h.GetCtor().Resolved() {
		return nil, false
	}
	if len(h.GetArgs()) != len(s.GetArgs()) {
		return nil, false
	}

	binding := map[string]*ir.ID{}
	for i, arg := range h.GetArgs() {
		ref := l.model.Type(arg).GetParam()
		if ref == nil {
			return nil, false
		}
		binding[ref.GetName()] = s.GetArgs()[i]
	}
	return binding, true
}

// discharge checks an instance's conditions under a binding.
func (l *lowerer) discharge(inst *ir.Instance, binding map[string]*ir.ID, depth int) bool {
	for _, req := range inst.GetRequires() {
		for _, arg := range req.GetArgs() {
			ref := l.model.Type(arg).GetParam()
			if ref == nil {
				continue // a concrete condition; the ground index covers it
			}

			bound, ok := binding[ref.GetName()]
			if !ok {
				return false
			}
			if !l.satisfiesType(req.GetClass(), bound, depth+1) {
				return false
			}
		}
	}
	return true
}
