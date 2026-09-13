package golang_test

import (
	"context"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/golang"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
	"github.com/unstoppablemango/tdl/prelude"
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
			Meta: &ir.Meta{Name: name, Position: &ir.Position{Filename: prelude.Name}},
			Node: &ir.Decl_Primitive{Primitive: &ir.Primitive{}},
		})
	}
	m.decl(&ir.Decl{
		Meta: &ir.Meta{Name: "Option", Position: &ir.Position{Filename: prelude.Name}},
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

// files keys a response by path, and asserts the response is a Go package
// that compiles.
//
// Type checking rather than parsing is the assertion that matters. A
// substring check can pass while the output is something go build refuses,
// and so can parsing: `map[[]byte]struct{}` parses, and an undeclared type
// a skipped declaration left behind parses too.
//
// The whole response is checked as one package, since one file per
// declaration means a struct's field type is usually declared in another
// file.
func files(t *testing.T, resp *plugin.Response) map[string]string {
	t.Helper()
	out := map[string]string{}
	fset := token.NewFileSet()

	var parsed []*ast.File
	for _, f := range resp.GetFiles() {
		src := string(f.GetContent())
		out[f.GetPath()] = src

		file, err := parser.ParseFile(fset, f.GetPath(), src, parser.AllErrors)
		if err != nil {
			t.Errorf("%s is not parseable Go: %v\n%s", f.GetPath(), err, src)
			continue
		}
		parsed = append(parsed, file)
	}
	if len(parsed) != len(resp.GetFiles()) || len(parsed) == 0 {
		return out
	}

	// The source importer reads GOROOT rather than a build cache, so this
	// needs nothing installed and no module on disk. The generated package
	// imports only the standard library.
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	if _, err := conf.Check(parsed[0].Name.Name, fset, parsed, nil); err != nil {
		t.Errorf("the generated package does not type check: %v\n%s", err, strings.Join(sources(out), "\n"))
	}
	return out
}

// sources returns the response's files in a stable order, for a failure to
// print.
func sources(out map[string]string) []string {
	paths := keys(out)
	sort.Strings(paths)
	srcs := make([]string, 0, len(paths))
	for _, p := range paths {
		srcs = append(srcs, "==> "+p+" <==\n"+out[p])
	}
	return srcs
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
				{Meta: &ir.Meta{Name: "id"}, Type: m.named("string")},
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

// A newtype's `where` constraints are not enforced yet, and the backend
// says so rather than letting the model believe they are. The type itself
// is still emitted: skipping it would leave every field naming it referring
// to something the package does not declare.
func TestConstrainedNewtypeIsEmittedWithAWarning(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Email", Position: &ir.Position{Filename: "shop.tdl", Line: 7}},
		Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{
			Base:             m.named("string"),
			ValueConstraints: []*ir.Constraint{{Name: "matches"}},
		}},
	})
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Contact"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{field("email", m.named("Email"))},
		}},
	})

	resp := generate(t, m)
	if len(resp.GetDiagnostics()) != 1 {
		t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
	}
	if line := resp.GetDiagnostics()[0].GetPosition().GetLine(); line != 7 {
		t.Errorf("position line = %d", line)
	}

	got := files(t, resp)
	contains(t, got["email.go"], "type Email string")
	contains(t, got["contact.go"], "Email Email")
}

