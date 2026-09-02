package sema

import (
	"sort"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// dims is a quantity as exponents over base units, keyed by the declaration
// index of the base. It is the working form; ir.Unit holds the frozen one.
type dims map[int32]int32

// lowerUnits resolves every unit declaration in a file.
//
// It runs before the rest of lowering rather than as part of it, for two
// reasons. A unit may be written after the unit that derives from it, so
// file order is not resolution order. And a type argument naming a unit
// needs the answer already computed, since `decimal<N>` interns on what N
// reduces to.
//
// Resolution is on demand with a seen set, which is also what makes a cycle
// reportable at the declaration that closes it.
func (l *lowerer) lowerUnits(file *ast.File) {
	decls := map[string]*ast.UnitDecl{}
	var order []*ast.UnitDecl
	for _, decl := range file.Decls {
		u, ok := decl.(*ast.UnitDecl)
		if !ok {
			continue
		}
		order = append(order, u)
		if _, dup := decls[u.N]; !dup {
			decls[u.N] = u
		}
	}

	for _, u := range order {
		l.resolveUnit(u, decls, map[string]bool{})
	}
}

// resolveUnit lowers one unit declaration, resolving whatever it derives
// from first. The lowered node is the memo: a second call returns it.
func (l *lowerer) resolveUnit(u *ast.UnitDecl, decls map[string]*ast.UnitDecl, seen map[string]bool) (*ir.ID, bool) {
	b, ok := l.scope.lookup(u.N)
	if !ok || b.kind != bindDecl {
		return unresolvedUnit(), false // a duplicate, already reported
	}
	decl := l.model.Decls[b.id.GetIndex()]
	if def := decl.GetUnit(); def != nil {
		return def.GetUnit(), def.GetUnit().Resolved()
	}

	if seen[u.N] {
		l.diags.add(u.Pos(), "unit %s is defined in terms of itself", u.N)
		return unresolvedUnit(), false
	}
	seen[u.N] = true
	defer delete(seen, u.N)

	// A base unit measures itself, which is the dimension every derived one
	// reduces to.
	if u.Expr == nil {
		id := l.internUnit(dims{b.id.GetIndex(): 1}, u.N, u.Pos())
		decl.Node = &ir.Decl_Unit{Unit: &ir.UnitDef{Unit: id, Base: true}}
		return id, true
	}

	acc := dims{}
	if !l.reduce(u.Expr, 1, acc, decls, seen) {
		decl.Node = &ir.Decl_Unit{Unit: &ir.UnitDef{Unit: unresolvedUnit()}}
		return unresolvedUnit(), false
	}
	id := l.internUnit(acc, ast.PrintUnitExpr(u.Expr), u.Expr.P)
	decl.Node = &ir.Decl_Unit{Unit: &ir.UnitDef{Unit: id}}
	return id, true
}

// reduce accumulates a unit expression into acc.
//
// `sign` carries whether the expression sits under a division, so `/` flips
// it and a parenthesized sub-expression inherits it: the `m` in
// `kg/(s*m)` is negative because the group is.
func (l *lowerer) reduce(e *ast.UnitExpr, sign int32, acc dims, decls map[string]*ast.UnitDecl, seen map[string]bool) bool {
	ok := true
	for _, t := range e.Terms {
		s := sign
		if t.Op == "/" {
			s = -sign
		}

		if t.Paren != nil {
			if !l.reduce(t.Paren, s, acc, decls, seen) {
				ok = false
			}
			continue
		}

		base, got := l.dimsOf(t.N, t.P, decls, seen)
		if !got {
			ok = false
			continue
		}
		for id, exp := range base {
			acc[id] += s * int32(t.Exp) * exp
		}
	}
	return ok
}

// dimsOf returns what a name in a unit expression reduces to, lowering the
// declaration it names if nothing has yet.
func (l *lowerer) dimsOf(name string, pos ast.Position, decls map[string]*ast.UnitDecl, seen map[string]bool) (dims, bool) {
	b, ok := l.scope.lookup(name)
	if !ok || b.kind != bindDecl {
		l.diags.add(pos, "undefined unit: %s", name)
		return nil, false
	}
	if !l.units[name] {
		l.diags.add(pos, "%s is not a unit", name)
		return nil, false
	}

	decl := l.model.Decls[b.id.GetIndex()]
	if def := decl.GetUnit(); def != nil {
		return l.dimsOfID(def.GetUnit())
	}
	if u, found := decls[name]; found {
		if id, resolved := l.resolveUnit(u, decls, seen); resolved {
			return l.dimsOfID(id)
		}
		return nil, false
	}

	// Declared, a unit, and not lowered: the declaration came from a pass
	// that is already finished, so nothing here can fix it.
	l.diags.add(pos, "undefined unit: %s", name)
	return nil, false
}

func (l *lowerer) dimsOfID(id *ir.ID) (dims, bool) {
	u := l.model.Unit(id)
	if u == nil {
		return nil, false
	}
	out := dims{}
	for _, dim := range u.GetDims() {
		out[dim.GetBase().GetIndex()] = dim.GetExponent()
	}
	return out, true
}

// internUnit returns the ID of a quantity, adding it to the table only if
// an equal one is not already there.
//
// The key is the reduced dimensions and nothing else, which is what makes
// `decimal<N>` and `decimal<kg*m/s^2>` one type: the spec says they are the
// same, and interning is what makes an ID comparison answer that. `wrote`
// records the first spelling for a reader and does not take part.
func (l *lowerer) internUnit(d dims, wrote string, pos ast.Position) *ir.ID {
	frozen := l.freeze(d)
	key := unitKey(frozen)
	if idx, ok := l.unitKeys[key]; ok {
		return &ir.ID{Index: idx, Name: l.model.Units[idx].GetWrote()}
	}

	idx := int32(len(l.model.Units))
	l.unitKeys[key] = idx
	l.model.Units = append(l.model.Units, &ir.Unit{
		Dims:     frozen,
		Wrote:    wrote,
		Position: position(pos),
	})
	return &ir.ID{Index: idx, Name: wrote}
}

// freeze canonicalizes a working quantity: cancelled dimensions are
// dropped and the rest are ordered by base, so equal quantities have equal
// vectors however they were written.
func (l *lowerer) freeze(d dims) []*ir.Dimension {
	bases := make([]int32, 0, len(d))
	for id, exp := range d {
		if exp != 0 {
			bases = append(bases, id)
		}
	}
	sort.Slice(bases, func(i, j int) bool { return bases[i] < bases[j] })

	out := make([]*ir.Dimension, 0, len(bases))
	for _, id := range bases {
		out = append(out, &ir.Dimension{
			Base:     &ir.ID{Index: id, Name: l.model.Decls[id].GetMeta().GetName()},
			Exponent: d[id],
		})
	}
	return out
}

func unitKey(dims []*ir.Dimension) string {
	var b strings.Builder
	for _, d := range dims {
		b.WriteString(strconv.Itoa(int(d.GetBase().GetIndex())))
		b.WriteByte('^')
		b.WriteString(strconv.Itoa(int(d.GetExponent())))
		b.WriteByte(';')
	}
	return b.String()
}

func unresolvedUnit() *ir.ID {
	return &ir.ID{Index: ir.Unresolved}
}
