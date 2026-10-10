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
// becomes `Nullable<T>`, each recording the syntactic form it was written
// in. `T? | null` is `Nullable<Option<T>>`.
func (l *lowerer) typeRef(t *ast.TypeRef) *ir.ID {
	if t == nil {
		return &ir.ID{Index: ir.Unresolved}
	}

	id := l.coreType(t)
	if t.Optional {
		id = l.intern(&ir.Type{
			Ctor:     l.ctor(preludeOption, t),
			Args:     []*ir.ID{id},
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_QUESTION,
			Position: position(t),
		})
	}
	if t.Nullable {
		id = l.intern(&ir.Type{
			Ctor:     l.ctor(preludeNullable, t),
			Args:     []*ir.ID{id},
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_OR_NULL,
			Position: position(t),
		})
	}
	return id
}

func (l *lowerer) coreType(t *ast.TypeRef) *ir.ID {
	switch {
	case t.List != nil:
		return l.intern(&ir.Type{
			Ctor:     l.ctor(preludeList, t),
			Args:     []*ir.ID{l.typeRef(t.List)},
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_BRACKETS,
			Position: position(t),
		})

	case t.Set != nil:
		return l.intern(&ir.Type{
			Ctor:     l.ctor(preludeSet, t),
			Args:     []*ir.ID{l.typeRef(t.Set)},
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_BRACES,
			Position: position(t),
		})

	case t.MapKey != nil:
		return l.intern(&ir.Type{
			Ctor:     l.ctor(preludeMap, t),
			Args:     []*ir.ID{l.typeRef(t.MapKey), l.typeRef(t.MapValue)},
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_ARROW,
			Position: position(t),
		})
	}

	if t.Qualifier != "" {
		return l.intern(&ir.Type{
			Extern:   l.qualified(t),
			Args:     l.typeArgs(t.Args),
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
			Position: position(t),
		})
	}

	l.recordLookup(t.P, t.N)

	// A name a `_` import merged in is an extern.
	if b, ok := l.scope.lookup(t.N); ok && b.kind == bindExtern {
		return l.intern(&ir.Type{
			Extern:   b.id,
			Args:     l.typeArgs(t.Args),
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
			Position: position(t),
		})
	}

	// A type parameter shadows a declaration of the same name.
	if b, ok := l.scope.lookup(t.N); ok && b.kind == bindParam {
		if len(t.Args) > 0 {
			// A higher-kinded parameter applied, as in `f<T>`.
			return l.intern(&ir.Type{
				Param:    &ir.ParamRef{Name: t.N, Index: b.index, Owner: b.owner},
				Args:     l.typeArgs(t.Args),
				Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
				Position: position(t),
			})
		}
		return l.intern(&ir.Type{
			Param:    &ir.ParamRef{Name: t.N, Index: b.index, Owner: b.owner},
			Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
			Position: position(t),
		})
	}

	return l.intern(&ir.Type{
		Ctor:     l.ctor(t.N, word(t.P, t.N)),
		Args:     l.typeArgs(t.Args),
		Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
		Position: position(t),
	})
}

// typeArgs lowers a `<...>` argument list. A bare name naming a unit
// declaration is a unit argument; any other bare name is a type.
func (l *lowerer) typeArgs(args []*ast.TypeArg) []*ir.ID {
	var out []*ir.ID
	for _, a := range args {
		switch {
		case a.Unit != nil:
			out = append(out, l.unitArg(a.Unit, a))
		default:
			if id, ok := l.namedUnit(a.Type); ok {
				out = append(out, id)
				continue
			}
			out = append(out, l.typeRef(a.Type))
		}
	}
	return out
}

