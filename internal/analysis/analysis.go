// Package analysis derives from a model the facts every backend serving
// one target would otherwise compute for itself: each type reference
// resolved to the prelude's shapes, and what each declaration names.
//
// tdl ships the result in [plugin.Request.Analysis], so a plugin in any
// language reads it. See docs/design/plugin-sdk.md.
package analysis

import (
	"slices"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
	"github.com/unstoppablemango/tdl/prelude"
)

// DefaultDiscriminant is the property a fielded enum's variants carry when
// no directive names one.
const DefaultDiscriminant = "kind"

// Analyze derives the analysis of a model for the target block named
// target.
func Analyze(m *ir.Model, target string) *plugin.Analysis {
	a := &plugin.Analysis{
		Types: make([]*plugin.TypeView, len(m.GetTypes())),
		Decls: make([]*plugin.DeclView, len(m.GetDecls())),
	}
	r := resolver{m: m, views: a.Types}
	for i := range a.Types {
		r.view(int32(i))
	}
	for i, d := range m.GetDecls() {
		a.Decls[i] = declView(m, target, &r, d)
	}
	return a
}

type resolver struct {
	m     *ir.Model
	views []*plugin.TypeView
	// busy marks the entries being resolved, so an alias naming itself
	// ends rather than recursing forever.
	busy map[int32]bool
}

// view returns the view of the type table entry at i, computing it on
// first use.
func (r *resolver) view(i int32) *plugin.TypeView {
	if v := r.views[i]; v != nil {
		return v
	}
	if r.busy == nil {
		r.busy = map[int32]bool{}
	}
	if r.busy[i] {
		t := r.m.GetTypes()[i]
		return &plugin.TypeView{Id: &ir.ID{Index: i}, Refusal: refusal(plugin.Refusal_REASON_UNRESOLVED, t.GetCtor().GetName(), t)}
	}
	r.busy[i] = true
	v := r.resolve(i)
	v.References = references(r.m, &ir.ID{Index: i})
	delete(r.busy, i)
	r.views[i] = v
	return v
}

// expand is the entry id names after alias expansion, or id itself when it
// names no entry.
func (r *resolver) expand(id *ir.ID) *ir.ID {
	if r.m.Type(id) == nil {
		return id
	}
	return r.view(id.GetIndex()).GetId()
}

