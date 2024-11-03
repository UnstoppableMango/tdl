package io

import (
	"io"

	"github.com/unmango/go/option"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	"github.com/unstoppablemango/tdl/pkg/pipe"
	"github.com/unstoppablemango/tdl/pkg/spec"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type PipelineOptions struct {
	MediaType mediatype.Option
}

type PipelineOption func(*PipelineOptions)

type SpecReader[T any] struct {
	tdl.Pipeline[*tdlv1alpha1.Spec, T]
	options []spec.ReaderOption
}

// Execute implements tdl.Pipeline.
func (p *SpecReader[T]) Execute(reader io.Reader, sink T) error {
	spec, err := spec.ReadAll(reader, p.options...)
	if err != nil {
		return err
	}

	return p.Pipeline.Execute(spec, sink)
}

func ReadSpec[T any](
	pipeline tdl.Pipeline[*tdlv1alpha1.Spec, T],
	options ...spec.ReaderOption,
) tdl.Pipeline[io.Reader, T] {
	return &SpecReader[T]{
		Pipeline: pipeline,
		options:  options,
	}
}

func NewPipeline(
	token tdl.Token,
	input io.Reader,
	output io.Writer,
	options ...PipelineOption,
) pipe.IO {
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
