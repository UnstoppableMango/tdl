package golang_test

import (
	"context"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
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

// sourceImporter type checks what generated code imports from GOROOT rather
// than a build cache, so it needs nothing installed and no module on disk;
// the generated package imports only the standard library. It is shared
// because it keeps what it has checked, and checking fmt and regexp from
// source for every test would dominate the run.
var sourceImporter = importer.ForCompiler(token.NewFileSet(), "source", nil)

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

	conf := types.Config{Importer: sourceImporter}
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

// The constraint set is open, so a name the backend does not know warns at
// the constraint, and the checks it does know are still generated, as is
// the type every field naming it refers to.
func TestUnknownConstraintIsAWarning(t *testing.T) {
	m := newModel("shop")
	m.own(newtype("Quantity", m.named("int"), where("min", 3, intArg("1")), where("shout", 8)))
	m.own(structure("Order", nil, field("quantity", m.named("Quantity"))))

	resp := generate(t, m)
	onlyWarningAt(t, resp, 8)
	got := files(t, resp)
	contains(t, got["quantity.go"], "type Quantity int64", "if q < 1 {")
	contains(t, got["order.go"], "Quantity Quantity")
}

// A newtype's constraints are a Validate method, joining every violation,
// and an unexported validate that threads the path a container prefixes.
func TestNewtypeMinMax(t *testing.T) {
	m := newModel("shop")
	m.own(newtype("Quantity", m.named("int"), where("min", 3, intArg("1")), where("max", 4, intArg("100"))))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	contains(t, files(t, resp)["quantity.go"],
		"func (q Quantity) Validate() error {",
		`return errors.Join(q.validate("Quantity", nil)...)`,
		"func (q Quantity) validate(path string, errs []error) []error {",
		"if q < 1 {",
		`errs = append(errs, fmt.Errorf("%s: min(1): got %d", path, q))`,
		"if q > 100 {",
	)
}

// TestValidationRuns is where a check is shown to fail on a value violating
// it, rather than only to compile: the generated package is built and a test
// written against it is run.
func TestValidationRuns(t *testing.T) {
	m := newModel("shop")
	m.own(newtype("Quantity", m.named("int"), where("min", 3, intArg("1")), where("max", 4, intArg("100"))))
	m.own(newtype("Email", m.named("string"),
		where("matches", 5, regexArg("^[^@]+@[^@]+$")),
		where("length", 6, rangeArg(bound(3), bound(254))),
	))
	m.own(newtype("Initials", m.named("string"), where("length", 7, rangeArg(bound(1), bound(3)))))
	m.own(enum("Status", variant("Active"), variant("Pending"), variant("Closed")))
	m.own(enum("Payment",
		variant("Cash"),
		variant("Card", constrained(field("last4", m.named("string")), where("length", 8, intArg("4")))),
	))
	m.own(structure("LineItem", nil,
		constrained(field("quantity", m.named("int")), where("min", 9, intArg("1")), where("max", 9, intArg("100"))),
		constrained(field("size", m.named("string")), where("oneOf", 10, text("S"), text("M"), text("L"))),
		constrained(field("tags", m.named("List", m.named("string"))), where("unique", 11)),
		constrained(field("status", m.named("Status")), where("oneOf", 12, name("Active"), name("Pending"))),
		constrained(field("note", m.named("Option", m.named("string"))), where("length", 13, rangeArg(bound(1), bound(5)))),
		field("contact", m.named("Email")),
	))
	m.own(structure("Order", nil,
		constrained(field("items", m.named("List", m.named("LineItem"))), where("length", 14, rangeArg(bound(1), nil))),
		field("payment", m.named("Payment")),
	))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	goTest(t, resp, validationTest)
}

// validationTest is run inside the generated package by [TestValidationRuns].
// Error lines are compared as sets, since a map is walked in no fixed order.
const validationTest = `package shop

import (
	"slices"
	"strings"
	"testing"
)

func check(t *testing.T, err error, want ...string) {
	t.Helper()
	var got []string
	if err != nil {
		got = strings.Split(err.Error(), "\n")
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("errors = %q, want %q", got, want)
	}
}

func TestQuantity(t *testing.T) {
	check(t, Quantity(5).Validate())
	check(t, Quantity(0).Validate(), "Quantity: min(1): got 0")
	check(t, Quantity(101).Validate(), "Quantity: max(100): got 101")
}

func TestEmail(t *testing.T) {
	check(t, Email("a@b.c").Validate())
	check(t, Email("ab").Validate(),
		"Email: matches(/^[^@]+@[^@]+$/): no match",
		"Email: length(3..254): got 2",
	)
}

// A string's length is its characters, not its bytes.
func TestRunes(t *testing.T) {
	check(t, Initials("hé").Validate())
	check(t, Initials("héll").Validate(), "Initials: length(1..3): got 4")
}

func TestLineItem(t *testing.T) {
	good := LineItem{Quantity: 1, Size: "S", Tags: []string{"a"}, Status: StatusActive, Contact: "a@b.c"}
	check(t, good.Validate())

	note := "toolong"
	bad := LineItem{Quantity: 0, Size: "XL", Tags: []string{"a", "a"}, Status: StatusClosed, Note: &note, Contact: "ab"}
	check(t, bad.Validate(),
		"LineItem.quantity: min(1): got 0",
		"LineItem.size: oneOf(\"S\", \"M\", \"L\"): not one of them",
		"LineItem.tags: unique: [1] repeats [0]",
		"LineItem.status: oneOf(Active, Pending): got \"Closed\"",
		"LineItem.note: length(1..5): got 7",
		"LineItem.contact: matches(/^[^@]+@[^@]+$/): no match",
		"LineItem.contact: length(3..254): got 2",
	)
}

func TestOrder(t *testing.T) {
	check(t, Order{}.Validate(), "Order.items: length(1..): got 0")

	o := Order{
		Items: []LineItem{
			{Quantity: 1, Size: "S", Status: StatusActive, Contact: "a@b.c"},
			{Quantity: 200, Size: "M", Status: StatusPending, Contact: "a@b.c"},
		},
		Payment: PaymentCard{Last4: "12"},
	}
	check(t, o.Validate(),
		"Order.items[1].quantity: max(100): got 200",
		"Order.payment.last4: length(4): got 2",
	)
	check(t, PaymentCard{Last4: "12"}.Validate(), "Payment.Card.last4: length(4): got 2")
	check(t, Order{Items: o.Items[:1], Payment: PaymentCash{}}.Validate())
}
`

// goTest runs a test file inside the generated package, in a module of its
// own, since type checking shows a check compiles and only running it shows
// what it rejects.
func goTest(t *testing.T, resp *plugin.Response, test string) {
	t.Helper()
	if testing.Short() {
		t.Skip("runs go test on the generated package")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go is not on PATH")
	}
	got := files(t, resp)

	dir := t.TempDir()
	write := func(name, src string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/generated\n\ngo 1.21\n")
	for path, src := range got {
		write(path, src)
	}
	write("tdl_validation_test.go", test)

	cmd := exec.CommandContext(t.Context(), goBin, "test", "-count=1", ".")
	cmd.Dir = dir
	// The generated package imports only the standard library, so nothing
	// here needs a network, a workspace, or another toolchain.
	cmd.Env = append(os.Environ(), "GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go test on the generated package: %v\n%s\n%s", err, out, strings.Join(sources(got), "\n"))
	}
}

// newtype is a newtype owned by the model, carrying constraints.
func newtype(declName string, base *ir.ID, cs ...*ir.Constraint) *ir.Decl {
	return &ir.Decl{
		Meta: &ir.Meta{Name: declName},
		Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: base, ValueConstraints: cs}},
	}
}

