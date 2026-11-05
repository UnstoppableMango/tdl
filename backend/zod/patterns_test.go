package zod_test

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/zod"
)

const patternModel = `package shop

type Sku: string where { matches(/^[A-Z]{3}-\d+$/) }

type Code: Sku where { matches(/^ABC/) }

enum Payment {
  Card { last4: string where { matches(/^\d{4}$/) } }
  Cash
}

type LineItem {
  sku: Sku
  note: string? where { matches(/(?i)^ok/) matches(/\S$/) }
  label: string | null where { matches(/^\p{Greek}+$/) }
}
`

// Without the patterns directive, each matches warns and no schema runs a
// pattern.
func TestPatternsOff(t *testing.T) {
	resp, src := gen(t, patternModel)
	absent(t, src, ".regex(/", "Pattern")
	for _, words := range [][]string{
		{"Sku: matches(/^[A-Z]{3}-\\d+$/)", "patterns directive"},
		{"Code: matches(/^ABC/)"},
		{"Payment.Card.last4"},
		{"LineItem.note: matches(/(?i)^ok/)"},
		{"LineItem.note: matches(/\\S$/)"},
		{"LineItem.label"},
	} {
		if !warned(resp, emit.LossConstraint, words...) {
			t.Errorf("no warning mentions %v: %v", words, resp.GetDiagnostics())
		}
	}
	// Code inherits Sku's pattern, which Sku warns for.
	if n := len(resp.GetDiagnostics()); n != 6 {
		t.Errorf("%d diagnostics, want 6: %v", n, resp.GetDiagnostics())
	}
	typecheck(t, nil, resp)
}

// Under patterns, each matches is a module constant printed from RE2's
// parse, checked by .regex() on the string schema.
func TestPatternsOn(t *testing.T) {
	resp, src := gen(t, patternModel+`
target zod for shop {
  patterns
}
`)
	quiet(t, resp)
	contains(t, src,
		"// matches(/^[A-Z]{3}-\\d+$/)\nconst SkuPattern = /^[A-Z]{3}-[0-9]+$/u;\n",
		`export const Sku = z.string().regex(SkuPattern, "Sku: matches(/^[A-Z]{3}-\\d+$/)").brand<"shop.Sku">();`,
		"const CodePattern = /^ABC/u;",
		`export const Code = Sku.regex(CodePattern, "Code: matches(/^ABC/)").brand<"shop.Code">();`,
		"const PaymentCardLast4Pattern = /^[0-9]{4}$/u;",
		"last4: PaymentCardLast4,",
		"const LineItemNotePattern = /^[Oo][Kk\\u{212A}]/u;",
		"const LineItemNotePattern2 = /[^\\u{9}\\u{A}\\u{C}\\u{D} ]$/u;",
		`export const LineItemNote = z.string().regex(LineItemNotePattern, "LineItem.note: matches(/(?i)^ok/)").regex(LineItemNotePattern2, "LineItem.note: matches(/\\S$/)").brand<"shop.LineItem.note">();`,
		"note: LineItemNote.exactOptional(),",
		"label: LineItemLabel.nullable(),",
		"sku: Sku,")
	absent(t, src, "export const SkuPattern")
	typecheck(t, nil, resp)
	parse(t, `import { Sku, Code, LineItem, PaymentCard } from "./shop.zod.js";
accepts("Sku", Sku, '"ABC-12"');
refuses("Sku", Sku, '"abc-12"');
refuses("Sku", Sku, '"ABC-\\u0661\\u0662"');
accepts("Code", Code, '"ABC-1"');
refuses("Code", Code, '"ABD-1"');
refuses("Code inherits Sku's pattern", Code, '"ABC"');
accepts("PaymentCard", PaymentCard, '{"kind": "Card", "last4": "1234"}');
refuses("PaymentCard", PaymentCard, '{"kind": "Card", "last4": "12345"}');
accepts("LineItem", LineItem, '{"sku": "ABC-1", "label": null}');
accepts("the Kelvin sign folds to k", LineItem, '{"sku": "ABC-1", "note": "o\\u212a!", "label": "\\u03bb"}');
accepts("a no-break space is not RE2's \\s", LineItem, '{"sku": "ABC-1", "note": "ok\\u00a0", "label": null}');
refuses("a space is", LineItem, '{"sku": "ABC-1", "note": "ok ", "label": null}');
refuses("LineItem", LineItem, '{"sku": "ABC-1", "label": "abc"}');
`, resp)
}

