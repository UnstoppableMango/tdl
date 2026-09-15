package gen_test

import (
	"context"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/backend/debug"
	"github.com/unstoppablemango/tdl/backend/golang"
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
	// debug exercises the protocol and is not shipped.
	packaged bool

	// valid checks that a file is something the target language accepts.
	valid func(t *testing.T, f *plugin.File)
}{
	{backend: debug.Backend{}, model: sampleModel},
	{backend: golang.Backend{}, model: goModel, packaged: true, valid: parseGo},
}

// The protocol's one real claim: a compiled-in backend and the same
// backend as a subprocess produce the same thing. A plan that shipped only
// the in-process path could state that and never check it, and a code
// generator is where it would break first.
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

// A description survives the wire, so tdl can check a target block against
// what a plugin says it understands.
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

// A backend that is compiled in but missing from the package still works
// for `tdl gen`, and is absent for anyone who wanted it as a plugin, which
// nothing else would notice.
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

// goModel is a model with one keyed entity, enough to generate a Go file
// from.
func goModel() *ir.Model {
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

func parseGo(t *testing.T, f *plugin.File) {
	t.Helper()
	if _, err := parser.ParseFile(token.NewFileSet(), f.GetPath(), f.GetContent(), parser.AllErrors); err != nil {
		t.Errorf("%s is not parseable Go: %v\n%s", f.GetPath(), err, f.GetContent())
	}
}
