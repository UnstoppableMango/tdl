package unmarshal

import (
	"fmt"
	"io"

	"github.com/unstoppablemango/tdl/pkg/tdl"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type (
	Input[T any]  tdl.Pipeline[*tdlv1alpha1.Spec, T]
	Output[T any] tdl.Pipeline[io.Reader, T]
	Unmarshaler   func([]byte, *tdlv1alpha1.Spec) error
)

func With[T any, I Input[T], O Output[T]](pipeline I, unmarshal Unmarshaler) O {
	return func(r io.Reader, t T) error {
		data, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("reading spec: %w", err)
		}

		var spec tdlv1alpha1.Spec
		if err = unmarshal(data, &spec); err != nil {
			return fmt.Errorf("unmarshaling spec: %w", err)
		}

		return pipeline(&spec, t)
	}
}

func WithMediaType[T any, I Input[T], O Output[T]](pipeline I, media tdl.MediaType) O {
	return With[T, I, O](pipeline, func(b []byte, s *tdlv1alpha1.Spec) error {
		panic("unimplemented")
	})
}
