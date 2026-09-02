package treesitter

import "strings"

// wrapWidth is where a rule stops fitting on one line. The grammar's own
// alternations are long enough that a flat file would be unreadable, and a
// generator nobody reads is a generator nobody checks.
const wrapWidth = 96

// An expr is a piece of the emitted grammar.js. It renders flat when it
// fits and broken over lines when it does not, which is the only thing
// about the output that is not a direct transcription.
type expr interface {
	flat() string
	write(b *strings.Builder, indent string, used int)
}

// A leaf is text with no structure: '$.name', a quoted spelling, or a
// regular expression.
type leaf string

func (l leaf) flat() string { return string(l) }

func (l leaf) write(b *strings.Builder, _ string, _ int) { b.WriteString(string(l)) }

// A call is a tree-sitter combinator applied to arguments.
type call struct {
	fn   string
	args []expr
}

func (c call) flat() string {
	parts := make([]string, len(c.args))
	for i, arg := range c.args {
		parts[i] = arg.flat()
	}
	return c.fn + "(" + strings.Join(parts, ", ") + ")"
}

// write breaks only the outermost call that does not fit, and lets each
// argument decide again at its own indent.
func (c call) write(b *strings.Builder, indent string, used int) {
	if flat := c.flat(); used+len(flat) <= wrapWidth {
		b.WriteString(flat)
		return
	}

	inner := indent + "  "
	b.WriteString(c.fn)
	b.WriteString("(\n")
	for _, arg := range c.args {
		b.WriteString(inner)
		arg.write(b, inner, len(inner))
		b.WriteString(",\n")
	}
	b.WriteString(indent)
	b.WriteString(")")
}

// render writes one expression as the body of a rule, where prefix is the
// text already on the line.
func render(e expr, indent, prefix string) string {
	var b strings.Builder
	e.write(&b, indent, len(indent)+len(prefix))
	return b.String()
}

// quote spells a terminal as a JavaScript string. The grammar's terminals
// are punctuation and keywords, but nothing stops one holding a quote.
func quote(text string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range text {
		switch r {
		case '\'', '\\':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('\'')
	return b.String()
}

// regex spells a lex pattern as a JavaScript regular expression literal.
// The patterns are written for Go's regexp, which needs no delimiter, so a
// '/' in one is bare and has to be escaped here.
func regex(pattern string) string {
	var b strings.Builder
	b.WriteByte('/')
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			b.WriteByte('\\')
			if i+1 < len(pattern) {
				i++
				b.WriteByte(pattern[i])
			}
		case '/':
			b.WriteString(`\/`)
		default:
			b.WriteByte(pattern[i])
		}
	}
	b.WriteByte('/')
	return b.String()
}
