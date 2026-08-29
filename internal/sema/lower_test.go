package sema

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

// preamble declares the prelude names sugar lowers to. Phase 4 loads
// prelude/std.tdl for real; until then a file that uses `[T]` has to
// declare `List` itself, which is what "no type is built in" means.
const preamble = `primitive string
primitive T
primitive K
primitive V
primitive List: type -> type
primitive Set: type -> type
primitive Map: type -> type -> type
primitive Option: type -> type
primitive Nullable: type -> type
`

func lower(t *testing.T, src string) *ir.Model {
	t.Helper()
	file, err := parser.Parse("test.tdl", strings.NewReader(preamble+src))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	model, diags := Lower(file)
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	return model
}

func lowerDiags(t *testing.T, src string) Diagnostics {
	t.Helper()
	file, err := parser.Parse("test.tdl", strings.NewReader(preamble+src))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	_, diags := Lower(file)
	if len(diags) == 0 {
		t.Fatal("expected diagnostics, got none")
	}
	return diags
}

// declCount is how many declarations the preamble contributes.
const declCount = 9

func TestDeclarationTable(t *testing.T) {
	model := lower(t, `
alias Names = [string]
type Email: string

entity Order {
  key id: string
  items: [string] owned
}

value Money { amount: string }

mixin Timestamps { createdAt: string }

enum Status { Draft Placed }
`)

	if got := len(model.GetDecls()); got != declCount+6 {
		t.Fatalf("got %d decls, want %d", got, declCount+6)
	}

	order, id, ok := model.FindDecl("Order")
	if !ok {
		t.Fatal("Order is missing from the table")
	}
	if model.Decl(id) != order {
		t.Error("the ID does not resolve back to the declaration")
	}
	if order.GetStructure().GetKind() != ir.StructKind_STRUCT_KIND_ENTITY {
		t.Errorf("Order kind = %v", order.GetStructure().GetKind())
	}
	if got := len(order.Fields()); got != 2 {
		t.Errorf("Order has %d fields, want 2", got)
	}
	if !order.Fields()[0].GetKey() {
		t.Error("id lost its key marker")
	}
	if !order.Fields()[1].GetOwned() {
		t.Error("items lost its owned marker")
	}

	money, _, _ := model.FindDecl("Money")
	if money.GetStructure().GetKind() != ir.StructKind_STRUCT_KIND_VALUE {
		t.Error("Money is not a value")
	}
	mixin, _, _ := model.FindDecl("Timestamps")
	if mixin.GetStructure().GetKind() != ir.StructKind_STRUCT_KIND_MIXIN {
		t.Error("Timestamps is not a mixin")
	}

	status, _, _ := model.FindDecl("Status")
	if got := len(status.GetEnumeration().GetVariants()); got != 2 {
		t.Errorf("Status has %d variants, want 2", got)
	}
}

// A reference to a declaration further down the file resolves like one
// above it, which is why the table is collected before anything is lowered.
func TestForwardReference(t *testing.T) {
	model := lower(t, `
value A { b: B }
value B { name: string }
`)

	a, _, _ := model.FindDecl("A")
	ctor := model.Type(a.Fields()[0].GetType()).GetCtor()
	if !ctor.Resolved() || ctor.GetName() != "B" {
		t.Errorf("B did not resolve: %+v", ctor)
	}
}

func TestSugarLowering(t *testing.T) {
	tests := []struct {
		src   string
		ctor  string
		wrote ir.SyntacticForm
	}{
		{"[T]", "List", ir.SyntacticForm_SYNTACTIC_FORM_BRACKETS},
		{"{T}", "Set", ir.SyntacticForm_SYNTACTIC_FORM_BRACES},
		{"{K -> V}", "Map", ir.SyntacticForm_SYNTACTIC_FORM_ARROW},
		{"T?", "Option", ir.SyntacticForm_SYNTACTIC_FORM_QUESTION},
		{"T | null", "Nullable", ir.SyntacticForm_SYNTACTIC_FORM_OR_NULL},
		{"List<T>", "List", ir.SyntacticForm_SYNTACTIC_FORM_NAMED},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			model := lower(t, "alias X = "+tt.src)
			x, _, _ := model.FindDecl("X")
			target := x.GetAlias().GetTarget()
			ty := model.Type(target)
			if got := ty.GetCtor().GetName(); got != tt.ctor {
				t.Errorf("ctor = %q, want %q", got, tt.ctor)
			}
			if ty.GetWrote() != tt.wrote {
				t.Errorf("wrote = %v, want %v", ty.GetWrote(), tt.wrote)
			}
		})
	}
}

