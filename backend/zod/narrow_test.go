package zod_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
)

const narrowModel = `package shop

enum Status {
  Draft
  Placed
  Shipped
}

type Grade: string where { oneOf("A", "B") }

type Small: int where { min(1) max(3) }

type Box {
  currency: string where { oneOf("USD", "EUR") }
  ratio: float64 where { oneOf(0.5, 2) }
  step: int where { min(1) max(4) }
  open: Status where { oneOf(Draft, Placed) }
  dims: [float64] where { length(3) }
  corners: [Corner]? where { length(2) }
  grade: Grade
  small: Small
  tinier: Small where { max(2) }
  wide: int where { min(0) max(16) }
  later: Later where { oneOf(Low) }
}

type Corner {
  x: float64
}

enum Later {
  Low
  High
}
`

// The constraints TypeScript states as a type are schemas inferring that
// type: literal unions and tuples, as the typescript backend writes them.
func TestNarrowedTypes(t *testing.T) {
	resp, src := gen(t, narrowModel)
	contains(t, src,
		`export const Grade = z.enum(["A", "B"]);`,
		`export const Small = z.literal([1, 2, 3]);`,
		`currency: z.enum(["USD", "EUR"]),`,
		`ratio: z.literal([0.5, 2]),`,
		`step: z.literal([1, 2, 3, 4]),`,
		`open: Status.extract(["Draft", "Placed"]),`,
		`dims: z.tuple([z.number(), z.number(), z.number()]),`,
		`get corners(): z.ZodExactOptional<z.ZodTuple<[typeof Corner, typeof Corner], null>> {`,
		`grade: Grade,`,
		`small: Small,`,
		`tinier: z.literal([1, 2]),`,
		`wide: z.int(),`,
		`get later(): z.ZodEnum<{ "Low": "Low" }> {`)
	if !warned(resp, emit.LossConstraint, "Box.wide") || len(resp.GetDiagnostics()) != 1 {
		t.Errorf("diagnostics = %v, want one for Box.wide", resp.GetDiagnostics())
	}
	typecheck(t, nil, resp)
	parse(t, `import { Box } from "./shop.zod.js";
const ok = '{"currency": "USD", "ratio": 2, "step": 4, "open": "Placed", "dims": [1, 2, 3], "grade": "A", "small": 3, "tinier": 2, "wide": 9, "later": "Low"}';
accepts("Box", Box, ok);
for (const [key, value] of [["currency", '"GBP"'], ["ratio", "1"], ["step", "5"], ["open", '"Shipped"'], ["dims", "[1, 2]"], ["dims", "[1, 2, 3, 4]"], ["tinier", "3"], ["later", '"High"']]) {
  const bad = JSON.parse(ok);
  bad[key] = JSON.parse(value);
  refuses("Box." + key + " = " + value, Box, JSON.stringify(bad));
}
`, resp)
	rejects(t, `import type { Box } from "./shop.zod.js";
const b: Box["currency"] = "GBP";
`, resp)
	rejects(t, `import type { Box } from "./shop.zod.js";
const d: Box["dims"] = [1, 2];
`, resp)
}

// A default outside the narrowed type warns and is not written, since Zod
// does not parse a default.
func TestNarrowedDefaults(t *testing.T) {
	resp, src := gen(t, `package shop

type Box {
  in: string where { oneOf("a", "b") } = "b"
  out: string where { oneOf("a", "b") } = "c"
  step: int where { min(1) max(3) } = 4
  dims: [int] where { length(2) } = [1, 2]
  short: [int] where { length(2) } = [1]
}
`)
	contains(t, src,
		`in: z.enum(["a", "b"]).default("b"),`,
		`out: z.enum(["a", "b"]),`,
		`step: z.literal([1, 2, 3]),`,
		`dims: z.tuple([z.int(), z.int()]).default([1, 2]),`,
		`short: z.tuple([z.int(), z.int()]),`)
	for _, field := range []string{"Box.out", "Box.step", "Box.short"} {
		if !warned(resp, emit.LossDefault, field) {
			t.Errorf("no default warning for %s: %v", field, resp.GetDiagnostics())
		}
	}
	typecheck(t, nil, resp)
}
