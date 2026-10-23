package protobuf

import (
	"regexp"
	"strings"
)

// The layout below is buf format's, so that formatting the generated files
// changes nothing and a consumer can hold them to `buf format` and
// `tdl gen --verify` at once.

// brackets is a field's or an enum value's options as a bracket list, the
// statement indented by indent: one option stays on the line, and several
// take a line each.
func brackets(indent string, opts []string) string {
	switch len(opts) {
	case 0:
		return ""
	case 1:
		return " [" + reflow(opts[0], indent) + "]"
	}
	inner := indent + "  "
	lines := make([]string, len(opts))
	for i, o := range opts {
		lines[i] = inner + reflow(o, inner)
	}
	return " [\n" + strings.Join(lines, ",\n") + "\n" + indent + "]"
}

// optionStmt is an option statement indented by indent.
func optionStmt(indent, opt string) string {
	return indent + "option " + reflow(opt, indent) + ";\n"
}

// reflow lays out an option, `name = value`, whose line starts at indent.
// A message-literal value is written one field per line; any other value,
// or a literal reflow cannot read, is written as given.
func reflow(opt, indent string) string {
	name, value, ok := strings.Cut(opt, " = ")
	if !ok || !strings.HasPrefix(strings.TrimSpace(value), "{") {
		return opt
	}
	lit, ok := parseLiteral(value)
	if !ok {
		return opt
	}
	return name + " = " + lit.layout(indent)
}

// emptyBody matches a block holding nothing, which buf format closes on
// its opening line.
var emptyBody = regexp.MustCompile(`(?m)^( *)((?:message|enum|service|oneof) \w+) \{\n *\}$`)

// collapseEmpty writes each empty block as `name {}`.
func collapseEmpty(src string) string {
	return emptyBody.ReplaceAllString(src, "$1$2 {}")
}

// tvalue is a text-format value: a scalar's source text, a message, or a
// list.
type tvalue struct {
	scalar string
	msg    *tmessage
	list   []tvalue
	isList bool
}

type tfield struct {
	name  string
	value tvalue
}

type tmessage struct {
	fields []tfield
}

func (v tvalue) layout(indent string) string {
	switch {
	case v.msg != nil:
		return v.msg.layout(indent)
	case v.isList:
		return layoutList(v.list, indent)
	}
	return v.scalar
}

func (v tvalue) isScalar() bool { return v.msg == nil && !v.isList }

// layout is a message literal: `{}` when empty, `{name: value}` for one
// scalar field, and a field per line otherwise.
func (m *tmessage) layout(indent string) string {
	switch {
	case len(m.fields) == 0:
		return "{}"
	case len(m.fields) == 1 && m.fields[0].value.isScalar():
		return "{" + m.fields[0].name + ": " + m.fields[0].value.scalar + "}"
	}
	inner := indent + "  "
	var b strings.Builder
	b.WriteString("{\n")
	for _, f := range m.fields {
		b.WriteString(inner + f.name + ": " + f.value.layout(inner) + "\n")
	}
	b.WriteString(indent + "}")
	return b.String()
}

// layoutList is a list literal: `[]` when empty, `[value]` for one scalar,
// and an element per line otherwise. buf format keeps the closing bracket
// of a list holding one message that fits a line on that line.
func layoutList(elems []tvalue, indent string) string {
	inner := indent + "  "
	switch {
	case len(elems) == 0:
		return "[]"
	case len(elems) == 1 && elems[0].isScalar():
		return "[" + elems[0].scalar + "]"
	case len(elems) == 1 && !strings.Contains(elems[0].layout(inner), "\n"):
		return "[\n" + inner + elems[0].layout(inner) + "]"
	}
	lines := make([]string, len(elems))
	for i, e := range elems {
		lines[i] = inner + e.layout(inner)
	}
	return "[\n" + strings.Join(lines, ",\n") + "\n" + indent + "]"
}

// parseLiteral reads a text-format message literal. It accepts the subset
// option values use and refuses the rest, angle brackets, comments, and
// adjacent strings among it, so a value it cannot lay out is kept as given.
func parseLiteral(src string) (*tmessage, bool) {
	toks, ok := tokenize(src)
	if !ok {
		return nil, false
	}
	p := &tparser{toks: toks}
	m, ok := p.message()
	if !ok || p.pos != len(p.toks) {
		return nil, false
	}
	return m, true
}

