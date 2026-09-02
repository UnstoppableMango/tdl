package sema

import (
	"fmt"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

// preamble declares the placeholder names these tests use as stand-in
// types. The prelude supplies everything else: `string`, `List`, `Option`,
// and the rest are loaded, not declared here.
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

entity Order {
  key id: string
  items: [string] owned
}

value Money { amount: string }

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

// A base unit measures itself, which is the dimension every derived unit
// reduces to.
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

// File order is not resolution order: a unit may derive from one written
// after it.
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

// A unit that reaches itself has no reduction, so it is an error at the
// declaration that closes the cycle.
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

// Without a loader, an import is a diagnostic rather than a filesystem
// read nobody asked for.
func TestImportNeedsALoader(t *testing.T) {
	diags := lowerDiags(t, `import "common.tdl" as common`)
	if !strings.Contains(diags.Error(), "imports need a loader") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestUndefinedImportAlias(t *testing.T) {
	diags := lowerDiags(t, `value Holder { a: common.Address }`)
	if !strings.Contains(diags.Error(), "undefined import alias: common") {
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
	model := lower(t, `
class Auditable { createdAt: string }
instance Auditable for A
instance Auditable for B
value A { x: string }
value B { x: string }
`)

	if got := len(model.GetInstances()); got != 2 {
		t.Errorf("got %d instances, want 2", got)
	}
}

// A target block names a backend, not a type, so it does not collide with
// a declaration of the same name.
func TestTargetsAreNotTypeNames(t *testing.T) {
	model := lower(t, `
value go { x: string }
target go for p { out("./gen") }
`)

	if _, _, ok := model.FindDecl("go"); !ok {
		t.Error("the value named go is missing")
	}
	if got := len(model.GetTargets()); got != 1 {
		t.Errorf("got %d target blocks, want 1", got)
	}
}

// A type ID's name is what a person reads; the interning key is what
// separates entries. `[T]` and `List<T>` are two entries with one name.
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

// Interning keys use argument indices, so two types that differ only in an
// argument's written form stay apart.
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

// The prelude is replaceable: `[T]` means whatever the loaded prelude says
// `List` is, and nothing in lowering knows more than the spelling.
func TestPreludeIsReplaceable(t *testing.T) {
	file, err := parser.Parse("test.tdl", strings.NewReader(`value V { items: [string] }`))
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

// A file may declare a name the prelude already has, and its own wins.
func TestFileShadowsPrelude(t *testing.T) {
	model := lower(t, `
primitive string
value Shadowed { s: string }
`)

	v, _, _ := model.FindDecl("Shadowed")
	ctor := model.Decl(model.Type(v.Fields()[0].GetType()).GetCtor())
	if got := ctor.GetMeta().GetPosition().GetFilename(); got != "test.tdl" {
		t.Errorf("string resolved to %q, want the file's own", got)
	}
}

// Compiling a prelude means having no prelude, and then nothing resolves
// that the file does not declare.
func TestWithoutPrelude(t *testing.T) {
	file, err := parser.Parse("test.tdl", strings.NewReader(`value V { s: string }`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	_, diags := Lower(file, WithoutPrelude())
	if !strings.Contains(diags.Error(), "undefined: string") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// A qualified reference carries the dependency's package, and the
// declaration is not inlined.
func TestCrossPackageReference(t *testing.T) {
	file, err := parser.Parse("main.tdl", strings.NewReader(`
package shop

import "common.tdl" as common

value Order { ship: common.Address }
`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	model, diags := Lower(file, WithLoader(MapLoader{
		"common.tdl": "package shop.common\nvalue Address { line1: string }\n",
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

	// Not inlined: the dependency's declaration is not in this table.
	if _, _, ok := model.FindDecl("Address"); ok {
		t.Error("the dependency's declaration was inlined")
	}
}

// A `_` import merges the dependency's exported names, so it has to know
// what they are. Package-private names are not merged.
func TestUnderscoreImportMerges(t *testing.T) {
	file, err := parser.Parse("main.tdl", strings.NewReader(`
import "common.tdl" as _

value Order { ship: Address }
`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	model, diags := Lower(file, WithLoader(MapLoader{
		"common.tdl": "package shop.common\nvalue Address { line1: string }\nvalue internal { x: string }\n",
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

// Two occurrences of one foreign declaration share an extern entry.
func TestExternsAreInterned(t *testing.T) {
	file, err := parser.Parse("main.tdl", strings.NewReader(`
import "common.tdl" as common

value A { x: common.Address }
value B { y: common.Address }
`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	model, _ := Lower(file, WithLoader(MapLoader{
		"common.tdl": "package shop.common\nvalue Address { line1: string }\n",
	}))
	if got := len(model.GetExterns()); got != 1 {
		t.Errorf("got %d externs, want 1", got)
	}
}

func TestClassLowering(t *testing.T) {
	model := lower(t, `
class Timestamped { createdAt: string }

class Auditable: Timestamped {
  key
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
	if !c.GetRequiresKey() {
		t.Error("the key requirement was lost")
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

// `instance C for T` is sugar for `instance C<T>`, so lowering normalizes
// it and there is one form from here on.
func TestInstanceFormsNormalize(t *testing.T) {
	model := lower(t, `
class Auditable { createdAt: string }
value A { x: string }

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

// Satisfaction comes from declared conformance and ground instances, and
// closes over the classes a class requires.
func TestSatisfaction(t *testing.T) {
	model := lower(t, `
class Timestamped { createdAt: string }
class Auditable: Timestamped { updatedAt: string }

entity Declared: Auditable { key id: string }
value ByInstance { x: string }
value Neither { x: string }

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

// Including a mixin does not confer conformance: the spec says the two are
// independent and conformance is nominal.
func TestIncludeDoesNotConfer(t *testing.T) {
	model := lower(t, `
class Auditable { createdAt: string }
mixin Stamps: Auditable { createdAt: string }
value Uses { include Stamps }
`)

	_, auditable, _ := model.FindDecl("Auditable")
	if names := satisfyingNames(model, auditable); contains(names, "Uses") {
		t.Errorf("including a mixin conferred conformance: %v", names)
	}
}

// A conditional instance stands for a family of types rather than one, so
// it is recorded but does not name a satisfying declaration.
func TestConditionalInstanceNotIndexed(t *testing.T) {
	model := lower(t, `
class Auditable { createdAt: string }
value Page<T> { items: [T] }

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
value Uses { include Outer c: string }
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

func TestIncludeOfNonMixin(t *testing.T) {
	diags := lowerDiags(t, `
value NotAMixin { x: string }
value Uses { include NotAMixin }
`)
	if !strings.Contains(diags.Error(), "is not a mixin") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// A `requires` clause says nothing checkable until a type is instantiated,
// so the diagnostic points at the use site.
func TestRequiresCheckedAtUse(t *testing.T) {
	diags := lowerDiags(t, `
class Auditable { createdAt: string }
value Envelope<P> requires Auditable<P> { body: P }
value Plain { x: string }
value Holder { e: Envelope<Plain> }
`)
	if !strings.Contains(diags.Error(), "Plain does not satisfy Auditable") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestRequiresSatisfied(t *testing.T) {
	lower(t, `
class Auditable { createdAt: string }
value Envelope<P> requires Auditable<P> { body: P }
value Audited: Auditable { createdAt: string }
value Holder { e: Envelope<Audited> }
`)
}

// An argument that is still a parameter is not checked here: whether it
// satisfies anything is the outer instantiation's business.
func TestRequiresDefersToOuterInstantiation(t *testing.T) {
	lower(t, `
class Auditable { createdAt: string }
value Envelope<P> requires Auditable<P> { body: P }
value Outer<P> requires Auditable<P> { e: Envelope<P> }
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

// A conditional instance makes an instantiated type satisfy a class, which
// a declaration cannot stand in for: `Page` satisfies nothing on its own.
func TestConditionalInstanceSearch(t *testing.T) {
	model := lower(t, `
class Auditable { createdAt: string }
value Page<P> { items: [P] }
value Audited: Auditable { createdAt: string }
value Plain { x: string }

instance <P> Auditable<Page<P>> requires Auditable<P>

value Uses {
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

// The search recurses: a page of pages of auditable things is auditable.
func TestConditionalInstanceNests(t *testing.T) {
	model := lower(t, `
class Auditable { createdAt: string }
value Page<P> { items: [P] }
value Audited: Auditable { createdAt: string }

instance <P> Auditable<Page<P>> requires Auditable<P>

value Uses { nested: Page<Page<Audited>> }
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

// A `requires` clause is discharged through the search, so an instantiated
// argument satisfies it.
func TestRequiresThroughConditionalInstance(t *testing.T) {
	lower(t, `
class Auditable { createdAt: string }
value Page<P> { items: [P] }
value Audited: Auditable { createdAt: string }
value Envelope<P> requires Auditable<P> { body: P }

instance <P> Auditable<Page<P>> requires Auditable<P>

value Holder { e: Envelope<Page<Audited>> }
`)
}

// The spec's two rules are checked where the instance is written, not
// where it is used.
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
value Pair<a, b> { x: a y: b }
instance <P> Rel<Pair<P, P>>
`)
	if !strings.Contains(diags.Error(), "repeats the parameter") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestInstanceConditionMustBeSmaller(t *testing.T) {
	diags := lowerDiags(t, `
class Auditable { createdAt: string }
value Page<P> { items: [P] }

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

value Holder {
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

// The standard names are checked; everything else is passed through.
func TestStandardConstraintChecking(t *testing.T) {
	tests := []struct{ src, want string }{
		{`type T: string where { unique(1) }`, "unique takes 0 arguments"},
		{`type T: string where { min(1, 2) }`, "min takes 1 argument"},
		{`type T: string where { matches("nope") }`, "matches does not take a string"},
		{`type T: string where { min("nope") }`, "min does not take a string"},
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

// A newtype narrows its parent and never replaces it, so the accumulated
// set is what a backend reads and the origin is what explains it.
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

// A name default denotes an enum variant, and which enum is the field's
// type, so nothing before now could check it.
func TestNameDefaults(t *testing.T) {
	model := lower(t, `
enum Status { Draft Placed }
value Holder { status: Status = Draft optional: Status? = Placed }
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
		{"enum Status { Draft }\nvalue Holder { s: Status = Missing }", "Status has no variant Missing"},
		{"value Other { x: string }\nvalue Holder { s: Other = Draft }", "Other is not an enum"},
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

func TestLiteralDefaults(t *testing.T) {
	model := lower(t, `value Holder { n: int = 3 s: string = "x" b: bool = true xs: [string] = [] }`)

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
value Money { amount: string }

entity Order {
  key id: string
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

// A path naming a class applies to everything satisfying it, which is what
// lets a rule be written once rather than repeated per type.
func TestClassPathExpands(t *testing.T) {
	model := lower(t, `
class Auditable { createdAt: string }
entity A: Auditable { key id: string createdAt: string }
entity B: Auditable { key id: string createdAt: string }
entity C { key id: string }

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

// The ladder: a directive on a field beats one on its type, and a subclass
// beats a class it requires.
func TestSpecificityLadder(t *testing.T) {
	model := lower(t, `
class Base { x: string }
class Derived: Base { y: string }
entity Ent: Derived { key id: string x: string y: string }

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

func TestEqualSpecificityIsAnError(t *testing.T) {
	diags := lowerDiags(t, `
class One { x: string }
class Two { y: string }
entity Ent: One, Two { key id: string x: string y: string }

target go for p {
  One => rule("a")
  Two => rule("b")
}
`)
	if !strings.Contains(diags.Error(), "same specificity") {
		t.Errorf("diagnostics = %v", diags)
	}
}

func TestTargetPathNamesNothing(t *testing.T) {
	tests := []struct{ src, want string }{
		{"target go for p { Missing => rule }", "target path Missing names nothing"},
		{"value V2 { x: string }\ntarget go for p { V2.nope => rule }", "V2 has no field nope"},
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

// A directive name may be a reserved word: the namespace belongs to the
// backend.
func TestDirectiveNameMayBeAKeyword(t *testing.T) {
	model := lower(t, `target go for p { package("github.com/acme/x") }`)

	if got := model.GetTargets()[0].GetDirectives()[0].GetName(); got != "package" {
		t.Errorf("directive = %q", got)
	}
}
