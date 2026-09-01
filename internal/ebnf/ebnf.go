// Package ebnf reads the notation docs/grammar.ebnf is written in.
//
// That notation is Wirth syntax notation rather than ISO 14977, so no
// off-the-shelf tool reads the file: an ISO parser fails on its first
// production. docs/notation.ebnf describes the dialect, in itself.
//
// What is here is the scanner and the checks a scanner can make, which is
// most of what goes wrong in a grammar file: a name used and never
// defined, a name defined and never used, a bracket left open, a
// production missing its terminator, and a quoted terminal the lexer
// never produces. The last one is why this package imports lex rather
// than restating the token set.
package ebnf

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/unstoppablemango/tdl/lex"
)

// Options say what a grammar file is allowed to leave undefined and how
// hard to hold it to the lexer.
type Options struct {
	// Terminals are the lower-case names the file uses and does not
	// define. Any other one is a typo.
	Terminals []string

	// LexSpellings requires every quoted terminal to be text lex turns
	// into exactly one token, which is what makes a spelling the grammar
	// invents an error rather than a rule that can never match. It holds
	// for a grammar of TDL and not for a grammar of anything else.
	LexSpellings bool
}

// GrammarOptions lints docs/grammar.ebnf: the six token classes lex scans
// by shape, plus the reserved words Name admits.
var GrammarOptions = Options{
	Terminals: []string{
		"identifier",
		"string_lit",
		"int_lit",
		"float_lit",
		"bool_lit",
		"regex_lit",
		"reserved_word",
	},
	LexSpellings: true,
}

// A Diagnostic is one problem with a grammar file.
type Diagnostic struct {
	Line int
	Msg  string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("%d: %s", d.Line, d.Msg)
}

type tokenKind int

const (
	name tokenKind = iota
	quoted
	punct
)

type token struct {
	kind tokenKind
	text string
	line int
}

// Lint reports every problem it finds, in source order.
func Lint(src string, opts Options) []Diagnostic {
	toks, diags := scan(src)
	prods, more := split(toks)
	diags = append(diags, more...)
	diags = append(diags, checkNames(prods)...)
	diags = append(diags, checkTerminals(prods, opts)...)
	sort.SliceStable(diags, func(i, j int) bool { return diags[i].Line < diags[j].Line })
	return diags
}

func scan(src string) ([]token, []Diagnostic) {
	var toks []token
	var diags []Diagnostic
	line := 1

	for i := 0; i < len(src); {
		switch c := src[i]; {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r':
			i++
		case strings.HasPrefix(src[i:], "(*"):
			end := strings.Index(src[i+2:], "*)")
			if end < 0 {
				diags = append(diags, Diagnostic{line, "unterminated comment"})
				return toks, diags
			}
			line += strings.Count(src[i:i+2+end+2], "\n")
			i += 2 + end + 2
		case c == '"':
			end := strings.IndexByte(src[i+1:], '"')
			if end < 0 {
				diags = append(diags, Diagnostic{line, "unterminated quoted terminal"})
				return toks, diags
			}
			toks = append(toks, token{quoted, src[i+1 : i+1+end], line})
			i += 1 + end + 1
		case isNameByte(c):
			j := i
			for j < len(src) && isNameByte(src[j]) {
				j++
			}
			toks = append(toks, token{name, src[i:j], line})
			i = j
		default:
			toks = append(toks, token{punct, string(c), line})
			i++
		}
	}
	return toks, diags
}

func isNameByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// A production is one `Name = ... .` rule.
type production struct {
	lhs  string
	line int
	body []token
}

