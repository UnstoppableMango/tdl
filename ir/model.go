// Package ir is the resolved semantic model backends consume: what the
// parse tree becomes once names are resolved and sugar is lowered.
//
// The messages are generated from proto/tdl/ir/v1/ir.proto, which is the
// schema plugins read over the wire. The helpers here are hand written.
package ir

// Unresolved is the index an [ID] carries when its name did not resolve to
// a declaration. The name is still recorded, so a diagnostic can say what
// was written.
const Unresolved = -1

// Resolved reports whether id points at a table entry.
func (x *ID) Resolved() bool {
	return x != nil && x.GetIndex() >= 0
}

// Decl returns the declaration id references, or nil if it does not resolve
// to one.
func (x *Model) Decl(id *ID) *Decl {
	if !id.Resolved() || int(id.GetIndex()) >= len(x.GetDecls()) {
		return nil
	}
	return x.GetDecls()[id.GetIndex()]
}

// Type returns the type reference id points at, or nil if it does not
// resolve to one.
func (x *Model) Type(id *ID) *Type {
	if !id.Resolved() || int(id.GetIndex()) >= len(x.GetTypes()) {
		return nil
	}
	return x.GetTypes()[id.GetIndex()]
}

// FindDecl returns the declaration with the given fully qualified name and
// its ID. It reports ok false when nothing carries that name.
func (x *Model) FindDecl(name string) (*Decl, *ID, bool) {
	for i, d := range x.GetDecls() {
		if d.GetMeta().GetName() == name {
			return d, &ID{Index: int32(i), Name: name}, true
		}
	}
	return nil, nil, false
}

// Fields returns the fields of a declaration that has them, and nil for one
// that does not. An enum's fields belong to its variants, not to it.
func (d *Decl) Fields() []*Field {
	if s := d.GetStructure(); s != nil {
		return s.GetFields()
	}
	return nil
}

// Params returns the type parameters of a declaration, or nil.
func (d *Decl) Params() []*Param {
	switch {
	case d.GetAlias() != nil:
		return d.GetAlias().GetParams()
	case d.GetNewtype() != nil:
		return d.GetNewtype().GetParams()
	case d.GetStructure() != nil:
		return d.GetStructure().GetParams()
	case d.GetEnumeration() != nil:
		return d.GetEnumeration().GetParams()
	}
	return nil
}

// IsDeprecated reports whether the node is marked deprecated.
func (m *Meta) IsDeprecated() bool { return m.GetDeprecated() != nil }

// Satisfying returns the declarations satisfying a class, closed over the
// classes that class requires.
//
// A backend applying a class-scoped target directive reads this and never
// reasons about instances. The index covers declarations in this model:
// a foreign type made to satisfy a local class is in the instance table but
// not here, because there is no local ID to name it by.
func (x *Model) Satisfying(class *ID) []*ID {
	for _, sat := range x.GetSatisfies() {
		if sat.GetClass().GetIndex() == class.GetIndex() {
			return sat.GetDecls()
		}
	}
	return nil
}

// SatisfyingTypes returns the instantiated types that satisfy a class
// through a conditional instance, such as `Page<Order>` given
// `instance <T> Auditable<Page<T>> requires Auditable<T>`.
//
// These cannot appear in [Model.Satisfying] because they are types rather
// than declarations: `Page` satisfies nothing on its own.
func (x *Model) SatisfyingTypes(class *ID) []*ID {
	for _, sat := range x.GetSatisfies() {
		if sat.GetClass().GetIndex() == class.GetIndex() {
			return sat.GetTypes()
		}
	}
	return nil
}
