package gen

import (
	"io"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/pipe"
	"github.com/unstoppablemango/tdl/pkg/sink"
	"github.com/unstoppablemango/tdl/pkg/spec"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type ioPipe struct {
	input  FromReader
	output ToWriter
}

func PipeFromReader(generator tdl.Generator, options ...spec.ReaderOption) FromReader {
	return func(r io.Reader, s tdl.Sink) error {
		if spec, err := spec.ReadAll(r, options...); err != nil {
			return err
		} else {
			return generator.Execute(spec, s)
		}
	}
}

func PipeToWriter(generator tdl.Generator) ToWriter {
	return func(s *tdlv1alpha1.Spec, w io.Writer) error {
		return generator.Execute(s, sink.WriteTo(w))
	}
}

func PipeIO(generator tdl.Generator, options ...spec.ReaderOption) pipe.IO {
	return func(r io.Reader, w io.Writer) error {
		spec, err := spec.ReadAll(r, options...)
		if err != nil {
			return err
		}

		return generator.Execute(spec, sink.WriteTo(w))
	}
}
