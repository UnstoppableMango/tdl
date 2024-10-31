package token

import "github.com/unstoppablemango/tdl/pkg/tdl"

func Parse(tokenish string) (tdl.Token, error) {
	// This should eventually be more robust
	return tdl.Token{Name: tokenish}, nil
}
