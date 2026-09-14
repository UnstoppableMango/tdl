package golang_test

import (
	"context"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
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
	m.decl(&ir.Decl{
		Meta: &ir.Meta{Name: "Entity", Position: &ir.Position{Filename: prelude.Name}},
		Node: &ir.Decl_Class{Class: &ir.Class{}},
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
	for _, want := range []string{"package", "name", "tag", "key"} {
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

// A skipped declaration is one the package does not declare, so whatever
// names it is skipped too, however far away, rather than emitted naming an
// undeclared type.
func TestReferringToASkippedDeclarationSkipsItToo(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Index", Position: &ir.Position{Filename: "shop.tdl", Line: 3}},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{field("by", m.named("Set", m.named("bytes")))},
		}},
	})
	// Declared before what it names, so a single pass in declaration order
	// would have rendered it before learning its field is skipped. Its field
	// is attached once Top exists, since a type reference is made by name.
	outer := &ir.Struct{}
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Outer", Position: &ir.Position{Filename: "shop.tdl", Line: 5}},
		Node: &ir.Decl_Structure{Structure: outer},
	})
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Top", Position: &ir.Position{Filename: "shop.tdl", Line: 7}},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{field("index", m.named("Index"))},
		}},
	})
	outer.Fields = []*ir.Field{field("top", m.named("Option", m.named("Top")))}
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Note"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Fields: []*ir.Field{field("body", m.named("string"))},
		}},
	})

	resp := generate(t, m)
	if len(resp.GetDiagnostics()) != 3 {
		t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
	}
	for _, d := range resp.GetDiagnostics() {
		if d.GetSeverity() != plugin.Severity_SEVERITY_WARNING {
			t.Errorf("severity = %v", d.GetSeverity())
		}
	}

	got := files(t, resp)
	if want := []string{"note.go"}; !slices.Equal(keys(got), want) {
		t.Errorf("files = %v, want %v", keys(got), want)
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
		Meta: &ir.Meta{Name: "kg", Position: &ir.Position{Filename: "shop.tdl", Line: 4}},
		Node: &ir.Decl_Unit{Unit: &ir.UnitDef{Base: true}},
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

// An entity's key is a target directive naming fields, and it becomes a key
// type and a method returning it, so a consumer can index entities without
// reading the .tdl file.
func TestEntityKey(t *testing.T) {
	m := newModel("shop")
	m.own(keyed("LineItem", []*ir.Field{
		field("order", m.named("string")),
		field("sku", m.named("string")),
		field("quantity", m.named("int")),
	}, name("order"), name("sku")))

	resp := generate(t, m)
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	got := files(t, resp)
	contains(t, got["line_item.go"],
		"type LineItemKey struct { Order string Sku string }",
		"func (l LineItem) Key() LineItemKey {",
		"return LineItemKey{Order: l.Order, Sku: l.Sku}",
	)
}

// A key of one field is that field's type, with no struct around it.
func TestSingleFieldKey(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "UserID"},
		Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: m.named("string")}},
	})
	m.own(keyed("User", []*ir.Field{
		field("id", m.named("UserID")),
		field("email", m.named("string")),
	}, name("id")))

	got := files(t, generate(t, m))
	src := got["user.go"]
	contains(t, src, "func (u User) Key() UserID {", "return u.Id")
	if strings.Contains(src, "UserKey") {
		t.Errorf("a single-field key generated a key type:\n%s", src)
	}
}

