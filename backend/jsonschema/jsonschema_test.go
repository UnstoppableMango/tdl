package jsonschema_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	sjs "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/backend/jsonschema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

func generate(t *testing.T, b *irtest.Builder) *plugin.Response {
	t.Helper()
	resp, err := jsonschema.Backend{}.Generate(context.Background(), &plugin.Request{
		Target: jsonschema.Name,
		Model:  b.Model,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return resp
}

// doc is a generated document, compiled against its metaschema.
type doc struct {
	t    *testing.T
	src  string
	defs string
	c    *sjs.Compiler
}

const loc = "file:///model.schema.json"

// check returns the response's one file, compiled.
func check(t *testing.T, resp *plugin.Response) *doc {
	t.Helper()
	if len(resp.GetFiles()) != 1 {
		t.Fatalf("files = %d, diagnostics = %+v", len(resp.GetFiles()), resp.GetDiagnostics())
	}
	return compile(t, resp.GetFiles()[0])
}

func compile(t *testing.T, f *plugin.File) *doc {
	t.Helper()
	src := string(f.GetContent())
	v, err := sjs.UnmarshalJSON(strings.NewReader(src))
	if err != nil {
		t.Fatalf("%s is not JSON: %v\n%s", f.GetPath(), err, src)
	}
	c := sjs.NewCompiler()
	c.AssertFormat()
	if err := c.AddResource(loc, v); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Compile(loc); err != nil {
		t.Fatalf("%s does not compile: %v\n%s", f.GetPath(), err, src)
	}
	defs := "$defs"
	if strings.Contains(src, `"definitions"`) {
		defs = "definitions"
	}
	return &doc{t: t, src: src, defs: defs, c: c}
}

// accepts asserts that a definition accepts each instance.
func (d *doc) accepts(def string, instances ...string) {
	d.t.Helper()
	s := d.schema(def)
	for _, in := range instances {
		if err := s.Validate(d.value(in)); err != nil {
			d.t.Errorf("%s rejects %s: %v", def, in, err)
		}
	}
}

// rejects asserts that a definition rejects each instance.
func (d *doc) rejects(def string, instances ...string) {
	d.t.Helper()
	s := d.schema(def)
	for _, in := range instances {
		if err := s.Validate(d.value(in)); err == nil {
			d.t.Errorf("%s accepts %s", def, in)
		}
	}
}

func (d *doc) schema(def string) *sjs.Schema {
	d.t.Helper()
	at := loc
	if def != "" {
		at += "#/" + d.defs + "/" + def
	}
	s, err := d.c.Compile(at)
	if err != nil {
		d.t.Fatalf("compile %s: %v\n%s", at, err, d.src)
	}
	return s
}

func (d *doc) value(in string) any {
	d.t.Helper()
	v, err := sjs.UnmarshalJSON(strings.NewReader(in))
	if err != nil {
		d.t.Fatalf("instance %s: %v", in, err)
	}
	return v
}

func (d *doc) contains(wants ...string) {
	d.t.Helper()
	flat := collapse(d.src)
	for _, want := range wants {
		if !strings.Contains(flat, collapse(want)) {
			d.t.Errorf("output missing %q:\n%s", want, d.src)
		}
	}
}

func (d *doc) absent(unwanted ...string) {
	d.t.Helper()
	for _, u := range unwanted {
		if strings.Contains(d.src, u) {
			d.t.Errorf("output has %q:\n%s", u, d.src)
		}
	}
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

func diagnostics(t *testing.T, resp *plugin.Response, n int) {
	t.Helper()
	if len(resp.GetDiagnostics()) != n {
		t.Errorf("diagnostics = %d, want %d: %+v", len(resp.GetDiagnostics()), n, resp.GetDiagnostics())
	}
}

func value(name string, fields ...*ir.Field) *ir.Decl {
	return &ir.Decl{
		Meta: &ir.Meta{Name: name},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{Kind: ir.StructKind_STRUCT_KIND_VALUE, Fields: fields}},
	}
}

func enum(name string, variants ...*ir.Variant) *ir.Decl {
	return &ir.Decl{
		Meta: &ir.Meta{Name: name},
		Node: &ir.Decl_Enumeration{Enumeration: &ir.Enum{Variants: variants}},
	}
}

func variant(name string, fields ...*ir.Field) *ir.Variant {
	return &ir.Variant{Meta: &ir.Meta{Name: name}, Fields: fields}
}

func newtype(name string, base *ir.ID, cs ...*ir.Constraint) *ir.Decl {
	return &ir.Decl{Meta: &ir.Meta{Name: name}, Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: base, ValueConstraints: cs}}}
}

