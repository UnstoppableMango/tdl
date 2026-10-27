package emit

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Tagged reports whether a node carries an argument-less directive for
// this target.
func (s *Session) Tagged(all []*ir.Directive, name string) bool {
	for _, d := range plugin.Directives(s.Target, all) {
		if d.GetName() == name {
			return true
		}
	}
	return false
}

// Bare reports whether this target's block carries an argument-less
// directive.
func (s *Session) Bare(name string) bool {
	for _, block := range s.Model.GetTargets() {
		if block.GetMeta().GetName() == s.Target && s.Tagged(block.GetDirectives(), name) {
			return true
		}
	}
	return false
}

// Lose warns that a fact is lost, unless [Session.Roundtrip] is set and an
// annotation carries it.
func (s *Session) Lose(code string, pos *ir.Position, format string, args ...any) {
	if !s.Roundtrip {
		s.Lossy(code, pos, format, args...)
	}
}

// DeclaresLossy is [Declares] for a target that imports back: what it cannot
// generate is a loss, and an alias, expanded where it is used, is a loss
// too, with aliased called on its target so the backend can warn about
// what the expansion loses.
func (s *Session) DeclaresLossy(d *ir.Decl, aliased func(target *ir.ID)) (bool, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()

	switch {
	case d.GetClass() != nil:
		return false, Lost(LossClass, pos, "%s is a class, and classes are not generated yet", name)
	case d.GetUnit() != nil:
		return false, Lost(LossUnit, pos, "%s is a unit, and units are not generated yet", name)
	case d.GetAlias() != nil:
		if len(d.Params()) == 0 {
			s.Lose(LossAlias, pos, "alias %s is expanded where used", name)
			aliased(d.GetAlias().GetTarget())
		}
		return false, nil
	case d.GetStructure() == nil && d.GetEnumeration() == nil && d.GetNewtype() == nil:
		// A primitive declares nothing.
		return false, nil
	}
	if len(d.Params()) > 0 {
		return false, Lost(LossGeneric, pos, "%s is parameterized, and generics are not generated yet", name)
	}
	return true, nil
}

// LoseKind warns that an entity or a mixin reads back as a value, and
// returns its kind: "entity", "mixin", or "" for anything else.
func (s *Session) LoseKind(d *ir.Decl) string {
	name := d.GetMeta().GetName()
	switch d.GetStructure().GetKind() {
	case ir.StructKind_STRUCT_KIND_ENTITY:
		s.Lose(LossStructKind, d.GetMeta().GetPosition(), "%s is an entity, and reads back as a value", name)
		return "entity"
	case ir.StructKind_STRUCT_KIND_MIXIN:
		s.Lose(LossStructKind, d.GetMeta().GetPosition(), "%s is a mixin, and reads back as a value", name)
		return "mixin"
	}
	return ""
}

// LoseConforms warns about each conformance of a struct or an enum that a
// target does not carry, which is all of them but a struct's plain
// conformance to Entity, since its kind says that. It reports whether any
// was lost.
func (s *Session) LoseConforms(d *ir.Decl, target string) bool {
	conforms := d.GetStructure().GetConforms()
	if e := d.GetEnumeration(); e != nil {
		conforms = e.GetConforms()
	}
	name := d.GetMeta().GetName()
	lost := false
	for _, c := range conforms {
		if n := c.GetClass().GetName(); d.GetStructure() == nil || n != "Entity" || c.GetExtern() != nil || len(c.GetArgs()) > 0 {
			s.Lose(LossClass, c.GetPosition(), "%s conforms to %s, which %s does not carry", name, n, target)
			lost = true
		}
	}
	return lost
}

// Skip reports a declaration that is not generated. Under roundtrip it is
// carried as TDL instead, so nothing is reported, unless the model is
// invalid.
func (s *Session) Skip(err error) {
	if _, invalid := errors.AsType[*InvalidError](err); !s.Roundtrip || invalid {
		s.Warn(err)
	}
}

