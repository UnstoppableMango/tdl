package emit

import "github.com/unstoppablemango/tdl/ir"

// References returns the declarations d names: through its fields, its
// variants' fields, a newtype's base, and the type arguments of each, with
// aliases followed to what they expand to. A declaration naming itself is
// included.
func (s *Session) References(d *ir.Decl) []*ir.Decl {
	var ids []*ir.ID
	switch {
	case d.GetStructure() != nil:
		for _, f := range d.Fields() {
			ids = append(ids, f.GetType())
		}
	case d.GetEnumeration() != nil:
		for _, v := range d.GetEnumeration().GetVariants() {
			for _, f := range v.GetFields() {
				ids = append(ids, f.GetType())
			}
		}
	case d.GetNewtype() != nil:
		ids = append(ids, d.GetNewtype().GetBase())
	case d.GetAlias() != nil:
		ids = append(ids, d.GetAlias().GetTarget())
	}

	var out []*ir.Decl
	seen := map[*ir.Decl]bool{}
	var walk func(id *ir.ID)
	walk = func(id *ir.ID) {
		t := s.Model.Type(id)
		if t == nil {
			return
		}
		for _, arg := range t.GetArgs() {
			walk(arg)
		}
		decl := s.Model.Decl(t.GetCtor())
		if decl == nil || seen[decl] {
			return
		}
		seen[decl] = true
		if a := decl.GetAlias(); a != nil {
			walk(a.GetTarget())
			return
		}
		out = append(out, decl)
	}
	for _, id := range ids {
		walk(id)
	}
	return out
}

// Cascade extends skipped with every declaration in decls that names a
// skipped one, until nothing more is added, and warns about each one it
// adds.
//
// A backend renders what it can and marks what it cannot. Emitting a
// declaration whose field names a skipped one produces output that refers
// to something it does not declare, which no target accepts, so the
// referrer is skipped too and the warning says which declaration caused it.
func (s *Session) Cascade(decls []*ir.Decl, skipped map[*ir.Decl]bool) {
	for changed := true; changed; {
		changed = false
		for _, d := range decls {
			// An alias emits nothing, and [Session.References] already sees
			// through it to what it expands to.
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