// where is a constraint as a `where` block writes it, at a line.
func where(constraintName string, line int32, args ...*ir.Literal) *ir.Constraint {
	return &ir.Constraint{Name: constraintName, Args: args, Position: &ir.Position{Filename: "shop.tdl", Line: line}}
}

func intArg(s string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: s}
}

func floatArg(s string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_FLOAT, Text: s}
}

func regexArg(s string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_REGEX, Text: s}
}

func boolArg(s string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_BOOL, Text: s}
}

// rangeArg is a range literal; a nil bound is an open end.
func rangeArg(low, high *int64) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_RANGE, Range: &ir.Range{Low: low, High: high}}
}

func bound(n int64) *int64 { return &n }

func enum(declName string, variants ...*ir.Variant) *ir.Decl {
	return &ir.Decl{
		Meta: &ir.Meta{Name: declName},
		Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{Variants: variants}},
	}
}

func variant(variantName string, fields ...*ir.Field) *ir.Variant {
	return &ir.Variant{Meta: &ir.Meta{Name: variantName}, Fields: fields}
}

func constrained(f *ir.Field, cs ...*ir.Constraint) *ir.Field {
	f.Constraints = cs
	return f
}

func TestLength(t *testing.T) {
	m := newModel("shop")
	m.own(newtype("Initials", m.named("string"), where("length", 3, rangeArg(bound(1), bound(3)))))
	m.own(newtype("Blob", m.named("bytes"), where("length", 4, intArg("16"))))
	m.own(newtype("Tags", m.named("List", m.named("string")), where("length", 5, rangeArg(nil, bound(5)))))
	m.own(newtype("Index", m.named("Map", m.named("string"), m.named("int")), where("length", 6, rangeArg(bound(1), nil))))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	contains(t, got["initials.go"], "if count := utf8.RuneCountInString(string(i)); count < 1 || count > 3 {")
	contains(t, got["blob.go"], "if count := len(b); count != 16 {")
	contains(t, got["tags.go"], "if count := len(t); count > 5 {")
	contains(t, got["index.go"], "if count := len(i); count < 1 {")
}