func directive(name string, args ...string) *ir.Directive {
	d := &ir.Directive{Name: name, Target: jsonschema.Name}
	for _, a := range args {
		d.Args = append(d.Args, irtest.Text(a))
	}
	return d
}

func block(b *irtest.Builder, ds ...*ir.Directive) {
	b.Model.Targets = []*ir.TargetBlock{{Meta: &ir.Meta{Name: jsonschema.Name}, Directives: ds}}
}

func constraint(name string, args ...*ir.Literal) *ir.Constraint {
	return &ir.Constraint{Name: name, Args: args}
}

func integer(text string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: text}
}

func span(low, high *int64) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_RANGE, Range: &ir.Range{Low: low, High: high}}
}

func regex(p string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_REGEX, Text: p}
}

func ptr(n int64) *int64 { return &n }

func TestDescribe(t *testing.T) {
	d := jsonschema.Backend{}.Describe()
	if d.Name != "jsonschema" || !d.Reuse {
		t.Errorf("description = %+v", d)
	}
	declared := map[string]bool{}
	for _, spec := range d.Directives {
		declared[spec.GetName()] = true
	}
	for _, want := range []string{"name", "discriminant", "draft", "root", "id", "closed"} {
		if !declared[want] {
			t.Errorf("directive %q is acted on but not declared", want)
		}
	}
}

func TestObjects(t *testing.T) {
	b := irtest.New("shop.billing")
	b.Own(value("LineItem", irtest.Field("sku", b.Named("string")), irtest.Field("quantity", b.Named("int"))))
	b.Own(value("Order",
		irtest.Field("id", b.Named("uuid")),
		irtest.Field("items", b.Named("List", b.Named("LineItem"))),
		irtest.Field("tags", b.Named("Set", b.Named("string"))),
		irtest.Field("totals", b.Named("Map", b.Named("string"), b.Named("decimal"))),
		irtest.Field("note", b.Named("Option", b.Named("string"))),
		irtest.Field("shipOn", b.Named("Nullable", b.Named("date"))),
		irtest.Field("either", b.Named("Nullable", b.Named("Option", b.Named("string")))),
		irtest.Field("gaps", b.Named("List", b.Named("Option", b.Named("int")))),
		irtest.Field("paid", b.Named("bool")),
		irtest.Field("content-type", b.Named("string")),
	))

	resp := generate(t, b)
	diagnostics(t, resp, 0)
	if path := resp.GetFiles()[0].GetPath(); path != "billing.schema.json" {
		t.Errorf("path = %q", path)
	}
	d := check(t, resp)
	d.contains(
		`"$schema": "https://json-schema.org/draft/2020-12/schema"`,
		`"$comment": "Code generated by tdl. DO NOT EDIT."`,
		`"required": [ "id", "items", "tags", "totals", "shipOn", "gaps", "paid", "content-type" ]`,
	)

	full := `{"id": "0b0e6f1c-6c1a-4d8a-9a3e-2f1d7b6c5a4e", "items": [{"sku": "a", "quantity": 1}],
		"tags": ["x"], "totals": {"usd": "1.50"}, "shipOn": null, "gaps": [1, null], "paid": true,
		"content-type": "text/plain", "extra": 1}`
	d.accepts("Order", full,
		`{"id": "0b0e6f1c-6c1a-4d8a-9a3e-2f1d7b6c5a4e", "items": [], "tags": [], "totals": {}, "shipOn": "2026-10-05", "gaps": [], "paid": false, "content-type": "", "either": null}`,
	)
	d.rejects("Order",
		`{}`,
		strings.Replace(full, `"shipOn": null, `, "", 1),
		strings.Replace(full, `"tags": ["x"]`, `"tags": ["x", "x"]`, 1),
		strings.Replace(full, `"0b0e6f1c-6c1a-4d8a-9a3e-2f1d7b6c5a4e"`, `"not a uuid"`, 1),
		strings.Replace(full, `"quantity": 1`, `"quantity": 1.5`, 1),
		strings.Replace(full, `"paid": true`, `"paid": true, "note": null`, 1),
	)
}

