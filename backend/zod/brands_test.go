package zod_test

import (
	"regexp"
	"slices"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/irtest"
	"github.com/unstoppablemango/tdl/backend/zod"
)

const checkedModel = `package shop

type Sku: string where { length(1..8) }

type Code: Sku

type Qty: int where { min(1) }

type Row {
  sku: Sku
  qty: Qty
  price: float64 where { min(0.5) max(100) }
  name: string where { length(4) }
  tags: [string] where { length(1..20) unique }
  set: {string} where { unique length(..2) }
  meta: {string -> int} where { length(1..) }
  note: string? where { length(..3) }
  count: int where { max(9) } = 3
  first: Qty = 2
  code: Code
}

enum Payment {
  Card { last4: string where { length(4) } }
  Cash
}
`

// Each standard constraint is a check on a branded schema, a constrained
// field's own, and refuses what it constrains under node.
func TestChecks(t *testing.T) {
	resp, src := gen(t, checkedModel)
	quiet(t, resp)
	contains(t, src,
		`export const Sku = z.string().refine(codePoints(1, 8), "Sku: length(1..8)").brand<"shop.Sku">();`,
		`export const Code = Sku.brand<"shop.Code">();`,
		`export const Qty = z.int().min(1, "Qty: min(1)").brand<"shop.Qty">();`,
		`export const RowPrice = z.number().min(0.5, "Row.price: min(0.5)").max(100, "Row.price: max(100)").brand<"shop.Row.price">();`,
		`export const RowName = z.string().refine(codePoints(4, 4), "Row.name: length(4)").brand<"shop.Row.name">();`,
		`export const RowTags = z.array(z.string()).min(1, "Row.tags: length(1..20)").max(20, "Row.tags: length(1..20)").refine(unique, "Row.tags: unique").brand<"shop.Row.tags">();`,
		`export const RowSet = z.array(z.string()).refine(unique, "a set holds each element once").max(2, "Row.set: length(..2)").brand<"shop.Row.set">();`,
		`export const RowMeta = z.record(z.string(), z.int()).refine(size(1, Infinity), "Row.meta: length(1..)").brand<"shop.Row.meta">();`,
		`note: RowNote.exactOptional(),`,
		`count: RowCount.prefault(3),`,
		`first: Qty.prefault(2),`,
		`export const PaymentCardLast4 = z.string().refine(codePoints(4, 4), "Payment.Card.last4: length(4)").brand<"shop.Payment.Card.last4">();`,
		`last4: PaymentCardLast4,`)
	typecheck(t, nil, resp)
	parse(t, `import * as m from "./shop.zod.js";
const row = { sku: "A1", qty: 1, price: 0.5, name: "😀😀😀😀", tags: ["a"], set: [], meta: { a: 1 }, code: "C" };
const json = (o: object) => JSON.stringify(o);
accepts("Row", m.Row, json(row));
accepts("Row with a note", m.Row, json({ ...row, note: "abc" }));
refuses("an empty Sku", m.Row, json({ ...row, sku: "" }));
refuses("a long Sku", m.Row, json({ ...row, sku: "123456789" }));
refuses("a long Code", m.Row, json({ ...row, code: "123456789" }));
refuses("qty 0", m.Row, json({ ...row, qty: 0 }));
refuses("a cheap price", m.Row, json({ ...row, price: 0.4 }));
refuses("a dear price", m.Row, json({ ...row, price: 101 }));
refuses("a three-letter name", m.Row, json({ ...row, name: "abc" }));
refuses("four letters and an emoji", m.Row, json({ ...row, name: "abc😀😀" }));
refuses("no tags", m.Row, json({ ...row, tags: [] }));
refuses("a repeated tag", m.Row, json({ ...row, tags: ["a", "a"] }));
refuses("a big set", m.Row, json({ ...row, set: ["a", "b", "c"] }));
refuses("empty meta", m.Row, json({ ...row, meta: {} }));
refuses("a long note", m.Row, json({ ...row, note: "abcd" }));
refuses("count 10", m.Row, json({ ...row, count: 10 }));
accepts("PaymentCard", m.Payment, json({ kind: "Card", last4: "1234" }));
refuses("PaymentCard with five digits", m.Payment, json({ kind: "Card", last4: "12345" }));
const filled = m.Row.parse(row);
if (filled.count !== 3 || filled.first !== 2) {
  throw new Error("defaults not filled: " + json(filled));
}
`, resp)
	// A checked type is refused where a plain value is written.
	rejects(t, `import type { Row } from "./shop.zod.js";
export const r: Row["qty"] = 0;
`, resp)
	rejects(t, `import type { Sku } from "./shop.zod.js";
export const s: Sku = "A1";
`, resp)
	rejects(t, `import type { Code, Sku } from "./shop.zod.js";
declare const s: Sku;
export const c: Code = s;
`, resp)
	// A Code is a Sku, since brands stack.
	typecheck(t, map[string]string{"ok.ts": `import type { Code, Sku } from "./shop.zod.js";
declare const c: Code;
export const s: Sku = c;
`}, resp)
}

