package openapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	sjs "github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"

	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/backend/openapi"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

var versions = []string{"3.1", "3.0", "2.0"}

// metaschemas maps each version to the schema an OpenAPI document of that
// version is validated against, and the files it needs loaded. They are
// the OpenAPI Initiative's published schemas, from
// github.com/OAI/spec.openapis.org under the Apache License 2.0.
var metaschemas = map[string]struct {
	url   string
	files map[string]string
}{
	"2.0": {"http://swagger.io/v2/schema.json", map[string]string{
		"http://swagger.io/v2/schema.json": "oas-2.0.json",
	}},
	"3.0": {"https://spec.openapis.org/oas/3.0/schema/2024-10-18", map[string]string{
		"https://spec.openapis.org/oas/3.0/schema/2024-10-18": "oas-3.0.json",
	}},
	// The base schema also holds every Schema Object to the OpenAPI
	// dialect, rather than accepting any JSON.
	"3.1": {"https://spec.openapis.org/oas/3.1/schema-base/2026-08-03", map[string]string{
		"https://spec.openapis.org/oas/3.1/schema-base/2026-08-03": "oas-3.1-base.json",
		"https://spec.openapis.org/oas/3.1/schema/2026-08-03":      "oas-3.1.json",
		"https://spec.openapis.org/oas/3.1/dialect/2024-11-10":     "oas-3.1-dialect.json",
		"https://spec.openapis.org/oas/3.1/meta/2024-11-10":        "oas-3.1-meta.json",
	}},
}

