package spec

import (
	"encoding/json"
	"fmt"

	mt "github.com/unstoppablemango/tdl/pkg/media"
	"github.com/unstoppablemango/tdl/pkg/tdl"
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

func ToJson(spec *tdlv1alpha1.Spec) ([]byte, error) {
	return json.Marshal(spec)
}

func ToProto(spec *tdlv1alpha1.Spec) ([]byte, error) {
	return proto.Marshal(spec)
}

func ToYaml(spec *tdlv1alpha1.Spec) ([]byte, error) {
	return yaml.Marshal(spec)
}

func ToMediaType(media tdl.MediaType, spec *tdlv1alpha1.Spec) ([]byte, error) {
	return mt.Match(media, mt.Matcher[[]byte]{
		Json: func() ([]byte, error) {
			return ToJson(spec)
		},
		Protobuf: func() ([]byte, error) {
			return ToProto(spec)
		},
		Yaml: func() ([]byte, error) {
			return ToYaml(spec)
		},
		Other: func() ([]byte, error) {
			return nil, mt.UnsupportedErr(media)
		},
	})
}
