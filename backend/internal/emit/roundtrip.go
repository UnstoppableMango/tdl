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

// Conforms is a struct's conformance list as TDL: `Entity, Auditable`.
func (s *Session) Conforms(name string) string {
	for _, decl := range s.Unlowered().Decls {
		if sd, ok := decl.(*ast.StructDecl); ok && sd.N == name {
			head := ast.PrintDecl(&ast.StructDecl{DeclHead: ast.DeclHead{N: "T"}, Keyword: "type", Conforms: sd.Conforms})
			head = strings.TrimPrefix(head, "type T: ")
			return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(head), "{ }"))
		}
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
