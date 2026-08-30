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
	STRING // "..."
	INT    // 123
	FLOAT  // 1.23

	// Keywords
	PACKAGE
	IMPORT
	AS
	TYPE
	ENUM
	UNION // reserved, not yet implemented by the parser
	TRUE
	FALSE

	// Punctuation
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
	AT       // @
)

var keywords = map[string]Kind{
	"package": PACKAGE,
	"import":  IMPORT,
	"as":      AS,
	"type":    TYPE,
	"enum":    ENUM,
	"union":   UNION,
	"true":    TRUE,
	"false":   FALSE,
}

// LookupIdent returns the keyword Kind for ident, or IDENT if ident is not a
// keyword.
func LookupIdent(ident string) Kind {
	if kind, ok := keywords[ident]; ok {
		return kind
	}
	return IDENT
}

func (k Kind) String() string {
	switch k {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case IDENT:
		return "IDENT"
	case STRING:
		return "STRING"
	case INT:
		return "INT"
	case FLOAT:
		return "FLOAT"
	case PACKAGE:
		return "package"
	case IMPORT:
		return "import"
	case AS:
		return "as"
	case TYPE:
		return "type"
	case ENUM:
		return "enum"
	case UNION:
		return "union"
	case TRUE:
		return "true"
	case FALSE:
		return "false"
	case LBRACE:
		return "{"
	case RBRACE:
		return "}"
	case LPAREN:
		return "("
	case RPAREN:
		return ")"
	case LBRACK:
		return "["
	case RBRACK:
		return "]"
	case LT:
		return "<"
	case GT:
		return ">"
	case COLON:
		return ":"
	case COMMA:
		return ","
	case EQUAL:
		return "="
	case QUESTION:
		return "?"
	case DOT:
		return "."
	case AT:
		return "@"
	default:
		return fmt.Sprintf("Kind(%d)", int(k))
	}
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
	Text string // literal source text; for STRING, the decoded value
	Pos  Position
}
