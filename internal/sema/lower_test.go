package sema

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

func lower(t *testing.T, src string) *ir.Model {
	t.Helper()
	file, err := parser.Parse("test.tdl", strings.NewReader(src))
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
	file, err := parser.Parse("test.tdl", strings.NewReader(src))
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
package shop

primitive string
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

	if model.GetPackage() != "shop" {
		t.Errorf("package = %q", model.GetPackage())
	}
	if got := len(model.GetDecls()); got != 7 {
		t.Fatalf("got %d decls, want 7", got)
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
			target := model.GetDecls()[0].GetAlias().GetTarget()
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

	a := model.GetDecls()[0].GetAlias().GetTarget()
	b := model.GetDecls()[1].GetAlias().GetTarget()
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

	outer := model.Type(model.GetDecls()[0].GetAlias().GetTarget())
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

	a := model.GetDecls()[0].Fields()[0].GetType()
	b := model.GetDecls()[1].Fields()[0].GetType()
	if a.GetIndex() != b.GetIndex() {
		t.Errorf("the same type interned twice: %d and %d", a.GetIndex(), b.GetIndex())
	}

	// [string] is List<string>, so string is in the table too.
	if got := len(model.GetTypes()); got != 2 {
		t.Errorf("type table holds %d entries, want 2", got)
	}
}

// An unresolved name keeps its text and carries ir.Unresolved. Turning that
// into a diagnostic is phase 2.
func TestUnresolvedNameIsRecorded(t *testing.T) {
	model := lower(t, `value A { x: Missing }`)

	ctor := model.Type(model.GetDecls()[0].Fields()[0].GetType()).GetCtor()
	if ctor.Resolved() {
		t.Error("a name that matches nothing resolved")
	}
	if ctor.GetName() != "Missing" {
		t.Errorf("name = %q, want Missing", ctor.GetName())
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

	order := model.GetDecls()[0]
	m := order.GetMeta()
	if len(m.GetDoc()) != 1 || m.GetDoc()[0] != "An order someone placed." {
		t.Errorf("doc = %q", m.GetDoc())
	}
	if !m.IsDeprecated() || m.GetDeprecated().GetReason() != "use PurchaseOrder" {
		t.Errorf("deprecation = %+v", m.GetDeprecated())
	}
	// The position is the declaration keyword, not the doc comment or the
	// deprecation that precede it.
	if m.GetPosition().GetLine() != 4 {
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
	diags := lowerDiags(t, `value W { net: decimal<kg*m/s^2> }`)
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