// The smoke model's LineItem refuses a quantity of 0 and a plain string
// for its Sku in tsc.
func TestSmokeBrands(t *testing.T) {
	resp := generate(t, zod.Backend{}, zod.Name, irtest.Conformance(t, smoke))
	rejects(t, `import type { LineItem, Sku } from "./smoke.zod.js";
declare const sku: Sku;
export const item: LineItem = { sku, quantity: 0 };
`, resp)
	rejects(t, `import type { LineItem, LineItemQuantity } from "./smoke.zod.js";
declare const quantity: LineItemQuantity;
export const item: LineItem = { sku: "A-1", quantity };
`, resp)
}

// structural writes no brands and keeps every check and every name.
func TestStructural(t *testing.T) {
	_, branded := gen(t, checkedModel)
	resp, src := gen(t, checkedModel+`
target zod for shop {
  structural
}
`)
	quiet(t, resp)
	absent(t, src, ".brand<")
	contains(t, src,
		`export const Qty = z.int().min(1, "Qty: min(1)");`,
		`export const RowPrice = z.number().min(0.5, "Row.price: min(0.5)").max(100, "Row.price: max(100)");`)
	exported := func(src string) []string {
		var out []string
		for _, m := range regexp.MustCompile(`(?m)^export (?:const|type) (\w+)`).FindAllStringSubmatch(src, -1) {
			out = append(out, m[1])
		}
		return out
	}
	if a, b := exported(branded), exported(src); !slices.Equal(a, b) {
		t.Errorf("structural exports %v, branded %v", b, a)
	}
	typecheck(t, map[string]string{"plain.ts": `import type { Row } from "./shop.zod.js";
export const r: Row = { sku: "A1", qty: 1, price: 1, name: "abcd", tags: ["a"], set: [], meta: {}, count: 3, first: 2, code: "C" };
`}, resp)
}

// A constraint with no check on its type warns, as does a oneOf argument
// another constraint excludes, and a default a constraint refuses.
func TestUnchecked(t *testing.T) {
	resp, src := gen(t, `package shop

type Small: int where { min(1) max(3) }

type Row {
  name: string where { min(1) }
  big: int where { length(2) }
  pick: string where { oneOf("ab", "abcd") length(..3) }
  num: int where { oneOf(1, 5) max(4) }
  bad: int where { min(1) } = 0
  worse: Small = 7
  later: Later where { min(1) }
}

type Later: int
`)
	contains(t, src, `pick: z.enum(["ab"]),`, `num: z.literal([1]),`, `bad: RowBad,`, `worse: Small,`, `get later(): typeof Later {`)
	absent(t, src, "RowName", "RowBig", "RowLater")
	for _, w := range []struct {
		code  string
		words []string
	}{
		{emit.LossConstraint, []string{"Row.name: min(1) is not checked", "only on a number"}},
		{emit.LossConstraint, []string{"Row.big: length(2) is not checked"}},
		{emit.LossConstraint, []string{`Row.pick: oneOf argument "abcd" is excluded by length(..3)`}},
		{emit.LossConstraint, []string{"Row.num: oneOf argument 5 is excluded by max(4)"}},
		{emit.LossDefault, []string{"Row.bad's default 0", "min(1)"}},
		{emit.LossDefault, []string{"Row.worse's default 7", "Small requires"}},
		{emit.LossConstraint, []string{"Row.later: min(1) is not checked", "written later"}},
	} {
		if !warned(resp, w.code, w.words...) {
			t.Errorf("no %s warning mentions %v: %v", w.code, w.words, resp.GetDiagnostics())
		}
	}
	typecheck(t, nil, resp)
}

// The newtype directive names a constrained field's schema.
func TestFieldSchemaName(t *testing.T) {
	resp, src := gen(t, `package shop

type Row {
  qty: int where { min(1) }
}

target zod for shop {
  Row.qty => newtype("Quantity")
}
`)
	quiet(t, resp)
	contains(t, src, `export const Quantity = z.int().min(1, "Row.qty: min(1)").brand<"shop.Row.qty">();`, `qty: Quantity,`)
	typecheck(t, nil, resp)
}
