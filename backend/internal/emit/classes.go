package emit

import "github.com/unstoppablemango/tdl/ir"

// The class plan shared by every target that writes a class as a nominal
// interface, which a satisfying declaration lists or marks in its own
// declaration. What such an interface cannot carry is the same in each:
// an interface cannot reach itself through what it requires, cannot be
// implemented for only some instantiations of a generic type, and cannot be
// added to a type this package does not declare. A target adds its own
// refusals through [InterfaceRules].

// InterfacePlan says which classes are generated and which declarations
// implement each.
type InterfacePlan struct {
	// Classes holds the classes generated as interfaces.
	Classes map[int32]bool
	// Cyclic holds the classes left out because their requires clause
	// reaches themselves.
	Cyclic map[int32]bool
	// Implements maps a declaration to the generated classes it implements,
	// in declaration order.
	Implements map[int32][]int32
}

// InterfaceRules is what a target decides for itself.
type InterfaceRules struct {
	// Generates reports whether one of the model's classes can be an
	// interface at all, before cycles are removed. Nil allows every class.
	Generates func(class *ir.Decl) bool
	// Carry says why a satisfying declaration cannot implement a class, or
	// returns nil. The error is reported and the declaration does not
	// implement the class. Nil allows every declaration.
	Carry func(target, class *ir.Decl) error
}

// PlanInterfaces decides the plan, warning about each declaration that
// satisfies a generated class and cannot implement it.
func (s *Session) PlanInterfaces(r InterfaceRules) *InterfacePlan {
	p := &InterfacePlan{
		Classes:    map[int32]bool{},
		Cyclic:     map[int32]bool{},
		Implements: map[int32][]int32{},
	}
	decls := s.Model.GetDecls()
	for i, d := range decls {
		if d.GetClass() != nil && IsOwn(d) && (r.Generates == nil || r.Generates(d)) {
			p.Classes[int32(i)] = true
		}
	}

	// An interface cannot extend itself, so a class whose requires clause
	// reaches itself is dropped first.
	for i := range decls {
		if p.Classes[int32(i)] && p.requiresCycle(decls, int32(i)) {
			p.Cyclic[int32(i)] = true
		}
	}
	for i := range p.Cyclic {
		delete(p.Classes, i)
	}

	for i, class := range decls {
		if !p.Classes[int32(i)] {
			continue
		}
		for _, id := range s.Model.Satisfying(&ir.ID{Index: int32(i)}) {
			target := s.Model.Decl(id)
			if target == nil || !IsOwn(target) {
				continue
			}
			if r.Carry != nil {
				if err := r.Carry(target, class); err != nil {
					s.Warn(err)
					continue
				}
			}
			p.Implements[id.GetIndex()] = append(p.Implements[id.GetIndex()], int32(i))
		}
	}
	return p
}

// requiresCycle reports whether a class's requires clause reaches the class
// itself through candidate classes.
func (p *InterfacePlan) requiresCycle(decls []*ir.Decl, start int32) bool {
	seen := map[int32]bool{}
	var reaches func(int32) bool
	reaches = func(i int32) bool {
		for _, ref := range decls[i].GetClass().GetRequiresClasses() {
			next := ref.GetClass()
			if next == nil || !p.Classes[next.GetIndex()] {
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

// Refusal is why an instance of a generated class has no implementation.
type Refusal int

const (
	// Implemented: the instance needs nothing the plan refuses.
	Implemented Refusal = iota
	// Conditional: the instance holds for some type arguments only.
	Conditional
	// ForeignType: the instance is for a type another package declares.
	ForeignType
	// PreludeType: the instance is for a type the prelude declares.
	PreludeType
)

// Instance says whether a separately declared instance of a generated class
// can become an implementation, and names the type it is for when it is a
// [ForeignType] or a [PreludeType]. An instance of a class the plan does not
// generate is [Implemented], since there is nothing to refuse.
func (p *InterfacePlan) Instance(m *ir.Model, inst *ir.Instance) (Refusal, string) {
	ref := inst.GetClass()
	if ref.GetExtern() != nil || !p.Classes[ref.GetClass().GetIndex()] {
		return Implemented, ""
	}
	if len(inst.GetParams()) > 0 || len(inst.GetRequires()) > 0 {
		return Conditional, ""
	}
	if len(ref.GetArgs()) != 1 {
		return Implemented, ""
	}
	t := m.Type(ref.GetArgs()[0])
	if ext := t.GetExtern(); ext != nil {
		return ForeignType, ext.GetName()
	}
	if d := m.Decl(t.GetCtor()); d != nil && !IsOwn(d) {
		return PreludeType, d.GetMeta().GetName()
	}
	return Implemented, ""
}
