package sema

import (
	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// metaOf records the source fidelity of a node: its name, doc comment,
// position, deprecation, and where it sat among its siblings.
func metaOf(h *ast.DeclHead, order int) *ir.Meta {
	m := &ir.Meta{
		Name:     h.N,
		Doc:      h.Doc,
		Position: position(h.P),
		Order:    int32(order),
	}
	if h.Dep != nil {
		m.Deprecated = &ir.Deprecation{Reason: h.Dep.Reason, Position: position(h.Dep.P)}
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
