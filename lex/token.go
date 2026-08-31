// Package lex implements the TDL lexer, turning source text into a stream
// of tokens for the parser.
package lex

import "fmt"

// Kind identifies the lexical class of a [Token].
type Kind int

const (
	ILLEGAL Kind = iota
	EOF

	IDENT  // identifiers and keywords share a scanning path; Kind distinguishes them
	DOC    // /// doc comment, text only
	STRING // "..."
	INT    // 123
	FLOAT  // 1.23
	REGEX  // /.../, scanned only on demand; see [Lexer.RescanRegexAt]

	// Reserved keywords. Declaration keywords are reserved; modifiers and
	// constraint names (key, owned, deprecated, min, length, ...) are
	// contextual and lex as IDENT.
	PACKAGE
	IMPORT
	AS
	PRIMITIVE
	UNIT
	ALIAS
	TYPE
	VALUE
	ENTITY
	ENUM
	CLASS
	MIXIN
	INSTANCE
	TARGET
	FOR
	REQUIRES
	WHERE
	INCLUDE
	NULL
	TRUE
	FALSE
	UNION // reserved, not yet implemented by the parser

	// Punctuation. The sentinels bound the range so [Punctuation] does not
	// restate the list; a new operator declared between them is picked up.
	punctBeg
	LBRACE   // {
	RBRACE   // }
	LPAREN   // (
	RPAREN   // )
	LBRACK   // [
	RBRACK   // ]
	LT       // <
	GT       // >
	COLON    // :
	COMMA    // ,
	EQUAL    // =
	QUESTION // ?
	DOT      // .
	PIPE     // |
	ARROW    // ->
	RANGE    // ..
	CARET    // ^
	STAR     // *
	SLASH    // /
	FATARROW // =>
	punctEnd
)

var keywords = map[string]Kind{
	"package":   PACKAGE,
	"import":    IMPORT,
	"as":        AS,
	"primitive": PRIMITIVE,
	"unit":      UNIT,
	"alias":     ALIAS,
	"type":      TYPE,
	"value":     VALUE,
	"entity":    ENTITY,
	"enum":      ENUM,
	"class":     CLASS,
	"mixin":     MIXIN,
	"instance":  INSTANCE,
	"target":    TARGET,
	"for":       FOR,
	"requires":  REQUIRES,
	"where":     WHERE,
	"include":   INCLUDE,
	"null":      NULL,
	"true":      TRUE,
	"false":     FALSE,
	"union":     UNION,
}

var kindNames = map[Kind]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",
	IDENT:   "IDENT",
	DOC:     "DOC",
	STRING:  "STRING",
	INT:     "INT",
	FLOAT:   "FLOAT",
	REGEX:   "REGEX",

	LBRACE:   "{",
	RBRACE:   "}",
	LPAREN:   "(",
	RPAREN:   ")",
	LBRACK:   "[",
	RBRACK:   "]",
	LT:       "<",
	GT:       ">",
	COLON:    ":",
	COMMA:    ",",
	EQUAL:    "=",
	QUESTION: "?",
	DOT:      ".",
	PIPE:     "|",
	ARROW:    "->",
	RANGE:    "..",
	CARET:    "^",
	STAR:     "*",
	SLASH:    "/",
	FATARROW: "=>",
}

func init() {
	for text, kind := range keywords {
		kindNames[kind] = text
	}
}

// LookupIdent returns the keyword Kind for ident, or IDENT if ident is not a
// reserved keyword.
func LookupIdent(ident string) Kind {
	if kind, ok := keywords[ident]; ok {
		return kind
	}
	return IDENT
}

// IsKeyword reports whether text is a reserved keyword.
func IsKeyword(text string) bool {
	_, ok := keywords[text]
	return ok
}

func (k Kind) String() string {
	if name, ok := kindNames[k]; ok {
		return name
	}
	return fmt.Sprintf("Kind(%d)", int(k))
}

// Position identifies a location in a source file.
type Position struct {
	Filename string
	Line     int // 1-based
	Col      int // 1-based, in bytes
	Offset   int // 0-based byte offset
}

func (p Position) String() string {
	if p.Filename == "" {
		return fmt.Sprintf("%d:%d", p.Line, p.Col)
	}
	return fmt.Sprintf("%s:%d:%d", p.Filename, p.Line, p.Col)
}

// Token is a single lexical token.
type Token struct {
	Kind Kind
	Text string // literal source text; decoded for STRING, body only for DOC and REGEX
	Pos  Position
}
