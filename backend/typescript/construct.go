package typescript

import (
	"fmt"
	re2 "regexp/syntax"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/ts"
	"github.com/unstoppablemango/tdl/ir"
)

// Under a `construct` directive, which `brand` turns on, a newtype has a
// constructor: a function with the type's own name taking its base and
// returning it cast to the newtype, so a call site builds a branded value
// without a cast. It checks at runtime only what the type cannot state: a
// constraint a narrowed type states is the parameter's type, and every
// other constraint it can check, its own and inherited, throws a
// RangeError naming the newtype and the constraint. A `matches` is checked
// only under the `patterns` directive, as in zod.

// constructed reports whether a newtype has a constructor: by its own
// construct or brand directive, or the target block's.
func (g *generator) constructed(d *ir.Decl) bool {
	return g.construct || g.branded(d) || g.Tagged(d.GetDirectives(), "construct")
}

// unbindable is the names no constructor takes: the globals its checks
// refer to, which it would shadow, and eval and arguments, which a module
// cannot bind.
var unbindable = map[string]bool{"RangeError": true, "Set": true, "Object": true, "eval": true, "arguments": true}

// constructor writes a newtype's constructor taking param, its base as
// written, narrowed where its constraints narrow it, and records each
// constraint it checks. A constraint it cannot check warns why.
func (g *generator) constructor(b *strings.Builder, d *ir.Decl, name, param string) error {
	subject := emit.LastSegment(d.GetMeta().GetName())
	if unbindable[name] {
		g.Warn(emit.Unsupported(d.GetMeta().GetPosition(), "%s would shadow %s in TypeScript, so it has no constructor", subject, name))
		return nil
	}
	ref, err := g.Resolve(d.GetNewtype().GetBase())
	if err != nil {
		return err
	}
	x, nullable, err := g.underlying(ref)
	if err != nil {
		return err
	}

	// A branded base is made only by its own constructor, which checks
	// every constraint the newtype inherits.
	viaBrand := ref.Form == emit.Named && ref.Decl.GetNewtype() != nil && g.branded(ref.Decl)

	var checks []string
	for _, c := range d.GetNewtype().GetValueConstraints() {
		if viaBrand && c.GetFrom() != nil {
			g.Enforce(c)
			continue
		}
		// Skip what the narrowed type states, and what already warned it
		// cannot be checked: a second rendering, under roundtrip, writes
		// the same checks and warns nothing again.
		if g.unchecked[c] || (g.Enforced(c) && !g.checked[c]) {
			continue
		}
		text := emit.ConstraintText(c)
		cond, why := g.condition(x, c)
		g.Enforce(c)
		if why != "" {
			g.unchecked[c] = true
			g.Lossy(emit.LossConstraint, c.GetPosition(), "%s: %s is not checked: %s", subject, text, why)
			continue
		}
		g.checked[c] = true
		if nullable {
			cond = "v !== null && (" + cond + ")"
		}
		checks = append(checks, fmt.Sprintf("  if (%s) {\n    throw new RangeError(%s);\n  }\n", cond, strconv.Quote(subject+": "+text)))
	}

	if len(checks) == 0 {
		fmt.Fprintf(b, "export const %s = (v: %s): %s => v as %s;\n", name, param, name, name)
		return nil
	}
	fmt.Fprintf(b, "export const %s = (v: %s): %s => {\n", name, param, name)
	for _, c := range checks {
		b.WriteString(c)
	}
	fmt.Fprintf(b, "  return v as %s;\n};\n", name)
	return nil
}

// underlying is what a newtype's value holds once every newtype is
// followed and null is taken off, and whether it may be null.
func (g *generator) underlying(r *emit.Ref) (*emit.Ref, bool, error) {
	nullable := false
	for {
		x, err := g.Expand(r)
		if err != nil {
			return nil, false, err
		}
		if x.Form != emit.Option && x.Form != emit.Nullable {
			return x, nullable, nil
		}
		nullable = true
		r = x.Elem
	}
}

// condition is the expression over v that holds when v fails c, a
// constraint on a value of x, or why a constructor cannot check it.
func (g *generator) condition(x *emit.Ref, c *ir.Constraint) (string, string) {
	args := c.GetArgs()
	scalar := ""
	if x.Form == emit.Prim {
		scalar = ts.Scalars[x.Name]
	}

	switch c.GetName() {
	case "min", "max":
		if scalar != "number" {
			return "", "a constructor checks min and max only on a number"
		}
		if c.GetName() == "min" {
			return "v < " + args[0].GetText(), ""
		}
		return "v > " + args[0].GetText(), ""
	case "length":
		low, high, err := emit.Length(args[0])
		if err != nil {
			return "", err.Error()
		}
		var n string
		switch {
		case scalar == "string" && x.Name != "bytes":
			// TDL counts code points, which a string's iterator yields.
			n = "[...v].length"
		case x.Form == emit.List || x.Form == emit.Set:
			n = "v.length"
		case x.Form == emit.Map:
			n = "Object.keys(v).length"
		default:
			return "", "a constructor checks a length only on a string, a list, a set, or a map"
		}
		return bounds(n, low, high), ""
	case "unique":
		if x.Form != emit.List && x.Form != emit.Set {
			return "", "a constructor checks unique only on a list or a set"
		}
		if !g.primitive(x.Elem) {
			return "", "a constructor compares elements with a Set, which compares an object by reference"
		}
		return "new Set(v).size !== v.length", ""
	case "matches":
		if scalar != "string" || x.Name == "bytes" {
			return "", "a constructor checks a pattern only on a string"
		}
		p, why := g.checkable(c)
		if why != "" {
			return "", why
		}
		return "!/" + emit.ECMAScript(p) + "/u.test(v)", ""
	}
	return "", "a constructor gives " + c.GetName() + " no check"
}

// bounds is the expression that holds when n is outside low..high.
func bounds(n string, low, high *int64) string {
	switch {
	case low != nil && high != nil && *low == *high:
		return fmt.Sprintf("%s !== %d", n, *low)
	case low != nil && high != nil:
		return fmt.Sprintf("%s < %d || %s > %d", n, *low, n, *high)
	case low != nil:
		return fmt.Sprintf("%s < %d", n, *low)
	case high != nil:
		return fmt.Sprintf("%s > %d", n, *high)
	}
	return "false"
}

// primitive reports whether a collection's element is a JavaScript
// primitive, which a Set compares by value: a scalar, a fieldless enum's
// string, or null.
func (g *generator) primitive(r *emit.Ref) bool {
	x, _, err := g.underlying(r)
	if err != nil {
		return false
	}
	switch x.Form {
	case emit.Prim:
		_, ok := ts.Scalars[x.Name]
		return ok
	case emit.Named:
		e := x.Decl.GetEnumeration()
		return e != nil && !emit.Fielded(e)
	}
	return false
}

// checkable is the pattern a `matches` constraint is checked against, or
// why it is not: without the patterns directive, a pattern RE2 refuses,
// one naming a surrogate, or one whose shape a backtracking engine takes
// exponential time on.
func (g *generator) checkable(c *ir.Constraint) (*re2.Regexp, string) {
	if !g.patterns {
		return nil, "a constructor checks a pattern only under the patterns directive"
	}
	re, err := emit.Pattern(c.GetArgs()[0])
	if err != nil {
		return nil, err.Error()
	}
	if emit.Surrogates(re) {
		return nil, "it names a surrogate code point, which RE2 never sees and JavaScript can match"
	}
	if emit.Backtracks(re) {
		return nil, "JavaScript's engine backtracks, and can take exponential time on this pattern's shape"
	}
	return re, ""
}