// A range constraining nothing, or nothing at all, is a mistake in the
// model, and a check that can never pass is not generated.
func TestLengthThatCannotHoldIsAWarning(t *testing.T) {
	for _, tt := range []struct {
		name string
		arg  *ir.Literal
	}{
		{"an unbounded range", rangeArg(nil, nil)},
		{"a range whose low end passes its high", rangeArg(bound(5), bound(3))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel("shop")
			m.own(newtype("Name", m.named("string"), where("length", 4, tt.arg)))
			resp := generate(t, m)
			onlyWarningAt(t, resp, 4)
			if src := files(t, resp)["name.go"]; strings.Contains(src, "Validate") {
				t.Errorf("a length that cannot hold was checked:\n%s", src)
			}
		})
	}
}

func TestMatches(t *testing.T) {
	m := newModel("shop")
	m.own(newtype("Slug", m.named("string"), where("matches", 3, regexArg("^[a-z-]+$"))))
	// A raw string cannot hold a backquote.
	m.own(newtype("Quoted", m.named("string"), where("matches", 4, regexArg("a`b"))))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	contains(t, got["slug.go"],
		"var patternSlug_0 = regexp.MustCompile(`^[a-z-]+$`)",
		"if !patternSlug_0.MatchString(string(s)) {",
		`errs = append(errs, fmt.Errorf("%s: matches(/^[a-z-]+$/): no match", path))`,
	)
	contains(t, got["quoted.go"], "var patternQuoted_0 = regexp.MustCompile(\"a`b\")")
}

// Go's regexp is RE2, which refuses what other engines accept, and a
// pattern it refuses would panic when the package loads.
func TestMatchesGoRefusesIsAWarning(t *testing.T) {
	m := newModel("shop")
	m.own(newtype("After", m.named("string"), where("matches", 4, regexArg("(?<=a)b"))))

	resp := generate(t, m)
	onlyWarningAt(t, resp, 4)
	if src := files(t, resp)["after.go"]; strings.Contains(src, "regexp") {
		t.Errorf("a pattern Go refuses was compiled:\n%s", src)
	}
}