func TestScalars(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Sizes",
		irtest.Field("i32", b.Named("int32")),
		irtest.Field("u32", b.Named("uint32")),
		irtest.Field("u64", b.Named("uint64")),
		irtest.Field("f", b.Named("float32")),
		irtest.Field("blob", b.Named("bytes")),
		irtest.Field("at", b.Named("instant")),
		irtest.Field("took", b.Named("duration")),
	))
	d := check(t, generate(t, b))
	d.contains(
		`"i32": { "type": "integer", "minimum": -2147483648, "maximum": 2147483647 }`,
		`"u32": { "type": "integer", "minimum": 0, "maximum": 4294967295 }`,
		`"u64": { "type": "integer", "minimum": 0 }`,
		`"f": { "type": "number" }`,
		`"blob": { "type": "string", "contentEncoding": "base64" }`,
		`"at": { "type": "string", "format": "date-time" }`,
		`"took": { "type": "string", "format": "duration" }`,
	)
	ok := `{"i32": -1, "u32": 4294967295, "u64": 1, "f": 0.5, "blob": "AA==", "at": "2026-10-05T12:00:00Z", "took": "PT1H"}`
	d.accepts("Sizes", ok)
	d.rejects("Sizes",
		strings.Replace(ok, `"u32": 4294967295`, `"u32": 4294967296`, 1),
		strings.Replace(ok, `"u64": 1`, `"u64": -1`, 1),
		strings.Replace(ok, `"at": "2026-10-05T12:00:00Z"`, `"at": "noon"`, 1),
	)
}

func TestEnums(t *testing.T) {
	b := irtest.New("shop")
	b.Own(enum("Status", variant("Active"), variant("InProgress")))
	b.Own(enum("Payment", variant("Card", irtest.Field("last4", b.Named("string"))), variant("Cash")))
	b.Own(value("Ticket", irtest.Field("status", b.Named("Status")), irtest.Field("payment", b.Named("Payment"))))

	d := check(t, generate(t, b))
	d.contains(
		`"Status": { "type": "string", "enum": [ "Active", "InProgress" ] }`,
		`"status": { "$ref": "#/$defs/Status" }`,
	)
	d.accepts("Ticket",
		`{"status": "Active", "payment": {"kind": "Card", "last4": "4242"}}`,
		`{"status": "InProgress", "payment": {"kind": "Cash"}}`,
	)
	d.rejects("Ticket",
		`{"status": "Done", "payment": {"kind": "Cash"}}`,
		`{"status": "Active", "payment": {"kind": "Card"}}`,
		`{"status": "Active", "payment": {"kind": "Cheque"}}`,
		`{"status": "Active", "payment": {"last4": "4242"}}`,
	)
}

// The discriminant defaults to `kind`, the target block overrides it, and
// an enum's directive overrides the block's.
func TestDiscriminant(t *testing.T) {
	b := irtest.New("shop")
	b.Own(enum("Payment", variant("Card", irtest.Field("last4", b.Named("string"))), variant("Cash")))
	shape := enum("Shape", variant("Circle", irtest.Field("radius", b.Named("int"))), variant("Point"))
	shape.Directives = []*ir.Directive{directive("discriminant", "$type")}
	b.Own(shape)
	block(b, directive("discriminant", "type"))

	d := check(t, generate(t, b))
	d.accepts("Payment", `{"type": "Cash"}`)
	d.accepts("Shape", `{"$type": "Circle", "radius": 1}`)
	d.rejects("Payment", `{"kind": "Cash"}`)
}

func TestAFieldTakingTheDiscriminantIsSkipped(t *testing.T) {
	b := irtest.New("shop")
	b.Own(enum("Broken", variant("A", irtest.Field("kind", b.Named("string"))), variant("B")))
	b.Own(value("Fine", irtest.Field("a", b.Named("string"))))

	resp := generate(t, b)
	diagnostics(t, resp, 1)
	d := check(t, resp)
	d.absent("Broken")
	d.contains(`"Fine"`)
}