func metaschema(t *testing.T, version string) *sjs.Schema {
	t.Helper()
	m := metaschemas[version]
	c := sjs.NewCompiler()
	for url, file := range m.files {
		f, err := os.Open(filepath.Join("testdata", file))
		if err != nil {
			t.Fatal(err)
		}
		v, err := sjs.UnmarshalJSON(f)
		_ = f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if err := c.AddResource(url, v); err != nil {
			t.Fatal(err)
		}
	}
	s, err := c.Compile(m.url)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func directive(name string, args ...string) *ir.Directive {
	d := &ir.Directive{Name: name, Target: openapi.Name}
	for _, a := range args {
		d.Args = append(d.Args, irtest.Text(a))
	}
	return d
}

func block(b *irtest.Builder, ds ...*ir.Directive) {
	b.Model.Targets = []*ir.TargetBlock{{Meta: &ir.Meta{Name: openapi.Name}, Directives: ds}}
}

func generate(t *testing.T, model *ir.Model) *plugin.Response {
	t.Helper()
	resp, err := openapi.Backend{}.Generate(context.Background(), &plugin.Request{Target: openapi.Name, Model: model})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return resp
}

// doc is a generated document, decoded and validated against the schema
// for its version.
type doc struct {
	t    *testing.T
	src  string
	json string
	v    any
}

// check returns the response's one file, validated against the schema for
// version.
func check(t *testing.T, resp *plugin.Response, version string) *doc {
	t.Helper()
	if len(resp.GetFiles()) != 1 {
		t.Fatalf("files = %d, diagnostics = %+v", len(resp.GetFiles()), resp.GetDiagnostics())
	}
	f := resp.GetFiles()[0]
	src := string(f.GetContent())
	var raw []byte
	if strings.HasSuffix(f.GetPath(), ".yaml") {
		var v any
		if err := yaml.Unmarshal(f.GetContent(), &v); err != nil {
			t.Fatalf("%s is not YAML: %v\n%s", f.GetPath(), err, src)
		}
		var err error
		if raw, err = json.MarshalIndent(v, "", "  "); err != nil {
			t.Fatal(err)
		}
	} else {
		raw = f.GetContent()
	}
	v, err := sjs.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("%s is not JSON: %v\n%s", f.GetPath(), err, src)
	}
	if err := metaschema(t, version).Validate(v); err != nil {
		t.Fatalf("%s is not an OpenAPI %s document: %v\n%s", f.GetPath(), version, err, src)
	}
	return &doc{t: t, src: src, json: string(raw), v: v}
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

// schema compiles one of a 3.1 document's components, which is JSON
// Schema 2020-12.
func (d *doc) schema(name string) *sjs.Schema {
	d.t.Helper()
	const loc = "file:///openapi.json"
	c := sjs.NewCompiler()
	c.AssertFormat()
	if err := c.AddResource(loc, d.v); err != nil {
		d.t.Fatal(err)
	}
	s, err := c.Compile(loc + "#/components/schemas/" + name)
	if err != nil {
		d.t.Fatalf("compile %s: %v\n%s", name, err, d.src)
	}
	return s
}

func (d *doc) accepts(name string, instances ...string) {
	d.t.Helper()
	s := d.schema(name)
	for _, in := range instances {
		if err := s.Validate(instance(d.t, in)); err != nil {
			d.t.Errorf("%s rejects %s: %v", name, in, err)
		}
	}
}

func (d *doc) rejects(name string, instances ...string) {
	d.t.Helper()
	s := d.schema(name)
	for _, in := range instances {
		if err := s.Validate(instance(d.t, in)); err == nil {
			d.t.Errorf("%s accepts %s", name, in)
		}
	}
}

func instance(t *testing.T, in string) any {
	t.Helper()
	v, err := sjs.UnmarshalJSON(strings.NewReader(in))
	if err != nil {
		t.Fatalf("instance %s: %v", in, err)
	}
	return v
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

func diagnostics(t *testing.T, resp *plugin.Response, n int) {
	t.Helper()
	if len(resp.GetDiagnostics()) != n {
		t.Errorf("diagnostics = %d, want %d: %+v", len(resp.GetDiagnostics()), n, resp.GetDiagnostics())
	}
}

func isError(resp *plugin.Response) bool {
	return len(resp.GetFiles()) == 0 && len(resp.GetDiagnostics()) == 1 &&
		resp.GetDiagnostics()[0].GetSeverity() == plugin.Severity_SEVERITY_ERROR
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

func newtype(name string, base *ir.ID) *ir.Decl {
	return &ir.Decl{Meta: &ir.Meta{Name: name}, Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: base}}}
}

// shop is a model reaching every shape the mapping distinguishes, with the
// target block's directives given.
func shop(ds ...*ir.Directive) *ir.Model {
	b := irtest.New("shop.billing")
	b.Own(newtype("Email", b.Named("string")))
	b.Own(enum("Status", variant("Open"), variant("Closed")))
	b.Own(enum("Payment", variant("Card", irtest.Field("last4", b.Named("string"))), variant("Cash")))
	b.Own(value("LineItem", irtest.Field("sku", b.Named("string")), irtest.Field("quantity", b.Named("int32"))))
	fax := irtest.Field("fax", b.Named("string"))
	fax.Meta.Deprecated = &ir.Deprecation{Reason: "nobody has one"}
	contact := irtest.Field("contact", b.Named("Nullable", b.Named("Email")))
	contact.Meta.Doc = []string{"Where to write."}
	b.Own(&ir.Decl{
		Meta: &ir.Meta{Name: "Order", Doc: []string{"An order."}},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{Fields: []*ir.Field{
			irtest.Field("id", b.Named("uuid")),
			irtest.Field("items", b.Named("List", b.Named("LineItem"))),
			irtest.Field("tags", b.Named("Set", b.Named("string"))),
			irtest.Field("byStatus", b.Named("Map", b.Named("Status"), b.Named("int"))),
			irtest.Field("note", b.Named("Option", b.Named("string"))),
			irtest.Field("shipOn", b.Named("Nullable", b.Named("date"))),
			irtest.Field("gaps", b.Named("List", b.Named("Option", b.Named("float64")))),
			irtest.Field("receipt", b.Named("bytes")),
			irtest.Field("status", b.Named("Status")),
			contact,
			fax,
		}}},
	})
	b.Own(value("Checkout", irtest.Field("order", b.Named("Order")), irtest.Field("payment", b.Named("Payment"))))
	block(b, ds...)
	return b.Model
}

func TestDescribe(t *testing.T) {
	d := openapi.Backend{}.Describe()
	if d.Name != "openapi" || !d.Reuse {
		t.Errorf("description = %+v", d)
	}
	declared := map[string]bool{}
	for _, spec := range d.Directives {
		declared[spec.GetName()] = true
	}
	for _, want := range []string{"name", "discriminant", "closed", "openapi", "format", "title", "version"} {
		if !declared[want] {
			t.Errorf("directive %q is acted on but not declared", want)
		}
	}
}

// Every version, in both formats, is a document its version's schema
// accepts.
func TestEveryVersionValidates(t *testing.T) {
	for _, v := range versions {
		for _, format := range []string{"yaml", "json"} {
			t.Run(v+"/"+format, func(t *testing.T) {
				check(t, generate(t, shop(directive("openapi", v), directive("format", format))), v)
			})
		}
	}
}

// A document reads the same in either format.
func TestYAMLAndJSONAgree(t *testing.T) {
	for _, v := range versions {
		y := check(t, generate(t, shop(directive("openapi", v))), v)
		j := check(t, generate(t, shop(directive("openapi", v), directive("format", "json"))), v)
		jv := j.v.(map[string]any)
		if jv["x-generated"] != "Code generated by tdl. DO NOT EDIT." {
			t.Errorf("%s: x-generated = %v", v, jv["x-generated"])
		}
		delete(jv, "x-generated")
		if !reflect.DeepEqual(y.v, j.v) {
			t.Errorf("%s: YAML and JSON differ\nYAML:\n%s\nJSON:\n%s", v, y.src, j.src)
		}
	}
}