func TestOneOf(t *testing.T) {
	m := newModel("shop")
	m.own(enum("Status", variant("Active"), variant("Pending")))
	m.own(structure("Order", nil,
		constrained(field("size", m.named("string")), where("oneOf", 3, text("S"), text("M"))),
		constrained(field("level", m.named("int")), where("oneOf", 4, intArg("1"), intArg("2"))),
		constrained(field("ratio", m.named("int")), where("oneOf", 5, floatArg("0.5"))),
		constrained(field("flag", m.named("bool")), where("oneOf", 6, boolArg("true"))),
		constrained(field("status", m.named("Status")), where("oneOf", 7, name("Active"))),
	))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	contains(t, files(t, resp)["order.go"],
		`if o.Size != "S" && o.Size != "M" {`,
		"if o.Level != 1 && o.Level != 2 {",
		"if float64(o.Ratio) != 0.5 {",
		"if o.Flag != true {",
		"if o.Status != StatusActive {",
	)
}

func TestOneOfItCannotCheckIsAWarning(t *testing.T) {
	for _, tt := range []struct {
		name string
		typ  string
		arg  *ir.Literal
	}{
		{"a variant the enum lacks", "Status", name("Closed")},
		{"a quoted variant", "Status", text("Active")},
		{"a list", "string", &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_LIST, Items: []*ir.Literal{text("a")}}},
		{"an integer for a string", "string", intArg("1")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel("shop")
			m.own(enum("Status", variant("Active")))
			m.own(structure("Order", nil, constrained(field("value", m.named(tt.typ)), where("oneOf", 4, tt.arg))))
			resp := generate(t, m)
			onlyWarningAt(t, resp, 4)
			if src := files(t, resp)["order.go"]; strings.Contains(src, "Validate") {
				t.Errorf("a oneOf it cannot check was generated:\n%s", src)
			}
		})
	}
}

func TestUnique(t *testing.T) {
	m := newModel("shop")
	m.own(structure("Bag", nil,
		constrained(field("tags", m.named("List", m.named("string"))), where("unique", 3)),
		// A set holds distinct values already.
		constrained(field("set", m.named("Set", m.named("string"))), where("unique", 4)),
		constrained(field("blobs", m.named("List", m.named("bytes"))), where("unique", 5)),
	))

	resp := generate(t, m)
	onlyWarningAt(t, resp, 5)
	contains(t, files(t, resp)["bag.go"], "seen := make(map[string]int, len(b.Tags))")
}

// A constraint whose meaning this backend cannot give to a type warns, and
// the rest of the type is still checked.
func TestConstraintOnTheWrongTypeIsAWarning(t *testing.T) {
	for _, tt := range []struct {
		name string
		typ  func(m *modelBuilder) *ir.ID
		c    *ir.Constraint
	}{
		{"min on a string", func(m *modelBuilder) *ir.ID { return m.named("string") }, where("min", 4, intArg("1"))},
		{"min on a decimal", func(m *modelBuilder) *ir.ID { return m.named("decimal") }, where("min", 4, intArg("1"))},
		{"length on a decimal", func(m *modelBuilder) *ir.ID { return m.named("decimal") }, where("length", 4, intArg("3"))},
		{"min on a duration", func(m *modelBuilder) *ir.ID { return m.named("duration") }, where("min", 4, intArg("1"))},
		{"matches on an integer", func(m *modelBuilder) *ir.ID { return m.named("int") }, where("matches", 4, regexArg("^1$"))},
		{"min on a type parameter", func(m *modelBuilder) *ir.ID { return m.param("T", 0) }, where("min", 4, intArg("1"))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel("shop")
			m.own(structure("Box", params("T"),
				constrained(field("value", tt.typ(m)), tt.c),
				constrained(field("count", m.named("int")), where("min", 9, intArg("0"))),
			))
			resp := generate(t, m)
			onlyWarningAt(t, resp, 4)
			contains(t, files(t, resp)["box.go"], "if b.Count < 0 {")
		})
	}
}