// A key the backend cannot generate is a warning at the directive, and the
// entity is still emitted: skipping it would leave every field naming it
// referring to an undeclared type.
func TestKeyIsAWarningWhenItCannotBeGenerated(t *testing.T) {
	cases := []struct {
		name  string
		build func(m *modelBuilder)
		file  string
	}{
		{"on a value", func(m *modelBuilder) {
			d := keyed("Money", []*ir.Field{field("amount", m.named("int"))}, name("amount"))
			d.GetStructure().Kind = ir.StructKind_STRUCT_KIND_VALUE
			m.own(d)
		}, "money.go"},
		{"an argument that is not a name", func(m *modelBuilder) {
			m.own(keyed("User", []*ir.Field{field("id", m.named("string"))}, text("id")))
		}, "user.go"},
		{"a name that is no field", func(m *modelBuilder) {
			m.own(keyed("User", []*ir.Field{field("id", m.named("string"))}, name("nope")))
		}, "user.go"},
		{"a repeated field", func(m *modelBuilder) {
			m.own(keyed("User", []*ir.Field{field("id", m.named("string"))}, name("id"), name("id")))
		}, "user.go"},
		{"an incomparable field", func(m *modelBuilder) {
			m.own(keyed("Blob", []*ir.Field{field("hash", m.named("bytes"))}, name("hash")))
		}, "blob.go"},
		{"a field the method would collide with", func(m *modelBuilder) {
			m.own(keyed("Secret", []*ir.Field{
				field("id", m.named("string")),
				field("key", m.named("string")),
			}, name("id")))
		}, "secret.go"},
		{"a declaration the key type would collide with", func(m *modelBuilder) {
			m.own(&ir.Decl{
				Meta: &ir.Meta{Name: "LineItemKey"},
				Node: &ir.Decl_Structure{Structure: &ir.Struct{
					Fields: []*ir.Field{field("raw", m.named("string"))},
				}},
			})
			m.own(keyed("LineItem", []*ir.Field{
				field("order", m.named("string")),
				field("sku", m.named("string")),
			}, name("order"), name("sku")))
		}, "line_item.go"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := newModel("shop")
			c.build(m)

			resp := generate(t, m)
			if len(resp.GetDiagnostics()) != 1 {
				t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
			}
			d := resp.GetDiagnostics()[0]
			if d.GetSeverity() != plugin.Severity_SEVERITY_WARNING {
				t.Errorf("severity = %v", d.GetSeverity())
			}
			if d.GetPosition().GetLine() != keyLine {
				t.Errorf("position = %+v, and the directive is what to point at", d.GetPosition())
			}

			got := files(t, resp)
			src, ok := got[c.file]
			if !ok {
				t.Fatalf("the entity was skipped along with its key: %v", keys(got))
			}
			if strings.Contains(src, "Key()") {
				t.Errorf("a key that warned was generated anyway:\n%s", src)
			}
		})
	}
}

// A model carries directives for every target block, and another backend's
// key is not this one's.
func TestKeyFromAnotherTargetIsIgnored(t *testing.T) {
	m := newModel("shop")
	d := keyed("User", []*ir.Field{field("id", m.named("string"))}, name("id"))
	d.Directives[0].Target = "sql"
	m.own(d)

	resp := generate(t, m)
	if len(resp.GetDiagnostics()) != 0 {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	if src := files(t, resp)["user.go"]; strings.Contains(src, "Key()") {
		t.Errorf("another target's key was generated:\n%s", src)
	}
}

// The receiver is derived from the Go name, and `name("")` passes the
// compiler's checks, so an empty name has to be a warning rather than an
// index out of range.
func TestKeyOnAnEmptyNameIsAWarning(t *testing.T) {
	m := newModel("shop")
	d := keyed("User", []*ir.Field{field("id", m.named("string"))}, name("id"))
	d.Directives = append(d.Directives, &ir.Directive{Name: "name", Target: "go", Args: []*ir.Literal{text("")}})
	m.own(d)

	resp := generate(t, m)
	for _, diag := range resp.GetDiagnostics() {
		if diag.GetSeverity() == plugin.Severity_SEVERITY_WARNING && diag.GetPosition().GetLine() == keyLine {
			return
		}
	}
	t.Errorf("no warning at the key directive: %+v", resp.GetDiagnostics())
}

// keyLine is where [keyed] says its directive was written.
const keyLine = 9

// keyed is an entity carrying a key directive for the go target.
func keyed(declName string, fields []*ir.Field, args ...*ir.Literal) *ir.Decl {
	return &ir.Decl{
		Meta: &ir.Meta{Name: declName},
		Directives: []*ir.Directive{{
			Name:     "key",
			Target:   "go",
			Args:     args,
			Position: &ir.Position{Filename: "shop.tdl", Line: keyLine},
		}},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Kind:   ir.StructKind_STRUCT_KIND_ENTITY,
			Fields: fields,
		}},
	}
}

func text(s string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_STRING, Text: s}
}

// name is a bare identifier, which is how a key directive names a field.
func name(s string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_NAME, Text: s}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// param interns a reference to a type parameter, by its position in the
// declaring node's parameter list.
func (m *modelBuilder) param(paramName string, index int32, args ...*ir.ID) *ir.ID {
	t := &ir.Type{Param: &ir.ParamRef{Name: paramName, Index: index}, Args: args}
	id := &ir.ID{Index: int32(len(m.model.GetTypes())), Name: paramName}
	m.model.Types = append(m.model.Types, t)
	return id
}

func params(names ...string) []*ir.Param {
	out := make([]*ir.Param, len(names))
	for i, n := range names {
		out[i] = &ir.Param{Name: n}
	}
	return out
}

// structure is a value declaration, parameterized when ps is not empty.
func structure(declName string, ps []*ir.Param, fields ...*ir.Field) *ir.Decl {
	return &ir.Decl{
		Meta: &ir.Meta{Name: declName},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{Params: ps, Fields: fields}},
	}
}