func TestDefaults(t *testing.T) {
	resp := generate(t, shop())
	diagnostics(t, resp, 0)
	if path := resp.GetFiles()[0].GetPath(); path != "billing.openapi.yaml" {
		t.Errorf("path = %q", path)
	}
	d := check(t, resp, "3.1")
	if !strings.HasPrefix(d.src, "# Code generated by tdl. DO NOT EDIT.\nopenapi: \"3.1.1\"\n") {
		t.Errorf("header:\n%s", d.src)
	}
	d.contains(`info: title: shop.billing version: "0.0.0"`, `paths: {}`)

	b := irtest.New("")
	b.Own(value("Thing", irtest.Field("a", b.Named("string"))))
	if path := generate(t, b.Model).GetFiles()[0].GetPath(); path != "model.openapi.yaml" {
		t.Errorf("path without a package = %q", path)
	}
}

func TestInfo(t *testing.T) {
	d := check(t, generate(t, shop(directive("title", "Billing API"), directive("version", "1.2.0"))), "3.1")
	d.contains(`title: "Billing API" version: "1.2.0"`)
}

func TestUnknownDirectiveValuesAreErrors(t *testing.T) {
	for _, d := range []*ir.Directive{directive("openapi", "3.2"), directive("format", "toml")} {
		if resp := generate(t, shop(d)); !isError(resp) {
			t.Errorf("%s(%s): response = %+v", d.GetName(), d.GetArgs()[0].GetText(), resp)
		}
	}
}

func TestOpenAPI31(t *testing.T) {
	resp := generate(t, shop(directive("format", "json")))
	diagnostics(t, resp, 0)
	d := check(t, resp, "3.1")
	d.contains(
		`"components": { "schemas": {`,
		`"quantity": { "type": "integer", "format": "int32" }`,
		`"receipt": { "type": "string", "contentEncoding": "base64" }`,
		`"contact": { "description": "Where to write.", "anyOf": [ { "$ref": "#/components/schemas/Email" }, { "type": "null" } ] }`,
		`"propertyNames": { "$ref": "#/components/schemas/Status" }`,
		`"fax": { "deprecated": true, "type": "string" }`,
		`"Payment": { "oneOf": [ { "$ref": "#/components/schemas/PaymentCard" }, { "$ref": "#/components/schemas/PaymentCash" } ],
			"discriminator": { "propertyName": "kind", "mapping": {
				"Card": "#/components/schemas/PaymentCard", "Cash": "#/components/schemas/PaymentCash" } } }`,
		`"PaymentCard": { "type": "object", "properties": { "kind": { "const": "Card" }`,
	)
	d.absent("nullable", "allOf")

	full := `{"id": "0b0e6f1c-6c1a-4d8a-9a3e-2f1d7b6c5a4e", "items": [{"sku": "a", "quantity": 1}], "tags": ["x"],
		"byStatus": {"Open": 1}, "shipOn": null, "gaps": [1.5, null], "receipt": "AA==", "status": "Open",
		"contact": null, "fax": ""}`
	d.accepts("Order", full)
	d.rejects("Order",
		strings.Replace(full, `"Open": 1`, `"Ajar": 1`, 1),
		strings.Replace(full, `"quantity": 1`, `"quantity": 1.5`, 1),
		strings.Replace(full, `"shipOn": null, `, "", 1),
	)
	d.accepts("Payment", `{"kind": "Card", "last4": "4242"}`, `{"kind": "Cash"}`)
	d.rejects("Payment", `{"kind": "Card"}`, `{"kind": "Cheque"}`)
}

func TestOpenAPI30(t *testing.T) {
	resp := generate(t, shop(directive("openapi", "3.0"), directive("format", "json")))
	diagnostics(t, resp, 0)
	d := check(t, resp, "3.0")
	d.contains(
		`"openapi": "3.0.4"`,
		`"receipt": { "type": "string", "format": "byte" }`,
		`"shipOn": { "type": "string", "format": "date", "nullable": true }`,
		`"contact": { "description": "Where to write.", "nullable": true, "allOf": [ { "$ref": "#/components/schemas/Email" } ] }`,
		`"items": { "type": "number", "format": "double", "nullable": true }`,
		`"fax": { "deprecated": true, "type": "string" }`,
		`"kind": { "type": "string", "enum": [ "Card" ] }`,
		`"discriminator": { "propertyName": "kind"`,
	)
	d.absent("propertyNames", "const", "anyOf", "contentEncoding")
}

