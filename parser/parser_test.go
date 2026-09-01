package parser_test

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/parser"
)

func parse(t *testing.T, src string) *ast.File {
	t.Helper()
	file, err := parser.Parse("test.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	return file
}

func parseErr(t *testing.T, src string) string {
	t.Helper()
	_, err := parser.Parse("test.tdl", strings.NewReader(src))
	if err == nil {
		t.Fatal("expected a parse error, got none")
	}
	return err.Error()
}

func TestPackageAndImports(t *testing.T) {
	file := parse(t, `
package shop.orders

import "common.tdl" as common
import "std/prelude" as _
`)

	if file.Package == nil || file.Package.Path != "shop.orders" {
		t.Fatalf("package = %+v, want shop.orders", file.Package)
	}
	if len(file.Imports) != 2 {
		t.Fatalf("got %d imports, want 2", len(file.Imports))
	}
	if got := file.Imports[1].Alias; got != "_" {
		t.Errorf("second import alias = %q, want _", got)
	}
}

func TestPrimitiveKinds(t *testing.T) {
	file := parse(t, `
primitive string
primitive Map: type -> type -> type
primitive Higher: (type -> type) -> type
primitive Measured: unit -> type
`)

	if len(file.Decls) != 4 {
		t.Fatalf("got %d decls, want 4", len(file.Decls))
	}

	if p := file.Decls[0].(*ast.PrimitiveDecl); p.Kind != nil {
		t.Errorf("string kind = %+v, want nil", p.Kind)
	}

	// `type -> type -> type` associates to the right.
	m := file.Decls[1].(*ast.PrimitiveDecl)
	if m.Kind.N != "type" || m.Kind.Arrow == nil || m.Kind.Arrow.Arrow == nil {
		t.Fatalf("Map kind did not associate right: %+v", m.Kind)
	}
	if m.Kind.Arrow.Arrow.Arrow != nil {
		t.Error("Map kind has a fourth atom")
	}

	// A parenthesized kind is the left operand, not the right.
	h := file.Decls[2].(*ast.PrimitiveDecl)
	if h.Kind.Paren == nil {
		t.Fatalf("Higher kind lost its parentheses: %+v", h.Kind)
	}
	if h.Kind.Paren.Arrow == nil || h.Kind.Arrow == nil {
		t.Errorf("Higher kind = %+v", h.Kind)
	}
}

func TestAliasParameters(t *testing.T) {
	file := parse(t, `alias Applied<f: type -> type, T> = f<T>`)

	a := file.Decls[0].(*ast.AliasDecl)
	if len(a.Params) != 2 {
		t.Fatalf("got %d params, want 2", len(a.Params))
	}
	if a.Params[0].Kind == nil {
		t.Error("first param lost its kind annotation")
	}
	if a.Params[1].Kind != nil {
		t.Error("second param gained a kind annotation")
	}
	if a.Target.N != "f" || len(a.Target.Args) != 1 {
		t.Errorf("target = %+v", a.Target)
	}
}

func TestTypeRefForms(t *testing.T) {
	tests := []struct {
		src   string
		check func(*testing.T, *ast.TypeRef)
	}{
		{"[T]", func(t *testing.T, r *ast.TypeRef) {
			if r.List == nil || r.List.N != "T" {
				t.Errorf("list = %+v", r)
			}
		}},
		{"{T}", func(t *testing.T, r *ast.TypeRef) {
			if r.Set == nil || r.Set.N != "T" {
				t.Errorf("set = %+v", r)
			}
		}},
		{"{K -> V}", func(t *testing.T, r *ast.TypeRef) {
			if r.MapKey == nil || r.MapValue == nil {
				t.Errorf("map = %+v", r)
			}
		}},
		{"T?", func(t *testing.T, r *ast.TypeRef) {
			if !r.Optional || r.Nullable {
				t.Errorf("optional = %+v", r)
			}
		}},
		{"T | null", func(t *testing.T, r *ast.TypeRef) {
			if r.Optional || !r.Nullable {
				t.Errorf("nullable = %+v", r)
			}
		}},
		{"T? | null", func(t *testing.T, r *ast.TypeRef) {
			if !r.Optional || !r.Nullable {
				t.Errorf("both = %+v", r)
			}
		}},
		{"common.Address", func(t *testing.T, r *ast.TypeRef) {
			if r.Qualifier != "common" || r.N != "Address" {
				t.Errorf("qualified = %+v", r)
			}
		}},
		{"Map<string, [T]>", func(t *testing.T, r *ast.TypeRef) {
			if len(r.Args) != 2 || r.Args[1].Type == nil || r.Args[1].Type.List == nil {
				t.Errorf("args = %+v", r)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			file := parse(t, "alias X = "+tt.src)
			tt.check(t, file.Decls[0].(*ast.AliasDecl).Target)
		})
	}
}

func TestDocComments(t *testing.T) {
	file := parse(t, `
/// A handler table.
/// Keyed by event name.
alias Handler = {string -> [Event]}
`)

	doc := ast.Doc(file.Decls[0])
	if len(doc) != 2 || doc[0] != "A handler table." {
		t.Errorf("doc = %q", doc)
	}
}

// Whitespace is insignificant: a declaration ends where the next begins.
func TestWhitespaceInsignificant(t *testing.T) {
	oneLine := parse(t, `primitive string primitive int alias A = string`)
	if len(oneLine.Decls) != 3 {
		t.Fatalf("got %d decls on one line, want 3", len(oneLine.Decls))
	}
}

func TestErrorsAccumulate(t *testing.T) {
	msg := parseErr(t, `
import "a.tdl"
alias Broken =
`)
	if !strings.Contains(msg, "expected as") {
		t.Errorf("error missing the import problem: %s", msg)
	}
}

func TestReservedKeywordsCannotName(t *testing.T) {
	if msg := parseErr(t, `alias entity = string`); !strings.Contains(msg, "expected identifier") {
		t.Errorf("error = %s", msg)
	}
}

// Modifier and constraint names are contextual, so they stay usable as
// ordinary identifiers.
func TestContextualKeywordsAreIdentifiers(t *testing.T) {
	for _, name := range []string{"key", "owned", "deprecated", "min", "max", "length", "matches", "oneOf", "unique"} {
		t.Run(name, func(t *testing.T) {
			file := parse(t, "alias "+name+" = string")
			if got := file.Decls[0].Name(); got != name {
				t.Errorf("name = %q, want %q", got, name)
			}
		})
	}
}

func TestNewtype(t *testing.T) {
	file := parse(t, `type Email: string`)

	d := file.Decls[0].(*ast.NewtypeDecl)
	if d.N != "Email" || d.Base.N != "string" {
		t.Errorf("newtype = %+v", d)
	}
}

// `type` introduces a newtype, so the M1 record form is a syntax error.
func TestRecordFormRejected(t *testing.T) {
	if msg := parseErr(t, "type User {\n  id: string\n}"); !strings.Contains(msg, "expected :, got {") {
		t.Errorf("error = %s", msg)
	}
}

func TestStructDecls(t *testing.T) {
	file := parse(t, `
entity Order: Auditable, Tenanted requires Ord<T> {
  key id: OrderId
  key sku: string
  include Timestamps
  items: [LineItem] owned
  status: Status = Draft
  deprecated("gone soon") legacy: string
}

value Money { amount: decimal }

mixin Timestamps { createdAt: instant }
`)

	order := file.Decls[0].(*ast.StructDecl)
	if order.Keyword != "entity" {
		t.Errorf("keyword = %q", order.Keyword)
	}
	if len(order.Conforms) != 2 || len(order.Requires) != 1 {
		t.Errorf("conforms %d, requires %d", len(order.Conforms), len(order.Requires))
	}
	if len(order.Members) != 6 {
		t.Fatalf("got %d members, want 6", len(order.Members))
	}

	if f := order.Members[0].(*ast.Field); !f.Key || f.N != "id" {
		t.Errorf("first member = %+v", f)
	}
	if _, ok := order.Members[2].(*ast.Include); !ok {
		t.Errorf("third member is not an include: %T", order.Members[2])
	}
	if f := order.Members[3].(*ast.Field); !f.Owned {
		t.Errorf("items lost `owned`: %+v", f)
	}
	if f := order.Members[4].(*ast.Field); f.Default == nil || f.Default.Kind != ast.LitName {
		t.Errorf("status default = %+v", f.Default)
	}
	if f := order.Members[5].(*ast.Field); f.Dep == nil || f.Dep.Reason != "gone soon" {
		t.Errorf("legacy deprecation = %+v", f.Dep)
	}

	if file.Decls[1].(*ast.StructDecl).Keyword != "value" {
		t.Error("value keyword lost")
	}
	if file.Decls[2].(*ast.StructDecl).Keyword != "mixin" {
		t.Error("mixin keyword lost")
	}
}

// `key` and `deprecated` are contextual, so they are still legal field
// names. A modifier is a modifier only when a colon does not follow it.
func TestContextualModifiersAsFieldNames(t *testing.T) {
	file := parse(t, `value V {
  key: string
  deprecated: bool
  owned: int
  key id: string
}`)

	members := file.Decls[0].(*ast.StructDecl).Members
	if len(members) != 4 {
		t.Fatalf("got %d members, want 4", len(members))
	}
	for i, want := range []string{"key", "deprecated", "owned"} {
		f := members[i].(*ast.Field)
		if f.N != want || f.Key {
			t.Errorf("member %d = %+v, want a field named %q", i, f, want)
		}
	}
	if f := members[3].(*ast.Field); !f.Key || f.N != "id" {
		t.Errorf("last member = %+v", f)
	}
}

func TestEnumVariants(t *testing.T) {
	file := parse(t, `enum Payment {
  Card { last4: string brand: CardBrand }
  deprecated Credit
  Cash
}`)

	d := file.Decls[0].(*ast.EnumDecl)
	if len(d.Variants) != 3 {
		t.Fatalf("got %d variants, want 3", len(d.Variants))
	}
	if len(d.Variants[0].Fields) != 2 {
		t.Errorf("Card payload = %d fields", len(d.Variants[0].Fields))
	}
	if d.Variants[1].Dep == nil {
		t.Error("Credit lost its deprecation")
	}
	if len(d.Variants[2].Fields) != 0 {
		t.Error("Cash gained a payload")
	}
}

func TestEnumValuesRejected(t *testing.T) {
	if msg := parseErr(t, `enum Role { Admin = "admin" }`); !strings.Contains(msg, "carry fields, not values") {
		t.Errorf("error = %s", msg)
	}
}

func TestCommasRejectedInBlocks(t *testing.T) {
	if msg := parseErr(t, "value P {\n  x: int,\n  y: int\n}"); !strings.Contains(msg, "not separators inside a block") {
		t.Errorf("error = %s", msg)
	}
}

func TestDeprecatedDecl(t *testing.T) {
	file := parse(t, `
deprecated("use Contact")
entity Legacy { key id: string }
`)

	dep := ast.Deprecated(file.Decls[0])
	if dep == nil || dep.Reason != "use Contact" {
		t.Errorf("deprecation = %+v", dep)
	}
}

func TestTargetBlock(t *testing.T) {
	file := parse(t, `target go for billing {
  out("./gen/go")
  package("github.com/acme/billing")
  User {
    name("Account")
    email => tag("x")
  }
  Order.items => slice
}`)

	d := file.Decls[0].(*ast.TargetDecl)
	if d.N != "go" || d.For != "billing" {
		t.Errorf("target = %+v", d)
	}
	if len(d.Entries) != 4 {
		t.Fatalf("got %d entries, want 4", len(d.Entries))
	}

	// A reserved keyword is a legal directive name: the namespace belongs to
	// the backend, not to TDL.
	if got := d.Entries[1].Directive.N; got != "package" {
		t.Errorf("second directive = %q, want package", got)
	}
	if nested := d.Entries[2]; nested.Path != "User" || len(nested.Entries) != 2 {
		t.Errorf("nested block = %+v", nested)
	}
	if last := d.Entries[3]; last.Path != "Order.items" || last.Directive.N != "slice" {
		t.Errorf("last entry = %+v", last)
	}
}

func TestDirectiveArguments(t *testing.T) {
	file := parse(t, `target go for p {
  foreign("a", "b")
  slice
  numbers(1, 2.5, true, [1, 2])
}`)

	entries := file.Decls[0].(*ast.TargetDecl).Entries
	if got := len(entries[0].Directive.Args); got != 2 {
		t.Errorf("foreign args = %d, want 2", got)
	}
	if got := len(entries[1].Directive.Args); got != 0 {
		t.Errorf("slice args = %d, want 0", got)
	}
	args := entries[2].Directive.Args
	if len(args) != 4 || args[3].Kind != ast.LitList || len(args[3].Items) != 2 {
		t.Errorf("numbers args = %+v", args)
	}
}

// A reserved word followed by `:` is a field name. The prelude's Option<T>
// depends on it.
func TestKeywordFieldNames(t *testing.T) {
	file := parse(t, `value V {
  value: string
  type: string
  unit: string
  target: string
  include: string
  where: string
}`)

	members := file.Decls[0].(*ast.StructDecl).Members
	if len(members) != 6 {
		t.Fatalf("got %d members, want 6", len(members))
	}
	for i, want := range []string{"value", "type", "unit", "target", "include", "where"} {
		f, ok := members[i].(*ast.Field)
		if !ok {
			t.Fatalf("member %d is %T, want a field", i, members[i])
		}
		if f.N != want {
			t.Errorf("member %d = %q, want %q", i, f.N, want)
		}
	}
}

// A field named `where` follows a field, which is exactly where the
// previous field looks for a constraint block. One token of lookahead
// separates the two, the way it does for the contextual modifiers.
func TestWhereFieldNameAfterAField(t *testing.T) {
	file := parse(t, `value V {
  a: string
  where: int where { min(0) }
  b: string
}`)

	members := file.Decls[0].(*ast.StructDecl).Members
	if len(members) != 3 {
		t.Fatalf("got %d members, want 3", len(members))
	}

	if a := members[0].(*ast.Field); len(a.Constraints) != 0 {
		t.Errorf("field a took %d constraints, want none", len(a.Constraints))
	}

	w := members[1].(*ast.Field)
	if w.N != "where" {
		t.Errorf("second field = %q, want \"where\"", w.N)
	}
	if len(w.Constraints) != 1 || w.Constraints[0].N != "min" {
		t.Errorf("field where constraints = %+v, want min", w.Constraints)
	}
}

// `include Foo` is still an include, not a field named include.
func TestIncludeStillParses(t *testing.T) {
	file := parse(t, `value V { include Timestamps }`)
	if _, ok := file.Decls[0].(*ast.StructDecl).Members[0].(*ast.Include); !ok {
		t.Error("include parsed as something else")
	}
}

func TestConstraints(t *testing.T) {
	file := parse(t, `type Email: string where {
  matches(/^[^@]+@[^@]+$/)
  length(3..254)
  unique
}`)

	cs := file.Decls[0].(*ast.NewtypeDecl).Constraints
	if len(cs) != 3 {
		t.Fatalf("got %d constraints, want 3", len(cs))
	}

	if cs[0].N != "matches" || cs[0].Args[0].Kind != ast.LitRegex {
		t.Errorf("matches = %+v", cs[0])
	}
	if got := cs[0].Args[0].Text; got != "^[^@]+@[^@]+$" {
		t.Errorf("regex body = %q", got)
	}
	if cs[1].Args[0].Kind != ast.LitRange {
		t.Fatalf("length arg = %+v", cs[1].Args[0])
	}
	if r := cs[1].Args[0]; r.Lo.Text != "3" || r.Hi.Text != "254" {
		t.Errorf("range = %+v", r)
	}
	if len(cs[2].Args) != 0 {
		t.Errorf("unique took arguments: %+v", cs[2])
	}
}

func TestRangeForms(t *testing.T) {
	tests := []struct {
		src    string
		lo, hi string
	}{
		{"3..254", "3", "254"},
		{"1..", "1", ""},
		{"..64", "", "64"},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			file := parse(t, "type T: string where { length("+tt.src+") }")
			arg := file.Decls[0].(*ast.NewtypeDecl).Constraints[0].Args[0]
			if arg.Kind != ast.LitRange {
				t.Fatalf("kind = %v", arg.Kind)
			}
			var lo, hi string
			if arg.Lo != nil {
				lo = arg.Lo.Text
			}
			if arg.Hi != nil {
				hi = arg.Hi.Text
			}
			if lo != tt.lo || hi != tt.hi {
				t.Errorf("got %q..%q, want %q..%q", lo, hi, tt.lo, tt.hi)
			}
		})
	}
}

