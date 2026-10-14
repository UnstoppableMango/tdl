package emit

import (
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Form is which of the prelude's shapes a type reference resolved to.
type Form int

const (
	// Prim is a primitive that is not a collection: `string`, `int`, and so
	// on. [Ref.Name] says which.
	Prim Form = iota + 1
	List
	Set
	Map
	// Option is `T?`.
	Option
	// Nullable is `T | null`.
	Nullable
	// Named is a struct, an enum, or a newtype. [Ref.Decl] is it.
	Named
	// Extern is a declaration from another package; [Ref.Name] is its
	// qualified name. Only a [Session] with Externs set resolves one.
	Extern
)

// Ref is a type reference with its aliases expanded and its prelude
// spellings recognized.
type Ref struct {
	Form Form

	// Name is the primitive's name when Form is [Prim].
	Name string

	// Elem is the element of a List or a Set, the value of a Map, and what
	// an Option or a Nullable wraps.
	Elem *Ref
	// Key is the key of a Map.
	Key *Ref

	// Decl is the declaration when Form is [Named].
	Decl *ir.Decl

	// Extern is the extern when Form is [Extern].
	Extern *ir.Extern

	// ID is the type table entry, after alias expansion.
	ID  *ir.ID
	Pos *ir.Position
}

// Resolve walks a type reference into a [Ref], or returns an
// [UnsupportedError] saying why it cannot be generated.
//
// It reads the request's analysis, which matches the spellings lowering
// knows (List, Set, Map, Option, Nullable) by name, since the prelude is
// replaceable. A primitive of any name resolves; whether the target has a
// type for it is the backend's decision.
func (s *Session) Resolve(id *ir.ID) (*Ref, error) {
	if s.Model.Type(id) == nil {
		return nil, Unsupported(nil, "type %s did not resolve", id.GetName())
	}
	v := s.analysis().GetTypes()[id.GetIndex()]
	if r := v.GetRefusal(); r != nil && r.GetReason() != plugin.Refusal_REASON_MISSING_ARG {
		return nil, s.refuse(r)
	}

	ref := &Ref{ID: id}
	if v.GetId().GetIndex() != id.GetIndex() {
		ref.ID = v.GetId()
	}
	t := s.Model.Type(ref.ID)
	ref.Pos = t.GetPosition()

	// part resolves an element or a key, which a missing type argument
	// leaves unset.
	part := func(id *ir.ID) (*Ref, error) {
		if id == nil {
			return nil, s.refuse(v.GetRefusal())
		}
		return s.Resolve(id)
	}

	var err error
	switch v.GetForm() {
	case plugin.Form_FORM_EXTERN:
		ext := t.GetExtern()
		if !s.Externs || v.GetExtern() == nil {
			return nil, Unsupported(ref.Pos, "%s is declared in another package, and foreign types are not generated yet", ext.GetName())
		}
		ref.Form, ref.Name, ref.Extern = Extern, ext.GetName(), s.Model.GetExterns()[v.GetExtern().GetIndex()]
	case plugin.Form_FORM_PRIMITIVE:
		ref.Form, ref.Name = Prim, v.GetName()
	case plugin.Form_FORM_LIST, plugin.Form_FORM_SET:
		ref.Form = List
		if v.GetForm() == plugin.Form_FORM_SET {
			ref.Form = Set
		}
		ref.Elem, err = part(v.GetElem())
	case plugin.Form_FORM_MAP:
		ref.Form = Map
		if ref.Key, err = part(v.GetKey()); err == nil {
			ref.Elem, err = part(v.GetElem())
		}
	case plugin.Form_FORM_OPTION, plugin.Form_FORM_NULLABLE:
		ref.Form = Option
		if v.GetForm() == plugin.Form_FORM_NULLABLE {
			ref.Form = Nullable
		}
		ref.Elem, err = part(v.GetElem())
	default:
		ref.Form, ref.Decl = Named, s.Model.Decl(v.GetDecl())
	}
	if err != nil {
		return nil, err
	}
	return ref, nil
}

// refuse words a refusal from the analysis as this target's warning.
func (s *Session) refuse(r *plugin.Refusal) error {
	pos, subject := r.GetPosition(), r.GetSubject()
	switch r.GetReason() {
	case plugin.Refusal_REASON_PARAM:
		return Unsupported(pos, "type parameter %s: generics are not generated yet", subject)
	case plugin.Refusal_REASON_UNIT:
		return Unsupported(pos, "a unit-typed field has no %s type yet", s.Lang)
	case plugin.Refusal_REASON_CLASS:
		return Unsupported(pos, "%s is a class, and a class is not a %s type", subject, s.Lang)
	case plugin.Refusal_REASON_UNIT_DECL:
		return Unsupported(pos, "%s is a unit, and units are not generated yet", subject)
	case plugin.Refusal_REASON_TYPE_ARGS:
		return Unsupported(pos, "%s is applied to type arguments, and generics are not generated yet", subject)
	case plugin.Refusal_REASON_MISSING_ARG:
		return Unsupported(pos, "%s is missing a type argument", subject)
	default:
		return Unsupported(pos, "type %s did not resolve", subject)
	}
}

// Expand follows a newtype to what it wraps, for a target with no distinct
// type to give it. Any other reference is returned as it is.
func (s *Session) Expand(r *Ref) (*Ref, error) {
	seen := map[*ir.Decl]bool{}
	for r.Form == Named && r.Decl.GetNewtype() != nil {
		if seen[r.Decl] {
			return nil, Unsupported(r.Pos, "%s wraps itself", r.Decl.GetMeta().GetName())
		}
		seen[r.Decl] = true

		next, err := s.Resolve(r.Decl.GetNewtype().GetBase())
		if err != nil {
			return nil, err
		}
		r = next
	}
	return r, nil
}
