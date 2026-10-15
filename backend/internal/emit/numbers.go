package emit

import (
	"errors"

	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
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
type NumberRule = analysis.Rule

// DeclareRule is rule as a backend declares it in its Numbering.
func DeclareRule(rule NumberRule) *plugin.NumberRule {
	r := &plugin.NumberRule{Max: rule.Max}
	for _, rr := range rule.Reserved {
		r.Reserved = append(r.Reserved, &ir.Range{Low: &rr[0], High: &rr[1]})
	}
	return r
}

// Space is one list of members a wire format numbers together: a
// declaration's fields, its variants, or one variant's fields.
type Space struct {
	decl    *ir.Decl
	variant int
}

const (
	ownFields = -1
	variants  = -2
)

// FieldSpace is a structure's fields.
func FieldSpace(d *ir.Decl) Space { return Space{d, ownFields} }

// VariantSpace is an enum's variants.
func VariantSpace(d *ir.Decl) Space { return Space{d, variants} }

// VariantFieldSpace is the fields of an enum's variant i.
func VariantFieldSpace(d *ir.Decl, i int) Space { return Space{d, i} }

// NumbersIn assigns each member of space its wire number: the numbers the
// request's analysis carries, which tdl assigned under the backend's
// Numbering, or else [Session.Numbers]. members lists the space as the
// analysis does, an inlined field contributing its enum's variants.
func (s *Session) NumbersIn(space Space, owner string, members []Member, rule NumberRule) ([]int64, error) {
	if nums := s.numbered(space); len(nums) == len(members) {
		return nums, nil
	}
	return s.Numbers(owner, members, rule)
}

// numbered is the numbers the analysis gives space, or nil when it gives
// any member none.
func (s *Session) numbered(space Space) []int64 {
	v := s.declView(space.decl)
	if v == nil {
		return nil
	}
	views := v.GetFields()
	switch {
	case space.variant == variants:
		views = v.GetVariants()
	case space.variant >= 0:
		if space.variant >= len(v.GetVariants()) {
			return nil
		}
		views = v.GetVariants()[space.variant].GetFields()
	}
	var nums []int64
	for _, m := range views {
		switch {
		case len(m.GetInlined()) > 0:
			nums = append(nums, m.GetInlined()...)
		case m.GetNumber() > 0:
			nums = append(nums, m.GetNumber())
		default:
			return nil
		}
	}
	return nums
}

// Numbers assigns each member its wire number, as tdl does for a request
// it numbers. A `number` directive pins a member's number; each unpinned
// member, in declaration order, takes the lowest number from 1 that no pin
// or earlier member holds and the rule does not reserve or skip. A pin the
// rule refuses, or two pins sharing one, is an [InvalidError]; running out
// of numbers is an [UnsupportedError].
func (s *Session) Numbers(owner string, members []Member, rule NumberRule) ([]int64, error) {
	ms := make([]analysis.Member, len(members))
	for i, m := range members {
		ms[i] = analysis.Member{Name: m.Name, Position: m.Position}
		if d, ok := s.Find(m.Directives, "number"); ok {
			ms[i].Pin = d
		}
	}
	nums, err := analysis.Assign(owner, s.Lang, ms, rule)
	if e, ok := errors.AsType[*analysis.NumberError](err); ok {
		if e.Invalid {
			return nil, Invalid(e.Position, "%s", e.What)
		}
		return nil, Unsupported(e.Position, "%s", e.What)
	}
	return nums, err
}

// Pins is which members need a `number` directive for [Session.Numbers] to
// assign them nums, the numbers a schema already gives them: the fewest,
// found by pinning the first member allocation gets wrong until none is.
// A number the rule refuses is pinned, and [Session.Numbers] refuses it.
func Pins(nums []int64, rule NumberRule) []bool { return analysis.Pins(nums, rule) }