// The constraint set is open: the parser recognizes no name in particular.
func TestUnknownConstraintParses(t *testing.T) {
	file := parse(t, `entity E { x: int where { between(0, 100) } }`)

	f := file.Decls[0].(*ast.StructDecl).Members[0].(*ast.Field)
	if len(f.Constraints) != 1 || f.Constraints[0].N != "between" {
		t.Fatalf("constraints = %+v", f.Constraints)
	}
	if len(f.Constraints[0].Args) != 2 {
		t.Errorf("got %d args, want 2", len(f.Constraints[0].Args))
	}
}

func TestFieldConstraintsThenDefault(t *testing.T) {
	file := parse(t, `entity E { status: Status where { unique } = Draft }`)

	f := file.Decls[0].(*ast.StructDecl).Members[0].(*ast.Field)
	if len(f.Constraints) != 1 {
		t.Errorf("constraints = %+v", f.Constraints)
	}
	if f.Default == nil || f.Default.Text != "Draft" {
		t.Errorf("default = %+v", f.Default)
	}
}

// A constraint block must be introduced by `where`, so `{` after a type
// reference is never a constraint block.
func TestConstraintBlockNeedsWhere(t *testing.T) {
	if msg := parseErr(t, "type Email: string {\n  length(3..254)\n}"); !strings.Contains(msg, "expected a declaration") {
		t.Errorf("error = %s", msg)
	}
}

