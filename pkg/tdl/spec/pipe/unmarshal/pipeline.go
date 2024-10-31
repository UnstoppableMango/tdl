package unmarshal

import (
	"fmt"
	"io"

	"github.com/unstoppablemango/tdl/pkg/tdl"
	"github.com/unstoppablemango/tdl/pkg/tdl/spec"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type (
	Input[T any]  tdl.Pipeline[*tdlv1alpha1.Spec, T]
	Output[T any] tdl.Pipeline[io.Reader, T]

	// The name doesn't match the API, but I'll fix that later
	Unmarshaler func([]byte) (*tdlv1alpha1.Spec, error)
)

func With[T any, I Input[T], O Output[T]](pipeline I, unmarshal Unmarshaler) O {
	return func(r io.Reader, t T) error {
		data, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("reading spec: %w", err)
		}

		spec, err := unmarshal(data)
		if err != nil {
			return fmt.Errorf("unmarshaling spec: %w", err)
		}

		return pipeline(spec, t)
	}
}

func WithMediaType[T any, I Input[T], O Output[T]](pipeline I, media tdl.MediaType) O {
	return With[T, I, O](pipeline, func(b []byte) (*tdlv1alpha1.Spec, error) {
		return spec.FromMediaType(media, b)
	})
}