// Unlowered is the model as TDL, built once, for a Go backend comparing
// what it reads back with what the model says. A plugin in another
// language reads [Session.Items] instead.
func (s *Session) Unlowered() *ast.File {
	if s.unlowered == nil {
		s.unlowered = unlower.File(s.Model)
	}
	return s.unlowered
}

// source is the model as TDL: the request's, which the host sends under
// the roundtrip directive, or derived when the request carries none.
func (s *Session) source() *plugin.Source {
	if src := s.analysis().GetSource(); src != nil {
		return src
	}
	if s.derived == nil {
		s.derived = analysis.ModelSource(s.Model, s.Target)
	}
	return s.derived
}

// Items is the model's top-level items as TDL, in order.
func (s *Session) Items() []*plugin.Item {
	return s.source().GetItems()
}

// Index is a declaration's index among the model's top-level items.
func (s *Session) Index(name string) int {
	for i, it := range s.Items() {
		if it.GetName() == name {
			return i
		}
	}
	return -1
}

// DeclSource is a declaration as TDL, doc comment included, or "" when the
// model declares nothing by that name.
func (s *Session) DeclSource(name string) string {
	if i := s.Index(name); i >= 0 {
		return s.Items()[i].GetSource()
	}
	return ""
}

// IsDecl reports whether a top-level item is a declaration, which has a
// name of its own, rather than an instance or a target block.
func IsDecl(d ast.Decl) bool {
	return analysis.IsDecl(d)
}

// Conforms is a struct's or an enum's conformance list as TDL:
// `Entity, Auditable`.
func (s *Session) Conforms(name string) string {
	if i := s.Index(name); i >= 0 {
		return s.Items()[i].GetConforms()
	}
	return ""
}

// FieldSource is a field as TDL, without its doc comment, and without its
// deprecation unless deprecated is set.
func (s *Session) FieldSource(f *ir.Field, deprecated bool) string {
	var src, dep string
	if v := s.fieldView(f); s.analysis().GetSource() != nil && v != nil {
		src, dep = v.GetSource(), v.GetDeprecation()
	} else {
		src, dep = analysis.FieldSource(s.Model, f)
	}
	if deprecated && dep != "" {
		return dep + " " + src
	}
	return src
}

// fieldView is the analysis of a field of a declaration in Model, or nil.
func (s *Session) fieldView(f *ir.Field) *plugin.MemberView {
	views := s.analysis().GetDecls()
	if s.fields == nil {
		s.fields = map[*ir.Field]*plugin.MemberView{}
		for i, d := range s.Model.GetDecls() {
			for j, f := range d.Fields() {
				s.fields[f] = views[i].GetFields()[j]
			}
			for j, v := range d.GetEnumeration().GetVariants() {
				for k, f := range v.GetFields() {
					s.fields[f] = views[i].GetVariants()[j].GetFields()[k]
				}
			}
		}
	}
	return s.fields[f]
}

// Carried is a top-level item an annotation carries as TDL, with its index
// among the model's items.
type Carried struct {
	At     int
	Source string
}

// Carried is every top-level item that is not an emitted declaration,
// this target's block without its roundtrip directive included.
func (s *Session) Carried(emitted map[string]bool) []Carried {
	var out []Carried
	for i, it := range s.Items() {
		if it.GetName() != "" && emitted[it.GetName()] || it.GetSource() == "" {
			continue
		}
		out = append(out, Carried{i, it.GetSource()})
	}
	return out
}

// Imports is each import of the model as TDL.
func (s *Session) Imports() []string {
	return s.source().GetImports()
}

// PrintItem is a top-level item as TDL, doc comment included.
func PrintItem(d ast.Decl) string {
	return analysis.PrintItem(d)
}