func noDiagnostics(t *testing.T, resp *plugin.Response) {
	t.Helper()
	if len(resp.GetDiagnostics()) != 0 {
		t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
	}
}

// onlyWarningAt asserts the response carries one diagnostic, a warning, at
// line; a line of 0 is a use the fixture gave no position.
func onlyWarningAt(t *testing.T, resp *plugin.Response, line int32) {
	t.Helper()
	if len(resp.GetDiagnostics()) != 1 {
		t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
	}
	d := resp.GetDiagnostics()[0]
	if d.GetSeverity() != plugin.Severity_SEVERITY_WARNING {
		t.Errorf("severity = %v", d.GetSeverity())
	}
	if line != 0 && d.GetPosition().GetLine() != line {
		t.Errorf("position = %+v, want line %d", d.GetPosition(), line)
	}
}

// A parameterized declaration is a Go generic type, and applying it is
// instantiating one.
func TestGenericStruct(t *testing.T) {
	m := newModel("shop")
	m.own(structure("Page", params("T"),
		field("items", m.named("List", m.param("T", 0))),
		field("next", m.named("Option", m.param("T", 0))),
	))
	m.own(structure("Order", nil, field("id", m.named("string"))))
	m.own(structure("Orders", nil,
		field("page", m.named("Page", m.named("Order"))),
		field("byName", m.named("Map", m.named("string"), m.named("Page", m.named("int")))),
	))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	contains(t, got["page.go"], "type Page[T any] struct {", "Items []T", "Next *T")
	contains(t, got["orders.go"], "Page Page[Order]", "ByName map[string]Page[int64]")
}

// A sealed enum's marker method takes the enum's parameters, so a variant of
// one instantiation does not satisfy another.
func TestGenericSealedEnum(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Result"},
		Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{
			Params: params("T"),
			Variants: []*ir.Variant{
				{Meta: &ir.Meta{Name: "Ok"}, Fields: []*ir.Field{field("value", m.param("T", 0))}},
				{Meta: &ir.Meta{Name: "Err"}, Fields: []*ir.Field{field("message", m.named("string"))}},
			},
		}},
	})
	m.own(structure("Reply", nil, field("result", m.named("Result", m.named("string")))))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	contains(t, got["result.go"],
		"type Result[T any] interface{ isResult(T) }",
		"type ResultOk[T any] struct {",
		"Value T",
		"func (ResultOk[T]) isResult(T) {}",
		"type ResultErr[T any] struct {",
		"func (ResultErr[T]) isResult(T) {}",
	)
	contains(t, got["reply.go"], "Result Result[string]")
}

func TestGenericNewtype(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Ids"},
		Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Params: params("T"), Base: m.named("List", m.param("T", 0))}},
	})
	m.own(structure("User", nil, field("friends", m.named("Ids", m.named("string")))))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	contains(t, got["ids.go"], "type Ids[T any] []T")
	contains(t, got["user.go"], "Friends Ids[string]")
}

// Go refuses a type parameter as the whole right-hand side of a type
// declaration, so a newtype over a bare parameter has no Go shape.
func TestNewtypeOverABareParameterIsAWarning(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Tagged", Position: &ir.Position{Filename: "shop.tdl", Line: 3}},
		Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Params: params("T"), Base: m.param("T", 0)}},
	})
	m.own(structure("Note", nil, field("body", m.named("string"))))

	resp := generate(t, m)
	onlyWarningAt(t, resp, 3)
	if src, ok := files(t, resp)["tagged.go"]; ok {
		t.Errorf("a newtype over a bare parameter was emitted:\n%s", src)
	}
}

// A fieldless enum is constants, and a constant cannot have a generic type,
// so its parameters are dropped with a warning rather than changing the
// enum's shape.
func TestParameterizedFieldlessEnumDropsItsParams(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Status", Position: &ir.Position{Filename: "shop.tdl", Line: 3}},
		Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{
			Params:   params("T"),
			Variants: []*ir.Variant{{Meta: &ir.Meta{Name: "Active"}}, {Meta: &ir.Meta{Name: "Pending"}}},
		}},
	})
	m.own(structure("Job", nil, field("status", m.named("Status", m.named("int")))))

	resp := generate(t, m)
	onlyWarningAt(t, resp, 3)
	got := files(t, resp)
	contains(t, got["status.go"], "type Status string", `StatusActive Status = "Active"`)
	contains(t, got["job.go"], "Status Status")
}

