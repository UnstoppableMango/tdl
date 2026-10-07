package sema

import (
	"fmt"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

// preamble declares the placeholder types these tests use; the prelude
// supplies the rest.
const preamble = `package p

primitive T
primitive K
primitive V
primitive E
`

// preambleLines is how many lines the preamble adds before a test's source.
const preambleLines = 6

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

func TestDeclarationTable(t *testing.T) {
	model := lower(t, `
alias Names = [string]
type Email: string

type Order: Entity {
  id: string
  items: [string] owned
}

type Money { amount: string }

mixin Timestamps { createdAt: string }

enum Status { Draft Placed }
`)

	for _, name := range []string{"Names", "Email", "Order", "Money", "Timestamps", "Status"} {
		if _, _, ok := model.FindDecl(name); !ok {
			t.Errorf("%s is missing from the table", name)
		}
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

func TestForwardReference(t *testing.T) {
	model := lower(t, `
type A { b: B }
type B { name: string }
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

// `[T]` and `List<T>` stay separate entries.
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

// `T? | null` is Nullable<Option<T>>, each entry recording its form.
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

// Lowering the same type twice yields one entry.
func TestInterning(t *testing.T) {
	model := lower(t, `
type A { x: [string] }
type B { y: [string] }
`)

	da, _, _ := model.FindDecl("A")
	db, _, _ := model.FindDecl("B")
	a := da.Fields()[0].GetType()
	b := db.Fields()[0].GetType()
	if a.GetIndex() != b.GetIndex() {
		t.Errorf("the same type interned twice: %d and %d", a.GetIndex(), b.GetIndex())
	}

	// One entry, however many times it was written.
	var listOfString int
	for _, ty := range model.GetTypes() {
		if ty.GetCtor().GetName() == "List" {
			listOfString++
		}
	}
	if listOfString != 1 {
		t.Errorf("List<string> has %d entries, want 1", listOfString)
	}
}

// An unresolved name is a diagnostic, and the ID keeps the text.
func TestUndefinedName(t *testing.T) {
	diags := lowerDiags(t, `type A { x: Missing }`)
	if !strings.Contains(diags.Error(), "undefined: Missing") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestSourceFidelity(t *testing.T) {
	model := lower(t, `
/// An order someone placed.
deprecated("use PurchaseOrder")
type Order: Entity {
  /// What identifies it.
  id: string
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
	// The position is the declaration keyword, not the doc comment or
	// deprecation before it.
	if m.GetPosition().GetLine() != preambleLines+4 {
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
type A { x: string }
type A { y: string }
`)
	if !strings.Contains(diags.Error(), "declared twice") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// decimal<N> and decimal<kg*m/s^2> are one entry in the type table.
func TestUnitSpellingsInternTogether(t *testing.T) {
	model := lower(t, `
primitive decimal
unit kg
unit m
unit s
unit N = kg*m/s^2
type W {
  named: decimal<N>
  written: decimal<kg*m/s^2>
  parenthesized: decimal<(kg*m)/s^2>
  cancelled: decimal<kg*m/(s^2*m)*m>
  other: decimal<kg>
}`)

	decl, _, _ := model.FindDecl("W")
	fields := decl.GetStructure().GetFields()
	first := fields[0].GetType().GetIndex()
	for _, f := range fields[:4] {
		if got := f.GetType().GetIndex(); got != first {
			t.Errorf("%s = types[%d], want types[%d]", f.GetMeta().GetName(), got, first)
		}
	}
	if got := fields[4].GetType().GetIndex(); got == first {
		t.Errorf("decimal<kg> interned with decimal<N> at types[%d]", got)
	}
}

// A unit argument is a type table entry with `unit` set instead of `ctor`.
func TestUnitArgumentIsAType(t *testing.T) {
	model := lower(t, `
primitive decimal
unit kg
type W { net: decimal<kg> }`)

	decl, _, _ := model.FindDecl("W")
	arg := model.Type(decl.GetStructure().GetFields()[0].GetType()).GetArgs()[0]
	unit := model.Type(arg).GetUnit()
	if unit == nil {
		t.Fatalf("argument is not a unit: %v", model.Type(arg))
	}
	if got := model.Unit(unit).GetDims(); len(got) != 1 || got[0].GetBase().GetName() != "kg" {
		t.Errorf("dims = %v", got)
	}
}

func TestBaseUnitMeasuresItself(t *testing.T) {
	model := lower(t, `unit kg`)

	decl, _, ok := model.FindDecl("kg")
	if !ok || decl.GetUnit() == nil {
		t.Fatalf("kg did not lower to a unit: %v", decl)
	}
	if !decl.GetUnit().GetBase() {
		t.Error("kg is not marked a base unit")
	}

	dims := model.Unit(decl.GetUnit().GetUnit()).GetDims()
	if len(dims) != 1 || dims[0].GetBase().GetName() != "kg" || dims[0].GetExponent() != 1 {
		t.Errorf("dims = %v", dims)
	}
}

func TestDerivedUnitResolvesForward(t *testing.T) {
	model := lower(t, `
unit N = kg*m/s^2
unit kg
unit m
unit s`)

	decl, _, _ := model.FindDecl("N")
	if got := dimsText(t, model, decl); got != "kg^1 m^1 s^-2" {
		t.Errorf("N = %s", got)
	}
}

// An exponent applies to a parenthesized group the way it does to a name.
func TestParenthesizedUnitExponent(t *testing.T) {
	model := lower(t, `
unit kg
unit m
unit s
unit Squared = (kg*m)^2
unit Nested = (kg*m^2)^3/s
unit Twice = ((kg/m)^2)^2`)

	for _, c := range []struct{ name, want string }{
		{"Squared", "kg^2 m^2"},
		{"Nested", "kg^3 m^6 s^-1"},
		{"Twice", "kg^4 m^-4"},
	} {
		decl, _, _ := model.FindDecl(c.name)
		if got := dimsText(t, model, decl); got != c.want {
			t.Errorf("%s = %s, want %s", c.name, got, c.want)
		}
	}
}

// The error is at the declaration that closes the cycle.
func TestUnitCycleIsAnError(t *testing.T) {
	diags := lowerDiags(t, `
unit m
unit A = B*m
unit B = A/m`)
	if !strings.Contains(diags.Error(), "defined in terms of itself") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// A unit expression names units and nothing else.
func TestUnitExpressionNamingANonUnit(t *testing.T) {
	diags := lowerDiags(t, `
primitive string
unit A = string`)
	if !strings.Contains(diags.Error(), "string is not a unit") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// A unit losing its name to an earlier declaration never reaches the unit
// table, and a type argument naming it is a type argument.
func TestUnitNameBoundToANonUnit(t *testing.T) {
	file, err := parser.Parse("test.tdl", strings.NewReader(preamble+`
primitive foo
unit foo
type W { a: decimal<foo> }`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	model, diags := Lower(file)

	if !strings.Contains(diags.Error(), "foo is declared twice") {
		t.Errorf("diagnostics = %v", diags)
	}
	decl, id, _ := model.FindDecl("foo")
	if decl.GetUnit() != nil {
		t.Errorf("foo lowered as a unit: %v", decl)
	}
	for _, u := range model.Units {
		if u.GetWrote() == "foo" {
			t.Error("the duplicate unit reached the unit table")
		}
	}

	w, _, _ := model.FindDecl("W")
	arg := model.Type(model.Type(w.GetStructure().GetFields()[0].GetType()).GetArgs()[0])
	if arg.GetUnit() != nil {
		t.Errorf("decimal<foo> took a unit argument: %v", arg)
	}
	if got := arg.GetCtor(); got.GetIndex() != id.GetIndex() {
		t.Errorf("argument ctor = %v, want the primitive at %v", got, id)
	}
}

// A duplicate unit is not a unit inside a unit expression either.
func TestUnitExpressionNamingADuplicateUnit(t *testing.T) {
	diags := lowerDiags(t, `
primitive foo
unit foo
unit A = foo`)
	if !strings.Contains(diags.Error(), "foo is not a unit") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestUndefinedUnit(t *testing.T) {
	diags := lowerDiags(t, `unit A = nope`)
	if !strings.Contains(diags.Error(), "undefined unit: nope") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// dimsText renders a declaration's dimensions for a test failure message.
func dimsText(t *testing.T, model *ir.Model, decl *ir.Decl) string {
	t.Helper()
	var parts []string
	for _, d := range model.Unit(decl.GetUnit().GetUnit()).GetDims() {
		parts = append(parts, fmt.Sprintf("%s^%d", d.GetBase().GetName(), d.GetExponent()))
	}
	return strings.Join(parts, " ")
}

// A type parameter shadows a declaration only inside its own declaration.
func TestTypeParameterShadowing(t *testing.T) {
	model := lower(t, `
type Box<string> { held: string }
type Plain { held: string }
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
	model := lower(t, `type Collection<f, E> { items: f<E> }`)

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
	diags := lowerDiags(t, `type Holder<P, P> { x: P }`)
	if !strings.Contains(diags.Error(), "type parameter P is declared twice") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestDuplicateField(t *testing.T) {
	diags := lowerDiags(t, `type Holder { x: string x: string }`)
	if !strings.Contains(diags.Error(), "field x is declared twice") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// Entities may be mutually recursive without restriction.
func TestEntityRecursionAllowed(t *testing.T) {
	lower(t, `
type Order: Entity { items: [LineItem] owned self: Order }
type LineItem: Entity { order: Order }
`)
}

// A value may reach itself only through a collection or an optional.
func TestValueRecursion(t *testing.T) {
	lower(t, `
type Ok { next: Ok? children: [Ok] byName: {string -> Ok} }
`)

	for _, src := range []string{
		`type Node { next: Node }`,
		`type A { b: B }
type B { a: A }`,
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

// A struct's kind comes from satisfying the prelude's Entity, however that
// arises, and only an entity is exempt from the value recursion rule.
func TestEntityKindFromConformance(t *testing.T) {
	model := lower(t, `
class Aggregate: Entity { }
type Direct: Entity { self: Direct }
type ViaClass: Aggregate { self: ViaClass }
type ViaInstance { self: ViaInstance }
instance Entity for ViaInstance
type Plain { }
mixin Stamped: Entity { }
`)

	for name, want := range map[string]ir.StructKind{
		"Direct":      ir.StructKind_STRUCT_KIND_ENTITY,
		"ViaClass":    ir.StructKind_STRUCT_KIND_ENTITY,
		"ViaInstance": ir.StructKind_STRUCT_KIND_ENTITY,
		"Plain":       ir.StructKind_STRUCT_KIND_VALUE,
		"Stamped":     ir.StructKind_STRUCT_KIND_MIXIN,
	} {
		decl, _, _ := model.FindDecl(name)
		if got := decl.GetStructure().GetKind(); got != want {
			t.Errorf("%s kind = %v, want %v", name, got, want)
		}
	}
}

// Conforming to a file's class named Entity says nothing about identity.
func TestShadowedEntityIsNotIdentity(t *testing.T) {
	diags := lowerDiags(t, `
class Entity { }
type Node: Entity { next: Node }`)
	if !strings.Contains(diags.Error(), "Node contains itself") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// An enum conforming to Entity still holds its variants' fields inline,
// so a cycle through one is not a reference.
func TestEntityEnumRecursion(t *testing.T) {
	diags := lowerDiags(t, `enum Tree: Entity { Branch { left: Tree } }`)
	if !strings.Contains(diags.Error(), "Tree contains itself") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// An alias cycle is an error even through a collection.
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

// Without a loader, an import is a diagnostic.
func TestImportNeedsALoader(t *testing.T) {
	diags := lowerDiags(t, `import "common.tdl" as common`)
	if !strings.Contains(diags.Error(), "imports need a loader") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestUndefinedImportAlias(t *testing.T) {
	diags := lowerDiags(t, `type Holder { a: common.Address }`)
	if !strings.Contains(diags.Error(), "undefined import alias: common") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestDiagnosticsAccumulate(t *testing.T) {
	diags := lowerDiags(t, `type Holder { a: Missing b: AlsoMissing }`)
	if len(diags) != 2 {
		t.Errorf("got %d diagnostics, want 2: %v", len(diags), diags)
	}
}

// Several instances of one class collide neither with each other nor
// with the class.
func TestInstancesAreNotTypeNames(t *testing.T) {
	model := lower(t, `
class Auditable { createdAt: string }
instance Auditable for A
instance Auditable for B
type A { createdAt: string }
type B { createdAt: string }
`)

	if got := len(model.GetInstances()); got != 2 {
		t.Errorf("got %d instances, want 2", got)
	}
}

// A target block does not collide with a declaration of the same name.
func TestTargetsAreNotTypeNames(t *testing.T) {
	model := lower(t, `
type go { x: string }
target go for p { out("./gen") }
`)

	if _, _, ok := model.FindDecl("go"); !ok {
		t.Error("the value named go is missing")
	}
	if got := len(model.GetTargets()); got != 1 {
		t.Errorf("got %d target blocks, want 1", got)
	}
}

// `[T]` and `List<T>` are two entries with one name.
func TestTypeIDNameIsReadable(t *testing.T) {
	model := lower(t, `
alias A = [string]
alias B = List<string>
alias C = {string -> [string]}
`)

	da, _, _ := model.FindDecl("A")
	db, _, _ := model.FindDecl("B")
	if got := da.GetAlias().GetTarget().GetName(); got != "List<string>" {
		t.Errorf("[string] is named %q", got)
	}
	if got := db.GetAlias().GetTarget().GetName(); got != "List<string>" {
		t.Errorf("List<string> is named %q", got)
	}
	if da.GetAlias().GetTarget().GetIndex() == db.GetAlias().GetTarget().GetIndex() {
		t.Error("the two forms share an entry")
	}

	dc, _, _ := model.FindDecl("C")
	if got := dc.GetAlias().GetTarget().GetName(); got != "Map<string, List<string>>" {
		t.Errorf("nested name = %q", got)
	}
}

// Two types differing only in an argument's written form stay apart.
func TestInterningDistinguishesArgumentForms(t *testing.T) {
	model := lower(t, `
alias A = Set<[string]>
alias B = Set<List<string>>
`)

	da, _, _ := model.FindDecl("A")
	db, _, _ := model.FindDecl("B")
	if da.GetAlias().GetTarget().GetIndex() == db.GetAlias().GetTarget().GetIndex() {
		t.Error("Set<[string]> and Set<List<string>> share an entry")
	}
}

// `[T]` means whatever the loaded prelude says `List` is.
func TestPreludeIsReplaceable(t *testing.T) {
	file, err := parser.Parse("test.tdl", strings.NewReader(`type V { items: [string] }`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	replacement := `package other
primitive string
primitive List: type -> type
primitive Set: type -> type
primitive Map: type -> type -> type
primitive Option: type -> type
primitive Nullable: type -> type
`
	model, diags := Lower(file, WithPrelude("other.tdl", replacement))
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	v, _, _ := model.FindDecl("V")
	items := model.Type(v.Fields()[0].GetType())
	ctor := model.Decl(items.GetCtor())
	if ctor == nil {
		t.Fatal("List did not resolve through the replacement prelude")
	}
	if got := ctor.GetMeta().GetPosition().GetFilename(); got != "other.tdl" {
		t.Errorf("List came from %q, want other.tdl", got)
	}
}

func TestFileShadowsPrelude(t *testing.T) {
	model := lower(t, `
primitive string
type Shadowed { s: string }
`)

	v, _, _ := model.FindDecl("Shadowed")
	ctor := model.Decl(model.Type(v.Fields()[0].GetType()).GetCtor())
	if got := ctor.GetMeta().GetPosition().GetFilename(); got != "test.tdl" {
		t.Errorf("string resolved to %q, want the file's own", got)
	}
}

// A qualified reference carries the dependency's package and is not
// inlined.
func TestCrossPackageReference(t *testing.T) {
	file, err := parser.Parse("main.tdl", strings.NewReader(`
package shop

import "common.tdl" as common

type Order { ship: common.Address }
`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	model, diags := Lower(file, WithLoader(MapLoader{
		"common.tdl": "package shop.common\ntype Address { line1: string }\n",
	}))
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := len(model.GetImports()); got != 1 {
		t.Fatalf("got %d imports, want 1", got)
	}
	if got := model.GetImports()[0].GetPackage(); got != "shop.common" {
		t.Errorf("import package = %q", got)
	}

	order, _, _ := model.FindDecl("Order")
	ship := model.Type(order.Fields()[0].GetType())
	if ship.GetExtern() == nil {
		t.Fatalf("the reference is not an extern: %+v", ship)
	}
	ext := model.GetExterns()[ship.GetExtern().GetIndex()]
	if ext.GetPackage() != "shop.common" || ext.GetName() != "Address" {
		t.Errorf("extern = %+v", ext)
	}

	if _, _, ok := model.FindDecl("Address"); ok {
		t.Error("the dependency's declaration was inlined")
	}
}

// A `_` import merges the dependency's exported names only.
func TestUnderscoreImportMerges(t *testing.T) {
	file, err := parser.Parse("main.tdl", strings.NewReader(`
import "common.tdl" as _

type Order { ship: Address }
`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	model, diags := Lower(file, WithLoader(MapLoader{
		"common.tdl": "package shop.common\ntype Address { line1: string }\ntype internal { x: string }\n",
	}))
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	order, _, _ := model.FindDecl("Order")
	if model.Type(order.Fields()[0].GetType()).GetExtern() == nil {
		t.Error("the merged name did not become an extern")
	}

	for _, e := range model.GetExterns() {
		if e.GetName() == "internal" {
			t.Error("a package-private name was merged")
		}
	}
}

// A primitive is exported whatever its case.
func TestUnderscoreImportMergesPrimitive(t *testing.T) {
	file, err := parser.Parse("main.tdl", strings.NewReader(`
import "dep.tdl" as _

type Reading { count: int32 }
`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	model, diags := Lower(file, WithLoader(MapLoader{
		"dep.tdl": "package acme.scalar\nprimitive int32\n",
	}))
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	reading, _, _ := model.FindDecl("Reading")
	count := model.Type(reading.Fields()[0].GetType())
	if count.GetExtern() == nil {
		t.Fatalf("the reference is not an extern: %+v", count)
	}
	ext := model.GetExterns()[count.GetExtern().GetIndex()]
	if ext.GetPackage() != "acme.scalar" || ext.GetName() != "int32" {
		t.Errorf("extern = %+v, want acme.scalar.int32", ext)
	}
}

// A local primitive repeating one a `_` import merges is a collision.
func TestUnderscoreImportPrimitiveCollides(t *testing.T) {
	file, err := parser.Parse("main.tdl", strings.NewReader(`
import "dep.tdl" as _

primitive int32
`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	_, diags := Lower(file, WithLoader(MapLoader{
		"dep.tdl": "package acme.scalar\nprimitive int32\n",
	}))
	if !strings.Contains(diags.Error(), "int32 is declared twice") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// A unit is exported whatever its case, and a type argument can name it.
func TestUnderscoreImportMergesUnit(t *testing.T) {
	file, err := parser.Parse("main.tdl", strings.NewReader(`
import "dep.tdl" as _

type Length { value: decimal<m> }
`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	model, diags := Lower(file, WithLoader(MapLoader{
		"dep.tdl": "package acme.si\nunit m\n",
	}))
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	length, _, _ := model.FindDecl("Length")
	args := model.Type(length.Fields()[0].GetType()).GetArgs()
	if len(args) != 1 {
		t.Fatalf("decimal<m> has %d arguments, want 1", len(args))
	}
	arg := model.Type(args[0])
	if arg.GetExtern() == nil {
		t.Fatalf("the unit argument is not an extern: %+v", arg)
	}
	ext := model.GetExterns()[arg.GetExtern().GetIndex()]
	if ext.GetPackage() != "acme.si" || ext.GetName() != "m" {
		t.Errorf("extern = %+v, want acme.si.m", ext)
	}
}

func TestUnderscoreImportSkipsLowerCaseType(t *testing.T) {
	file, err := parser.Parse("main.tdl", strings.NewReader(`
import "dep.tdl" as _

type Order { part: widget }
`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	_, diags := Lower(file, WithLoader(MapLoader{
		"dep.tdl": "package acme.parts\ntype widget { }\n",
	}))
	if !strings.Contains(diags.Error(), "undefined: widget") {
		t.Errorf("diagnostics = %v, want undefined: widget", diags)
	}
}

func TestImportCycle(t *testing.T) {
	file, err := parser.Parse("a.tdl", strings.NewReader(`import "b.tdl" as b`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	_, diags := Lower(file, WithLoader(MapLoader{
		"b.tdl": `import "c.tdl" as c`,
		"c.tdl": `import "b.tdl" as b`,
	}))
	if !strings.Contains(diags.Error(), "import cycle") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestMissingImport(t *testing.T) {
	file, err := parser.Parse("a.tdl", strings.NewReader(`import "gone.tdl" as gone`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	_, diags := Lower(file, WithLoader(MapLoader{}))
	if !strings.Contains(diags.Error(), "cannot read import") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestExternsAreInterned(t *testing.T) {
	file, err := parser.Parse("main.tdl", strings.NewReader(`
import "common.tdl" as common

type A { x: common.Address }
type B { y: common.Address }
`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	model, _ := Lower(file, WithLoader(MapLoader{
		"common.tdl": "package shop.common\ntype Address { line1: string }\n",
	}))
	if got := len(model.GetExterns()); got != 1 {
		t.Errorf("got %d externs, want 1", got)
	}
}

func TestClassLowering(t *testing.T) {
	model := lower(t, `
class Timestamped { createdAt: string }

class Auditable: Timestamped {
  type Cursor: type
  updatedAt: string
}

class Projection<from, to> | from -> to { }
`)

	auditable, _, _ := model.FindDecl("Auditable")
	c := auditable.GetClass()
	if len(c.GetRequiresClasses()) != 1 {
		t.Errorf("Auditable requires %d classes, want 1", len(c.GetRequiresClasses()))
	}
	if len(c.GetAssocTypes()) != 1 || c.GetAssocTypes()[0].GetMeta().GetName() != "Cursor" {
		t.Errorf("assoc types = %+v", c.GetAssocTypes())
	}
	if len(c.GetFields()) != 1 {
		t.Errorf("Auditable has %d fields, want 1", len(c.GetFields()))
	}

	proj, _, _ := model.FindDecl("Projection")
	deps := proj.GetClass().GetFunDeps()
	if len(deps) != 1 || deps[0].GetFrom()[0] != "from" || deps[0].GetTo()[0] != "to" {
		t.Errorf("fundeps = %+v", deps)
	}
}

// `instance C for T` lowers to the same form as `instance C<T>`.
func TestInstanceFormsNormalize(t *testing.T) {
	model := lower(t, `
class Auditable { createdAt: string }
type A { createdAt: string }

instance Auditable<A>
instance Auditable for A
`)

	if got := len(model.GetInstances()); got != 2 {
		t.Fatalf("got %d instances, want 2", got)
	}
	for i, inst := range model.GetInstances() {
		if got := len(inst.GetClass().GetArgs()); got != 1 {
			t.Errorf("instance %d has %d arguments, want 1", i, got)
		}
	}
}

// Satisfaction comes from declared conformance and ground instances and
// closes over required classes.
func TestSatisfaction(t *testing.T) {
	model := lower(t, `
class Timestamped { createdAt: string }
class Auditable: Timestamped { updatedAt: string }

type Declared: Entity, Auditable {
  id: string
  createdAt: string
  updatedAt: string
}
type ByInstance {
  createdAt: string
  updatedAt: string
}
type Neither { x: string }

instance Auditable<ByInstance>
`)

	_, auditable, _ := model.FindDecl("Auditable")
	names := satisfyingNames(model, auditable)
	if !contains(names, "Declared") || !contains(names, "ByInstance") {
		t.Errorf("Auditable is satisfied by %v", names)
	}
	if contains(names, "Neither") {
		t.Errorf("Neither satisfies Auditable: %v", names)
	}

	// Auditable requires Timestamped, so satisfying one satisfies the other.
	_, timestamped, _ := model.FindDecl("Timestamped")
	if names := satisfyingNames(model, timestamped); !contains(names, "Declared") {
		t.Errorf("Timestamped is satisfied by %v, want Declared among them", names)
	}
}

func TestIncludeDoesNotConfer(t *testing.T) {
	model := lower(t, `
class Auditable { createdAt: string }
mixin Stamps: Auditable { createdAt: string }
type Uses { include Stamps }
`)

	_, auditable, _ := model.FindDecl("Auditable")
	if names := satisfyingNames(model, auditable); contains(names, "Uses") {
		t.Errorf("including a mixin conferred conformance: %v", names)
	}
}

// A conditional instance is recorded but names no satisfying declaration.
func TestConditionalInstanceNotIndexed(t *testing.T) {
	model := lower(t, `
class Auditable { }
type Page<T> { items: [T] }

instance <T> Auditable<Page<T>> requires Auditable<T>
`)

	if got := len(model.GetInstances()); got != 1 {
		t.Fatalf("got %d instances, want 1", got)
	}
	_, auditable, _ := model.FindDecl("Auditable")
	if names := satisfyingNames(model, auditable); len(names) != 0 {
		t.Errorf("a conditional instance was indexed: %v", names)
	}
}

func TestIncludeExpansion(t *testing.T) {
	model := lower(t, `
mixin Inner { a: string }
mixin Outer { include Inner b: string }
type Uses { include Outer c: string }
`)

	uses, _, _ := model.FindDecl("Uses")
	var names []string
	for _, f := range uses.Fields() {
		names = append(names, f.GetMeta().GetName())
	}
	if len(names) != 3 {
		t.Fatalf("fields = %v, want three", names)
	}

	for _, f := range uses.Fields() {
		switch f.GetMeta().GetName() {
		case "c":
			if f.GetIncludedFrom() != nil {
				t.Error("a field the declaration wrote records a mixin")
			}
		case "a":
			// A field copied through two mixins records where it started.
			if got := f.GetIncludedFrom().GetName(); got != "Inner" {
				t.Errorf("a came from %q, want Inner", got)
			}
		case "b":
			if got := f.GetIncludedFrom().GetName(); got != "Outer" {
				t.Errorf("b came from %q, want Outer", got)
			}
		}
	}
}

// `include` copies a field's constraints and default through any number
// of mixins.
func TestIncludeCopiesConstraintsAndDefaults(t *testing.T) {
	model := lower(t, `
mixin Audited { note: string where { length(1..280) } = "none" }
mixin Tracked { include Audited }
type Direct { include Audited }
type Nested { include Tracked }
`)

	for _, name := range []string{"Direct", "Nested"} {
		decl, _, _ := model.FindDecl(name)
		if len(decl.Fields()) != 1 {
			t.Fatalf("%s fields = %v, want one", name, decl.Fields())
		}
		note := decl.Fields()[0]
		if cs := note.GetConstraints(); len(cs) != 1 || cs[0].GetName() != "length" {
			t.Errorf("%s.note constraints = %v, want length", name, cs)
		}
		if got := note.GetDefaultValue().GetText(); got != "none" {
			t.Errorf("%s.note default = %q, want \"none\"", name, got)
		}
	}
}

func TestIncludeOfNonMixin(t *testing.T) {
	diags := lowerDiags(t, `
type NotAMixin { x: string }
type Uses { include NotAMixin }
`)
	if !strings.Contains(diags.Error(), "is not a mixin") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// The diagnostic points at the use site.
func TestRequiresCheckedAtUse(t *testing.T) {
	diags := lowerDiags(t, `
class Auditable { createdAt: string }
type Envelope<P> requires Auditable<P> { body: P }
type Plain { x: string }
type Holder { e: Envelope<Plain> }
`)
	if !strings.Contains(diags.Error(), "Plain does not satisfy Auditable") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestRequiresSatisfied(t *testing.T) {
	lower(t, `
class Auditable { createdAt: string }
type Envelope<P> requires Auditable<P> { body: P }
type Audited: Auditable { createdAt: string }
type Holder { e: Envelope<Audited> }
`)
}

// An argument that is still a parameter is not checked.
func TestRequiresDefersToOuterInstantiation(t *testing.T) {
	lower(t, `
class Auditable { createdAt: string }
type Envelope<P> requires Auditable<P> { body: P }
type Outer<P> requires Auditable<P> { e: Envelope<P> }
`)
}

func satisfyingNames(model *ir.Model, class *ir.ID) []string {
	var names []string
	for _, id := range model.Satisfying(class) {
		names = append(names, id.GetName())
	}
	return names
}

func contains(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

// An instantiated type satisfies a class through a conditional instance;
// `Page` alone satisfies nothing.
func TestConditionalInstanceSearch(t *testing.T) {
	model := lower(t, `
class Auditable { }
type Page<P> { items: [P] }
type Audited: Auditable { createdAt: string }
type Plain { x: string }

instance <P> Auditable<Page<P>> requires Auditable<P>

type Uses {
  good: Page<Audited>
  bad: Page<Plain>
}
`)

	_, auditable, _ := model.FindDecl("Auditable")

	var names []string
	for _, id := range model.SatisfyingTypes(auditable) {
		names = append(names, id.GetName())
	}
	if !contains(names, "Page<Audited>") {
		t.Errorf("Page<Audited> is not satisfying: %v", names)
	}
	if contains(names, "Page<Plain>") {
		t.Errorf("Page<Plain> satisfies Auditable: %v", names)
	}
}

// A page of pages of auditable things is auditable.
func TestConditionalInstanceNests(t *testing.T) {
	model := lower(t, `
class Auditable { }
type Page<P> { items: [P] }
type Audited: Auditable { createdAt: string }

instance <P> Auditable<Page<P>> requires Auditable<P>

type Uses { nested: Page<Page<Audited>> }
`)

	_, auditable, _ := model.FindDecl("Auditable")
	var names []string
	for _, id := range model.SatisfyingTypes(auditable) {
		names = append(names, id.GetName())
	}
	if !contains(names, "Page<Page<Audited>>") {
		t.Errorf("the search did not recurse: %v", names)
	}
}

func TestRequiresThroughConditionalInstance(t *testing.T) {
	lower(t, `
class Auditable { }
type Page<P> { items: [P] }
type Audited: Auditable { createdAt: string }
type Envelope<P> requires Auditable<P> { body: P }

instance <P> Auditable<Page<P>> requires Auditable<P>

type Holder { e: Envelope<Page<Audited>> }
`)
}

// The spec's instance head rules are checked where the instance is
// written.
func TestInstanceHeadMustBeAConstructor(t *testing.T) {
	diags := lowerDiags(t, `
class Auditable { createdAt: string }
instance <P> Auditable<P>
`)
	if !strings.Contains(diags.Error(), "must be a type constructor") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestInstanceHeadParametersMustBeDistinct(t *testing.T) {
	diags := lowerDiags(t, `
class Rel<a, b> { }
type Pair<a, b> { x: a y: b }
instance <P> Rel<Pair<P, P>>
`)
	if !strings.Contains(diags.Error(), "repeats the parameter") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestInstanceConditionMustBeSmaller(t *testing.T) {
	diags := lowerDiags(t, `
class Auditable { createdAt: string }
type Page<P> { items: [P] }

instance <P> Auditable<Page<P>> requires Auditable<Page<P>>
`)
	if !strings.Contains(diags.Error(), "would not terminate") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestConstraintsLower(t *testing.T) {
	model := lower(t, `
type Email: string where {
  matches(/^[^@]+@[^@]+$/)
  length(3..254)
  unique
}

type Holder {
  region: string where { oneOf("us-east", "eu-west") }
  ratio: int where { between(0, 100) }
}
`)

	email, _, _ := model.FindDecl("Email")
	cs := email.GetNewtype().GetValueConstraints()
	if len(cs) != 3 {
		t.Fatalf("got %d constraints, want 3", len(cs))
	}
	if cs[0].GetArgs()[0].GetKind() != ir.LiteralKind_LITERAL_KIND_REGEX {
		t.Errorf("matches argument = %v", cs[0].GetArgs()[0].GetKind())
	}
	if r := cs[1].GetArgs()[0].GetRange(); r.GetLow() != 3 || r.GetHigh() != 254 {
		t.Errorf("range = %+v", r)
	}
	if cs[0].GetPosition().GetLine() == 0 {
		t.Error("a constraint lost its position")
	}

	// The set is open: an unknown name lowers without complaint.
	v, _, _ := model.FindDecl("Holder")
	if got := v.Fields()[1].GetConstraints()[0].GetName(); got != "between" {
		t.Errorf("unknown constraint = %q", got)
	}
}

func TestOpenRanges(t *testing.T) {
	model := lower(t, `
type Open: string where { length(1..) }
type Capped: string where { length(..64) }
`)

	open, _, _ := model.FindDecl("Open")
	if r := open.GetNewtype().GetValueConstraints()[0].GetArgs()[0].GetRange(); r.Low == nil || r.High != nil {
		t.Errorf("1.. = %+v", r)
	}
	capped, _, _ := model.FindDecl("Capped")
	if r := capped.GetNewtype().GetValueConstraints()[0].GetArgs()[0].GetRange(); r.Low != nil || r.High == nil {
		t.Errorf("..64 = %+v", r)
	}
}

// The standard constraint names are checked; others pass through.
func TestStandardConstraintChecking(t *testing.T) {
	tests := []struct{ src, want string }{
		{`type W: string where { unique(1) }`, "unique takes 0 arguments"},
		{`type W: string where { min(1, 2) }`, "min takes 1 argument"},
		{`type W: string where { matches("nope") }`, "matches does not take a string"},
		{`type W: string where { min("nope") }`, "min does not take a string"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			diags := lowerDiags(t, tt.src)
			if !strings.Contains(diags.Error(), tt.want) {
				t.Errorf("diagnostics = %v", diags)
			}
		})
	}
}

// A newtype's constraints accumulate its parent's, each with its origin.
func TestConstraintAccumulation(t *testing.T) {
	model := lower(t, `
type Email: string where { length(3..254) }
type WorkEmail: Email where { matches(/@acme/) }
type SeniorEmail: WorkEmail where { unique }
`)

	senior, _, _ := model.FindDecl("SeniorEmail")
	cs := senior.GetNewtype().GetValueConstraints()
	if len(cs) != 3 {
		t.Fatalf("SeniorEmail has %d constraints, want 3", len(cs))
	}

	origin := map[string]string{}
	for _, c := range cs {
		origin[c.GetName()] = c.GetFrom().GetName()
	}
	if origin["unique"] != "" {
		t.Errorf("a constraint written here records an origin: %q", origin["unique"])
	}
	if origin["matches"] != "WorkEmail" || origin["length"] != "Email" {
		t.Errorf("origins = %v", origin)
	}
}

// A name default must be a variant of the field's enum type.
func TestNameDefaults(t *testing.T) {
	model := lower(t, `
enum Status { Draft Placed }
type Holder { status: Status = Draft optional: Status? = Placed }
`)

	v, _, _ := model.FindDecl("Holder")
	for _, f := range v.Fields() {
		def := f.GetDefaultValue()
		if def.GetVariant() == nil {
			t.Errorf("%s default did not resolve: %+v", f.GetMeta().GetName(), def)
		}
	}
}

func TestBadNameDefaults(t *testing.T) {
	tests := []struct{ src, want string }{
		{"enum Status { Draft }\ntype Holder { s: Status = Missing }", "Status has no variant Missing"},
		{"type Other { x: string }\ntype Holder { s: Other = Draft }", "Other is not an enum"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			diags := lowerDiags(t, tt.src)
			if !strings.Contains(diags.Error(), tt.want) {
				t.Errorf("diagnostics = %v", diags)
			}
		})
	}
}

// A name in a constraint argument resolves to an enum variant.
func TestNameConstraintArgs(t *testing.T) {
	model := lower(t, `
enum Status { Draft Placed }
type Holder { status: Status where { oneOf(Draft, Placed) } }
`)

	v, _, _ := model.FindDecl("Holder")
	args := v.Fields()[0].GetConstraints()[0].GetArgs()
	if len(args) != 2 {
		t.Fatalf("oneOf has %d arguments, want 2", len(args))
	}
	for i, want := range []string{"Draft", "Placed"} {
		got := args[i].GetVariant()
		if got.GetName() != want || got.GetIndex() != int32(i) {
			t.Errorf("argument %d resolved to %+v, want variant %d %s", i, got, i, want)
		}
	}
}

func TestBadNameConstraintArgs(t *testing.T) {
	diags := lowerDiags(t, "enum Status { Draft }\ntype Holder { s: Status where { oneOf(Draft, Missing) } }")
	if !strings.Contains(diags.Error(), "Status has no variant Missing") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestLiteralDefaults(t *testing.T) {
	model := lower(t, `type Holder { n: int = 3 s: string = "x" b: bool = true xs: [string] = [] }`)

	v, _, _ := model.FindDecl("Holder")
	kinds := []ir.LiteralKind{
		ir.LiteralKind_LITERAL_KIND_INT,
		ir.LiteralKind_LITERAL_KIND_STRING,
		ir.LiteralKind_LITERAL_KIND_BOOL,
		ir.LiteralKind_LITERAL_KIND_LIST,
	}
	for i, want := range kinds {
		if got := v.Fields()[i].GetDefaultValue().GetKind(); got != want {
			t.Errorf("field %d default kind = %v, want %v", i, got, want)
		}
	}
}

func TestTargetDirectivesAttach(t *testing.T) {
	model := lower(t, `
type Money { amount: string }

type Order: Entity {
  id: string
  items: [string]
}

target go for p {
  out("./gen/go")

  Order {
    name("PurchaseOrder")
    items => slice
  }

  Money => foreign("x", "Money")
}
`)

	block := model.GetTargets()[0]
	if got := len(block.GetDirectives()); got != 1 || block.GetDirectives()[0].GetName() != "out" {
		t.Errorf("package-level directives = %+v", block.GetDirectives())
	}

	order, _, _ := model.FindDecl("Order")
	if got := len(order.GetDirectives()); got != 1 || order.GetDirectives()[0].GetName() != "name" {
		t.Errorf("Order directives = %+v", order.GetDirectives())
	}

	// A bare directive inside a nested block applies to the path it is under.
	var items *ir.Field
	for _, f := range order.Fields() {
		if f.GetMeta().GetName() == "items" {
			items = f
		}
	}
	if got := len(items.GetDirectives()); got != 1 || items.GetDirectives()[0].GetName() != "slice" {
		t.Errorf("items directives = %+v", items.GetDirectives())
	}

	money, _, _ := model.FindDecl("Money")
	if got := len(money.GetDirectives()); got != 1 {
		t.Errorf("Money directives = %+v", money.GetDirectives())
	}
}

// A path naming a class applies to everything satisfying it.
func TestClassPathExpands(t *testing.T) {
	model := lower(t, `
class Auditable { createdAt: string }
type A: Entity, Auditable { id: string createdAt: string }
type B: Entity, Auditable { id: string createdAt: string }
type C: Entity { id: string }

target sql for p {
  Auditable => trigger("touch")
}
`)

	for _, name := range []string{"A", "B"} {
		decl, _, _ := model.FindDecl(name)
		if len(decl.GetDirectives()) != 1 {
			t.Errorf("%s got %d directives, want 1", name, len(decl.GetDirectives()))
			continue
		}
		if decl.GetDirectives()[0].GetFromClass() == nil {
			t.Errorf("%s directive does not record the class it came from", name)
		}
	}

	c, _, _ := model.FindDecl("C")
	if len(c.GetDirectives()) != 0 {
		t.Errorf("C satisfies nothing but got %+v", c.GetDirectives())
	}
}

// A directive on a field beats one on its type, and a subclass beats a
// class it requires.
func TestSpecificityLadder(t *testing.T) {
	model := lower(t, `
class Base { x: string }
class Derived: Base { y: string }
type Ent: Entity, Derived { id: string x: string y: string }

target go for p {
  Base => rule("base")
  Derived => rule("derived")
}
`)

	e, _, _ := model.FindDecl("Ent")
	if got := len(e.GetDirectives()); got != 1 {
		t.Fatalf("got %d directives, want the closer class to win: %+v", got, e.GetDirectives())
	}
	if got := e.GetDirectives()[0].GetArgs()[0].GetText(); got != "derived" {
		t.Errorf("winner = %q, want derived", got)
	}
}

// A path reaches an enum's variant and a field of that variant.
func TestVariantDirectivesAttach(t *testing.T) {
	model := lower(t, `
enum Payment {
  Card { last4: string }
  Cash
}

target proto for p {
  Payment.Card => number(4)
  Payment.Card.last4 => number(2)
  Payment {
    Cash => number(1)
  }
}
`)

	payment, _, _ := model.FindDecl("Payment")
	variants := payment.GetEnumeration().GetVariants()
	for _, tt := range []struct {
		what string
		got  []*ir.Directive
		want string
	}{
		{"Card", variants[0].GetDirectives(), "4"},
		{"Card.last4", variants[0].GetFields()[0].GetDirectives(), "2"},
		{"Cash", variants[1].GetDirectives(), "1"},
	} {
		if len(tt.got) != 1 || tt.got[0].GetArgs()[0].GetText() != tt.want {
			t.Errorf("%s directives = %+v, want number(%s)", tt.what, tt.got, tt.want)
		}
	}
	if len(payment.GetDirectives()) != 0 {
		t.Errorf("a variant's directive landed on the enum: %+v", payment.GetDirectives())
	}
}

// Both entries survive, in source order; gen.CheckDirectives reports the
// tie.
func TestEqualSpecificityKeepsEveryCandidate(t *testing.T) {
	model := lower(t, `
class One { x: string }
class Two { y: string }
type Ent: Entity, One, Two { id: string x: string y: string }

target go for p {
  One => rule("a")
  Two => rule("b")
}
`)

	e, _, _ := model.FindDecl("Ent")
	got := e.GetDirectives()
	if len(got) != 2 {
		t.Fatalf("Ent directives = %+v, want rule(\"a\") then rule(\"b\")", got)
	}
	for i, want := range []string{"a", "b"} {
		if got[i].GetName() != "rule" || got[i].GetArgs()[0].GetText() != want {
			t.Errorf("directive %d = %+v, want rule(%q)", i, got[i], want)
		}
	}
}

func TestTargetPathNamesNothing(t *testing.T) {
	tests := []struct{ src, want string }{
		{"target go for p { Missing => rule }", "target path Missing names nothing"},
		{"type V2 { x: string }\ntarget go for p { V2.nope => rule }", "V2 has no field nope"},
		{"type V3 { x: string }\ntarget go for p { V3.x.y => rule }", "nothing is beneath a field"},
		{"enum Pay { A { x: string } B }\ntarget go for p { Pay.C => rule }", "Pay has no variant C"},
		{"enum Pay { A { x: string } B }\ntarget go for p { Pay.A.nope => rule }", "Pay.A has no field nope"},
		{"class Klass { x: string }\ntarget go for p { Klass.x.y => rule }", "a class path reaches a field and no further"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			diags := lowerDiags(t, tt.src)
			if !strings.Contains(diags.Error(), tt.want) {
				t.Errorf("diagnostics = %v", diags)
			}
		})
	}
}

func TestTargetForAnotherPackage(t *testing.T) {
	diags := lowerDiags(t, `target go for elsewhere { out("./gen") }`)
	if !strings.Contains(diags.Error(), "is for package elsewhere") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestDirectiveNameMayBeAKeyword(t *testing.T) {
	model := lower(t, `target go for p { package("github.com/acme/x") }`)

	if got := model.GetTargets()[0].GetDirectives()[0].GetName(); got != "package" {
		t.Errorf("directive = %q", got)
	}
}

// Top reaches Root in four steps through Long, written first, and in three
// through Short.
func TestHopsTakesTheShortestRequiresChain(t *testing.T) {
	model := lower(t, `
class Root { }
class Via: Root { }
class Deep: Via { }
class Short: Via { }
class Long: Deep { }
class Top: Long, Short { }`)

	_, top, ok := model.FindDecl("Top")
	if !ok {
		t.Fatal("no class Top")
	}
	_, root, ok := model.FindDecl("Root")
	if !ok {
		t.Fatal("no class Root")
	}

	l := &lowerer{model: model}
	if got := l.hops(top, root.GetIndex(), 0, map[int32]bool{}); got != 3 {
		t.Errorf("hops(Top, Root) = %d, want 3", got)
	}
}

// `kg?` is a type reference and does not intern the same entry as `kg`.
func TestModifiedUnitNameIsNotABareUnit(t *testing.T) {
	model := lower(t, `
primitive decimal
unit kg
type W {
  net: decimal<kg>
  maybe: decimal<kg?>
}`)

	decl, _, _ := model.FindDecl("W")
	fields := decl.GetStructure().GetFields()
	if got, other := fields[0].GetType().GetIndex(), fields[1].GetType().GetIndex(); got == other {
		t.Errorf("decimal<kg?> interned with decimal<kg> at types[%d]", got)
	}
}

// A directive on a path naming a `_`-merged declaration is carried on the
// extern entry.
func TestTargetPathNamesUnderscoreImport(t *testing.T) {
	file, err := parser.Parse("main.tdl", strings.NewReader(`
package shop

import "dep.tdl" as _

type Price { amount: Money }

target go for shop {
  Money => foreign("github.com/acme/money", "Money")
}
`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	model, diags := Lower(file, WithLoader(MapLoader{
		"dep.tdl": "package acme.money\ntype Money { units: int }\n",
	}))
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var money *ir.Extern
	for _, e := range model.GetExterns() {
		if e.GetPackage() == "acme.money" && e.GetName() == "Money" {
			money = e
		}
	}
	if money == nil {
		t.Fatalf("no extern for acme.money.Money in %+v", model.GetExterns())
	}

	ds := money.GetDirectives()
	if len(ds) != 1 {
		t.Fatalf("got %d directives on the extern, want 1: %+v", len(ds), ds)
	}
	d := ds[0]
	if d.GetName() != "foreign" || d.GetTarget() != "go" {
		t.Errorf("directive = %s for %s, want foreign for go", d.GetName(), d.GetTarget())
	}
	var args []string
	for _, a := range d.GetArgs() {
		args = append(args, a.GetText())
	}
	if got, want := strings.Join(args, ","), "github.com/acme/money,Money"; got != want {
		t.Errorf("args = %s, want %s", got, want)
	}
}
