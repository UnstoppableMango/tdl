package gen_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"go/parser"
	"go/token"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/bufbuild/protocompile"
	thriftparser "github.com/cloudwego/thriftgo/parser"
	"github.com/cloudwego/thriftgo/semantic"
	sjs "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/vektah/gqlparser/v2"
	gqlast "github.com/vektah/gqlparser/v2/ast"
	"go.yaml.in/yaml/v3"
	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/backend/debug"
	"github.com/unstoppablemango/tdl/backend/golang"
	"github.com/unstoppablemango/tdl/backend/graphql"
	"github.com/unstoppablemango/tdl/backend/jsonschema"
	"github.com/unstoppablemango/tdl/backend/likec4"
	"github.com/unstoppablemango/tdl/backend/openapi"
	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/backend/salesforce"
	"github.com/unstoppablemango/tdl/backend/smithy"
	"github.com/unstoppablemango/tdl/backend/thrift"
	"github.com/unstoppablemango/tdl/backend/typescript"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// shipped is every backend tdl compiles in, with the smallest model each
// one generates a file from. [TestEveryBuiltinHasARow] fails until a
// backend added to the registry has a row here.
var shipped = []struct {
	backend plugin.Backend
	model   func() *ir.Model

	// packaged is whether nix/cmd.nix ships the backend as tdl-gen-<name>.
	packaged bool

	// valid checks that a file is something the target language accepts.
	valid func(t *testing.T, f *plugin.File)

	// reverse is the source a backend that imports reads in
	// [TestImportHostsAgree]. It is nil for one that only generates.
	reverse func() []*plugin.File
}{
	{backend: debug.Backend{}, model: sampleModel, packaged: true},
	{backend: golang.Backend{}, model: goModel, packaged: true, valid: parseGo},
	{backend: graphql.Backend{}, model: orderModel, packaged: true, valid: loadGraphQL},
	{backend: jsonschema.Backend{}, model: orderModel, packaged: true, valid: compileJSONSchema},
	{backend: likec4.Backend{}, model: orderModel, packaged: true},
	{backend: openapi.Backend{}, model: orderModel, packaged: true, valid: parseYAML},
	{backend: protobuf.Backend{}, model: orderModel, packaged: true, valid: compileProto},
	{backend: salesforce.Backend{}, model: orderModel, packaged: true, valid: parseXML},
	{backend: smithy.Backend{}, model: orderModel, packaged: true},
	{backend: thrift.Backend{}, model: orderModel, packaged: true, valid: checkThrift},
	{backend: typescript.Backend{}, model: orderModel, packaged: true},
}

// A compiled-in backend and the same backend as a subprocess produce the
// same bytes.
func TestHostsAgree(t *testing.T) {
	onPath(t, pluginDir(t))

	for _, row := range shipped {
		name := row.backend.Describe().Name
		t.Run(name, func(t *testing.T) {
			sub, err := gen.Find(name)
			if err != nil {
				t.Fatalf("find: %v", err)
			}

			req := &plugin.Request{Target: name, Model: row.model(), Out: "out"}
			inProcess, err := row.backend.Generate(context.Background(), req)
			if err != nil {
				t.Fatalf("in process: %v", err)
			}
			viaPipe, err := sub.Generate(context.Background(), req)
			if err != nil {
				t.Fatalf("subprocess: %v", err)
			}

			if len(inProcess.GetFiles()) == 0 {
				t.Fatalf("the model generated nothing to compare: %+v", inProcess.GetDiagnostics())
			}
			if len(inProcess.GetFiles()) != len(viaPipe.GetFiles()) {
				t.Fatalf("file counts differ: %d and %d", len(inProcess.GetFiles()), len(viaPipe.GetFiles()))
			}
			for i, a := range inProcess.GetFiles() {
				b := viaPipe.GetFiles()[i]
				if a.GetPath() != b.GetPath() {
					t.Errorf("path %d: %q and %q", i, a.GetPath(), b.GetPath())
				}
				if string(a.GetContent()) != string(b.GetContent()) {
					t.Errorf("content of %s differs between hosts", a.GetPath())
				}
				if row.valid != nil {
					row.valid(t, a)
				}
			}
		})
	}
}