// split groups the token stream into productions. A production runs from
// a name followed by `=` to the next `.` outside any bracket, so a `"."`
// terminal in the body cannot end it.
func split(toks []token) ([]production, []Diagnostic) {
	var prods []production
	var diags []Diagnostic

	for i := 0; i < len(toks); {
		if toks[i].kind != name || i+1 >= len(toks) || !isPunct(toks[i+1], "=") {
			diags = append(diags, Diagnostic{toks[i].line, fmt.Sprintf("expected a production, got %q", toks[i].text)})
			i++
			continue
		}

		p := production{lhs: toks[i].text, line: toks[i].line}
		depth, closed := 0, false
		j := i + 2
		for ; j < len(toks); j++ {
			t := toks[j]
			// The next production starting is how a missing terminator
			// shows up, and stopping here keeps it from being reported as
			// an undefined name in this one.
			if depth == 0 && t.kind == name && j+1 < len(toks) && isPunct(toks[j+1], "=") {
				break
			}
			if t.kind == punct {
				switch t.text {
				case "(", "[", "{":
					depth++
				case ")", "]", "}":
					depth--
				case ".":
					if depth == 0 {
						closed = true
						j++
					}
				}
			}
			if closed {
				break
			}
			p.body = append(p.body, t)
		}

		if depth != 0 {
			diags = append(diags, Diagnostic{p.line, fmt.Sprintf("%s: unbalanced brackets", p.lhs)})
		} else if !closed {
			diags = append(diags, Diagnostic{p.line, fmt.Sprintf("%s: missing the closing '.'", p.lhs)})
		}
		prods = append(prods, p)
		i = j
	}
	return prods, diags
}

func isPunct(t token, text string) bool {
	return t.kind == punct && t.text == text
}

// checkNames reports a nonterminal used and never defined, one defined
// twice, and one defined and never used. The first production is the
// start symbol, so nothing referring to it is not a problem.
func checkNames(prods []production) []Diagnostic {
	var diags []Diagnostic
	defined := map[string]int{}
	for _, p := range prods {
		if prev, ok := defined[p.lhs]; ok {
			diags = append(diags, Diagnostic{p.line, fmt.Sprintf("%s: defined again, first at line %d", p.lhs, prev)})
			continue
		}
		defined[p.lhs] = p.line
	}

	used := map[string]bool{}
	for _, p := range prods {
		for _, t := range p.body {
			if t.kind != name || !isNonterminal(t.text) {
				continue
			}
			used[t.text] = true
			if _, ok := defined[t.text]; !ok {
				diags = append(diags, Diagnostic{t.line, fmt.Sprintf("%s: used in %s and never defined", t.text, p.lhs)})
			}
		}
	}

	start := ""
	if len(prods) > 0 {
		start = prods[0].lhs
	}
	for _, p := range prods {
		if p.lhs != start && !used[p.lhs] {
			diags = append(diags, Diagnostic{p.line, fmt.Sprintf("%s: defined and never used", p.lhs)})
		}
	}
	return diags
}

// checkTerminals holds the grammar to the lexer. A lower-case name must
// be a token class this package knows, and a quoted terminal must be text
// the lexer turns into exactly one token, so a spelling the grammar
// invents is a rule that could never match.
func checkTerminals(prods []production, opts Options) []Diagnostic {
	known := make(map[string]bool, len(opts.Terminals))
	for _, t := range opts.Terminals {
		known[t] = true
	}

	var diags []Diagnostic
	for _, p := range prods {
		for _, t := range p.body {
			switch {
			case t.kind == name && !isNonterminal(t.text) && !known[t.text]:
				diags = append(diags, Diagnostic{t.line, fmt.Sprintf("%s: unknown terminal in %s", t.text, p.lhs)})
			case t.kind == quoted && opts.LexSpellings:
				if !lexesAsOneToken(t.text) {
					diags = append(diags, Diagnostic{t.line, fmt.Sprintf("%q in %s is not text the lexer produces", t.text, p.lhs)})
				}
			}
		}
	}
	return diags
}

func lexesAsOneToken(text string) bool {
	l := lex.New("grammar.ebnf", text)
	first := l.Next()
	if first.Kind == lex.ILLEGAL || first.Text != text {
		return false
	}
	return l.Next().Kind == lex.EOF
}

func isNonterminal(s string) bool {
	return s != "" && unicode.IsUpper(rune(s[0]))
}
