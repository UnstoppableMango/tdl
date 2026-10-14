package analysis_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

const source = `package shop

type Email: string
type WorkEmail: Email
alias Tags = {string}

type Order {
  id: string
  userID: int
  tags: Tags
  byName: {string -> Email}
  note: Email?
  shipOn: WorkEmail | null
  pay: Payment
}

enum Payment {
  Card { last4: string }
  Cash
}

enum Wire {
  Wired { at: string }
}

enum Status {
  Draft
}

class Named {
  name: string
}

target go for shop {
  discriminant("type")
  Wire => discriminant("via")
}
`

func analyze(t *testing.T, target string) (*ir.Model, *plugin.Analysis) {
	t.Helper()
	file, err := parser.Parse("tdl", strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatal(diags)
	}
	return model, analysis.Analyze(model, target)
}

// field returns the view of a field's type.
func field(t *testing.T, m *ir.Model, a *plugin.Analysis, decl, name string) *plugin.TypeView {
	t.Helper()
	d, _, ok := m.FindDecl(decl)
	if !ok {
		t.Fatalf("no declaration %s", decl)
	}
	for _, f := range d.Fields() {
		if f.GetMeta().GetName() == name {
			return a.GetTypes()[f.GetType().GetIndex()]
		}
	}
	t.Fatalf("no field %s.%s", decl, name)
	return nil
}

func declName(m *ir.Model, id *ir.ID) string {
	return m.Decl(id).GetMeta().GetName()
}

func TestTypeForms(t *testing.T) {
	m, a := analyze(t, "go")
	typ := func(id *ir.ID) *plugin.TypeView { return a.GetTypes()[id.GetIndex()] }

	if v := field(t, m, a, "Order", "id"); v.GetForm() != plugin.Form_FORM_PRIMITIVE || v.GetName() != "string" {
		t.Errorf("id = %v, want the primitive string", v)
	}

	// An alias is described as what it expands to.
	tags := field(t, m, a, "Order", "tags")
	if tags.GetForm() != plugin.Form_FORM_SET || typ(tags.GetElem()).GetName() != "string" {
		t.Errorf("tags = %v, want a set of string", tags)
	}
	if m.Type(tags.GetId()).GetCtor().GetName() == "Tags" {
		t.Errorf("tags.id = %v, want the entry Tags expands to", tags.GetId())
	}

	byName := field(t, m, a, "Order", "byName")
	if byName.GetForm() != plugin.Form_FORM_MAP ||
		typ(byName.GetKey()).GetName() != "string" ||
		declName(m, typ(byName.GetElem()).GetDecl()) != "Email" {
		t.Errorf("byName = %v, want a map of string to Email", byName)
	}

	if v := field(t, m, a, "Order", "note"); v.GetForm() != plugin.Form_FORM_OPTION {
		t.Errorf("note = %v, want an option", v)
	}
	if v := field(t, m, a, "Order", "shipOn"); v.GetForm() != plugin.Form_FORM_NULLABLE {
		t.Errorf("shipOn = %v, want a nullable", v)
	}
	if v := field(t, m, a, "Order", "pay"); v.GetForm() != plugin.Form_FORM_NAMED || declName(m, v.GetDecl()) != "Payment" {
		t.Errorf("pay = %v, want the declaration Payment", v)
	}
}

func TestTypeRefusals(t *testing.T) {
	m, a := analyze(t, "go")
	reasons := map[plugin.Refusal_Reason][]string{}
	for i, v := range a.GetTypes() {
		if r := v.GetRefusal(); r != nil {
			reasons[r.GetReason()] = append(reasons[r.GetReason()], r.GetSubject())
			if r.GetPosition() != m.GetTypes()[i].GetPosition() && r.GetReason() != plugin.Refusal_REASON_UNRESOLVED {
				t.Errorf("type %d: refusal at %v, want the type's position", i, r.GetPosition())
			}
		}
	}
	// The prelude's Option and Nullable are written over a parameter.
	if !slices.Contains(reasons[plugin.Refusal_REASON_PARAM], "T") {
		t.Errorf("refusals = %v, want the prelude's parameter T refused", reasons)
	}
}

func TestDeclViews(t *testing.T) {
	m, a := analyze(t, "go")
	view := func(name string) *plugin.DeclView {
		_, id, ok := m.FindDecl(name)
		if !ok {
			t.Fatalf("no declaration %s", name)
		}
		return a.GetDecls()[id.GetIndex()]
	}

	order := view("Order")
	if !order.GetOwn() {
		t.Error("Order is not the model's own")
	}
	if view("string").GetOwn() {
		t.Error("the prelude's string is the model's own")
	}

	var refs []string
	for _, id := range order.GetReferences() {
		refs = append(refs, declName(m, id))
	}
	want := []string{"string", "int", "Set", "Email", "Map", "Option", "WorkEmail", "Nullable", "Payment"}
	if !slices.Equal(refs, want) {
		t.Errorf("Order references %v, want %v", refs, want)
	}

	var words [][]string
	for _, f := range order.GetFields() {
		words = append(words, f.GetWords())
	}
	if !slices.Equal(words[1], []string{"user", "ID"}) || len(words) != 7 {
		t.Errorf("Order's field words = %v", words)
	}

	pay := view("Payment")
	if !pay.GetFielded() || pay.GetDiscriminant() != "type" {
		t.Errorf("Payment fielded %v, discriminant %q; want fielded on the block's \"type\"", pay.GetFielded(), pay.GetDiscriminant())
	}
	if got := pay.GetVariants()[0].GetFields()[0].GetWords(); !slices.Equal(got, []string{"last4"}) {
		t.Errorf("Card.last4 words = %v", got)
	}
	if got := view("Wire").GetDiscriminant(); got != "via" {
		t.Errorf("Wire discriminant = %q, want its own \"via\"", got)
	}
	if view("Status").GetFielded() {
		t.Error("Status is fielded")
	}

	under := view("WorkEmail").GetUnderlying()
	if v := a.GetTypes()[under.GetIndex()]; v.GetForm() != plugin.Form_FORM_PRIMITIVE || v.GetName() != "string" {
		t.Errorf("WorkEmail's underlying type = %v, want string", v)
	}
}

func TestDiscriminantIsPerTarget(t *testing.T) {
	m, a := analyze(t, "typescript")
	_, id, _ := m.FindDecl("Wire")
	if got := a.GetDecls()[id.GetIndex()].GetDiscriminant(); got != analysis.DefaultDiscriminant {
		t.Errorf("Wire discriminant for typescript = %q, want the default", got)
	}
}

func TestWords(t *testing.T) {
	for in, want := range map[string][]string{
		"userID":     {"user", "ID"},
		"HTTPServer": {"HTTP", "Server"},
		"last4":      {"last4"},
		"snake_case": {"snake", "case"},
		"":           nil,
	} {
		if got := analysis.Words(in); !slices.Equal(got, want) {
			t.Errorf("Words(%q) = %q, want %q", in, got, want)
		}
	}
}
