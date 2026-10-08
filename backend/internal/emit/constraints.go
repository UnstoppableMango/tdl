package emit

import (
	"errors"
	"fmt"
	"regexp/syntax"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/ir"
)

// Length reads a `length` constraint's argument as inclusive bounds: an
// integer is both, and a range leaves an open end nil. A length counts a
// string's code points, not its bytes or UTF-16 units, in every target.
func Length(l *ir.Literal) (low, high *int64, err error) {
	switch l.GetKind() {
	case ir.LiteralKind_LITERAL_KIND_INT:
		n, err := strconv.ParseInt(l.GetText(), 0, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("%s does not fit in an int64", l.GetText())
		}
		return &n, &n, nil
	case ir.LiteralKind_LITERAL_KIND_RANGE:
		r := l.GetRange()
		low, high = r.Low, r.High
		switch {
		case low == nil && high == nil:
			return nil, nil, errors.New("a range with neither end constrains nothing")
		case low != nil && high != nil && *low > *high:
			return nil, nil, fmt.Errorf("%d..%d can never hold", *low, *high)
		}
		return low, high, nil
	}
	return nil, nil, fmt.Errorf("its length is %s", ir.KindName(l.GetKind()))
}

// Pattern checks a `matches` pattern against RE2's syntax, which a target
// whose engine accepts every RE2 pattern with the same meaning reads too.
// A pattern RE2 refuses is reported at generation time rather than failing
// when the generated code loads.
func Pattern(l *ir.Literal) (*syntax.Regexp, error) {
	if l.GetKind() != ir.LiteralKind_LITERAL_KIND_REGEX {
		return nil, errors.New("it takes one pattern")
	}
	re, err := syntax.Parse(l.GetText(), syntax.Perl)
	if err != nil {
		return nil, fmt.Errorf("RE2 refuses the pattern: %v", err)
	}
	return re, nil
}

// ConstraintText is a constraint as TDL source writes it.
func ConstraintText(c *ir.Constraint) string {
	if len(c.GetArgs()) == 0 {
		return c.GetName()
	}
	args := make([]string, len(c.GetArgs()))
	for i, a := range c.GetArgs() {
		args[i] = LiteralText(a)
	}
	return c.GetName() + "(" + strings.Join(args, ", ") + ")"
}

// LiteralText is a literal as TDL source writes it.
func LiteralText(l *ir.Literal) string {
	switch l.GetKind() {
	case ir.LiteralKind_LITERAL_KIND_STRING:
		return strconv.Quote(l.GetText())
	case ir.LiteralKind_LITERAL_KIND_REGEX:
		return "/" + l.GetText() + "/"
	case ir.LiteralKind_LITERAL_KIND_LIST:
		items := make([]string, len(l.GetItems()))
		for i, item := range l.GetItems() {
			items[i] = LiteralText(item)
		}
		return "[" + strings.Join(items, ", ") + "]"
	case ir.LiteralKind_LITERAL_KIND_RANGE:
		var s string
		if r := l.GetRange(); r != nil {
			if r.Low != nil {
				s = strconv.FormatInt(*r.Low, 10)
			}
			s += ".."
			if r.High != nil {
				s += strconv.FormatInt(*r.High, 10)
			}
		}
		return s
	}
	return l.GetText()
}