// A pattern a backtracking engine takes exponential time on warns and is
// not checked, under patterns too.
func TestPatternsBacktrack(t *testing.T) {
	for _, p := range []string{`(a+)+$`, `(a|aa)+$`} {
		resp, src := gen(t, `package shop

type Word: string where { matches(/`+p+`/) }

target zod for shop {
  patterns
}
`)
		absent(t, src, "Pattern", ".regex(/")
		if !warned(resp, emit.LossConstraint, "Word: matches(/"+p+"/)", "exponential") {
			t.Errorf("%s: no warning: %v", p, resp.GetDiagnostics())
		}
	}
}

// A matches on a schema that is not a z.string() warns, as does a pattern
// RE2 refuses.
func TestPatternsUnchecked(t *testing.T) {
	resp, src := gen(t, `package shop

type Id: uuid where { matches(/^0/) }

type Thing {
  later: Later where { matches(/x/) }
  look: string where { matches(/(?=a)/) }
}

type Later: string

target zod for shop {
  patterns
}
`)
	absent(t, src, "Pattern")
	for _, words := range [][]string{
		{"Id: matches(/^0/)", "string schema"},
		{"Thing.later", "written later"},
		{"Thing.look", "RE2 refuses"},
	} {
		if !warned(resp, emit.LossConstraint, words...) {
			t.Errorf("no warning mentions %v: %v", words, resp.GetDiagnostics())
		}
	}
	typecheck(t, nil, resp)
}

// A default the pattern refuses warns and is not written, since Zod parses
// no default; on a newtype, its own pattern is checked.
func TestPatternDefaults(t *testing.T) {
	resp, src := gen(t, `package shop

type Sku: string where { matches(/^[A-Z]+$/) }

type Line {
  good: string where { matches(/^a/) } = "ab"
  bad: string where { matches(/^a/) } = "ba"
  sku: Sku = "abc"
  fine: Sku = "ABC"
}

target zod for shop {
  patterns
}
`)
	contains(t, src,
		`good: LineGood.prefault("ab"),`,
		`bad: LineBad,`,
		`sku: Sku,`,
		`fine: Sku.prefault("ABC"),`)
	if !warned(resp, emit.LossDefault, "Line.bad", "does not match matches(/^a/)") {
		t.Errorf("no warning for Line.bad: %v", resp.GetDiagnostics())
	}
	if !warned(resp, emit.LossDefault, "Line.sku", "does not match matches(/^[A-Z]+$/), which Sku requires") {
		t.Errorf("no warning for Line.sku: %v", resp.GetDiagnostics())
	}
	typecheck(t, nil, resp)
}

// A pattern constant takes a module name, so a declaration taking it
// collides.
func TestPatternNameCollision(t *testing.T) {
	resp := generate(t, zod.Backend{}, zod.Name, lower(t, `package shop

type Sku: string where { matches(/^A/) }

type SkuPattern {
  x: int
}

target zod for shop {
  patterns
}
`))
	diags := resp.GetDiagnostics()
	if len(diags) != 1 || !strings.Contains(diags[0].GetMessage(), "SkuPattern in TypeScript, and Sku already does") {
		t.Errorf("diagnostics = %v, want the collision", diags)
	}
}

// A oneOf argument a pattern refuses is left out, so the union states the
// pattern and runs no expression; a default is held to a pattern only
// when a schema checks it.
func TestPatternsNarrowed(t *testing.T) {
	resp, src := gen(t, `package shop

type Id: uuid where { matches(/^0/) }

type Row {
  pick: string where { oneOf("ab", "x1") matches(/^[a-z]+$/) }
  id: Id = "123e4567-e89b-12d3-a456-426614174000"
}

target zod for shop {
  patterns
}
`)
	contains(t, src, `pick: z.enum(["ab"]),`, `id: Id.prefault("123e4567-e89b-12d3-a456-426614174000"),`)
	absent(t, src, "RowPick", "Pattern")
	if !warned(resp, emit.LossConstraint, `Row.pick: oneOf argument "x1" is excluded by matches(/^[a-z]+$/)`) {
		t.Errorf("no exclusion warning: %v", resp.GetDiagnostics())
	}
	if warned(resp, emit.LossDefault) {
		t.Errorf("a default is held to a pattern no schema checks: %v", resp.GetDiagnostics())
	}
	typecheck(t, nil, resp)
}
