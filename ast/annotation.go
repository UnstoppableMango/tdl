package ast

// Annotation is a `@namespace(key: value, ...)` extension node, attachable
// to type, field, and enum/enum-value declarations. The parser has no
// knowledge of what any given namespace means; only a backend that
// recognizes the namespace interprets it (see the ir package's
// Annotations helper).
type Annotation struct {
	Pos       Position
	Namespace string
	Args      []AnnotationArg
}

// AnnotationArg is a single `key: value` pair within an Annotation's
// argument list.
type AnnotationArg struct {
	Name  string
	Value *Literal
}

// LiteralKind identifies the kind of value held by a Literal.
type LiteralKind int

const (
	LitString LiteralKind = iota
	LitInt
	LitFloat
	LitBool
	LitList
)

// Literal is a literal value: a field/enum-value default, or an
// annotation argument's value. TDL has no expressions, so this is the
// only value representation the grammar needs.
type Literal struct {
	Pos  Position
	Kind LiteralKind

	Str   string
	Int   int64
	Float float64
	Bool  bool
	List  []*Literal
}
