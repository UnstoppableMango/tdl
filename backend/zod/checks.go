package zod

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/ts"
	"github.com/unstoppablemango/tdl/ir"
)

// A constraint TypeScript cannot state as a type is a check on the schema,
// and the schema is branded, so its inferred type is a checked type: a
// value that went through the check. A constrained newtype brands its own
// schema; a constrained field gets a schema of its own, named for its
// struct and field. docs/architecture/zod-backend.md's "Checked and
// branded" is the table.

// qualified is a declaration's or a field's name qualified by the model's
// package,
// which a brand carries so two packages' declarations stay distinct.
func (g *generator) qualified(name string) string {
	if pkg := g.Model.GetPackage(); pkg != "" && !strings.HasPrefix(name, pkg+".") {
		return pkg + "." + name
	}
	return name
}

// brand brands a schema with name, unless the target block is structural.
func (g *generator) brand(s schema, name string) schema {
	if g.structural {
		return s
	}
	lit := strconv.Quote(name)
	return schema{expr: s.expr + ".brand<" + lit + ">()", typ: "z.core.$ZodBranded<" + s.typ + ", " + lit + ">", nullable: s.nullable}
}

// exclude drops each value of a literal union another constraint among cs
// refuses, warning that it could never be held, and records the
// constraints the remaining values all satisfy as stated by the union.
func (g *generator) exclude(subject string, n *emit.Narrowed, own, cs []*ir.Constraint) {
	if n == nil || (n.Kind != emit.NarrowStrings && n.Kind != emit.NarrowNumbers) {
		return
	}
	var bounds []*ir.Constraint
	for _, c := range cs {
		switch c.GetName() {
		case "min", "max":
			if n.Kind == emit.NarrowNumbers {
				bounds = append(bounds, c)
			}
		case "length":
			if n.Kind == emit.NarrowStrings {
				bounds = append(bounds, c)
			}
		case "matches":
			// Filtered here, the union states the pattern, and runs no
			// expression.
			if _, why := g.checkable(c); n.Kind == emit.NarrowStrings && why == "" {
				g.regexChecked[c] = true
				bounds = append(bounds, c)
			}
		}
	}
	if len(bounds) == 0 {
		return
	}
	kept := n.Values[:0:0]
	for _, v := range n.Values {
		lit := &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_STRING, Text: v}
		shown := strconv.Quote(v)
		if n.Kind == emit.NarrowNumbers {
			lit = &ir.Literal{Kind: ir.LiteralKind_LITERAL_KIND_FLOAT, Text: v}
			shown = v
		}
		if c, _ := g.refusing(lit, bounds); c != nil {
			g.Lossy(emit.LossConstraint, c.GetPosition(), "%s: oneOf argument %s is excluded by %s, and is left out", subject, shown, emit.ConstraintText(c))
			continue
		}
		kept = append(kept, v)
	}
	n.Values = kept
	for _, c := range own {
		if slices.Contains(bounds, c) {
			g.enforced[c] = true
		}
	}
}

// checks is the Zod method calls stating each constraint in cs on a
// schema of x, an expanded reference, and records each it states. A
// constraint with no check on x warns. subject names the value in each
// check's message and each warning.
func (g *generator) checks(subject string, x *emit.Ref, cs []*ir.Constraint) string {
	var b strings.Builder
	for _, c := range cs {
		if g.enforced[c] {
			continue
		}
		text := emit.ConstraintText(c)
		msg := strconv.Quote(subject + ": " + text)
		call, why := g.check(x, c, msg)
		g.enforced[c] = true
		if why != "" {
			g.Lossy(emit.LossConstraint, c.GetPosition(), "%s: %s is not checked: %s", subject, text, why)
			continue
		}
		b.WriteString(call)
	}
	return b.String()
}