// `/` is division in a unit expression and a regex delimiter in a
// constraint. Nothing before it says which, so the parser asks.
func TestRegexVersusDivision(t *testing.T) {
	file := parse(t, `type Path: string where { matches(/a\/b/) }`)

	arg := file.Decls[0].(*ast.NewtypeDecl).Constraints[0].Args[0]
	if arg.Kind != ast.LitRegex || arg.Text != `a\/b` {
		t.Errorf("regex = %v %q", arg.Kind, arg.Text)
	}
}

func TestUnterminatedRegex(t *testing.T) {
	if msg := parseErr(t, "type T: string where { matches(/nope) }"); !strings.Contains(msg, "unterminated regex") {
		t.Errorf("error = %s", msg)
	}
}

func TestClassDecl(t *testing.T) {
	file := parse(t, `class Auditable: Timestamped requires Ord<T> {
  key
  type Cursor: type
  createdAt: instant
}`)

	d := file.Decls[0].(*ast.ClassDecl)
	if len(d.Conforms) != 1 || len(d.Requires) != 1 {
		t.Errorf("conforms %d, requires %d", len(d.Conforms), len(d.Requires))
	}
	if len(d.Members) != 3 {
		t.Fatalf("got %d members, want 3", len(d.Members))
	}
	if _, ok := d.Members[0].(*ast.KeyRequirement); !ok {
		t.Errorf("first member is %T, want a key requirement", d.Members[0])
	}
	req, ok := d.Members[1].(*ast.AssocTypeReq)
	if !ok {
		t.Fatalf("second member is %T, want an associated type", d.Members[1])
	}
	if req.N != "Cursor" || req.Kind == nil {
		t.Errorf("assoc type = %+v", req)
	}
	if _, ok := d.Members[2].(*ast.Field); !ok {
		t.Errorf("third member is %T, want a field", d.Members[2])
	}
}

