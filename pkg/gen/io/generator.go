package io

import (
	"io"

	"github.com/unstoppablemango/tdl/pkg/pipe"
	"github.com/unstoppablemango/tdl/pkg/tdl"
	"github.com/unstoppablemango/tdl/pkg/tdl/constraint"
	"github.com/unstoppablemango/tdl/pkg/tdl/spec"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type reader[T any] constraint.Pipeline[*tdlv1alpha1.Spec, T]

type SpecReader[T any] struct {
	tdl.Pipeline[*tdlv1alpha1.Spec, T]
	options []spec.ReaderOption
}

// Execute implements tdl.Pipeline.
func (p *SpecReader[T]) Execute(reader io.Reader, sink T) error {
	spec, err := spec.ReadAll(reader)
	if err != nil {
		return err
	}

	return p.Pipeline.Execute(spec, sink)
}

func ReadSpec[T any, P reader[T]](
	pipeline P,
	options ...spec.ReaderOption,
) tdl.Pipeline[io.Reader, T] {
	return &SpecReader[T]{
		Pipeline: pipe.Lift(pipeline),
		options:  options,
	}
}
