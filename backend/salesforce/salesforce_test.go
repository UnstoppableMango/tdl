package salesforce_test

import (
	"context"
	"encoding/xml"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/backend/salesforce"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

func generate(t *testing.T, b *irtest.Builder) *plugin.Response {
	t.Helper()
	resp, err := salesforce.Backend{}.Generate(context.Background(), &plugin.Request{
		Target: salesforce.Name,
		Model:  b.Model,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return resp
}

// files returns the response's files by path, each XML one checked to be
// well formed.
func files(t *testing.T, resp *plugin.Response) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, f := range resp.GetFiles() {
		if strings.HasSuffix(f.GetPath(), ".xml") {
			wellFormed(t, f)
		}
		out[f.GetPath()] = string(f.GetContent())
	}
	return out
}

func wellFormed(t *testing.T, f *plugin.File) {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader(string(f.GetContent())))
	for {
		_, err := d.Token()
		if err != nil {
			if err.Error() != "EOF" {
				t.Errorf("%s is not well formed: %v\n%s", f.GetPath(), err, f.GetContent())
			}
			return
		}
	}
}

func file(t *testing.T, all map[string]string, path string) string {
	t.Helper()
	src, ok := all[path]
	if !ok {
		paths := make([]string, 0, len(all))
		for p := range all {
			paths = append(paths, p)
		}
		slices.Sort(paths)
		t.Fatalf("no file %s among %v", path, paths)
	}
	return src
}

func contains(t *testing.T, src string, wants ...string) {
	t.Helper()
	flat := collapse(src)
	for _, want := range wants {
		if !strings.Contains(flat, collapse(want)) {
			t.Errorf("output missing %q:\n%s", want, src)
		}
	}
}

func absent(t *testing.T, src string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(src, u) {
			t.Errorf("output has %q:\n%s", u, src)
		}
	}
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

func warned(t *testing.T, resp *plugin.Response, substr string) {
	t.Helper()
	for _, d := range resp.GetDiagnostics() {
		if strings.Contains(d.GetMessage(), substr) {
			return
		}
	}
	t.Errorf("no diagnostic mentions %q: %+v", substr, resp.GetDiagnostics())
}

func structure(kind ir.StructKind, name string, fields ...*ir.Field) *ir.Decl {
	return &ir.Decl{
		Meta: &ir.Meta{Name: name},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{Kind: kind, Fields: fields}},
	}
}

func entity(name string, fields ...*ir.Field) *ir.Decl {
	return structure(ir.StructKind_STRUCT_KIND_ENTITY, name, fields...)
}