// An alias is expanded at every use, so a parameterized one is expanded with
// its arguments substituted for its parameters.
func TestParameterizedAliasIsSubstituted(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Pair"},
		Node: &ir.Decl_Alias{Alias: &ir.Alias{
			Params: params("K", "V"),
			Target: m.named("Map", m.param("K", 0), m.param("V", 1)),
		}},
	})
	// An alias of an alias, applying the inner one to its own parameter.
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "ByName"},
		Node: &ir.Decl_Alias{Alias: &ir.Alias{
			Params: params("V"),
			Target: m.named("Pair", m.named("string"), m.param("V", 0)),
		}},
	})
	m.own(structure("Wrap", params("T"),
		field("direct", m.named("Pair", m.named("string"), m.named("bool"))),
		field("nested", m.named("ByName", m.named("int"))),
		field("generic", m.named("Pair", m.named("string"), m.param("T", 0))),
	))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	if _, ok := got["pair.go"]; ok {
		t.Error("an alias should not generate a declaration")
	}
	contains(t, got["wrap.go"],
		"Direct map[string]bool",
		"Nested map[string]int64",
		"Generic map[string]T",
	)
}

func TestAliasGivenTheWrongNumberOfArgumentsIsAWarning(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Pair"},
		Node: &ir.Decl_Alias{Alias: &ir.Alias{
			Params: params("K", "V"),
			Target: m.named("Map", m.param("K", 0), m.param("V", 1)),
		}},
	})
	m.own(structure("Bad", nil, field("pair", m.named("Pair", m.named("string")))))

	resp := generate(t, m)
	onlyWarningAt(t, resp, 0)
	if src, ok := files(t, resp)["bad.go"]; ok {
		t.Errorf("a declaration with a misapplied alias was emitted:\n%s", src)
	}
}

// A parameter Go cannot express is a warning at the parameter, and the
// declaration taking it is skipped.
func TestParamsGoCannotExpress(t *testing.T) {
	atom := func(a ir.KindAtom) *ir.Kind { return &ir.Kind{Atom: a} }
	arrow := &ir.Kind{Atom: ir.KindAtom_KIND_ATOM_TYPE, Arrow: atom(ir.KindAtom_KIND_ATOM_TYPE)}
	at := &ir.Position{Filename: "shop.tdl", Line: 4}

	cases := []struct {
		name  string
		param *ir.Param
		body  func(m *modelBuilder) *ir.ID
		line  int32
	}{
		{"a higher kind", &ir.Param{Name: "f", Kind: arrow, Position: at}, nil, 4},
		{"a parenthesized higher kind", &ir.Param{Name: "f", Kind: &ir.Kind{Paren: arrow}, Position: at}, nil, 4},
		{"a unit", &ir.Param{Name: "u", Kind: atom(ir.KindAtom_KIND_ATOM_UNIT), Position: at}, nil, 4},
		{"a Go keyword", &ir.Param{Name: "range", Position: at}, nil, 4},
		{"a predeclared type", &ir.Param{Name: "string", Position: at}, nil, 4},
		{"a predeclared constraint", &ir.Param{Name: "any", Position: at}, nil, 4},
		{"the time package", &ir.Param{Name: "time", Position: at}, nil, 4},
		{"a generated declaration", &ir.Param{Name: "Note", Position: at}, nil, 4},
		// The kind is left to inference, and only the use says it is higher.
		{"applied to arguments", &ir.Param{Name: "f"}, func(m *modelBuilder) *ir.ID {
			return m.param("f", 0, m.named("string"))
		}, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := newModel("shop")
			m.own(structure("Note", nil, field("body", m.named("string"))))
			body := m.named("string")
			if c.body != nil {
				body = c.body(m)
			}
			m.own(structure("Box", []*ir.Param{c.param}, field("value", body)))

			resp := generate(t, m)
			onlyWarningAt(t, resp, c.line)
			got := files(t, resp)
			if src, ok := got["box.go"]; ok {
				t.Errorf("a declaration taking %s was emitted:\n%s", c.name, src)
			}
			if _, ok := got["note.go"]; !ok {
				t.Errorf("the rest of the model was not generated: %v", keys(got))
			}
		})
	}
}