type tparser struct {
	toks []string
	pos  int
}

func (p *tparser) peek() string {
	if p.pos < len(p.toks) {
		return p.toks[p.pos]
	}
	return ""
}

func (p *tparser) next() string {
	t := p.peek()
	p.pos++
	return t
}

func (p *tparser) message() (*tmessage, bool) {
	if p.next() != "{" {
		return nil, false
	}
	m := &tmessage{}
	for p.peek() != "}" {
		if p.peek() == "" {
			return nil, false
		}
		name := p.next()
		if !isName(name) {
			return nil, false
		}
		colon := p.peek() == ":"
		if colon {
			p.pos++
		}
		v, ok := p.value()
		if !ok || (!colon && v.isScalar()) {
			return nil, false
		}
		m.fields = append(m.fields, tfield{name, v})
		if t := p.peek(); t == "," || t == ";" {
			p.pos++
		}
	}
	p.pos++
	return m, true
}

func (p *tparser) value() (tvalue, bool) {
	switch t := p.peek(); {
	case t == "{":
		m, ok := p.message()
		return tvalue{msg: m}, ok
	case t == "[":
		p.pos++
		v := tvalue{isList: true}
		for p.peek() != "]" {
			e, ok := p.value()
			if !ok || e.isList {
				return tvalue{}, false
			}
			v.list = append(v.list, e)
			switch p.peek() {
			case ",":
				p.pos++
				if p.peek() == "]" {
					return tvalue{}, false
				}
			case "]":
			default:
				return tvalue{}, false
			}
		}
		p.pos++
		return v, true
	case isScalar(t):
		p.pos++
		if isScalar(p.peek()) && strings.ContainsAny(p.peek()[:1], `"'`) {
			// Adjacent strings concatenate, which buf format lays out
			// differently.
			return tvalue{}, false
		}
		return tvalue{scalar: t}, true
	}
	return tvalue{}, false
}

func isName(t string) bool {
	return t != "" && (t[0] == '[' || t[0] == '_' || isLetter(t[0]))
}

func isScalar(t string) bool {
	return t != "" && !strings.Contains("{}[]:,;", t) && t[0] != '['
}

func isLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// tokenize splits a literal into names, `[extension]` names, strings,
// numbers, and punctuation.
func tokenize(src string) ([]string, bool) {
	var toks []string
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case strings.IndexByte("{}:,;]", c) >= 0:
			toks = append(toks, string(c))
			i++
		case c == '[':
			// A name in brackets is an extension or a type URL; anything
			// else opens a list.
			if j := strings.IndexByte(src[i:], ']'); j > 0 && isBracketName(src[i+1:i+j]) && followsName(toks) {
				toks = append(toks, src[i:i+j+1])
				i += j + 1
				continue
			}
			toks = append(toks, "[")
			i++
		case c == '"' || c == '\'':
			j := i + 1
			for j < len(src) && src[j] != c {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			if j >= len(src) {
				return nil, false
			}
			toks = append(toks, src[i:j+1])
			i = j + 1
		case c == '-' || c == '.' || c == '_' || isLetter(c) || c >= '0' && c <= '9':
			j := i + 1
			for j < len(src) && (src[j] == '.' || src[j] == '_' || src[j] == '+' && (src[j-1] == 'e' || src[j-1] == 'E') || isLetter(src[j]) || src[j] >= '0' && src[j] <= '9') {
				j++
			}
			toks = append(toks, src[i:j])
			i = j
		default:
			return nil, false
		}
	}
	return toks, true
}

// isBracketName reports whether s, the text inside `[...]`, is an extension
// name or a type URL.
func isBracketName(s string) bool {
	if s == "" || !(s[0] == '_' || isLetter(s[0])) {
		return false
	}
	for i := range len(s) {
		c := s[i]
		if !(c == '.' || c == '_' || c == '/' || isLetter(c) || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// followsName reports whether a `[` at this point names a field: at the
// start of a message or after a field ends.
func followsName(toks []string) bool {
	if len(toks) == 0 {
		return false
	}
	switch toks[len(toks)-1] {
	case ":", "[":
		return false
	}
	return true
}
