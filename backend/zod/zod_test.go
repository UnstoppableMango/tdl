package zod_test

import (
	"context"
	"maps"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/backend/internal/ts"
	"github.com/unstoppablemango/tdl/backend/typescript"
	"github.com/unstoppablemango/tdl/backend/zod"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

const smoke = "../../testdata/gen/smoke"

func lower(t *testing.T, src string) *ir.Model {
	t.Helper()
	file, err := parser.Parse("shop.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering: %v", diags.Error())
	}
	return model
}

func generate(t *testing.T, backend plugin.Backend, target string, m *ir.Model) *plugin.Response {
	t.Helper()
	resp, err := backend.Generate(context.Background(), &plugin.Request{Target: target, Model: m})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return resp
}

// gen generates a model from source with the zod backend and returns its
// one file.
func gen(t *testing.T, src string) (*plugin.Response, string) {
	t.Helper()
	resp := generate(t, zod.Backend{}, zod.Name, lower(t, src))
	if len(resp.GetFiles()) != 1 {
		t.Fatalf("files = %d, diagnostics = %v", len(resp.GetFiles()), resp.GetDiagnostics())
	}
	return resp, string(resp.GetFiles()[0].GetContent())
}

func contains(t *testing.T, src string, wants ...string) {
	t.Helper()
	flat := collapse(src)
	for _, want := range wants {
		if !strings.Contains(flat, collapse(want)) {
			t.Errorf("output missing %q:\n%s", want, src)
		}
	}
}

func absent(t *testing.T, src string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(src, u) {
			t.Errorf("output has %q:\n%s", u, src)
		}
	}
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// warned reports whether a warning carries code and mentions every one of
// words.
func warned(resp *plugin.Response, code string, words ...string) bool {
	for _, d := range resp.GetDiagnostics() {
		if d.GetSeverity() == plugin.Severity_SEVERITY_WARNING && d.GetCode() == code &&
			!slices.ContainsFunc(words, func(w string) bool { return !strings.Contains(d.GetMessage(), w) }) {
			return true
		}
	}
	return false
}

func quiet(t *testing.T, resp *plugin.Response) {
	t.Helper()
	if len(resp.GetDiagnostics()) > 0 {
		t.Errorf("diagnostics = %v", resp.GetDiagnostics())
	}
}

func TestDescribe(t *testing.T) {
	d := zod.Backend{}.Describe()
	if d.Name != "zod" || !d.Reuse || d.Reverse {
		t.Errorf("description = %+v", d)
	}
	declared := map[string]bool{}
	for _, spec := range d.Directives {
		declared[spec.GetName()] = true
	}
	for _, want := range []string{"name", "discriminant", "closed"} {
		if !declared[want] {
			t.Errorf("directive %q is acted on but not declared", want)
		}
	}
}

