package protobuf_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/protobuf"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// sources is an in-memory [sema.Loader] keyed by import path.
type sources map[string]string

func (s sources) Load(_, path string) (string, string, error) {
	src, ok := s[path]
	if !ok {
		return path, "", os.ErrNotExist
	}
	return path, src, nil
}

// lower parses and lowers src as name, importing from deps, and fails on
// any diagnostic.
func lower(t *testing.T, name, src string, deps sources) *ir.Model {
	t.Helper()
	file, err := parser.Parse(name, strings.NewReader(src))
	if err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}
	model, diags := sema.Lower(file, sema.WithLoader(deps))
	if len(diags) > 0 {
		t.Fatalf("lowering %s: %v", name, diags.Error())
	}
	return model
}

// generateIR runs the backend over a lowered model.
func generateIR(t *testing.T, model *ir.Model) *plugin.Response {
	t.Helper()
	resp, err := protobuf.Backend{}.Generate(context.Background(), &plugin.Request{
		Target: protobuf.Name,
		Model:  model,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return resp
}

const shopSource = `package acme.shop.v1

import "dep/money.tdl" as money

type Price { amount: money.Money }

type Fine { a: string }
`

// A reference into the dependency imports the file it generates.
func TestDependencyWithProtobufBlockIsImported(t *testing.T) {
	const moneySource = `package acme.money.v1

type Money { units: int }

target protobuf for acme.money.v1 {
  file("money.proto")
}
`
	deps := sources{"dep/money.tdl": moneySource}

	depResp := generateIR(t, lower(t, "dep/money.tdl", moneySource, nil))
	if len(uncoded(depResp.GetDiagnostics())) != 0 {
		t.Fatalf("dependency diagnostics = %+v", uncoded(depResp.GetDiagnostics()))
	}
	if len(depResp.GetFiles()) != 1 {
		t.Fatalf("dependency files = %d", len(depResp.GetFiles()))
	}
	depFile := depResp.GetFiles()[0]
	if got, want := depFile.GetPath(), "acme/money/v1/money.proto"; got != want {
		t.Fatalf("dependency path = %q, want %q", got, want)
	}

	resp := generateIR(t, lower(t, "main.tdl", shopSource, deps))
	if len(uncoded(resp.GetDiagnostics())) != 0 {
		t.Errorf("diagnostics = %+v", uncoded(resp.GetDiagnostics()))
	}
	src := compileWith(t, resp, map[string]string{depFile.GetPath(): string(depFile.GetContent())})
	contains(t, src,
		`import "acme/money/v1/money.proto";`,
		"acme.money.v1.Money amount = 1;",
		"message Fine { string a = 1; }",
	)
	absent(t, src, "message Money")
}

func TestDependencyWithoutProtobufBlockIsSkipped(t *testing.T) {
	deps := sources{"dep/money.tdl": `package acme.money.v1

type Money { units: int }
`}

	resp := generateIR(t, lower(t, "main.tdl", shopSource, deps))
	diags := uncoded(resp.GetDiagnostics())
	if len(diags) != 1 {
		t.Fatalf("diagnostics = %+v, want one warning", diags)
	}
	if diags[0].GetSeverity() != plugin.Severity_SEVERITY_WARNING {
		t.Errorf("severity = %v", diags[0].GetSeverity())
	}
	contains(t, diags[0].GetMessage(), "acme.money.v1.Money", "foreign types are not generated yet")

	src := compile(t, resp)
	absent(t, src, "message Price", "import ")
	contains(t, src, "message Fine { string a = 1; }")
}
