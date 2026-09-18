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
	Max      int64
	Reserved [][2]int64 // inclusive ranges the target keeps for itself
}

// Numbers assigns each member its wire number: its position counted from
// one, unless a `number` directive pins it.
//
// The IR carries no numbers, so position is the only default there is, and
// it is fragile: inserting a member anywhere but the end renumbers every
// member after it. A pinned number is how a model keeps its wire format
// while its source moves. A number outside the rule, or two members sharing
// one, is an [UnsupportedError] naming both.
func (s *Session) Numbers(owner string, members []Member, rule NumberRule) ([]int64, error) {
	nums := make([]int64, len(members))
	by := map[int64]string{}

	for i, m := range members {
		n, pos := int64(i+1), m.Position
		if d, ok := s.Find(m.Directives, "number"); ok {
			arg := d.GetArgs()[0]
			pos = d.GetPosition()
			v, err := strconv.ParseInt(arg.GetText(), 0, 64)
			if arg.GetKind() != ir.LiteralKind_LITERAL_KIND_INT || err != nil {
				return nil, Unsupported(pos, "%s.%s is numbered %q, and a number is an integer", owner, m.Name, arg.GetText())
			}
			n = v
		}

		if n < 1 || n > rule.Max {
			return nil, Unsupported(pos, "%s.%s is numbered %d, and %s numbers run from 1 to %d", owner, m.Name, n, s.Lang, rule.Max)
		}
		for _, r := range rule.Reserved {
			if n >= r[0] && n <= r[1] {
				return nil, Unsupported(pos, "%s.%s is numbered %d, which %s reserves (%d to %d)", owner, m.Name, n, s.Lang, r[0], r[1])
			}
		}
		if other, ok := by[n]; ok {
			return nil, Unsupported(pos, "%s.%s and %s.%s are both numbered %d", owner, other, owner, m.Name, n)
		}
		by[n] = m.Name
		nums[i] = n
	}
	return nums, nil
}