// The smoke model generates one module tsc accepts, parsing what the
// wire convention writes and refusing what it does not.
func TestSmoke(t *testing.T) {
	m := irtest.Conformance(t, smoke)
	resp := generate(t, zod.Backend{}, zod.Name, m)
	if len(resp.GetFiles()) != 1 || resp.GetFiles()[0].GetPath() != "smoke.zod.ts" {
		t.Fatalf("files = %v, diagnostics = %v", resp.GetFiles(), resp.GetDiagnostics())
	}
	if !warned(resp, emit.LossConstraint, "LineItem.quantity") || len(resp.GetDiagnostics()) != 1 {
		t.Errorf("diagnostics = %v, want one for LineItem.quantity's constraint", resp.GetDiagnostics())
	}
	typecheck(t, nil, resp)
	parse(t, `import * as m from "./smoke.zod.js";

const order = {
  id: "123e4567-e89b-12d3-a456-426614174000",
  createdAt: "2026-10-22T13:12:46Z",
  updatedAt: "2026-10-22T13:12:46.123+02:00",
  items: [{ sku: "A-1", quantity: 2 }],
  tags: ["a", "b"],
  shipOn: null,
  took: "PT1H",
  receipt: "aGVsbG8=",
  paid: true,
  contact: "a@example.com",
  skus: ["A-1"],
  payment: { kind: "Card", last4: "4242" },
  status: "Placed",
  fax: "",
};
const json = (o: object) => JSON.stringify(o);
accepts("Order", m.Order, json(order));
accepts("Order with a note", m.Order, json({ ...order, note: "leave at door" }));
refuses("Order without shipOn", m.Order, json({ ...order, shipOn: undefined }));
refuses("Order with a null note", m.Order, json({ ...order, note: null }));
refuses("Order with a duplicate tag", m.Order, json({ ...order, tags: ["a", "a"] }));
refuses("Order without seconds", m.Order, json({ ...order, createdAt: "2026-10-22T13:12Z" }));
refuses("Order without an offset", m.Order, json({ ...order, createdAt: "2026-10-22T13:12:46" }));
refuses("Order with a bad id", m.Order, json({ ...order, id: "not-a-uuid" }));
refuses("Order with a bad date", m.Order, json({ ...order, shipOn: "22/10/2026" }));
refuses("Order with a bad duration", m.Order, json({ ...order, took: "an hour" }));
refuses("Order with bad bytes", m.Order, json({ ...order, receipt: "%%%" }));
refuses("Order with a fractional quantity", m.Order, json({ ...order, items: [{ sku: "A-1", quantity: 1.5 }] }));
refuses("Order with an unknown status", m.Order, json({ ...order, status: "Lost" }));
refuses("Order with a Card without last4", m.Order, json({ ...order, payment: { kind: "Card" } }));
accepts("Payment", m.Payment, json({ kind: "Cash" }));
refuses("Payment of an unknown kind", m.Payment, json({ kind: "Cheque" }));
accepts("Totals", m.Totals, json({ byCurrency: { USD: "12.50", EUR: "-3" } }));
refuses("Totals with a number", m.Totals, json({ byCurrency: { USD: 12.5 } }));
refuses("Totals with an exponent", m.Totals, json({ byCurrency: { USD: "1e3" } }));
accepts("Counts", m.Counts, json({ byStatus: { Draft: 1 } }));
accepts("Counts with no key", m.Counts, json({ byStatus: {} }));
refuses("Counts with an unknown key", m.Counts, json({ byStatus: { Lost: 1 } }));
accepts("Sparse", m.Sparse, json({ values: [1, null, 3] }));
`, resp)
}

// The agreement test: each type the module infers and the type the
// typescript backend writes for the same declaration are assignable to
// each other.
func TestAgreesWithTypeScript(t *testing.T) {
	m := irtest.Conformance(t, smoke)
	z := generate(t, zod.Backend{}, zod.Name, m)
	tsResp := generate(t, typescript.Backend{}, typescript.Name, m)

	names := regexp.MustCompile(`(?m)^export type (\w+) =`).FindAllStringSubmatch(string(z.GetFiles()[0].GetContent()), -1)
	if len(names) == 0 {
		t.Fatal("the module declares no types")
	}
	var b strings.Builder
	b.WriteString(`import type * as T from "./smoke.js";
import type * as Z from "./smoke.zod.js";

type Same<A, B> = [A] extends [B] ? ([B] extends [A] ? true : false) : false;
`)
	declared := string(tsResp.GetFiles()[0].GetContent())
	for _, n := range names {
		if !regexp.MustCompile(`(?m)^export (interface|type) ` + n[1] + `\b`).MatchString(declared) {
			t.Errorf("typescript declares no %s", n[1])
			continue
		}
		b.WriteString("export const same" + n[1] + ": Same<T." + n[1] + ", Z." + n[1] + "> = true;\n")
	}
	typecheck(t, map[string]string{"agree.ts": b.String()}, z, tsResp)
}

