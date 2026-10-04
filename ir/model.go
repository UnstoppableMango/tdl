// Package ir is the resolved semantic model backends consume. The messages
// are generated from proto/tdl/ir/v1/ir.proto; the helpers are hand written.
package ir

// Unresolved is the index an [ID] carries when its name did not resolve.
// The ID keeps the name as written.
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
// its ID, reporting false when there is none.
func (x *Model) FindDecl(name string) (*Decl, *ID, bool) {
	for i, d := range x.GetDecls() {
		if d.GetMeta().GetName() == name {
			return d, &ID{Index: int32(i), Name: name}, true
		}
	}
	return nil, nil, false
}

// Fields returns a struct declaration's fields, or nil. An enum's fields
// belong to its variants.
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

// Constraints returns the `requires` clause on a declaration's type
// parameters, or nil.
func (d *Decl) Constraints() []*ClassRef {
	switch {
	case d.GetNewtype() != nil:
		return d.GetNewtype().GetConstraints()
	case d.GetStructure() != nil:
		return d.GetStructure().GetConstraints()
	case d.GetEnumeration() != nil:
		return d.GetEnumeration().GetConstraints()
	}
	return nil
}

// IsDeprecated reports whether the node is marked deprecated.
func (m *Meta) IsDeprecated() bool { return m.GetDeprecated() != nil }

// Satisfying returns the declarations in this model satisfying a class,
// closed over the classes that class requires.
func (x *Model) Satisfying(class *ID) []*ID {
	return x.satisfaction(class).GetDecls()
}

// satisfaction is the index entry for a class, or nil.
func (x *Model) satisfaction(class *ID) *Satisfaction {
	for _, sat := range x.GetSatisfies() {
		if sat.GetClass().GetIndex() == class.GetIndex() {
			return sat
		}
	}
	return nil
}

// Unit returns the unit an [ID] indexes, or nil when the ID did not
// resolve.
func (x *Model) Unit(id *ID) *Unit {
	if !id.Resolved() || int(id.GetIndex()) >= len(x.GetUnits()) {
		return nil
	}
	return x.GetUnits()[id.GetIndex()]
}

// SatisfyingTypes returns the instantiated types that satisfy a class
// through a conditional instance, such as `Page<Order>` given
// `instance <T> Auditable<Page<T>> requires Auditable<T>`.
func (x *Model) SatisfyingTypes(class *ID) []*ID {
	return x.satisfaction(class).GetTypes()
}

// KindName is how a diagnostic names a literal kind: "a string", "an
// integer", and so on.
func KindName(k LiteralKind) string {
	switch k {
	case LiteralKind_LITERAL_KIND_STRING:
		return "a string"
	case LiteralKind_LITERAL_KIND_INT:
		return "an integer"
	case LiteralKind_LITERAL_KIND_FLOAT:
		return "a float"
	case LiteralKind_LITERAL_KIND_BOOL:
		return "a boolean"
	case LiteralKind_LITERAL_KIND_NAME:
		return "a name"
	case LiteralKind_LITERAL_KIND_REGEX:
		return "a regex"
	case LiteralKind_LITERAL_KIND_LIST:
		return "a list"
	case LiteralKind_LITERAL_KIND_RANGE:
		return "a range"
	case LiteralKind_LITERAL_KIND_UNSPECIFIED:
		return "an unspecified value"
	}
	// A kind from a newer schema.
	return "an unrecognized value"
}