// A class may not declare key fields, so a bare `key` before a field is the
// requirement and not a modifier on that field.
func TestClassKeyIsARequirement(t *testing.T) {
	file := parse(t, `class Tenanted {
  key
  tenant: string
}`)

	members := file.Decls[0].(*ast.ClassDecl).Members
	if len(members) != 2 {
		t.Fatalf("got %d members, want 2", len(members))
	}
	if _, ok := members[0].(*ast.KeyRequirement); !ok {
		t.Errorf("first member is %T, want a key requirement", members[0])
	}
	if f := members[1].(*ast.Field); f.N != "tenant" || f.Key {
		t.Errorf("second member = %+v", f)
	}
}

// A field named `key` or `type` still works inside a class.
func TestClassKeywordFields(t *testing.T) {
	file := parse(t, `class C {
  key: string
  type: string
}`)

	members := file.Decls[0].(*ast.ClassDecl).Members
	for i, want := range []string{"key", "type"} {
		f, ok := members[i].(*ast.Field)
		if !ok {
			t.Fatalf("member %d is %T, want a field", i, members[i])
		}
		if f.N != want {
			t.Errorf("member %d = %q, want %q", i, f.N, want)
		}
	}
}

func TestFunDeps(t *testing.T) {
	file := parse(t, `class Projection<from, to, extra> | from -> to, from to -> extra { }`)

	deps := file.Decls[0].(*ast.ClassDecl).FunDeps
	if len(deps) != 2 {
		t.Fatalf("got %d fundeps, want 2", len(deps))
	}
	if len(deps[0].From) != 1 || deps[0].To[0] != "to" {
		t.Errorf("first fundep = %+v", deps[0])
	}
	if len(deps[1].From) != 2 || deps[1].To[0] != "extra" {
		t.Errorf("second fundep = %+v", deps[1])
	}
}

