package target

import (
	"errors"
	"fmt"
	"strings"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

var ErrUnsupported = errors.New("unsupported target")

func Parse(target string) (tdl.Target, error) {
	switch strings.ToLower(target) {
	case "ts":
		fallthrough
	case "typescript":
		return &TypeScript{}, nil
	default:
		return nil, UnsupportedErr(target)
	}
}

func UnsupportedErr(target string) error {
	return fmt.Errorf("%w: %s", ErrUnsupported, target)
}