// A field naming a declaration written later, or its own, is a getter
// with its return type, so the module loads and a cycle infers.
func TestRecursion(t *testing.T) {
	resp, src := gen(t, `package shop

type Customer: Entity {
  id: string
  orders: [Order]
  referrer: Customer?
}

type Order: Entity {
  id: string
  customer: Customer | null
  lines: {string -> Line}
}

type Forest: [Tree]

type Tree {
  label: string
  kids: Forest
}

enum Line {
  Item { sku: string }
  Bundle { lines: [Line] }
}
`)
	quiet(t, resp)
	contains(t, src,
		`get orders(): z.ZodArray<typeof Order> { return z.array(Order); },`,
		`get referrer(): z.ZodExactOptional<typeof Customer> { return Customer.exactOptional(); },`,
		`customer: Customer.nullable(),`,
		`get lines(): z.ZodRecord<z.ZodString, typeof Line> { return z.record(z.string(), Line); },`,
		`export const Forest: z.ZodLazy<z.ZodArray<typeof Tree>> = z.lazy(() => z.array(Tree));`,
		`get kids(): typeof Forest { return Forest; },`,
		`get lines(): z.ZodArray<typeof Line> { return z.array(Line); },`,
	)
	typecheck(t, map[string]string{"use.ts": `import type { Customer, Tree, Line } from "./shop.zod.js";
const c: Customer = { id: "c", orders: [{ id: "o", customer: null, lines: { a: { kind: "Bundle", lines: [{ kind: "Item", sku: "x" }] } } }] };
const t: Tree = { label: "root", kids: [{ label: "leaf", kids: [] }] };
const l: Line = { kind: "Item", sku: "y" };
export { c, t, l };
`}, resp)
	parse(t, `import * as m from "./shop.zod.js";

accepts("Customer", m.Customer, '{"id":"c","orders":[{"id":"o","customer":{"id":"c","orders":[]},"lines":{}}],"referrer":{"id":"r","orders":[]}}');
refuses("Customer with a bad nested order", m.Customer, '{"id":"c","orders":[{"id":"o","lines":{}}]}');
accepts("Tree", m.Tree, '{"label":"a","kids":[{"label":"b","kids":[]}]}');
refuses("Tree with a bad kid", m.Tree, '{"label":"a","kids":[{"label":1,"kids":[]}]}');
accepts("Line", m.Line, '{"kind":"Bundle","lines":[{"kind":"Item","sku":"x"}]}');
`, resp)
}

// Every primitive maps to the schema checking its JSON value.
func TestPrimitives(t *testing.T) {
	resp, src := gen(t, `package shop

type Values {
  s: string
  i: int
  i32: int32
  i64: int64
  u32: uint32
  u64: uint64
  f32: float32
  f64: float64
  b: bool
  raw: bytes
  d: decimal
  id: uuid
  at: instant
  on: date
  took: duration
}
`)
	quiet(t, resp)
	contains(t, src,
		`s: z.string(),`, `i: z.int(),`, `i32: z.int32(),`, `i64: z.int(),`, `u32: z.uint32(),`,
		`u64: z.int().nonnegative(),`, `f32: z.float32(),`, `f64: z.number(),`, `b: z.boolean(),`,
		`raw: z.base64(),`, `d: decimal,`, `id: z.uuid(),`, `at: instant,`, `on: z.iso.date(),`,
		`took: z.iso.duration(),`)
	typecheck(t, nil, resp)

	value := `{"s":"","i":1,"i32":1,"i64":1,"u32":1,"u64":1,"f32":1.5,"f64":1.5,"b":true,"raw":"","d":"1.5","id":"123e4567-e89b-12d3-a456-426614174000","at":"2026-01-01T00:00:00Z","on":"2026-01-01","took":"P1D"}`
	with := func(field, json string) string {
		return regexp.MustCompile(`"`+field+`":[^,}]*`).ReplaceAllString(value, `"`+field+`":`+json)
	}
	var script strings.Builder
	script.WriteString(`import { Values } from "./shop.zod.js";

accepts("Values", Values, '` + value + `');
`)
	for _, c := range [][2]string{
		{"i", "1e17"}, {"i", "1.5"}, {"i32", "2147483648"}, {"u32", "-1"}, {"u32", "4294967296"},
		{"u64", "-1"}, {"f32", "1e39"}, {"b", `"true"`}, {"d", `"1."`}, {"d", `"one"`},
		{"at", `"2026-01-01"`}, {"on", `"2026-13-01"`}, {"took", `"1 day"`},
	} {
		script.WriteString(`refuses("` + c[0] + ` = ` + strings.ReplaceAll(c[1], `"`, `\"`) + `", Values, '` + with(c[0], c[1]) + "');\n")
	}
	parse(t, script.String(), resp)
}