// Go requires a map key to be comparable, so a Set or a Map that would
// become one it refuses is reported rather than emitted.
func TestIncomparableKeysAreUnsupported(t *testing.T) {
	for _, tt := range []struct {
		name string
		typ  func(m *modelBuilder) *ir.ID
	}{
		{"set of bytes", func(m *modelBuilder) *ir.ID { return m.named("Set", m.named("bytes")) }},
		{"set of lists", func(m *modelBuilder) *ir.ID {
			return m.named("Set", m.named("List", m.named("string")))
		}},
		{"map keyed by a list", func(m *modelBuilder) *ir.ID {
			return m.named("Map", m.named("List", m.named("string")), m.named("string"))
		}},
		{"map keyed by a struct holding a list", func(m *modelBuilder) *ir.ID {
			return m.named("Map", m.named("Path"), m.named("string"))
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel("shop")
			m.own(&ir.Decl{
				Meta: &ir.Meta{Name: "Path"},
				Node: &ir.Decl_Structure{Structure: &ir.Struct{
					Fields: []*ir.Field{field("segments", m.named("List", m.named("string")))},
				}},
			})
			m.own(&ir.Decl{
				Meta: &ir.Meta{Name: "Index", Position: &ir.Position{Filename: "shop.tdl", Line: 9}},
				Node: &ir.Decl_Structure{Structure: &ir.Struct{
					Fields: []*ir.Field{field("by", tt.typ(m))},
				}},
			})

			resp := generate(t, m)
			if len(resp.GetDiagnostics()) != 1 {
				t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
			}
			got := files(t, resp)
			if _, ok := got["index.go"]; ok {
				t.Errorf("a declaration with an uncompilable field was emitted:\n%s", got["index.go"])
			}
		})
	}
}

// Reaching the same declaration twice on separate paths is not a cycle, so
// the walk that stops one has to unwind as it returns.
func TestARepeatedFieldTypeIsNotACycle(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Coord"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{field("value", m.named("int"))},
		}},
	})
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Point"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{
				field("x", m.named("Coord")),
				field("y", m.named("Coord")),
			},
		}},
	})
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Grid"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{field("seen", m.named("Set", m.named("Point")))},
		}},
	})

	resp := generate(t, m)
	if len(resp.GetDiagnostics()) != 0 {
		t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
	}
	contains(t, files(t, resp)["grid.go"], "Seen map[Point]struct{}")
}

// A comparable element is still a map, since that is the only Go shape that
// keeps what a Set promises.
func TestComparableSetsStillGenerate(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Sku"},
		Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: m.named("string")}},
	})
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Basket"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{
				field("skus", m.named("Set", m.named("Sku"))),
				// A pointer is comparable whatever it points at, so an
				// optional element is a legal key even when its element
				// would not be.
				field("maybe", m.named("Set", m.named("Option", m.named("bytes")))),
			},
		}},
	})

	resp := generate(t, m)
	if len(resp.GetDiagnostics()) != 0 {
		t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
	}
	contains(t, files(t, resp)["basket.go"],
		"Skus map[Sku]struct{}",
		"Maybe map[*[]byte]struct{}",
	)
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

// A package clause is one identifier that every file carries, so a name Go
// will not accept is an error rather than a warning: nothing is generated
// and the host writes nothing.
func TestAKeywordPackageNameIsAnError(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Note"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{field("body", m.named("string"))},
		}},
	})
	m.model.Targets = []*ir.TargetBlock{{
		Meta: &ir.Meta{Name: "go"},
		Directives: []*ir.Directive{{
			Name:     "package",
			Target:   "go",
			Args:     []*ir.Literal{text("github.com/acme/type")},
			Position: &ir.Position{Filename: "shop.tdl", Line: 2},
		}},
	}}

	resp := generate(t, m)
	if len(resp.GetFiles()) != 0 {
		t.Errorf("a package clause Go refuses still generated files: %v", keys(files(t, resp)))
	}
	if len(resp.GetDiagnostics()) != 1 {
		t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
	}
	d := resp.GetDiagnostics()[0]
	if d.GetSeverity() != plugin.Severity_SEVERITY_ERROR {
		t.Errorf("severity = %v, and the host only stops on an error", d.GetSeverity())
	}
	if d.GetPosition().GetLine() != 2 {
		t.Errorf("position = %+v, and the directive is what to point at", d.GetPosition())
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

// The prelude is recognized by its whole name and not by how the name ends,
// so a user's file that happens to end the same way is still theirs.
func TestAFileEndingInStdIsNotThePrelude(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Note", Position: &ir.Position{Filename: "my-std.tdl"}},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{field("body", m.named("string"))},
		}},
	})

	got := files(t, generate(t, m))
	if _, ok := got["note.go"]; !ok {
		t.Errorf("a user file was mistaken for the prelude: %v", keys(got))
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
