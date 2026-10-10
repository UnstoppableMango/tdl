package golang_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/golang"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// deps is an in-memory [sema.Loader] keyed by import path.
type deps map[string]string

func (d deps) Load(_, path string) (string, string, error) {
	src, ok := d[path]
	if !ok {
		return path, "", os.ErrNotExist
	}
	return path, src, nil
}

// lowerSource parses and lowers src as name, importing from d, and fails on
// any diagnostic.
func lowerSource(t *testing.T, name, src string, d deps) *ir.Model {
	t.Helper()
	file, err := parser.Parse(name, strings.NewReader(src))
	if err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}
	model, diags := sema.Lower(file, sema.WithLoader(d))
	if len(diags) > 0 {
		t.Fatalf("lowering %s: %v", name, diags.Error())
	}
	return model
}

func generateModel(t *testing.T, model *ir.Model) *plugin.Response {
	t.Helper()
	resp, err := golang.Backend{}.Generate(context.Background(), &plugin.Request{
		Target: golang.Name,
		Model:  model,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return resp
}

// warnings is the response's warnings, past those about what import would
// read back.
func warnings(resp *plugin.Response) []*plugin.Diagnostic {
	var out []*plugin.Diagnostic
	for _, d := range resp.GetDiagnostics() {
		if !readsBackOtherwise(d) {
			out = append(out, d)
		}
	}
	return out
}

const moneySource = `package acme.money

type Money { units: int }

type Currency { code: string }

target go for acme.money {
  package("example.com/generated/money")
  Currency => name("ISO4217")
}
`

// buildModule writes each package's files into its directory of one module
// and builds it.
func buildModule(t *testing.T, pkgs map[string]*plugin.Response) {
	t.Helper()
	if testing.Short() {
		t.Skip("runs go build on the generated packages")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go is not on PATH")
	}
	dir := t.TempDir()
	write := func(name, src string) {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/generated\n\ngo 1.21\n")
	var all []string
	for sub, resp := range pkgs {
		for _, f := range resp.GetFiles() {
			write(filepath.Join(sub, f.GetPath()), string(f.GetContent()))
			all = append(all, "==> "+sub+"/"+f.GetPath()+" <==\n"+string(f.GetContent()))
		}
	}

	cmd := exec.CommandContext(t.Context(), goBin, "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build on the generated packages: %v\n%s\n%s", err, out, strings.Join(all, "\n"))
	}
}

// A reference into a dependency with a go target block imports the package
// that block generates, under the name it gives the declaration.
func TestDependencyWithGoBlockIsImported(t *testing.T) {
	depResp := generateModel(t, lowerSource(t, "dep/money.tdl", moneySource, nil))
	if w := warnings(depResp); len(w) != 0 {
		t.Fatalf("dependency diagnostics = %+v", w)
	}

	for _, imp := range []string{`as money`, `as _`} {
		t.Run(imp, func(t *testing.T) {
			prefix := ""
			if imp == `as money` {
				prefix = "money."
			}
			resp := generateModel(t, lowerSource(t, "main.tdl", `package acme.shop

import "dep/money.tdl" `+imp+`

type Price {
  amount: `+prefix+`Money
  currency: `+prefix+`Currency
  history: List<`+prefix+`Money>
}

target go for acme.shop {
  package("example.com/generated/shop")
}
`, deps{"dep/money.tdl": moneySource}))
			if w := warnings(resp); len(w) != 0 {
				t.Errorf("diagnostics = %+v", w)
			}
			got := raw(resp)
			contains(t, got["price.go"],
				`money "example.com/generated/money"`,
				"Amount money.Money",
				"Currency money.ISO4217",
				"History []money.Money",
			)
			buildModule(t, map[string]*plugin.Response{"money": depResp, "shop": resp})
		})
	}
}

// A dependency generated into the same Go package is referred to
// unqualified.
func TestDependencyInSamePackageIsUnqualified(t *testing.T) {
	resp := generateModel(t, lowerSource(t, "main.tdl", `package acme.shop

import "dep/money.tdl" as money

type Price { amount: money.Money }

target go for acme.shop {
  package("example.com/generated/money")
}
`, deps{"dep/money.tdl": moneySource}))
	if w := warnings(resp); len(w) != 0 {
		t.Errorf("diagnostics = %+v", w)
	}
	got := raw(resp)
	contains(t, got["price.go"], "Amount Money")
	if strings.Contains(got["price.go"], "import") {
		t.Errorf("price.go imports a package:\n%s", got["price.go"])
	}
}

// Without a package directive the dependency has no import path, so a
// reference into it is skipped as before.
func TestDependencyWithoutGoPackageIsSkipped(t *testing.T) {
	resp := generateModel(t, lowerSource(t, "main.tdl", `package acme.shop

import "dep/money.tdl" as money

type Price { amount: money.Money }

type Fine { a: string }
`, deps{"dep/money.tdl": `package acme.money

type Money { units: int }
`}))
	w := warnings(resp)
	if len(w) != 1 {
		t.Fatalf("diagnostics = %+v, want one warning", w)
	}
	contains(t, w[0].GetMessage(), "acme.money.Money", "foreign types are not generated yet")
	got := raw(resp)
	if _, ok := got["price.go"]; ok {
		t.Errorf("Price was generated: files = %v", keys(got))
	}
	if _, ok := got["fine.go"]; !ok {
		t.Errorf("Fine was not generated: files = %v", keys(got))
	}
}
