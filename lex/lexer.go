package lex

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Lexer scans TDL source text into a stream of [Token]s.
type Lexer struct {
	filename string
	src      string

	offset    int // offset of ch
	rdOffset  int // offset after ch
	line      int
	lineStart int // offset of the start of the current line

	ch rune // current character, or -1 at EOF
}

// New returns a Lexer over src, reporting positions against filename.
func New(filename, src string) *Lexer {
	l := &Lexer{
		filename: filename,
		src:      src,
		line:     1,
	}
	l.next()
	return l
}

const eof = -1

func (l *Lexer) next() {
	if l.rdOffset >= len(l.src) {
		l.offset = len(l.src)
		l.ch = eof
		return
	}
	r, w := utf8.DecodeRuneInString(l.src[l.rdOffset:])
	if l.ch == '\n' {
		l.line++
		l.lineStart = l.rdOffset
	}
	l.offset = l.rdOffset
	l.rdOffset += w
	l.ch = r
}

func (l *Lexer) peekByte() byte {
	if l.rdOffset < len(l.src) {
		return l.src[l.rdOffset]
	}
	return 0
}

func (l *Lexer) pos() Position {
	return Position{
		Filename: l.filename,
		Line:     l.line,
		Col:      l.offset - l.lineStart + 1,
		Offset:   l.offset,
	}
}

func isLetter(ch rune) bool {
	return ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

// Next scans and returns the next token. It returns an EOF token forever
// once the end of input is reached.
func (l *Lexer) Next() Token {
	l.skipWhitespaceAndComments()

	pos := l.pos()

	switch ch := l.ch; {
	case ch == eof:
		return Token{Kind: EOF, Pos: pos}
	case isLetter(ch):
		return l.scanIdent(pos)
	case isDigit(ch):
		return l.scanNumber(pos)
	case ch == '"':
		return l.scanString(pos)
	}

	ch := l.ch
	l.next()

	switch ch {
	case '{':
		return Token{Kind: LBRACE, Text: "{", Pos: pos}
	case '}':
		return Token{Kind: RBRACE, Text: "}", Pos: pos}
	case '(':
		return Token{Kind: LPAREN, Text: "(", Pos: pos}
	case ')':
		return Token{Kind: RPAREN, Text: ")", Pos: pos}
	case '[':
		return Token{Kind: LBRACK, Text: "[", Pos: pos}
	case ']':
		return Token{Kind: RBRACK, Text: "]", Pos: pos}
	case '<':
		return Token{Kind: LT, Text: "<", Pos: pos}
	case '>':
		return Token{Kind: GT, Text: ">", Pos: pos}
	case ':':
		return Token{Kind: COLON, Text: ":", Pos: pos}
	case ',':
		return Token{Kind: COMMA, Text: ",", Pos: pos}
	case '=':
		return Token{Kind: EQUAL, Text: "=", Pos: pos}
	case '?':
		return Token{Kind: QUESTION, Text: "?", Pos: pos}
	case '.':
		return Token{Kind: DOT, Text: ".", Pos: pos}
	case '@':
		return Token{Kind: AT, Text: "@", Pos: pos}
	default:
		return Token{Kind: ILLEGAL, Text: string(ch), Pos: pos}
	}
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
			l.next()
		}
		if l.ch == '/' && l.peekByte() == '/' {
			for l.ch != '\n' && l.ch != eof {
				l.next()
			}
			continue
		}
		break
	}
}

func (l *Lexer) scanIdent(pos Position) Token {
	start := l.offset
	for isLetter(l.ch) || isDigit(l.ch) {
		l.next()
	}
	text := l.src[start:l.offset]
	return Token{Kind: LookupIdent(text), Text: text, Pos: pos}
}

func (l *Lexer) scanNumber(pos Position) Token {
	start := l.offset
	for isDigit(l.ch) {
		l.next()
	}
	kind := INT
	if l.ch == '.' && isDigit(l.peekRuneAfterDot()) {
		kind = FLOAT
		l.next() // consume '.'
		for isDigit(l.ch) {
			l.next()
		}
	}
	return Token{Kind: kind, Text: l.src[start:l.offset], Pos: pos}
}

func (l *Lexer) peekRuneAfterDot() rune {
	if l.rdOffset < len(l.src) {
		r, _ := utf8.DecodeRuneInString(l.src[l.rdOffset:])
		return r
	}
	return eof
}

func (l *Lexer) scanString(pos Position) Token {
	l.next() // consume opening quote
	var b strings.Builder
	for l.ch != '"' {
		if l.ch == eof || l.ch == '\n' {
			return Token{Kind: ILLEGAL, Text: fmt.Sprintf("unterminated string starting at %s", pos), Pos: pos}
		}
		if l.ch == '\\' {
			l.next()
			switch l.ch {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				return Token{Kind: ILLEGAL, Text: fmt.Sprintf("invalid escape sequence \\%c at %s", l.ch, l.pos()), Pos: pos}
			}
			l.next()
			continue
		}
		b.WriteRune(l.ch)
		l.next()
	}
	l.next() // consume closing quote
	return Token{Kind: STRING, Text: b.String(), Pos: pos}
}
