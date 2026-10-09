package typescript

import (
	"math"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// maxRange is the most values an integer `min`/`max` pair narrows to a
// literal union. A wider range keeps `number` and warns.
const maxRange = 16

// integers are the primitives a `min`/`max` pair narrows.
var integers = map[string]bool{"int": true, "int32": true, "int64": true, "uint32": true, "uint64": true}

// narrow is the literal union that own spells exactly on a value of r, or
// "" when it spells none. inherited holds the constraints r's newtype
// carries, which bound the union without being enforced here; with no own
// constraint, r keeps its name. Each of own the union enforces is
// recorded, so it does not warn.
//
// A `oneOf` narrows a string, a number, or a fieldless enum to its values.
// An integer bounded by both `min` and `max`, with at most [maxRange]
// values between them, narrows to every one.
func (g *generator) narrow(r *emit.Ref, own, inherited []*ir.Constraint) string {
	if len(own) == 0 {
		return ""
	}
	x, err := g.Expand(r)
	if err != nil {
		return ""
	}

	all := append(append([]*ir.Constraint{}, own...), inherited...)
	lo, hi, bounds := bounds(all)

	var oneOf *ir.Constraint
	for _, c := range all {
		if c.GetName() == "oneOf" {
			if oneOf != nil {
				return "" // two lists; their intersection is not computed
			}
			oneOf = c
		}
	}

	enforce := func(names ...string) {
		for _, c := range own {
			for _, n := range names {
				if c.GetName() == n {
					g.enforced[c] = true
				}
			}
		}
	}

	if oneOf != nil {
		values, inRange, ok := literals(x, oneOf.GetArgs(), lo, hi)
		if !ok {
			return ""
		}
		enforce("oneOf")
		if inRange {
			enforce("min", "max")
		}
		return strings.Join(values, " | ")
	}

	if !bounds || x.Form != emit.Prim || !integers[x.Name] || hi < lo || hi-lo >= maxRange {
		return ""
	}
	values := make([]string, 0, hi-lo+1)
	for n := lo; n <= hi; n++ {
		values = append(values, strconv.FormatInt(n, 10))
	}
	enforce("min", "max")
	return strings.Join(values, " | ")
}

// bounds reads the tightest integer `min` and `max` among cs. ok is false
// unless both are present.
func bounds(cs []*ir.Constraint) (lo, hi int64, ok bool) {
	lo, hi = math.MinInt64, math.MaxInt64
	var hasLo, hasHi bool
	for _, c := range cs {
		args := c.GetArgs()
		if len(args) != 1 || args[0].GetKind() != ir.LiteralKind_LITERAL_KIND_INT {
			continue
		}
		n, err := strconv.ParseInt(args[0].GetText(), 10, 64)
		if err != nil {
			continue
		}
		switch c.GetName() {
		case "min":
			lo, hasLo = max(lo, n), true
		case "max":
			hi, hasHi = min(hi, n), true
		}
	}
	return lo, hi, hasLo && hasHi
}

// literals renders each `oneOf` value as the TypeScript literal of what x
// holds in JSON, and reports whether every one lies within [lo, hi]. ok is
// false when a value has no literal of that type.
func literals(x *emit.Ref, args []*ir.Literal, lo, hi int64) (values []string, inRange, ok bool) {
	var str, num, enum bool
	switch {
	case x.Form == emit.Prim && x.Name == "string":
		str = true
	case x.Form == emit.Prim && scalars[x.Name] == "number":
		num = true
	case x.Form == emit.Named && x.Decl.GetEnumeration() != nil && !emit.Fielded(x.Decl.GetEnumeration()):
		enum = true
	default:
		return nil, false, false
	}

	inRange = true
	for _, a := range args {
		switch kind := a.GetKind(); {
		case str && kind == ir.LiteralKind_LITERAL_KIND_STRING, enum && kind == ir.LiteralKind_LITERAL_KIND_NAME:
			values = append(values, strconv.Quote(a.GetText()))
			inRange = false
		case num && (kind == ir.LiteralKind_LITERAL_KIND_INT || kind == ir.LiteralKind_LITERAL_KIND_FLOAT):
			f, err := strconv.ParseFloat(a.GetText(), 64)
			if err != nil || (kind == ir.LiteralKind_LITERAL_KIND_FLOAT && integers[x.Name]) {
				return nil, false, false
			}
			values = append(values, strconv.FormatFloat(f, 'f', -1, 64))
			inRange = inRange && f >= float64(lo) && f <= float64(hi)
		default:
			return nil, false, false
		}
	}
	return values, inRange, len(values) > 0
}