func TestNewtypes(t *testing.T) {
	b := irtest.New("shop")
	b.Own(newtype("Email", b.Named("string"), constraint("length", span(ptr(12), ptr(254)))))
	b.Own(newtype("WorkEmail", b.Named("Email"),
		constraint("matches", regex(`@acme\.com$`)),
		&ir.Constraint{Name: "length", Args: []*ir.Literal{span(ptr(12), ptr(254))}, From: &ir.ID{Name: "Email"}},
	))
	b.Own(newtype("Skus", b.Named("List", b.Named("WorkEmail"))))
	b.Own(value("Contact", irtest.Field("email", b.Named("WorkEmail")), irtest.Field("all", b.Named("Skus"))))

	resp := generate(t, b)
	diagnostics(t, resp, 0)
	d := check(t, resp)
	d.contains(
		`"Email": { "type": "string", "minLength": 12, "maxLength": 254 }`,
		`"WorkEmail": { "$ref": "#/$defs/Email", "pattern": "@acme\\.com$" }`,
		`"Skus": { "type": "array", "items": { "$ref": "#/$defs/WorkEmail" } }`,
	)
	d.accepts("Contact", `{"email": "abc@acme.com", "all": ["xyz@acme.com"]}`)
	d.rejects("Contact", `{"email": "abc@other.com", "all": []}`, `{"email": "a@acme.com", "all": []}`, `{"email": "abc@acme.com", "all": ["x@acme.com"]}`)
}

func TestConstraints(t *testing.T) {
	b := irtest.New("shop")
	b.Own(enum("Status", variant("Draft"), variant("Placed"), variant("Gone")))
	qty := irtest.Field("quantity", b.Named("int"))
	qty.Constraints = []*ir.Constraint{constraint("min", integer("1")), constraint("max", integer("100"))}
	items := irtest.Field("items", b.Named("List", b.Named("string")))
	items.Constraints = []*ir.Constraint{constraint("length", span(ptr(1), nil)), constraint("unique")}
	region := irtest.Field("region", b.Named("Option", b.Named("string")))
	region.Constraints = []*ir.Constraint{constraint("oneOf", irtest.Text("us-east"), irtest.Text("eu-west"))}
	code := irtest.Field("code", b.Named("Nullable", b.Named("string")))
	code.Constraints = []*ir.Constraint{constraint("length", integer("2"))}
	status := irtest.Field("status", b.Named("Status"))
	status.Constraints = []*ir.Constraint{constraint("oneOf", irtest.Name("Draft"), irtest.Name("Placed"))}
	b.Own(value("Order", qty, items, region, code, status))

	resp := generate(t, b)
	diagnostics(t, resp, 0)
	d := check(t, resp)
	ok := `{"quantity": 1, "items": ["a"], "region": "us-east", "code": "us", "status": "Draft"}`
	d.accepts("Order", ok, strings.Replace(ok, `"code": "us"`, `"code": null`, 1))
	d.rejects("Order",
		strings.Replace(ok, `"quantity": 1`, `"quantity": 0`, 1),
		strings.Replace(ok, `"quantity": 1`, `"quantity": 101`, 1),
		strings.Replace(ok, `["a"]`, `[]`, 1),
		strings.Replace(ok, `["a"]`, `["a", "a"]`, 1),
		strings.Replace(ok, `"us-east"`, `"ap-south"`, 1),
		strings.Replace(ok, `"code": "us"`, `"code": "usa"`, 1),
		strings.Replace(ok, `"Draft"`, `"Gone"`, 1),
	)
}

// A constraint JSON Schema has no keyword for, or one on a type it does
// not apply to, is a warning, and the field is still written.
func TestConstraintsWithoutAKeywordWarn(t *testing.T) {
	b := irtest.New("shop")
	ratio := irtest.Field("ratio", b.Named("int"))
	ratio.Constraints = []*ir.Constraint{constraint("between", integer("0"), integer("100"))}
	name := irtest.Field("name", b.Named("string"))
	name.Constraints = []*ir.Constraint{constraint("min", integer("1"))}
	b.Own(value("Line", ratio, name))

	resp := generate(t, b)
	diagnostics(t, resp, 2)
	check(t, resp).accepts("Line", `{"ratio": 500, "name": ""}`)
}