// check is one constraint's method call on a schema of x, or why it has
// none.
func (g *generator) check(x *emit.Ref, c *ir.Constraint, msg string) (string, string) {
	args := c.GetArgs()
	number := x.Form == emit.Prim && ts.Scalars[x.Name] == "number"
	str := x.Form == emit.Prim && ts.Scalars[x.Name] == "string" && x.Name != "bytes"
	list := x.Form == emit.List || x.Form == emit.Set

	switch c.GetName() {
	case "min", "max":
		if !number {
			return "", "Zod checks min and max only on a number"
		}
		return fmt.Sprintf(".%s(%s, %s)", c.GetName(), args[0].GetText(), msg), ""
	case "length":
		low, high, err := emit.Length(args[0])
		if err != nil {
			return "", err.Error()
		}
		switch {
		case str:
			g.helpers[helperCodePoints] = true
			return fmt.Sprintf(".refine(codePoints(%s, %s), %s)", bound(low, "0"), bound(high, "Infinity"), msg), ""
		case list:
			if low != nil && high != nil && *low == *high {
				return fmt.Sprintf(".length(%d, %s)", *low, msg), ""
			}
			var calls string
			if low != nil {
				calls += fmt.Sprintf(".min(%d, %s)", *low, msg)
			}
			if high != nil {
				calls += fmt.Sprintf(".max(%d, %s)", *high, msg)
			}
			return calls, ""
		case x.Form == emit.Map:
			g.helpers[helperSize] = true
			return fmt.Sprintf(".refine(size(%s, %s), %s)", bound(low, "0"), bound(high, "Infinity"), msg), ""
		}
		return "", "Zod checks a length only on a string, a list, a set, or a map"
	case "unique":
		switch x.Form {
		case emit.Set:
			// A set's schema already refuses a duplicate.
			return "", ""
		case emit.List:
			g.helpers[helperUnique] = true
			return ".refine(unique, " + msg + ")", ""
		}
		return "", "Zod checks unique only on a list or a set"
	case "oneOf":
		return "", "Zod states oneOf only on a string, a number, or a fieldless enum"
	case "matches":
		call, why := g.pattern(c, msg)
		if why == "" {
			g.regexChecked[c] = true
		}
		return call, why
	}
	return "", "Zod gives " + c.GetName() + " no meaning"
}

// bound is a length bound's text, or the open end's.
func bound(n *int64, open string) string {
	if n == nil {
		return open
	}
	return strconv.FormatInt(*n, 10)
}

// refusing is the first of cs a literal fails, and why, or nil. Zod
// parses no default and the backend writes no check for a literal union,
// so both are checked here; a pattern only when a schema checks it.
func (g *generator) refusing(l *ir.Literal, cs []*ir.Constraint) (*ir.Constraint, string) {
	for _, c := range cs {
		args := c.GetArgs()
		switch c.GetName() {
		case "min", "max":
			v, err1 := strconv.ParseFloat(l.GetText(), 64)
			b, err2 := strconv.ParseFloat(args[0].GetText(), 64)
			if err1 != nil || err2 != nil || l.GetKind() == ir.LiteralKind_LITERAL_KIND_STRING {
				continue
			}
			if c.GetName() == "min" && v < b || c.GetName() == "max" && v > b {
				return c, "is outside " + emit.ConstraintText(c)
			}
		case "length":
			low, high, err := emit.Length(args[0])
			if err != nil {
				continue
			}
			var n int64
			switch l.GetKind() {
			case ir.LiteralKind_LITERAL_KIND_STRING:
				n = int64(len([]rune(l.GetText())))
			case ir.LiteralKind_LITERAL_KIND_LIST:
				n = int64(len(l.GetItems()))
			default:
				continue
			}
			if low != nil && n < *low || high != nil && n > *high {
				return c, "fails " + emit.ConstraintText(c)
			}
		case "matches":
			if re, why := g.checkable(c); why == "" && g.regexChecked[c] && l.GetKind() == ir.LiteralKind_LITERAL_KIND_STRING && !re.MatchString(l.GetText()) {
				return c, "does not match " + emit.ConstraintText(c)
			}
		case "unique":
			seen := map[string]bool{}
			for _, item := range l.GetItems() {
				key := emit.LiteralText(item)
				if seen[key] {
					return c, "holds " + key + " twice, and " + emit.ConstraintText(c) + " refuses that"
				}
				seen[key] = true
			}
		}
	}
	return nil, ""
}
