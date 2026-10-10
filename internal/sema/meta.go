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
		Position: position(h),
		Order:    int32(order),
	}
	if h.Dep != nil {
		m.Deprecated = &ir.Deprecation{Reason: h.Dep.Reason, Position: position(h.Dep)}
	}
	return m
}

// position records where n was written: where it starts and, when the
// parser recorded it, where it ends.
func position(n ast.Node) *ir.Position {
	start, end := n.Pos(), n.End()
	return &ir.Position{
		Filename:  start.Filename,
		Line:      int32(start.Line),
		Column:    int32(start.Col),
		EndLine:   int32(end.Line),
		EndColumn: int32(end.Col),
	}
}

// kind lowers a kind expression.
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