// unitArg interns a unit expression written in an argument list, such as
// the `kg*m/s^2` in `decimal<kg*m/s^2>`. It shares the unit table entry of
// a declaration measuring the same quantity.
func (l *lowerer) unitArg(e *ast.UnitExpr, at ast.Node) *ir.ID {
	acc := dims{}
	if !l.reduce(e, 1, acc, nil, map[string]bool{}) {
		return &ir.ID{Index: ir.Unresolved}
	}
	return l.unitType(l.internUnit(acc, ast.PrintUnitExpr(e), at), at)
}

// namedUnit interns a bare name that resolves to a unit declaration. A unit
// that failed to resolve is an unresolved unit rather than a type.
func (l *lowerer) namedUnit(t *ast.TypeRef) (*ir.ID, bool) {
	// A modifier makes it a type reference: `kg?` is not a unit.
	if t == nil || t.N == "" || t.Qualifier != "" || len(t.Args) > 0 ||
		t.Optional || t.Nullable {
		return nil, false
	}
	b, ok := l.scope.lookup(t.N)
	if !ok || b.kind != bindDecl {
		return nil, false
	}
	def := l.model.Decl(b.id).GetUnit()
	if def == nil {
		return nil, false
	}

	// A unit never reaches coreType, so it is recorded here.
	l.record(t.P, t.N, b, true)
	if !def.GetUnit().Resolved() {
		return &ir.ID{Index: ir.Unresolved}, true
	}
	return l.unitType(def.GetUnit(), t), true
}

// unitType wraps a unit in a type-table entry, so it can sit in Type.args.
func (l *lowerer) unitType(unit *ir.ID, at ast.Node) *ir.ID {
	return l.intern(&ir.Type{
		Unit:     unit,
		Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
		Position: position(at),
	})
}

// qualified resolves `alias.Name` to an extern. Whether the dependency
// declares that name is not checked; the backend resolves it.
func (l *lowerer) qualified(t *ast.TypeRef) *ir.ID {
	// Recorded with no target, so a cursor on `money.Money` finds nothing
	// rather than a neighboring reference.
	l.record(t.P, t.Qualifier+"."+t.N, binding{}, false)

	pkg, ok := l.aliases[t.Qualifier]
	if !ok {
		l.diags.add(t, "undefined import alias: %s", t.Qualifier)
		return &ir.ID{Index: ir.Unresolved, Name: t.Qualifier + "." + t.N}
	}
	return l.extern(pkg, t.N, t)
}

// ctor resolves a constructor name against the enclosing scope. An
// undefined name is a diagnostic and an unresolved ID that keeps the text.
func (l *lowerer) ctor(name string, at ast.Node) *ir.ID {
	if b, ok := l.scope.lookup(name); ok && b.kind == bindDecl {
		return b.id
	}
	l.diags.add(at, "undefined: %s", name)
	return &ir.ID{Index: ir.Unresolved, Name: name}
}

// intern returns the ID of a type, adding it to the table only if an equal
// type is not already there. The key separates `[T]` from `List<T>`; the
// name, which a person reads, calls both `List<T>`.
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

// typeName renders a type as its constructor applied to its arguments.
func typeName(t *ir.Type) string {
	name := t.GetCtor().GetName()
	if p := t.GetParam(); p != nil {
		name = p.GetName()
	}
	if e := t.GetExtern(); e != nil {
		name = e.GetName()
	}
	if u := t.GetUnit(); u != nil {
		name = u.GetName()
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
// the form it was written in, which a backend reads.
func internKey(t *ir.Type) string {
	var b strings.Builder
	b.WriteString(t.GetCtor().GetName())
	if p := t.GetParam(); p != nil {
		b.WriteString("param:" + p.GetOwner().GetName() + "." + p.GetName())
	}
	if e := t.GetExtern(); e != nil {
		b.WriteString("extern:" + e.GetName())
	}
	if u := t.GetUnit(); u != nil {
		// The index, not the spelling, so `decimal<N>` and
		// `decimal<kg*m/s^2>` are one type.
		b.WriteString("unit:" + strconv.Itoa(int(u.GetIndex())))
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
