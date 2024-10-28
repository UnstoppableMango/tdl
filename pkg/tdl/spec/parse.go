package spec

import (
	"fmt"

	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

func FromYaml(bytes []byte) (*tdlv1alpha1.Spec, error) {
	var spec tdlv1alpha1.Spec
	if err := yaml.Unmarshal(bytes, &spec); err != nil {
		return nil, fmt.Errorf("reading spec: %w", err)
	}

	return &spec, nil
}

func FromProto(bytes []byte) (*tdlv1alpha1.Spec, error) {
	var spec tdlv1alpha1.Spec
	if err := proto.Unmarshal(bytes, &spec); err != nil {
		return nil, fmt.Errorf("reading spec: %w", err)
	}

	return &spec, nil
}
