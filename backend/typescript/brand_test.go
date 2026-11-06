package typescript_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/typescript"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

const brandSource = `package shop

type Secret: string
type Username: string
type Email: string
type WorkEmail: Email
type Grade: string where { oneOf("a", "b") }
type Maybe: string | null
type Login { user: Username secret: Secret }

target typescript for shop {
  brand
}
`

// With brand on the block every newtype is branded by its qualified name,
// a newtype over a newtype adding its own brand to its parent's.
func TestBrandOnTheBlock(t *testing.T) {
	resp := generateModel(t, lowerSource(t, "shop.tdl", brandSource, nil))
	if len(resp.GetDiagnostics()) > 0 {
		t.Errorf("diagnostics = %v", resp.GetDiagnostics())
	}
	src := check(t, resp)
	contains(t, src,
		`export type Secret = string & { readonly __brand: { readonly "shop.Secret": true } };`,
		`export type Username = string & { readonly __brand: { readonly "shop.Username": true } };`,
		`export type WorkEmail = Email & { readonly __brand: { readonly "shop.WorkEmail": true } };`,
		`export type Grade = ("a" | "b") & { readonly __brand: { readonly "shop.Grade": true } };`,
		`export type Maybe = string & { readonly __brand: { readonly "shop.Maybe": true } } | null;`,
		"user: Username;",
	)

	for _, tc := range []struct {
		name, snippet string
		ok            bool
	}{
		{"distinct newtypes", `declare const s: Secret; export const u: Username = s;`, false},
		{"base to newtype", `export const s: Secret = "hunter2";`, false},
		{"newtype to base", `declare const s: Secret; export const raw: string = s;`, true},
		{"child to parent", `declare const w: WorkEmail; export const e: Email = w;`, true},
		{"parent to child", `declare const e: Email; export const w: WorkEmail = e;`, false},
		{"null", `export const m: Maybe = null;`, true},
		{"cast", `export const l: Login = { user: "ann" as Username, secret: "x" as Secret };`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			compiles(t, src, tc.snippet, tc.ok)
		})
	}
}

// brand on a newtype brands that one alone.
func TestBrandOnANewtype(t *testing.T) {
	model := lowerSource(t, "shop.tdl", `package shop

type Secret: string
type Username: string

target typescript for shop {
  Secret => brand
}
`, nil)
	src := check(t, generateModel(t, model))
	contains(t, src,
		`export type Secret = string & { readonly __brand: { readonly "shop.Secret": true } };`,
		"export type Username = string;",
	)
}

// Placement cannot say newtype only, so brand anywhere else warns, with
// no loss code, and changes nothing.
func TestBrandOnAStructWarns(t *testing.T) {
	model := lowerSource(t, "shop.tdl", `package shop

type Login { user: string }

target typescript for shop {
  Login => brand
}
`, nil)
	resp := generateModel(t, model)
	diags := resp.GetDiagnostics()
	if len(diags) != 1 || diags[0].GetCode() != "" || diags[0].GetSeverity() != plugin.Severity_SEVERITY_WARNING ||
		!strings.Contains(diags[0].GetMessage(), "Login is not a newtype") {
		t.Errorf("diagnostics = %v", diags)
	}
	contains(t, check(t, resp), "export interface Login { user: string; }")
}

// Under readback, a brand the block gives reads back on each newtype.
func TestBrandOnTheBlockUnderReadback(t *testing.T) {
	model := lowerSource(t, "shop.tdl", `package shop

type Secret: string

target typescript for shop {
  brand
  readback
}
`, nil)
	resp := generateRaw(t, model)
	var found bool
	for _, d := range resp.GetDiagnostics() {
		if d.GetCode() == emit.LossName && strings.Contains(d.GetMessage(), "brand") {
			found = true
		}
	}
	if !found {
		t.Errorf("diagnostics = %v, want a %s warning about the block's brand", resp.GetDiagnostics(), emit.LossName)
	}
}

