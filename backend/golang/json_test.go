package golang_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/golang"
	"github.com/unstoppablemango/tdl/backend/jsonschema"
	"github.com/unstoppablemango/tdl/backend/openapi"
	"github.com/unstoppablemango/tdl/backend/typescript"
	"github.com/unstoppablemango/tdl/backend/zod"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// generateTDL lowers src and generates Go from it, failing on a lowering
// diagnostic.
func generateTDL(t *testing.T, src string) *plugin.Response {
	t.Helper()
	file, err := parser.Parse("main.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering: %v", diags.Error())
	}
	resp, err := golang.Backend{}.Generate(context.Background(), &plugin.Request{Target: golang.Name, Model: model})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return resp
}

// treeSource is a tree tagged by type, in the shape of mdast.
const treeSource = `package tree

type Root {
  children: [Node]
  data: Data?
  index: Map<string, Node>
  first: Node?
  ids: Names
}

type Data { title: string }

type Names: [Node]

enum Node {
  Text { value: string }
  Emphasis { children: [Node] }
  Link {
    url: string
    title: string?
  }
  Break
}

target go for tree {
  json
  discriminant("type")
}
`

func TestJSONTagsFields(t *testing.T) {
	resp := generateTDL(t, treeSource)
	if w := uncodedWarnings(resp); len(w) != 0 {
		t.Fatalf("diagnostics = %+v", w)
	}
	got := files(t, resp)
	contains(t, got["root.go"],
		"Children []Node `json:\"children\"`",
		"Data *Data `json:\"data,omitempty\"`",
		"First *Node `json:\"first,omitempty\"`",
		"func (r *Root) UnmarshalJSON(data []byte) error",
	)
	contains(t, got["node.go"],
		"Title *string `json:\"title,omitempty\"`",
		"func (n NodeText) MarshalJSON() ([]byte, error)",
		"func UnmarshalNode(data []byte) (Node, error)",
		"func (n *NodeEmphasis) UnmarshalJSON(data []byte) error",
	)
	if strings.Contains(got["data.go"], "UnmarshalJSON") {
		t.Errorf("Data decodes itself, and has an UnmarshalJSON:\n%s", got["data.go"])
	}
}

// uncodedWarnings is the warnings past those about what import would read
// back.
func uncodedWarnings(resp *plugin.Response) []*plugin.Diagnostic {
	var out []*plugin.Diagnostic
	for _, d := range resp.GetDiagnostics() {
		if !readsBackOtherwise(d) {
			out = append(out, d)
		}
	}
	return out
}

// The generated types read and write the JSON the TypeScript from the same
// model describes.
func TestJSONRoundTrips(t *testing.T) {
	resp := generateTDL(t, treeSource)
	goTest(t, resp, `package tree

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const doc = `+"`"+`{"children":[{"type":"Text","value":"a"},{"type":"Emphasis","children":[{"type":"Break"},{"type":"Link","url":"u"}]}],"data":{"title":"t"},"index":{"k":{"type":"Text","value":"b"}},"first":{"type":"Break"},"ids":[{"type":"Link","url":"v","title":"w"}]}`+"`"+`

func TestRoundTrip(t *testing.T) {
	var r Root
	if err := json.Unmarshal([]byte(doc), &r); err != nil {
		t.Fatal(err)
	}
	want := Root{
		Children: []Node{
			NodeText{Value: "a"},
			NodeEmphasis{Children: []Node{NodeBreak{}, NodeLink{Url: "u"}}},
		},
		Data:  &Data{Title: "t"},
		Index: map[string]Node{"k": NodeText{Value: "b"}},
		First: func() *Node { var n Node = NodeBreak{}; return &n }(),
		Ids:   Names{NodeLink{Url: "v", Title: func() *string { s := "w"; return &s }()}},
	}
	if !reflect.DeepEqual(r, want) {
		t.Fatalf("decoded %#v, want %#v", r, want)
	}
	out, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != doc {
		t.Fatalf("encoded\n%s\nwant\n%s", out, doc)
	}
}

func TestUnknownVariant(t *testing.T) {
	_, err := UnmarshalNode([]byte(`+"`"+`{"type":"Table"}`+"`"+`))
	if err == nil || !strings.Contains(err.Error(), "Table") {
		t.Fatalf("err = %v", err)
	}
	var r Root
	err = json.Unmarshal([]byte(`+"`"+`{"children":[{"value":"a"}]}`+"`"+`), &r)
	if err == nil || !strings.Contains(err.Error(), "children: [0]") {
		t.Fatalf("err = %v", err)
	}
}
`)
}

