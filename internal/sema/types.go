package sema

import (
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// typeRef lowers a type reference and returns its ID in the type table.
//
// Sugar is lowered here: `[T]` becomes `List<T>`, `{T}` becomes `Set<T>`,
// `{K -> V}` becomes `Map<K, V>`, `T?` becomes `Option<T>`, and `T | null`
// becomes `Nullable<T>`. The lowering is authoritative and the syntactic
// form is recorded alongside it, so a backend that wants to tell `T?` from
// an explicit `Option<T>` can, and one that does not is unaffected.
//
// `T? | null` is `Nullable<Option<T>>`: two entries, each recording the
// form that produced it.
func (l *lowerer) typeRef(t *ast.TypeRef) *ir.ID {
	if t == nil {
		return &ir.ID{Index: ir.Unresolved}
	}

	id := l.coreType(t)
	if t.Optional {
		id = l.intern(&ir.Type{
			Ctor:     l.ctor(preludeOption, t.P),
			Args:     []*ir.ID{id},
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_QUESTION,
			Position: position(t.P),
		})
	}
	if t.Nullable {
		id = l.intern(&ir.Type{
			Ctor:     l.ctor(preludeNullable, t.P),
			Args:     []*ir.ID{id},
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_OR_NULL,
			Position: position(t.P),
		})
	}
	return id
}

func (l *lowerer) coreType(t *ast.TypeRef) *ir.ID {
	switch {
	case t.List != nil:
		return l.intern(&ir.Type{
			Ctor:     l.ctor(preludeList, t.P),
			Args:     []*ir.ID{l.typeRef(t.List)},
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_BRACKETS,
			Position: position(t.P),
		})

	case t.Set != nil:
		return l.intern(&ir.Type{
			Ctor:     l.ctor(preludeSet, t.P),
			Args:     []*ir.ID{l.typeRef(t.Set)},
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_BRACES,
			Position: position(t.P),
		})

	case t.MapKey != nil:
		return l.intern(&ir.Type{
			Ctor:     l.ctor(preludeMap, t.P),
			Args:     []*ir.ID{l.typeRef(t.MapKey), l.typeRef(t.MapValue)},
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_ARROW,
			Position: position(t.P),
		})
	}

	name := t.N
	if t.Qualifier != "" {
		name = t.Qualifier + "." + t.N
	}
	return l.intern(&ir.Type{
		Ctor:     l.ctor(name, t.P),
		Args:     l.typeArgs(t.Args),
		Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
		Position: position(t.P),
	})
}

func (l *lowerer) typeArgs(args []*ast.TypeArg) []*ir.ID {
	var out []*ir.ID
	for _, a := range args {
		if a.Unit != nil {
			// Units are deferred; ir.md says so and says lowering should say
			// so rather than dropping them.
			l.diags.add(a.P, "unit arguments are not lowered yet")
			out = append(out, &ir.ID{Index: ir.Unresolved})
			continue
		}
		out = append(out, l.typeRef(a.Type))
	}
	return out
}

// ctor resolves a constructor name against the declaration table.
//
// A name that matches nothing keeps its text and carries [ir.Unresolved].
// Turning that into a diagnostic is phase 2's job, along with scopes and
// shadowing; until then the tree records what was written.
func (l *lowerer) ctor(name string, _ ast.Position) *ir.ID {
	if idx, ok := l.byName[name]; ok {
		return &ir.ID{Index: idx, Name: name}
	}
	return &ir.ID{Index: ir.Unresolved, Name: name}
}

// intern returns the ID of a type, adding it to the table only if an equal
// type is not already there.
//
// Interning is what makes an ID comparison a type comparison, and it is why
// lowering the same type twice yields one entry.
func (l *lowerer) intern(t *ir.Type) *ir.ID {
	key := internKey(t)
	if idx, ok := l.types[key]; ok {
		return &ir.ID{Index: idx, Name: key}
	}

	idx := int32(len(l.model.Types))
	l.types[key] = idx
	l.model.Types = append(l.model.Types, t)
	return &ir.ID{Index: idx, Name: key}
}

// internKey is the identity of a type: its constructor, its arguments, and
// the form it was written in.
//
// The form is part of the key because it is part of what a backend reads.
// Folding `[T]` and `List<T>` into one entry would throw away the
// distinction the moment a model used both.
func internKey(t *ir.Type) string {
	var b strings.Builder
	b.WriteString(t.GetCtor().GetName())
	if len(t.GetArgs()) > 0 {
		b.WriteByte('<')
		for i, a := range t.GetArgs() {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(a.GetName())
		}
		b.WriteByte('>')
	}
	b.WriteByte('#')
	b.WriteString(strconv.Itoa(int(t.GetWrote())))
	return b.String()
}