// generateRaw generates model, keeping every diagnostic.
func generateRaw(t *testing.T, model *ir.Model) *plugin.Response {
	t.Helper()
	resp, err := typescript.Backend{}.Generate(context.Background(), &plugin.Request{Target: typescript.Name, Model: model})
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// compiles runs tsc over the generated module with snippet appended, and
// fails unless tsc accepts it exactly when ok is set.
func compiles(t *testing.T, module, snippet string, ok bool) {
	t.Helper()
	if testing.Short() {
		t.Skip("runs tsc")
	}
	tsc, err := exec.LookPath("tsc")
	if err != nil {
		t.Skip("tsc is not on PATH")
	}
	path := filepath.Join(t.TempDir(), "snippet.ts")
	if err := os.WriteFile(path, []byte(module+"\n"+snippet+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(tsc, "--noEmit", "--strict", path).CombinedOutput()
	if got := err == nil; got != ok {
		t.Errorf("tsc accepts %q = %v, want %v\n%s", snippet, got, ok, out)
	}
}

// A branded alias reads back as its newtype with a brand directive; a
// brand keyed by another name is dropped with a warning.
func TestImportReadsBrands(t *testing.T) {
	src, diags := importTS(t, map[string]string{"shop.ts": `
export type Secret = string & { readonly __brand: { readonly "shop.Secret": true } };
export type Email = string;
export type WorkEmail = Email & { readonly __brand: { readonly "shop.WorkEmail": true } };
export type Grade = ("a" | "b") & { readonly __brand: { readonly "shop.Grade": true } };
export type Maybe = string & { readonly __brand: { readonly "shop.Maybe": true } } | null;
export type Other = string & { readonly __brand: { readonly "elsewhere.Other": true } };
`})
	for _, s := range []string{
		"type Secret: string\n",
		"type WorkEmail: Email\n",
		`type Grade: string where { oneOf("a", "b") }`,
		"type Maybe: string | null\n",
		"type Other: string\n",
		"Secret => brand",
		"WorkEmail => brand",
		"Grade => brand",
		"Maybe => brand",
	} {
		if !strings.Contains(src, s) {
			t.Errorf("import lacks %q:\n%s", s, src)
		}
	}
	for _, line := range strings.Split(src, "\n") {
		if l := strings.TrimSpace(line); l == "Email => brand" || l == "Other => brand" {
			t.Errorf("import writes %q:\n%s", l, src)
		}
	}
	var warned bool
	for _, d := range diags {
		if d.GetCode() == emit.LossUnsupported && strings.Contains(d.GetMessage(), "elsewhere.Other") {
			warned = true
		}
	}
	if !warned {
		t.Errorf("diagnostics = %v, want a warning about Other's brand", diags)
	}
}

// An extern newtype branded in its own package is the same type through
// import type, so it keeps its brand.
func TestAnExternKeepsItsBrand(t *testing.T) {
	const ids = `package acme.ids

type UserId: string

target typescript for acme.ids {
  out("gen/ids")
  brand
}
`
	idsResp := generateModel(t, lowerSource(t, "ids.tdl", ids, nil))
	resp := generateModel(t, lowerSource(t, "shop.tdl", `package acme.shop

import "ids.tdl" as ids

type Order { owner: ids.UserId }

target typescript for acme.shop {
  out("gen/shop")
}
`, deps{"ids.tdl": ids}))
	if len(resp.GetFiles()) != 1 {
		t.Fatalf("files = %d, diagnostics = %v", len(resp.GetFiles()), resp.GetDiagnostics())
	}
	shop := resp.GetFiles()[0].GetContent()
	contains(t, string(shop), `import type { UserId } from "../ids/ids.js";`, "owner: UserId;")

	tree := func(use string) map[string][]byte {
		return map[string][]byte{
			"gen/ids/ids.ts":   idsResp.GetFiles()[0].GetContent(),
			"gen/shop/shop.ts": shop,
			"gen/shop/use.ts":  []byte(`import type { Order } from "./shop.js";` + "\n" + use + "\n"),
		}
	}
	typecheckTree(t, tree(`import type { UserId } from "../ids/ids.js";
export const o: Order = { owner: "ann" as UserId };`), "gen/shop/use.ts")
	if testing.Short() {
		return
	}
	tsc, err := exec.LookPath("tsc")
	if err != nil {
		return
	}
	dir := t.TempDir()
	for name, content := range tree(`export const o: Order = { owner: "ann" };`) {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := exec.Command(tsc, "--noEmit", "--strict", filepath.Join(dir, "gen/shop/use.ts")).Run(); err == nil {
		t.Error("tsc accepts a plain string for an extern branded newtype")
	}
}