func TestJSONTagDirectiveWins(t *testing.T) {
	resp := generateTDL(t, `package tree

type Root { first: Node }

enum Node {
  Text { value: string }
}

target go for tree {
  json
  Root.first => tag("json:\"head\"")
}
`)
	got := files(t, resp)
	contains(t, got["root.go"], "First Node `json:\"head\"`", "First json.RawMessage `json:\"head\"`")
}

func TestJSONDiscriminantCollisionIsSkipped(t *testing.T) {
	resp := generateTDL(t, `package tree

enum Node {
  Text { kind: string }
}

target go for tree {
  json
}
`)
	w := uncodedWarnings(resp)
	if len(w) != 1 {
		t.Fatalf("diagnostics = %+v, want one warning", w)
	}
	contains(t, w[0].GetMessage(), "takes the name of the discriminant")
}

func TestJSONIsOptIn(t *testing.T) {
	resp := generateTDL(t, `package tree

type Root { first: Node }

enum Node {
  Text { value: string }
}
`)
	got := files(t, resp)
	for path, src := range got {
		if strings.Contains(src, "json") {
			t.Errorf("%s mentions json without the directive:\n%s", path, src)
		}
	}
}

func TestJSONGenerics(t *testing.T) {
	resp := generateTDL(t, `package tree

enum Result<T> {
  Ok { value: T }
  Err { message: string }
}

type Page<T> {
  items: [T]
  last: Result<string>
  results: Map<string, Result<T>>
}

target go for tree {
  json
}
`)
	if w := uncodedWarnings(resp); len(w) != 0 {
		t.Fatalf("diagnostics = %+v", w)
	}
	got := files(t, resp)
	contains(t, got["result.go"],
		"func (r ResultOk[T]) MarshalJSON() ([]byte, error)",
		"func UnmarshalResult[T any](data []byte) (Result[T], error)",
	)
	contains(t, got["page.go"],
		"func (p *Page[T]) UnmarshalJSON(data []byte) error",
		"UnmarshalResult[string](raw.Last)",
		"UnmarshalResult[T](raw)",
	)
}

// emptySource holds a collection at every depth the JSON wire convention
// writes as [] or {}, never null.
const emptySource = `package org

type Tags: [string]
type Groups: Map<string, [string]>
type Wrapped: Shape
type MaybeTags: [string]?

type ScratchOrg {
  flags: {string -> [string]}
  tags: [string]
  named: Tags
  maybe: [string]?
  nullable: [string] | null
  nested: [[string]]
  boxes: [Shape]
  groups: Groups
  wrapped: Wrapped
  maybeTags: MaybeTags
  shape: Shape
  renamed: [string]
}

enum Shape {
  Poly { points: [int32] }
  Dot
}

type Plain { name: string }

target go for org {
  json
  ScratchOrg.renamed => tag("json:\"other\"")
}
`

