// Package sema lowers a parse tree to the resolved semantic model in the ir
// package: it resolves names, lowers sugar to prelude types, and interns
// type references.
//
// It is private and free to change. `ir` and `proto` are the compatibility
// surface, not this.
//
// See docs/design/ir-plan.md. This is phase 1: the declaration table and
// the interned type table. Scopes, shadowing, and the diagnostics for an
// unresolved name are phase 2, so a name that matches nothing lowers to an
// ID carrying the name and [ir.Unresolved].
package sema

import (
	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// Prelude names the sugar lowers to. Phase 4 loads prelude/std.tdl as a
// real compilation unit and these stop being names lowering knows.
const (
	preludeList     = "List"
	preludeSet      = "Set"
	preludeMap      = "Map"
	preludeOption   = "Option"
	preludeNullable = "Nullable"
)

// Lower turns a parsed file into a model. It returns the model and every
// diagnostic the pass produced; a non-empty diagnostic list means the model
// is incomplete and no later pass should run against it.
func Lower(file *ast.File) (*ir.Model, Diagnostics) {
	l := &lowerer{
		model:  &ir.Model{},
		byName: map[string]int32{},
		types:  map[string]int32{},
	}
	if file.Package != nil {
		l.model.Package = file.Package.Path
	}

	// Declarations are collected before anything is lowered, so a reference
	// to a declaration further down the file resolves like one above it.
	l.collect(file)
	l.lower(file)
	return l.model, l.diags
}

type lowerer struct {
	model  *ir.Model
	byName map[string]int32 // declaration name to index
	types  map[string]int32 // interning key to index
	diags  Diagnostics
}

// collect fills the declaration table with an empty entry per declaration,
// so every name in the file is known before any type reference is resolved.
func (l *lowerer) collect(file *ast.File) {
	for i, decl := range file.Decls {
		name := decl.Name()
		if _, dup := l.byName[name]; dup {
			l.diags.add(decl.Pos(), "%s is declared twice", name)
			continue
		}
		l.byName[name] = int32(len(l.model.Decls))
		l.model.Decls = append(l.model.Decls, &ir.Decl{Meta: meta(decl, i)})
	}
}

func (l *lowerer) lower(file *ast.File) {
	for _, decl := range file.Decls {
		idx, ok := l.byName[decl.Name()]
		if !ok {
			continue // a duplicate, already reported
		}
		l.setNode(l.model.Decls[idx], decl)
	}
}

// setNode fills in the oneof. The generated interface behind it is
// unexported, so the wrapper is assigned here rather than returned.
func (l *lowerer) setNode(out *ir.Decl, decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.PrimitiveDecl:
		out.Node = &ir.Decl_Primitive{Primitive: &ir.Primitive{Kind: kind(d.Kind)}}

	case *ast.AliasDecl:
		out.Node = &ir.Decl_Alias{Alias: &ir.Alias{
			Params: l.params(d.Params),
			Target: l.typeRef(d.Target),
		}}

	case *ast.NewtypeDecl:
		out.Node = &ir.Decl_Newtype{Newtype: &ir.Newtype{
			Params: l.params(d.Params),
			Base:   l.typeRef(d.Base),
		}}

	case *ast.StructDecl:
		out.Node = &ir.Decl_Structure{Structure: &ir.Struct{
			Kind:   structKind(d.Keyword),
			Params: l.params(d.Params),
			Fields: l.fields(d.Members),
		}}

	case *ast.EnumDecl:
		e := &ir.Enum{Params: l.params(d.Params)}
		for i, v := range d.Variants {
			e.Variants = append(e.Variants, &ir.Variant{
				Meta:   metaOf(v.N, v.Doc, v.P, v.Dep, i),
				Fields: l.variantFields(v.Fields),
			})
		}
		out.Node = &ir.Decl_Enumeration{Enumeration: e}

	default:
		// Classes, instances, units, and targets arrive in later phases.
		l.diags.add(decl.Pos(), "%s is not lowered yet", decl.Name())
	}
}

func structKind(keyword string) ir.StructKind {
	switch keyword {
	case "entity":
		return ir.StructKind_STRUCT_KIND_ENTITY
	case "value":
		return ir.StructKind_STRUCT_KIND_VALUE
	case "mixin":
		return ir.StructKind_STRUCT_KIND_MIXIN
	}
	return ir.StructKind_STRUCT_KIND_UNSPECIFIED
}

func (l *lowerer) fields(members []ast.Member) []*ir.Field {
	var fields []*ir.Field
	for i, m := range members {
		f, ok := m.(*ast.Field)
		if !ok {
			// `include` is expanded in phase 6, alongside class satisfaction.
			continue
		}
		fields = append(fields, l.field(f, i))
	}
	return fields
}

func (l *lowerer) variantFields(in []*ast.Field) []*ir.Field {
	var fields []*ir.Field
	for i, f := range in {
		fields = append(fields, l.field(f, i))
	}
	return fields
}

func (l *lowerer) field(f *ast.Field, order int) *ir.Field {
	return &ir.Field{
		Meta:  metaOf(f.N, f.Doc, f.P, f.Dep, order),
		Type:  l.typeRef(f.Type),
		Key:   f.Key,
		Owned: f.Owned,
	}
}

func (l *lowerer) params(in []*ast.TypeParam) []*ir.Param {
	var params []*ir.Param
	for _, p := range in {
		params = append(params, &ir.Param{
			Name:     p.N,
			Kind:     kind(p.Kind),
			Position: position(p.P),
		})
	}
	return params
}
