package emit

import (
	"regexp/syntax"

	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/ir"
)

// Length reads a `length` constraint's argument as inclusive bounds: an
// integer is both, and a range leaves an open end nil. A length counts a
// string's code points, not its bytes or UTF-16 units, in every target.
// It reads as tdl does for [plugin.ConstraintView], where a length that
// cannot hold is an error reported before any backend runs.
func Length(l *ir.Literal) (low, high *int64, err error) { return analysis.Length(l) }

// Pattern checks a `matches` pattern against RE2's syntax, which a target
// whose engine accepts every RE2 pattern with the same meaning reads too.
// A pattern RE2 refuses is reported at generation time rather than failing
// when the generated code loads.
func Pattern(l *ir.Literal) (*syntax.Regexp, error) { return analysis.Pattern(l) }

// ConstraintText is a constraint as TDL source writes it, as
// [plugin.ConstraintView] carries it.
func ConstraintText(c *ir.Constraint) string { return analysis.Source(c) }

// LiteralText is a literal as TDL source writes it.
func LiteralText(l *ir.Literal) string { return analysis.LiteralSource(l) }
