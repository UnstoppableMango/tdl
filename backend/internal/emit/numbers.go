package emit

import (
	"strconv"

	"github.com/unstoppablemango/tdl/ir"
)

// Member is something a wire format numbers: a field or a variant.
type Member struct {
	Name       string
	Directives []*ir.Directive
	Position   *ir.Position
}

// FieldMembers is the members of a field list.
func FieldMembers(fields []*ir.Field) []Member {
	out := make([]Member, len(fields))
	for i, f := range fields {
		out[i] = Member{f.GetMeta().GetName(), f.GetDirectives(), f.GetMeta().GetPosition()}
	}
	return out
}

// VariantMembers is the members of a variant list.
func VariantMembers(variants []*ir.Variant) []Member {
	out := make([]Member, len(variants))
	for i, v := range variants {
		out[i] = Member{v.GetMeta().GetName(), v.GetDirectives(), v.GetMeta().GetPosition()}
	}
	return out
}

// NumberRule is what a target allows a member's number to be.
type NumberRule struct {
	Max int64
	// Reserved is inclusive ranges that are refused as pins and skipped
	// when allocating.
	Reserved [][2]int64
	// Skip is numbers allocation passes over; the caller refuses pins.
	Skip map[int64]bool
}

// Numbers assigns each member its wire number. A `number` directive pins a
// member's number; each unpinned member, in declaration order, takes the
// lowest number from 1 that no pin or earlier member holds and the rule does
// not reserve or skip. A pin the rule refuses, or two pins sharing one, is
// an [InvalidError]; running out of numbers is an [UnsupportedError].
func (s *Session) Numbers(owner string, members []Member, rule NumberRule) ([]int64, error) {
	nums := make([]int64, len(members))
	by := map[int64]string{}

	for i, m := range members {
		d, ok := s.Find(m.Directives, "number")
		if !ok {
			continue
		}
		arg := d.GetArgs()[0]
		pos := d.GetPosition()
		n, err := strconv.ParseInt(arg.GetText(), 0, 64)
		if arg.GetKind() != ir.LiteralKind_LITERAL_KIND_INT || err != nil {
			return nil, Invalid(pos, "%s.%s is numbered %q, and a number is an integer", owner, m.Name, arg.GetText())
		}
		if err := s.checkNumber(owner, m.Name, n, pos, rule, Invalid); err != nil {
			return nil, err
		}
		if other, ok := by[n]; ok {
			return nil, Invalid(pos, "%s.%s and %s.%s are both numbered %d", owner, other, owner, m.Name, n)
		}
		by[n] = m.Name
		nums[i] = n
	}

	taken := func(n int64) bool {
		return by[n] != "" || rule.Skip[n] || reserved(n, rule) != nil
	}
	next := int64(1)
	for i, m := range members {
		if nums[i] != 0 {
			continue
		}
		for taken(next) {
			next++
		}
		if err := s.checkNumber(owner, m.Name, next, m.Position, rule, Unsupported); err != nil {
			return nil, err
		}
		by[next] = m.Name
		nums[i] = next
	}
	return nums, nil
}

// checkNumber refuses a number the rule does not allow, through report.
func (s *Session) checkNumber(owner, name string, n int64, pos *ir.Position, rule NumberRule, report func(*ir.Position, string, ...any) error) error {
	if n < 1 || n > rule.Max {
		return report(pos, "%s.%s is numbered %d, and %s numbers run from 1 to %d", owner, name, n, s.Lang, rule.Max)
	}
	if r := reserved(n, rule); r != nil {
		return report(pos, "%s.%s is numbered %d, which %s reserves (%d to %d)", owner, name, n, s.Lang, r[0], r[1])
	}
	return nil
}

// reserved is the range in rule holding n, or nil when none does.
func reserved(n int64, rule NumberRule) *[2]int64 {
	for i, r := range rule.Reserved {
		if n >= r[0] && n <= r[1] {
			return &rule.Reserved[i]
		}
	}
	return nil
}