// `[T]` and `List<T>` mean the same type and differ only in how they were
// written, so they stay separate entries: folding them would throw the
// distinction away the moment a model used both.
func TestSyntacticFormIsPartOfIdentity(t *testing.T) {
	model := lower(t, `
alias A = [T]
alias B = List<T>
`)

	da, _, _ := model.FindDecl("A")
	db, _, _ := model.FindDecl("B")
	a := da.GetAlias().GetTarget()
	b := db.GetAlias().GetTarget()
	if a.GetIndex() == b.GetIndex() {
		t.Error("the bracket and named forms share an entry")
	}
	if model.Type(a).GetCtor().GetName() != model.Type(b).GetCtor().GetName() {
		t.Error("the two forms lowered to different constructors")
	}
}

// `T? | null` is Nullable<Option<T>>, two entries, each recording its form.
func TestOptionalAndNullable(t *testing.T) {
	model := lower(t, `alias X = T? | null`)

	x, _, _ := model.FindDecl("X")
	outer := model.Type(x.GetAlias().GetTarget())
	if outer.GetCtor().GetName() != "Nullable" {
		t.Fatalf("outer ctor = %q", outer.GetCtor().GetName())
	}
	inner := model.Type(outer.GetArgs()[0])
	if inner.GetCtor().GetName() != "Option" {
		t.Errorf("inner ctor = %q", inner.GetCtor().GetName())
	}
	if inner.GetWrote() != ir.SyntacticForm_SYNTACTIC_FORM_QUESTION {
		t.Errorf("inner form = %v", inner.GetWrote())
	}
}

// Lowering the same type twice yields one entry, which is what makes an ID
// comparison a type comparison.
func TestInterning(t *testing.T) {
	model := lower(t, `
value A { x: [string] }
value B { y: [string] }
`)

	da, _, _ := model.FindDecl("A")
	db, _, _ := model.FindDecl("B")
	a := da.Fields()[0].GetType()
	b := db.Fields()[0].GetType()
	if a.GetIndex() != b.GetIndex() {
		t.Errorf("the same type interned twice: %d and %d", a.GetIndex(), b.GetIndex())
	}

	// [string] is List<string>, so string is in the table too, and nothing
	// else is.
	if got := len(model.GetTypes()); got != 2 {
		t.Errorf("type table holds %d entries, want 2", got)
	}
}

