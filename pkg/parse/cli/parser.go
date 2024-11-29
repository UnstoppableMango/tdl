package cli

import (
	"context"

	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type parser struct {
	name string
	args []string
}

func (p parser) Execute(ctx context.Context) (*tdlv1alpha1.Spec, error) {
	panic("unimplemented")
}
