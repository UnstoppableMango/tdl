package emit

import (
	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/ir"
)

// References returns the declarations d names through its fields, its
// variants' fields, a newtype's base, and their type arguments, with
// aliases expanded. A declaration naming itself is included.
func (s *Session) References(d *ir.Decl) []*ir.Decl {
	if v := s.declView(d); v != nil {
		return s.decl(v.GetReferences())
	}
	return s.TypeReferences(analysis.BodyTypes(d)...)
}

// TypeReferences returns the declarations the type references ids name,
// walked as [Session.References] walks a declaration's.
func (s *Session) TypeReferences(ids ...*ir.ID) []*ir.Decl {
	var refs []*ir.ID
	seen := map[int32]bool{}
	for _, id := range ids {
		if s.Model.Type(id) == nil {
			continue
		}
		for _, ref := range s.analysis().GetTypes()[id.GetIndex()].GetReferences() {
			if !seen[ref.GetIndex()] {
				seen[ref.GetIndex()] = true
				refs = append(refs, ref)
			}
		}
	}
	return s.decl(refs)
}

// decl returns the declarations ids name.
func (s *Session) decl(ids []*ir.ID) []*ir.Decl {
	out := make([]*ir.Decl, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.Model.Decl(id))
	}
	return out
}

// Cascade extends skipped, transitively, with every declaration in decls
// that names a skipped one, and warns about each one it adds, so the output
// never refers to something it does not declare.
func (s *Session) Cascade(decls []*ir.Decl, skipped map[*ir.Decl]bool) {
	for changed := true; changed; {
		changed = false
		for _, d := range decls {
			// An alias emits nothing; References already expands it.
			if skipped[d] || d.GetAlias() != nil {
				continue
			}
			for _, ref := range s.References(d) {
				if !skipped[ref] {
					continue
				}
				skipped[d] = true
				changed = true
				s.Warn(Unsupported(d.GetMeta().GetPosition(),
					"%s names %s, which is not generated", d.GetMeta().GetName(), ref.GetMeta().GetName()))
				break
			}
		}
	}
}
