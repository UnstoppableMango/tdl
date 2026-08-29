package sema

import (
	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// meta records the source fidelity of a declaration: its name, doc comment,
// position, deprecation, and where it sat among its siblings.
func meta(decl ast.Decl, order int) *ir.Meta {
	return metaOf(decl.Name(), ast.Doc(decl), decl.Pos(), ast.Deprecated(decl), order)
}

func metaOf(name string, doc []string, pos ast.Position, dep *ast.Deprecation, order int) *ir.Meta {
	m := &ir.Meta{
		Name:     name,
		Doc:      doc,
		Position: position(pos),
		Order:    int32(order),
	}
	if dep != nil {
		m.Deprecated = &ir.Deprecation{Reason: dep.Reason, Position: position(dep.P)}
	}
	return m
}

func position(p ast.Position) *ir.Position {
	return &ir.Position{Filename: p.Filename, Line: int32(p.Line), Column: int32(p.Col)}
}

// kind lowers a kind expression. Arrows associate to the right, which the
// parse tree already reflects.
func kind(k *ast.Kind) *ir.Kind {
	if k == nil {
		return nil
	}

	out := &ir.Kind{Arrow: kind(k.Arrow)}
	switch {
	case k.Paren != nil:
		out.Paren = kind(k.Paren)
	case k.N == "unit":
		out.Atom = ir.KindAtom_KIND_ATOM_UNIT
	default:
		out.Atom = ir.KindAtom_KIND_ATOM_TYPE
	}
	return out
}