func TestJSONEmptyCollections(t *testing.T) {
	resp := generateTDL(t, emptySource)
	if w := uncodedWarnings(resp); len(w) != 0 {
		t.Fatalf("diagnostics = %+v", w)
	}
	got := files(t, resp)
	if strings.Contains(got["plain.go"], "MarshalJSON") {
		t.Errorf("Plain holds no collection, and has a MarshalJSON:\n%s", got["plain.go"])
	}
	goTest(t, resp, `package org

import (
	"encoding/json"
	"testing"
)

func TestZeroValue(t *testing.T) {
	out, err := json.Marshal(ScratchOrg{})
	if err != nil {
		t.Fatal(err)
	}
	const zero = `+"`"+`{"flags":{},"tags":[],"named":[],"nullable":null,"nested":[],"boxes":[],"groups":{},"wrapped":null,"maybeTags":null,"shape":null,"other":[]}`+"`"+`
	if string(out) != zero {
		t.Fatalf("encoded\n%s\nwant\n%s", out, zero)
	}

	org := ScratchOrg{
		Flags:  map[string][]string{"a": nil},
		Nested: [][]string{nil},
		Boxes:  []Shape{ShapePoly{}, ShapeDot{}},
		Groups: Groups{"g": nil},
		Shape:  ShapePoly{},
	}
	out, err = json.Marshal(org)
	if err != nil {
		t.Fatal(err)
	}
	const want = `+"`"+`{"flags":{"a":[]},"tags":[],"named":[],"nullable":null,"nested":[[]],"boxes":[{"kind":"Poly","points":[]},{"kind":"Dot"}],"groups":{"g":[]},"wrapped":null,"maybeTags":null,"shape":{"kind":"Poly","points":[]},"other":[]}`+"`"+`
	if string(out) != want {
		t.Fatalf("encoded\n%s\nwant\n%s", out, want)
	}
	if org.Flags["a"] != nil || org.Nested[0] != nil || org.Groups["g"] != nil {
		t.Fatal("marshalling changed the value marshalled")
	}

	var empty []string
	maybe := &empty
	org = ScratchOrg{Maybe: maybe, Nullable: maybe, MaybeTags: MaybeTags(maybe)}
	var wrapped Shape = ShapePoly{}
	org.Wrapped = Wrapped(wrapped)
	out, err = json.Marshal(org)
	if err != nil {
		t.Fatal(err)
	}
	const some = `+"`"+`{"flags":{},"tags":[],"named":[],"maybe":[],"nullable":[],"nested":[],"boxes":[],"groups":{},"wrapped":{"kind":"Poly","points":[]},"maybeTags":[],"shape":null,"other":[]}`+"`"+`
	if string(out) != some {
		t.Fatalf("encoded\n%s\nwant\n%s", out, some)
	}
	if *maybe != nil {
		t.Fatal("marshalling changed the value marshalled")
	}
}

func TestNullDecodes(t *testing.T) {
	var org ScratchOrg
	doc := `+"`"+`{"flags":null,"tags":null,"named":null,"maybe":null,"nullable":null,"nested":[null],"boxes":null,"groups":{"g":null},"maybeTags":null,"shape":{"kind":"Poly","points":null}}`+"`"+`
	if err := json.Unmarshal([]byte(doc), &org); err != nil {
		t.Fatal(err)
	}
	if org.Tags != nil || org.Named != nil || org.Maybe != nil || len(org.Nested) != 1 || org.Groups["g"] != nil {
		t.Fatalf("decoded %#v", org)
	}
	if p, ok := org.Shape.(ShapePoly); !ok || p.Points != nil {
		t.Fatalf("decoded shape %#v", org.Shape)
	}
}
`)
}

func TestJSONGenericOverCollectionWarns(t *testing.T) {
	resp := generateTDL(t, `package tree

type Box<T> { value: T }
type Wrap<T>: [T]

type Root {
  box: Box<[string]>
  boxes: [Box<{string -> int32}>]
  plain: Box<string>
  wrap: Wrap<[string]>
}

target go for tree {
  json
}
`)
	w := uncodedWarnings(resp)
	if len(w) != 3 {
		t.Fatalf("diagnostics = %+v, want three warnings", w)
	}
	for _, d := range w {
		contains(t, d.GetMessage(), "so an empty one there is null")
	}
	got := files(t, resp)
	contains(t, got["wrap.go"], "func (w Wrap[T]) MarshalJSON() ([]byte, error)", "self := []T(w)")
}

// Import reads a MarshalJSON filling collections as this backend's own, and
// warns about it no more than about the other JSON methods.
func TestImportSkipsFillingMarshal(t *testing.T) {
	resp := generateTDL(t, emptySource)
	_, diags := importGo(t, files(t, resp))
	for _, d := range diags {
		msg := d.GetMessage()
		if strings.Contains(msg, "MarshalJSON") && !strings.Contains(msg, "UnmarshalJSON") && !strings.Contains(msg, "ShapeDot") {
			t.Errorf("import warns about a filling MarshalJSON: %s", msg)
		}
	}
}

