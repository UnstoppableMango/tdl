package ast

// DeclHead is the part every declaration shares.
type DeclHead struct {
	Doc []string

	// DocP holds where each Doc line was written.
	DocP []Position

	P   Position
	N   string
	Dep *Deprecation
}

func (h *DeclHead) Pos() Position   { return h.P }
func (h *DeclHead) Name() string    { return h.N }
func (h *DeclHead) Head() *DeclHead { return h }

// Deprecation marks a declaration, field, or variant as deprecated.
type Deprecation struct {
	P      Position
	Reason string // "" when written without a reason
}

// ClassRef names a class, optionally qualified and applied to arguments.
type ClassRef struct {
	P         Position
	Qualifier string
	N         string
	Args      []*TypeArg
}

// NewtypeDecl is a `type Name: Base` declaration.
type NewtypeDecl struct {
	DeclHead
	Params      []*TypeParam
	Base        *TypeRef
	Requires    []*ClassRef
	Constraints []*Constraint
	End         Position // the constraint block's `}`; zero without one
}

// StructDecl is a `type` or `mixin` declaration with a body of members.
type StructDecl struct {
	DeclHead
	Keyword  string // "type" or "mixin"
	Params   []*TypeParam
	Conforms []*ClassRef
	Requires []*ClassRef
	Members  []Member
	End      Position // the body's `}`
}

// EnumDecl is a closed set of variants, each optionally carrying fields.
type EnumDecl struct {
	DeclHead
	Params   []*TypeParam
	Conforms []*ClassRef
	Requires []*ClassRef
	Variants []*Variant
	End      Position // the body's `}`
}

// TargetDecl is a `target go for billing { ... }` block.
type TargetDecl struct {
	DeclHead
	For     string // the dotted package name the target applies to
	Entries []*TargetEntry
	End     Position // the block's `}`
}

// Member is one item in a [StructDecl] body: a [Field] or an [Include].
type Member interface {
	Pos() Position
}

// Field is a named, typed member.
type Field struct {
	DeclHead
	Owned       bool // composition rather than reference
	Type        *TypeRef
	Constraints []*Constraint
	Default     *Literal
	End         Position // the constraint block's `}`; zero without one
}

// Include copies a mixin's fields into the including declaration.
type Include struct {
	P    Position
	Type *ClassRef
}

func (i *Include) Pos() Position { return i.P }

// Variant is one alternative in an [EnumDecl].
type Variant struct {
	DeclHead
	Fields []*Field // nil for a variant without a payload
	End    Position // the payload's `}`; zero without one
}

// TargetEntry is one entry in a [TargetDecl]: a nested block, a path
// mapped to a directive, or a bare directive.
type TargetEntry struct {
	P         Position
	Path      string         // "" for a bare directive
	Directive *Directive     // nil when Entries is set
	Entries   []*TargetEntry // nil when Directive is set
	End       Position       // the nested block's `}`; zero without one
}

// Directive is an instruction to a backend, opaque to the compiler.
type Directive struct {
	P    Position
	N    string
	Args []*Literal
}

// LiteralKind is the form a [Literal] takes.
type LiteralKind int

const (
	LitString LiteralKind = iota
	LitInt
	LitFloat
	LitBool
	LitList
	LitName  // a dotted name, denoting an enum variant
	LitRegex // /.../, a constraint argument
	LitRange // 3..254, 1.., ..254
)

// Literal is a field default, constraint argument, or directive argument.
type Literal struct {
	P     Position
	Kind  LiteralKind
	Text  string     // decoded for LitString, pattern body for LitRegex, source text otherwise
	Items []*Literal // set for LitList
	Lo    *Literal   // set for LitRange; nil when the range is open below
	Hi    *Literal   // set for LitRange; nil when the range is open above
}

// Constraint is one entry in a `where { ... }` block. The set of names is
// open.
type Constraint struct {
	P    Position
	N    string
	Args []*Literal
}

// ClassDecl is a class. Conformance to it is nominal and always declared.
type ClassDecl struct {
	DeclHead
	Params   []*TypeParam
	FunDeps  []*FunDep
	Conforms []*ClassRef // superclasses
	Requires []*ClassRef
	Members  []Member
	End      Position // the body's `}`
}

// FunDep states that some class parameters determine others.
type FunDep struct {
	P    Position
	From []string
	To   []string
}

// AssocTypeReq is a `type Cursor` requirement in a class. Dep is always
// nil.
type AssocTypeReq struct {
	DeclHead
	Kind *Kind
}

// InstanceDecl declares that a type satisfies a class.
type InstanceDecl struct {
	DeclHead // N is the class name
	Params   []*TypeParam
	Class    *ClassRef
	For      *TypeRef // set for `instance C for T`, nil for `instance C<T>`
	Requires []*ClassRef
	Binds    []*AssocTypeBind
	End      Position // the bind block's `}`; zero without one
}

// AssocTypeBind binds an associated type in an instance.
type AssocTypeBind struct {
	P      Position
	N      string
	Target *TypeRef
}

// UnitDecl is a base `unit kg` or a derived `unit N = kg*m/s^2`
// declaration.
type UnitDecl struct {
	DeclHead
	Expr *UnitExpr // nil for a base unit
}

// UnitExpr is a product and quotient of unit terms.
type UnitExpr struct {
	P     Position
	Terms []*UnitTerm
}

// UnitTerm is one factor of a [UnitExpr].
type UnitTerm struct {
	P     Position
	Op    string    // "" for the first term, otherwise "*" or "/"
	N     string    // unit name; "" when Paren is set
	Exp   int       // exponent; 1 when written without one
	Paren *UnitExpr // set for a parenthesized sub-expression
}

// TypeArg is one argument in a `<...>` list: a type or a unit. A bare name
// is recorded as a type and the resolver decides.
type TypeArg struct {
	P    Position
	Type *TypeRef  // set unless Unit is
	Unit *UnitExpr // set only when operators made the argument unambiguous
}