func TestInstanceForms(t *testing.T) {
	file := parse(t, `
instance Auditable<shipping.Address>
instance Auditable for shipping.Address
instance Paged for OrderList {
  type Cursor = OrderCursor
}
instance <T> Auditable<Page<T>> requires Auditable<T>
`)

	if len(file.Decls) != 4 {
		t.Fatalf("got %d decls, want 4", len(file.Decls))
	}

	// `instance C<T>` and `instance C for T` mean the same thing, and the
	// tree records which was written.
	args := file.Decls[0].(*ast.InstanceDecl)
	if args.For != nil || len(args.Class.Args) != 1 {
		t.Errorf("argument form = %+v", args)
	}
	sugar := file.Decls[1].(*ast.InstanceDecl)
	if sugar.For == nil || sugar.For.Qualifier != "shipping" {
		t.Errorf("for form = %+v", sugar)
	}

	binds := file.Decls[2].(*ast.InstanceDecl).Binds
	if len(binds) != 1 || binds[0].N != "Cursor" || binds[0].Target.N != "OrderCursor" {
		t.Errorf("binds = %+v", binds)
	}

	conditional := file.Decls[3].(*ast.InstanceDecl)
	if len(conditional.Params) != 1 || len(conditional.Requires) != 1 {
		t.Errorf("conditional instance = %+v", conditional)
	}
}

