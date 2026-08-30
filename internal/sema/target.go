package sema

import (
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

// lowerTargets resolves every target block against the model and attaches
// each directive to the node it applies to.
//
// By the time a backend runs, paths are resolved, class paths are expanded
// across everything satisfying them, the ladder has been applied, and
// conflicts have been reported. A backend reads one field on the node in
// front of it.
func (l *lowerer) lowerTargets(file *ast.File) {
	// Candidates are collected before any are applied, because deciding
	// which of two entries wins needs both of them.
	byDecl := map[int32][]candidate{}
	byField := map[fieldKey][]candidate{}

	for _, decl := range file.Decls {
		block, ok := decl.(*ast.TargetDecl)
		if !ok {
			continue
		}
		l.lowerTargetBlock(block, byDecl, byField)
	}

	for idx, cands := range byDecl {
		l.model.Decls[idx].Directives = l.resolveConflicts(cands)
	}
	for key, cands := range byField {
		l.fieldAt(key).Directives = l.resolveConflicts(cands)
	}
}

type fieldKey struct {
	decl  int32
	field int
}

func (l *lowerer) fieldAt(key fieldKey) *ir.Field {
	return l.model.GetDecls()[key.decl].Fields()[key.field]
}

func (l *lowerer) lowerTargetBlock(block *ast.TargetDecl, byDecl map[int32][]candidate, byField map[fieldKey][]candidate) {
	out := &ir.TargetBlock{
		Meta:       metaOf(block.N, block.Doc, block.P, block.Dep, len(l.model.GetTargets())),
		ForPackage: block.For,
	}
	l.model.Targets = append(l.model.Targets, out)

	if block.For != l.model.GetPackage() {
		l.diags.add(block.P, "target %s is for package %s, not %s",
			block.N, block.For, l.model.GetPackage())
		return
	}

	l.walkEntries(block, "", block.Entries, out, byDecl, byField)
}

// walkEntries resolves the entries of a block, with scope naming the path a
// nested block is under.
func (l *lowerer) walkEntries(block *ast.TargetDecl, scope string, entries []*ast.TargetEntry, out *ir.TargetBlock, byDecl map[int32][]candidate, byField map[fieldKey][]candidate) {
	for _, entry := range entries {
		path := entry.Path
		if scope != "" && path != "" {
			path = scope + "." + path
		}

		switch {
		case entry.Entries != nil:
			l.walkEntries(block, path, entry.Entries, out, byDecl, byField)

		case entry.Path == "":
			// A bare directive applies to the enclosing scope: the package at
			// the top level, or the path a nested block is under.
			d := l.directive(block.N, entry.Directive)
			if scope == "" {
				out.Directives = append(out.Directives, d)
				continue
			}
			l.attach(block, scope, entry.P, d, byDecl, byField)

		default:
			l.attach(block, path, entry.P, l.directive(block.N, entry.Directive), byDecl, byField)
		}
	}
}

// attach resolves a path and records the directive as a candidate for every
// node it reaches.
func (l *lowerer) attach(block *ast.TargetDecl, path string, pos ast.Position, d *ir.Directive, byDecl map[int32][]candidate, byField map[fieldKey][]candidate) {
	head, member := split(path)

	b, ok := l.scope.lookup(head)
	if !ok || b.kind != bindDecl {
		l.diags.add(pos, "target path %s names nothing", path)
		return
	}
	idx := b.id.GetIndex()
	decl := l.model.GetDecls()[idx]

	// A path naming a class applies to everything satisfying it, which is
	// what lets a rule be written once rather than repeated per type.
	if decl.GetClass() != nil {
		l.expandClass(b.id, member, pos, d, byDecl, byField)
		return
	}

	if member == "" {
		byDecl[idx] = append(byDecl[idx], candidate{directive: d, spec: specDecl, pos: pos})
		return
	}

	field, found := fieldIndex(decl, member)
	if !found {
		l.diags.add(pos, "target path %s names nothing: %s has no field %s", path, head, member)
		return
	}
	byField[fieldKey{idx, field}] = append(byField[fieldKey{idx, field}],
		candidate{directive: d, spec: specField, pos: pos})
}

// expandClass applies a directive to every declaration satisfying a class.
//
// A closer class wins: a directive on Auditable beats one on the
// Timestamped it requires, because a type conforming to Auditable is more
// specifically that than it is timestamped.
func (l *lowerer) expandClass(class *ir.ID, member string, pos ast.Position, d *ir.Directive, byDecl map[int32][]candidate, byField map[fieldKey][]candidate) {
	for _, id := range l.model.Satisfying(class) {
		idx := id.GetIndex()
		decl := l.model.GetDecls()[idx]

		expanded := &ir.Directive{
			Name:      d.GetName(),
			Args:      d.GetArgs(),
			Position:  d.GetPosition(),
			Target:    d.GetTarget(),
			FromClass: class,
		}
		spec := specClass - l.classDistance(decl, class)

		if member == "" {
			byDecl[idx] = append(byDecl[idx], candidate{directive: expanded, spec: spec, pos: pos})
			continue
		}
		if field, found := fieldIndex(decl, member); found {
			byField[fieldKey{idx, field}] = append(byField[fieldKey{idx, field}],
				candidate{directive: expanded, spec: spec, pos: pos})
		}
	}
}

// classDistance is how many `requires` hops separate a declaration's own
// conformance from the class a directive was written on.
func (l *lowerer) classDistance(decl *ir.Decl, class *ir.ID) int {
	best := -1
	for _, ref := range conformsOf(decl) {
		if d := l.hops(ref.GetClass(), class, 0, map[int32]bool{}); d >= 0 && (best < 0 || d < best) {
			best = d
		}
	}
	if best < 0 {
		return 0 // reached through an instance rather than a conformance
	}
	return best
}

func (l *lowerer) hops(from, to *ir.ID, depth int, seen map[int32]bool) int {
	if !from.Resolved() || seen[from.GetIndex()] {
		return -1
	}
	if from.GetIndex() == to.GetIndex() {
		return depth
	}
	seen[from.GetIndex()] = true

	best := -1
	for _, ref := range l.model.Decl(from).GetClass().GetRequiresClasses() {
		if d := l.hops(ref.GetClass(), to, depth+1, seen); d >= 0 && (best < 0 || d < best) {
			best = d
		}
	}
	return best
}

// resolveConflicts applies the ladder: per directive name, the most
// specific candidate wins, and two at the same specificity are an error
// rather than a silent choice.
func (l *lowerer) resolveConflicts(cands []candidate) []*ir.Directive {
	best := map[string]candidate{}
	tied := map[string]bool{}

	for _, c := range cands {
		key := c.directive.GetTarget() + "\x00" + c.directive.GetName()
		prev, seen := best[key]
		switch {
		case !seen || c.spec > prev.spec:
			best[key] = c
			tied[key] = false
		case c.spec == prev.spec:
			tied[key] = true
		}
	}

	var out []*ir.Directive
	for _, c := range cands {
		key := c.directive.GetTarget() + "\x00" + c.directive.GetName()
		if best[key].directive != c.directive {
			continue
		}
		if tied[key] {
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

// split divides a path into its head and the rest. Paths are at most two
// deep: a declaration and one of its fields.
func split(path string) (head, member string) {
	for i := 0; i < len(path); i++ {
		if path[i] == '.' {
			return path[:i], path[i+1:]
		}
	}
	return path, ""
}

func fieldIndex(decl *ir.Decl, name string) (int, bool) {
	for i, f := range decl.Fields() {
		if f.GetMeta().GetName() == name {
			return i, true
		}
	}
	return 0, false
}