func value(name string, fields ...*ir.Field) *ir.Decl {
	return structure(ir.StructKind_STRUCT_KIND_VALUE, name, fields...)
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

func directive(name string, args ...*ir.Literal) *ir.Directive {
	return &ir.Directive{Name: name, Target: salesforce.Name, Args: args}
}

func TestDescribe(t *testing.T) {
	d := salesforce.Backend{}.Describe()
	if d.Name != "salesforce" || !d.Reuse {
		t.Errorf("description = %+v", d)
	}
	declared := map[string]bool{}
	for _, spec := range d.Directives {
		declared[spec.GetName()] = true
	}
	for _, want := range []string{"name", "label", "plural", "key", "length", "scale", "prefix", "apiVersion"} {
		if !declared[want] {
			t.Errorf("directive %q is acted on but not declared", want)
		}
	}
}

func TestObject(t *testing.T) {
	b := irtest.New("shop")
	b.Own(enum("Status", variant("Draft"), variant("Placed")))
	b.Own(entity("Customer", irtest.Field("email", b.Named("string"))))
	b.Own(entity("Order",
		irtest.Field("total", b.Named("decimal")),
		irtest.Field("count", b.Named("int")),
		irtest.Field("paid", b.Named("bool")),
		irtest.Field("placedAt", b.Named("instant")),
		irtest.Field("shipOn", b.Named("Option", b.Named("date"))),
		irtest.Field("status", b.Named("Status")),
		irtest.Field("customer", b.Named("Customer")),
	))

	all := files(t, generate(t, b))
	contains(t, file(t, all, "objects/Order__c/Order__c.object-meta.xml"),
		`<CustomObject xmlns="http://soap.sforce.com/2006/04/metadata">`,
		"<deploymentStatus>Deployed</deploymentStatus>",
		"<label>Order</label>",
		"<displayFormat>Order-{0000}</displayFormat>",
		"<type>AutoNumber</type>",
		"<pluralLabel>Orders</pluralLabel>",
		"<sharingModel>ReadWrite</sharingModel>",
	)
	contains(t, file(t, all, "objects/Order__c/fields/Total__c.field-meta.xml"),
		"<fullName>Total__c</fullName>", "<precision>18</precision>", "<required>true</required>",
		"<scale>6</scale>", "<type>Number</type>")
	contains(t, file(t, all, "objects/Order__c/fields/Count__c.field-meta.xml"), "<scale>0</scale>")
	paid := file(t, all, "objects/Order__c/fields/Paid__c.field-meta.xml")
	contains(t, paid, "<defaultValue>false</defaultValue>", "<type>Checkbox</type>")
	absent(t, paid, "<required>")
	contains(t, file(t, all, "objects/Order__c/fields/PlacedAt__c.field-meta.xml"),
		"<label>Placed At</label>", "<type>DateTime</type>")
	absent(t, file(t, all, "objects/Order__c/fields/ShipOn__c.field-meta.xml"), "<required>")
	contains(t, file(t, all, "objects/Order__c/fields/Status__c.field-meta.xml"),
		"<type>Picklist</type>", "<restricted>true</restricted>",
		"<value> <fullName>Draft</fullName> <default>false</default> <label>Draft</label> </value>")
	contains(t, file(t, all, "objects/Order__c/fields/Customer__c.field-meta.xml"),
		"<deleteConstraint>Restrict</deleteConstraint>", "<referenceTo>Customer__c</referenceTo>",
		"<relationshipName>OrderCustomer</relationshipName>", "<type>Lookup</type>")

	// An entity is an SObject in Apex, so it has no class of its own.
	for p := range all {
		if strings.HasPrefix(p, "classes/Order") || strings.HasPrefix(p, "classes/Customer") {
			t.Errorf("entity generated %s", p)
		}
	}
}

// A field no column holds is dropped from the object, which is still
// written.
func TestUnstorableField(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Address", irtest.Field("line", b.Named("string"))))
	b.Own(entity("Order",
		irtest.Field("id", b.Named("string")),
		irtest.Field("address", b.Named("Address")),
		irtest.Field("receipt", b.Named("bytes")),
		irtest.Field("tags", b.Named("List", b.Named("string"))),
	))

	resp := generate(t, b)
	all := files(t, resp)
	file(t, all, "objects/Order__c/Order__c.object-meta.xml")
	file(t, all, "objects/Order__c/fields/Id__c.field-meta.xml")
	for _, p := range []string{"Address__c", "Receipt__c", "Tags__c"} {
		if _, ok := all["objects/Order__c/fields/"+p+".field-meta.xml"]; ok {
			t.Errorf("%s has a field file", p)
		}
	}
	warned(t, resp, "Order.address has no Salesforce field type, and the object is written without it")
	warned(t, resp, "Order.receipt is bytes")
	warned(t, resp, "Order.tags has no Salesforce field type")
}

func TestMultiselect(t *testing.T) {
	b := irtest.New("shop")
	b.Own(enum("Tag", variant("Gift"), variant("Rush")))
	b.Own(entity("Order", irtest.Field("tags", b.Named("Set", b.Named("Tag")))))

	src := file(t, files(t, generate(t, b)), "objects/Order__c/fields/Tags__c.field-meta.xml")
	contains(t, src, "<type>MultiselectPicklist</type>", "<visibleLines>4</visibleLines>", "<fullName>Rush</fullName>")
	absent(t, src, "<required>")
}

func TestKey(t *testing.T) {
	b := irtest.New("shop")
	order := entity("Order", irtest.Field("id", b.Named("uuid")), irtest.Field("paid", b.Named("bool")))
	order.Directives = []*ir.Directive{directive("key", irtest.Name("id"))}
	b.Own(order)

	all := files(t, generate(t, b))
	contains(t, file(t, all, "objects/Order__c/fields/Id__c.field-meta.xml"),
		"<externalId>true</externalId>", "<length>36</length>", "<unique>true</unique>")
	absent(t, file(t, all, "objects/Order__c/fields/Paid__c.field-meta.xml"), "externalId")

	b = irtest.New("shop")
	line := entity("Line", irtest.Field("order", b.Named("string")), irtest.Field("sku", b.Named("string")))
	line.Directives = []*ir.Directive{directive("key", irtest.Name("order"), irtest.Name("sku"))}
	b.Own(line)
	resp := generate(t, b)
	absent(t, file(t, files(t, resp), "objects/Line__c/fields/Order__c.field-meta.xml"), "externalId")
	warned(t, resp, "a Salesforce external ID is one field")
}

func TestFieldDirectives(t *testing.T) {
	b := irtest.New("shop")
	code := irtest.Field("code", b.Named("string"))
	code.Directives = []*ir.Directive{
		directive("length", &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: "12"}),
		directive("label", irtest.Text("SKU Code")),
		directive("name", irtest.Text("Sku")),
	}
	price := irtest.Field("price", b.Named("decimal"))
	price.Directives = []*ir.Directive{directive("scale", &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_INT, Text: "2"})}
	product := entity("Product", code, price)
	product.Directives = []*ir.Directive{directive("label", irtest.Text("Item")), directive("plural", irtest.Text("Items"))}
	b.Own(product)

	all := files(t, generate(t, b))
	contains(t, file(t, all, "objects/Product__c/Product__c.object-meta.xml"),
		"<label>Item</label>", "<pluralLabel>Items</pluralLabel>")
	contains(t, file(t, all, "objects/Product__c/fields/Sku__c.field-meta.xml"),
		"<label>SKU Code</label>", "<length>12</length>")
	contains(t, file(t, all, "objects/Product__c/fields/Price__c.field-meta.xml"), "<scale>2</scale>")
}

