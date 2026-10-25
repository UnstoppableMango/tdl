package analysis

import (
	"errors"
	"fmt"
	"regexp/syntax"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
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

// Source is a constraint as TDL source writes it.
func Source(c *ir.Constraint) string {
	if len(c.GetArgs()) == 0 {
		return c.GetName()
	}
	args := make([]string, len(c.GetArgs()))
	for i, a := range c.GetArgs() {
		args[i] = LiteralSource(a)
	}
	return c.GetName() + "(" + strings.Join(args, ", ") + ")"
}

// LiteralSource is a literal as TDL source writes it.
func LiteralSource(l *ir.Literal) string { return l.Source() }

// constraintViews reads each constraint in cs.
func constraintViews(cs []*ir.Constraint) []*plugin.ConstraintView {
	var views []*plugin.ConstraintView
	for _, c := range cs {
		views = append(views, constraintView(c))
	}
	return views
}

func constraintView(c *ir.Constraint) *plugin.ConstraintView {
	v := &plugin.ConstraintView{Source: Source(c)}
	switch c.GetName() {
	case "length":
		if len(c.GetArgs()) != 1 {
			break
		}
		low, high, err := Length(c.GetArgs()[0])
		if err != nil {
			v.Problem = "length " + err.Error()
			break
		}
		v.Length = &ir.Range{Low: low, High: high}
	case "matches":
		if len(c.GetArgs()) != 1 {
			break
		}
		_, err := Pattern(c.GetArgs()[0])
		v.Re2 = err == nil
	}
	return v
}

// Problems reports each constraint in m whose view in a carries a problem,
// as an error at the constraint. A constraint a newtype inherits is
// reported where it is written, once.
func Problems(m *ir.Model, a *plugin.Analysis) []*plugin.Diagnostic {
	var diags []*plugin.Diagnostic
	report := func(cs []*ir.Constraint, views []*plugin.ConstraintView) {
		for i, c := range cs {
			if i >= len(views) || views[i].GetProblem() == "" || c.GetFrom() != nil {
				continue
			}
			diags = append(diags, &plugin.Diagnostic{
				Severity: plugin.Severity_SEVERITY_ERROR,
				Message:  views[i].GetProblem(),
				Position: c.GetPosition(),
			})
		}
	}
	for i, d := range m.GetDecls() {
		v := a.GetDecls()[i]
		report(d.GetNewtype().GetValueConstraints(), v.GetConstraints())
		for j, f := range d.Fields() {
			report(f.GetConstraints(), v.GetFields()[j].GetConstraints())
		}
		for j, variant := range d.GetEnumeration().GetVariants() {
			for k, f := range variant.GetFields() {
				report(f.GetConstraints(), v.GetVariants()[j].GetFields()[k].GetConstraints())
			}
		}
	}
	return diags
}
