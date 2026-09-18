// Package irtest builds [ir.Model] values by hand for backend tests.
//
// A model is built by hand rather than parsed, so a failure in a backend's
// test is the backend's and not lowering's.
package irtest

import (
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/prelude"
)

// OwnFile is the file [Builder.Own] attributes a declaration to when it
// carries no position of its own.
const OwnFile = "shop.tdl"

// Builder accumulates a model.
type Builder struct {
	Model *ir.Model
}

// New starts a model in a package, seeded with the prelude declarations a
// field can name.
//
// The prelude arrives merged into the declaration table, so every fixture
// carries the primitives, Option, and Nullable.
func New(pkg string) *Builder {
	b := &Builder{Model: &ir.Model{Package: pkg}}
	for _, name := range []string{"string", "int", "bool", "bytes", "decimal", "uuid", "instant", "date", "duration", "List", "Set", "Map"} {
		b.Decl(&ir.Decl{
			Meta: &ir.Meta{Name: name, Position: &ir.Position{Filename: prelude.Name}},
			Node: &ir.Decl_Primitive{Primitive: &ir.Primitive{}},
		})
	}
	for _, e := range []struct{ name, some, none string }{
		{"Option", "Some", "None"},
		{"Nullable", "Present", "Null"},
	} {
		b.Decl(&ir.Decl{
			Meta: &ir.Meta{Name: e.name, Position: &ir.Position{Filename: prelude.Name}},
			Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{
				Params:   []*ir.Param{{Name: "T"}},
				Variants: []*ir.Variant{{Meta: &ir.Meta{Name: e.some}}, {Meta: &ir.Meta{Name: e.none}}},
			}},
		})
	}
	// Identity is conformance to Entity, so the class a model's entities
	// name belongs to the prelude and arrives with it.
	b.Decl(&ir.Decl{
		Meta: &ir.Meta{Name: "Entity", Position: &ir.Position{Filename: prelude.Name}},
		Node: &ir.Decl_Class{Class: &ir.Class{}},
	})
	return b
}

// Decl appends a declaration. A type reference to it is made by name with
// [Builder.Named], so the index never has to be carried around.
func (b *Builder) Decl(d *ir.Decl) {
	b.Model.Decls = append(b.Model.Decls, d)
}

// Own appends a declaration attributed to the model's own file, which is
// how a backend tells it apart from the prelude.
func (b *Builder) Own(d *ir.Decl) {
	if d.GetMeta().GetPosition() == nil {
		d.GetMeta().Position = &ir.Position{Filename: OwnFile}
	}
	b.Decl(d)
}

// Named interns a type reference to a declaration, applied to arguments.
// It panics when nothing carries the name, since that is a broken fixture.
func (b *Builder) Named(declName string, args ...*ir.ID) *ir.ID {
	_, ctor, ok := b.Model.FindDecl(declName)
	if !ok {
		panic("no declaration named " + declName)
	}
	t := &ir.Type{Ctor: ctor, Args: args, Wrote: ir.SyntacticForm_SYNTACTIC_FORM_NAMED}
	id := &ir.ID{Index: int32(len(b.Model.GetTypes())), Name: declName}
	b.Model.Types = append(b.Model.Types, t)
	return id
}

// Param interns a reference to a type parameter, by its position in the
// parameter list of the node declaring it.
//
// A parameter reference carries the name for a diagnostic to print and the
// index for anything reading it to find the parameter, and lowering fills
// both in, so a fixture states both too.
func (b *Builder) Param(paramName string, index int32, args ...*ir.ID) *ir.ID {
	t := &ir.Type{Param: &ir.ParamRef{Name: paramName, Index: index}, Args: args}
	id := &ir.ID{Index: int32(len(b.Model.GetTypes())), Name: paramName}
	b.Model.Types = append(b.Model.Types, t)
	return id
}

// Field is a field with no directives.
func Field(name string, typ *ir.ID) *ir.Field {
	return &ir.Field{Meta: &ir.Meta{Name: name}, Type: typ}
}

// Params is a parameter list, by name.
func Params(names ...string) []*ir.Param {
	out := make([]*ir.Param, len(names))
	for i, n := range names {
		out[i] = &ir.Param{Name: n}
	}
	return out
}

// Text is a string literal.
func Text(s string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_STRING, Text: s}
}

// Name is a bare identifier literal.
func Name(s string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_NAME, Text: s}
}

// Ref is the ID of a declaration, which is how a class reference and the
// satisfaction index name one.
func (b *Builder) Ref(declName string) *ir.ID {
	_, id, ok := b.Model.FindDecl(declName)
	if !ok {
		panic("no declaration named " + declName)
	}
	return id
}

// Class declares a class owned by the model, requiring the classes named.
func (b *Builder) Class(declName string, supers ...string) *ir.Decl {
	c := &ir.Class{}
	for _, s := range supers {
		c.RequiresClasses = append(c.RequiresClasses, &ir.ClassRef{Class: b.Ref(s)})
	}
	d := &ir.Decl{Meta: &ir.Meta{Name: declName}, Node: &ir.Decl_Class{Class: c}}
	b.Own(d)
	return d
}

// Satisfies records declarations as satisfying a class, which is what the
// compiler computes from conformance and instances.
func (b *Builder) Satisfies(class string, decls ...string) {
	id := b.Ref(class)
	var sat *ir.Satisfaction
	for _, s := range b.Model.GetSatisfies() {
		if s.GetClass().GetIndex() == id.GetIndex() {
			sat = s
		}
	}
	if sat == nil {
		sat = &ir.Satisfaction{Class: id}
		b.Model.Satisfies = append(b.Model.Satisfies, sat)
	}
	for _, d := range decls {
		sat.Decls = append(sat.Decls, b.Ref(d))
	}
}

// Requires is a `requires` clause entry naming a class.
func (b *Builder) Requires(class string, line int32, args ...*ir.ID) *ir.ClassRef {
	return &ir.ClassRef{Class: b.Ref(class), Args: args, Position: &ir.Position{Filename: OwnFile, Line: line}}
}

// Extern interns a reference to a declaration in another package.
func (b *Builder) Extern(qualified string) *ir.ID {
	id := &ir.ID{Index: int32(len(b.Model.GetTypes())), Name: qualified}
	b.Model.Types = append(b.Model.Types, &ir.Type{Extern: &ir.ID{Name: qualified}})
	return id
}

// Unit interns a type reference to the quantity a unit declaration
// measures, which is what the `kg` of `decimal<kg>` lowers to: an ordinary
// entry in the type table with `unit` set where a named type sets `ctor`.
func (b *Builder) Unit(declName string, line int32) *ir.ID {
	decl := b.Ref(declName)
	quantity := &ir.ID{Index: int32(len(b.Model.GetUnits())), Name: declName}
	b.Model.Units = append(b.Model.Units, &ir.Unit{
		Dims:  []*ir.Dimension{{Base: decl, Exponent: 1}},
		Decl:  decl,
		Wrote: declName,
	})

	id := &ir.ID{Index: int32(len(b.Model.GetTypes())), Name: declName}
	b.Model.Types = append(b.Model.Types, &ir.Type{
		Unit:     quantity,
		Wrote:    ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
		Position: &ir.Position{Filename: OwnFile, Line: line},
	})
	return id
}
