package jsonschema

import (
	"encoding/json"
	"errors"
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
		err := b.constraint(s, k, owner, c)
		switch lost, ok := errors.AsType[lostError](err); {
		case ok:
			b.Lossy(emit.LossConstraint, c.GetPosition(), "%s: %s is not written: %s", owner, emit.ConstraintText(c), string(lost))
		case err != nil:
			b.Warn(emit.Unsupported(c.GetPosition(), "%s: %v", owner, err))
		}
	}
}

type constraintError string

func (e constraintError) Error() string { return string(e) }

// lostError is why a constraint JSON Schema has a keyword for is not
// written: a pattern no validator should run.
type lostError string

func (e lostError) Error() string { return string(e) }

func (b *Builder) constraint(s *Object, k kind, owner string, c *ir.Constraint) error {
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
		low, high, err := emit.Length(args[0])
		if err != nil {
			return constraintError("length " + err.Error())
		}
		if low != nil {
			s.Set(lo, *low)
		}
		if high != nil {
			s.Set(hi, *high)
		}
	case "matches":
		if k != text || len(args) != 1 || args[0].GetKind() != ir.LiteralKind_LITERAL_KIND_REGEX {
			return misplaced
		}
		return b.pattern(s, owner, c)
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

// pattern writes a `matches` constraint as `pattern`, printed by
// [emit.ECMAScriptSource] so that a validator reading it as ECMA-262 with
// the u flag matches what RE2 matches. A pattern RE2 refuses, one naming
// a surrogate, and one whose shape a backtracking engine takes
// exponential time on are not written. Under a dialect that says nothing
// of the flag, a pattern reading differently without it warns.
func (b *Builder) pattern(s *Object, owner string, c *ir.Constraint) error {
	re, err := emit.Pattern(c.GetArgs()[0])
	switch {
	case err != nil:
		return lostError(err.Error())
	case emit.Surrogates(re):
		return lostError("it names a surrogate code point, which RE2 never sees and a validator can match")
	case emit.Backtracks(re):
		return lostError("validators commonly backtrack, and can take exponential time on this pattern's shape")
	}
	s.Set("pattern", emit.ECMAScriptSource(re))
	if !b.Dialect.Unicode && emit.NeedsUnicode(re) {
		b.Warn(emit.Unsupported(c.GetPosition(),
			"%s: %s is written for Unicode mode, ECMA-262's u flag, which this dialect leaves to the validator; one without it reads the pattern differently",
			owner, emit.ConstraintText(c)))
	}
	return nil
}
