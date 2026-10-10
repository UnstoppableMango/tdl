package sema

import (
	"fmt"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
)

// Diagnostic is one problem found while lowering, about the source text
// from Pos to End.
type Diagnostic struct {
	Pos ast.Position
	End ast.Position // just past the text; zero when only Pos is known
	Msg string
}

func (d *Diagnostic) Error() string {
	return fmt.Sprintf("%s: %s", d.Pos, d.Msg)
}

// Diagnostics is every problem one pass found. A non-empty list means no
// later pass should run.
type Diagnostics []*Diagnostic

func (ds Diagnostics) Error() string {
	switch len(ds) {
	case 0:
		return "no diagnostics"
	case 1:
		return ds[0].Error()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d problems:", len(ds))
	for _, d := range ds {
		b.WriteString("\n\t")
		b.WriteString(d.Error())
	}
	return b.String()
}

func (ds *Diagnostics) add(at ast.Node, format string, args ...any) {
	*ds = append(*ds, &Diagnostic{Pos: at.Pos(), End: at.End(), Msg: fmt.Sprintf(format, args...)})
}

// span is a stretch of source text a diagnostic is about, for a problem
// found somewhere other than the tree.
type span struct{ start, end ast.Position }

func (s span) Pos() ast.Position { return s.start }
func (s span) End() ast.Position { return s.end }

// point is a span known only by where it starts.
func point(pos ast.Position) span { return span{start: pos} }

// word is the span of name, written at pos.
func word(pos ast.Position, name string) span {
	end := pos
	end.Col += len(name)
	end.Offset += len(name)
	return span{start: pos, end: end}
}
