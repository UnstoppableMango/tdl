package golang_test

import (
	"context"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/golang"
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
