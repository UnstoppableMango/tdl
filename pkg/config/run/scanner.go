package run

import tdl "github.com/unstoppablemango/tdl/pkg"

type Scanner interface {
	Scan() bool
	Err() error
	Input() tdl.Input
	Output() tdl.Output
}

type scanner struct {
	args []string
	i    int
	err  error
}

// Err implements Scanner.
func (s *scanner) Err() error {
	return s.err
}

// Input implements Scanner.
func (s *scanner) Input() tdl.Input {
	panic("unimplemented")
}

// Output implements Scanner.
func (s *scanner) Output() tdl.Output {
	panic("unimplemented")
}

// Scan implements Scanner.
func (s *scanner) Scan() bool {
	if s.i >= len(s.args) {
		return false
	} else {
		s.i++
		return true
	}
}

func NewScanner(args []string) Scanner {
	return &scanner{args, 0, nil}
}
