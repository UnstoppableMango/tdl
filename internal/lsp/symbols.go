package lsp

import (
	"context"

	"go.lsp.dev/protocol"

	"github.com/unstoppablemango/tdl/ast"
)

// DocumentSymbol is the outline of a document: each declaration, with its
// fields and variants as children. It reads the tree, so a file that does
// not parse still outlines what the parser recovered.
func (s *Server) DocumentSymbol(_ context.Context, params *protocol.DocumentSymbolParams) (protocol.DocumentSymbolResult, error) {
	doc := s.store.get(params.TextDocument.URI.FsPath())
	if doc == nil {
		return nil, nil
	}

	snap := s.store.analyze(doc, s.opts...)
	out := protocol.DocumentSymbolSlice{}
	if snap.file == nil {
		return out, nil
	}
	for _, decl := range snap.file.Decls {
		out = append(out, declSymbol(snap.index, decl))
	}
	return out, nil
}

func declSymbol(x *lineIndex, decl ast.Decl) protocol.DocumentSymbol {
	head := decl.Head()
	kind, detail, end := protocol.SymbolKindClass, "", ast.Position{}
	var children []protocol.DocumentSymbol

	switch d := decl.(type) {
	case *ast.PrimitiveDecl:
		detail = "primitive"
	case *ast.AliasDecl:
		detail = "= " + ast.PrintTypeRef(d.Target)
	case *ast.NewtypeDecl:
		detail, end = ast.PrintTypeRef(d.Base), d.Rbrace
	case *ast.StructDecl:
		kind, detail, end = protocol.SymbolKindStruct, d.Keyword, d.Rbrace
		children = memberSymbols(x, d.Members)
	case *ast.EnumDecl:
		kind, end = protocol.SymbolKindEnum, d.Rbrace
		for _, v := range d.Variants {
			children = append(children, variantSymbol(x, v))
		}
	case *ast.ClassDecl:
		kind, detail, end = protocol.SymbolKindInterface, "class", d.Rbrace
		children = memberSymbols(x, d.Members)
	case *ast.InstanceDecl:
		kind, detail, end = protocol.SymbolKindObject, "instance", d.Rbrace
	case *ast.UnitDecl:
		kind, detail = protocol.SymbolKindConstant, "unit"
		if d.Expr != nil {
			detail = "= " + ast.PrintUnitExpr(d.Expr)
		}
	case *ast.TargetDecl:
		kind, detail, end = protocol.SymbolKindNamespace, "target for "+d.For, d.Rbrace
	}
	return symbol(x, head, kind, detail, end, children)
}

// memberSymbols is a body's fields; an include declares nothing.
func memberSymbols(x *lineIndex, members []ast.Member) []protocol.DocumentSymbol {
	var out []protocol.DocumentSymbol
	for _, m := range members {
		if f, ok := m.(*ast.Field); ok {
			out = append(out, fieldSymbol(x, f))
		}
	}
	return out
}

func fieldSymbol(x *lineIndex, f *ast.Field) protocol.DocumentSymbol {
	return symbol(x, &f.DeclHead, protocol.SymbolKindField, ast.PrintTypeRef(f.Type), f.Rbrace, nil)
}

func variantSymbol(x *lineIndex, v *ast.Variant) protocol.DocumentSymbol {
	var children []protocol.DocumentSymbol
	for _, f := range v.Fields {
		children = append(children, fieldSymbol(x, f))
	}
	return symbol(x, &v.DeclHead, protocol.SymbolKindEnumMember, "", v.Rbrace, children)
}

// symbol builds one entry. Its range runs from the node's start to its
// closing brace, or to the end of its line when it has none; the selection
// is the name.
func symbol(x *lineIndex, head *ast.DeclHead, kind protocol.SymbolKind, detail string, end ast.Position, children []protocol.DocumentSymbol) protocol.DocumentSymbol {
	start := head.P.Offset
	stop := x.lineEnd(head.P.Line - 1)
	if end.Line > 0 {
		stop = end.Offset + 1
	}
	rng := x.span(start, stop-start)

	sel, ok := x.nameSpan(head.P, head.N)
	if !ok || !within(rng, sel.Start) || !within(rng, sel.End) {
		sel = protocol.Range{Start: rng.Start, End: rng.Start}
	}

	sym := protocol.DocumentSymbol{
		Name:           head.N,
		Kind:           kind,
		Range:          rng,
		SelectionRange: sel,
		Children:       children,
	}
	if detail != "" {
		sym.Detail = &detail
	}
	if head.Dep != nil {
		sym.Tags = []protocol.SymbolTag{protocol.SymbolTagDeprecated}
	}
	return sym
}