// Go needs a map key to be comparable and TDL has no way to say so, so a
// parameter is constrained by what reaches a key.
func TestComparableIsInferred(t *testing.T) {
	m := newModel("shop")
	m.own(structure("Bag", params("T"), field("items", m.named("Set", m.param("T", 0)))))
	m.own(structure("Index", params("K", "V"), field("by", m.named("Map", m.param("K", 0), m.param("V", 1)))))
	m.own(structure("Shelf", params("T"), field("bag", m.named("Bag", m.param("T", 0)))))
	m.own(structure("Pair", params("X"), field("x", m.param("X", 0))))
	m.own(structure("Pairs", params("T"), field("seen", m.named("Set", m.named("Pair", m.param("T", 0))))))
	// A pointer is comparable whatever it points at.
	m.own(structure("Maybe", params("T"), field("seen", m.named("Set", m.named("Option", m.param("T", 0))))))
	// Declared before what makes it comparable, so one pass in declaration
	// order is not enough. Its field is attached once Late exists.
	early := &ir.Struct{Params: params("T")}
	m.own(&ir.Decl{Meta: &ir.Meta{Name: "Early"}, Node: &ir.Decl_Structure{Structure: early}})
	m.own(structure("Late", params("T"), field("items", m.named("Set", m.param("T", 0)))))
	early.Fields = []*ir.Field{field("late", m.named("Late", m.param("T", 0)))}
	m.own(structure("Uses", nil, field("bag", m.named("Bag", m.named("string")))))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	contains(t, got["bag.go"], "type Bag[T comparable] struct {", "Items map[T]struct{}")
	contains(t, got["index.go"], "type Index[K comparable, V any] struct {")
	contains(t, got["shelf.go"], "type Shelf[T comparable] struct {", "Bag Bag[T]")
	contains(t, got["pair.go"], "type Pair[X any] struct {")
	contains(t, got["pairs.go"], "type Pairs[T comparable] struct {", "Seen map[Pair[T]]struct{}")
	contains(t, got["maybe.go"], "type Maybe[T any] struct {", "Seen map[*T]struct{}")
	contains(t, got["early.go"], "type Early[T comparable] struct {")
	contains(t, got["uses.go"], "Bag Bag[string]")
}

// An argument Go cannot compare, given to a parameter that has to be, is a
// warning at the use, and the declaration using it is skipped.
func TestComparableArgumentsAtUse(t *testing.T) {
	m := newModel("shop")
	m.own(structure("Bag", params("T"), field("items", m.named("Set", m.param("T", 0)))))
	m.own(structure("Path", nil, field("segments", m.named("List", m.named("string")))))
	m.own(structure("Blobs", nil, field("bag", m.named("Bag", m.named("bytes")))))
	m.own(structure("Paths", nil, field("bag", m.named("Bag", m.named("Path")))))
	m.own(structure("Names", nil, field("bag", m.named("Bag", m.named("string")))))

	resp := generate(t, m)
	if len(resp.GetDiagnostics()) != 2 {
		t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
	}
	got := files(t, resp)
	for _, skipped := range []string{"blobs.go", "paths.go"} {
		if src, ok := got[skipped]; ok {
			t.Errorf("%s gives an incomparable argument and was emitted:\n%s", skipped, src)
		}
	}
	contains(t, got["names.go"], "Bag Bag[string]")
}

// A generic entity's key is generic too, and a key field has to be
// comparable, so the parameter it names is.
func TestGenericEntityKey(t *testing.T) {
	m := newModel("shop")
	entry := keyed("Entry", []*ir.Field{
		field("id", m.param("T", 0)),
		field("note", m.named("string")),
	}, name("id"))
	entry.GetStructure().Params = params("T")
	m.own(entry)

	item := keyed("LineItem", []*ir.Field{
		field("order", m.param("T", 0)),
		field("sku", m.named("string")),
	}, name("order"), name("sku"))
	item.GetStructure().Params = params("T")
	m.own(item)

	// A parameter spelled like the receiver would be redeclared by it.
	edge := keyed("Edge", []*ir.Field{field("id", m.param("e", 0))}, name("id"))
	edge.GetStructure().Params = params("e")
	m.own(edge)

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	contains(t, got["entry.go"],
		"type Entry[T comparable] struct {",
		"func (e Entry[T]) Key() T {",
		"return e.Id",
	)
	contains(t, got["line_item.go"],
		"type LineItemKey[T comparable] struct {",
		"func (l LineItem[T]) Key() LineItemKey[T] {",
		"return LineItemKey[T]{Order: l.Order, Sku: l.Sku}",
	)
	contains(t, got["edge.go"], "func (r Edge[e]) Key() e {", "return r.Id")
}

// ref is the ID of a declaration, which is how a class reference and the
// satisfaction index name one.
func (m *modelBuilder) ref(declName string) *ir.ID {
	_, id, ok := m.model.FindDecl(declName)
	if !ok {
		panic("no declaration named " + declName)
	}
	return id
}

