package sema

import (
	"strconv"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// standard describes the constraints the spec specifies. The set of names
// is open, so this is what the compiler checks rather than what it allows:
// a name that is not here passes through untouched, for a backend that
// knows what it means.
var standard = map[string]struct {
	min, max int // argument count; max -1 for any number
	kinds    []ir.LiteralKind
}{
	"min":     {1, 1, []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_INT, ir.LiteralKind_LITERAL_KIND_FLOAT}},
	"max":     {1, 1, []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_INT, ir.LiteralKind_LITERAL_KIND_FLOAT}},
	"length":  {1, 1, []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_RANGE, ir.LiteralKind_LITERAL_KIND_INT}},
	"matches": {1, 1, []ir.LiteralKind{ir.LiteralKind_LITERAL_KIND_REGEX}},
	"oneOf":   {1, -1, nil},
	"unique":  {0, 0, nil},
}

// constraints lowers a `where { ... }` block and checks the standard names.
func (l *lowerer) constraints(in []*ast.Constraint) []*ir.Constraint {
	var out []*ir.Constraint
	for _, c := range in {
		lowered := &ir.Constraint{
			Name:     c.N,
			Position: position(c.P),
		}
		for _, a := range c.Args {
			lowered.Args = append(lowered.Args, l.literal(a))
		}
		l.checkStandard(c, lowered)
		out = append(out, lowered)
	}
	return out
}

// checkStandard reports a standard constraint used with the wrong number or
// kind of arguments. It says nothing about any other name.
func (l *lowerer) checkStandard(src *ast.Constraint, c *ir.Constraint) {
	spec, known := standard[c.GetName()]
	if !known {
		return
	}

	n := len(c.GetArgs())
	switch {
	case spec.max < 0 && n < spec.min:
		l.diags.add(src.P, "%s takes at least %d argument%s, got %d", c.GetName(), spec.min, plural(spec.min), n)
		return
	case spec.max >= 0 && (n < spec.min || n > spec.max):
		l.diags.add(src.P, "%s takes %d argument%s, got %d", c.GetName(), spec.min, plural(spec.min), n)
		return
	}

	if spec.kinds == nil {
		return
	}
	for _, arg := range c.GetArgs() {
		if !allowed(arg.GetKind(), spec.kinds) {
			l.diags.add(positionOf(arg.GetPosition()), "%s does not take %s", c.GetName(), kindName(arg.GetKind()))
		}
	}
}

func allowed(k ir.LiteralKind, kinds []ir.LiteralKind) bool {
	for _, want := range kinds {
		if k == want {
			return true
		}
	}
	return false
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func kindName(k ir.LiteralKind) string {
	switch k {
	case ir.LiteralKind_LITERAL_KIND_STRING:
		return "a string"
	case ir.LiteralKind_LITERAL_KIND_INT:
		return "an integer"
	case ir.LiteralKind_LITERAL_KIND_FLOAT:
		return "a float"
	case ir.LiteralKind_LITERAL_KIND_BOOL:
		return "a boolean"
	case ir.LiteralKind_LITERAL_KIND_NAME:
		return "a name"
	case ir.LiteralKind_LITERAL_KIND_REGEX:
		return "a regex"
	case ir.LiteralKind_LITERAL_KIND_LIST:
		return "a list"
	case ir.LiteralKind_LITERAL_KIND_RANGE:
		return "a range"
	}
	return "that"
}

// literal lowers a literal value.
func (l *lowerer) literal(lit *ast.Literal) *ir.Literal {
	if lit == nil {
		return nil
	}

	out := &ir.Literal{Text: lit.Text, Position: position(lit.P)}
	switch lit.Kind {
	case ast.LitString:
		out.Kind = ir.LiteralKind_LITERAL_KIND_STRING
	case ast.LitInt:
		out.Kind = ir.LiteralKind_LITERAL_KIND_INT
	case ast.LitFloat:
		out.Kind = ir.LiteralKind_LITERAL_KIND_FLOAT
	case ast.LitBool:
		out.Kind = ir.LiteralKind_LITERAL_KIND_BOOL
	case ast.LitName:
		out.Kind = ir.LiteralKind_LITERAL_KIND_NAME
	case ast.LitRegex:
		out.Kind = ir.LiteralKind_LITERAL_KIND_REGEX
	case ast.LitList:
		out.Kind = ir.LiteralKind_LITERAL_KIND_LIST
		for _, item := range lit.Items {
			out.Items = append(out.Items, l.literal(item))
		}
	case ast.LitRange:
		out.Kind = ir.LiteralKind_LITERAL_KIND_RANGE
		out.Text = ""
		out.Range = &ir.Range{Low: bound(l, lit.Lo), High: bound(l, lit.Hi)}
	}
	return out
}

// bound parses one end of a range, which is absent when the range is open
// on that side.
func bound(l *lowerer, lit *ast.Literal) *int64 {
	if lit == nil {
		return nil
	}
	n, err := strconv.ParseInt(lit.Text, 10, 64)
	if err != nil {
		l.diags.add(lit.P, "range bound out of range: %s", lit.Text)
		return nil
	}
	return &n
}
