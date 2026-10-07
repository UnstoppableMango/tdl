package jsonschema

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

const target = "test"

// dialects mirrors what the jsonschema and openapi backends pass, so each
// branch a dialect selects is reached here, where coverage is counted.
var dialects = map[string]Dialect{
	"2020-12": {
		Name: "JSON Schema", Ref: "#/$defs/", Siblings: true, Deprecated: true,
		Const: true, ContentEncoding: true, PropertyNames: true,
	},
	"draft-07": {
		Name: "JSON Schema", Ref: "#/definitions/",
		Const: true, ContentEncoding: true, PropertyNames: true,
	},
	"3.1": {
		Name: "OpenAPI 3.1", Ref: "#/components/schemas/", Siblings: true, Deprecated: true, Const: true,
		ContentEncoding: true, PropertyNames: true, Unions: UnionsReferenced, Formats: true,
	},
	"3.0": {
		Name: "OpenAPI 3.0", Ref: "#/components/schemas/", Deprecated: true, Null: NullNullable,
		Unions: UnionsReferenced, Formats: true,
	},
	"2.0": {
		Name: "OpenAPI 2.0", Ref: "#/definitions/", Null: NullExtension,
		Unions: UnionsNone, Formats: true,
	},
}

// built is the output of one Build, rendered.
type built struct {
	t     *testing.T
	src   string
	defs  *Object
	diags []*plugin.Diagnostic
}

func build(t *testing.T, model *ir.Model, d Dialect) *built {
	t.Helper()
	s := emit.NewSession(&plugin.Request{Target: target, Model: model}, d.Name)
	b := New(s, d)
	defs, _ := b.Build(s.Own())
	if !d.Siblings {
		Hoist(defs)
	}
	var buf bytes.Buffer
	WriteJSON(&buf, defs)
	if !json.Valid(buf.Bytes()) {
		t.Fatalf("not JSON:\n%s", buf.String())
	}
	return &built{t: t, src: buf.String(), defs: defs, diags: s.Diags}
}

func (b *built) contains(wants ...string) {
	b.t.Helper()
	got := collapse(b.src)
	for _, w := range wants {
		if !strings.Contains(got, collapse(w)) {
			b.t.Errorf("missing %s in:\n%s", w, b.src)
		}
	}
}

func (b *built) absent(unwanted ...string) {
	b.t.Helper()
	got := collapse(b.src)
	for _, w := range unwanted {
		if strings.Contains(got, collapse(w)) {
			b.t.Errorf("unexpected %s in:\n%s", w, b.src)
		}
	}
}

// warns asserts one warning containing each of wants.
func (b *built) warns(wants ...string) {
	b.t.Helper()
	if len(b.diags) != 1 {
		b.t.Fatalf("diagnostics = %+v, want 1", b.diags)
	}
	for _, w := range wants {
		if !strings.Contains(b.diags[0].GetMessage(), w) {
			b.t.Errorf("warning %q does not contain %q", b.diags[0].GetMessage(), w)
		}
	}
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

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
	d := &ir.Directive{Name: name, Target: target}
	for _, a := range args {
		d.Args = append(d.Args, irtest.Text(a))
	}
	return d
}

func block(b *irtest.Builder, ds ...*ir.Directive) {
	b.Model.Targets = []*ir.TargetBlock{{Meta: &ir.Meta{Name: target}, Directives: ds}}
}

func constraint(name string, args ...*ir.Literal) *ir.Constraint {
	return &ir.Constraint{Name: name, Args: args}
}

func integer(text string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: text}
}

// shop reaches every shape a dialect writes differently.
func shop() *irtest.Builder {
	b := irtest.New("shop")
	b.Own(newtype("Email", b.Named("string")))
	b.Own(enum("Status", variant("Open"), variant("Closed")))
	b.Own(enum("Payment", variant("Card", irtest.Field("last4", b.Named("string"))), variant("Cash")))
	fax := irtest.Field("fax", b.Named("string"))
	fax.Meta.Deprecated = &ir.Deprecation{Reason: "nobody has one"}
	fax.Meta.Doc = []string{"A fax number."}
	b.Own(value("Order",
		irtest.Field("count", b.Named("int32")),
		irtest.Field("small", b.Named("uint32")),
		irtest.Field("big", b.Named("uint64")),
		irtest.Field("receipt", b.Named("bytes")),
		irtest.Field("shipOn", b.Named("Nullable", b.Named("date"))),
		irtest.Field("gaps", b.Named("List", b.Named("Option", b.Named("float64")))),
		irtest.Field("tags", b.Named("Set", b.Named("string"))),
		irtest.Field("byStatus", b.Named("Map", b.Named("Status"), b.Named("int"))),
		irtest.Field("byCount", b.Named("Map", b.Named("int"), b.Named("string"))),
		irtest.Field("byEmail", b.Named("Map", b.Named("Email"), b.Named("bool"))),
		irtest.Field("byName", b.Named("Map", b.Named("string"), b.Named("bool"))),
		irtest.Field("email", b.Named("Email")),
		irtest.Field("payment", b.Named("Payment")),
		fax,
	))
	return b
}

