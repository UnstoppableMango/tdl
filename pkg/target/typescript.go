package target

import (
	"context"
	"errors"

	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type typescript string

// Generator implements tdl.Target.
func (t typescript) Generator(available iter.Seq[tdl.Plugin]) (tdl.Generator, error) {
	for p := range available {
		if p.String() == "uml2ts" {
			return p.Generator(context.TODO(), t)
		}
	}

	return nil, errors.New("no suitable plugin")
}

var TypeScript typescript = "TypeScript"

// Choose implements tdl.Target.
func (t typescript) Choose(available []tdl.SinkGenerator) (tdl.SinkGenerator, error) {
	if len(available) == 0 {
		return nil, errors.New("no generators to choose from")
	}

	errs := []error{}
	for _, g := range available {
		return g, nil
	}

	return nil, errors.Join(errs...)
}

// String implements tdl.Target.
func (t typescript) String() string {
	return string(t)
}
