package analysis

import (
	"fmt"
	"strconv"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// numberDirective pins a member's number.
const numberDirective = "number"

// Member is something a wire format numbers: a field or a variant.
type Member struct {
	Name string
	// Pin is the member's `number` directive, or nil.
	Pin      *ir.Directive
	Position *ir.Position
}

// Rule is what one number space allows.
type Rule struct {
	Max int64
	// Reserved is inclusive ranges that are refused as pins and skipped
	// when allocating.
	Reserved [][2]int64
	// Skip is numbers the declaration reserves: allocation passes over
	// them, and a pin on one is refused.
	Skip map[int64]bool
}

// NumberError is a number a rule refuses. Invalid is set for a pin, which
// is a mistake in the model; otherwise allocation ran out of numbers.
type NumberError struct {
	Position *ir.Position
	What     string
	Invalid  bool
}

func (e *NumberError) Error() string { return e.What }

// Assign gives each member its number. A pin keeps its number; each
// unpinned member, in order, takes the lowest number from 1 that no pin or
// earlier member holds and the rule allows. owner and lang name the space
// and the target in a message.
func Assign(owner, lang string, members []Member, rule Rule) ([]int64, error) {
	nums := make([]int64, len(members))
	by := map[int64]string{}

	for i, m := range members {
		if m.Pin == nil {
			continue
		}
		pos := m.Pin.GetPosition()
		arg := m.Pin.GetArgs()[0]
		n, err := strconv.ParseInt(arg.GetText(), 0, 64)
		if arg.GetKind() != ir.LiteralKind_LITERAL_KIND_INT || err != nil {
			return nil, &NumberError{pos, fmt.Sprintf("%s.%s is numbered %q, and a number is an integer", owner, m.Name, arg.GetText()), true}
		}
		if what := refused(owner, m.Name, lang, n, rule); what != "" {
			return nil, &NumberError{pos, what, true}
		}
		if rule.Skip[n] {
			return nil, &NumberError{pos, fmt.Sprintf("%s.%s is numbered %d, which %s reserves", owner, m.Name, n, owner), true}
		}
		if other, ok := by[n]; ok {
			return nil, &NumberError{pos, fmt.Sprintf("%s.%s and %s.%s are both numbered %d", owner, other, owner, m.Name, n), true}
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
		if what := refused(owner, m.Name, lang, next, rule); what != "" {
			return nil, &NumberError{m.Position, what, false}
		}
		by[next] = m.Name
		nums[i] = next
	}
	return nums, nil
}

// refused says why rule refuses n, or nothing when it allows it.
func refused(owner, name, lang string, n int64, rule Rule) string {
	if n < 1 || n > rule.Max {
		return fmt.Sprintf("%s.%s is numbered %d, and %s numbers run from 1 to %d", owner, name, n, lang, rule.Max)
	}
	if r := reserved(n, rule); r != nil {
		return fmt.Sprintf("%s.%s is numbered %d, which %s reserves (%d to %d)", owner, name, n, lang, r[0], r[1])
	}
	return ""
}

// reserved is the range in rule holding n, or nil when none does.
func reserved(n int64, rule Rule) *[2]int64 {
	for i, r := range rule.Reserved {
		if n >= r[0] && n <= r[1] {
			return &rule.Reserved[i]
		}
	}
	return nil
}

// Pins is which members need a pin for [Assign] to give them nums, the
// numbers a schema already gives them: the fewest, found by pinning the
// first member allocation gets wrong until none is. A number the rule
// refuses is pinned, and [Assign] refuses it.
func Pins(nums []int64, rule Rule) []bool {
	pinned := make([]bool, len(nums))
	for {
		by := map[int64]bool{}
		for i, n := range nums {
			if pinned[i] {
				by[n] = true
			}
		}
		next, wrong := int64(1), -1
		for i, n := range nums {
			if pinned[i] {
				continue
			}
			for by[next] || rule.Skip[next] || reserved(next, rule) != nil {
				next++
			}
			if next != n {
				wrong = i
				break
			}
			by[next] = true
		}
		if wrong < 0 {
			return pinned
		}
		pinned[wrong] = true
	}
}

// RuleOf turns a declared rule into a [Rule].
func RuleOf(r *plugin.NumberRule) Rule {
	rule := Rule{Max: r.GetMax()}
	for _, rr := range r.GetReserved() {
		rule.Reserved = append(rule.Reserved, [2]int64{rr.GetLow(), rr.GetHigh()})
	}
	return rule
}

// Reserves is the numbers a declaration's reserve directive names.
func Reserves(target, directive string, d *ir.Decl) map[int64]bool {
	if directive == "" {
		return nil
	}
	nums := map[int64]bool{}
	for _, r := range plugin.Directives(target, d.GetDirectives()) {
		if r.GetName() != directive {
			continue
		}
		for _, a := range r.GetArgs() {
			if a.GetKind() != ir.LiteralKind_LITERAL_KIND_INT {
				continue
			}
			if n, err := strconv.ParseInt(a.GetText(), 0, 64); err == nil {
				nums[n] = true
			}
		}
	}
	return nums
}

// Number gives every field and variant in a its number under n, the rules
// of the backend serving target, which lang names in a message. It
// reports a pin the rules refuse, or a space that runs out of numbers, as
// an error, and leaves that space unnumbered. A nil n numbers nothing.
func Number(m *ir.Model, target, lang string, a *plugin.Analysis, n *plugin.Numbering) []*plugin.Diagnostic {
	if n == nil {
		return nil
	}
	nb := numberer{m: m, target: target, lang: lang, a: a, n: n}
	for i, d := range m.GetDecls() {
		v := a.GetDecls()[i]
		name := d.GetMeta().GetName()
		switch {
		case d.GetStructure() != nil:
			rule := RuleOf(n.GetFields())
			rule.Skip = Reserves(target, n.GetReserve(), d)
			nb.fields(name, d.Fields(), v.GetFields(), rule)
		case d.GetEnumeration() != nil:
			variants := d.GetEnumeration().GetVariants()
			rule := RuleOf(n.GetValues())
			if Fielded(d.GetEnumeration()) {
				rule = RuleOf(n.GetFields())
			}
			nums := nb.assign(name, variantMembers(target, variants), rule)
			for j, num := range nums {
				v.GetVariants()[j].Number = num
			}
			for j, variant := range variants {
				owner := name + "." + variant.GetMeta().GetName()
				nb.fields(owner, variant.GetFields(), v.GetVariants()[j].GetFields(), RuleOf(n.GetFields()))
			}
		}
	}
	return nb.diags
}

type numberer struct {
	m      *ir.Model
	target string
	lang   string
	a      *plugin.Analysis
	n      *plugin.Numbering
	diags  []*plugin.Diagnostic
}

// fields numbers a field list and records the numbers on views, each
// inlined field contributing its enum's variants in its place.
func (nb *numberer) fields(owner string, fields []*ir.Field, views []*plugin.MemberView, rule Rule) {
	var members []Member
	inlined := make([][]*ir.Variant, len(fields))
	for i, f := range fields {
		if variants := nb.inlined(f); variants != nil {
			inlined[i] = variants
			members = append(members, variantMembers(nb.target, variants)...)
			continue
		}
		members = append(members, Member{f.GetMeta().GetName(), nb.pin(f.GetDirectives()), f.GetMeta().GetPosition()})
	}
	nums := nb.assign(owner, members, rule)
	if nums == nil {
		return
	}
	at := 0
	for i := range fields {
		if inlined[i] != nil {
			views[i].Inlined = nums[at : at+len(inlined[i])]
			at += len(inlined[i])
			continue
		}
		views[i].Number = nums[at]
		at++
	}
}

// inlined is the variants a field carrying the inline directive numbers
// in its place, or nil for a field that does not carry it or whose type is
// not an enum.
func (nb *numberer) inlined(f *ir.Field) []*ir.Variant {
	if nb.n.GetInline() == "" {
		return nil
	}
	if !hasDirective(nb.target, nb.n.GetInline(), f.GetDirectives()) {
		return nil
	}
	t := f.GetType()
	if t == nil || int(t.GetIndex()) >= len(nb.a.GetTypes()) {
		return nil
	}
	v := nb.a.GetTypes()[t.GetIndex()]
	if v.GetForm() != plugin.Form_FORM_NAMED {
		return nil
	}
	e := nb.m.Decl(v.GetDecl()).GetEnumeration()
	if e == nil {
		return nil
	}
	return e.GetVariants()
}

// assign numbers one space, reporting a refusal as an error and returning
// nil.
func (nb *numberer) assign(owner string, members []Member, rule Rule) []int64 {
	nums, err := Assign(owner, nb.lang, members, rule)
	if err != nil {
		e := err.(*NumberError)
		nb.diags = append(nb.diags, &plugin.Diagnostic{
			Severity: plugin.Severity_SEVERITY_ERROR,
			Message:  e.What,
			Position: e.Position,
		})
		return nil
	}
	return nums
}

func (nb *numberer) pin(ds []*ir.Directive) *ir.Directive {
	return pinOf(nb.target, ds)
}

func variantMembers(target string, variants []*ir.Variant) []Member {
	out := make([]Member, len(variants))
	for i, v := range variants {
		out[i] = Member{v.GetMeta().GetName(), pinOf(target, v.GetDirectives()), v.GetMeta().GetPosition()}
	}
	return out
}

// pinOf is the first `number` directive for target carrying an argument,
// as a backend reads one.
func pinOf(target string, ds []*ir.Directive) *ir.Directive {
	for _, d := range plugin.Directives(target, ds) {
		if d.GetName() == numberDirective && len(d.GetArgs()) > 0 {
			return d
		}
	}
	return nil
}

func hasDirective(target, name string, ds []*ir.Directive) bool {
	for _, d := range plugin.Directives(target, ds) {
		if d.GetName() == name {
			return true
		}
	}
	return false
}
