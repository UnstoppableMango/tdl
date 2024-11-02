package io

import (
	"io"

	"github.com/unstoppablemango/tdl/pkg/pipe"
	"github.com/unstoppablemango/tdl/pkg/tdl"
	"github.com/unstoppablemango/tdl/pkg/tdl/constraint"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type SpecReader[T any] struct {
	gen tdl.Pipeline[*tdlv1alpha1.Spec, T]
}

// Execute implements tdl.Pipeline.
func (p *SpecReader[T]) Execute(reader io.Reader, sink tdl.Sink) error {
	panic("unimplemented")
}

func WithSpecFrom[
	T any,
	G constraint.Pipeline[*tdlv1alpha1.Spec, T],
](generator G) tdl.Pipeline[io.Reader, tdl.Sink] {
	return &SpecReader[T]{gen: pipe.Lift(generator)}
}