// class declares a class owned by the model, requiring the classes named.
func (m *modelBuilder) class(declName string, supers ...string) *ir.Decl {
	c := &ir.Class{}
	for _, s := range supers {
		c.RequiresClasses = append(c.RequiresClasses, &ir.ClassRef{Class: m.ref(s)})
	}
	d := &ir.Decl{Meta: &ir.Meta{Name: declName}, Node: &ir.Decl_Class{Class: c}}
	m.own(d)
	return d
}

// satisfies records declarations as satisfying a class, which is what the
// compiler computes from conformance and instances.
func (m *modelBuilder) satisfies(class string, decls ...string) {
	id := m.ref(class)
	var sat *ir.Satisfaction
	for _, s := range m.model.GetSatisfies() {
		if s.GetClass().GetIndex() == id.GetIndex() {
			sat = s
		}
	}
	if sat == nil {
		sat = &ir.Satisfaction{Class: id}
		m.model.Satisfies = append(m.model.Satisfies, sat)
	}
	for _, d := range decls {
		sat.Decls = append(sat.Decls, m.ref(d))
	}
}

// requires is a `requires` clause entry naming a class.
func (m *modelBuilder) requires(class string, line int32, args ...*ir.ID) *ir.ClassRef {
	return &ir.ClassRef{Class: m.ref(class), Args: args, Position: &ir.Position{Filename: "shop.tdl", Line: line}}
}

// extern interns a reference to a declaration in another package.
func (m *modelBuilder) extern(qualified string) *ir.ID {
	id := &ir.ID{Index: int32(len(m.model.GetTypes())), Name: qualified}
	m.model.Types = append(m.model.Types, &ir.Type{Extern: &ir.ID{Name: qualified}})
	return id
}

// A class is a contract, and conformance to it is declared, so it is an
// interface only the declarations that said so implement: its method is
// unexported, which seals it to the package.
func TestClassIsAMarkerInterface(t *testing.T) {
	m := newModel("shop")
	m.class("Timestamped")
	m.class("Auditable", "Timestamped")
	m.own(structure("Order", nil, field("id", m.named("string"))))
	m.satisfies("Timestamped", "Order")
	m.satisfies("Auditable", "Order")

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	contains(t, got["timestamped.go"], "type Timestamped interface { isTimestamped() }")
	contains(t, got["auditable.go"], "type Auditable interface { Timestamped isAuditable() }")
	contains(t, got["order.go"], "func (Order) isTimestamped() {}", "func (Order) isAuditable() {}")
}

func TestMarkersOnEveryShape(t *testing.T) {
	m := newModel("shop")
	m.class("Auditable")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Status"},
		Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{
			Variants: []*ir.Variant{{Meta: &ir.Meta{Name: "Active"}}},
		}},
	})
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Payment"},
		Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{
			Variants: []*ir.Variant{
				{Meta: &ir.Meta{Name: "Cash"}},
				{Meta: &ir.Meta{Name: "Card"}, Fields: []*ir.Field{field("last4", m.named("string"))}},
			},
		}},
	})
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Sku"},
		Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: m.named("string")}},
	})
	m.own(structure("Page", params("T"), field("items", m.named("List", m.param("T", 0)))))
	m.satisfies("Auditable", "Status", "Payment", "Sku", "Page")

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	contains(t, got["status.go"], "func (Status) isAuditable() {}")
	// A sealed enum is an interface, which cannot carry a method, so it
	// embeds the class and every variant carries the marker.
	contains(t, got["payment.go"],
		"type Payment interface { Auditable isPayment() }",
		"func (PaymentCash) isAuditable() {}",
		"func (PaymentCard) isAuditable() {}",
	)
	contains(t, got["sku.go"], "func (Sku) isAuditable() {}")
	contains(t, got["page.go"], "func (Page[T]) isAuditable() {}")
}