// A description survives the wire.
func TestDescribeOverTheWire(t *testing.T) {
	onPath(t, pluginDir(t))

	for _, row := range shipped {
		want := row.backend.Describe()
		t.Run(want.Name, func(t *testing.T) {
			sub, err := gen.Find(want.Name)
			if err != nil {
				t.Fatalf("find: %v", err)
			}

			got := sub.Describe()
			if got.Name != want.Name || got.Version != want.Version {
				t.Errorf("got %s %s, want %s %s", got.Name, got.Version, want.Name, want.Version)
			}
			if len(got.Directives) != len(want.Directives) {
				t.Fatalf("directives: %d over the wire, %d in process", len(got.Directives), len(want.Directives))
			}
			for i := range want.Directives {
				if !proto.Equal(got.Directives[i], want.Directives[i]) {
					t.Errorf("directive %d: %v over the wire, %v in process", i, got.Directives[i], want.Directives[i])
				}
			}
		})
	}
}

// A compiled-in backend resolves without anything on PATH.
func TestBuiltinsResolveWithoutPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	for _, row := range shipped {
		name := row.backend.Describe().Name
		b, err := gen.Resolve(name)
		if err != nil {
			t.Errorf("resolve %s: %v", name, err)
			continue
		}
		if b.Describe().Name != name {
			t.Errorf("resolved %q for %q", b.Describe().Name, name)
		}
	}
}

func TestEveryBuiltinHasARow(t *testing.T) {
	var names []string
	for _, row := range shipped {
		names = append(names, row.backend.Describe().Name)
	}
	slices.Sort(names)
	if !slices.Equal(names, gen.BuiltinNames()) {
		t.Errorf("rows %v, registry %v", names, gen.BuiltinNames())
	}
}

// Every packaged backend is listed in nix/cmd.nix's subPackages.
func TestPackagedBackendsShip(t *testing.T) {
	src, err := os.ReadFile("../../nix/cmd.nix")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range shipped {
		if !row.packaged {
			continue
		}
		pkg := `"cmd/` + gen.CommandPrefix + row.backend.Describe().Name + `"`
		if !strings.Contains(string(src), pkg) {
			t.Errorf("nix/cmd.nix does not list %s in subPackages", pkg)
		}
	}
}

// orderModel is a model with one keyed entity.
func orderModel() *ir.Model {
	return &ir.Model{
		Package: "shop",
		Decls: []*ir.Decl{
			{
				Meta: &ir.Meta{Name: "string", Position: &ir.Position{Filename: "/nix/store/x/std.tdl"}},
				Node: &ir.Decl_Primitive{Primitive: &ir.Primitive{}},
			},
			{
				Meta: &ir.Meta{Name: "Order", Position: &ir.Position{Filename: "shop.tdl"}},
				Directives: []*ir.Directive{{
					Name:   "key",
					Target: golang.Name,
					Args:   []*ir.Literal{{Kind: ir.LiteralKind_LITERAL_KIND_NAME, Text: "id"}},
				}},
				Node: &ir.Decl_Structure{Structure: &ir.Struct{
					Kind: ir.StructKind_STRUCT_KIND_ENTITY,
					Fields: []*ir.Field{{
						Meta: &ir.Meta{Name: "id"},
						Type: &ir.ID{Index: 0, Name: "string"},
					}},
				}},
			},
		},
		Types: []*ir.Type{{
			Ctor:  &ir.ID{Index: 0, Name: "string"},
			Wrote: ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
		}},
	}
}

