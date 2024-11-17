package gen

import (
	"context"
	"io"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/pipe"
	"github.com/unstoppablemango/tdl/pkg/sink"
	"github.com/unstoppablemango/tdl/pkg/spec"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

func NoOp(context.Context, *tdlv1alpha1.Spec, tdl.Sink) error { return nil }

func PipeFromReader(generator tdl.SinkGenerator, options ...spec.ReaderOption) FromReader {
	return func(ctx context.Context, r io.Reader, s tdl.Sink) error {
		if spec, err := spec.ReadAll(r, options...); err != nil {
			return err
		} else {
			return generator.Execute(ctx, spec, s)
		}
	}
}

func PipeToWriter(generator tdl.SinkGenerator) ToWriter {
	return func(ctx context.Context, s *tdlv1alpha1.Spec, w io.Writer) error {
		return generator.Execute(ctx, s, sink.WriteTo(w))
	}
}

func PipeIO(generator tdl.SinkGenerator, options ...spec.ReaderOption) pipe.IO {
	return func(ctx context.Context, r io.Reader, w io.Writer) error {
		spec, err := spec.ReadAll(r, options...)
		if err != nil {
			return err
		}

		return generator.Execute(ctx, spec, sink.WriteTo(w))
	}
}
