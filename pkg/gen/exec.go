package gen

import (
	"context"
	"errors"

	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
	uxv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/ux/v1alpha1"
)

func Execute(ctx context.Context, config *uxv1alpha1.RunConfig, plugins iter.Seq[tdl.GeneratorPlugin]) error {
	return errors.New("unimplemented")
}
