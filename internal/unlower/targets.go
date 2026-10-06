package unlower

import (
	"cmp"
	"slices"

	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// placed is a directive read back off the node lowering attached it to,
// with the path that reaches that node again.
type placed struct {
	path      string
	directive *ir.Directive
}

// targets rebuilds the target blocks. Lowering keeps only a block's bare
// top-level directives on the block and attaches every other directive to
// the node its path reaches, so each is written back as one flat
// `path => directive` entry. A directive written on a class was expanded
// onto everything satisfying it, and is written back once, on the class.
//
// Entries keep source order, read from directive positions, so the
// directives on each node come back in the order lowering found them.
// Lowering drops a directive a more specific one overrides, and since the
// overriding one is written back too, it wins again.
func (u *unlowerer) targets() []ast.Decl {
	blocks := make([]*ast.TargetDecl, 0, len(u.model.GetTargets()))
	entries := make([][]placed, 0, len(u.model.GetTargets()))
	for _, tb := range u.model.GetTargets() {
		blocks = append(blocks, &ast.TargetDecl{DeclHead: declHead(tb.GetMeta()), For: tb.GetForPackage()})
		var bare []placed
		for _, d := range tb.GetDirectives() {
			bare = append(bare, placed{directive: d})
		}
		entries = append(entries, bare)
	}

	for _, p := range u.placements() {
		i := u.blockFor(p.directive, blocks)
		if i == len(blocks) {
			// A directive with no block, as a reverse backend may write, gets
			// a block for the model's package.
			blocks = append(blocks, &ast.TargetDecl{DeclHead: ast.DeclHead{N: p.directive.GetTarget()}, For: u.model.GetPackage()})
			entries = append(entries, nil)
		}
		entries[i] = append(entries[i], p)
	}

	out := make([]ast.Decl, len(blocks))
	for i, b := range blocks {
		slices.SortStableFunc(entries[i], func(x, y placed) int {
			px, py := x.directive.GetPosition(), y.directive.GetPosition()
			return cmp.Or(cmp.Compare(px.GetLine(), py.GetLine()), cmp.Compare(px.GetColumn(), py.GetColumn()))
		})
		for _, p := range entries[i] {
			b.Entries = append(b.Entries, &ast.TargetEntry{
				Path:      p.path,
				Directive: &ast.Directive{N: p.directive.GetName(), Args: literals(p.directive.GetArgs())},
			})
		}
		out[i] = b
	}
	return out
}

// placements reads every directive attached to a node, in declaration
// order.
func (u *unlowerer) placements() []placed {
	var out []placed
	add := func(path, member string, d *ir.Directive) {
		if class := d.GetFromClass(); class != nil {
			path = join(class.GetName(), member)
			if slices.ContainsFunc(out, func(p placed) bool { return p.path == path && sameDirective(p.directive, d) }) {
				return
			}
		}
		out = append(out, placed{path: path, directive: d})
	}

	for _, decl := range u.model.GetDecls() {
		name := decl.GetMeta().GetName()
		for _, d := range decl.GetDirectives() {
			add(name, "", d)
		}
		for _, f := range decl.Fields() {
			for _, d := range f.GetDirectives() {
				add(join(name, f.GetMeta().GetName()), f.GetMeta().GetName(), d)
			}
		}
		for _, v := range decl.GetEnumeration().GetVariants() {
			variant := join(name, v.GetMeta().GetName())
			for _, d := range v.GetDirectives() {
				add(variant, "", d)
			}
			for _, f := range v.GetFields() {
				for _, d := range f.GetDirectives() {
					add(join(variant, f.GetMeta().GetName()), "", d)
				}
			}
		}
	}
	for _, ext := range u.model.GetExterns() {
		for _, d := range ext.GetDirectives() {
			add(ext.GetName(), "", d)
		}
	}
	return out
}

// sameDirective reports whether two expansions came from one directive
// written on a class.
func sameDirective(a, b *ir.Directive) bool {
	return a.GetName() == b.GetName() &&
		a.GetTarget() == b.GetTarget() &&
		proto.Equal(a.GetPosition(), b.GetPosition()) &&
		slices.EqualFunc(a.GetArgs(), b.GetArgs(), func(x, y *ir.Literal) bool { return proto.Equal(x, y) })
}

// blockFor is the index of the block a directive was written in: the only
// block for its target, or of several, the last one starting before it.
// It is len(blocks) when no block is for its target.
func (u *unlowerer) blockFor(d *ir.Directive, blocks []*ast.TargetDecl) int {
	found := len(blocks)
	for i, b := range blocks {
		if b.N != d.GetTarget() {
			continue
		}
		if found == len(blocks) || (i < len(u.model.GetTargets()) && before(u.model.GetTargets()[i].GetMeta().GetPosition(), d.GetPosition())) {
			found = i
		}
	}
	return found
}

func before(a, b *ir.Position) bool {
	if a.GetLine() != b.GetLine() {
		return a.GetLine() < b.GetLine()
	}
	return a.GetColumn() < b.GetColumn()
}

func join(path, member string) string {
	if member == "" {
		return path
	}
	return path + "." + member
}
