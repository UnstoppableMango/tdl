package lex

import "sort"

// The tables here describe the lexer to a program: a tool deriving a second
// parser from docs/grammar.ebnf resolves its terminals here.

// Patterns for the token classes scanned by shape, as accepted by
// [Lexer.Next] and [Lexer.RescanRegexAt]. TestPatternsMatchTheLexer holds
// them to the lexer. They are unanchored.
const (
	IdentPattern  = `[_A-Za-z][_A-Za-z0-9]*`
	IntPattern    = `-?[0-9]+`
	FloatPattern  = `-?[0-9]+\.[0-9]+`
	StringPattern = `"([^"\\\n]|\\["\\nt])*"`
	DocPattern    = `///[^\n]*`
	RegexPattern  = `/([^/\\\n]|\\[^\n])*/`

	// LineCommentPattern is the comment the lexer skips; it has no Kind.
	// It also matches a doc comment, so a consumer tries DocPattern first.
	LineCommentPattern = `//[^\n]*`
)

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
