package emit

import (
	"math"
	"strconv"

	"github.com/unstoppablemango/tdl/ir"
)

// MaxNarrow is the most values an integer `min`/`max` pair narrows to a
// literal union, and the longest list an exact `length` narrows to a
// tuple. A wider range or a longer list keeps its type, and warns.
const MaxNarrow = 16

// integers are the primitives a `min`/`max` pair narrows.
var integers = map[string]bool{"int": true, "int32": true, "int64": true, "uint32": true, "uint64": true}

// jsonNumbers are the primitives JSON writes as a number.
var jsonNumbers = map[string]bool{"int": true, "int32": true, "int64": true, "uint32": true, "uint64": true, "float32": true, "float64": true}

// NarrowKind is what a narrowed type is.
type NarrowKind int

const (
	// NarrowStrings is a union of string literals.
	NarrowStrings NarrowKind = iota
	// NarrowNumbers is a union of number literals.
	NarrowNumbers
	// NarrowVariants is a union of some of a fieldless enum's variants.
	NarrowVariants
	// NarrowTuple is a list of exactly [Narrowed.Len] elements.
	NarrowTuple
)

// Narrowed is a narrowed type: a constraint a target with literal and
// tuple types states as a type (CONTEXT.md).
type Narrowed struct {
	Kind NarrowKind
	// Values is a union's members: each string's text, each number as
	// JSON writes it, or each variant's name.
	Values []string
	// Len is a tuple's length. Elem is a tuple's element, or the enum a
	// union of variants narrows.
	Len  int
	Elem *Ref
	// Enforced is each own constraint the narrowed type states, which
	// therefore does not warn.
	Enforced []*ir.Constraint
}

// Narrow is the narrowed type own spells exactly on a value of r, or nil
// when it spells none. inherited holds the constraints r's newtype
// carries, which bound the narrowed type without being stated by it; with
// no own constraint, r keeps its name.
//
// A `oneOf` narrows a string, a number, or a fieldless enum to its values.
// An integer bounded by both `min` and `max`, with at most [MaxNarrow]
// values between them, narrows to every one. A list with an exact
// `length` from 1 to [MaxNarrow] narrows to a tuple.
func (s *Session) Narrow(r *Ref, own, inherited []*ir.Constraint) *Narrowed {
	if len(own) == 0 {
		return nil
	}
	x, err := s.Expand(r)
	if err != nil {
		return nil
	}

	all := append(append([]*ir.Constraint{}, own...), inherited...)
	n := &Narrowed{}
	enforce := func(names ...string) {
		for _, c := range own {
			for _, name := range names {
				if c.GetName() == name {
					n.Enforced = append(n.Enforced, c)
				}
			}
		}
	}

	if x.Form == List {
		return tuple(x, all, n, enforce)
	}

	lo, hi, bounded := bounds(all)
	var oneOf *ir.Constraint
	for _, c := range all {
		if c.GetName() == "oneOf" {
			if oneOf != nil {
				return nil // two lists; their intersection is not computed
			}
			oneOf = c
		}
	}

	if oneOf != nil {
		inRange, ok := n.literals(x, oneOf.GetArgs(), lo, hi)
		if !ok {
			return nil
		}
		enforce("oneOf")
		if inRange {
			enforce("min", "max")
		}
		return n
	}

	if !bounded || x.Form != Prim || !integers[x.Name] || hi < lo || hi-lo >= MaxNarrow {
		return nil
	}
	n.Kind = NarrowNumbers
	for v := lo; v <= hi; v++ {
		n.Values = append(n.Values, strconv.FormatInt(v, 10))
	}
	enforce("min", "max")
	return n
}

// GetEnforced is n's Enforced, or nil for a nil n.
func (n *Narrowed) GetEnforced() []*ir.Constraint {
	if n == nil {
		return nil
	}
	return n.Enforced
}

// tuple narrows a list with an exact `length` among cs.
func tuple(x *Ref, cs []*ir.Constraint, n *Narrowed, enforce func(...string)) *Narrowed {
	length := -1
	for _, c := range cs {
		if c.GetName() != "length" || len(c.GetArgs()) != 1 {
			continue
		}
		low, high, err := Length(c.GetArgs()[0])
		if err != nil || low == nil || high == nil || *low != *high {
			return nil
		}
		if length >= 0 && int64(length) != *low {
			return nil
		}
		length = int(*low)
	}
	if length < 1 || length > MaxNarrow {
		return nil
	}
	n.Kind, n.Len, n.Elem = NarrowTuple, length, x.Elem
	enforce("length")
	return n
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
		v, err := strconv.ParseInt(args[0].GetText(), 10, 64)
		if err != nil {
			continue
		}
		switch c.GetName() {
		case "min":
			lo, hasLo = max(lo, v), true
		case "max":
			hi, hasHi = min(hi, v), true
		}
	}
	return lo, hi, hasLo && hasHi
}

// literals reads each `oneOf` value as a literal of what x holds in JSON
// into n, and reports whether every one lies within [lo, hi]. ok is false
// when a value has no literal of that type.
func (n *Narrowed) literals(x *Ref, args []*ir.Literal, lo, hi int64) (inRange, ok bool) {
	switch {
	case x.Form == Prim && x.Name == "string":
		n.Kind = NarrowStrings
	case x.Form == Prim && jsonNumbers[x.Name]:
		n.Kind = NarrowNumbers
	case x.Form == Named && x.Decl.GetEnumeration() != nil && !Fielded(x.Decl.GetEnumeration()):
		n.Kind, n.Elem = NarrowVariants, x
	default:
		return false, false
	}

	inRange = true
	for _, a := range args {
		switch kind := a.GetKind(); {
		case n.Kind == NarrowStrings && kind == ir.LiteralKind_LITERAL_KIND_STRING,
			n.Kind == NarrowVariants && kind == ir.LiteralKind_LITERAL_KIND_NAME:
			n.Values = append(n.Values, a.GetText())
			inRange = false
		case n.Kind == NarrowNumbers && (kind == ir.LiteralKind_LITERAL_KIND_INT || kind == ir.LiteralKind_LITERAL_KIND_FLOAT):
			f, err := strconv.ParseFloat(a.GetText(), 64)
			if err != nil || (kind == ir.LiteralKind_LITERAL_KIND_FLOAT && integers[x.Name]) {
				return false, false
			}
			n.Values = append(n.Values, strconv.FormatFloat(f, 'f', -1, 64))
			inRange = inRange && f >= float64(lo) && f <= float64(hi)
		default:
			return false, false
		}
	}
	return inRange, len(n.Values) > 0
}