// A `requires` clause is a Go constraint, since the class it names is an
// interface, and a parameter that also has to be comparable says both.
func TestRequiresBecomesAConstraint(t *testing.T) {
	m := newModel("shop")
	m.class("Timestamped")
	m.class("Auditable", "Timestamped")
	m.own(structure("Order", nil, field("id", m.named("string"))))
	m.satisfies("Timestamped", "Order")
	m.satisfies("Auditable", "Order")

	envelope := structure("Envelope", params("T"), field("body", m.param("T", 0)))
	envelope.GetStructure().Constraints = []*ir.ClassRef{m.requires("Auditable", 0, m.param("T", 0))}
	m.own(envelope)
	stamped := structure("Stamped", params("T"), field("body", m.param("T", 0)))
	stamped.GetStructure().Constraints = []*ir.ClassRef{m.requires("Timestamped", 0, m.param("T", 0))}
	m.own(stamped)
	tracked := structure("Tracked", params("T"), field("seen", m.named("Set", m.param("T", 0))))
	tracked.GetStructure().Constraints = []*ir.ClassRef{m.requires("Auditable", 0, m.param("T", 0))}
	m.own(tracked)
	// Auditable requires Timestamped, so a parameter that is Auditable
	// satisfies a constraint naming either.
	forward := structure("Forward", params("T"),
		field("envelope", m.named("Envelope", m.param("T", 0))),
		field("stamped", m.named("Stamped", m.param("T", 0))),
	)
	forward.GetStructure().Constraints = []*ir.ClassRef{m.requires("Auditable", 0, m.param("T", 0))}
	m.own(forward)
	m.own(structure("Uses", nil,
		field("envelope", m.named("Envelope", m.named("Order"))),
		field("stamped", m.named("Stamped", m.named("Order"))),
		field("tracked", m.named("Tracked", m.named("Order"))),
	))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	contains(t, got["envelope.go"], "type Envelope[T Auditable] struct {")
	contains(t, got["stamped.go"], "type Stamped[T Timestamped] struct {")
	// gofmt spreads an interface of several elements over lines.
	contains(t, got["tracked.go"], "type Tracked[T interface { comparable Auditable }] struct {")
	contains(t, got["forward.go"], "type Forward[T Auditable] struct {")
	contains(t, got["uses.go"], "Envelope Envelope[Order]", "Tracked Tracked[Order]")
}

// The compiler does not check a constraint whose argument is a parameter,
// and Go does, so the backend checks every use and skips what Go would
// refuse.
func TestConstraintViolatedAtUse(t *testing.T) {
	m := newModel("shop")
	m.class("Auditable")
	m.own(structure("Order", nil, field("id", m.named("string"))))
	m.satisfies("Auditable", "Order")
	envelope := structure("Envelope", params("T"), field("body", m.param("T", 0)))
	envelope.GetStructure().Constraints = []*ir.ClassRef{m.requires("Auditable", 0, m.param("T", 0))}
	m.own(envelope)

	m.own(structure("Strings", nil, field("envelope", m.named("Envelope", m.named("string")))))
	m.own(structure("Outer", params("T"), field("envelope", m.named("Envelope", m.param("T", 0)))))
	m.own(structure("Good", nil, field("envelope", m.named("Envelope", m.named("Order")))))

	resp := generate(t, m)
	if len(resp.GetDiagnostics()) != 2 {
		t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
	}
	got := files(t, resp)
	for _, skipped := range []string{"strings.go", "outer.go"} {
		if src, ok := got[skipped]; ok {
			t.Errorf("%s breaks a constraint and was emitted:\n%s", skipped, src)
		}
	}
	contains(t, got["good.go"], "Envelope Envelope[Order]")
}

