// Package ast defines the TDL abstract syntax tree: a parse tree that
// mirrors source text 1:1, with names left unresolved. See docs/design/ir.md
// for the resolved semantic model backends consume.
package ast

import "github.com/unstoppablemango/tdl/lex"

// Position identifies a location in a source file.
type Position = lex.Position

// File is a single parsed .tdl source file.
type File struct {
	Filename string
	Package  *PackageDecl // nil if omitted
	Imports  []*ImportDecl
	Decls    []Decl
}

// Decl is a top-level declaration. `class` and `instance` arrive with
// phase 4 of docs/design/parser-plan.md; every other form is here.
type Decl interface {
	Pos() Position
	Name() string
	docLines() []string
	deprecation() *Deprecation
}

// PackageDecl is a `package <dotted.ident>` declaration.
type PackageDecl struct {
	P    Position
	Path string // dotted, e.g. "shop.orders"
}

// ImportDecl is an `import "path.tdl" as alias` declaration.
type ImportDecl struct {
	Doc   []string
	P     Position
	Path  string
	Alias string // "_" merges the imported names into the current scope
}

// PrimitiveDecl is a `primitive Name` or `primitive Name: Kind` declaration.
// It introduces an opaque, irreducible root type.
type PrimitiveDecl struct {
	DeclHead
	Kind *Kind // nil when the kind is left to inference
}

// AliasDecl is an `alias Name = TypeRef` declaration, optionally
// parameterized. An alias is transparent: it is expanded rather than
// referenced.
type AliasDecl struct {
	DeclHead
	Params []*TypeParam
	Target *TypeRef
}

// Doc returns the doc comment lines attached to a declaration.
func Doc(d Decl) []string { return d.docLines() }

// TypeParam is one parameter in a `<...>` parameter list, with an optional
// kind annotation.
type TypeParam struct {
	P    Position
	N    string
	Kind *Kind // nil when inferred from use
}

// Kind is a kind expression. Name is "type" or "unit" for an atom, or Paren
// holds a parenthesized kind; Arrow is set when this kind is the left side
// of an arrow, which associates to the right.
type Kind struct {
	P     Position
	N     string // "type" or "unit"; empty when Paren is set
	Paren *Kind
	Arrow *Kind // `left -> Arrow`; nil for a bare atom
}

// TypeRef is a reference to a type.
//
// The collection and optionality forms are sugar for prelude types, and the
// parser records the form as written: lowering to List, Set, Map, Option,
// and Nullable is the resolver's job.
type TypeRef struct {
	P Position

	// Named form: an optionally qualified name with optional arguments.
	Qualifier string // "" if unqualified; set for "alias.Type"
	N         string // "" for the collection forms below
	Args      []*TypeArg

	List *TypeRef // [T]
	Set  *TypeRef // {T}

	MapKey   *TypeRef // {K -> V}
	MapValue *TypeRef

	Optional bool // trailing ?
	Nullable bool // trailing | null
}
