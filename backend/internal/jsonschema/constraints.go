package jsonschema

import (
	"encoding/json"
	"strconv"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// kind is the JSON type a constraint is checked against: what the
// constrained type is once newtypes are expanded.
type kind int

const (
	other kind = iota
	numeric
	text
	array
	dict
	// fieldless is an enum whose variants carry no fields, a string in JSON.
	fieldless
)

// kindOf reads what a reference holds, through any newtype and any
// optionality.
func (b *Builder) kindOf(r *emit.Ref) kind {
	for r.Form == emit.Option || r.Form == emit.Nullable {
		r = r.Elem
	}
	x, err := b.Expand(r)
	if err != nil {
		return other
	}
	switch x.Form {
	case emit.Prim:
		switch scalars[x.Name].typ {
		case "integer", "number":
			return numeric
		case "string":
			return text
		}
	case emit.List, emit.Set:
		return array
	case emit.Map:
		return dict
	case emit.Named:
		if e := x.Decl.GetEnumeration(); e != nil && !emit.Fielded(e) {
			return fieldless
		}
	}
	return other
}

// constrain writes each constraint as the keyword that checks it on a value
// of kind k. One JSON Schema cannot say is a warning, and the schema is
// still written without it.
func (b *Builder) constrain(s *Object, k kind, owner string, cs []*ir.Constraint) {
	for _, c := range cs {
		if err := b.constraint(s, k, c); err != nil {
			b.Warn(emit.Unsupported(c.GetPosition(), "%s: %v", owner, err))
		}
	}
}

type constraintError string

func (e constraintError) Error() string { return string(e) }

func (b *Builder) constraint(s *Object, k kind, c *ir.Constraint) error {
	name, args := c.GetName(), c.GetArgs()
	misplaced := constraintError(name + " has no JSON Schema keyword for this type")

	switch name {
	case "min", "max":
		if k != numeric || len(args) != 1 {
			return misplaced
		}
		n, ok := number(args[0])
		if !ok {
			return constraintError(name + " takes a number")
		}
		s.Set(map[string]string{"min": "minimum", "max": "maximum"}[name], n)
	case "length":
		var lo, hi string
		switch k {
		case text:
			lo, hi = "minLength", "maxLength"
		case array:
			lo, hi = "minItems", "maxItems"
		case dict:
			lo, hi = "minProperties", "maxProperties"
		default:
			return misplaced
		}
		if len(args) != 1 {
			return constraintError("length takes one argument")
		}
		switch a := args[0]; a.GetKind() {
		case ir.LiteralKind_LITERAL_KIND_INT:
			n, err := strconv.ParseInt(a.GetText(), 10, 64)
			if err != nil {
				return constraintError("length takes an integer")
			}
			s.Set(lo, n).Set(hi, n)
		case ir.LiteralKind_LITERAL_KIND_RANGE:
			if r := a.GetRange(); r.Low != nil {
				s.Set(lo, r.GetLow())
			}
			if r := a.GetRange(); r.High != nil {
				s.Set(hi, r.GetHigh())
			}
		default:
			return constraintError("length takes an integer or a range")
		}
	case "matches":
		if k != text || len(args) != 1 || args[0].GetKind() != ir.LiteralKind_LITERAL_KIND_REGEX {
			return misplaced
		}
		s.Set("pattern", args[0].GetText())
	case "oneOf":
		if k != numeric && k != text && k != fieldless {
			return misplaced
		}
		values := make([]any, 0, len(args))
		for _, a := range args {
			v, ok := literal(a)
			if !ok {
				return constraintError("oneOf takes literal values")
			}
			values = append(values, v)
		}
		s.Set("enum", values)
	case "unique":
		if k != array {
			return misplaced
		}
		s.Set("uniqueItems", true)
	default:
		return constraintError(name + " is not a constraint JSON Schema knows")
	}
	return nil
}

// number is a numeric literal in JSON's spelling, which has no leading
// zeros.
func number(l *ir.Literal) (json.Number, bool) {
	switch l.GetKind() {
	case ir.LiteralKind_LITERAL_KIND_INT:
		if n, err := strconv.ParseInt(l.GetText(), 10, 64); err == nil {
			return json.Number(strconv.FormatInt(n, 10)), true
		}
	case ir.LiteralKind_LITERAL_KIND_FLOAT:
		if f, err := strconv.ParseFloat(l.GetText(), 64); err == nil {
			return json.Number(strconv.FormatFloat(f, 'f', -1, 64)), true
		}
	}
	return "", false
}

// literal is a constraint argument as a JSON value. A name is an enum
// variant, which is its name in JSON.
func literal(l *ir.Literal) (any, bool) {
	switch l.GetKind() {
	case ir.LiteralKind_LITERAL_KIND_STRING, ir.LiteralKind_LITERAL_KIND_NAME:
		return l.GetText(), true
	case ir.LiteralKind_LITERAL_KIND_BOOL:
		return l.GetText() == "true", true
	}
	if n, ok := number(l); ok {
		return n, true
	}
	return nil, false
}
