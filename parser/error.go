package parser

import (
	"fmt"
	"strings"

	"github.com/unstoppablemango/tdl/lex"
)

// Error is a single parse error at a source position.
type Error struct {
	Pos lex.Position
	Msg string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Pos, e.Msg)
}

// ErrorList is a non-empty list of parse errors, returned by [Parse] when
// parsing fails. It reports every syntax error found in one pass rather
// than stopping at the first.
type ErrorList []*Error

func (el ErrorList) Error() string {
	switch len(el) {
	case 0:
		return "no errors"
	case 1:
		return el[0].Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d parse errors:", len(el))
	for _, e := range el {
		b.WriteString("\n\t")
		b.WriteString(e.Error())
	}
	return b.String()
}

func (el *ErrorList) add(pos lex.Position, format string, args ...any) {
	*el = append(*el, &Error{Pos: pos, Msg: fmt.Sprintf(format, args...)})
}
