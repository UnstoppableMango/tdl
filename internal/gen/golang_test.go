package gen_test

import (
	"context"
	"go/parser"
	"go/token"
	"testing"

	"github.com/unstoppablemango/tdl/backend/golang"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// goModel is a model with one entity, enough to generate a Go file from.
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

// The Go backend is the second thing holding the protocol's one real
// claim: a compiled-in backend and the same backend as a subprocess
// produce the same thing. debug proved the claim for a backend that
// answers in one line; a code generator is where it would break first.
func TestGoHostsAgree(t *testing.T) {
	onPath(t, pluginDir(t))

	sub, err := gen.Find(golang.Name)
	if err != nil {
		t.Fatalf("find: %v", err)
	}

	req := &plugin.Request{Target: golang.Name, Model: goModel(), Out: "out"}

	inProcess, err := golang.Backend{}.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("in process: %v", err)
	}
	viaPipe, err := sub.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("subprocess: %v", err)
	}

	if len(inProcess.GetFiles()) != 1 {
		t.Fatalf("files = %+v", inProcess.GetFiles())
	}
	if len(inProcess.GetFiles()) != len(viaPipe.GetFiles()) {
		t.Fatalf("file counts differ: %d and %d", len(inProcess.GetFiles()), len(viaPipe.GetFiles()))
	}
	for i := range inProcess.GetFiles() {
		a, b := inProcess.GetFiles()[i], viaPipe.GetFiles()[i]
		if a.GetPath() != b.GetPath() {
			t.Errorf("path %d: %q and %q", i, a.GetPath(), b.GetPath())
		}
		if string(a.GetContent()) != string(b.GetContent()) {
			t.Errorf("content of %s differs between hosts", a.GetPath())
		}
	}

	// The output is Go, so the assertion that matters is that Go accepts
	// it. A substring check can pass while the file is something go build
	// refuses.
	file := inProcess.GetFiles()[0]
	if _, err := parser.ParseFile(token.NewFileSet(), file.GetPath(), file.GetContent(), parser.AllErrors); err != nil {
		t.Errorf("%s is not parseable Go: %v\n%s", file.GetPath(), err, file.GetContent())
	}
}

// The Go backend is compiled in, so a target block naming it resolves
// without anything on PATH.
func TestGoIsBuiltin(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	b, err := gen.Resolve(golang.Name)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if b.Describe().Name != golang.Name {
		t.Errorf("resolved %q", b.Describe().Name)
	}
}
