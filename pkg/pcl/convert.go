package pcl

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	tdl "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

type converter struct{}

type PclConverter interface {
	uml.Converter
	FromPcl(ctx context.Context, pcl schema.PackageSpec) (*tdl.Spec, error)
	ToPcl(ctx context.Context, spec *tdl.Spec) (*schema.PackageSpec, error)
}

var Converter PclConverter = &converter{}

// From implements uml.Converter.
func (c *converter) From(ctx context.Context, reader io.Reader) (*tdl.Spec, error) {
	panic("unimplemented")
}

// FromPcl implements PclConverter.
func (c *converter) FromPcl(ctx context.Context, pcl schema.PackageSpec) (*tdl.Spec, error) {
	spec := tdl.Spec{
		Name:        pcl.Name,
		Source:      pcl.Repository,
		Version:     pcl.Version,
		DisplayName: pcl.DisplayName,
		Description: pcl.Description,
		Labels: map[string]string{
			"keywords": strings.Join(pcl.Keywords, ","),
		},
	}

	for name, typeSpec := range pcl.Types {
		typ, err := FromType(typeSpec)
		if err != nil {
			return nil, err
		}

		spec.Types[name] = typ
	}

	return &spec, nil
}

func FromType(typ schema.ComplexTypeSpec) (*tdl.Type, error) {
	return nil, errors.New("TODO")
}

// To implements uml.Converter.
func (c *converter) To(ctx context.Context, spec *tdl.Spec, writer io.Writer) error {
	panic("unimplemented")
}

// ToPcl implements PclConverter.
func (c *converter) ToPcl(ctx context.Context, spec *tdl.Spec) (*schema.PackageSpec, error) {
	panic("unimplemented")
}
