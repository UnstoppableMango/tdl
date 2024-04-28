package pcl

import (
	"context"
	"io"

	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	tdl "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

type converter struct{}

type PclConverter interface {
	uml.Converter
	FromPcl(ctx context.Context, pcl schema.PackageSpec) (*tdl.Spec, error)
	ToPcl(ctx context.Context, spec tdl.Spec) (*schema.PackageSpec, error)
}

var Converter uml.Converter = &converter{}

// From implements uml.Converter.
func (c *converter) From(ctx context.Context, reader io.Reader) (*tdl.Spec, error) {
	panic("unimplemented")
}

// To implements uml.Converter.
func (c *converter) To(ctx context.Context, spec *tdl.Spec, writer io.Writer) error {
	panic("unimplemented")
}