// A field whose type validates is validated with it, through a pointer and
// through every collection, and a struct with nothing to check gets no
// methods.
func TestValidationRecurses(t *testing.T) {
	m := newModel("shop")
	m.own(newtype("Email", m.named("string"), where("length", 3, rangeArg(bound(3), nil))))
	m.own(structure("Contact", nil, field("email", m.named("Email"))))
	m.own(structure("Plain", nil, field("name", m.named("string"))))
	m.own(structure("Book", nil,
		field("contacts", m.named("List", m.named("Contact"))),
		field("primary", m.named("Option", m.named("Contact"))),
		field("byName", m.named("Map", m.named("string"), m.named("Contact"))),
		field("grid", m.named("List", m.named("List", m.named("Contact")))),
		field("plain", m.named("Plain")),
	))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	contains(t, got["contact.go"], `errs = c.Email.validate(path+".email", errs)`)
	contains(t, got["book.go"],
		"for idx, elem := range b.Contacts {",
		`errs = elem.validate(fmt.Sprintf("%s.contacts[%d]", path, idx), errs)`,
		"if b.Primary != nil {",
		`errs = (*b.Primary).validate(path+".primary", errs)`,
		"for _, val := range b.ByName {",
		`errs = val.validate(path+".byName[?]", errs)`,
		"for idx1, elem1 := range elem {",
	)
	if strings.Contains(got["book.go"], "Plain.validate") || strings.Contains(got["plain.go"], "Validate") {
		t.Errorf("a struct with nothing to check was validated:\n%s\n%s", got["plain.go"], got["book.go"])
	}
}

// Two structs reaching each other through pointers settle on both
// validating, which a walk stopping at what it has seen would get wrong.
func TestValidationThroughACycle(t *testing.T) {
	m := newModel("shop")
	b := &ir.Struct{}
	m.own(&ir.Decl{Meta: &ir.Meta{Name: "B"}, Node: &ir.Decl_Structure{Structure: b}})
	m.own(structure("A", nil,
		field("b", m.named("Option", m.named("B"))),
		constrained(field("n", m.named("int")), where("min", 3, intArg("1"))),
	))
	b.Fields = []*ir.Field{field("a", m.named("Option", m.named("A")))}

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	contains(t, got["b.go"], `errs = (*b.A).validate(path+".a", errs)`)
	contains(t, got["a.go"], `errs = (*a.B).validate(path+".b", errs)`)
}

// The compiler hands a newtype its whole accumulated set, so it is checked
// in one place, and a constraint the backend cannot check warns where it
// was written and not again where it was inherited.
func TestNewtypeChainChecksOnce(t *testing.T) {
	m := newModel("shop")
	email := newtype("Email", m.named("string"), where("length", 3, rangeArg(bound(3), nil)), where("shout", 4))
	m.own(email)
	inherited := func(c *ir.Constraint) *ir.Constraint {
		return &ir.Constraint{Name: c.GetName(), Args: c.GetArgs(), Position: c.GetPosition(), From: m.ref("Email")}
	}
	cs := email.GetNewtype().GetValueConstraints()
	m.own(newtype("WorkEmail", m.named("Email"), where("matches", 5, regexArg("@acme$")), inherited(cs[0]), inherited(cs[1])))

	resp := generate(t, m)
	onlyWarningAt(t, resp, 4)
	src := files(t, resp)["work_email.go"]
	contains(t, src, "if !patternWorkEmail_0.MatchString(string(w)) {", "count < 3")
	if strings.Contains(src, "Email(w)") {
		t.Errorf("an accumulated constraint was checked through the parent:\n%s", src)
	}
}

func TestNewtypeOverAStructValidatesIt(t *testing.T) {
	m := newModel("shop")
	m.own(structure("Person", nil, constrained(field("age", m.named("int")), where("min", 3, intArg("0")))))
	m.own(newtype("Customer", m.named("Person")))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	contains(t, files(t, resp)["customer.go"], "errs = Person(c).validate(path, errs)")
}

// Go gives no method to a type whose underlying type is a pointer or an
// interface, so such a newtype's constraints cannot be checked.
func TestConstrainedNewtypeOverAPointerIsAWarning(t *testing.T) {
	m := newModel("shop")
	m.own(&ir.Decl{
		Meta: &ir.Meta{Name: "Maybe", Position: &ir.Position{Filename: "shop.tdl", Line: 3}},
		Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{
			Base:             m.named("Option", m.named("string")),
			ValueConstraints: []*ir.Constraint{where("length", 4, rangeArg(bound(1), nil))},
		}},
	})

	resp := generate(t, m)
	onlyWarningAt(t, resp, 3)
	if src := files(t, resp)["maybe.go"]; strings.Contains(src, "Validate") {
		t.Errorf("a newtype over a pointer was given methods:\n%s", src)
	}
}