// CheckRenamed warns when import would write a `name` directive and the
// model has none, or the reverse: readerWrites is whether the name the
// node is written with differs from the convention import reads. It
// reports whether the name directive is lost.
func (s *Session) CheckRenamed(dirs []*ir.Directive, readerWrites bool, format string, args ...any) bool {
	d, has := s.Find(dirs, "name")
	if has == readerWrites {
		return false
	}
	what := fmt.Sprintf(format, args...)
	if has {
		s.Lose(LossName, d.GetPosition(), "%s's name directive restates the convention, and does not read back", what)
		return true
	}
	s.Lose(LossName, nil, "%s reads back with a name directive", what)
	return true
}

// LoseDeclName warns when a declaration is written as a name that reads
// back as another, and otherwise checks its name directive against the
// PascalCase convention import reads. It reports whether anything is lost.
func (s *Session) LoseDeclName(d *ir.Decl, written string) bool {
	name := d.GetMeta().GetName()
	if written != name {
		s.Lose(LossName, d.GetMeta().GetPosition(), "%s is written %s, which reads back as its name", name, written)
		return true
	}
	return s.CheckRenamed(d.GetDirectives(), Pascal(written) != written, "%s", name)
}

// LoseFieldName is [Session.LoseDeclName] for a field of owner, for a
// target that reads a field's name back as written.
func (s *Session) LoseFieldName(owner string, f *ir.Field, written string) bool {
	name := f.GetMeta().GetName()
	if written != name {
		s.Lose(LossName, f.GetMeta().GetPosition(), "%s.%s is written %s, which reads back as its name", owner, name, written)
		return true
	}
	return s.CheckRenamed(f.GetDirectives(), false, "%s.%s", owner, name)
}

// LoseIncludes warns at each include among fields, flattened into owner's
// members, which a target names: includes is [unlower.Includes] of fields,
// or empty names for fields that copy none. It reports whether any include
// is lost.
func (s *Session) LoseIncludes(owner string, fields []*ir.Field, includes []string, members string) bool {
	lost := false
	for i, inc := range includes {
		if inc != "" && (i == 0 || includes[i-1] != inc) {
			s.Lose(LossInclude, fields[i].GetMeta().GetPosition(), "%s's include of %s is flattened into its %s", owner, inc, members)
			lost = true
		}
	}
	return lost
}

// LoseOwnedDefault warns when a field of owner is owned, which target does
// not carry, or has a default, which is not written. It reports whether
// either is lost.
func (s *Session) LoseOwnedDefault(owner, target string, f *ir.Field) bool {
	name, pos := f.GetMeta().GetName(), f.GetMeta().GetPosition()
	if f.GetOwned() {
		s.Lose(LossOwned, pos, "%s.%s is owned, which %s does not carry", owner, name, target)
	}
	if f.GetDefaultValue() != nil {
		s.Lose(LossDefault, pos, "%s.%s has a default, which is not written", owner, name)
	}
	return f.GetOwned() || f.GetDefaultValue() != nil
}

// CheckPins warns when import would pin a member's number differently: it
// pins the fewest members that give every member its number.
func (s *Session) CheckPins(owner string, members []Member, nums []int64, rule NumberRule) {
	pins := Pins(nums, rule)
	for i, m := range members {
		d, has := s.Find(m.Directives, "number")
		switch {
		case has && !pins[i]:
			s.Lose(LossNumber, d.GetPosition(), "%s.%s is pinned to %d, which it would be given anyway, and the pin does not read back", owner, m.Name, nums[i])
		case !has && pins[i]:
			s.Lose(LossNumber, m.Position, "%s.%s reads back pinned to %d", owner, m.Name, nums[i])
		}
	}
}

// FromScreaming turns SCREAMING_SNAKE into Pascal case: IN_TRANSIT is
// InTransit.
func FromScreaming(s string) string {
	var b strings.Builder
	for w := range strings.SplitSeq(s, "_") {
		r := []rune(strings.ToLower(w))
		if len(r) == 0 {
			continue
		}
		r[0] = unicode.ToUpper(r[0])
		b.WriteString(string(r))
	}
	return b.String()
}
