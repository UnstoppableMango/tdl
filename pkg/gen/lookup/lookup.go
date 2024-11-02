package lookup

import (
	"errors"
	"fmt"

	"github.com/unstoppablemango/tdl/pkg/tdl"
)

var ErrNotFound = errors.New("not found")

func Name(token tdl.Token) (tdl.Generator, error) {
	switch token.Name {
	case "ts":
		fallthrough
	case "uml2ts":
		return localRepo("uml2ts")
	}

	return nil, fmt.Errorf("%w: %s", ErrNotFound, token.Name)
}

func Lookup(tokenish string) (tdl.Generator, error) {
	token := tdl.Token{Name: tokenish}

	generator, err := Name(token)
	if err == nil {
		return generator, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("lookup: %w", err)
	}

	generator, err = FromPath(token)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotFound, err)
	}

	return generator, nil
}
