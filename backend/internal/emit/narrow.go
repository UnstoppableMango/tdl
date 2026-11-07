package emit

import (
	"math"
	"slices"
	"strconv"

	"github.com/unstoppablemango/tdl/ir"
)

// MaxNarrow is the most values an integer `min`/`max` pair narrows to a
// literal union, and the longest list an exact `length` narrows to a
// tuple. A wider range or a longer list keeps its type, and warns.
const MaxNarrow = 16

// Numerics maps each primitive JSON writes as a number to whether it is
// an integer, which a `min`/`max` pair narrows. Targets writing the JSON
// wire convention read their numeric primitives from it.
var Numerics = map[string]bool{
	"int": true, "int32": true, "int64": true, "uint32": true, "uint64": true,
	"float32": false, "float64": false,
}

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
//
// Each constraint the narrowed type states is recorded as [Session.Enforce]
// does, so it does not warn.
func (s *Session) Narrow(r *Ref, own, inherited []*ir.Constraint) *Narrowed {
	n := s.narrow(r, own, inherited)
	s.Enforce(n.GetEnforced()...)
	return n
}

func (s *Session) narrow(r *Ref, own, inherited []*ir.Constraint) *Narrowed {
	if len(own) == 0 {
		return nil
	}
	if s.foreign(r) {
		return nil
	}
	x, err := s.Expand(r)
	if err != nil || s.foreign(x) {
		return nil
	}

	all := append(append([]*ir.Constraint{}, own...), inherited...)
	n := &Narrowed{}
	enforce := func(names ...string) {
		for _, c := range own {
			if slices.Contains(names, c.GetName()) {
				n.Enforced = append(n.Enforced, c)
			}
		}
	}
	// enforceBounds enforces each own `min` and `max` bounds read.
	enforceBounds := func(read map[*ir.Constraint]bool) {
		for _, c := range own {
			if read[c] {
				n.Enforced = append(n.Enforced, c)
			}
		}
	}

	if x.Form == List {
		return stated(tuple(x, all, n, enforce))
	}

	lo, hi, bounded, read := bounds(all)
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
			enforceBounds(read)
		}
		return stated(n)
	}

	// hi-lo overflows int64 for a wide range, so the width is unsigned.
	if !bounded || x.Form != Prim || !Numerics[x.Name] || hi < lo || uint64(hi)-uint64(lo) >= MaxNarrow {
		return nil
	}
	n.Kind = NarrowNumbers
	for v := lo; ; v++ {
		n.Values = append(n.Values, strconv.FormatInt(v, 10))
		if v == hi {
			break // v++ would overflow at math.MaxInt64
		}
	}
	enforceBounds(read)
	return stated(n)
}

// stated is n when it states at least one own constraint, or nil: a
// narrowed type stating only inherited constraints would replace a
// newtype's name and gain nothing.
func stated(n *Narrowed) *Narrowed {
	if n == nil || len(n.Enforced) == 0 {
		return nil
	}
	return n
}

// foreign reports whether r names a declaration a `foreign` directive
// maps to another type, whose schema or type is not narrowed.
func (s *Session) foreign(r *Ref) bool {
	if r.Form != Named {
		return false
	}
	_, ok := s.Find(r.Decl.GetDirectives(), "foreign")
	return ok
}

// GetEnforced is n's Enforced, or nil for a nil n.
func (n *Narrowed) GetEnforced() []*ir.Constraint {
	if n == nil {
		return nil
	}
	return n.Enforced
}

// tuple narrows a list with an exact `length` among cs, which every
// other `length` among them, exact or a range, allows.
func tuple(x *Ref, cs []*ir.Constraint, n *Narrowed, enforce func(...string)) *Narrowed {
	type span struct{ low, high *int64 }
	var spans []span
	length := int64(-1)
	for _, c := range cs {
		if c.GetName() != "length" || len(c.GetArgs()) != 1 {
			continue
		}
		low, high, err := Length(c.GetArgs()[0])
		if err != nil {
			return nil
		}
		spans = append(spans, span{low, high})
		if low != nil && high != nil && *low == *high {
			if length >= 0 && length != *low {
				return nil
			}
			length = *low
		}
	}
	if length < 1 || length > MaxNarrow {
		return nil
	}
	for _, sp := range spans {
		if (sp.low != nil && length < *sp.low) || (sp.high != nil && length > *sp.high) {
			return nil
		}
	}
	// The tuple's length lies within every one, so it states them all.
	n.Kind, n.Len, n.Elem = NarrowTuple, int(length), x.Elem
	enforce("length")
	return n
}

// bounds reads the tightest integer `min` and `max` among cs. ok is false
// unless both are present. read holds each bound read: a float bound, or
// one past int64, is not, so nothing narrowed states it.
func bounds(cs []*ir.Constraint) (lo, hi int64, ok bool, read map[*ir.Constraint]bool) {
	lo, hi = math.MinInt64, math.MaxInt64
	read = map[*ir.Constraint]bool{}
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
			lo, hasLo, read[c] = max(lo, v), true, true
		case "max":
			hi, hasHi, read[c] = min(hi, v), true, true
		}
	}
	return lo, hi, hasLo && hasHi, read
}

// literals reads each `oneOf` value as a literal of what x holds in JSON
// into n, and reports whether every one lies within [lo, hi]. ok is false
// when a value has no literal of that type.
func (n *Narrowed) literals(x *Ref, args []*ir.Literal, lo, hi int64) (inRange, ok bool) {
	switch {
	case x.Form == Prim && x.Name == "string":
		n.Kind = NarrowStrings
	case x.Form == Prim && numeric(x.Name):
		n.Kind = NarrowNumbers
	case x.Form == Named && x.Decl.GetEnumeration() != nil && !Fielded(x.Decl.GetEnumeration()):
		n.Kind, n.Elem = NarrowVariants, x
	default:
		return false, false
	}

	inRange = true
	seen := map[string]bool{}
	add := func(v string) {
		if !seen[v] {
			seen[v] = true
			n.Values = append(n.Values, v)
		}
	}
	for _, a := range args {
		switch kind := a.GetKind(); {
		case n.Kind == NarrowStrings && kind == ir.LiteralKind_LITERAL_KIND_STRING,
			n.Kind == NarrowVariants && kind == ir.LiteralKind_LITERAL_KIND_NAME:
			add(a.GetText())
			inRange = false
		case n.Kind == NarrowNumbers && (kind == ir.LiteralKind_LITERAL_KIND_INT || kind == ir.LiteralKind_LITERAL_KIND_FLOAT):
			f, err := strconv.ParseFloat(a.GetText(), 64)
			if err != nil || (kind == ir.LiteralKind_LITERAL_KIND_FLOAT && Numerics[x.Name]) {
				return false, false
			}
			add(strconv.FormatFloat(f, 'f', -1, 64))
			inRange = inRange && f >= float64(lo) && f <= float64(hi)
		default:
			return false, false
		}
	}
	return inRange, len(n.Values) > 0
}

// numeric reports whether JSON writes the primitive name as a number.
func numeric(name string) bool {
	_, ok := Numerics[name]
	return ok
}
