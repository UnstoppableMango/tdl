package zod_test

import (
	"os"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/zod"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
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

// lowerWith parses and lowers src as name, importing from d.
func lowerWith(t *testing.T, name, src string, d deps) *ir.Model {
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

// A default is written on its field, so z.input may leave the field out
// and z.infer always holds it, as the typescript interface does.
func TestDefaults(t *testing.T) {
	resp, src := gen(t, `package shop

enum Status {
  Draft
  Placed
}

type Count: int

type Defaults {
  name: string = "anon"
  count: int64 = 1
  small: int32 = -5
  ratio: float64 = 0.5
  whole: float32 = 2
  on: bool = true
  price: decimal = "12.50"
  rate: decimal = 1.25
  id: uuid = "123e4567-e89b-12d3-a456-426614174000"
  at: instant = "2026-10-22T13:12:46Z"
  day: date = "2026-02-28"
  took: duration = "PT1H30M"
  raw: bytes = "aGk="
  status: Status = Placed
  n: Count = 3
  tags: [string] = ["a", "b"]
  unique: {int} = [1, 2]
  maybe: string | null = "set"
}
`)
	quiet(t, resp)
	contains(t, src,
		`name: z.string().default("anon"),`,
		`count: z.int().default(1),`,
		`small: z.int32().default(-5),`,
		`ratio: z.number().default(0.5),`,
		`whole: z.float32().default(2),`,
		`on: z.boolean().default(true),`,
		`price: decimal.default("12.50"),`,
		`rate: decimal.default("1.25"),`,
		`id: z.uuid().default("123e4567-e89b-12d3-a456-426614174000"),`,
		`at: instant.default("2026-10-22T13:12:46Z"),`,
		`day: z.iso.date().default("2026-02-28"),`,
		`took: z.iso.duration().default("PT1H30M"),`,
		`raw: z.base64().default("aGk="),`,
		`status: Status.default("Placed"),`,
		`n: Count.default(3),`,
		`tags: z.array(z.string()).default(["a", "b"]),`,
		`unique: z.array(z.int()).refine(unique, "a set holds each element once").default([1, 2]),`,
		`maybe: z.string().nullable().default("set"),`)
	typecheck(t, map[string]string{"use.ts": `import type * as z from "zod";
import { Defaults } from "./shop.zod.js";
const input: z.input<typeof Defaults> = {};
const name: string = Defaults.parse(input).name;
export { input, name };
`}, resp)
	rejects(t, `import type { Defaults } from "./shop.zod.js";
export const d: Defaults = {};
`, resp)
	parse(t, `import { Defaults } from "./shop.zod.js";

const d = Defaults.parse({});
const want = {
  name: "anon", count: 1, small: -5, ratio: 0.5, whole: 2, on: true, price: "12.50", rate: "1.25",
  id: "123e4567-e89b-12d3-a456-426614174000", at: "2026-10-22T13:12:46Z", day: "2026-02-28",
  took: "PT1H30M", raw: "aGk=", status: "Placed", n: 3, tags: ["a", "b"], unique: [1, 2], maybe: "set",
};
if (JSON.stringify(d) !== JSON.stringify(want)) {
  throw new Error("defaults: " + JSON.stringify(d));
}
// Each default is one the schema it defaults accepts.
for (const [key, field] of Object.entries(Defaults.shape)) {
  accepts("the default of " + key, field.unwrap(), JSON.stringify((want as Record<string, unknown>)[key]));
}
`, resp)
}

// A default the field's schema would refuse warns and is not written, by
// the rules Zod checks: each refused value here is one Zod refuses too.
func TestBadDefaults(t *testing.T) {
	cases := []struct{ field, typ, literal, json string }{
		{"big", "int", "9007199254740992", "9007199254740992"},
		{"wide", "int32", "2147483648", "2147483648"},
		{"negative", "uint32", "-1", "-1"},
		{"unsigned", "uint64", "-1", "-1"},
		{"huge", "float32", "1000000000000000000000000000000000000000.0", "1e39"},
		{"exponent", "decimal", `"1e3"`, `"1e3"`},
		{"id", "uuid", `"not-a-uuid"`, `"not-a-uuid"`},
		{"minutes", "instant", `"2026-10-22T13:12Z"`, `"2026-10-22T13:12Z"`},
		{"local", "instant", `"2026-10-22T13:12:46"`, `"2026-10-22T13:12:46"`},
		{"leap", "date", `"2026-02-29"`, `"2026-02-29"`},
		{"empty", "duration", `"PT"`, `"PT"`},
		{"bare", "duration", `"P"`, `"P"`},
		{"raw", "bytes", `"abc"`, `"abc"`},
		{"twice", "{string}", `["a", "a"]`, `["a","a"]`},
	}
	var fields strings.Builder
	for _, c := range cases {
		fields.WriteString("  " + c.field + ": " + c.typ + " = " + c.literal + "\n")
	}
	fields.WriteString("  note: string? = \"x\"\n")
	resp, src := gen(t, "package shop\n\ntype Bad {\n"+fields.String()+"}\n")
	absent(t, src, ".default(")
	for _, c := range append(cases, struct{ field, typ, literal, json string }{field: "note"}) {
		if !warned(resp, emit.LossDefault, "Bad."+c.field+"'s default") {
			t.Errorf("no warning for Bad.%s's default: %v", c.field, resp.GetDiagnostics())
		}
	}

	var script strings.Builder
	script.WriteString("import { Bad } from \"./shop.zod.js\";\n\n")
	for _, c := range cases {
		script.WriteString("refuses(\"" + c.field + "\", Bad.shape." + c.field + ", '" + c.json + "');\n")
	}
	parse(t, script.String(), resp)
}

// The good side of the same rules: a value at each edge Zod accepts is
// written.
func TestEdgeDefaults(t *testing.T) {
	resp, src := gen(t, `package shop

type Edge {
  safe: int = 9007199254740991
  low: int32 = -2147483648
  top: uint32 = 4294967295
  nil: uuid = "00000000-0000-0000-0000-000000000000"
  offset: instant = "2026-10-22T13:12:46.123456+05:30"
  leap: date = "2024-02-29"
  weeks: duration = "P2W"
  fraction: duration = "PT0,5S"
  none: bytes = ""
}
`)
	quiet(t, resp)
	if n := strings.Count(src, ".default("); n != 9 {
		t.Errorf("%d defaults written, want 9:\n%s", n, src)
	}
	parse(t, `import { Edge } from "./shop.zod.js";

for (const [key, field] of Object.entries(Edge.shape)) {
  accepts("the default of " + key, field.unwrap(), JSON.stringify(field.parse(undefined)));
}
`, resp)
}

// A foreign declaration is imported from the module its directive names,
// and not declared; a field naming it uses the import.
func TestForeignDeclaration(t *testing.T) {
	resp, src := gen(t, `package shop

type Money {
  units: int
}

type Sku: string where { matches(/^[A-Z]+$/) }

type Order {
  total: Money
  sku: Sku
  lines: [Money]
}

target zod for shop {
  Money => foreign("./money.js", "MoneySchema")
  Sku => foreign("@acme/catalog", "Sku")
}
`)
	contains(t, src,
		`import { Sku } from "@acme/catalog";`,
		`import { MoneySchema } from "./money.js";`,
		`total: MoneySchema,`,
		`sku: Sku,`,
		`lines: z.array(MoneySchema),`)
	absent(t, src, "export const Money", "export const Sku")
	if !warned(resp, emit.LossConstraint, "Sku is foreign") || len(resp.GetDiagnostics()) != 1 {
		t.Errorf("diagnostics = %v, want one for Sku's constraint", resp.GetDiagnostics())
	}
	typecheck(t, map[string]string{
		"money.ts":                                "import * as z from \"zod\";\nexport const MoneySchema = z.object({ units: z.int() });\n",
		"node_modules/@acme/catalog/index.ts":     "import * as z from \"zod\";\nexport const Sku = z.string();\n",
		"node_modules/@acme/catalog/package.json": `{"name": "@acme/catalog", "type": "module", "types": "./index.ts"}`,
	}, resp)
}

// A default on a foreign type warns and is dropped, since the backend
// cannot check it.
func TestForeignDefault(t *testing.T) {
	resp, src := gen(t, `package shop

type Code: string

type Order {
  code: Code = "x"
}

target zod for shop {
  Code => foreign("./code.js", "Code")
}
`)
	absent(t, src, ".default(")
	if !warned(resp, emit.LossDefault, "foreign") {
		t.Errorf("diagnostics = %v", resp.GetDiagnostics())
	}
}

// An import naming what a declaration declares skips the declaration
// importing it.
func TestForeignConflict(t *testing.T) {
	resp, src := gen(t, `package shop

type Money {
  units: int
}

type Total {
  amount: int
}

type Order {
  total: Money
}

target zod for shop {
  Money => foreign("./money.js", "Total")
}
`)
	contains(t, src, "export const Total")
	absent(t, src, "export const Order", "import {")
	if !warned(resp, "", "Order would import Total, which Total declares") {
		t.Errorf("diagnostics = %v", resp.GetDiagnostics())
	}
}

const moneySource = `package acme.money

type Money {
  units: int
}

type Currency {
  code: string
}

target zod for acme.money {
  out("gen/money")
  Currency => name("ISO4217")
}
`

const shopSource = `package acme.shop

import "money.tdl" as money

type Order {
  total: money.Money
  currency: money.Currency
  lines: [money.Money]
}

target zod for acme.shop {
  out("gen/shop")
}
`

// An extern is imported from the module its dependency's zod block
// generates, by a path relative to this one, and the two modules type
// check and parse as one tree.
func TestExternFromDependency(t *testing.T) {
	money := generate(t, zod.Backend{}, zod.Name, lowerWith(t, "money.tdl", moneySource, nil))
	shop := generate(t, zod.Backend{}, zod.Name, lowerWith(t, "shop.tdl", shopSource, deps{"money.tdl": moneySource}))
	quiet(t, shop)
	src := string(shop.GetFiles()[0].GetContent())
	contains(t, src,
		`import { ISO4217, Money } from "../money/money.zod.js";`,
		`total: Money,`,
		`currency: ISO4217,`,
		`lines: z.array(Money),`)
	absent(t, src, "export const Money")

	files := map[string]string{
		"gen/money/money.zod.ts": string(money.GetFiles()[0].GetContent()),
		"gen/shop/shop.zod.ts":   src,
	}
	typecheckFiles(t, files)
	parseFiles(t, files, `import { Order } from "./gen/shop/shop.zod.js";

accepts("Order", Order, '{"total":{"units":1},"currency":{"code":"EUR"},"lines":[]}');
refuses("Order with a bad total", Order, '{"total":{"units":1.5},"currency":{"code":"EUR"},"lines":[]}');
`)
}

// A foreign directive on an extern names the module and schema it is
// imported as.
func TestExternForeign(t *testing.T) {
	const money = `package acme.money

type Money {
  units: int
}

target zod for acme.money {
  Money => foreign("@acme/money", "MoneySchema")
}
`
	resp := generate(t, zod.Backend{}, zod.Name, lowerWith(t, "shop.tdl", `package acme.shop

import "money.tdl" as money

type Order {
  total: money.Money
}
`, deps{"money.tdl": money}))
	quiet(t, resp)
	contains(t, string(resp.GetFiles()[0].GetContent()),
		`import { MoneySchema } from "@acme/money";`, `total: MoneySchema,`)
}

// An extern with neither a foreign directive nor a zod block in its
// dependency warns, and so does each declaration naming it.
func TestExternUnmapped(t *testing.T) {
	const money = `package acme.money

type Money {
  units: int
}
`
	resp := generate(t, zod.Backend{}, zod.Name, lowerWith(t, "shop.tdl", `package acme.shop

import "money.tdl" as money

type Order {
  total: money.Money
}

type Fine {
  a: string
}
`, deps{"money.tdl": money}))
	if len(resp.GetFiles()) != 1 || strings.Contains(string(resp.GetFiles()[0].GetContent()), "Order") {
		t.Fatalf("files = %v", resp.GetFiles())
	}
	if !warned(resp, "", "no zod target block") {
		t.Errorf("diagnostics = %v", resp.GetDiagnostics())
	}
}

// A dependency's zod block needs an out directive, and so does this one,
// for one module to import the other by a relative path.
func TestExternNeedsOut(t *testing.T) {
	noOut := strings.Replace(moneySource, `  out("gen/money")`, `  closed`, 1)
	resp := generate(t, zod.Backend{}, zod.Name, lowerWith(t, "shop.tdl", shopSource, deps{"money.tdl": noOut}))
	if !warned(resp, "", "no out directive") {
		t.Errorf("diagnostics = %v", resp.GetDiagnostics())
	}
}
