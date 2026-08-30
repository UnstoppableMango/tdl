package ast

// DeclHead is the part every declaration shares: its doc comment, its
// position, its name, and whether it is deprecated.
type DeclHead struct {
	Doc []string
	P   Position
	N   string
	Dep *Deprecation
}

func (h *DeclHead) Pos() Position      { return h.P }
func (h *DeclHead) Name() string       { return h.N }
func (h *DeclHead) docLines() []string { return h.Doc }

// Deprecated returns the deprecation attached to a declaration, or nil.
func Deprecated(d Decl) *Deprecation { return d.deprecation() }

// Deprecation marks a declaration, field, or variant as on its way out.
type Deprecation struct {
	P      Position
	Reason string // "" when written without a reason
}

// ClassRef names a class, optionally qualified and applied to arguments.
// It is syntactically a named type reference, but the two are different
// kinds of thing and the tree keeps them apart.
type ClassRef struct {
	P         Position
	Qualifier string
	N         string
	Args      []*TypeArg
}

// NewtypeDecl is a `type Name: Base` declaration. A newtype is distinct
// from the type it is built on.
type NewtypeDecl struct {
	DeclHead
	Params      []*TypeParam
	Base        *TypeRef
	Requires    []*ClassRef
	Constraints []*Constraint
}

// StructDecl is a declaration with a body of members: `entity`, `value`, or
// `mixin`. The three share a shape and differ in meaning, so the keyword is
// recorded rather than split across three identical node types.
type StructDecl struct {
	DeclHead
	Keyword  string // "entity", "value", or "mixin"
	Params   []*TypeParam
	Conforms []*ClassRef
	Requires []*ClassRef
	Members  []Member
}

// EnumDecl is a closed set of variants. A variant may carry fields, which
// makes enum the language's sum type.
type EnumDecl struct {
	DeclHead
	Params   []*TypeParam
	Conforms []*ClassRef
	Requires []*ClassRef
	Variants []*Variant
}

// TargetDecl is a `target go for billing { ... }` block. Everything a code
// generator needs lives here rather than in the model.
type TargetDecl struct {
	DeclHead
	For     string // the dotted package name the target applies to
	Entries []*TargetEntry
}

// Member is one item in a [StructDecl] body: a [Field] or an [Include].
type Member interface {
	MemberPos() Position
}

// Field is a named, typed member.
type Field struct {
	Doc         []string
	P           Position
	N           string
	Key         bool // part of the entity's identity
	Owned       bool // composition rather than reference
	Dep         *Deprecation
	Type        *TypeRef
	Constraints []*Constraint
	Default     *Literal
}

func (f *Field) MemberPos() Position { return f.P }

// Include copies a mixin's fields into the including declaration.
type Include struct {
	P    Position
	Type *ClassRef
}

func (i *Include) MemberPos() Position { return i.P }

// Variant is one alternative in an [EnumDecl].
type Variant struct {
	Doc    []string
	P      Position
	N      string
	Dep    *Deprecation
	Fields []*Field // nil for a variant without a payload
}

// TargetEntry is one entry in a [TargetDecl]: a path scoping a nested
// block, a path mapped to a directive, or a bare directive applying to the
// enclosing scope.
type TargetEntry struct {
	P         Position
	Path      string         // "" for a bare directive
	Directive *Directive     // nil when Entries is set
	Entries   []*TargetEntry // nil when Directive is set
}

// Directive is an opaque instruction to a backend. The compiler checks its
// shape and hands it over; what it means is the backend's business.
type Directive struct {
	P    Position
	N    string
	Args []*Literal
}

// LiteralKind identifies which form a [Literal] takes.
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

// Literal is a literal value: a field default, a constraint argument, or a
// directive argument.
type Literal struct {
	P     Position
	Kind  LiteralKind
	Text  string     // decoded for LitString, pattern body for LitRegex, source text otherwise
	Items []*Literal // set for LitList
	Lo    *Literal   // set for LitRange; nil when the range is open below
	Hi    *Literal   // set for LitRange; nil when the range is open above
}

// Constraint is one entry in a `where { ... }` block.
//
// The set of names is open. The compiler checks the arity and argument
// kinds of the standard names and passes everything else through, so a
// backend may understand a constraint the compiler has never heard of.
type Constraint struct {
	P    Position
	N    string
	Args []*Literal
}

func (h *DeclHead) deprecation() *Deprecation { return h.Dep }

// ClassDecl is a contract. It declares nothing into the types that satisfy
// it; conformance is nominal and always declared.
type ClassDecl struct {
	DeclHead
	Params   []*TypeParam
	FunDeps  []*FunDep
	Conforms []*ClassRef // classes this one requires
	Requires []*ClassRef
	Members  []Member
}

// FunDep states that some parameters determine others, which makes a
// multi-parameter class a function rather than a table.
type FunDep struct {
	P    Position
	From []string
	To   []string
}

// KeyRequirement is a bare `key` in a class body: an implementor must have
// some key, without the class saying which.
type KeyRequirement struct {
	P Position
}

func (k *KeyRequirement) MemberPos() Position { return k.P }

// AssocTypeReq is a `type Cursor` requirement: an implementor supplies a
// type, and an instance binds it.
type AssocTypeReq struct {
	Doc  []string
	P    Position
	N    string
	Kind *Kind
}

func (a *AssocTypeReq) MemberPos() Position { return a.P }

// InstanceDecl declares that a type satisfies a class.
//
// `instance C for T` is sugar for `instance C<T>`, available when the class
// takes one parameter. The parser records which was written.
type InstanceDecl struct {
	DeclHead // N is the class name
	Params   []*TypeParam
	Class    *ClassRef
	For      *TypeRef // set when written with `for`, nil when written with type arguments
	Requires []*ClassRef
	Binds    []*AssocTypeBind
}

// AssocTypeBind supplies a type for one of a class's associated type
// requirements.
type AssocTypeBind struct {
	P      Position
	N      string
	Target *TypeRef
}

// UnitDecl is a `unit kg` or `unit N = kg*m/s^2` declaration. A unit
// without an expression is a base unit; one with an expression is derived
// and reduces to base dimensions before comparison.
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

// TypeArg is one argument in a `<...>` list. It is a type or a unit, and
// the two are told apart by kind rather than by syntax: a bare name could
// be either, so the parser records what was written and the resolver
// decides against the declaration being applied.
type TypeArg struct {
	P    Position
	Type *TypeRef  // set unless Unit is
	Unit *UnitExpr // set only when operators made the argument unambiguous
}
