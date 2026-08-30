package sema

import (
	"fmt"
	"strings"

	"github.com/unstoppablemango/tdl/ast"
)

// Diagnostic is one problem found while lowering, at a source position.
type Diagnostic struct {
	Pos ast.Position
	Msg string
}

func (d *Diagnostic) Error() string {
	return fmt.Sprintf("%s: %s", d.Pos, d.Msg)
}

// Diagnostics is every problem one pass found.
//
// A pass reports all of them rather than stopping at the first, the way the
// parser reports every syntax error. A non-empty list means no later pass
// should run: every diagnostic it produced against a model this one
// rejected would be noise.
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

func (ds *Diagnostics) add(pos ast.Position, format string, args ...any) {
	*ds = append(*ds, &Diagnostic{Pos: pos, Msg: fmt.Sprintf(format, args...)})
}
