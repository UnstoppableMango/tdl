package io

import (
	"io"

	"github.com/unstoppablemango/tdl/pkg/tdl"
	"github.com/unstoppablemango/tdl/pkg/tdl/spec"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

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
