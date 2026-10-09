package sema

import (
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// candidate is one directive that could apply to a node, with the
// specificity that decides whether it does.
type candidate struct {
	directive *ir.Directive
	spec      int // higher wins
}

// Specificity, as the spec's ladder: field beats type beats class, and a
// subclass beats a class it requires.
const (
	specDependency = -1  // a dependency's entry loses to any of the root's
	specClass      = 100 // minus the distance from the conforming class
	specDecl       = 1000
	specField      = 2000
)

// targetPass is the walk over every target block in a file. Candidates are
// all collected before any are applied.
type targetPass struct {
	*lowerer
	byDecl   map[int32][]candidate
	byMember map[memberKey][]candidate
	byExtern map[int32][]candidate
}

// memberKey names a node beneath a declaration: a struct's field (variant
// -1), an enum's variant (field -1), or a field of a variant.
type memberKey struct {
	decl           int32
	variant, field int
}

// lowerTargets resolves every target block against the model and attaches
// each directive to the node it applies to, with class paths expanded and
// the ladder applied. Ties are kept in source order for
// gen.CheckDirectives.
func (l *lowerer) lowerTargets(file *ast.File) {
	t := &targetPass{
		lowerer:  l,
		byDecl:   map[int32][]candidate{},
		byMember: map[memberKey][]candidate{},
		byExtern: map[int32][]candidate{},
	}

	// Origin outranks specificity: what a dependency's own target blocks
	// say about a declaration reaches its extern beneath the root's entries.
	for idx, ext := range l.model.GetExterns() {
		for _, d := range l.depDecls[ext.GetPackage()+"."+ext.GetName()] {
			t.byExtern[int32(idx)] = append(t.byExtern[int32(idx)], candidate{directive: d, spec: specDependency})
		}
	}

	for _, decl := range file.Decls {
		if block, ok := decl.(*ast.TargetDecl); ok {
			t.block(block)
		}
	}

	for idx, cands := range t.byDecl {
		l.model.Decls[idx].Directives = l.resolveConflicts(cands)
	}
	for idx, cands := range t.byExtern {
		l.model.Externs[idx].Directives = l.resolveConflicts(cands)
	}
	for key, cands := range t.byMember {
		directives := l.resolveConflicts(cands)
		decl := l.model.GetDecls()[key.decl]
		switch {
		case key.variant < 0:
			decl.Fields()[key.field].Directives = directives
		case key.field < 0:
			decl.GetEnumeration().GetVariants()[key.variant].Directives = directives
		default:
			decl.GetEnumeration().GetVariants()[key.variant].GetFields()[key.field].Directives = directives
		}
	}
}

func (t *targetPass) block(block *ast.TargetDecl) {
	out := &ir.TargetBlock{
		Meta:       metaOf(&block.DeclHead, len(t.model.GetTargets())),
		ForPackage: block.For,
	}
	t.model.Targets = append(t.model.Targets, out)

	if block.For != t.model.GetPackage() {
		t.diags.add(block.P, "target %s is for package %s, not %s",
			block.N, block.For, t.model.GetPackage())
		return
	}

	t.walkEntries(block, "", block.Entries, out)
}

// walkEntries resolves the entries of a block, with scope naming the path a
// nested block is under.
func (t *targetPass) walkEntries(block *ast.TargetDecl, scope string, entries []*ast.TargetEntry, out *ir.TargetBlock) {
	for _, entry := range entries {
		path := entry.Path
		if scope != "" && path != "" {
			path = scope + "." + path
		}

		// Only a top-level entry's first segment is recorded: it names a
		// declaration, and later segments have no position of their own.
		if scope == "" && entry.Path != "" {
			head, _, _ := strings.Cut(entry.Path, ".")
			t.recordLookup(entry.P, head)
		}

		switch {
		case entry.Entries != nil:
			t.walkEntries(block, path, entry.Entries, out)

		case entry.Path == "":
			// A bare directive applies to the enclosing scope: the package at
			// the top level, or the path a nested block is under.
			d := t.directive(block.N, entry.Directive)
			if scope == "" {
				out.Directives = append(out.Directives, d)
				continue
			}
			t.attach(scope, entry.P, d)

		default:
			t.attach(path, entry.P, t.directive(block.N, entry.Directive))
		}
	}
}

// attach resolves a path and records the directive as a candidate for every
// node it reaches.
func (t *targetPass) attach(path string, pos ast.Position, d *ir.Directive) {
	// A path is a declaration and one of its fields, or an enum, one of its
	// variants, and one of that variant's fields.
	head, rest, _ := strings.Cut(path, ".")
	member, sub, _ := strings.Cut(rest, ".")

	b, ok := t.scope.lookup(head)

	// A path to an extern reaches no further than the extern.
	if ok && b.kind == bindExtern {
		if member != "" {
			t.diags.add(pos, "target path %s names nothing: %s is imported, and its members are not visible here", path, head)
			return
		}
		idx := b.id.GetIndex()
		t.byExtern[idx] = append(t.byExtern[idx], candidate{directive: d, spec: specDecl})
		return
	}

	if !ok || b.kind != bindDecl {
		t.diags.add(pos, "target path %s names nothing", path)
		return
	}
	idx := b.id.GetIndex()
	decl := t.model.GetDecls()[idx]

	// A path naming a class applies to everything satisfying it.
	if decl.GetClass() != nil {
		if sub != "" {
			t.diags.add(pos, "target path %s names nothing: a class path reaches a field and no further", path)
			return
		}
		t.expandClass(b.id, member, d)
		return
	}

	if member == "" {
		t.byDecl[idx] = append(t.byDecl[idx], candidate{directive: d, spec: specDecl})
		return
	}

	key := memberKey{decl: idx, variant: -1, field: -1}
	if e := decl.GetEnumeration(); e != nil {
		key.variant = slices.IndexFunc(e.GetVariants(), func(v *ir.Variant) bool { return v.GetMeta().GetName() == member })
		if key.variant < 0 {
			t.diags.add(pos, "target path %s names nothing: %s has no variant %s", path, head, member)
			return
		}
		if sub != "" {
			key.field = slices.IndexFunc(e.GetVariants()[key.variant].GetFields(), func(f *ir.Field) bool { return f.GetMeta().GetName() == sub })
			if key.field < 0 {
				t.diags.add(pos, "target path %s names nothing: %s.%s has no field %s", path, head, member, sub)
				return
			}
		}
	} else {
		key.field = fieldIndex(decl, member)
		if key.field < 0 {
			t.diags.add(pos, "target path %s names nothing: %s has no field %s", path, head, member)
			return
		}
		if sub != "" {
			t.diags.add(pos, "target path %s names nothing: %s.%s is a field, and nothing is beneath a field", path, head, member)
			return
		}
	}
	t.byMember[key] = append(t.byMember[key], candidate{directive: d, spec: specField})
}

// expandClass applies a directive to every declaration satisfying a class.
// A closer class wins: Auditable beats the Timestamped it requires.
func (t *targetPass) expandClass(class *ir.ID, member string, d *ir.Directive) {
	for _, id := range t.model.Satisfying(class) {
		idx := id.GetIndex()
		decl := t.model.GetDecls()[idx]

		expanded := &ir.Directive{
			Name:      d.GetName(),
			Args:      d.GetArgs(),
			Position:  d.GetPosition(),
			Target:    d.GetTarget(),
			FromClass: class,
		}
		c := candidate{directive: expanded, spec: specClass - t.classDistance(decl, class)}

		if member == "" {
			t.byDecl[idx] = append(t.byDecl[idx], c)
			continue
		}
		if field := fieldIndex(decl, member); field >= 0 {
			key := memberKey{decl: idx, variant: -1, field: field}
			t.byMember[key] = append(t.byMember[key], c)
		}
	}
}

// classDistance is how many `requires` hops separate a declaration's own
// conformance from the class a directive was written on.
func (l *lowerer) classDistance(decl *ir.Decl, class *ir.ID) int {
	best := -1
	for _, ref := range conformsOf(decl) {
		if d := l.hops(ref.GetClass(), class.GetIndex(), 0, map[int32]bool{}); d >= 0 && (best < 0 || d < best) {
			best = d
		}
	}
	if best < 0 {
		return 0 // reached through an instance rather than a conformance
	}
	return best
}

// resolveConflicts applies the ladder: per directive name, the most
// specific candidates win, all kept in source order. gen.CheckDirectives
// reports ties.
func (l *lowerer) resolveConflicts(cands []candidate) []*ir.Directive {
	key := func(c candidate) string { return c.directive.GetTarget() + "\x00" + c.directive.GetName() }
	best := map[string]int{}

	for _, c := range cands {
		k := key(c)
		if prev, seen := best[k]; !seen || c.spec > prev {
			best[k] = c.spec
		}
	}

	var out []*ir.Directive
	for _, c := range cands {
		if c.spec == best[key(c)] {
			out = append(out, c.directive)
		}
	}
	return out
}

func (l *lowerer) directive(target string, d *ast.Directive) *ir.Directive {
	out := &ir.Directive{
		Name:     d.N,
		Position: position(d.P),
		Target:   target,
	}
	for _, a := range d.Args {
		out.Args = append(out.Args, l.literal(a))
	}
	return out
}

// fieldIndex is the position of a named field in a declaration, or -1.
func fieldIndex(decl *ir.Decl, name string) int {
	return slices.IndexFunc(decl.Fields(), func(f *ir.Field) bool { return f.GetMeta().GetName() == name })
}
