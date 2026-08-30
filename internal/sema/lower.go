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
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/prelude"
)

// The names sugar lowers to. Lowering knows the spellings because the
// grammar has the sugar; it does not know what they mean, which is why the
// prelude declares them and a replacement may declare them differently.
const (
	preludeList     = "List"
	preludeSet      = "Set"
	preludeMap      = "Map"
	preludeOption   = "Option"
	preludeNullable = "Nullable"
)

// Option configures a lowering.
type Option func(*config)

type config struct {
	preludeName string
	preludeSrc  string
	noPrelude   bool
	loader      Loader
}

// WithPrelude lowers against the given prelude source instead of the
// embedded one. Pointing at a replacement is what makes the prelude
// replaceable rather than built in.
func WithPrelude(name, src string) Option {
	return func(c *config) {
		c.preludeName, c.preludeSrc, c.noPrelude = name, src, false
	}
}

// WithoutPrelude lowers with no prelude at all, which is what compiling a
// prelude itself needs.
func WithoutPrelude() Option {
	return func(c *config) { c.noPrelude = true }
}

// WithLoader supplies the [Loader] that reads imported files. Without one,
// a file that imports anything is a diagnostic rather than a filesystem
// read nobody asked for.
func WithLoader(l Loader) Option {
	return func(c *config) { c.loader = l }
}

// Lower turns a parsed file into a model. It returns the model and every
// diagnostic the pass produced; a non-empty diagnostic list means the model
// is incomplete and no later pass should run against it.
//
// The prelude is loaded into an outer scope, so a file may declare a name
// the prelude already has and its own declaration wins.
func Lower(file *ast.File, opts ...Option) (*ir.Model, Diagnostics) {
	cfg := config{preludeName: prelude.Name, preludeSrc: prelude.Source}
	for _, opt := range opts {
		opt(&cfg)
	}

	l := &lowerer{
		model:   &ir.Model{},
		byName:  map[string]int32{},
		types:   map[string]int32{},
		units:   map[string]bool{},
		aliases: map[string]string{},
		externs: map[string]int32{},
		loader:  cfg.loader,
	}
	l.file = newScope(l.loadPrelude(cfg))
	l.scope = l.file
	if file.Package != nil {
		l.model.Package = file.Package.Path
	}

	// Imports are walked before the file's own declarations, so a `_`
	// import's names are in scope by the time anything refers to them.
	l.loadImports(file)

	// Declarations are collected before anything is lowered, so a reference
	// to a declaration further down the file resolves like one above it.
	l.collect(file)
	l.checkRecursion(file)
	l.lower(file)
	l.expandIncludes(file)
	l.buildSatisfaction()
	l.checkConstraints()
	return l.model, l.diags
}

// loadPrelude parses and lowers the prelude into the model, returning the
// scope its declarations bind in. That scope is the parent of the file's,
// so a file may shadow a prelude name.
//
// The prelude's declarations are merged untagged: to a backend they are
// declarations like any other, which is what lets a replacement prelude
// change what a collection is without every backend learning about it.
func (l *lowerer) loadPrelude(cfg config) *scope {
	if cfg.noPrelude || cfg.preludeSrc == "" {
		return nil
	}

	file, err := parser.Parse(cfg.preludeName, strings.NewReader(cfg.preludeSrc))
	if err != nil {
		l.diags.add(ast.Position{Filename: cfg.preludeName}, "the prelude does not parse: %v", err)
		return nil
	}

	outer := newScope(nil)
	l.file, l.scope = outer, outer
	l.inPrelude = true
	l.collect(file)
	l.lower(file)
	l.inPrelude = false
	return outer
}

type lowerer struct {
	model   *ir.Model
	byName  map[string]int32  // declaration name to index
	types   map[string]int32  // interning key to index
	units   map[string]bool   // declaration names that are units
	aliases map[string]string // import alias to package name
	externs map[string]int32  // "pkg.Name" to index
	loader  Loader
	// inPrelude suppresses the diagnostics that say a declaration form is
	// not lowered yet. The prelude is the compiler's own input, so telling a
	// user that `class Entity` is unimplemented on every file is noise they
	// cannot act on. Real problems in the prelude are still reported.
	inPrelude bool
	file      *scope // the file's declarations
	scope     *scope // the scope a type reference resolves against
	diags     Diagnostics
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
		if _, isInstance := decl.(*ast.InstanceDecl); isInstance {
			continue // instances have their own table, filled while lowering
		}
		if !namesAType(decl) {
			l.deferral(decl.Pos(), "%s is not lowered yet", declLabel(decl))
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
// declarations can refer to. An instance names the class it is about and a
// target block names a backend; neither belongs in the type namespace, and
// both have tables of their own.
func namesAType(decl ast.Decl) bool {
	switch decl.(type) {
	case *ast.InstanceDecl, *ast.TargetDecl:
		return false
	}
	return true
}

// declLabel names a declaration for a diagnostic about the declaration
// itself rather than about a name in it. Saying which form it is lets a
// reader tell a deferral from a mistake, and lets the corpus test say which
// phase each deferral belongs to.
func declLabel(decl ast.Decl) string {
	switch d := decl.(type) {
	case *ast.InstanceDecl:
		return "instance " + d.Class.N
	case *ast.TargetDecl:
		return "target " + d.N
	case *ast.ClassDecl:
		return "class " + d.N
	case *ast.UnitDecl:
		return "unit " + d.N
	}
	return decl.Name()
}

func (l *lowerer) lower(file *ast.File) {
	for i, decl := range file.Decls {
		if inst, ok := decl.(*ast.InstanceDecl); ok {
			l.model.Instances = append(l.model.Instances, l.instance(inst, i))
			continue
		}
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
				Params:      l.params(d.Params),
				Base:        l.typeRef(d.Base),
				Constraints: l.classRefs(d.Requires),
			}}
		})

	case *ast.StructDecl:
		l.inScope(l.paramScope(l.declID(d.N), d.Params), func() {
			out.Node = &ir.Decl_Structure{Structure: &ir.Struct{
				Kind:        structKind(d.Keyword),
				Params:      l.params(d.Params),
				Fields:      l.fields(d.Members),
				Conforms:    l.classRefs(d.Conforms),
				Constraints: l.classRefs(d.Requires),
			}}
		})

	case *ast.ClassDecl:
		l.inScope(l.paramScope(l.declID(d.N), d.Params), func() {
			out.Node = &ir.Decl_Class{Class: l.classNode(d)}
		})

	case *ast.EnumDecl:
		l.inScope(l.paramScope(l.declID(d.N), d.Params), func() {
			e := &ir.Enum{
				Params:      l.params(d.Params),
				Conforms:    l.classRefs(d.Conforms),
				Constraints: l.classRefs(d.Requires),
			}
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
		l.deferral(decl.Pos(), "%s is not lowered yet", declLabel(decl))
	}
}

// deferral reports a form the compiler has not implemented yet, unless the
// declaration came from the prelude.
func (l *lowerer) deferral(pos ast.Position, format string, args ...any) {
	if l.inPrelude {
		return
	}
	l.diags.add(pos, format, args...)
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
