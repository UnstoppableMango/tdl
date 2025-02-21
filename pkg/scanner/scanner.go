package scanner

import (
	"github.com/unstoppablemango/tdl/pkg/token"
)

type Scanner struct {
	offset int
	ch     rune
	src    []byte
}

func New(src []byte) *Scanner {
	return &Scanner{src: src}
}

func next() {
	
}

func (s *Scanner) Scan() (pos token.Pos, tok token.Token, lit string) {
	return 0, 0, ""
}
