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
	spec      int          // higher wins
	pos       ast.Position // where the entry was written
}

// Specificity, as the spec's ladder: a directive on a field beats one on
// its type, which beats one on a class the type satisfies, and a subclass
// beats a class it requires.
const (
	specClass = 100 // minus the distance from the conforming class
	specDecl  = 1000
	specField = 2000
)

// targetPass is the walk over every target block in a file. Candidates are
// collected before any are applied, because deciding which of two entries
// wins needs both of them.
type targetPass struct {
	*lowerer
	byDecl  map[int32][]candidate
	byField map[fieldKey][]candidate
}

type fieldKey struct {
	decl  int32
	field int
}

// lowerTargets resolves every target block against the model and attaches
// each directive to the node it applies to.
//
// By the time a backend runs, paths are resolved, class paths are expanded
// across everything satisfying them, the ladder has been applied, and
// conflicts have been reported. A backend reads one field on the node in
// front of it.
func (l *lowerer) lowerTargets(file *ast.File) {
	t := &targetPass{
		lowerer: l,
		byDecl:  map[int32][]candidate{},
		byField: map[fieldKey][]candidate{},
	}

	for _, decl := range file.Decls {
		if block, ok := decl.(*ast.TargetDecl); ok {
			t.block(block)
		}
	}

	for idx, cands := range t.byDecl {
		l.model.Decls[idx].Directives = l.resolveConflicts(cands)
	}
	for key, cands := range t.byField {
		l.model.GetDecls()[key.decl].Fields()[key.field].Directives = l.resolveConflicts(cands)
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
	// Paths are at most two deep: a declaration and one of its fields.
	head, member, _ := strings.Cut(path, ".")

	b, ok := t.scope.lookup(head)
	if !ok || b.kind != bindDecl {
		t.diags.add(pos, "target path %s names nothing", path)
		return
	}
	idx := b.id.GetIndex()
	decl := t.model.GetDecls()[idx]

	// A path naming a class applies to everything satisfying it, which is
	// what lets a rule be written once rather than repeated per type.
	if decl.GetClass() != nil {
		t.expandClass(b.id, member, pos, d)
		return
	}

	if member == "" {
		t.byDecl[idx] = append(t.byDecl[idx], candidate{directive: d, spec: specDecl, pos: pos})
		return
	}

	field := fieldIndex(decl, member)
	if field < 0 {
		t.diags.add(pos, "target path %s names nothing: %s has no field %s", path, head, member)
		return
	}
	key := fieldKey{idx, field}
	t.byField[key] = append(t.byField[key], candidate{directive: d, spec: specField, pos: pos})
}

// expandClass applies a directive to every declaration satisfying a class.
//
// A closer class wins: a directive on Auditable beats one on the
// Timestamped it requires, because a type conforming to Auditable is more
// specifically that than it is timestamped.
func (t *targetPass) expandClass(class *ir.ID, member string, pos ast.Position, d *ir.Directive) {
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
		c := candidate{directive: expanded, spec: specClass - t.classDistance(decl, class), pos: pos}

		if member == "" {
			t.byDecl[idx] = append(t.byDecl[idx], c)
			continue
		}
		if field := fieldIndex(decl, member); field >= 0 {
			key := fieldKey{idx, field}
			t.byField[key] = append(t.byField[key], c)
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
// specific candidate wins, and two at the same specificity are an error
// rather than a silent choice.
func (l *lowerer) resolveConflicts(cands []candidate) []*ir.Directive {
	key := func(c candidate) string { return c.directive.GetTarget() + "\x00" + c.directive.GetName() }
	best := map[string]candidate{}
	tied := map[string]bool{}

	for _, c := range cands {
		k := key(c)
		prev, seen := best[k]
		switch {
		case !seen || c.spec > prev.spec:
			best[k] = c
			tied[k] = false
		case c.spec == prev.spec:
			tied[k] = true
		}
	}

	var out []*ir.Directive
	for _, c := range cands {
		k := key(c)
		if best[k].directive != c.directive {
			continue
		}
		if tied[k] {
			l.diags.add(c.pos, "two entries at the same specificity set %s; one of them has to go",
				c.directive.GetName())
		}
		out = append(out, c.directive)
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
