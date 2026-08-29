// Package sema lowers a parse tree to the resolved semantic model in the ir
// package: it resolves names, lowers sugar to prelude types, and interns
// type references.
//
// It is private and free to change. `ir` and `proto` are the compatibility
// surface, not this.
//
// See docs/design/ir-plan.md. Phases 1 and 2 are done: the declaration
// table, the interned type table, scopes, shadowing, recursion rules, and
// diagnostics for names that resolve to nothing. Imports, classes,
// instances, constraints, and targets are still to come.
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
		units:  map[string]bool{},
	}
	l.file = newScope(nil)
	l.scope = l.file
	if file.Package != nil {
		l.model.Package = file.Package.Path
	}

	for _, imp := range file.Imports {
		l.diags.add(imp.P, "imports are not resolved yet: %q", imp.Path)
	}

	// Declarations are collected before anything is lowered, so a reference
	// to a declaration further down the file resolves like one above it.
	l.collect(file)
	l.checkRecursion(file)
	l.lower(file)
	return l.model, l.diags
}

type lowerer struct {
	model  *ir.Model
	byName map[string]int32 // declaration name to index
	types  map[string]int32 // interning key to index
	units  map[string]bool  // declaration names that are units
	file   *scope           // the file's declarations
	scope  *scope           // the scope a type reference resolves against
	diags  Diagnostics
}

// collect fills the declaration table with an empty entry per declaration,
// so every name in the file is known before any type reference is resolved.
//
// Only declarations that introduce a type name are bound. An instance is
// not named, it names the class it is about, and a target block names a
// backend rather than a type; both live in tables of their own and neither
// belongs in the type namespace.
func (l *lowerer) collect(file *ast.File) {
	for i, decl := range file.Decls {
		if !namesAType(decl) {
			l.diags.add(decl.Pos(), "%s is not lowered yet", declKind(decl))
			continue
		}
		name := decl.Name()
		idx := int32(len(l.model.Decls))
		if prev, ok := l.file.bind(name, binding{
			kind: bindDecl,
			id:   &ir.ID{Index: idx, Name: name},
			pos:  decl.Pos(),
		}); !ok {
			l.diags.add(decl.Pos(), "%s is declared twice, first at %s", name, prev.pos)
			continue
		}
		l.byName[name] = idx
		if _, isUnit := decl.(*ast.UnitDecl); isUnit {
			l.units[name] = true
		}
		l.model.Decls = append(l.model.Decls, &ir.Decl{Meta: meta(decl, i)})
	}
}

// namesAType reports whether a declaration introduces a name other
// declarations can refer to.
func namesAType(decl ast.Decl) bool {
	switch decl.(type) {
	case *ast.InstanceDecl, *ast.TargetDecl:
		return false
	}
	return true
}

// declKind names a declaration for a diagnostic about the declaration
// itself rather than about a name in it.
func declKind(decl ast.Decl) string {
	switch d := decl.(type) {
	case *ast.InstanceDecl:
		return "instance " + d.Class.N
	case *ast.TargetDecl:
		return "target " + d.N
	}
	return decl.Name()
}

func (l *lowerer) lower(file *ast.File) {
	for _, decl := range file.Decls {
		if !namesAType(decl) {
			continue
		}
		idx, ok := l.byName[decl.Name()]
		if !ok {
			continue // a duplicate, already reported
		}
		l.setNode(l.model.Decls[idx], decl)
	}
}

// declID returns the ID of a declaration by name, for use as a parameter's
// owner. Every name here has already been collected.
func (l *lowerer) declID(name string) *ir.ID {
	return &ir.ID{Index: l.byName[name], Name: name}
}

// inScope lowers within a scope and restores the previous one, so a
// declaration's type parameters are visible in its body and nowhere else.
func (l *lowerer) inScope(s *scope, f func()) {
	prev := l.scope
	l.scope = s
	f()
	l.scope = prev
}

// setNode fills in the oneof. The generated interface behind it is
// unexported, so the wrapper is assigned here rather than returned.
func (l *lowerer) setNode(out *ir.Decl, decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.PrimitiveDecl:
		out.Node = &ir.Decl_Primitive{Primitive: &ir.Primitive{Kind: kind(d.Kind)}}

	case *ast.AliasDecl:
		l.inScope(l.paramScope(l.declID(d.N), d.Params), func() {
			out.Node = &ir.Decl_Alias{Alias: &ir.Alias{
				Params: l.params(d.Params),
				Target: l.typeRef(d.Target),
			}}
		})

	case *ast.NewtypeDecl:
		l.inScope(l.paramScope(l.declID(d.N), d.Params), func() {
			out.Node = &ir.Decl_Newtype{Newtype: &ir.Newtype{
				Params: l.params(d.Params),
				Base:   l.typeRef(d.Base),
			}}
		})

	case *ast.StructDecl:
		l.inScope(l.paramScope(l.declID(d.N), d.Params), func() {
			out.Node = &ir.Decl_Structure{Structure: &ir.Struct{
				Kind:   structKind(d.Keyword),
				Params: l.params(d.Params),
				Fields: l.fields(d.Members),
			}}
		})

	case *ast.EnumDecl:
		l.inScope(l.paramScope(l.declID(d.N), d.Params), func() {
			e := &ir.Enum{Params: l.params(d.Params)}
			for i, v := range d.Variants {
				e.Variants = append(e.Variants, &ir.Variant{
					Meta:   metaOf(v.N, v.Doc, v.P, v.Dep, i),
					Fields: l.variantFields(v.Fields),
				})
			}
			out.Node = &ir.Decl_Enumeration{Enumeration: e}
		})

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
	var in []*ast.Field
	for _, m := range members {
		if f, ok := m.(*ast.Field); ok {
			in = append(in, f)
		}
		// `include` is expanded in phase 6, alongside class satisfaction.
	}
	return l.variantFields(in)
}

// variantFields lowers a field list, reporting a name used twice. Field
// names share one namespace per body, so the check is the same wherever
// fields appear.
func (l *lowerer) variantFields(in []*ast.Field) []*ir.Field {
	seen := map[string]ast.Position{}
	var fields []*ir.Field
	for i, f := range in {
		if prev, dup := seen[f.N]; dup {
			l.diags.add(f.P, "field %s is declared twice, first at %s", f.N, prev)
			continue
		}
		seen[f.N] = f.P
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
