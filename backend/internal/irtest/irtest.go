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

// Field is a field with no directives.
func Field(name string, typ *ir.ID) *ir.Field {
	return &ir.Field{Meta: &ir.Meta{Name: name}, Type: typ}
}

// Text is a string literal.
func Text(s string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_STRING, Text: s}
}

// Name is a bare identifier literal.
func Name(s string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_NAME, Text: s}
}