func TestMapKeys(t *testing.T) {
	b := irtest.New("shop")
	b.Own(enum("Status", variant("Open"), variant("Closed")))
	b.Own(newtype("Sku", b.Named("string"), constraint("matches", regex(`^[A-Z]+$`))))
	b.Own(value("Counts",
		irtest.Field("byStatus", b.Named("Map", b.Named("Status"), b.Named("int"))),
		irtest.Field("bySku", b.Named("Map", b.Named("Sku"), b.Named("int"))),
		irtest.Field("byNumber", b.Named("Map", b.Named("int"), b.Named("string"))),
	))
	b.Own(value("Broken", irtest.Field("byFlag", b.Named("Map", b.Named("bool"), b.Named("string")))))

	resp := generate(t, b)
	diagnostics(t, resp, 1)
	d := check(t, resp)
	d.absent("Broken")
	ok := `{"byStatus": {"Open": 1}, "bySku": {"AB": 2}, "byNumber": {"-12": "x"}}`
	d.accepts("Counts", ok)
	d.rejects("Counts",
		strings.Replace(ok, `"Open"`, `"Ajar"`, 1),
		strings.Replace(ok, `"AB"`, `"ab"`, 1),
		strings.Replace(ok, `"-12"`, `"012"`, 1),
		strings.Replace(ok, `"x"`, `1`, 1),
	)
}

func TestNames(t *testing.T) {
	b := irtest.New("shop")
	renamed := value("Order", irtest.Field("id", b.Named("string")))
	renamed.Directives = []*ir.Directive{directive("name", "PurchaseOrder")}
	b.Own(renamed)
	field := irtest.Field("createdAt", b.Named("string"))
	field.Directives = []*ir.Directive{directive("name", "created_at")}
	b.Own(value("Stamp", field))
	broken := value("Broken", irtest.Field("a", b.Named("string")))
	broken.Directives = []*ir.Directive{directive("name", "has space")}
	b.Own(broken)
	clash := value("Clash", irtest.Field("a", b.Named("string")))
	clash.Directives = []*ir.Directive{directive("name", "Stamp")}
	b.Own(clash)

	resp := generate(t, b)
	diagnostics(t, resp, 2)
	d := check(t, resp)
	d.accepts("PurchaseOrder", `{"id": "1"}`)
	d.accepts("Stamp", `{"created_at": "x"}`)
	d.rejects("Stamp", `{"createdAt": "x"}`)
	d.absent("has space")
}

func TestClosed(t *testing.T) {
	b := irtest.New("shop")
	closed := value("Strict", irtest.Field("a", b.Named("string")))
	closed.Directives = []*ir.Directive{{Name: "closed", Target: jsonschema.Name}}
	b.Own(closed)
	b.Own(value("Loose", irtest.Field("a", b.Named("string"))))

	d := check(t, generate(t, b))
	d.rejects("Strict", `{"a": "x", "b": 1}`)
	d.accepts("Loose", `{"a": "x", "b": 1}`)

	block(b, &ir.Directive{Name: "closed", Target: jsonschema.Name})
	d = check(t, generate(t, b))
	d.rejects("Loose", `{"a": "x", "b": 1}`)
}

func TestRootAndID(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Order", irtest.Field("id", b.Named("string"))))
	block(b, directive("root", "Order"), directive("id", "https://example.com/shop.schema.json"))

	d := check(t, generate(t, b))
	d.contains(`"$id": "https://example.com/shop.schema.json"`, `"$ref": "#/$defs/Order"`)
	d.accepts("", `{"id": "1"}`)
	d.rejects("", `{}`)
}

func TestRootMustNameAGeneratedDeclaration(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Order", irtest.Field("id", b.Named("string"))))
	block(b, directive("root", "Missing"))

	resp := generate(t, b)
	if len(resp.GetFiles()) != 0 || len(resp.GetDiagnostics()) != 1 ||
		resp.GetDiagnostics()[0].GetSeverity() != plugin.Severity_SEVERITY_ERROR {
		t.Errorf("response = %+v", resp)
	}
}