func TestJSONSchema202012(t *testing.T) {
	d := build(t, shop().Model, dialects["2020-12"])
	if len(d.diags) != 0 {
		t.Fatalf("diagnostics = %+v", d.diags)
	}
	d.contains(
		`"count": { "type": "integer", "minimum": -2147483648, "maximum": 2147483647 }`,
		`"big": { "type": "integer", "minimum": 0 }`,
		`"receipt": { "type": "string", "contentEncoding": "base64" }`,
		`"shipOn": { "anyOf": [ { "type": "string", "format": "date" }, { "type": "null" } ] }`,
		`"items": { "anyOf": [ { "type": "number" }, { "type": "null" } ] }`,
		`"uniqueItems": true`,
		`"propertyNames": { "$ref": "#/$defs/Status" }`,
		`"propertyNames": { "pattern": "^-?(0|[1-9][0-9]*)$" }`,
		`"propertyNames": { "$ref": "#/$defs/Email" }`,
		`"title": "Card"`,
		`"kind": { "const": "Card" }`,
		`"description": "A fax number.", "deprecated": true`,
	)
	d.absent(`"format": "int32"`, `"discriminator"`)
}

func TestDraft07HoistsRefs(t *testing.T) {
	b := shop()
	email := irtest.Field("primary", b.Named("Email"))
	email.Meta.Doc = []string{"Where to write."}
	b.Own(value("Contact", email))
	d := build(t, b.Model, dialects["draft-07"])
	d.contains(
		`"primary": { "description": "Where to write.", "allOf": [ { "$ref": "#/definitions/Email" } ] }`,
		`"description": "A fax number.\n\nDeprecated. nobody has one"`,
	)
	d.absent(`"deprecated": true`)
}

func TestOpenAPI31(t *testing.T) {
	d := build(t, shop().Model, dialects["3.1"])
	d.contains(
		`"count": { "type": "integer", "format": "int32" }`,
		`"small": { "type": "integer", "format": "int64", "minimum": 0, "maximum": 4294967295 }`,
		`"PaymentCard": {`,
		`"oneOf": [ { "$ref": "#/components/schemas/PaymentCard" }, { "$ref": "#/components/schemas/PaymentCash" } ]`,
		`"discriminator": { "propertyName": "kind", "mapping": { "Card": "#/components/schemas/PaymentCard", "Cash": "#/components/schemas/PaymentCash" } }`,
	)
	d.absent(`"title": "Card"`)
}

func TestOpenAPI30(t *testing.T) {
	d := build(t, shop().Model, dialects["3.0"])
	d.contains(
		`"shipOn": { "type": "string", "format": "date", "nullable": true }`,
		`"receipt": { "type": "string", "format": "byte" }`,
		`"kind": { "type": "string", "enum": [ "Card" ] }`,
	)
	d.absent(`"const"`, `"propertyNames"`, `"anyOf"`)
}

func TestOpenAPI20(t *testing.T) {
	d := build(t, shop().Model, dialects["2.0"])
	if len(d.diags) != 2 ||
		!strings.Contains(d.diags[0].GetMessage(), "OpenAPI 2.0 has no oneOf") ||
		!strings.Contains(d.diags[1].GetMessage(), "Order names Payment") {
		t.Errorf("diagnostics = %+v", d.diags)
	}
	d.absent(`"Payment"`, `"Order"`)

	b := irtest.New("shop")
	b.Own(value("Box", irtest.Field("v", b.Named("Nullable", b.Named("string")))))
	build(t, b.Model, dialects["2.0"]).contains(`"v": { "type": "string", "x-nullable": true }`)
}

