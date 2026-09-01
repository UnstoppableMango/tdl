// Package ebnf lints the grammar files under docs/.
//
// They are Wirth syntax notation rather than ISO 14977, so no ISO tool
// reads them. golang.org/x/exp/ebnf does: it documents this exact dialect,
// including the convention that an upper-case name is a nonterminal and a
// lower-case one is lexical. Parsing and reachability come from there.
//
// What is here is the check no library makes. A quoted terminal has to be
// text lex turns into exactly one token, so a spelling the grammar invents
// is an error rather than a rule that could never match.
package ebnf

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/exp/ebnf"

	"github.com/unstoppablemango/tdl/lex"
)

// Options say how to read one grammar file.
type Options struct {
	// Start is the production everything must be reachable from.
	Start string

	// LexSpellings requires every quoted terminal to be text lex turns
	// into exactly one token. It holds for a grammar of TDL and not for a
	// grammar of anything else.
	LexSpellings bool

	// Annotated reads the `/*@ ... */` comments and holds the file to
	// them: every production with no expression needs a token binding,
	// and every name an annotation mentions has to exist. A grammar
	// carrying no annotations is not one of these.
	Annotated bool
}

// A File is a grammar and what its annotations say about it.
type File struct {
	Grammar     ebnf.Grammar
	Annotations Annotations
}

// GrammarOptions reads docs/grammar.ebnf.
var GrammarOptions = Options{Start: "File", LexSpellings: true, Annotated: true}

// NotationOptions reads docs/notation.ebnf, which describes the notation
// rather than TDL, so its quoted terminals are not TDL tokens.
var NotationOptions = Options{Start: "Grammar"}

// Lint reports every problem it finds.
func Lint(filename, src string, opts Options) []error {
	_, errs := Read(filename, src, opts)
	return errs
}

// Read parses a grammar and its annotations, reporting every problem it
// finds along the way.
//
// A file that did not parse returns no grammar and its parse errors
// alone, since nothing about its contents would be worth saying.
func Read(filename, src string, opts Options) (*File, []error) {
	// Before the library, because it hands its scanner no error handler:
	// text/scanner then prints an unterminated comment or string to
	// stderr and the parser reports whatever the damage looks like
	// downstream. Saying it here keeps the diagnostic and the noise out.
	if errs := checkLexical(filename, src); len(errs) > 0 {
		return nil, errs
	}

	grammar, err := ebnf.Parse(filename, strings.NewReader(src))
	if err != nil {
		return nil, flatten(err)
	}

	errs := flatten(ebnf.Verify(grammar, opts.Start))
	if opts.LexSpellings {
		errs = append(errs, checkSpellings(grammar)...)
		errs = append(errs, checkReservedWords(grammar)...)
	}

	file := &File{Grammar: grammar}
	if opts.Annotated {
		annotations, more := readAnnotations(filename, src, grammar)
		file.Annotations = annotations
		errs = append(errs, more...)
	}

	sort.SliceStable(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return file, errs
}

// checkLexical reports what the library's scanner would print rather
// than return. A Go string cannot span a line, so an unterminated one is
// a quote with no closing quote before the newline.
func checkLexical(filename, src string) []error {
	line, lineStart := 1, 0
	at := func(i int) string {
		return fmt.Sprintf("%s:%d:%d", filename, line, i-lineStart+1)
	}

	for i := 0; i < len(src); {
		switch {
		case src[i] == '\n':
			line++
			i++
			lineStart = i
		case strings.HasPrefix(src[i:], "/*"):
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return []error{fmt.Errorf("%s: comment not terminated", at(i))}
			}
			for _, c := range src[i : i+2+end+2] {
				if c == '\n' {
					line++
				}
			}
			i += 2 + end + 2
			lineStart = i
		case strings.HasPrefix(src[i:], "//"):
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case src[i] == '"':
			j := i + 1
			for j < len(src) && src[j] != '"' && src[j] != '\n' {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			if j >= len(src) || src[j] != '"' {
				return []error{fmt.Errorf("%s: string not terminated", at(i))}
			}
			i = j + 1
		default:
			i++
		}
	}
	return nil
}

// flatten turns the library's joined error back into its parts, since a
// list of positions reads better than one string and tests better too.
func flatten(err error) []error {
	if err == nil {
		return nil
	}
	if list, ok := err.(interface{ Unwrap() []error }); ok {
		return list.Unwrap()
	}
	return []error{err}
}

// checkSpellings holds the grammar to the lexer.
func checkSpellings(grammar ebnf.Grammar) []error {
	var errs []error
	for _, name := range sorted(grammar) {
		walk(grammar[name].Expr, func(expr ebnf.Expression) {
			tok, ok := expr.(*ebnf.Token)
			if !ok || lexesAsOneToken(tok.String) {
				return
			}
			errs = append(errs, fmt.Errorf("%s: %q in %s is not text the lexer produces",
				tok.Pos(), tok.String, name))
		})
	}
	return errs
}

// checkReservedWords holds the reserved_word production to lex.Keywords.
// The grammar spells the set out rather than naming it, so a keyword added
// to lex and not to the grammar, or the other way round, is an error here
// rather than a difference nobody notices.
//
// A grammar without the production is not checked: not every grammar in
// this notation is about TDL declarations.
func checkReservedWords(grammar ebnf.Grammar) []error {
	prod, ok := grammar["reserved_word"]
	if !ok {
		return nil
	}

	spelled := map[string]bool{}
	walk(prod.Expr, func(expr ebnf.Expression) {
		if tok, ok := expr.(*ebnf.Token); ok {
			spelled[tok.String] = true
		}
	})

	var errs []error
	for _, kw := range lex.Keywords() {
		if !spelled[kw] {
			errs = append(errs, fmt.Errorf("%s: reserved_word is missing %q, which lex reserves",
				prod.Pos(), kw))
		}
		delete(spelled, kw)
	}
	for _, extra := range sortedSet(spelled) {
		errs = append(errs, fmt.Errorf("%s: reserved_word has %q, which lex does not reserve",
			prod.Pos(), extra))
	}
	return errs
}

// lexesAsOneToken reports whether text is the whole of exactly one token.
// It is not lex.Lookup, because "_" is a legal terminal the lexer scans as
// an identifier rather than as a fixed spelling.
func lexesAsOneToken(text string) bool {
	l := lex.New("grammar.ebnf", text)
	first := l.Next()
	if first.Kind == lex.ILLEGAL || first.Text != text {
		return false
	}
	return l.Next().Kind == lex.EOF
}

func walk(expr ebnf.Expression, fn func(ebnf.Expression)) {
	if expr == nil {
		return
	}
	fn(expr)
	switch e := expr.(type) {
	case ebnf.Alternative:
		for _, x := range e {
			walk(x, fn)
		}
	case ebnf.Sequence:
		for _, x := range e {
			walk(x, fn)
		}
	case *ebnf.Group:
		walk(e.Body, fn)
	case *ebnf.Option:
		walk(e.Body, fn)
	case *ebnf.Repetition:
		walk(e.Body, fn)
	case *ebnf.Range:
		walk(e.Begin, fn)
		walk(e.End, fn)
	}
}

func sortedSet(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sorted(grammar ebnf.Grammar) []string {
	names := make([]string, 0, len(grammar))
	for name := range grammar {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
