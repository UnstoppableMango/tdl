package gen

import (
	"io"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/spec"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

func ReadAll(generator tdl.Generator, options ...spec.ReaderOption) FromReader {
	return func(r io.Reader, s tdl.Sink) error {
		if spec, err := spec.ReadAll(r, options...); err != nil {
			return err
		} else {
			return generator.Execute(spec, s)
		}
	}
}

func WriteTo(generator tdl.Generator) ToWriter {
	return func(s *tdlv1alpha1.Spec, w io.Writer) error {
		return generator.Execute(s, nil) // TODO
	}
}