func TestJSONMarshalCollisionIsSkipped(t *testing.T) {
	resp := generateTDL(t, `package tree

type Root {
  marshalJSON: string
  tags: [string]
}

target go for tree {
  json
}
`)
	w := uncodedWarnings(resp)
	if len(w) != 1 {
		t.Fatalf("diagnostics = %+v, want one warning", w)
	}
	contains(t, w[0].GetMessage(), "has a field named MarshalJSON")
}

// wireSource renames a fieldless and a fielded variant's wire values.
const wireSource = `package shop

enum Status {
  Live
  Expired
}

enum Payment {
  Card { last4: string }
  Cash
}

type Order {
  status: Status
  payment: Payment
}

target go for shop {
  json
  Status.Live => wire("live")
  Payment.Card => wire("card")
}
`

func TestWireRenamesVariants(t *testing.T) {
	resp := generateTDL(t, wireSource)
	if w := uncodedWarnings(resp); len(w) != 0 {
		t.Fatalf("diagnostics = %+v", w)
	}
	got := files(t, resp)
	contains(t, got["status.go"], `StatusLive    Status = "live"`, `StatusExpired Status = "Expired"`)
	goTest(t, resp, `package shop

import (
	"encoding/json"
	"reflect"
	"testing"
)

const doc = `+"`"+`{"status":"live","payment":{"kind":"card","last4":"4242"}}`+"`"+`

func TestRoundTrip(t *testing.T) {
	var o Order
	if err := json.Unmarshal([]byte(doc), &o); err != nil {
		t.Fatal(err)
	}
	want := Order{Status: StatusLive, Payment: PaymentCard{Last4: "4242"}}
	if !reflect.DeepEqual(o, want) {
		t.Fatalf("decoded %#v, want %#v", o, want)
	}
	out, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != doc {
		t.Fatalf("encoded\n%s\nwant\n%s", out, doc)
	}
}
`)
}

func TestWireWithoutJSONWarns(t *testing.T) {
	resp := generateTDL(t, strings.Replace(wireSource, "  json\n", "", 1))
	w := uncodedWarnings(resp)
	if len(w) != 1 {
		t.Fatalf("diagnostics = %+v, want one warning", w)
	}
	contains(t, w[0].GetMessage(), "Payment.Card", "has no effect")
	contains(t, files(t, resp)["status.go"], `= "live"`)
}

func TestWireCollisionIsAnError(t *testing.T) {
	resp := generateTDL(t, strings.Replace(wireSource, `wire("live")`, `wire("Expired")`, 1))
	var errs []*plugin.Diagnostic
	for _, d := range resp.GetDiagnostics() {
		if d.GetSeverity() == plugin.Severity_SEVERITY_ERROR {
			errs = append(errs, d)
		}
	}
	if len(errs) != 1 {
		t.Fatalf("diagnostics = %+v, want one error", resp.GetDiagnostics())
	}
	contains(t, errs[0].GetMessage(), `Status.Live's wire value "Expired" is the name of Status.Expired`)
	if _, ok := files(t, resp)["status.go"]; ok {
		t.Error("the colliding enum is generated")
	}
}

// A wire directive on a field renames its JSON key, and name renames only
// the Go identifier.
func TestWireRenamesFieldKeys(t *testing.T) {
	resp := generateTDL(t, `package shop

type Org {
  name: string
  slug: string?
  head: string
}

target go for shop {
  json
  Org.name => wire("orgName")
  Org.slug { name("Handle") wire("org_slug") }
  Org.head => name("Lead")
}
`)
	if w := uncodedWarnings(resp); len(w) != 0 {
		t.Fatalf("diagnostics = %+v", w)
	}
	contains(t, files(t, resp)["org.go"],
		"Name string `json:\"orgName\"`",
		"Handle *string `json:\"org_slug,omitempty\"`",
		"Lead string `json:\"head\"`")
	goTest(t, resp, `package shop

import (
	"encoding/json"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	const doc = `+"`"+`{"orgName":"Acme","org_slug":"acme","head":"Ada"}`+"`"+`
	var o Org
	if err := json.Unmarshal([]byte(doc), &o); err != nil {
		t.Fatal(err)
	}
	if o.Name != "Acme" || o.Handle == nil || *o.Handle != "acme" || o.Lead != "Ada" {
		t.Fatalf("decoded %+v", o)
	}
	out, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != doc {
		t.Fatalf("encoded %s, want %s", out, doc)
	}
}
`)
}