// An unresolved name is a diagnostic, and the ID keeps the text so later
// output can say what was written.
func TestUndefinedName(t *testing.T) {
	diags := lowerDiags(t, `value A { x: Missing }`)
	if !strings.Contains(diags.Error(), "undefined: Missing") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestSourceFidelity(t *testing.T) {
	model := lower(t, `
/// An order someone placed.
deprecated("use PurchaseOrder")
entity Order {
  /// What identifies it.
  key id: string
  deprecated legacy: string
}
`)

	order, _, _ := model.FindDecl("Order")
	m := order.GetMeta()
	if len(m.GetDoc()) != 1 || m.GetDoc()[0] != "An order someone placed." {
		t.Errorf("doc = %q", m.GetDoc())
	}
	if !m.IsDeprecated() || m.GetDeprecated().GetReason() != "use PurchaseOrder" {
		t.Errorf("deprecation = %+v", m.GetDeprecated())
	}
	// The position is the declaration keyword, not the doc comment or the
	// deprecation that precede it.
	if m.GetPosition().GetLine() != declCount+4 {
		t.Errorf("position = %+v", m.GetPosition())
	}

	fields := order.Fields()
	if len(fields[0].GetMeta().GetDoc()) != 1 {
		t.Error("the field lost its doc comment")
	}
	if !fields[1].GetMeta().IsDeprecated() {
		t.Error("the field lost its deprecation")
	}
	if fields[0].GetMeta().GetOrder() != 0 || fields[1].GetMeta().GetOrder() != 1 {
		t.Error("declaration order was not preserved")
	}
}

func TestDuplicateDeclaration(t *testing.T) {
	diags := lowerDiags(t, `
value A { x: string }
value A { y: string }
`)
	if !strings.Contains(diags.Error(), "declared twice") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// Units are deferred, and lowering says so rather than dropping them.
func TestUnitArgumentsAreRejected(t *testing.T) {
	diags := lowerDiags(t, `
primitive decimal
unit kg
unit m
unit s
value W { net: decimal<kg*m/s^2> }`)
	if !strings.Contains(diags.Error(), "unit arguments are not lowered yet") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestUnloweredDeclarations(t *testing.T) {
	diags := lowerDiags(t, `class Auditable { createdAt: string }`)
	if !strings.Contains(diags.Error(), "not lowered yet") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// A type parameter shadows a declaration of the same name, inside the
// declaration that declares it and nowhere else.
func TestTypeParameterShadowing(t *testing.T) {
	model := lower(t, `
value Box<string> { held: string }
value Plain { held: string }
`)

	box, _, _ := model.FindDecl("Box")
	held := model.Type(box.Fields()[0].GetType())
	if held.GetParam() == nil {
		t.Fatalf("inside Box, string is not the parameter: %+v", held)
	}
	if held.GetParam().GetName() != "string" || held.GetParam().GetOwner().GetName() != "Box" {
		t.Errorf("param = %+v", held.GetParam())
	}

	plain, _, _ := model.FindDecl("Plain")
	outside := model.Type(plain.Fields()[0].GetType())
	if outside.GetParam() != nil {
		t.Error("the parameter escaped the declaration that declares it")
	}
	if outside.GetCtor().GetName() != "string" {
		t.Errorf("outside Box, string = %+v", outside.GetCtor())
	}
}

func TestHigherKindedParameterApplied(t *testing.T) {
	model := lower(t, `value Collection<f, E> { items: f<E> }`)

	c, _, _ := model.FindDecl("Collection")
	items := model.Type(c.Fields()[0].GetType())
	if items.GetParam().GetName() != "f" {
		t.Fatalf("ctor = %+v", items)
	}
	if len(items.GetArgs()) != 1 {
		t.Fatalf("got %d args, want 1", len(items.GetArgs()))
	}
	if arg := model.Type(items.GetArgs()[0]); arg.GetParam().GetName() != "E" {
		t.Errorf("argument = %+v", arg)
	}
}

func TestDuplicateTypeParameter(t *testing.T) {
	diags := lowerDiags(t, `value Holder<P, P> { x: P }`)
	if !strings.Contains(diags.Error(), "type parameter P is declared twice") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestDuplicateField(t *testing.T) {
	diags := lowerDiags(t, `value Holder { x: string x: string }`)
	if !strings.Contains(diags.Error(), "field x is declared twice") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// Entities may be mutually recursive without restriction: a cycle between
// them is a graph of references, which every backend can represent.
func TestEntityRecursionAllowed(t *testing.T) {
	lower(t, `
entity Order { items: [LineItem] owned self: Order }
entity LineItem { order: Order }
`)
}

// A value may reach itself only through a collection or an optional.
func TestValueRecursion(t *testing.T) {
	lower(t, `
value Ok { next: Ok? children: [Ok] byName: {string -> Ok} }
`)

	for _, src := range []string{
		`value Node { next: Node }`,
		`value A { b: B }
value B { a: A }`,
		`enum Tree { Branch { left: Tree } }`,
	} {
		t.Run(src, func(t *testing.T) {
			diags := lowerDiags(t, src)
			if !strings.Contains(diags.Error(), "contain") {
				t.Errorf("diagnostics = %v", diags)
			}
		})
	}
}

// Aliases are expanded rather than referenced, so no cycle terminates,
// not even one through a collection.
func TestAliasRecursion(t *testing.T) {
	for _, src := range []string{
		`alias A = A`,
		`alias A = [A]`,
		`alias A = B
alias B = A`,
	} {
		t.Run(src, func(t *testing.T) {
			diags := lowerDiags(t, src)
			if !strings.Contains(diags.Error(), "contain") {
				t.Errorf("diagnostics = %v", diags)
			}
		})
	}
}

func TestImportsNotResolvedYet(t *testing.T) {
	diags := lowerDiags(t, `import "common.tdl" as common`)
	if !strings.Contains(diags.Error(), "imports are not resolved yet") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestQualifiedNameNotResolvedYet(t *testing.T) {
	diags := lowerDiags(t, `value Holder { a: common.Address }`)
	if !strings.Contains(diags.Error(), "not resolved yet") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// Every diagnostic in a pass is reported, not just the first.
func TestDiagnosticsAccumulate(t *testing.T) {
	diags := lowerDiags(t, `value Holder { a: Missing b: AlsoMissing }`)
	if len(diags) != 2 {
		t.Errorf("got %d diagnostics, want 2: %v", len(diags), diags)
	}
}

// An instance names the class it is about, not itself, so several
// instances of one class do not collide and none collides with the class.
func TestInstancesAreNotTypeNames(t *testing.T) {
	diags := lowerDiags(t, `
class Auditable { createdAt: string }
instance Auditable for A
instance Auditable for B
value A { x: string }
value B { x: string }
`)

	for _, d := range diags {
		if strings.Contains(d.Msg, "declared twice") {
			t.Errorf("instances collided in the type namespace: %s", d.Error())
		}
	}
}

func TestTargetsAreNotTypeNames(t *testing.T) {
	diags := lowerDiags(t, `
value go { x: string }
target go for p { out("./gen") }
`)

	for _, d := range diags {
		if strings.Contains(d.Msg, "declared twice") {
			t.Errorf("a target collided with a type: %s", d.Error())
		}
	}
}
