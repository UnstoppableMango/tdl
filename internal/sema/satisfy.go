package sema

import (
	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// expandIncludes copies a mixin's fields into the declarations that include
// it.
//
// The copy runs after every declaration is lowered, so a mixin may be
// included before it is declared and a mixin may include another. Each
// copied field records the mixin it came from, so a backend that can
// express the grouping does not have to reconstruct it.
//
// Including a mixin is not what makes a type satisfy a class. The spec says
// the two are independent, and conformance is nominal and declared, so
// nothing here touches the satisfaction index.
func (l *lowerer) expandIncludes(file *ast.File) {
	// A mixin including a mixin has to be expanded before its own fields are
	// copied onward, so declarations are visited in dependency order.
	done := map[string]bool{}
	for _, decl := range file.Decls {
		l.expandInto(file, decl.Name(), done, map[string]bool{})
	}
}

func (l *lowerer) expandInto(file *ast.File, name string, done, onPath map[string]bool) {
	if done[name] || onPath[name] {
		if onPath[name] {
			l.diags.add(ast.Position{}, "include cycle through %s", name)
		}
		return
	}
	onPath[name] = true
	defer func() {
		delete(onPath, name)
		done[name] = true
	}()

	decl := findDecl(file, name)
	structure, ok := decl.(*ast.StructDecl)
	if !ok {
		return
	}

	idx, ok := l.byName[name]
	if !ok {
		return
	}
	target := l.model.Decls[idx].GetStructure()
	if target == nil {
		return
	}

	for _, m := range structure.Members {
		inc, ok := m.(*ast.Include)
		if !ok {
			continue
		}

		src, id, found := l.includedMixin(inc)
		if !found {
			continue
		}
		l.expandInto(file, src.GetMeta().GetName(), done, onPath)

		for _, f := range src.GetStructure().GetFields() {
			copied := &ir.Field{
				Meta:         f.GetMeta(),
				Type:         f.GetType(),
				Key:          f.GetKey(),
				Owned:        f.GetOwned(),
				IncludedFrom: id,
			}
			if f.GetIncludedFrom() != nil {
				copied.IncludedFrom = f.GetIncludedFrom()
			}
			target.Fields = append(target.Fields, copied)
		}
	}
}

// includedMixin resolves an `include` to the declaration it names,
// reporting anything that is not a mixin.
func (l *lowerer) includedMixin(inc *ast.Include) (*ir.Decl, *ir.ID, bool) {
	b, ok := l.scope.lookup(inc.Type.N)
	if !ok || b.kind != bindDecl {
		l.diags.add(inc.P, "undefined mixin: %s", inc.Type.N)
		return nil, nil, false
	}

	decl := l.model.Decl(b.id)
	if decl.GetStructure().GetKind() != ir.StructKind_STRUCT_KIND_MIXIN {
		l.diags.add(inc.P, "%s is not a mixin", inc.Type.N)
		return nil, nil, false
	}
	return decl, b.id, true
}

// buildSatisfaction computes, per class, the declarations that satisfy it.
//
// Satisfaction comes from two places and no others: a declaration that says
// it conforms, and a standalone instance with concrete arguments.
// Including a mixin does not confer it, because the spec says conformance
// is nominal and declared.
//
// The result is closed over the classes a class requires: conforming to
// Auditable when Auditable requires Timestamped means satisfying
// Timestamped too.
//
// Conditional instances are recorded in the model but not expanded here.
// Answering `Satisfying(Auditable)` for `Page<Order>` given
// `instance <T> Auditable<Page<T>> requires Auditable<T>` is a search, and
// it has a phase of its own.
func (l *lowerer) buildSatisfaction() {
	direct := map[int32]map[int32]bool{}

	add := func(class *ir.ID, decl *ir.ID) {
		if !class.Resolved() || !decl.Resolved() {
			return
		}
		if direct[class.GetIndex()] == nil {
			direct[class.GetIndex()] = map[int32]bool{}
		}
		direct[class.GetIndex()][decl.GetIndex()] = true
	}

	for i, decl := range l.model.GetDecls() {
		id := &ir.ID{Index: int32(i), Name: decl.GetMeta().GetName()}
		for _, ref := range conformsOf(decl) {
			add(ref.GetClass(), id)
		}
	}

	for _, inst := range l.model.GetInstances() {
		// A conditional instance has parameters and stands for a family of
		// types rather than one, so it does not name a satisfying
		// declaration.
		if len(inst.GetParams()) > 0 || len(inst.GetRequires()) > 0 {
			continue
		}
		for _, arg := range inst.GetClass().GetArgs() {
			ty := l.model.Type(arg)
			if ty == nil {
				continue
			}
			add(inst.GetClass().GetClass(), ty.GetCtor())
		}
	}

	// Every class gets an entry, not only the ones something conforms to
	// directly: a class nothing names may still be satisfied through a class
	// that requires it, and a backend asking about it deserves an answer
	// rather than a missing key.
	for i, decl := range l.model.GetDecls() {
		if decl.GetClass() == nil {
			continue
		}

		class := int32(i)
		sat := &ir.Satisfaction{Class: &ir.ID{Index: class, Name: decl.GetMeta().GetName()}}
		for _, d := range l.closure(class, direct[class]) {
			sat.Decls = append(sat.Decls, &ir.ID{
				Index: d,
				Name:  l.model.GetDecls()[d].GetMeta().GetName(),
			})
		}
		l.model.Satisfies = append(l.model.Satisfies, sat)
	}
}

// closure returns the declarations satisfying a class, including those that
// satisfy a class requiring it, in table order so the output is stable.
func (l *lowerer) closure(class int32, direct map[int32]bool) []int32 {
	reached := map[int32]bool{}
	for d := range direct {
		reached[d] = true
	}

	// A declaration conforming to a subclass satisfies this one too.
	changed := true
	for changed {
		changed = false
		for i, decl := range l.model.GetDecls() {
			if reached[int32(i)] {
				continue
			}
			for _, ref := range conformsOf(decl) {
				if l.requiresClass(ref.GetClass(), class, map[int32]bool{}) {
					reached[int32(i)] = true
					changed = true
				}
			}
		}
	}

	var out []int32
	for i := range l.model.GetDecls() {
		if reached[int32(i)] {
			out = append(out, int32(i))
		}
	}
	return out
}

// requiresClass reports whether sub is class, or requires it transitively.
func (l *lowerer) requiresClass(sub *ir.ID, class int32, seen map[int32]bool) bool {
	if !sub.Resolved() || seen[sub.GetIndex()] {
		return false
	}
	if sub.GetIndex() == class {
		return true
	}
	seen[sub.GetIndex()] = true

	decl := l.model.Decl(sub)
	for _, ref := range decl.GetClass().GetRequiresClasses() {
		if l.requiresClass(ref.GetClass(), class, seen) {
			return true
		}
	}
	return false
}

// conformsOf returns the classes a declaration says it satisfies.
func conformsOf(decl *ir.Decl) []*ir.ClassRef {
	switch {
	case decl.GetStructure() != nil:
		return decl.GetStructure().GetConforms()
	case decl.GetEnumeration() != nil:
		return decl.GetEnumeration().GetConforms()
	}
	return nil
}

// checkConstraints reports a `requires` clause that an instantiation does
// not satisfy.
//
// `value Envelope<T> requires Auditable<T>` says nothing checkable until
// someone writes `Envelope<Order>`, so the check happens at the use site
// and the diagnostic points there.
//
// An argument that is itself a parameter is not checked: whether it
// satisfies anything depends on the outer instantiation, and checking it
// here would reject `value Outer<T> requires Auditable<T> { e: Envelope<T> }`
// which is exactly what the constraint is for.
func (l *lowerer) checkConstraints() {
	for _, use := range l.model.GetTypes() {
		decl := l.model.Decl(use.GetCtor())
		if decl == nil || len(use.GetArgs()) == 0 {
			continue
		}

		params := decl.Params()
		for _, want := range constraintsOf(decl) {
			l.checkConstraint(use, params, want)
		}
	}
}

func (l *lowerer) checkConstraint(use *ir.Type, params []*ir.Param, want *ir.ClassRef) {
	for _, arg := range want.GetArgs() {
		subject := l.substitute(use, params, arg)
		if subject == nil {
			continue // a parameter, or something that did not resolve
		}
		if l.satisfies(want.GetClass(), subject) {
			continue
		}
		l.diags.add(positionOf(use.GetPosition()), "%s does not satisfy %s, required by %s",
			subject.GetName(), want.GetClass().GetName(), use.GetCtor().GetName())
	}
}

// substitute maps a constraint argument through an instantiation: given
// `Envelope<Order>` and the constraint argument `T`, it returns `Order`.
func (l *lowerer) substitute(use *ir.Type, params []*ir.Param, arg *ir.ID) *ir.ID {
	ty := l.model.Type(arg)
	if ty == nil {
		return nil
	}

	ref := ty.GetParam()
	if ref == nil {
		// A concrete argument in the constraint itself, as in `requires
		// Auditable<Order>`. Check it directly.
		return ty.GetCtor()
	}
	if int(ref.GetIndex()) >= len(params) || int(ref.GetIndex()) >= len(use.GetArgs()) {
		return nil
	}

	actual := l.model.Type(use.GetArgs()[ref.GetIndex()])
	if actual == nil || actual.GetParam() != nil {
		return nil // still a parameter: the outer instantiation decides
	}
	return actual.GetCtor()
}

// satisfies reports whether a declaration satisfies a class, reading the
// index rather than reasoning about instances.
func (l *lowerer) satisfies(class, decl *ir.ID) bool {
	if !class.Resolved() || !decl.Resolved() {
		return true // unresolved names are already reported
	}
	for _, sat := range l.model.GetSatisfies() {
		if sat.GetClass().GetIndex() != class.GetIndex() {
			continue
		}
		for _, d := range sat.GetDecls() {
			if d.GetIndex() == decl.GetIndex() {
				return true
			}
		}
	}
	return false
}

// constraintsOf returns the `requires` clause on a declaration's
// parameters.
func constraintsOf(decl *ir.Decl) []*ir.ClassRef {
	switch {
	case decl.GetStructure() != nil:
		return decl.GetStructure().GetConstraints()
	case decl.GetEnumeration() != nil:
		return decl.GetEnumeration().GetConstraints()
	case decl.GetNewtype() != nil:
		return decl.GetNewtype().GetConstraints()
	case decl.GetClass() != nil:
		return decl.GetClass().GetConstraints()
	}
	return nil
}

func positionOf(p *ir.Position) ast.Position {
	return ast.Position{
		Filename: p.GetFilename(),
		Line:     int(p.GetLine()),
		Col:      int(p.GetColumn()),
	}
}
