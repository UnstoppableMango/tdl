package pipe

import (
	"io"

	"github.com/unmango/go/option"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	"github.com/unstoppablemango/tdl/pkg/sink"
	"github.com/unstoppablemango/tdl/pkg/spec"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type PipelineOptions struct {
	MediaType mediatype.Option
}

type PipelineOption func(*PipelineOptions)

type specReader[T any] struct {
	tdl.Pipeline[*tdlv1alpha1.Spec, T]
	options []spec.ReaderOption
}

// Execute implements tdl.Pipeline.
func (p *specReader[T]) Execute(reader io.Reader, output T) error {
	spec, err := spec.ReadAll(reader, p.options...)
	if err != nil {
		return err
	}

	return p.Pipeline.Execute(spec, output)
}

type SinkWriter[T any] struct {
	tdl.Pipeline[T, tdl.Sink]
}

// Execute implements tdl.Pipeline.
func (s *SinkWriter[T]) Execute(input T, writer io.Writer) error {
	sink := sink.WriteTo(writer)
	return s.Pipeline.Execute(input, sink)
}

func ReadSpec[T any](
	pipeline tdl.Pipeline[*tdlv1alpha1.Spec, T],
	options ...spec.ReaderOption,
) tdl.Pipeline[io.Reader, T] {
	return &specReader[T]{
		Pipeline: pipeline,
		options:  options,
	}
}

func WriteSink[T any](
	pipeline tdl.Pipeline[T, tdl.Sink],
) tdl.Pipeline[T, io.Writer] {
	return &SinkWriter[T]{
		Pipeline: pipeline,
	}
}

func NewPipeline(
	input io.Reader,
	output io.Writer,
	options ...PipelineOption,
) IO {
	// opts := Options(options...)
	panic("unimplemented")
}

func Options(options ...PipelineOption) PipelineOptions {
	opts := DefaultOptions()
	option.ApplyAll(&opts, options)
	return opts
}

func DefaultOptions() PipelineOptions {
	return PipelineOptions{
		MediaType: func() tdl.MediaType {
			return mediatype.ApplicationProtobuf
		},
	}
}
