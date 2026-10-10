package ast

// Node is any node of the tree that records where it was written.
//
// Pos is where its first token starts and End is just past its last, so
// the two bound its source text. A node built rather than parsed, as
// internal/unlower builds them, has zero positions.
type Node interface {
	Pos() Position
	End() Position
}

// Pos is the start of the file.
func (f *File) Pos() Position {
	return Position{Filename: f.Filename, Line: 1, Col: 1}
}

// End is where the input ended.
func (f *File) End() Position { return f.EOF }

func (d *PackageDecl) Pos() Position   { return d.P }
func (d *PackageDecl) End() Position   { return d.E }
func (d *ImportDecl) Pos() Position    { return d.P }
func (d *ImportDecl) End() Position    { return d.E }
func (h *DeclHead) Pos() Position      { return h.P }
func (h *DeclHead) End() Position      { return h.E }
func (d *Deprecation) Pos() Position   { return d.P }
func (d *Deprecation) End() Position   { return d.E }
func (p *TypeParam) Pos() Position     { return p.P }
func (p *TypeParam) End() Position     { return p.E }
func (k *Kind) Pos() Position          { return k.P }
func (k *Kind) End() Position          { return k.E }
func (t *TypeRef) Pos() Position       { return t.P }
func (t *TypeRef) End() Position       { return t.E }
func (a *TypeArg) Pos() Position       { return a.P }
func (a *TypeArg) End() Position       { return a.E }
func (r *ClassRef) Pos() Position      { return r.P }
func (r *ClassRef) End() Position      { return r.E }
func (i *Include) Pos() Position       { return i.P }
func (i *Include) End() Position       { return i.E }
func (e *TargetEntry) Pos() Position   { return e.P }
func (e *TargetEntry) End() Position   { return e.E }
func (d *Directive) Pos() Position     { return d.P }
func (d *Directive) End() Position     { return d.E }
func (l *Literal) Pos() Position       { return l.P }
func (l *Literal) End() Position       { return l.E }
func (c *Constraint) Pos() Position    { return c.P }
func (c *Constraint) End() Position    { return c.E }
func (d *FunDep) Pos() Position        { return d.P }
func (d *FunDep) End() Position        { return d.E }
func (b *AssocTypeBind) Pos() Position { return b.P }
func (b *AssocTypeBind) End() Position { return b.E }
func (e *UnitExpr) Pos() Position      { return e.P }
func (e *UnitExpr) End() Position      { return e.E }
func (t *UnitTerm) Pos() Position      { return t.P }
func (t *UnitTerm) End() Position      { return t.E }
