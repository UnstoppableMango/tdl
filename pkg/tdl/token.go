package tdl

import "fmt"

type Token struct {
	Name string
}

// String implements fmt.Stringer.
func (t Token) String() string {
	return t.Name
}

var _ fmt.Stringer = Token{}