// A sealed enum is an interface, so each variant with something to check
// carries the methods, and a field holding the enum asks the value it holds.
func TestSealedEnumValidation(t *testing.T) {
	m := newModel("shop")
	m.own(enum("Payment",
		variant("Cash"),
		variant("Card", constrained(field("last4", m.named("string")), where("length", 3, intArg("4")))),
	))
	m.own(structure("Order", nil, field("payment", m.named("Payment"))))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	got := files(t, resp)
	contains(t, got["payment.go"],
		`return errors.Join(p.validate("Payment.Card", nil)...)`,
		"func (p PaymentCard) validate(path string, errs []error) []error {",
	)
	if strings.Contains(got["payment.go"], "func (p PaymentCash) Validate") {
		t.Errorf("a variant with nothing to check was given methods:\n%s", got["payment.go"])
	}
	contains(t, got["order.go"],
		"if inner, ok := o.Payment.(interface{ validate(string, []error) []error }); ok {",
		`errs = inner.validate(path+".payment", errs)`,
	)
}

// A generic struct checks its own fields and leaves its type arguments'
// values alone, since asserting a method on a T that is a nil pointer
// panics.
func TestGenericValidation(t *testing.T) {
	m := newModel("shop")
	m.own(structure("Page", params("T"),
		constrained(field("items", m.named("List", m.param("T", 0))), where("length", 3, rangeArg(nil, bound(50)))),
		constrained(field("total", m.named("int")), where("min", 4, intArg("0"))),
	))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	contains(t, files(t, resp)["page.go"],
		"func (p Page[T]) Validate() error {",
		"func (p Page[T]) validate(path string, errs []error) []error {",
		"if count := len(p.Items); count > 50 {",
	)
}

// The method's parameters and locals share a scope with the type's
// parameters, so none may be spelled like one.
func TestLocalsAvoidTypeParams(t *testing.T) {
	m := newModel("shop")
	m.own(structure("Box", params("path", "errs", "count", "seen"),
		constrained(field("tags", m.named("List", m.named("string"))), where("length", 3, rangeArg(bound(1), nil)), where("unique", 4)),
	))

	resp := generate(t, m)
	noDiagnostics(t, resp)
	contains(t, files(t, resp)["box.go"], "Validate() error {")
}

// A field Go would call Validate leaves no room for the method, which warns
// the way a field called Key does; a key and a check live side by side.
func TestValidateCollision(t *testing.T) {
	m := newModel("shop")
	clash := structure("Clash", nil, constrained(field("validate", m.named("int")), where("min", 4, intArg("1"))))
	clash.GetMeta().Position = &ir.Position{Filename: "shop.tdl", Line: 3}
	m.own(clash)
	m.own(keyed("User", []*ir.Field{
		field("id", m.named("string")),
		constrained(field("age", m.named("int")), where("min", 9, intArg("0"))),
	}, name("id")))

	resp := generate(t, m)
	onlyWarningAt(t, resp, 3)
	got := files(t, resp)
	if strings.Contains(got["clash.go"], "func (c Clash) Validate") {
		t.Errorf("a method collided with a field:\n%s", got["clash.go"])
	}
	contains(t, got["user.go"], "func (u User) Key() string {", "func (u User) Validate() error {")
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
		// Validation imports these, so a parameter spelled like one would
		// shadow it in whichever file it lands.
		{"the fmt package", &ir.Param{Name: "fmt", Position: at}, nil, 4},
		{"the errors package", &ir.Param{Name: "errors", Position: at}, nil, 4},
		{"the regexp package", &ir.Param{Name: "regexp", Position: at}, nil, 4},
		{"the utf8 package", &ir.Param{Name: "utf8", Position: at}, nil, 4},
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