// resolve describes one entry, following the rules lowering uses for the
// prelude's spellings. The checks run in the order a backend walking the
// type would meet them, so the first refusal is the one it would report.
func (r *resolver) resolve(i int32) *plugin.TypeView {
	t := r.m.GetTypes()[i]
	v := &plugin.TypeView{Id: &ir.ID{Index: i}}

	switch {
	case t.GetParam() != nil:
		v.Form, v.Name = plugin.Form_FORM_PARAM, t.GetParam().GetName()
		v.Refusal = refusal(plugin.Refusal_REASON_PARAM, v.Name, t)
		return v
	case t.GetUnit() != nil:
		v.Form = plugin.Form_FORM_UNIT
		v.Refusal = refusal(plugin.Refusal_REASON_UNIT, "", t)
		return v
	case t.GetExtern() != nil:
		v.Form = plugin.Form_FORM_EXTERN
		if j := int(t.GetExtern().GetIndex()); j >= 0 && j < len(r.m.GetExterns()) {
			v.Extern = t.GetExtern()
		}
		return v
	}

	decl := r.m.Decl(t.GetCtor())
	if decl == nil {
		v.Refusal = refusal(plugin.Refusal_REASON_UNRESOLVED, t.GetCtor().GetName(), t)
		return v
	}
	name := decl.GetMeta().GetName()

	if a := decl.GetAlias(); a != nil {
		if r.m.Type(a.GetTarget()) == nil {
			v.Refusal = refusal(plugin.Refusal_REASON_UNRESOLVED, a.GetTarget().GetName(), nil)
			return v
		}
		target := r.view(a.GetTarget().GetIndex())
		return &plugin.TypeView{
			Form:    target.GetForm(),
			Id:      target.GetId(),
			Name:    target.GetName(),
			Elem:    target.GetElem(),
			Key:     target.GetKey(),
			Decl:    target.GetDecl(),
			Extern:  target.GetExtern(),
			Refusal: target.GetRefusal(),
		}
	}

	args := t.GetArgs()
	// arg sets *to the i-th argument, or refuses v for its absence.
	arg := func(n int, to **ir.ID) bool {
		if n >= len(args) {
			v.Refusal = refusal(plugin.Refusal_REASON_MISSING_ARG, name, t)
			return false
		}
		*to = r.expand(args[n])
		return true
	}

	if decl.GetPrimitive() != nil {
		switch name {
		case "List", "Set":
			v.Form = plugin.Form_FORM_LIST
			if name == "Set" {
				v.Form = plugin.Form_FORM_SET
			}
			arg(0, &v.Elem)
		case "Map":
			v.Form = plugin.Form_FORM_MAP
			if arg(0, &v.Key) {
				arg(1, &v.Elem)
			}
		default:
			v.Form, v.Name = plugin.Form_FORM_PRIMITIVE, name
		}
		return v
	}

	// SyntacticForm is deliberately not read: the sugar is not semantic.
	if decl.GetEnumeration() != nil && (name == "Option" || name == "Nullable") && len(args) == 1 {
		v.Form = plugin.Form_FORM_OPTION
		if name == "Nullable" {
			v.Form = plugin.Form_FORM_NULLABLE
		}
		arg(0, &v.Elem)
		return v
	}

	v.Form, v.Decl = plugin.Form_FORM_NAMED, t.GetCtor()
	switch {
	case decl.GetClass() != nil:
		v.Refusal = refusal(plugin.Refusal_REASON_CLASS, name, t)
	case decl.GetUnit() != nil:
		v.Refusal = refusal(plugin.Refusal_REASON_UNIT_DECL, name, t)
	case len(args) > 0:
		v.Refusal = refusal(plugin.Refusal_REASON_TYPE_ARGS, name, t)
	}
	return v
}

func refusal(reason plugin.Refusal_Reason, subject string, t *ir.Type) *plugin.Refusal {
	return &plugin.Refusal{Reason: reason, Subject: subject, Position: t.GetPosition()}
}

// references returns the declarations the type references ids name: each
// argument's before its constructor's, with aliases expanded, each once.
func references(m *ir.Model, ids ...*ir.ID) []*ir.ID {
	var out []*ir.ID
	seen := map[int32]bool{}
	var walk func(id *ir.ID)
	walk = func(id *ir.ID) {
		t := m.Type(id)
		if t == nil {
			return
		}
		for _, arg := range t.GetArgs() {
			walk(arg)
		}
		decl := m.Decl(t.GetCtor())
		if decl == nil || seen[t.GetCtor().GetIndex()] {
			return
		}
		seen[t.GetCtor().GetIndex()] = true
		if a := decl.GetAlias(); a != nil {
			walk(a.GetTarget())
			return
		}
		out = append(out, t.GetCtor())
	}
	for _, id := range ids {
		walk(id)
	}
	return out
}

func declView(m *ir.Model, target string, r *resolver, d *ir.Decl) *plugin.DeclView {
	v := &plugin.DeclView{
		Own:        IsOwn(d),
		References: references(m, BodyTypes(d)...),
		Words:      Words(lastSegment(d.GetMeta().GetName())),
	}
	for _, f := range d.Fields() {
		v.Fields = append(v.Fields, fieldView(f))
	}
	if e := d.GetEnumeration(); e != nil {
		v.Fielded = Fielded(e)
		v.Discriminant = Discriminant(m, target, d)
		for _, variant := range e.GetVariants() {
			v.Variants = append(v.Variants, member(variant.GetMeta().GetName(), variant.GetFields()))
		}
	}
	if n := d.GetNewtype(); n != nil {
		v.Underlying = underlying(m, r, n.GetBase())
		v.Constraints = constraintViews(n.GetValueConstraints())
	}
	return v
}

