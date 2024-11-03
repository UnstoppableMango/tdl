package token

import (
	tdl "github.com/unstoppablemango/tdl/pkg"
)

func Parse(tokenish string) (tdl.Token, error) {
	// This should eventually be more robust
	return tdl.Token{Name: tokenish}, nil
}