func TestInstanceNeedsSubject(t *testing.T) {
	if msg := parseErr(t, "instance Auditable"); !strings.Contains(msg, "expected type arguments or 'for'") {
		t.Errorf("error = %s", msg)
	}
}

func TestUnitDecls(t *testing.T) {
	file := parse(t, `
unit kg
unit N = kg*m/s^2
unit Hz = s^-1
unit Complex = (kg*m)/(s^2*m)
`)

	if len(file.Decls) != 4 {
		t.Fatalf("got %d decls, want 4", len(file.Decls))
	}

	if base := file.Decls[0].(*ast.UnitDecl); base.Expr != nil {
		t.Errorf("base unit gained an expression: %+v", base.Expr)
	}

	// `*` and `/` associate left with equal precedence, so the expression is
	// a flat sequence of terms carrying their own operator.
	n := file.Decls[1].(*ast.UnitDecl).Expr
	if len(n.Terms) != 3 {
		t.Fatalf("N has %d terms, want 3", len(n.Terms))
	}
	if n.Terms[0].Op != "" || n.Terms[1].Op != "*" || n.Terms[2].Op != "/" {
		t.Errorf("operators = %q %q %q", n.Terms[0].Op, n.Terms[1].Op, n.Terms[2].Op)
	}
	if n.Terms[0].Exp != 1 || n.Terms[2].Exp != 2 {
		t.Errorf("exponents = %d %d", n.Terms[0].Exp, n.Terms[2].Exp)
	}

	if exp := file.Decls[2].(*ast.UnitDecl).Expr.Terms[0].Exp; exp != -1 {
		t.Errorf("Hz exponent = %d, want -1", exp)
	}

	compound := file.Decls[3].(*ast.UnitDecl).Expr
	if compound.Terms[0].Paren == nil || compound.Terms[1].Paren == nil {
		t.Errorf("parenthesized terms lost: %+v", compound.Terms)
	}
}

// A bare name in `<...>` could be a type or a unit, so it is recorded as a
// type reference and the resolver picks by kind. Operators settle it.
func TestUnitVersusTypeArguments(t *testing.T) {
	tests := []struct {
		src    string
		isUnit bool
	}{
		{"decimal<kg>", false},
		{"decimal<kg*m/s^2>", true},
		{"decimal<m^3>", true},
		{"decimal<(kg*m)/s>", true},
		{"Map<string, int>", false},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			file := parse(t, "alias X = "+tt.src)
			arg := file.Decls[0].(*ast.AliasDecl).Target.Args[0]
			if (arg.Unit != nil) != tt.isUnit {
				t.Errorf("unit = %v, want %v (arg %+v)", arg.Unit != nil, tt.isUnit, arg)
			}
		})
	}
}

func TestUnitExponentMustBeInteger(t *testing.T) {
	if msg := parseErr(t, "unit Bad = kg^x"); !strings.Contains(msg, "expected a unit exponent") {
		t.Errorf("error = %s", msg)
	}
}