// What Go cannot express about a class is a warning where it was written,
// and the rest is generated without it.
func TestClassesGoCannotExpress(t *testing.T) {
	at := &ir.Position{Filename: "shop.tdl", Line: 4}
	instance := func(m *modelBuilder, arg *ir.ID) *ir.Instance {
		return &ir.Instance{
			Meta:  &ir.Meta{Position: at},
			Class: &ir.ClassRef{Class: m.ref("Auditable"), Args: []*ir.ID{arg}},
		}
	}

	cases := []struct {
		name string
		// build returns the file expected, and what must not be in it.
		build func(m *modelBuilder) (file, absent string)
	}{
		{"a class taking parameters", func(m *modelBuilder) (string, string) {
			d := m.class("Projection")
			d.GetMeta().Position = at
			d.GetClass().Params = params("from", "to")
			return "", "projection.go"
		}},
		{"a class with an associated type", func(m *modelBuilder) (string, string) {
			d := m.class("Paged")
			d.GetMeta().Position = at
			d.GetClass().AssocTypes = []*ir.AssocType{{Meta: &ir.Meta{Name: "Cursor"}}}
			return "paged.go", ""
		}},
		{"a conditional instance", func(m *modelBuilder) (string, string) {
			m.class("Auditable")
			m.own(structure("Page", params("T"), field("items", m.named("List", m.param("T", 0)))))
			inst := instance(m, m.named("Page", m.param("T", 0)))
			inst.Params = params("T")
			inst.Requires = []*ir.ClassRef{{Class: m.ref("Auditable"), Args: []*ir.ID{m.param("T", 0)}}}
			m.model.Instances = append(m.model.Instances, inst)
			return "page.go", "isAuditable"
		}},
		{"an instance for a prelude type", func(m *modelBuilder) (string, string) {
			m.class("Auditable")
			m.model.Instances = append(m.model.Instances, instance(m, m.named("string")))
			return "auditable.go", "func (string)"
		}},
		{"an instance for a type in another package", func(m *modelBuilder) (string, string) {
			m.class("Auditable")
			m.model.Instances = append(m.model.Instances, instance(m, m.extern("acme.Money")))
			return "auditable.go", "Money"
		}},
		{"a newtype over a pointer", func(m *modelBuilder) (string, string) {
			m.class("Auditable")
			m.own(&ir.Decl{
				Meta: &ir.Meta{Name: "Maybe", Position: at},
				Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: m.named("Option", m.named("string"))}},
			})
			m.satisfies("Auditable", "Maybe")
			return "maybe.go", "isAuditable"
		}},
		{"a newtype over a sealed enum", func(m *modelBuilder) (string, string) {
			m.class("Auditable")
			m.own(&ir.Decl{
				Meta: &ir.Meta{Name: "Payment"},
				Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{
					Variants: []*ir.Variant{{Meta: &ir.Meta{Name: "Card"}, Fields: []*ir.Field{field("last4", m.named("string"))}}},
				}},
			})
			m.own(&ir.Decl{
				Meta: &ir.Meta{Name: "Tender", Position: at},
				Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: m.named("Payment")}},
			})
			m.satisfies("Auditable", "Tender")
			return "tender.go", "isAuditable"
		}},
		{"a field named like the marker", func(m *modelBuilder) (string, string) {
			m.class("Auditable")
			note := structure("Note", nil, &ir.Field{
				Meta:       &ir.Meta{Name: "audit"},
				Type:       m.named("string"),
				Directives: []*ir.Directive{{Name: "name", Target: "go", Args: []*ir.Literal{text("isAuditable")}}},
			})
			note.GetMeta().Position = at
			m.own(note)
			m.satisfies("Auditable", "Note")
			return "note.go", "func (Note)"
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := newModel("shop")
			file, absent := c.build(m)

			resp := generate(t, m)
			onlyWarningAt(t, resp, 4)
			got := files(t, resp)
			if file != "" {
				src, ok := got[file]
				if !ok {
					t.Fatalf("no %s in %v", file, keys(got))
				}
				if absent != "" && strings.Contains(src, absent) {
					t.Errorf("%s holds %q:\n%s", file, absent, src)
				}
			} else if _, ok := got[absent]; ok {
				t.Errorf("%s was emitted: %v", absent, keys(got))
			}
		})
	}
}

// A `requires` clause Go cannot state is a warning at the clause, and the
// declaration is emitted without that constraint, as it is without an
// unenforced `where`.
func TestUnexpressibleRequiresIsDropped(t *testing.T) {
	cases := []struct {
		name  string
		ref   func(m *modelBuilder) *ir.ClassRef
		diags int
	}{
		{"a prelude class", func(m *modelBuilder) *ir.ClassRef {
			return m.requires("Entity", 4, m.param("T", 0))
		}, 1},
		// The class warns where it is declared, and the clause where it is
		// written.
		{"a class taking parameters", func(m *modelBuilder) *ir.ClassRef {
			d := m.class("Projection")
			d.GetMeta().Position = &ir.Position{Filename: "shop.tdl", Line: 2}
			d.GetClass().Params = params("from", "to")
			return m.requires("Projection", 4, m.param("T", 0), m.named("string"))
		}, 2},
		{"an argument that is not a parameter", func(m *modelBuilder) *ir.ClassRef {
			m.class("Auditable")
			return m.requires("Auditable", 4, m.named("List", m.param("T", 0)))
		}, 1},
		{"a class in another package", func(m *modelBuilder) *ir.ClassRef {
			return &ir.ClassRef{
				Extern:   &ir.ID{Name: "acme.Auditable"},
				Args:     []*ir.ID{m.param("T", 0)},
				Position: &ir.Position{Filename: "shop.tdl", Line: 4},
			}
		}, 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := newModel("shop")
			ref := c.ref(m)
			envelope := structure("Envelope", params("T"), field("body", m.param("T", 0)))
			envelope.GetStructure().Constraints = []*ir.ClassRef{ref}
			m.own(envelope)

			resp := generate(t, m)
			if len(resp.GetDiagnostics()) != c.diags {
				t.Fatalf("diagnostics = %+v", resp.GetDiagnostics())
			}
			atClause := false
			for _, d := range resp.GetDiagnostics() {
				atClause = atClause || d.GetPosition().GetLine() == 4
			}
			if !atClause {
				t.Errorf("no warning at the clause: %+v", resp.GetDiagnostics())
			}
			contains(t, files(t, resp)["envelope.go"], "type Envelope[T any] struct {")
		})
	}
}
