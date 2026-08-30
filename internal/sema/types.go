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

	if t.Qualifier != "" {
		return l.intern(&ir.Type{
			Extern:   l.qualified(t),
			Args:     l.typeArgs(t.Args),
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
			Position: position(t.P),
		})
	}

	// A `_` import merges a dependency's exported names into this scope, and
	// a reference to one is an extern rather than a local declaration.
	if b, ok := l.scope.lookup(t.N); ok && b.kind == bindExtern {
		return l.intern(&ir.Type{
			Extern:   b.id,
			Args:     l.typeArgs(t.Args),
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
			Position: position(t.P),
		})
	}

	// A type parameter shadows a declaration of the same name inside the
	// declaration that declares it, so the scope is consulted before the
	// declaration table.
	if b, ok := l.scope.lookup(t.N); ok && b.kind == bindParam {
		if len(t.Args) > 0 {
			// A higher-kinded parameter applied to arguments, as in `f<T>`.
			return l.intern(&ir.Type{
				Param:    &ir.ParamRef{Name: t.N, Index: b.index, Owner: b.owner},
				Args:     l.typeArgs(t.Args),
				Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
				Position: position(t.P),
			})
		}
		return l.intern(&ir.Type{
			Param:    &ir.ParamRef{Name: t.N, Index: b.index, Owner: b.owner},
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
			Position: position(t.P),
		})
	}

	return l.intern(&ir.Type{
		Ctor:     l.ctor(t.N, t.P),
		Args:     l.typeArgs(t.Args),
		Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
		Position: position(t.P),
	})
}

// typeArgs lowers a `<...>` argument list.
//
// A bare name could be a type or a unit and the parser cannot tell them
// apart, so this is where the question is settled: an argument naming a
// unit declaration is a unit argument. Units are deferred, so that is a
// diagnostic rather than a lowering, which is what ir.md asks for now that
// the parser produces them.
func (l *lowerer) typeArgs(args []*ast.TypeArg) []*ir.ID {
	var out []*ir.ID
	for _, a := range args {
		switch {
		case a.Unit != nil:
			l.diags.add(a.P, "unit arguments are not lowered yet")
			out = append(out, &ir.ID{Index: ir.Unresolved})
		case a.Type != nil && l.namesAUnit(a.Type):
			l.diags.add(a.P, "unit arguments are not lowered yet")
			out = append(out, &ir.ID{Index: ir.Unresolved})
		default:
			out = append(out, l.typeRef(a.Type))
		}
	}
	return out
}

// namesAUnit reports whether a bare argument resolves to a unit
// declaration, which is what makes it a unit argument rather than a type
// argument.
func (l *lowerer) namesAUnit(t *ast.TypeRef) bool {
	if t.N == "" || t.Qualifier != "" || len(t.Args) > 0 {
		return false
	}
	if b, ok := l.scope.lookup(t.N); !ok || b.kind != bindDecl {
		return false
	}
	return l.units[t.N]
}

// qualified resolves `alias.Name` to an extern.
//
// Whether the dependency declares that name is not checked: the reference
// carries the dependency's package to the backend, which either resolves
// it through the import table or maps it with a target directive.
func (l *lowerer) qualified(t *ast.TypeRef) *ir.ID {
	pkg, ok := l.aliases[t.Qualifier]
	if !ok {
		l.diags.add(t.P, "undefined import alias: %s", t.Qualifier)
		return &ir.ID{Index: ir.Unresolved, Name: t.Qualifier + "." + t.N}
	}
	return l.extern(pkg, t.N, t.P)
}

// ctor resolves a constructor name against the enclosing scope.
//
// A name that matches nothing is a diagnostic, and the ID still keeps the
// text so the rest of the pass has something to carry and later output can
// say what was written.
func (l *lowerer) ctor(name string, pos ast.Position) *ir.ID {
	if b, ok := l.scope.lookup(name); ok && b.kind == bindDecl {
		return b.id
	}
	l.diags.add(pos, "undefined: %s", name)
	return &ir.ID{Index: ir.Unresolved, Name: name}
}

// intern returns the ID of a type, adding it to the table only if an equal
// type is not already there.
//
// Interning is what makes an ID comparison a type comparison, and it is why
// lowering the same type twice yields one entry.
//
// The key and the name are different things. The key separates `[T]` from
// `List<T>`, which are the same type written two ways and must stay two
// entries. The name is what a person reads in a diagnostic or a dump, so
// both of those entries are called `List<T>`.
func (l *lowerer) intern(t *ir.Type) *ir.ID {
	key := internKey(t)
	name := typeName(t)

	if idx, ok := l.types[key]; ok {
		return &ir.ID{Index: idx, Name: name}
	}

	idx := int32(len(l.model.Types))
	l.types[key] = idx
	l.model.Types = append(l.model.Types, t)
	return &ir.ID{Index: idx, Name: name}
}

// typeName renders a type the way it is written after lowering: the
// constructor applied to its arguments, which already carry their own
// rendered names.
func typeName(t *ir.Type) string {
	name := t.GetCtor().GetName()
	if p := t.GetParam(); p != nil {
		name = p.GetName()
	}
	if e := t.GetExtern(); e != nil {
		name = e.GetName()
	}
	if len(t.GetArgs()) == 0 {
		return name
	}

	var b strings.Builder
	b.WriteString(name)
	b.WriteByte('<')
	for i, a := range t.GetArgs() {
		if i > 0 {
			b.WriteString(", ")
		}
		if a.GetName() == "" {
			b.WriteByte('?')
			continue
		}
		b.WriteString(a.GetName())
	}
	b.WriteByte('>')
	return b.String()
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
	if p := t.GetParam(); p != nil {
		b.WriteString("param:" + p.GetOwner().GetName() + "." + p.GetName())
	}
	if e := t.GetExtern(); e != nil {
		b.WriteString("extern:" + e.GetName())
	}
	if len(t.GetArgs()) > 0 {
		b.WriteByte('<')
		for i, a := range t.GetArgs() {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.Itoa(int(a.GetIndex())))
		}
		b.WriteByte('>')
	}
	b.WriteByte('#')
	b.WriteString(strconv.Itoa(int(t.GetWrote())))
	return b.String()
}
