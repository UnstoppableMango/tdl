package golang_test

import (
	"context"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/golang"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// A model is built by hand rather than parsed, so a failure here is the
// generator's and not lowering's.
type modelBuilder struct {
	model *ir.Model
}

func newModel(pkg string) *modelBuilder {
	m := &modelBuilder{model: &ir.Model{Package: pkg}}
	// The prelude arrives merged into the declaration table, so every
	// fixture carries the primitives a field can name.
	for _, name := range []string{"string", "int", "bool", "bytes", "decimal", "uuid", "instant", "date", "duration", "List", "Set", "Map"} {
		m.decl(&ir.Decl{
			Meta: &ir.Meta{Name: name, Position: &ir.Position{Filename: "/nix/store/x/std.tdl"}},
			Node: &ir.Decl_Primitive{Primitive: &ir.Primitive{}},
		})
	}
	m.decl(&ir.Decl{
		Meta: &ir.Meta{Name: "Option", Position: &ir.Position{Filename: "/nix/store/x/std.tdl"}},
		Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{
			Params:   []*ir.Param{{Name: "T"}},
			Variants: []*ir.Variant{{Meta: &ir.Meta{Name: "Some"}}, {Meta: &ir.Meta{Name: "None"}}},
		}},
	})
	return m
}

// decl appends a declaration. A type reference to it is made by name with
// [modelBuilder.named], so the index never has to be carried around.
func (m *modelBuilder) decl(d *ir.Decl) {
	m.model.Decls = append(m.model.Decls, d)
}

// named interns a type reference to a declaration, applied to arguments.
func (m *modelBuilder) named(declName string, args ...*ir.ID) *ir.ID {
	_, ctor, ok := m.model.FindDecl(declName)
	if !ok {
		panic("no declaration named " + declName)
	}
	t := &ir.Type{Ctor: ctor, Args: args, Wrote: ir.SyntacticForm_SYNTACTIC_FORM_NAMED}
	id := &ir.ID{Index: int32(len(m.model.GetTypes())), Name: declName}
	m.model.Types = append(m.model.Types, t)
	return id
}

// own appends a declaration attributed to the model's own file, which is
// how the backend tells it apart from the prelude.
func (m *modelBuilder) own(d *ir.Decl) {
	if d.GetMeta().GetPosition() == nil {
		d.GetMeta().Position = &ir.Position{Filename: "shop.tdl"}
	}
	m.decl(d)
}

func field(name string, typ *ir.ID) *ir.Field {
	return &ir.Field{Meta: &ir.Meta{Name: name}, Type: typ}
}

func generate(t *testing.T, m *modelBuilder) *plugin.Response {
	t.Helper()
	resp, err := golang.Backend{}.Generate(context.Background(), &plugin.Request{
		Target: golang.Name,
		Model:  m.model,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return resp
}

// files keys a response by path, and asserts every file is parseable Go.
//
// Parsing is the assertion that matters: a substring check can pass while
// the output is something go build refuses.
func files(t *testing.T, resp *plugin.Response) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, f := range resp.GetFiles() {
		src := string(f.GetContent())
		if _, err := parser.ParseFile(token.NewFileSet(), f.GetPath(), src, parser.AllErrors); err != nil {
			t.Errorf("%s is not parseable Go: %v\n%s", f.GetPath(), err, src)
		}
		out[f.GetPath()] = src
	}
	return out
}

// contains asserts on the output with runs of whitespace collapsed, since
// gofmt aligns a struct's field types into columns and the alignment moves
// whenever a sibling field's name changes length.
func contains(t *testing.T, src string, wants ...string) {
	t.Helper()
	flat := collapse(src)
	for _, want := range wants {
		if !strings.Contains(flat, collapse(want)) {
			t.Errorf("output missing %q:\n%s", want, src)
		}
	}
}

func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func TestDescribe(t *testing.T) {
	d := golang.Backend{}.Describe()
	// The name is what a target block writes, and the package it lives in
	// is called something else on purpose.
	if d.Name != "go" {
		t.Errorf("name = %q", d.Name)
	}
	if !d.Reuse {
		t.Error("a request is answered from the request alone, so reuse should be declared")
	}

	declared := map[string]bool{}
	for _, spec := range d.Directives {
		declared[spec.GetName()] = true
	}
	for _, want := range []string{"package", "name", "tag"} {
		if !declared[want] {
			t.Errorf("directive %q is acted on but not declared, so the compiler would warn about it", want)
		}
	}
}

func TestStructs(t *testing.T) {
	m := newModel("shop.billing")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "LineItem", Doc: []string{"One line on an order."}},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Kind: ir.StructKind_STRUCT_KIND_VALUE,
			Fields: []*ir.Field{
				field("sku", m.named("string")),
				field("quantity", m.named("int")),
			},
		}},
	})

	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Order"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Kind: ir.StructKind_STRUCT_KIND_ENTITY,
			Fields: []*ir.Field{
				{Meta: &ir.Meta{Name: "id"}, Type: m.named("string"), Key: true},
				field("placedAt", m.named("instant")),
				field("items", m.named("List", m.named("LineItem"))),
				field("tags", m.named("Set", m.named("string"))),
				field("totals", m.named("Map", m.named("string"), m.named("decimal"))),
				field("note", m.named("Option", m.named("string"))),
			},
		}},
	})

	got := files(t, generate(t, m))

	item, ok := got["line_item.go"]
	if !ok {
		t.Fatalf("no line_item.go in %v", keys(got))
	}
	contains(t, item,
		"// Code generated by tdl. DO NOT EDIT.",
		"package billing",
		"// One line on an order.",
		"type LineItem struct {",
		"Sku string",
		"Quantity int64",
	)

	order, ok := got["order.go"]
	if !ok {
		t.Fatalf("no order.go in %v", keys(got))
	}
	contains(t, order,
		`import "time"`,
		"Id string",
		"PlacedAt time.Time",
		"Items []LineItem",
		"Tags map[string]struct{}",
		"Totals map[string]string",
		"Note *string",
	)
}

