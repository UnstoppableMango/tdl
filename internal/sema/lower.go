// Package sema lowers a parse tree to the resolved semantic model in the ir
// package: it resolves names, lowers sugar to prelude types, and interns
// type references. It is private and free to change; see
// docs/design/ir-plan.md for what each pass adds.
package sema

import (
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/prelude"
)

// The names sugar lowers to. The prelude declares what they mean.
const (
	preludeList     = "List"
	preludeSet      = "Set"
	preludeMap      = "Map"
	preludeOption   = "Option"
	preludeNullable = "Nullable"
	preludeEntity   = "Entity"
)

// Option configures a lowering.
type Option func(*config)

type config struct {
	preludeName string
	preludeSrc  string
	loader      Loader
	refs        *References
	checkDeps   bool                // check an instance for an imported type against its dependency
	deps        map[string]*lowerer // shared by the lowerers of one lowering; see [lowerer.dependency]
}

// WithPrelude lowers against the given prelude source instead of the
// embedded one.
func WithPrelude(name, src string) Option {
	return func(c *config) {
		c.preludeName, c.preludeSrc = name, src
	}
}

// WithLoader supplies the [Loader] that reads imported files. Without one,
// any import is a diagnostic.
func WithLoader(l Loader) Option {
	return func(c *config) { c.loader = l }
}

// Lower turns a parsed file into a model and every diagnostic the pass
// produced. A non-empty diagnostic list means the model is incomplete.
//
// The prelude is loaded into an outer scope, so a file's declaration
// shadows a prelude name.
func Lower(file *ast.File, opts ...Option) (*ir.Model, Diagnostics) {
	cfg := config{preludeName: prelude.Name, preludeSrc: prelude.Source, checkDeps: true, deps: map[string]*lowerer{}}
	for _, opt := range opts {
		opt(&cfg)
	}
	l := lowerFile(file, cfg)
	return l.model, l.diags
}

// lowerFile runs every pass over file, returning the lowerer so a caller can
// read its scopes as well as its model.
func lowerFile(file *ast.File, cfg config) *lowerer {
	l := &lowerer{
		model:    &ir.Model{},
		types:    map[string]int32{},
		unitKeys: map[string]int32{},
		aliases:  map[string]string{},
		externs:  map[string]int32{},
		depDecls: map[string][]*ir.Directive{},
		depFiles: map[string]*ast.File{},
		deps:     cfg.deps,
		loader:   cfg.loader,
		cfg:      cfg,
	}
	l.file = newScope(l.loadPrelude(cfg))
	l.scope = l.file
	l.preludeDecls = len(l.model.GetDecls())

	// Recording starts after the prelude, so its references are not indexed.
	l.refs = cfg.refs
	if file.Package != nil {
		l.model.Package = file.Package.Path
		l.model.Doc = file.Package.Doc
	}

	// Imports come first, so a `_` import's names are in scope.
	l.loadImports(file)

	// Collecting first lets a reference resolve a declaration below it.
	l.collect(file)
	l.lowerUnits(file)
	l.lower(file)
	l.expandIncludes(file)
	l.accumulateConstraints()
	l.resolveNames()
	l.validateInstances()
	l.buildSatisfaction()
	l.checkClassFields()
	l.markEntities()
	l.checkRecursion(file)
	l.searchSatisfaction()
	l.checkTypePositions()
	l.checkConstraints()
	l.lowerTargets(file)

	if l.refs != nil {
		l.refs.sort()
	}
	return l
}

// loadPrelude parses and lowers the prelude into the model, returning the
// scope its declarations bind in, the parent of the file's. Its
// declarations are merged untagged, so a backend sees them as any other.
func (l *lowerer) loadPrelude(cfg config) *scope {
	if cfg.preludeSrc == "" {
		return nil
	}

	file, err := parser.Parse(cfg.preludeName, strings.NewReader(cfg.preludeSrc))
	if err != nil {
		l.diags.add(ast.Position{Filename: cfg.preludeName}, "the prelude does not parse: %v", err)
		return nil
	}

	outer := newScope(nil)
	l.file, l.scope = outer, outer
	l.collect(file)
	l.lowerUnits(file)
	l.lower(file)
	return outer
}