// Every primitive the shared mapping types has a schema.
func TestEveryScalarHasASchema(t *testing.T) {
	var fields strings.Builder
	for _, name := range slices.Sorted(maps.Keys(ts.Scalars)) {
		fields.WriteString("  f_" + name + ": " + name + "\n")
	}
	resp, _ := gen(t, "package shop\n\ntype Values {\n"+fields.String()+"}\n")
	quiet(t, resp)
}

// A map's key is a string, a number Zod reads from the key's digits, or a
// fieldless enum, which may leave keys out.
func TestMaps(t *testing.T) {
	resp, src := gen(t, `package shop

type Sku: string

enum Status {
  Draft
  Placed
}

type Maps {
  byName: {string -> int}
  bySku: {Sku -> int}
  byNumber: {int -> string}
  bySmall: {uint32 -> string}
  byStatus: {Status -> int}
}
`)
	quiet(t, resp)
	contains(t, src,
		`byName: z.record(z.string(), z.int()),`,
		`bySku: z.record(Sku, z.int()),`,
		`byNumber: z.record(z.int(), z.string()),`,
		`bySmall: z.record(z.uint32(), z.string()),`,
		`byStatus: z.partialRecord(Status, z.int()),`)
	typecheck(t, map[string]string{"use.ts": `import type { Maps } from "./shop.zod.js";
const m: Maps = { byName: {}, bySku: {}, byNumber: { 1: "a" }, bySmall: {}, byStatus: { Draft: 1 } };
const n: Record<number, string> = m.byNumber;
export { m, n };
`}, resp)
	parse(t, `import { Maps } from "./shop.zod.js";

const json = (o: object) => JSON.stringify({ byName: {}, bySku: {}, byNumber: {}, bySmall: {}, byStatus: {}, ...o });
accepts("integer keys", Maps, json({ byNumber: { "1": "a", "-2": "b" } }));
refuses("a key that is not an integer", Maps, json({ byNumber: { "1.5": "a" } }));
refuses("a key that is not a number", Maps, json({ byNumber: { x: "a" } }));
refuses("a negative unsigned key", Maps, json({ bySmall: { "-1": "a" } }));
`, resp)
}

// A bool key has no Zod record, so its declaration is skipped.
func TestUnsupportedKey(t *testing.T) {
	resp := generate(t, zod.Backend{}, zod.Name, lower(t, `package shop

type Bad {
  byFlag: {bool -> int}
}
`))
	if len(resp.GetFiles()) != 0 || len(resp.GetDiagnostics()) != 1 ||
		!strings.Contains(resp.GetDiagnostics()[0].GetMessage(), "record key") {
		t.Errorf("files = %d, diagnostics = %v", len(resp.GetFiles()), resp.GetDiagnostics())
	}
}

