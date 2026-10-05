// Package ast defines the TDL parse tree, which mirrors source text 1:1
// with names left unresolved. See docs/design/ir.md for the resolved model.
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

	// Comments holds every ordinary `//` comment in source order. Doc
	// comments live in the Doc of the declaration they precede.
	Comments []*Comment

	End Position
}

// Comment is one ordinary `//` comment.
type Comment struct {
	P    Position
	Text string // the text after the slashes, with one leading space removed
}

// Decl is a top-level declaration. Every form embeds [DeclHead].
type Decl interface {
	Pos() Position
	Name() string
	Head() *DeclHead
}

// PackageDecl is a `package <dotted.ident>` declaration.
type PackageDecl struct {
	P    Position
	Path string // dotted, e.g. "shop.orders"
}

// ImportDecl is an `import "path.tdl" as alias` declaration.
type ImportDecl struct {
	Doc  []string
	DocP []Position // where each Doc line was written

	P     Position
	Path  string
	Alias string // "_" merges the imported names into the current scope
}

// PrimitiveDecl is a `primitive Name` or `primitive Name: Kind`
// declaration of an opaque root type.
type PrimitiveDecl struct {
	DeclHead
	Kind *Kind // nil when the kind is left to inference
}

// AliasDecl is an `alias Name = TypeRef` declaration, optionally
// parameterized.
type AliasDecl struct {
	DeclHead
	Params []*TypeParam
	Target *TypeRef
}

// TypeParam is one parameter in a `<...>` parameter list.
type TypeParam struct {
	P    Position
	N    string
	Kind *Kind // nil when inferred from use
}

// Kind is a kind expression. Arrow associates to the right.
type Kind struct {
	P     Position
	N     string // "type" or "unit"; empty when Paren is set
	Paren *Kind
	Arrow *Kind // `left -> Arrow`; nil for a bare atom
}

// TypeRef is a reference to a type, with collection and optionality sugar
// recorded as written. The resolver lowers it to prelude types.
type TypeRef struct {
	P Position

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