func TestNullIsNotDoubled(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Box", irtest.Field("v", b.Named("Nullable", b.Named("List", b.Named("Nullable", b.Named("Nullable", b.Named("string"))))))))
	d := build(t, b.Model, dialects["2020-12"])
	if strings.Count(d.src, `"type": "null"`) != 2 {
		t.Errorf("want one null per level:\n%s", d.src)
	}
}

func TestDirectives(t *testing.T) {
	b := irtest.New("shop")
	card := variant("Card", irtest.Field("last4", b.Named("string")))
	card.Directives = []*ir.Directive{directive("name", "ByCard")}
	pay := enum("Payment", card, variant("Cash"))
	pay.Directives = []*ir.Directive{directive("discriminant", "type")}
	b.Own(pay)
	b.Own(value("Open", irtest.Field("a", b.Named("string"))))
	block(b, directive("discriminant", "tag"), directive("closed"))

	d := build(t, b.Model, dialects["3.1"])
	d.contains(
		`"ByCard": {`,
		`"type": { "const": "Card" }`,
		`"propertyName": "type"`,
		`"additionalProperties": false`,
	)
}

func TestFieldDocAndConstraints(t *testing.T) {
	b := irtest.New("shop")
	f := irtest.Field("n", b.Named("int"))
	f.Constraints = []*ir.Constraint{constraint("min", integer("1"))}
	b.Own(value("Box", f))
	b.Own(newtype("Positive", b.Named("int"), constraint("min", integer("1"))))
	b.Own(newtype("Small", b.Named("Positive"), constraint("max", integer("9"))))
	d := build(t, b.Model, dialects["2020-12"])
	d.contains(`"n": { "type": "integer", "minimum": 1 }`, `"Small": { "$ref": "#/$defs/Positive", "maximum": 9 }`)
}

func TestUnsupported(t *testing.T) {
	for _, c := range []struct {
		name string
		decl func(b *irtest.Builder) *ir.Decl
		want string
	}{
		{"class", func(b *irtest.Builder) *ir.Decl {
			return &ir.Decl{Meta: &ir.Meta{Name: "Shape"}, Node: &ir.Decl_Class{Class: &ir.Class{}}}
		}, "classes are not generated"},
		{"unit", func(b *irtest.Builder) *ir.Decl {
			return &ir.Decl{Meta: &ir.Meta{Name: "kg"}, Node: &ir.Decl_Unit{Unit: &ir.UnitDef{}}}
		}, "units are not generated"},
		{"generic", func(b *irtest.Builder) *ir.Decl {
			return &ir.Decl{
				Meta: &ir.Meta{Name: "Box"},
				Node: &ir.Decl_Structure{Structure: &ir.Struct{Params: irtest.Params("T"), Fields: []*ir.Field{irtest.Field("v", b.Param("T", 0))}}},
			}
		}, "generics are not generated"},
		{"bad name", func(b *irtest.Builder) *ir.Decl {
			d := value("Box", irtest.Field("a", b.Named("string")))
			d.Directives = []*ir.Directive{directive("name", "a b")}
			return d
		}, "is not a JSON Schema definition name"},
		{"duplicate property", func(b *irtest.Builder) *ir.Decl {
			return value("Box", irtest.Field("a", b.Named("string")), irtest.Field("a", b.Named("int")))
		}, "two properties named a"},
		{"discriminant property", func(b *irtest.Builder) *ir.Decl {
			return enum("Pay", variant("Card", irtest.Field("kind", b.Named("string"))))
		}, "takes the name of the discriminant"},
		{"unknown primitive", func(b *irtest.Builder) *ir.Decl {
			b.Own(&ir.Decl{Meta: &ir.Meta{Name: "blob"}, Node: &ir.Decl_Primitive{Primitive: &ir.Primitive{}}})
			return value("Box", irtest.Field("a", b.Named("blob")))
		}, "primitive blob has no JSON Schema type"},
		{"bad map key", func(b *irtest.Builder) *ir.Decl {
			return value("Box", irtest.Field("a", b.Named("Map", b.Named("bool"), b.Named("string"))))
		}, "a JSON object key is"},
		{"bad element", func(b *irtest.Builder) *ir.Decl {
			b.Own(&ir.Decl{Meta: &ir.Meta{Name: "blob"}, Node: &ir.Decl_Primitive{Primitive: &ir.Primitive{}}})
			return value("Box", irtest.Field("a", b.Named("List", b.Named("Option", b.Named("blob")))))
		}, "primitive blob"},
		{"bad map value", func(b *irtest.Builder) *ir.Decl {
			b.Own(&ir.Decl{Meta: &ir.Meta{Name: "blob"}, Node: &ir.Decl_Primitive{Primitive: &ir.Primitive{}}})
			return value("Box", irtest.Field("a", b.Named("Map", b.Named("string"), b.Named("blob"))))
		}, "primitive blob"},
		{"bad newtype base", func(b *irtest.Builder) *ir.Decl {
			b.Own(&ir.Decl{Meta: &ir.Meta{Name: "blob"}, Node: &ir.Decl_Primitive{Primitive: &ir.Primitive{}}})
			return newtype("Box", b.Named("blob"))
		}, "primitive blob"},
	} {
		t.Run(c.name, func(t *testing.T) {
			b := irtest.New("shop")
			b.Own(c.decl(b))
			d := build(t, b.Model, dialects["2020-12"])
			d.warns(c.want)
		})
	}
}

