package emit

import (
	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/ir"
)

// The JSON wire convention, shared by every backend that writes or reads a
// model as JSON: a property is named as the model spells it, unless the
// target's `wire` directive renames its key, a fielded
// enum's variants are tagged internally by a discriminant property whose
// value is the variant's wire value, and a newtype is its base value. A
// variant's wire value is its name in the model, unless the target's
// `wire` directive renames it; a fieldless variant is written as its wire
// value.

// DefaultDiscriminant is the property a fielded enum's variants carry when
// no directive names one.
const DefaultDiscriminant = analysis.DefaultDiscriminant

// Discriminant is the property naming a fielded enum's variant: the enum's
// `discriminant` directive, else the target block's, else
// [DefaultDiscriminant]. It reads the directives for the session's target
// rather than the analysis, which a request may have built for another.
func (s *Session) Discriminant(enum *ir.Decl) string {
	return analysis.Discriminant(s.Model, s.Target, enum)
}

// Wire is the directive renaming a variant's wire value or a field's JSON
// key.
const Wire = "wire"

// Tag is a variant's wire value: the discriminant's value for a fielded
// variant, and the string a fieldless one is written as. It is the
// target's `wire` directive on the variant, else the variant's name in the
// model; a `name` directive does not change it.
func (s *Session) Tag(v *ir.Variant) string {
	if w, ok := s.Text(v.GetDirectives(), Wire); ok {
		return w
	}
	return v.GetMeta().GetName()
}

// TagOf is the wire value of the variant of enum named name, for a literal
// naming a variant, such as a `oneOf` argument or a default. A name no
// variant has is returned as written.
func (s *Session) TagOf(enum *ir.Decl, name string) string {
	for _, v := range enum.GetEnumeration().GetVariants() {
		if v.GetMeta().GetName() == name {
			return s.Tag(v)
		}
	}
	return name
}

// CheckTags returns an [InvalidError] at the first `wire` directive that
// makes two of an enum's variants indistinguishable on the wire: one giving
// another variant's name, or the wire value of an earlier variant. A
// backend skips the enum, and the error stops the host writing anything.
func (s *Session) CheckTags(enum *ir.Decl) error {
	variants := enum.GetEnumeration().GetVariants()
	names := make(map[string]bool, len(variants))
	for _, v := range variants {
		names[v.GetMeta().GetName()] = true
	}
	owner := LastSegment(enum.GetMeta().GetName())
	seen := make(map[string]string, len(variants))
	for _, v := range variants {
		vname, tag := v.GetMeta().GetName(), s.Tag(v)
		d, renamed := s.Find(v.GetDirectives(), Wire)
		if renamed && tag != vname && names[tag] {
			return Invalid(d.GetPosition(), "%s.%s's wire value %q is the name of %s.%s, so the two cannot be told apart on the wire",
				owner, vname, tag, owner, tag)
		}
		if other, taken := seen[tag]; taken {
			pos := v.GetMeta().GetPosition()
			if renamed {
				pos = d.GetPosition()
			}
			return Invalid(pos, "%s.%s's wire value %q is also %s.%s's, so the two cannot be told apart on the wire",
				owner, vname, tag, owner, other)
		}
		seen[tag] = vname
	}
	return nil
}

// LoseTag warns when a variant of owner is written as a wire value other
// than its name, which import does not read back as a `wire` directive. It
// reports whether anything is lost.
func (s *Session) LoseTag(owner string, v *ir.Variant) bool {
	tag := s.Tag(v)
	if tag == v.GetMeta().GetName() {
		return false
	}
	d, _ := s.Find(v.GetDirectives(), Wire)
	s.Lose(LossName, d.GetPosition(), "%s.%s's wire value %q does not read back", owner, v.GetMeta().GetName(), tag)
	return true
}

// Key is a field's JSON key: the target's `wire` directive on it, else its
// name in the model. A `name` directive does not change it.
func (s *Session) Key(f *ir.Field) string {
	if w, ok := s.Text(f.GetDirectives(), Wire); ok {
		return w
	}
	return f.GetMeta().GetName()
}

// Property is a field's property name in a target whose properties are
// its JSON keys: its [Session.Key] under a `wire` directive, else a `name`
// directive's argument, else its name in the model. A field's `name`
// renames its JSON key against the wire convention, which
// [Session.WarnPropertyName] says.
func (s *Session) Property(f *ir.Field) string {
	if _, ok := s.Find(f.GetDirectives(), Wire); ok {
		return s.Key(f)
	}
	return s.FieldName(f, func(n string) string { return n })
}

// WarnPropertyName warns, without a loss code, at a `name` directive on a
// field of owner in a target whose properties are its JSON keys: a JSON
// key is a fact of the wire convention, which `wire` states.
func (s *Session) WarnPropertyName(owner string, f *ir.Field) {
	d, ok := s.Find(f.GetDirectives(), "name")
	if !ok {
		return
	}
	field := owner + "." + f.GetMeta().GetName()
	name := d.GetArgs()[0].GetText()
	if key, ok := s.Text(f.GetDirectives(), Wire); ok {
		s.Warn(Unsupported(d.GetPosition(),
			"%s has both a name directive, %q, and a wire directive, %q, and its JSON key is wire's, so name has no effect", field, name, key))
		return
	}
	s.Warn(Unsupported(d.GetPosition(),
		"%s's name directive renames its JSON key to %q, which the wire convention spells as the model does: write wire(%q) to rename the key", field, name, name))
}
