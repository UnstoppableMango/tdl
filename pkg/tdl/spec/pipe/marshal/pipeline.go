package marshal

import (
	"bytes"
	"fmt"
	"io"

	"github.com/unstoppablemango/tdl/pkg/tdl"
	"github.com/unstoppablemango/tdl/pkg/tdl/constraint"
	"github.com/unstoppablemango/tdl/pkg/tdl/spec"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type (
	Input[T any]  constraint.Pipeline[io.Reader, T]
	Output[T any] constraint.Pipeline[*tdlv1alpha1.Spec, T]
	Marshaler     func(*tdlv1alpha1.Spec) ([]byte, error)
)

func With[T any, I Input[T], O Output[T]](pipeline I, marshal Marshaler) O {
	return func(spec *tdlv1alpha1.Spec, t T) error {
		if data, err := marshal(spec); err != nil {
			return fmt.Errorf("marshaling spec: %w", err)
		} else {
			return pipeline(bytes.NewReader(data), t)
		}
	}
}

func WithMediaType[T any, I Input[T], O Output[T]](pipeline I, media tdl.MediaType) O {
	return With[T, I, O](pipeline, func(s *tdlv1alpha1.Spec) ([]byte, error) {
		return spec.ToMediaType(media, s)
	})
}