func TestDraft07(t *testing.T) {
	b := irtest.New("shop")
	b.Own(newtype("Email", b.Named("string")))
	b.Own(newtype("WorkEmail", b.Named("Email"), constraint("matches", regex(`@acme\.com$`))))
	b.Own(&ir.Decl{
		Meta: &ir.Meta{Name: "Contact", Doc: []string{"How to reach someone."}, Deprecated: &ir.Deprecation{Reason: "use Person"}},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{Fields: []*ir.Field{irtest.Field("email", b.Named("WorkEmail"))}}},
	})
	block(b, directive("draft", "draft-07"), directive("root", "Contact"))

	d := check(t, generate(t, b))
	d.contains(
		`"$schema": "http://json-schema.org/draft-07/schema#"`,
		`"definitions": {`,
		`"WorkEmail": { "pattern": "@acme\\.com$", "allOf": [ { "$ref": "#/definitions/Email" } ] }`,
		`"description": "How to reach someone.\n\nDeprecated. use Person"`,
	)
	d.absent(`"$defs"`, `"deprecated"`)
	d.accepts("", `{"email": "a@acme.com"}`)
	d.rejects("", `{"email": "a@other.com"}`)
}

func TestAnUnknownDraftIsAnError(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Order", irtest.Field("id", b.Named("string"))))
	block(b, directive("draft", "draft-04"))

	resp := generate(t, b)
	if len(resp.GetFiles()) != 0 || len(resp.GetDiagnostics()) != 1 ||
		resp.GetDiagnostics()[0].GetSeverity() != plugin.Severity_SEVERITY_ERROR {
		t.Errorf("response = %+v", resp)
	}
}

func TestDocsAndDeprecation(t *testing.T) {
	b := irtest.New("shop")
	old := irtest.Field("fax", b.Named("string"))
	old.Meta.Deprecated = &ir.Deprecation{Reason: "nobody has one"}
	b.Own(&ir.Decl{
		Meta: &ir.Meta{Name: "Contact", Doc: []string{"How to reach someone."}, Deprecated: &ir.Deprecation{}},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{Fields: []*ir.Field{old}}},
	})

	check(t, generate(t, b)).contains(
		`"Contact": { "description": "How to reach someone.", "deprecated": true, "type": "object"`,
		`"fax": { "deprecated": true, "type": "string" }`,
	)
}

func TestUnsupportedDeclarationsCascade(t *testing.T) {
	b := irtest.New("shop")
	b.Own(&ir.Decl{
		Meta: &ir.Meta{Name: "Box"},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{Params: irtest.Params("T"), Fields: []*ir.Field{irtest.Field("v", b.Param("T", 0))}}},
	})
	b.Own(value("Holder", irtest.Field("box", b.Named("Box"))))
	b.Own(value("Fine", irtest.Field("a", b.Named("string"))))

	resp := generate(t, b)
	d := check(t, resp)
	d.absent("Box", "Holder")
	d.contains(`"Fine"`)
}

// Every conformance case and the smoke fixture generate a schema that
// compiles.
func TestConformance(t *testing.T) {
	corpus := filepath.Join("..", "..", "testdata", "conformance")
	dirs := []string{filepath.Join("..", "..", "testdata", "gen", "smoke")}
	for _, c := range []string{
		"aliases", "collections", "comments", "constraints", "deprecated", "entity",
		"enum_variants", "mixin_include", "newtype", "targets", "targets_variants", "value",
	} {
		dirs = append(dirs, filepath.Join(corpus, c))
	}

	for _, dir := range dirs {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			model := irtest.Conformance(t, dir)
			resp, err := jsonschema.Backend{}.Generate(context.Background(), &plugin.Request{Target: jsonschema.Name, Model: model})
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range resp.GetDiagnostics() {
				if d.GetSeverity() == plugin.Severity_SEVERITY_ERROR {
					t.Errorf("error: %s", d.GetMessage())
				}
			}
			if len(resp.GetFiles()) > 0 {
				check(t, resp)
			}
		})
	}
}