// OpenAPI 2.0 has no oneOf, so a fielded enum is a warning, and so is
// every declaration naming it.
func TestOpenAPI20(t *testing.T) {
	resp := generate(t, shop(directive("openapi", "2.0"), directive("format", "json")))
	diagnostics(t, resp, 2)
	d := check(t, resp, "2.0")
	d.contains(
		`"swagger": "2.0"`,
		`"definitions": {`,
		`"$ref": "#/definitions/LineItem"`,
		`"shipOn": { "type": "string", "format": "date", "x-nullable": true }`,
		`"fax": { "description": "Deprecated. nobody has one", "type": "string" }`,
	)
	d.absent("components", "Payment", "Checkout", "oneOf", `"deprecated"`, `"nullable"`)
}

func TestVariantNames(t *testing.T) {
	b := irtest.New("shop")
	card := variant("Card", irtest.Field("last4", b.Named("string")))
	card.Directives = []*ir.Directive{directive("name", "CardPayment")}
	b.Own(enum("Payment", card, variant("Cash")))
	// PaymentCash is taken, so Shape's variant cannot take it too.
	clash := variant("Cash", irtest.Field("a", b.Named("string")))
	clash.Directives = []*ir.Directive{directive("name", "PaymentCash")}
	b.Own(enum("Shape", clash))
	b.Own(value("Fine", irtest.Field("a", b.Named("string"))))

	resp := generate(t, b.Model)
	diagnostics(t, resp, 1)
	d := check(t, resp, "3.1")
	d.contains(`$ref: "#/components/schemas/CardPayment"`, `Card: "#/components/schemas/CardPayment"`)
	d.absent("Shape")
}

func TestDiscriminantAndClosed(t *testing.T) {
	b := irtest.New("shop")
	b.Own(enum("Payment", variant("Card", irtest.Field("last4", b.Named("string"))), variant("Cash")))
	block(b, directive("discriminant", "type"), &ir.Directive{Name: "closed", Target: openapi.Name})

	d := check(t, generate(t, b.Model), "3.1")
	d.contains(`propertyName: type`)
	d.accepts("Payment", `{"type": "Cash"}`)
	d.rejects("Payment", `{"type": "Cash", "extra": 1}`, `{"kind": "Cash"}`)
}

func TestScalars(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Sizes",
		irtest.Field("i", b.Named("int")),
		irtest.Field("u32", b.Named("uint32")),
		irtest.Field("u64", b.Named("uint64")),
		irtest.Field("f", b.Named("float32")),
		irtest.Field("d", b.Named("decimal")),
	))
	check(t, generate(t, b.Model), "3.1").contains(
		`i: type: integer format: int64`,
		`u32: type: integer format: int64 minimum: 0 maximum: 4294967295`,
		`u64: type: integer minimum: 0`,
		`f: type: number format: float`,
		`d: type: string`,
	)
}

// YAML reads a plain scalar as something other than a string when it
// looks like one, so those are quoted.
func TestYAMLQuotesAmbiguousStrings(t *testing.T) {
	b := irtest.New("shop")
	b.Own(enum("Answer", variant("yes"), variant("No"), variant("null"), variant("Off"), variant("Maybe")))
	b.Own(value("Odd", irtest.Field("true", b.Named("string")), irtest.Field("has space", b.Named("string"))))
	d := check(t, generate(t, b.Model), "3.1")
	d.contains(`- "yes" - "No" - "null" - "Off" - Maybe`, `"true": type: string`, `"has space":`)
	ans := d.v.(map[string]any)["components"].(map[string]any)["schemas"].(map[string]any)["Answer"].(map[string]any)["enum"]
	if !reflect.DeepEqual(ans, []any{"yes", "No", "null", "Off", "Maybe"}) {
		t.Errorf("enum = %#v", ans)
	}
}

// Every conformance case and the smoke fixture generate a document each
// version accepts.
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
		for _, v := range versions {
			t.Run(filepath.Base(dir)+"/"+v, func(t *testing.T) {
				model := irtest.Conformance(t, dir)
				model.Targets = append(model.Targets, &ir.TargetBlock{
					Meta:       &ir.Meta{Name: openapi.Name},
					Directives: []*ir.Directive{directive("openapi", v)},
				})
				resp := generate(t, model)
				for _, d := range resp.GetDiagnostics() {
					if d.GetSeverity() == plugin.Severity_SEVERITY_ERROR {
						t.Errorf("error: %s", d.GetMessage())
					}
				}
				if len(resp.GetFiles()) > 0 {
					check(t, resp, v)
				}
			})
		}
	}
}