// goModel is [orderModel] with a generic struct, a constraint, and a
// foreign mapping, which only the Go backend generates.
func goModel() *ir.Model {
	m := orderModel()
	m.Decls[1].GetStructure().Fields[0].Constraints = []*ir.Constraint{{
		Name: "matches",
		Args: []*ir.Literal{{Kind: ir.LiteralKind_LITERAL_KIND_REGEX, Text: "^[a-z0-9-]+$"}},
	}}
	m.Decls = append(m.Decls, &ir.Decl{
		Meta: &ir.Meta{Name: "Box", Position: &ir.Position{Filename: "shop.tdl"}},
		Node: &ir.Decl_Structure{Structure: &ir.Struct{
			Params: []*ir.Param{{Name: "T"}},
			Fields: []*ir.Field{{
				Meta: &ir.Meta{Name: "value"},
				Type: &ir.ID{Index: 1, Name: "T"},
			}},
		}},
	})
	m.Types = append(m.Types, &ir.Type{Param: &ir.ParamRef{Name: "T"}})

	m.Decls = append(m.Decls, &ir.Decl{
		Meta: &ir.Meta{Name: "Money", Position: &ir.Position{Filename: "shop.tdl"}},
		Directives: []*ir.Directive{{
			Name:   "foreign",
			Target: golang.Name,
			Args:   []*ir.Literal{irText("math/big"), irText("Int")},
		}},
		Node: &ir.Decl_Newtype{Newtype: &ir.Newtype{Base: &ir.ID{Index: 0, Name: "string"}}},
	})
	m.Types = append(m.Types, &ir.Type{
		Ctor:  &ir.ID{Index: 3, Name: "Money"},
		Wrote: ir.SyntacticForm_SYNTACTIC_FORM_NAMED,
	})
	m.Decls[1].GetStructure().Fields = append(m.Decls[1].GetStructure().Fields, &ir.Field{
		Meta: &ir.Meta{Name: "total"},
		Type: &ir.ID{Index: 2, Name: "Money"},
	})
	return m
}

func parseGo(t *testing.T, f *plugin.File) {
	t.Helper()
	if _, err := parser.ParseFile(token.NewFileSet(), f.GetPath(), f.GetContent(), parser.AllErrors); err != nil {
		t.Errorf("%s is not parseable Go: %v\n%s", f.GetPath(), err, f.GetContent())
	}
}

func compileProto(t *testing.T, f *plugin.File) {
	t.Helper()
	c := protocompile.Compiler{Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
		Accessor: protocompile.SourceAccessorFromMap(map[string]string{f.GetPath(): string(f.GetContent())}),
	})}
	if _, err := c.Compile(context.Background(), f.GetPath()); err != nil {
		t.Errorf("%s does not compile: %v\n%s", f.GetPath(), err, f.GetContent())
	}
}

func irText(s string) *ir.Literal {
	return &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_STRING, Text: s}
}

func checkThrift(t *testing.T, f *plugin.File) {
	t.Helper()
	ast, err := thriftparser.ParseString(f.GetPath(), string(f.GetContent()))
	if err == nil {
		err = semantic.ResolveSymbols(ast)
	}
	if err != nil {
		t.Errorf("%s is not valid Thrift: %v\n%s", f.GetPath(), err, f.GetContent())
	}
}

// parseXML checks that Salesforce metadata is well formed; Apex is not
// checked.
func parseXML(t *testing.T, f *plugin.File) {
	t.Helper()
	if !strings.HasSuffix(f.GetPath(), ".xml") {
		return
	}
	d := xml.NewDecoder(bytes.NewReader(f.GetContent()))
	for {
		if _, err := d.Token(); err == io.EOF {
			return
		} else if err != nil {
			t.Errorf("%s is not well formed: %v\n%s", f.GetPath(), err, f.GetContent())
			return
		}
	}
}

func loadGraphQL(t *testing.T, f *plugin.File) {
	t.Helper()
	if _, err := gqlparser.LoadSchema(&gqlast.Source{Name: f.GetPath(), Input: string(f.GetContent())}); err != nil {
		t.Errorf("%s does not load: %v\n%s", f.GetPath(), err, f.GetContent())
	}
}

func compileJSONSchema(t *testing.T, f *plugin.File) {
	t.Helper()
	doc, err := sjs.UnmarshalJSON(bytes.NewReader(f.GetContent()))
	if err != nil {
		t.Errorf("%s is not JSON: %v\n%s", f.GetPath(), err, f.GetContent())
		return
	}
	c := sjs.NewCompiler()
	loc := "file:///" + f.GetPath()
	if err := c.AddResource(loc, doc); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Compile(loc); err != nil {
		t.Errorf("%s does not compile: %v\n%s", f.GetPath(), err, f.GetContent())
	}
}

// parseYAML checks an OpenAPI document is YAML; backend/openapi's tests
// validate it against the OpenAPI schema.
func parseYAML(t *testing.T, f *plugin.File) {
	t.Helper()
	var v any
	if err := yaml.Unmarshal(f.GetContent(), &v); err != nil {
		t.Errorf("%s is not YAML: %v\n%s", f.GetPath(), err, f.GetContent())
	}
}
