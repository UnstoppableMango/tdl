package emit

import "github.com/unstoppablemango/tdl/ir"

// The JSON wire convention, shared by every backend that writes or reads a
// model as JSON: a property is named as the model spells it, a fielded
// enum's variants are tagged internally by a discriminant property whose
// value is the variant's name in the model, and a newtype is its base value.

// DefaultDiscriminant is the property a fielded enum's variants carry when
// no directive names one.
const DefaultDiscriminant = "kind"

// Discriminant is the property naming a fielded enum's variant: the enum's
// `discriminant` directive, else the target block's, else
// [DefaultDiscriminant].
func (s *Session) Discriminant(enum *ir.Decl) string {
	if t, ok := s.Text(enum.GetDirectives(), "discriminant"); ok {
		return t
	}
	if d, ok := s.Block("discriminant"); ok {
		return d.GetArgs()[0].GetText()
	}
	return DefaultDiscriminant
}

// Tag is a variant's discriminant value: its name in the model, which a
// `name` directive does not change, so every target agrees on the bytes.
func Tag(v *ir.Variant) string {
	return v.GetMeta().GetName()
}
