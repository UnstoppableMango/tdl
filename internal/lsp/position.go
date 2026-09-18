package lsp

import (
	"strings"
	"unicode/utf8"

	"go.lsp.dev/protocol"

	"github.com/unstoppablemango/tdl/ast"
)

// lineIndex maps between the positions the compiler produces and the ones
// the protocol carries.
//
// The two disagree twice. A lex.Position counts lines and columns from 1
// and measures a column in bytes; a protocol.Position counts from 0 and
// measures a character in UTF-16 code units, because the version of the
// protocol this server speaks has no way to ask for anything else.
//
// This is the only place in the package that knows either fact.
type lineIndex struct {
	text  string
	lines []int // byte offset of the start of each line
}

// newLineIndex indexes the line starts in text.
func newLineIndex(text string) *lineIndex {
	lines := []int{0}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			lines = append(lines, i+1)
		}
	}
	return &lineIndex{text: text, lines: lines}
}

// offset returns the byte offset of a 1-based line and byte column.
//
// This is for the position that reaches the server without an offset:
// ir.Position is what a lowering diagnostic carries, and the proto message
// has a line and a column and nothing else. A position naming a line or a
// column the text does not have returns -1, which happens when a
// diagnostic is about a file other than the one being indexed.
func (x *lineIndex) offset(line, col int) int {
	if line < 1 || line > len(x.lines) || col < 1 {
		return -1
	}

	off := x.lines[line-1] + col - 1
	if off > x.lineEnd(line-1) {
		return -1
	}
	return off
}

// lineEnd returns the byte offset of the newline ending a 0-based line, or
// of the end of the text for the last one.
func (x *lineIndex) lineEnd(line int) int {
	if line+1 < len(x.lines) {
		return x.lines[line+1] - 1
	}
	return len(x.text)
}

// position converts a byte offset to a protocol position.
//
// An offset past the end of the text clamps to the end. An editor clamps
// one anyway, and reporting a diagnostic at the end of the file beats
// dropping it because its position was implausible.
func (x *lineIndex) position(off int) protocol.Position {
	if off < 0 {
		off = 0
	}
	if off > len(x.text) {
		off = len(x.text)
	}

	line := 0
	for line+1 < len(x.lines) && x.lines[line+1] <= off {
		line++
	}
	return protocol.Position{
		Line:      uint32(line),
		Character: uint32(utf16Len(x.text[x.lines[line]:off])),
	}
}

// span is the range covering n bytes from a byte offset.
func (x *lineIndex) span(off, n int) protocol.Range {
	return protocol.Range{Start: x.position(off), End: x.position(off + n)}
}

// wordSpan is the range a diagnostic at a byte offset covers.
//
// The compiler reports a point and a diagnostic wants a range, so this
// covers the word the offset lands in. There is not always one: a
// diagnostic can point at a punctuation mark or at the end of the file,
// and the empty range that results renders as a caret, which is the honest
// answer when the compiler only knows where something started.
func (x *lineIndex) wordSpan(off int) protocol.Range {
	return x.span(off, wordLen(x.text, off))
}

// nameSpan is the range covering name, declared at pos.
//
// A declaration's position is where it starts, which is the keyword that
// declares it: ast.DeclHead carries one position and `type Email: string`
// begins at `type`. Jumping to the keyword is right and highlighting it is
// not, so this finds the name on that line and covers that instead.
//
// A type parameter's position is already the name, which this finds at
// once. When the name is not on the line, the range is the empty one at
// the declaration, which still puts the cursor in the right place.
func (x *lineIndex) nameSpan(pos ast.Position, name string) (protocol.Range, bool) {
	off := x.offset(pos.Line, pos.Col)
	if off < 0 {
		return protocol.Range{}, false
	}

	line := x.text[off:x.lineEnd(pos.Line-1)]
	if i := wordIndex(line, name); i >= 0 {
		return x.span(off+i, len(name)), true
	}
	return x.span(off, 0), true
}

// wordIndex is the offset of name in text as a whole word, or -1.
//
// Whole word, because a declaration line holds more than the name:
// searching `type UserKey: string` for `User` must not match inside
// `UserKey`.
func wordIndex(text, name string) int {
	if name == "" {
		return -1
	}

	for i := 0; ; {
		j := strings.Index(text[i:], name)
		if j < 0 {
			return -1
		}
		at := i + j

		before := at == 0 || !isWordRune(rune(text[at-1]), false)
		end := at + len(name)
		after := end == len(text) || !isWordRune(rune(text[end]), false)
		if before && after {
			return at
		}
		i = at + 1
	}
}

// byteOffset turns a protocol position into a byte offset into the text,
// which is what a request carrying a cursor has to be converted to before
// anything the compiler produced can be compared against it.
//
// A character past the end of its line clamps to the end of that line,
// which is what the protocol says a client may send.
func (x *lineIndex) byteOffset(pos protocol.Position) int {
	line := int(pos.Line)
	if line >= len(x.lines) {
		return len(x.text)
	}

	start := x.lines[line]
	text := x.text[start:x.lineEnd(line)]

	want, units := int(pos.Character), 0
	for i, r := range text {
		if units >= want {
			return start + i
		}
		units++
		if r > 0xFFFF {
			units++ // encoded as a surrogate pair
		}
	}
	return start + len(text)
}

// wordLen is the length in bytes of the identifier starting at off, or 0
// when nothing there looks like one.
func wordLen(text string, off int) int {
	if off < 0 || off >= len(text) {
		return 0
	}

	n := 0
	for n < len(text)-off {
		r, size := utf8.DecodeRuneInString(text[off+n:])
		if !isWordRune(r, n == 0) {
			break
		}
		n += size
	}
	return n
}

// isWordRune reports whether r continues an identifier, matching what the
// lexer accepts rather than what Unicode calls a letter: a diagnostic
// underlining more than the token it is about is worse than one
// underlining less.
func isWordRune(r rune, first bool) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
		return true
	case r >= '0' && r <= '9':
		return !first
	default:
		return false
	}
}

// utf16Len is the length of s in UTF-16 code units, which is what a
// protocol position counts.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n++
		if r > 0xFFFF {
			n++
		}
	}
	return n
}
