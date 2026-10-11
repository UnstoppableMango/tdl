package typescript_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/typescript"
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
	resp, err := typescript.Backend{}.Generate(context.Background(), &plugin.Request{
		Target: typescript.Name,
		Model:  model,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	resp.Diagnostics = slices.DeleteFunc(resp.Diagnostics, readsBackOtherwise)
	return resp
}

const moneySource = `package acme.money

type Money { units: int }

type Currency { code: string }

target typescript for acme.money {
  out("gen/money")
  Currency => name("ISO4217")
}
`

// typecheckTree writes each file under its out directory in one temporary
// tree and runs tsc over the last, so its imports resolve against the rest.
func typecheckTree(t *testing.T, files map[string][]byte, root string) {
	t.Helper()
	if testing.Short() {
		return
	}
	tsc, err := exec.LookPath("tsc")
	if err != nil {
		return
	}
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := exec.Command(tsc, "--noEmit", "--strict", filepath.Join(dir, root)).CombinedOutput(); err != nil {
		t.Errorf("tsc rejects %s: %v\n%s\n%s", root, err, out, files[root])
	}
}

func TestADependencyIsImportedFromItsOutFile(t *testing.T) {
	for _, tc := range []struct{ name, clause, ref string }{
		{"alias", "as money", "money.Money"},
		{"underscore", "as _", "Money"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := deps{"money.tdl": moneySource}
			money := generateModel(t, lowerSource(t, "money.tdl", moneySource, nil))
			model := lowerSource(t, "shop.tdl", `package acme.shop

import "money.tdl" `+tc.clause+`

type Order {
  total: `+tc.ref+`
  currency: `+strings.Replace(tc.ref, "Money", "Currency", 1)+`
}

target typescript for acme.shop {
  out("gen/shop")
}
`, d)
			resp := generateModel(t, model)
			if len(resp.GetDiagnostics()) > 0 {
				t.Fatalf("diagnostics: %+v", resp.GetDiagnostics())
			}
			if len(resp.GetFiles()) != 1 {
				t.Fatalf("files = %d", len(resp.GetFiles()))
			}
			src := string(resp.GetFiles()[0].GetContent())
			contains(t, src,
				`import type { ISO4217, Money } from "../money/money.js";`,
				"total: Money;",
				"currency: ISO4217;")
			typecheckTree(t, map[string][]byte{
				"gen/money/money.ts": money.GetFiles()[0].GetContent(),
				"gen/shop/shop.ts":   resp.GetFiles()[0].GetContent(),
			}, "gen/shop/shop.ts")
		})
	}
}

func TestAForeignDirectiveImportsAHandWrittenModule(t *testing.T) {
	d := deps{"unist.tdl": `package unist

type Point { line: int }

target typescript for unist {
  Point => foreign("unist", "Point")
}
`}
	model := lowerSource(t, "doc.tdl", `package acme.doc

import "unist.tdl" as unist

type Node { at: unist.Point }
`, d)
	resp := generateModel(t, model)
	if len(resp.GetDiagnostics()) > 0 {
		t.Fatalf("diagnostics: %+v", resp.GetDiagnostics())
	}
	if len(resp.GetFiles()) != 1 {
		t.Fatalf("files = %d", len(resp.GetFiles()))
	}
	src := string(resp.GetFiles()[0].GetContent())
	contains(t, src, `import type { Point } from "unist";`, "at: Point;")
}

// skipped generates shop against money and returns the response, which
// must skip Order with one warning containing want.
func skipped(t *testing.T, money, shop, want string) {
	t.Helper()
	model := lowerSource(t, "shop.tdl", shop, deps{"money.tdl": money})
	resp := generateModel(t, model)
	diags := resp.GetDiagnostics()
	if len(diags) != 1 || !strings.Contains(diags[0].GetMessage(), want) {
		t.Fatalf("diagnostics = %+v, want one containing %q", diags, want)
	}
	for _, f := range resp.GetFiles() {
		if strings.Contains(string(f.GetContent()), "Order") {
			t.Errorf("Order was written:\n%s", f.GetContent())
		}
	}
}

func TestADependencyWithoutAnOutIsSkipped(t *testing.T) {
	skipped(t, `package acme.money

type Money { units: int }

target typescript for acme.money {
  Money => name("Money")
}
`, `package acme.shop

import "money.tdl" as money

type Order { total: money.Money }

target typescript for acme.shop {
  out("gen/shop")
}
`, "no out directive")
}

func TestARootWithoutAnOutIsSkipped(t *testing.T) {
	skipped(t, moneySource, `package acme.shop

import "money.tdl" as money

type Order { total: money.Money }

target typescript for acme.shop {}
`, "this target block has no out directive")
}

func TestADependencyWithoutATargetBlockIsSkipped(t *testing.T) {
	skipped(t, `package acme.money

type Money { units: int }
`, `package acme.shop

import "money.tdl" as money

type Order { total: money.Money }
`, "no typescript target block")
}

func TestAnImportCollidingWithADeclarationIsSkipped(t *testing.T) {
	skipped(t, moneySource, `package acme.shop

import "money.tdl" as money

type Money { cents: int }

type Order { total: money.Money }

target typescript for acme.shop {
  out("gen/shop")
}
`, "which Money declares")
}
