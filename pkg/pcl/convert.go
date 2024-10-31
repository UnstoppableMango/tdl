package pcl

import (
	"errors"
	"strings"

	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	tdl "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

// FromPcl implements PclConverter.
func FromPcl(pcl schema.PackageSpec) (*tdl.Spec, error) {
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