// The three struct kinds mean different things and emit the same shape: Go
// has no way to say "identity that survives changes to its contents".
func TestMixinIsAStructToo(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Audit"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Kind:   ir.StructKind_STRUCT_KIND_MIXIN,
			Fields: []*ir.Field{field("createdAt", m.named("instant"))},
		}},
	})

	got := files(t, generate(t, m))
	contains(t, got["audit.go"], "type Audit struct {", "CreatedAt time.Time")
}

func TestFieldlessEnumIsConstants(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Status"},
		Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{
			Variants: []*ir.Variant{
				{Meta: &ir.Meta{Name: "Active"}},
				{Meta: &ir.Meta{Name: "Pending"}},
			},
		}},
	})

	got := files(t, generate(t, m))
	src := got["status.go"]
	contains(t, src,
		"type Status string",
		`StatusActive Status = "Active"`,
		`StatusPending Status = "Pending"`,
	)
	if strings.Contains(src, "interface") {
		t.Errorf("a fieldless enum should not be an interface:\n%s", src)
	}
}

func TestVariantWithFieldsIsASealedInterface(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Payment"},
		Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{
			Variants: []*ir.Variant{
				{Meta: &ir.Meta{Name: "Cash"}},
				{
					Meta:   &ir.Meta{Name: "Card"},
					Fields: []*ir.Field{field("last4", m.named("string"))},
				},
			},
		}},
	})

	got := files(t, generate(t, m))
	src := got["payment.go"]
	contains(t, src,
		"type Payment interface{ isPayment() }",
		"type PaymentCash struct {",
		"func (PaymentCash) isPayment() {}",
		"type PaymentCard struct {",
		"Last4 string",
		"func (PaymentCard) isPayment() {}",
	)
	// One variant carrying fields decides the shape for the whole enum,
	// which is the cost of the split and the reason it is written down.
	if strings.Contains(src, "const (") {
		t.Errorf("an enum with a field-carrying variant should not emit constants:\n%s", src)
	}
}

func TestNewtype(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Sku"},
		Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: m.named("string")}},
	})

	got := files(t, generate(t, m))
	contains(t, got["sku.go"], "type Sku string")
}

// An alias is transparent, so it is expanded at every use rather than
// declared.
func TestAliasIsExpanded(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Tags"},
		Node: &ir.Decl_Alias{Alias: &ir.Alias{Target: m.named("List", m.named("string"))}},
	})
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Post"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{field("tags", m.named("Tags"))},
		}},
	})

	got := files(t, generate(t, m))
	if _, ok := got["tags.go"]; ok {
		t.Error("an alias should not generate a declaration")
	}
	contains(t, got["post.go"], "Tags []string")
}

func TestDirectives(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta:       &ir.Meta{Name: "User"},
		Directives: []*ir.Directive{{Name: "name", Target: "go", Args: []*ir.Literal{text("Account")}}},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{
				{
					Meta: &ir.Meta{Name: "email"},
					Type: m.named("string"),
					Directives: []*ir.Directive{
						{Name: "tag", Target: "go", Args: []*ir.Literal{text(`json:"email_address"`)}},
						// A model carries directives for every target
						// block, so one that does not filter would read
						// another backend's.
						{Name: "tag", Target: "sql", Args: []*ir.Literal{text(`db:"nope"`)}},
					},
				},
			},
		}},
	})
	m.model.Targets = []*ir.TargetBlock{{
		Meta:       &ir.Meta{Name: "go"},
		Directives: []*ir.Directive{{Name: "package", Target: "go", Args: []*ir.Literal{text("github.com/acme/billing")}}},
	}}

	got := files(t, generate(t, m))
	src, ok := got["user.go"]
	if !ok {
		t.Fatalf("no user.go in %v", keys(got))
	}
	contains(t, src,
		"package billing",
		"type Account struct {",
		"`json:\"email_address\"`",
	)
	if strings.Contains(src, "db:") {
		t.Errorf("another target's directive leaked through:\n%s", src)
	}
}

// A backend says what it cannot handle in a diagnostic rather than by
// returning an error, and a warning does not stop the rest of the model
// from generating.
func TestUnsupportedIsAWarning(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Auditable", Position: &ir.Position{Filename: "shop.tdl", Line: 4}},
		Node: &ir.Decl_Class{Class: &ir.Class{}},
	})
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Note"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{field("body", m.named("string"))},
		}},
	})

	resp := generate(t, m)
	if len(resp.GetDiagnostics()) != 1 {
		t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
	}
	d := resp.GetDiagnostics()[0]
	if d.GetSeverity() != plugin.Severity_SEVERITY_WARNING {
		t.Errorf("severity = %v", d.GetSeverity())
	}
	if d.GetPosition().GetLine() != 4 {
		t.Errorf("position = %+v", d.GetPosition())
	}

	got := files(t, resp)
	if _, ok := got["note.go"]; !ok {
		t.Errorf("a warning suppressed the rest of the model: %v", keys(got))
	}
}

// The prelude is merged into the declaration table untagged, so a backend
// that emits per declaration has to tell it apart from the model's own.
func TestPreludeIsNotGenerated(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Note"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{field("body", m.named("string"))},
		}},
	})

	got := files(t, generate(t, m))
	if len(got) != 1 {
		t.Errorf("expected only the model's own declaration, got %v", keys(got))
	}
}

func text(s string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_STRING, Text: s}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
