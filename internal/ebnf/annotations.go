package ebnf

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/ebnf"

	"github.com/unstoppablemango/tdl/lex"
)

// Annotations are what docs/grammar.ebnf says about itself that the
// notation cannot: which productions are structure and which are
// plumbing, where a parser generator needs precedence or has to consider
// two readings, and which lex symbol defines a name the grammar leaves
// undefined.
//
// They live in comments, which golang.org/x/exp/ebnf drops, so they are
// scanned separately and attached by position. See docs/design/treesitter.md.
type Annotations struct {
	// Word is the production a generator extracts keywords from.
	Word string

	// Extras may appear between any two tokens. Each is a name with no
	// production, bound by a file-level token directive.
	Extras []string

	// Tokens maps a name to the pattern the named lex symbol holds.
	Tokens map[string]string

	// Conflicts are pairs of productions a generator has to consider
	// together.
	Conflicts [][]string

	// Prods holds what is said about each production, by name.
	Prods map[string]ProdAnnotations
}

// ProdAnnotations are the annotations on one production.
type ProdAnnotations struct {
	Hidden   bool   // structure the tree without appearing in it
	Inline   bool   // substitute into callers instead
	External bool   // a terminal a hand-written scanner produces
	Assoc    string // "", "left", or "right"
	Prec     int    // meaningful when Assoc is set or Prec is non-zero
	HasPrec  bool
	Token    string // the pattern the named lex symbol holds
}

// lexSymbols maps the name an annotation uses to the lex value it means.
//
// Written out rather than looked up by reflection, so renaming one of
// these in lex breaks this build rather than a generated grammar at run
// time, which is the whole reason the annotation names a symbol instead
// of repeating a pattern.
var lexSymbols = map[string]string{
	"IdentPattern":       lex.IdentPattern,
	"StringPattern":      lex.StringPattern,
	"IntPattern":         lex.IntPattern,
	"FloatPattern":       lex.FloatPattern,
	"RegexPattern":       lex.RegexPattern,
	"DocPattern":         lex.DocPattern,
	"LineCommentPattern": lex.LineCommentPattern,
}

// annotation is one comment, before it is understood.
type annotation struct {
	fields []string
	line   int
	pos    string
}

// scanAnnotations finds every `/*@ ... */` comment. It ignores the rest
// of the file, including ordinary comments and anything inside a string,
// which is why it tracks those rather than searching for the opener.
func scanAnnotations(filename, src string) []annotation {
	var out []annotation
	line := 1

	for i := 0; i < len(src); {
		switch {
		case src[i] == '\n':
			line++
			i++
		case strings.HasPrefix(src[i:], "/*"):
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return out // checkLexical reports this
			}
			body := src[i+2 : i+2+end]
			if strings.HasPrefix(body, "@") {
				out = append(out, annotation{
					fields: strings.Fields(body[1:]),
					line:   line,
					pos:    fmt.Sprintf("%s:%d", filename, line),
				})
			}
			line += strings.Count(src[i:i+2+end+2], "\n")
			i += 2 + end + 2
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
			i = j + 1
		default:
			i++
		}
	}
	return out
}

// readAnnotations understands the comments and attaches each to whatever
// it describes.
//
// A production annotation belongs to the next production in the file,
// which is what "precedes the production it describes" means once
// comments have been dropped and only line numbers are left.
func readAnnotations(filename, src string, grammar ebnf.Grammar) (Annotations, []error) {
	a := Annotations{
		Tokens: map[string]string{},
		Prods:  map[string]ProdAnnotations{},
	}
	var errs []error

	for _, ann := range scanAnnotations(filename, src) {
		if len(ann.fields) == 0 {
			errs = append(errs, fmt.Errorf("%s: empty annotation", ann.pos))
			continue
		}

		keyword, args := ann.fields[0], ann.fields[1:]
		switch keyword {
		case "word", "extra", "conflict":
			errs = append(errs, a.file(ann, keyword, args)...)
		case "token":
			// `token name = Symbol` binds a name with no production;
			// `token Symbol` binds the production it precedes.
			if len(args) == 3 && args[1] == "=" {
				errs = append(errs, a.file(ann, keyword, args)...)
				continue
			}
			fallthrough
		case "hidden", "inline", "external", "prec", "prec.left", "prec.right":
			name, err := nextProduction(ann, grammar)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			errs = append(errs, a.prod(ann, name, keyword, args)...)
		default:
			errs = append(errs, fmt.Errorf("%s: unknown annotation %q", ann.pos, keyword))
		}
	}

	return a, append(errs, a.check(grammar)...)
}

