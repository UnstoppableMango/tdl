package gen

import (
	"context"
	"io"

	tdl "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

type generator struct{}

var Go uml.Generator = &generator{}

// Gen implements uml.Generator.
func (g *generator) Gen(ctx context.Context, spec *tdl.Spec, writer io.Writer) error {
	panic("unimplemented")
}
