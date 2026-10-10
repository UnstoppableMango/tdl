package lex

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// Lexer scans TDL source text into a stream of [Token]s. It emits no
// newline tokens; whitespace is insignificant.
type Lexer struct {
	filename string
	src      string

	offset    int // offset of ch
	rdOffset  int // offset after ch
	line      int
	lineStart int // offset of the start of the current line

	ch rune // current character, or -1 at EOF

	comments []Comment
	seen     int // offset after the last comment recorded
}

// Comment is an ordinary `//` comment, which [Lexer.Next] skips and
// collects for the formatter to place by position. A `///` doc comment is a
// DOC token instead.
type Comment struct {
	Text string // the text after the slashes, with one leading space removed
	Pos  Position
	End  int // offset just past the comment's last character
}

// Comments returns a copy of every ordinary comment scanned so far, in
// source order.
func (l *Lexer) Comments() []Comment { return slices.Clone(l.comments) }

// New returns a Lexer over src, reporting positions against filename.
func New(filename, src string) *Lexer {
	l := &Lexer{filename: filename, src: src, line: 1}
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
	tok := l.scan()
	tok.End = l.pos()
	return tok
}

func (l *Lexer) scan() Token {
	for {
		l.skipSpace()

		pos := l.pos()

		switch ch := l.ch; {
		case ch == eof:
			return Token{Kind: EOF, Pos: pos}
		case isLetter(ch):
			return l.scanIdent(pos)
		case isDigit(ch):
			return l.scanNumber(pos, false)
		case ch == '"':
			return l.scanString(pos)
		case ch == '/' && l.peekByte() == '/':
			tok, isDoc := l.scanComment(pos)
			if isDoc {
				return tok
			}
			continue // ordinary comment: skip and keep going
		}

		return l.scanOperator()
	}
}

func (l *Lexer) skipSpace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.next()
	}
}

// scanComment consumes a `//` comment. A comment beginning `///` is a doc
// comment and produces a DOC token holding the text after the slashes with
// one leading space removed; anything else is recorded on the lexer and
// skipped.
func (l *Lexer) scanComment(pos Position) (Token, bool) {
	l.next() // first /
	l.next() // second /
	doc := l.ch == '/'
	if doc {
		l.next()
	}
	start := l.offset
	for l.ch != '\n' && l.ch != eof {
		l.next()
	}

	text := strings.TrimPrefix(l.src[start:l.offset], " ")
	if doc {
		return Token{Kind: DOC, Text: text, Pos: pos}, true
	}

	// RescanRegexAt rewinds the lexer; seen keeps a comment from being
	// collected twice.
	if l.offset > l.seen {
		l.comments = append(l.comments, Comment{Text: text, Pos: pos, End: l.offset})
		l.seen = l.offset
	}
	return Token{}, false
}

func (l *Lexer) scanOperator() Token {
	pos := l.pos()
	ch := l.ch
	l.next()

	switch ch {
	case '-':
		if l.ch == '>' {
			l.next()
			return Token{Kind: ARROW, Text: "->", Pos: pos}
		}
		if isDigit(l.ch) {
			return l.scanNumber(pos, true)
		}
		return Token{Kind: ILLEGAL, Text: "-", Pos: pos}
	case '=':
		if l.ch == '>' {
			l.next()
			return Token{Kind: FATARROW, Text: "=>", Pos: pos}
		}
		return Token{Kind: EQUAL, Text: "=", Pos: pos}
	case '.':
		if l.ch == '.' {
			l.next()
			return Token{Kind: RANGE, Text: "..", Pos: pos}
		}
		return Token{Kind: DOT, Text: ".", Pos: pos}
	}

	// Every remaining operator is one character. Slicing the source avoids
	// an allocation and keeps a multi-byte illegal character whole.
	text := l.src[pos.Offset:l.offset]
	if kind, ok := Lookup(text); ok {
		return Token{Kind: kind, Text: text, Pos: pos}
	}
	return Token{Kind: ILLEGAL, Text: text, Pos: pos}
}

func (l *Lexer) scanIdent(pos Position) Token {
	start := l.offset
	for isLetter(l.ch) || isDigit(l.ch) {
		l.next()
	}
	text := l.src[start:l.offset]
	return Token{Kind: LookupIdent(text), Text: text, Pos: pos}
}

// scanNumber scans an integer or float. A '.' continues the number only
// when a digit follows, so `1..2` is an integer and a range operator.
func (l *Lexer) scanNumber(pos Position, negative bool) Token {
	start := pos.Offset
	if !negative {
		start = l.offset
	}
	for isDigit(l.ch) {
		l.next()
	}
	kind := INT
	if l.ch == '.' && isDigit(l.peekRune()) {
		kind = FLOAT
		l.next()
		for isDigit(l.ch) {
			l.next()
		}
	}
	return Token{Kind: kind, Text: l.src[start:l.offset], Pos: pos}
}

func (l *Lexer) peekRune() rune {
	if l.rdOffset < len(l.src) {
		r, _ := utf8.DecodeRuneInString(l.src[l.rdOffset:])
		return r
	}
	return eof
}

func (l *Lexer) scanString(pos Position) Token {
	l.next() // opening quote
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
	l.next() // closing quote
	return Token{Kind: STRING, Text: b.String(), Pos: pos}
}

// RescanRegexAt rescans the input from pos as a regex literal and leaves the
// lexer positioned after it.
//
// `/` is also division in a unit expression, and only the parser knows
// which one it wants. Every other token is scanned without context.
func (l *Lexer) RescanRegexAt(pos Position) Token {
	tok := l.rescanRegexAt(pos)
	tok.End = l.pos()
	return tok
}

func (l *Lexer) rescanRegexAt(pos Position) Token {
	l.reset(pos)
	if l.ch != '/' {
		return Token{Kind: ILLEGAL, Text: string(l.ch), Pos: pos}
	}
	l.next() // opening slash
	var b strings.Builder
	for l.ch != '/' {
		if l.ch == eof || l.ch == '\n' {
			return Token{Kind: ILLEGAL, Text: fmt.Sprintf("unterminated regex starting at %s", pos), Pos: pos}
		}
		if l.ch == '\\' {
			b.WriteRune(l.ch)
			l.next()
			if l.ch == eof || l.ch == '\n' {
				return Token{Kind: ILLEGAL, Text: fmt.Sprintf("unterminated regex starting at %s", pos), Pos: pos}
			}
		}
		b.WriteRune(l.ch)
		l.next()
	}
	l.next() // closing slash
	return Token{Kind: REGEX, Text: b.String(), Pos: pos}
}

func (l *Lexer) reset(pos Position) {
	l.offset = pos.Offset
	l.rdOffset = pos.Offset
	l.line = pos.Line
	l.lineStart = pos.Offset - pos.Col + 1
	l.ch = 0 // not '\n', so next() does not advance the line counter
	l.next()
}