func TestNameCollisions(t *testing.T) {
	t.Run("declarations", func(t *testing.T) {
		b := irtest.New("shop")
		b.Own(value("Box", irtest.Field("a", b.Named("string"))))
		other := value("Crate", irtest.Field("a", b.Named("string")))
		other.Directives = []*ir.Directive{directive("name", "Box")}
		b.Own(other)
		build(t, b.Model, dialects["3.1"]).warns("Crate would be defined as Box, and Box already is")
	})
	t.Run("variant names a declaration", func(t *testing.T) {
		b := irtest.New("shop")
		b.Own(value("PayCard", irtest.Field("a", b.Named("string"))))
		b.Own(enum("Pay", variant("Card", irtest.Field("a", b.Named("string")))))
		build(t, b.Model, dialects["3.1"]).warns("Pay.Card would be defined as PayCard")
	})
	t.Run("two variants", func(t *testing.T) {
		b := irtest.New("shop")
		card := variant("Card", irtest.Field("a", b.Named("string")))
		card.Directives = []*ir.Directive{directive("name", "Pay")}
		b.Own(enum("Pay", card))
		build(t, b.Model, dialects["3.1"]).warns("Pay would define Pay twice")
	})
	t.Run("bad variant name", func(t *testing.T) {
		b := irtest.New("shop")
		card := variant("Card", irtest.Field("a", b.Named("string")))
		card.Directives = []*ir.Directive{directive("name", "a b")}
		b.Own(enum("Pay", card))
		build(t, b.Model, dialects["3.1"]).warns("is not a OpenAPI 3.1 definition name")
	})
}

func TestCascade(t *testing.T) {
	b := irtest.New("shop")
	b.Own(&ir.Decl{Meta: &ir.Meta{Name: "Shape"}, Node: &ir.Decl_Class{Class: &ir.Class{}}})
	b.Own(value("Fine", irtest.Field("a", b.Named("string"))))
	d := build(t, b.Model, dialects["2020-12"])
	d.contains(`"Fine"`)
	if d.defs.Len() != 1 || d.defs.Keys()[0] != "Fine" {
		t.Errorf("keys = %v", d.defs.Keys())
	}
}

// Every dialect builds every conformance case and the smoke model into
// valid JSON with no error.
func TestConformance(t *testing.T) {
	corpus := filepath.Join("..", "..", "..", "testdata", "conformance")
	dirs := []string{filepath.Join("..", "..", "..", "testdata", "gen", "smoke")}
	for _, c := range []string{
		"aliases", "collections", "comments", "constraints", "deprecated", "entity",
		"enum_variants", "mixin_include", "newtype", "targets", "targets_variants", "value",
	} {
		dirs = append(dirs, filepath.Join(corpus, c))
	}
	for _, dir := range dirs {
		model := irtest.Conformance(t, dir)
		for name, d := range dialects {
			t.Run(filepath.Base(dir)+"/"+name, func(t *testing.T) {
				for _, diag := range build(t, model, d).diags {
					if diag.GetSeverity() == plugin.Severity_SEVERITY_ERROR {
						t.Errorf("error: %s", diag.GetMessage())
					}
				}
			})
		}
	}
}

func TestWriteString(t *testing.T) {
	var buf bytes.Buffer
	WriteString(&buf, "a\"b\n<")
	var got string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil || got != "a\"b\n<" {
		t.Errorf("WriteString = %s (%v)", buf.String(), err)
	}
}