// Absence: T? may be left out and not null, T | null must be present and
// may be null, and both together may be either.
func TestAbsence(t *testing.T) {
	resp, src := gen(t, `package shop

type Note {
  optional: string?
  nullable: string | null
  both: string? | null
  elems: [string?]
}
`)
	quiet(t, resp)
	contains(t, src,
		`optional: z.string().exactOptional(),`,
		`nullable: z.string().nullable(),`,
		`both: z.string().nullable().exactOptional(),`,
		`elems: z.array(z.string().nullable()),`)
	typecheck(t, map[string]string{"use.ts": `import type { Note } from "./shop.zod.js";
const n: Note = { nullable: null, elems: [null, "a"] };
const o: Note = { nullable: "a", both: null, elems: [] };
export { n, o };
`}, resp)
	rejects(t, `import type { Note } from "./shop.zod.js";
export const n: Note = { optional: undefined, nullable: null, elems: [] };
`, resp)
	parse(t, `import { Note } from "./shop.zod.js";

accepts("absent", Note, '{"nullable":null,"elems":[]}');
accepts("present", Note, '{"optional":"a","nullable":"b","both":"c","elems":["d",null]}');
accepts("both null", Note, '{"nullable":null,"both":null,"elems":[]}');
refuses("optional null", Note, '{"optional":null,"nullable":null,"elems":[]}');
refuses("nullable absent", Note, '{"elems":[]}');
`, resp)
}

// An object strips a property it does not declare unless closed, on the
// declaration or the target block.
func TestClosed(t *testing.T) {
	resp, src := gen(t, `package shop

type Open {
  a: string
}

type Shut {
  a: string
}

enum Shape {
  Circle { r: float64 }
}

target zod for shop {
  Shut => closed
  Shape => closed
}
`)
	quiet(t, resp)
	contains(t, src, `export const Open = z.object({`, `export const Shut = z.strictObject({`, `export const ShapeCircle = z.strictObject({`)
	parse(t, `import * as m from "./shop.zod.js";

accepts("Open with an extra property", m.Open, '{"a":"x","b":1}');
refuses("Shut with an extra property", m.Shut, '{"a":"x","b":1}');
refuses("Shape with an extra property", m.Shape, '{"kind":"Circle","r":1,"d":2}');
`, resp)

	_, src = gen(t, `package shop

type Shut {
  a: string
}

target zod for shop {
  closed
}
`)
	contains(t, src, `export const Shut = z.strictObject({`)
}

// Enums: a fieldless one is a z.enum of its tags, a fielded one a
// discriminated union of variant objects named after the enum.
func TestEnums(t *testing.T) {
	resp, src := gen(t, `package shop

enum Status {
  Draft
  Placed
}

enum Event {
  Opened { at: instant }
  Closed
}

enum Shape {
  Circle { r: float64 }
  Square { side: float64 }
}

target zod for shop {
  Shape => discriminant("type")
  Shape.Square => name("Box")
}
`)
	quiet(t, resp)
	contains(t, src,
		`export const Status = z.enum(["Draft", "Placed"]);`,
		`export const EventOpened = z.object({ kind: z.literal("Opened"), at: instant, });`,
		`export const EventClosed = z.object({ kind: z.literal("Closed"), });`,
		`export const Event: z.ZodDiscriminatedUnion<[typeof EventOpened, typeof EventClosed], "kind"> = z.discriminatedUnion("kind", [EventOpened, EventClosed]);`,
		`export const Box = z.object({ type: z.literal("Square"), side: z.number(), });`,
		`export const Shape: z.ZodDiscriminatedUnion<[typeof ShapeCircle, typeof Box], "type"> = z.discriminatedUnion("type", [ShapeCircle, Box]);`)
	typecheck(t, nil, resp)
}

// A variant field named like the discriminant skips its enum, and the
// declarations naming it.
func TestDiscriminantCollision(t *testing.T) {
	resp := generate(t, zod.Backend{}, zod.Name, lower(t, `package shop

enum Event {
  Opened { kind: string }
}

type Log {
  last: Event
}

type Fine {
  a: string
}
`))
	src := string(resp.GetFiles()[0].GetContent())
	absent(t, src, "Event", "Log")
	contains(t, src, "export const Fine")
	if len(resp.GetDiagnostics()) != 2 ||
		!strings.Contains(resp.GetDiagnostics()[0].GetMessage(), "discriminant") ||
		!strings.Contains(resp.GetDiagnostics()[1].GetMessage(), "Log names") {
		t.Errorf("diagnostics = %v", resp.GetDiagnostics())
	}
}