func TestApexClass(t *testing.T) {
	b := irtest.New("shop")
	b.Own(entity("Customer", irtest.Field("email", b.Named("string"))))
	b.Own(value("LineItem",
		irtest.Field("sku", b.Named("string")),
		irtest.Field("quantity", b.Named("int")),
		irtest.Field("note", b.Named("Option", b.Named("string"))),
		irtest.Field("tags", b.Named("Set", b.Named("string"))),
		irtest.Field("prices", b.Named("Map", b.Named("string"), b.Named("decimal"))),
		irtest.Field("buyer", b.Named("Customer")),
	))

	all := files(t, generate(t, b))
	contains(t, file(t, all, "classes/LineItem.cls"),
		"// Code generated by tdl. DO NOT EDIT.",
		"public class LineItem {",
		"public String sku;",
		"public Long quantity;",
		"public String note;",
		"public Set<String> tags;",
		"public Map<String, Decimal> prices;",
		"public Customer__c buyer;",
	)
	contains(t, file(t, all, "classes/LineItem.cls-meta.xml"),
		`<ApexClass xmlns="http://soap.sforce.com/2006/04/metadata">`,
		"<apiVersion>66.0</apiVersion>", "<status>Active</status>")
}

func TestApexEnums(t *testing.T) {
	b := irtest.New("shop")
	b.Own(enum("Status", variant("Draft"), variant("Placed")))
	b.Own(enum("Payment", variant("Card", irtest.Field("last4", b.Named("string"))), variant("Cash")))

	all := files(t, generate(t, b))
	contains(t, file(t, all, "classes/Status.cls"), "public enum Status { Draft, Placed }")
	contains(t, file(t, all, "classes/Payment.cls"),
		"public class Payment {",
		"public enum Kind { Card, Cash }",
		"public Kind kind;",
		"public Card card;",
		"public class Card { public String last4; }",
	)
}

