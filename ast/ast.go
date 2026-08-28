// Package ast defines the TDL abstract syntax tree: a parse tree that
// mirrors source text 1:1, with names left unresolved. See the ir package
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
	Types    []*TypeDecl
	Enums    []*EnumDecl
}

// PackageDecl is a `package <dotted.ident>` declaration.
type PackageDecl struct {
	Pos  Position
	Name string // dotted, e.g. "example.v1"
}

// ImportDecl is an `import "path.tdl" as alias` declaration.
type ImportDecl struct {
	Pos   Position
	Path  string
	Alias string
}

// TypeDecl is a `type Name { ... }` record declaration.
type TypeDecl struct {
	Pos         Position
	Name        string
	Fields      []*Field
	Annotations []*Annotation
}

// Field is a single field within a TypeDecl.
type Field struct {
	Pos         Position
	Name        string
	Type        TypeRef
	Optional    bool // trailing '?'
	Default     *Literal
	Annotations []*Annotation
}

// EnumDecl is an `enum Name { ... }` declaration.
type EnumDecl struct {
	Pos         Position
	Name        string
	Values      []*EnumValue
	Annotations []*Annotation
}

// EnumValue is a single variant within an EnumDecl.
type EnumValue struct {
	Pos         Position
	Name        string
	Value       *Literal // nil if the variant has no explicit literal
	Annotations []*Annotation
}

// TypeRef is a reference to a type: a primitive/named type, or a list/map
// of other TypeRefs. Exactly one of Name (with optional Qualifier), List,
// or (MapKey and MapValue) is set.
type TypeRef struct {
	Pos Position

	Qualifier string // "" if unqualified; set for "alias.Type" references
	Name      string // primitive or named type identifier; "" for List/Map

	List *TypeRef // set for list<T>

	MapKey   *TypeRef // set for map<K, V>
	MapValue *TypeRef
}
