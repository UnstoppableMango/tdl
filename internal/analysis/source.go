package analysis

import (
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// roundtripDirective, on a target block with no arguments, asks the backend
// to carry what its target cannot express in annotations.
const roundtripDirective = "roundtrip"

// Roundtrip reports whether target's block carries the roundtrip directive.
func Roundtrip(m *ir.Model, target string) bool {
	for _, block := range m.GetTargets() {
		if block.GetMeta().GetName() != target {
			continue
		}
		for _, d := range plugin.Directives(target, block.GetDirectives()) {
			if d.GetName() == roundtripDirective && len(d.GetArgs()) == 0 {
				return true
			}
		}
	}
	return false
}

// Carry adds the model's TDL to a, when target's block carries the
// roundtrip directive: the top-level items and imports, and each field's
// source.
func Carry(m *ir.Model, target string, a *plugin.Analysis) {
	if !Roundtrip(m, target) {
		return
	}
	a.Source = ModelSource(m, target)
	for i, d := range m.GetDecls() {
		v := a.GetDecls()[i]
		carryFields(m, d.Fields(), v.GetFields())
		for j, variant := range d.GetEnumeration().GetVariants() {
			carryFields(m, variant.GetFields(), v.GetVariants()[j].GetFields())
		}
	}
}

func carryFields(m *ir.Model, fields []*ir.Field, views []*plugin.MemberView) {
	for i, f := range fields {
		views[i].Source, views[i].Deprecation = FieldSource(m, f)
	}
}

// ModelSource is the model as TDL, with target's block written without its
// roundtrip directive.
func ModelSource(m *ir.Model, target string) *plugin.Source {
	f := unlower.File(m)
	src := &plugin.Source{}
	for _, imp := range f.Imports {
		src.Imports = append(src.Imports, strings.TrimSuffix(ast.Fprint(&ast.File{Imports: []*ast.ImportDecl{imp}}), "\n"))
	}
	for _, d := range f.Decls {
		item := &plugin.Item{}
		if IsDecl(d) {
			item.Name = d.Name()
			item.Conforms = conforms(d)
		}
		if t, ok := d.(*ast.TargetDecl); ok && t.N == target {
			d = WithoutRoundtrip(t)
		}
		if d != nil {
			item.Source = PrintItem(d)
		}
		src.Items = append(src.Items, item)
	}
	return src
}

// FieldSource is a field as TDL without its doc comment or deprecation,
// and its deprecation, or "" when it has none.
func FieldSource(m *ir.Model, f *ir.Field) (source, deprecation string) {
	af := unlower.Field(m, f)
	whole := ast.PrintField(af)
	af.Dep = nil
	source = ast.PrintField(af)
	return source, strings.TrimSuffix(strings.TrimSuffix(whole, source), " ")
}

// conforms is a struct's or an enum's conformance list as TDL.
func conforms(decl ast.Decl) string {
	var refs []*ast.ClassRef
	switch d := decl.(type) {
	case *ast.StructDecl:
		refs = d.Conforms
	case *ast.EnumDecl:
		refs = d.Conforms
	}
	if len(refs) == 0 {
		return ""
	}
	head := ast.PrintDecl(&ast.StructDecl{DeclHead: ast.DeclHead{N: "T"}, Keyword: "type", Conforms: refs})
	head = strings.TrimPrefix(head, "type T: ")
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(head), "{ }"))
}

// IsDecl reports whether a top-level item is a declaration, which has a
// name of its own, rather than an instance or a target block.
func IsDecl(d ast.Decl) bool {
	switch d.(type) {
	case *ast.InstanceDecl, *ast.TargetDecl:
		return false
	}
	return true
}

// WithoutRoundtrip is a target block without the roundtrip directive, or
// nil when nothing else is in it.
func WithoutRoundtrip(t *ast.TargetDecl) ast.Decl {
	out := *t
	out.Entries = slices.DeleteFunc(slices.Clone(t.Entries), func(e *ast.TargetEntry) bool {
		return e.Path == "" && e.Directive != nil && e.Directive.N == roundtripDirective && len(e.Directive.Args) == 0
	})
	if len(out.Entries) == 0 && len(t.Entries) > 0 {
		return nil
	}
	return &out
}

// PrintItem is a top-level item as TDL, doc comment included.
func PrintItem(d ast.Decl) string {
	return strings.TrimSuffix(ast.Fprint(&ast.File{Decls: []ast.Decl{d}}), "\n")
}