type lowerer struct {
	model    *ir.Model
	types    map[string]int32           // interning key to index
	unitKeys map[string]int32           // reduced dimensions to Model.units index
	aliases  map[string]string          // import alias to package name
	externs  map[string]int32           // "pkg.Name" to index
	depDecls map[string][]*ir.Directive // "pkg.Name" to its dependency's declaration-level directives
	depFiles map[string]*ast.File       // package to the parse tree of a root import declaring it
	deps     map[string]*lowerer        // package to its lowered dependency, nil while it is lowered
	loader   Loader
	cfg      config
	// preludeDecls is how many of Model.decls the prelude declares.
	preludeDecls int
	file         *scope // the file's declarations
	scope        *scope // the scope a type reference resolves against
	diags        Diagnostics
	refs         *References // nil unless the caller asked for them
}

// collect fills the declaration table with an empty entry per declaration,
// so every name in the file is known before any type reference is resolved.
func (l *lowerer) collect(file *ast.File) {
	for i, decl := range file.Decls {
		if !namesAType(decl) {
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
		l.model.Decls = append(l.model.Decls, &ir.Decl{Meta: metaOf(decl.Head(), i)})
	}
}

// namesAType reports whether a declaration binds a name in the type
// namespace. Instances and target blocks have tables of their own.
func namesAType(decl ast.Decl) bool {
	switch decl.(type) {
	case *ast.InstanceDecl, *ast.TargetDecl:
		return false
	}
	return true
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
		b, ok := l.file.lookup(decl.Name())
		if !ok || b.pos != decl.Pos() {
			continue // a duplicate, already reported
		}
		l.setNode(l.model.Decl(b.id), decl)
	}
}

// declID returns the ID of a collected declaration by name.
func (l *lowerer) declID(name string) *ir.ID {
	b, _ := l.file.lookup(name)
	return b.id
}

// inScope lowers within a scope and restores the previous one.
func (l *lowerer) inScope(s *scope, f func()) {
	prev := l.scope
	l.scope = s
	f()
	l.scope = prev
}

// setNode fills in the oneof, whose generated interface is unexported.
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
				Params:           l.params(d.Params),
				Base:             l.typeRef(d.Base),
				Constraints:      l.classRefs(d.Requires),
				ValueConstraints: l.constraints(d.Constraints),
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
					Meta:   metaOf(&v.DeclHead, i),
					Fields: l.variantFields(v.Fields),
				})
			}
			out.Node = &ir.Decl_Enumeration{Enumeration: e}
		})

	case *ast.UnitDecl:
		// Already lowered by lowerUnits.
	}
}

// structKind is the kind the keyword gives. A `type` is a value until
// [lowerer.markEntities] finds it conforms to the prelude's Entity.
func structKind(keyword string) ir.StructKind {
	if keyword == "mixin" {
		return ir.StructKind_STRUCT_KIND_MIXIN
	}
	return ir.StructKind_STRUCT_KIND_VALUE
}

// markEntities makes an entity of every value satisfying the prelude's
// Entity, reading the satisfaction index so an instance counts.
func (l *lowerer) markEntities() {
	b, ok := l.file.parent.lookup(preludeEntity)
	if !ok || b.kind != bindDecl {
		return
	}
	for _, id := range l.model.Satisfying(b.id) {
		if s := l.model.Decl(id).GetStructure(); s.GetKind() == ir.StructKind_STRUCT_KIND_VALUE {
			s.Kind = ir.StructKind_STRUCT_KIND_ENTITY
		}
	}
}

func (l *lowerer) fields(members []ast.Member) []*ir.Field {
	var in []*ast.Field
	for _, m := range members {
		if f, ok := m.(*ast.Field); ok {
			in = append(in, f)
		}
		// `include` is expanded by expandIncludes.
	}
	return l.variantFields(in)
}

// variantFields lowers a field list, reporting a name used twice.
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
		Meta:         metaOf(&f.DeclHead, order),
		Type:         l.typeRef(f.Type),
		Owned:        f.Owned,
		Constraints:  l.constraints(f.Constraints),
		DefaultValue: l.literal(f.Default),
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
