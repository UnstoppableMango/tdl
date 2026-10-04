// Package ebnf lints the grammar files under docs/, which are Wirth syntax
// notation as read by golang.org/x/exp/ebnf. Parsing and reachability come
// from that library; this package adds the check that every quoted terminal
// is text lex turns into exactly one token.
package ebnf

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"golang.org/x/exp/ebnf"

	"github.com/unstoppablemango/tdl/lex"
)

// Options say how to read one grammar file.
type Options struct {
	// Start is the production everything must be reachable from.
	Start string

	// LexSpellings requires every quoted terminal to be text lex turns
	// into exactly one token.
	LexSpellings bool

	// Annotated reads the `/*@ ... */` comments and holds the file to
	// them: every production with no expression needs a token binding,
	// and every name an annotation mentions has to exist.
	Annotated bool
}

// A File is a grammar and what its annotations say about it.
type File struct {
	Grammar     ebnf.Grammar
	Annotations Annotations
}

// GrammarOptions reads docs/grammar.ebnf.
var GrammarOptions = Options{Start: "File", LexSpellings: true, Annotated: true}

// NotationOptions reads docs/notation.ebnf, whose terminals are not TDL
// tokens.
var NotationOptions = Options{Start: "Grammar"}

// ReadFile is [Read] over a file on disk, with every problem joined into
// one error.
func ReadFile(path string, opts Options) (*File, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	file, errs := Read(path, string(src), opts)
	if len(errs) > 0 {
		return nil, fmt.Errorf("%s: %d problems:\n%w", path, len(errs), errors.Join(errs...))
	}
	return file, nil
}

// Read parses a grammar and its annotations, reporting every problem it
// finds. A file that does not parse returns no grammar and only its parse
// errors.
func Read(filename, src string, opts Options) (*File, []error) {
	// The library gives text/scanner no error handler, so it would print
	// these to stderr instead of returning them.
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

	slices.SortStableFunc(errs, func(a, b error) int { return cmp.Compare(a.Error(), b.Error()) })
	return file, errs
}

// checkLexical reports an unterminated comment or string. A string cannot
// span a line.
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

// flatten splits the library's joined error into its parts.
func flatten(err error) []error {
	if err == nil {
		return nil
	}
	if list, ok := err.(interface{ Unwrap() []error }); ok {
		return list.Unwrap()
	}
	return []error{err}
}

func checkSpellings(grammar ebnf.Grammar) []error {
	var errs []error
	for _, name := range slices.Sorted(maps.Keys(grammar)) {
		Walk(grammar[name].Expr, func(expr ebnf.Expression) {
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

// checkReservedWords holds the reserved_word production, when the grammar
// has one, to lex.Keywords.
func checkReservedWords(grammar ebnf.Grammar) []error {
	prod, ok := grammar["reserved_word"]
	if !ok {
		return nil
	}

	spelled := map[string]bool{}
	Walk(prod.Expr, func(expr ebnf.Expression) {
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
	for _, extra := range slices.Sorted(maps.Keys(spelled)) {
		errs = append(errs, fmt.Errorf("%s: reserved_word has %q, which lex does not reserve",
			prod.Pos(), extra))
	}
	return errs
}

// lexesAsOneToken reports whether text is the whole of exactly one token.
// lex.Lookup would reject "_", which scans as an identifier.
func lexesAsOneToken(text string) bool {
	l := lex.New("grammar.ebnf", text)
	first := l.Next()
	if first.Kind == lex.ILLEGAL || first.Text != text {
		return false
	}
	return l.Next().Kind == lex.EOF
}

// Walk calls fn on expr and then on everything inside it, in source order.
func Walk(expr ebnf.Expression, fn func(ebnf.Expression)) {
	if expr == nil {
		return
	}
	fn(expr)
	switch e := expr.(type) {
	case ebnf.Alternative:
		for _, x := range e {
			Walk(x, fn)
		}
	case ebnf.Sequence:
		for _, x := range e {
			Walk(x, fn)
		}
	case *ebnf.Group:
		Walk(e.Body, fn)
	case *ebnf.Option:
		Walk(e.Body, fn)
	case *ebnf.Repetition:
		Walk(e.Body, fn)
	case *ebnf.Range:
		Walk(e.Begin, fn)
		Walk(e.End, fn)
	}
}
