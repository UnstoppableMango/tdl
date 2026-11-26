package gen_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

const (
	depSource = `package shop.billing

/// An invoice.
type Invoice {
  id: string
}

target fickle for shop.billing {
  out("gen/billing")
}
`
	dependentSource = `package shop

import "billing.tdl" as billing

type Order {
  invoice: billing.Invoice
}

target fickle for shop {
  out("gen/shop")
}
`
)

// depProject writes a dependency and a model importing it into a new
// directory, returning the directory.
func depProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range map[string]string{"billing.tdl": depSource, "shop.tdl": dependentSource} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// runFile generates path's one target with b, reading imports from disk.
func runFile(t *testing.T, b plugin.Backend, path string, mode gen.Mode) gen.Result {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.Parse(path, strings.NewReader(string(src)))
	if err != nil {
		t.Fatal(err)
	}
	model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
	if len(diags) > 0 {
		t.Fatal(diags)
	}
	targets, err := gen.Targets(model, "")
	if err != nil {
		t.Fatal(err)
	}
	targets[0].Loader = sema.FSLoader{}
	result, err := gen.Run(context.Background(), b, targets[0], model, mode)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// dependRequest is shop.tdl's request for a nondeterministic backend,
// with what [gen.Depend] adds.
func dependRequest(t *testing.T, dir string) *plugin.Request {
	t.Helper()
	path := filepath.Join(dir, "shop.tdl")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.Parse(path, strings.NewReader(string(src)))
	if err != nil {
		t.Fatal(err)
	}
	model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
	if len(diags) > 0 {
		t.Fatal(diags)
	}
	req, _ := analysis.Request(model, (&fickle{}).Describe(), "fickle", filepath.Join(dir, "gen/shop"))
	if _, err := gen.Depend(req, sema.FSLoader{}); err != nil {
		t.Fatal(err)
	}
	return req
}

const invoiceSource = "/// An invoice.\ntype Invoice {\n  id: string\n}"

// A nondeterministic backend's request carries each extern's declaration
// and what its dependency generated, with paths relative to its own out.
func TestDependSendsExternsAndOutput(t *testing.T) {
	dir := depProject(t)
	runFile(t, &fickle{}, filepath.Join(dir, "billing.tdl"), gen.ModeWrite)
	req := dependRequest(t, dir)

	externs := req.GetAnalysis().GetSource().GetExterns()
	if len(externs) != 1 || externs[0].GetSource() != invoiceSource {
		t.Errorf("externs = %v", externs)
	}
	deps := req.GetDependencies()
	if len(deps) != 1 {
		t.Fatalf("dependencies = %v", deps)
	}
	if d := deps[0]; d.GetPackage() != "shop.billing" || d.GetPath() != "../billing/out.txt" || !strings.HasPrefix(string(d.GetContent()), "call 1 ") {
		t.Errorf("dependency = %v", d)
	}
}

// Regenerating a dependency changes its dependent's request, so the
// dependent regenerates rather than keeping what it wrote.
func TestDependencyOutputEntersTheHash(t *testing.T) {
	dir := depProject(t)
	dep, shop := filepath.Join(dir, "billing.tdl"), filepath.Join(dir, "shop.tdl")
	depBackend, b := &fickle{}, &fickle{}

	runFile(t, depBackend, dep, gen.ModeWrite)
	runFile(t, b, shop, gen.ModeWrite)
	if r := runFile(t, b, shop, gen.ModeWrite); len(r.Kept) != 1 || b.calls != 1 {
		t.Fatalf("an unchanged dependency regenerated: kept %v after %d calls", r.Kept, b.calls)
	}

	// The dependency's backend answers differently, so its file changes.
	runFile(t, depBackend, dep, gen.ModeClean)
	if r := runFile(t, b, shop, gen.ModeWrite); len(r.Written) != 1 || b.calls != 2 {
		t.Errorf("a regenerated dependency kept the dependent: wrote %v after %d calls", r.Written, b.calls)
	}
}

// A dependency with no output yet sends no files, and still its externs.
func TestDependWithoutOutput(t *testing.T) {
	req := dependRequest(t, depProject(t))
	if deps := req.GetDependencies(); len(deps) != 0 {
		t.Errorf("dependencies = %v", deps)
	}
	if externs := req.GetAnalysis().GetSource().GetExterns(); len(externs) != 1 || externs[0].GetSource() != invoiceSource {
		t.Errorf("externs = %v", externs)
	}
}

// A dependency writing into the dependent's own out sends nothing, since
// that record is the dependent's own last output, and the dependent keeps
// its files.
func TestDependSkipsTheRequestsOwnOut(t *testing.T) {
	dir := depProject(t)
	shared := strings.Replace(depSource, "gen/billing", "gen/shop", 1)
	if err := os.WriteFile(filepath.Join(dir, "billing.tdl"), []byte(shared), 0o644); err != nil {
		t.Fatal(err)
	}
	b := &fickle{}
	shop := filepath.Join(dir, "shop.tdl")
	runFile(t, b, shop, gen.ModeWrite)
	if r := runFile(t, b, shop, gen.ModeWrite); len(r.Kept) != 1 || b.calls != 1 {
		t.Errorf("the dependent regenerated over its own output: kept %v after %d calls", r.Kept, b.calls)
	}
}

// Every file a package spans is read, and what was read is returned for
// --make.
func TestDependReadsEveryFileOfAPackage(t *testing.T) {
	dir := depProject(t)
	files := map[string]string{
		"payment.tdl": "package shop.billing\n\ntype Payment {\n  id: string\n}\n",
		"shop.tdl":    "package shop\n\nimport \"billing.tdl\" as billing\nimport \"payment.tdl\" as payment\n\ntype Order {\n  invoice: billing.Invoice\n  payment: payment.Payment\n}\n\ntarget fickle for shop {\n  out(\"gen/shop\")\n}\n",
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runFile(t, &fickle{}, filepath.Join(dir, "billing.tdl"), gen.ModeWrite)
	r := runFile(t, &fickle{}, filepath.Join(dir, "shop.tdl"), gen.ModeWrite)
	want := []string{filepath.Join(dir, "gen/billing", gen.MarkerName), filepath.Join(dir, "gen/billing/out.txt")}
	if !slices.Equal(r.Read, want) {
		t.Errorf("read %v, want %v", r.Read, want)
	}

	externs := dependRequest(t, dir).GetAnalysis().GetSource().GetExterns()
	if len(externs) != 2 || externs[1].GetSource() != "type Payment {\n  id: string\n}" {
		t.Errorf("externs = %v", externs)
	}
}
