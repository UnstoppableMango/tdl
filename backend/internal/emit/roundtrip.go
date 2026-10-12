package emit

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/unstoppablemango/tdl/ast"
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

// Unlowered is the model as TDL, built once, read for the TDL an
// annotation carries.
func (s *Session) Unlowered() *ast.File {
	if s.unlowered == nil {
		s.unlowered = unlower.File(s.Model)
	}
	return s.unlowered
}

// Index is a declaration's index among the unlowered file's items.
func (s *Session) Index(name string) int {
	for i, d := range s.Unlowered().Decls {
		if IsDecl(d) && d.Name() == name {
			return i
		}
	}
	return -1
}

// IsDecl reports whether a top-level item is a declaration, which has a
// name of its own, rather than an instance or a target block.
func IsDecl(d ast.Decl) bool {
	switch d.(type) {
	case *ast.InstanceDecl, *ast.TargetDecl:
		return false
	}
	return true
}

// Conforms is a struct's or an enum's conformance list as TDL:
// `Entity, Auditable`.
func (s *Session) Conforms(name string) string {
	for _, decl := range s.Unlowered().Decls {
		var conforms []*ast.ClassRef
		switch d := decl.(type) {
		case *ast.StructDecl:
			conforms = d.Conforms
		case *ast.EnumDecl:
			conforms = d.Conforms
		default:
			continue
		}
		if decl.Name() != name {
			continue
		}
		head := ast.PrintDecl(&ast.StructDecl{DeclHead: ast.DeclHead{N: "T"}, Keyword: "type", Conforms: conforms})
		head = strings.TrimPrefix(head, "type T: ")
		return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(head), "{ }"))
	}
	return ""
}

// FieldSource is a field as TDL, without its doc comment, and without its
// deprecation unless deprecated is set.
func (s *Session) FieldSource(f *ir.Field, deprecated bool) string {
	af := unlower.Field(s.Model, f)
	if !deprecated {
		af.Dep = nil
	}
	return ast.PrintField(af)
}

// Carried is a top-level item an annotation carries as TDL, with its index
// in the unlowered file.
type Carried struct {
	At     int
	Source string
}

// Carried is every top-level item of the unlowered file that is not an
// emitted declaration, this target's block without its roundtrip
// directive included.
func (s *Session) Carried(emitted map[string]bool) []Carried {
	var out []Carried
	for i, d := range s.Unlowered().Decls {
		if IsDecl(d) && emitted[d.Name()] {
			continue
		}
		if t, ok := d.(*ast.TargetDecl); ok && t.N == s.Target {
			if d = WithoutRoundtrip(t); d == nil {
				continue
			}
		}
		out = append(out, Carried{i, PrintItem(d)})
	}
	return out
}

// Imports is each import of the unlowered file as TDL.
func (s *Session) Imports() []string {
	var out []string
	for _, imp := range s.Unlowered().Imports {
		out = append(out, strings.TrimSuffix(ast.Fprint(&ast.File{Imports: []*ast.ImportDecl{imp}}), "\n"))
	}
	return out
}

// WithoutRoundtrip is a target block without the roundtrip directive, or
// nil when nothing else is in it.
func WithoutRoundtrip(t *ast.TargetDecl) ast.Decl {
	out := *t
	out.Entries = slices.DeleteFunc(slices.Clone(t.Entries), func(e *ast.TargetEntry) bool {
		return e.Path == "" && e.Directive != nil && e.Directive.N == "roundtrip" && len(e.Directive.Args) == 0
	})
	if len(out.Entries) == 0 && len(t.Entries) > 0 {
		return nil
	}
	return &out
}

// PrintItem is a top-level item as TDL, doc comment included.
func PrintItem(d ast.Decl) string {
	return strings.TrimSuffix(ast.Fprint(&ast.File{Decls: []ast.Decl{d}}), "\n")
}

// CheckRenamed warns when import would write a `name` directive and the
// model has none, or the reverse: readerWrites is whether the name the
// node is written with differs from the convention import reads.
func (s *Session) CheckRenamed(dirs []*ir.Directive, readerWrites bool, format string, args ...any) {
	d, has := s.Find(dirs, "name")
	if has == readerWrites {
		return
	}
	what := fmt.Sprintf(format, args...)
	if has {
		s.Lose(LossName, d.GetPosition(), "%s's name directive restates the convention, and does not read back", what)
		return
	}
	s.Lose(LossName, nil, "%s reads back with a name directive", what)
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
