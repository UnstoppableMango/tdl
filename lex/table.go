package lex

import "sort"

// The tables here describe the lexer to a program rather than to a person.
//
// docs/grammar.ebnf names most of its terminals by spelling and leaves the
// six scanned by shape undefined, because both are the lexer's business.
// A tool deriving a second parser from that file resolves them here, so a
// keyword added to this package reaches the derived parser and a spelling
// the grammar invents but the lexer never produces is an error rather than
// a rule that can never match.

// Patterns for the token classes scanned by shape. They are the shapes
// [Lexer.Next] and [Lexer.RescanRegexAt] accept, written as regular
// expressions because a generator consumes them; TestPatternsMatchTheLexer
// holds them to that.
//
// Unanchored, and a caller matching at a position anchors them itself.
const (
	IdentPattern  = `[_A-Za-z][_A-Za-z0-9]*`
	IntPattern    = `-?[0-9]+`
	FloatPattern  = `-?[0-9]+\.[0-9]+`
	StringPattern = `"([^"\\\n]|\\["\\nt])*"`
	DocPattern    = `///[^\n]*`
	RegexPattern  = `/([^/\\\n]|\\[^\n])*/`

	// LineCommentPattern is the shape scanComment consumes and discards.
	// It has no Kind, since the lexer never emits one, but it is a fact
	// about the language a second parser has to know: tree-sitter keeps
	// comments as extras where this one drops them.
	//
	// It also matches a doc comment, because `///` begins with `//`, so a
	// consumer tries DocPattern first. Three slashes or more is a doc
	// comment.
	LineCommentPattern = `//[^\n]*`
)

var patterns = map[Kind]string{
	IDENT:  IdentPattern,
	INT:    IntPattern,
	FLOAT:  FloatPattern,
	STRING: StringPattern,
	DOC:    DocPattern,
	REGEX:  RegexPattern,
}

// Pattern returns a regular expression matching the source text of a token
// class scanned by shape, and "" for every other kind.
//
// A kind with a fixed spelling reports "" rather than a pattern quoting
// itself: [Spelling] is the answer for those, and returning one from both
// would give a caller two ways to ask.
func Pattern(k Kind) string {
	return patterns[k]
}

// Keywords returns every reserved keyword, sorted.
func Keywords() []string {
	out := make([]string, 0, len(keywords))
	for text := range keywords {
		out = append(out, text)
	}
	sort.Strings(out)
	return out
}

// Punctuation returns the spelling of every operator and delimiter, sorted.
func Punctuation() []string {
	var out []string
	for k := punctBeg + 1; k < punctEnd; k++ {
		out = append(out, kindNames[k])
	}
	sort.Strings(out)
	return out
}

// Spelling returns the source text of a kind that has exactly one, and ""
// for a class scanned by shape.
func Spelling(k Kind) string {
	if _, shaped := patterns[k]; shaped {
		return ""
	}
	if k > punctBeg && k < punctEnd {
		return kindNames[k]
	}
	if IsKeyword(kindNames[k]) {
		return kindNames[k]
	}
	return ""
}

// Lookup returns the kind the lexer produces for a fixed spelling, whether
// keyword, operator, or delimiter. It reports false for anything scanned by
// shape, which has no single spelling to look up.
func Lookup(text string) (Kind, bool) {
	if kind, ok := keywords[text]; ok {
		return kind, true
	}
	for k := punctBeg + 1; k < punctEnd; k++ {
		if kindNames[k] == text {
			return k, true
		}
	}
	return ILLEGAL, false
}