func TestApexNames(t *testing.T) {
	b := irtest.New("shop")
	b.Own(value("Account", irtest.Field("id", b.Named("string"))))
	b.Own(value("Note", irtest.Field("type", b.Named("string"))))
	resp := generate(t, b)
	warned(t, resp, "shadows the standard type")
	warned(t, resp, "which is a reserved word")
	if len(resp.GetFiles()) != 0 {
		t.Errorf("files = %v", resp.GetFiles())
	}

	b = irtest.New("shop")
	b.Own(value("Account", irtest.Field("id", b.Named("string"))))
	b.Model.Targets = []*ir.TargetBlock{{
		Meta:       &ir.Meta{Name: salesforce.Name},
		Directives: []*ir.Directive{directive("prefix", irtest.Text("Shop")), directive("apiVersion", irtest.Text("65.0"))},
	}}
	all := files(t, generate(t, b))
	contains(t, file(t, all, "classes/ShopAccount.cls"), "public class ShopAccount {")
	contains(t, file(t, all, "classes/ShopAccount.cls-meta.xml"), "<apiVersion>65.0</apiVersion>")
}

func TestNewtypeExpands(t *testing.T) {
	b := irtest.New("shop")
	b.Own(&ir.Decl{Meta: &ir.Meta{Name: "Email"}, Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: b.Named("string")}}})
	b.Own(entity("Customer", irtest.Field("email", b.Named("Email"))))
	b.Own(value("Signup", irtest.Field("email", b.Named("Email"))))

	all := files(t, generate(t, b))
	contains(t, file(t, all, "objects/Customer__c/fields/Email__c.field-meta.xml"), "<type>Text</type>")
	contains(t, file(t, all, "classes/Signup.cls"), "public String email;")
	if _, ok := all["classes/Email.cls"]; ok {
		t.Error("newtype generated a class")
	}
}

func TestDocsAndDeprecation(t *testing.T) {
	b := irtest.New("shop")
	old := irtest.Field("fax", b.Named("string"))
	old.Meta.Deprecated = &ir.Deprecation{Reason: "nobody has one"}
	b.Own(&ir.Decl{
		Meta: &ir.Meta{Name: "Contact", Doc: []string{"How to reach someone."}},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{Kind: ir.StructKind_STRUCT_KIND_ENTITY, Fields: []*ir.Field{old}}},
	})
	b.Own(&ir.Decl{
		Meta: &ir.Meta{Name: "Card", Doc: []string{"A payment card."}, Deprecated: &ir.Deprecation{}},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{Fields: []*ir.Field{irtest.Field("last4", b.Named("string"))}}},
	})

	all := files(t, generate(t, b))
	contains(t, file(t, all, "objects/Contact__c/Contact__c.object-meta.xml"), "<description>How to reach someone.</description>")
	contains(t, file(t, all, "objects/Contact__c/fields/Fax__c.field-meta.xml"), "<description>Deprecated. nobody has one</description>")
	card := file(t, all, "classes/Card.cls")
	contains(t, card, "/** * A payment card. * * @deprecated */ public class Card {")
	absent(t, card, "@Deprecated")
}

// Every case in the conformance corpus, and the smoke fixture, generates
// without an error and writes only well-formed XML.
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
			resp, err := salesforce.Backend{}.Generate(context.Background(), &plugin.Request{Target: salesforce.Name, Model: model})
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range resp.GetDiagnostics() {
				if d.GetSeverity() == plugin.Severity_SEVERITY_ERROR {
					t.Errorf("error: %s", d.GetMessage())
				}
			}
			files(t, resp)
		})
	}
}