func (a *Annotations) file(ann annotation, keyword string, args []string) []error {
	switch keyword {
	case "word":
		if len(args) != 1 {
			return []error{fmt.Errorf("%s: word takes one production", ann.pos)}
		}
		if a.Word != "" {
			return []error{fmt.Errorf("%s: word is already %q", ann.pos, a.Word)}
		}
		a.Word = args[0]
	case "extra":
		if len(args) == 0 {
			return []error{fmt.Errorf("%s: extra takes at least one name", ann.pos)}
		}
		a.Extras = append(a.Extras, args...)
	case "conflict":
		if len(args) < 2 {
			return []error{fmt.Errorf("%s: conflict takes at least two productions", ann.pos)}
		}
		a.Conflicts = append(a.Conflicts, args)
	case "token":
		pattern, ok := lexSymbols[args[2]]
		if !ok {
			return []error{fmt.Errorf("%s: %s is not a lex symbol this reads", ann.pos, args[2])}
		}
		a.Tokens[args[0]] = pattern
	}
	return nil
}

func (a *Annotations) prod(ann annotation, name, keyword string, args []string) []error {
	p := a.Prods[name]
	defer func() { a.Prods[name] = p }()

	switch keyword {
	case "hidden":
		if len(args) != 0 {
			return []error{fmt.Errorf("%s: hidden takes no arguments", ann.pos)}
		}
		p.Hidden = true
	case "inline":
		if len(args) != 0 {
			return []error{fmt.Errorf("%s: inline takes no arguments", ann.pos)}
		}
		p.Inline = true
	case "external":
		if len(args) != 0 {
			return []error{fmt.Errorf("%s: external takes no arguments", ann.pos)}
		}
		p.External = true
	case "token":
		if len(args) != 1 {
			return []error{fmt.Errorf("%s: token takes one lex symbol", ann.pos)}
		}
		pattern, ok := lexSymbols[args[0]]
		if !ok {
			return []error{fmt.Errorf("%s: %s is not a lex symbol this reads", ann.pos, args[0])}
		}
		p.Token = pattern
	case "prec", "prec.left", "prec.right":
		if len(args) != 1 {
			return []error{fmt.Errorf("%s: %s takes one level", ann.pos, keyword)}
		}
		level, err := strconv.Atoi(args[0])
		if err != nil {
			return []error{fmt.Errorf("%s: %s takes a number, not %q", ann.pos, keyword, args[0])}
		}
		p.Prec, p.HasPrec = level, true
		p.Assoc = strings.TrimPrefix(strings.TrimPrefix(keyword, "prec"), ".")
	}
	return nil
}

// nextProduction finds what a production annotation describes: the first
// production starting below it.
func nextProduction(ann annotation, grammar ebnf.Grammar) (string, error) {
	best, bestLine := "", 0
	for name, prod := range grammar {
		line := prod.Name.Pos().Line
		if line > ann.line && (best == "" || line < bestLine) {
			best, bestLine = name, line
		}
	}
	if best == "" {
		return "", fmt.Errorf("%s: %s has no production below it", ann.pos, ann.fields[0])
	}
	return best, nil
}

// check reports what only the finished set can say.
func (a *Annotations) check(grammar ebnf.Grammar) []error {
	var errs []error

	if a.Word != "" {
		if _, ok := grammar[a.Word]; !ok {
			errs = append(errs, fmt.Errorf("word names %s, which is not a production", a.Word))
		}
	}

	for _, extra := range a.Extras {
		if _, ok := a.Tokens[extra]; !ok {
			errs = append(errs, fmt.Errorf("extra names %s, which no token annotation binds", extra))
		}
	}

	for _, pair := range a.Conflicts {
		for _, name := range pair {
			if _, ok := grammar[name]; !ok {
				errs = append(errs, fmt.Errorf("conflict names %s, which is not a production", name))
			}
		}
	}

	for _, name := range sorted(grammar) {
		p := a.Prods[name]
		if p.Hidden && p.Inline {
			errs = append(errs, fmt.Errorf("%s is both hidden and inline", name))
		}
		// A production with no expression is the lexer's, and the token
		// annotation is the only thing saying which part of the lexer.
		if grammar[name].Expr == nil && p.Token == "" {
			errs = append(errs, fmt.Errorf("%s: %s has no expression and no token annotation",
				grammar[name].Pos(), name))
		}
	}

	sort.SliceStable(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errs
}