// A tag naming a JSON key wins over a wire directive naming another, and
// warns; a wire directive without json warns that it has no effect.
func TestWireBesideTag(t *testing.T) {
	resp := generateTDL(t, `package shop

type Org {
  name: string
  slug: string
}

target go for shop {
  json
  Org.name { tag("json:\"title\"") wire("orgName") }
  Org.slug { tag("json:\"slug\" yaml:\"s\"") wire("slug") }
}
`)
	w := uncodedWarnings(resp)
	if len(w) != 1 {
		t.Fatalf("diagnostics = %+v, want one warning", w)
	}
	contains(t, w[0].GetMessage(), "Org.name", `"title"`, `"orgName"`, "the tag wins")
	contains(t, files(t, resp)["org.go"], "Name string `json:\"title\"`")

	resp = generateTDL(t, `package shop

type Org { name: string }

target go for shop {
  Org.name => wire("orgName")
}
`)
	w = uncodedWarnings(resp)
	if len(w) != 1 {
		t.Fatalf("diagnostics = %+v, want one warning", w)
	}
	contains(t, w[0].GetMessage(), "Org.name", "has no effect")
}

// One wire directive per target gives every JSON backend the same key, and
// the Go type reads the JSON the TypeScript type describes.
func TestWireKeyAgreesAcrossBackends(t *testing.T) {
	const src = `package shop

type Org { name: string }
`
	backends := map[string]plugin.Backend{
		golang.Name:     golang.Backend{},
		typescript.Name: typescript.Backend{},
		zod.Name:        zod.Backend{},
		jsonschema.Name: jsonschema.Backend{},
		openapi.Name:    openapi.Backend{},
	}
	var b strings.Builder
	b.WriteString(src)
	for name := range backends {
		fmt.Fprintf(&b, "\ntarget %s for shop {\n", name)
		if name == golang.Name {
			b.WriteString("  json\n")
		}
		b.WriteString("  Org.name => wire(\"orgName\")\n}\n")
	}
	file, err := parser.Parse("main.tdl", strings.NewReader(b.String()))
	if err != nil {
		t.Fatal(err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatal(diags.Error())
	}
	want := map[string]string{
		golang.Name:     "`json:\"orgName\"`",
		typescript.Name: "orgName: string;",
		zod.Name:        "orgName: z.string(),",
		jsonschema.Name: `"orgName"`,
		openapi.Name:    "orgName:",
	}
	var goResp *plugin.Response
	for name, backend := range backends {
		resp, err := backend.Generate(context.Background(), &plugin.Request{Target: name, Model: model})
		if err != nil {
			t.Fatal(err)
		}
		var all strings.Builder
		for _, f := range resp.GetFiles() {
			all.Write(f.GetContent())
		}
		if !strings.Contains(all.String(), want[name]) || strings.Contains(all.String(), `"name"`) {
			t.Errorf("%s writes\n%s\nwant the key orgName and never name", name, all.String())
		}
		if name == golang.Name {
			goResp = resp
		}
	}
	goTest(t, goResp, `package shop

import (
	"encoding/json"
	"testing"
)

// A value of the TypeScript type: interface Org { orgName: string; }
const doc = `+"`"+`{"orgName":"Acme"}`+"`"+`

func TestRoundTrip(t *testing.T) {
	var o Org
	if err := json.Unmarshal([]byte(doc), &o); err != nil || o.Name != "Acme" {
		t.Fatalf("decoded %+v, %v", o, err)
	}
	out, _ := json.Marshal(o)
	if string(out) != doc {
		t.Fatalf("encoded %s", out)
	}
}
`)
}