func member(name string, fields []*ir.Field) *plugin.MemberView {
	v := &plugin.MemberView{Words: Words(name)}
	for _, f := range fields {
		v.Fields = append(v.Fields, fieldView(f))
	}
	return v
}

func fieldView(f *ir.Field) *plugin.MemberView {
	return &plugin.MemberView{Words: Words(f.GetMeta().GetName()), Constraints: constraintViews(f.GetConstraints())}
}

// BodyTypes are the type references a declaration's body names: its
// fields, its variants' fields, a newtype's base, or an alias's target.
func BodyTypes(d *ir.Decl) []*ir.ID {
	var ids []*ir.ID
	switch {
	case d.GetStructure() != nil:
		for _, f := range d.Fields() {
			ids = append(ids, f.GetType())
		}
	case d.GetEnumeration() != nil:
		for _, v := range d.GetEnumeration().GetVariants() {
			for _, f := range v.GetFields() {
				ids = append(ids, f.GetType())
			}
		}
	case d.GetNewtype() != nil:
		ids = append(ids, d.GetNewtype().GetBase())
	case d.GetAlias() != nil:
		ids = append(ids, d.GetAlias().GetTarget())
	}
	return ids
}

// underlying follows a newtype's base through every newtype it names to
// the first type that is not one, or nil when the chain wraps itself.
func underlying(m *ir.Model, r *resolver, base *ir.ID) *ir.ID {
	var seen []int32
	for m.Type(base) != nil {
		v := r.view(base.GetIndex())
		decl := m.Decl(v.GetDecl())
		if v.GetForm() != plugin.Form_FORM_NAMED || decl.GetNewtype() == nil {
			return v.GetId()
		}
		if slices.Contains(seen, v.GetDecl().GetIndex()) {
			return nil
		}
		seen = append(seen, v.GetDecl().GetIndex())
		base = decl.GetNewtype().GetBase()
	}
	return nil
}

// IsOwn reports whether a declaration is the model's rather than the
// prelude's, which is merged into the declaration table untagged.
//
// The embedded prelude's filename is exactly [prelude.Name], so a user's
// `std.tdl` in any directory is still generated. A replacement prelude
// passed to `sema.WithPrelude` is not recognized and is generated too.
func IsOwn(d *ir.Decl) bool {
	return d.GetMeta().GetPosition().GetFilename() != prelude.Name
}

// Fielded reports whether any variant of an enum carries fields, making it
// a sum type rather than a plain enum.
func Fielded(e *ir.Enum) bool {
	for _, v := range e.GetVariants() {
		if len(v.GetFields()) > 0 {
			return true
		}
	}
	return false
}

// Discriminant is the property naming a fielded enum's variant on the JSON
// wire: the enum's `discriminant` directive for target, else the target
// block's, else [DefaultDiscriminant].
func Discriminant(m *ir.Model, target string, enum *ir.Decl) string {
	if d, ok := find(target, enum.GetDirectives(), "discriminant"); ok {
		return d.GetArgs()[0].GetText()
	}
	for _, block := range m.GetTargets() {
		if block.GetMeta().GetName() != target {
			continue
		}
		if d, ok := find(target, block.GetDirectives(), "discriminant"); ok {
			return d.GetArgs()[0].GetText()
		}
	}
	return DefaultDiscriminant
}

// find returns target's directive with this name carrying at least one
// argument.
func find(target string, all []*ir.Directive, name string) (*ir.Directive, bool) {
	for _, d := range plugin.Directives(target, all) {
		if d.GetName() == name && len(d.GetArgs()) > 0 {
			return d, true
		}
	}
	return nil, false
}
