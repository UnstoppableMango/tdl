package sema

import (
	"sort"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/ir"
)

// Reference is one name, as written, and what it resolved to.
//
// Lowering resolves every name and then keeps only the answer's effect: a
// type reference is interned into the type table, and the table is keyed
// by structure, so the second mention of a name is not a second entry and
// a cursor on it has nothing to find. This records the question instead,
// which is what a "go to definition" is asking.
//
// Target is where to go. It is where the name was bound rather than a
// position derived later, so the ladder that decided what a name means,
// shadowing, the prelude, and `_` imports, is applied once, by the code
// that already applies it.
type Reference struct {
	Pos  ast.Position // where the name was written
	Len  int          // its length in bytes, so a caller can underline it
	Name string       // the name as written

	// Target is where what the name refers to was declared, and is zero
	// when there is nowhere to go: a name that did not resolve has none,
	// and neither does a qualified one, whose declaration is in a
	// dependency this model parsed for its names and not for its
	// positions. A name a `_` import merged in has the position it was
	// declared at, in the dependency that declares it.
	Target ast.Position

	// Decl is the declaration the name refers to, for a caller that wants
	// the model entry rather than a position. It is unset for a type
	// parameter, which declares nothing, and for an unresolved name.
	Decl *ir.ID
}

// References is every name one lowering resolved, in source order.
type References []Reference

// At returns the reference covering a byte offset in a file, and whether
// there is one.
//
// References are sorted, so this is a binary search rather than a walk: an
// editor asks this on every hover, and a model is not small.
func (rs References) At(filename string, offset int) (Reference, bool) {
	i := sort.Search(len(rs), func(i int) bool {
		if rs[i].Pos.Filename != filename {
			return rs[i].Pos.Filename > filename
		}
		return rs[i].Pos.Offset+rs[i].Len > offset
	})
	if i == len(rs) {
		return Reference{}, false
	}

	r := rs[i]
	if r.Pos.Filename != filename || offset < r.Pos.Offset {
		return Reference{}, false
	}
	return r, true
}

// WithReferences records every name lowering resolves into out.
//
// Recording is opt-in because nothing but an editor wants it: `tdl check`
// and `tdl gen` resolve the same names and throw the answers away, and
// they should keep doing that rather than allocate a table nobody reads.
func WithReferences(out *References) Option {
	return func(c *config) { c.refs = out }
}

// record adds a reference, if this lowering is recording them.
//
// It is called where a name is written rather than where one is resolved:
// `[T]` resolves `List`, and a cursor in the brackets should not land on a
// name the author never typed.
func (l *lowerer) record(pos ast.Position, name string, b binding, ok bool) {
	if l.refs == nil {
		return
	}

	ref := Reference{Pos: pos, Len: len(name), Name: name}
	if ok {
		ref.Target = b.pos
		if b.kind == bindDecl {
			ref.Decl = b.id
		}
	}
	*l.refs = append(*l.refs, ref)
}

// recordLookup resolves a name for the side effect of recording it, for a
// caller that resolved it some other way and wants the record to agree.
func (l *lowerer) recordLookup(pos ast.Position, name string) {
	if l.refs == nil {
		return
	}
	b, ok := l.scope.lookup(name)
	l.record(pos, name, b, ok)
}

// sortReferences puts the table in source order, which is what makes At a
// binary search. Lowering visits declarations in order but resolves a
// name when it reaches it, and a constraint or a target block is reached
// in a later pass than the field it is about.
func (rs References) sort() {
	sort.SliceStable(rs, func(i, j int) bool {
		if rs[i].Pos.Filename != rs[j].Pos.Filename {
			return rs[i].Pos.Filename < rs[j].Pos.Filename
		}
		return rs[i].Pos.Offset < rs[j].Pos.Offset
	})
}
