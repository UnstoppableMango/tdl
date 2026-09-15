package emit

import "github.com/unstoppablemango/tdl/ir"

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

	// ID is the type table entry this was resolved from, after alias
	// expansion.
	ID  *ir.ID
	Pos *ir.Position
}

// Resolve walks a type reference into a [Ref], or says why this phase
// cannot generate it.
//
// The prelude is replaceable, so this reads the spellings lowering itself
// knows (List, Set, Map, Option, Nullable) rather than anything about what
// the declarations mean. A model that redeclares `primitive string` in its
// own file reaches the same entry, because what matters is that the
// constructor is a primitive named `string`.
//
// A primitive of any name resolves; whether the target has a type for it is
// the backend's decision.
func (s *Session) Resolve(id *ir.ID) (*Ref, error) {
	t := s.Model.Type(id)
	if t == nil {
		return nil, Unsupported(nil, "type %s did not resolve", id.GetName())
	}
	pos := t.GetPosition()

	switch {
	case t.GetParam() != nil:
		// Parameters survive lowering so a language with generics can emit
		// them. No backend emits them yet.
		return nil, Unsupported(pos, "type parameter %s: generics are not generated yet", t.GetParam().GetName())
	case t.GetUnit() != nil:
		return nil, Unsupported(pos, "a unit-typed field has no %s type yet", s.Lang)
	case t.GetExtern() != nil:
		ext := t.GetExtern()
		return nil, Unsupported(pos, "%s is declared in another package, and foreign types are not generated yet", ext.GetName())
	}

	decl := s.Model.Decl(t.GetCtor())
	if decl == nil {
		return nil, Unsupported(pos, "type %s did not resolve", t.GetCtor().GetName())
	}
	name := decl.GetMeta().GetName()

	// An alias is transparent, so it is expanded rather than referenced.
	if a := decl.GetAlias(); a != nil {
		return s.Resolve(a.GetTarget())
	}

	ref := &Ref{ID: id, Pos: pos}
	args := t.GetArgs()
	arg := func(i int) (*Ref, error) {
		if i >= len(args) {
			return nil, Unsupported(pos, "%s is missing a type argument", name)
		}
		return s.Resolve(args[i])
	}

	var err error
	if decl.GetPrimitive() != nil {
		switch name {
		case "List", "Set":
			ref.Form = List
			if name == "Set" {
				ref.Form = Set
			}
			ref.Elem, err = arg(0)
		case "Map":
			ref.Form = Map
			if ref.Key, err = arg(0); err == nil {
				ref.Elem, err = arg(1)
			}
		default:
			ref.Form, ref.Name = Prim, name
		}
		if err != nil {
			return nil, err
		}
		return ref, nil
	}

	// Option and Nullable both mean "may be absent". SyntacticForm records
	// which spelling was written and is deliberately not read: lowering is
	// authoritative, and generating two shapes would make the sugar
	// semantic after the compiler decided it was not.
	if decl.GetEnumeration() != nil && (name == "Option" || name == "Nullable") && len(args) == 1 {
		ref.Form = Option
		if name == "Nullable" {
			ref.Form = Nullable
		}
		if ref.Elem, err = arg(0); err != nil {
			return nil, err
		}
		return ref, nil
	}

	if decl.GetClass() != nil {
		return nil, Unsupported(pos, "%s is a class, and a class is not a %s type", name, s.Lang)
	}
	if decl.GetUnit() != nil {
		return nil, Unsupported(pos, "%s is a unit, and units are not generated yet", name)
	}
	if len(args) > 0 {
		return nil, Unsupported(pos, "%s is applied to type arguments, and generics are not generated yet", name)
	}

	ref.Form, ref.Decl = Named, decl
	return ref, nil
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
