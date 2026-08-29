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
			if len(r.Args) != 2 || r.Args[1].List == nil {
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
}`)

	members := file.Decls[0].(*ast.StructDecl).Members
	if len(members) != 5 {
		t.Fatalf("got %d members, want 5", len(members))
	}
	for i, want := range []string{"value", "type", "unit", "target", "include"} {
		f, ok := members[i].(*ast.Field)
		if !ok {
			t.Fatalf("member %d is %T, want a field", i, members[i])
		}
		if f.N != want {
			t.Errorf("member %d = %q, want %q", i, f.N, want)
		}
	}
}

// `include Foo` is still an include, not a field named include.
func TestIncludeStillParses(t *testing.T) {
	file := parse(t, `value V { include Timestamps }`)
	if _, ok := file.Decls[0].(*ast.StructDecl).Members[0].(*ast.Include); !ok {
		t.Error("include parsed as something else")
	}
}