// A name TypeScript reserves, one the module declares for itself, and one
// two declarations take each warn and skip the declaration.
func TestNames(t *testing.T) {
	resp, src := gen(t, `package shop

type Account {
  /// The holder.
  holder: string
  id: string
}

type Keyword {
  a: string
}

type Helper {
  a: string
}

type First {
  a: string
}

type Second {
  a: string
}

target zod for shop {
  Account => name("User")
  Account.id => name("userId")
  Account.holder => name("user-name")
  Keyword => name("default")
  Helper => name("unique")
  Second => name("First")
}
`)
	contains(t, src,
		`export const User = z.object({`,
		`/** * The holder. */ "user-name": z.string(),`,
		`userId: z.string(),`,
		`export const First = z.object({`)
	absent(t, src, "default", "const unique", "Second")
	for _, want := range []string{"cannot name a type", "declares for itself", "already does"} {
		if !slices.ContainsFunc(resp.GetDiagnostics(), func(d *plugin.Diagnostic) bool { return strings.Contains(d.GetMessage(), want) }) {
			t.Errorf("no warning says %q: %v", want, resp.GetDiagnostics())
		}
	}
	typecheck(t, nil, resp)
}

// A doc comment is JSDoc on the schema, its type, and a property, and a
// deprecation is @deprecated.
func TestDocs(t *testing.T) {
	resp, src := gen(t, `package shop

/// An account.
deprecated("use User")
type Account {
  /// When it opened.
  deprecated opened: instant
}
`)
	quiet(t, resp)
	contains(t, src,
		"/** * An account. * * @deprecated use User */ export const Account = z.object({",
		"/** * When it opened. * * @deprecated */ opened: instant,",
		"/** * An account. * * @deprecated use User */ export type Account = z.infer<typeof Account>;")
}

// Phase 1 checks no where constraint and gives a unit no brand, and each
// warns.
func TestNotYet(t *testing.T) {
	resp, src := gen(t, `package shop

unit kg

type Qty: int where { min(1) }

type Parcel {
  weight: decimal<kg>
  count: int where { max(9) }
  qty: Qty
}
`)
	contains(t, src, `weight: decimal,`, `count: z.int(),`, `export const Qty = z.int();`)
	absent(t, src, ".min(", ".max(")
	for _, w := range []struct {
		code  string
		words []string
	}{
		{emit.LossUnit, []string{"kg is a unit"}},
		{emit.LossUnit, []string{"Parcel.weight"}},
		{emit.LossConstraint, []string{"Parcel.count"}},
		{emit.LossConstraint, []string{"Qty"}},
	} {
		if !warned(resp, w.code, w.words...) {
			t.Errorf("no %s warning mentions %v: %v", w.code, w.words, resp.GetDiagnostics())
		}
	}
	typecheck(t, nil, resp)
}

// A helper is declared only when a declaration uses it.
func TestHelpersOnlyWhenUsed(t *testing.T) {
	_, src := gen(t, `package shop

type Plain {
  a: string
}
`)
	absent(t, src, "const decimal", "const instant", "const unique")

	_, src = gen(t, `package shop

type Tagged {
  tags: {string}
}
`)
	contains(t, src, "const unique = ")
	absent(t, src, "const decimal", "const instant")
}

// A model with nothing to declare writes no file.
func TestEmpty(t *testing.T) {
	resp := generate(t, zod.Backend{}, zod.Name, lower(t, "package shop\n\nalias Name = string\n"))
	if len(resp.GetFiles()) != 0 {
		t.Errorf("files = %v", resp.GetFiles())
	}
}

// A model without a package is model.zod.ts.
func TestNoPackage(t *testing.T) {
	resp := generate(t, zod.Backend{}, zod.Name, lower(t, "type A {\n  a: string\n}\n"))
	if len(resp.GetFiles()) != 1 || resp.GetFiles()[0].GetPath() != "model.zod.ts" {
		t.Errorf("files = %v", resp.GetFiles())
	}
}
