package spec

import (
	"encoding/json"
	"fmt"

	"github.com/unstoppablemango/tdl/pkg/tdl"
	mt "github.com/unstoppablemango/tdl/pkg/tdl/media"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

func FromJson(bytes []byte) (*tdlv1alpha1.Spec, error) {
	var spec tdlv1alpha1.Spec
	if err := json.Unmarshal(bytes, &spec); err != nil {
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

func FromYaml(bytes []byte) (*tdlv1alpha1.Spec, error) {
	var spec tdlv1alpha1.Spec
	if err := yaml.Unmarshal(bytes, &spec); err != nil {
		return nil, fmt.Errorf("reading spec: %w", err)
	}

	return &spec, nil
}

func FromMediaType(media tdl.MediaType, bytes []byte) (*tdlv1alpha1.Spec, error) {
	return mt.Match(media, mt.Matcher[*tdlv1alpha1.Spec]{
		Json: func() (*tdlv1alpha1.Spec, error) {
			return FromJson(bytes)
		},
		Protobuf: func() (*tdlv1alpha1.Spec, error) {
			return FromProto(bytes)
		},
		Yaml: func() (*tdlv1alpha1.Spec, error) {
			return FromYaml(bytes)
		},
		Other: func() (*tdlv1alpha1.Spec, error) {
			return nil, mt.UnsupportedErr(media)
		},
	})
}
