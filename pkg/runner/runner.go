package runner

import (
	"context"

	"github.com/unstoppablemango/tdl/pkg/from"
	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/result"
)

type Runner[Spec, I, O any] interface {
	from.Converter[I, Spec]
	gen.Generator[Spec, O]
}

type runner[Spec, I, O any] struct {
	Converter from.Converter[I, Spec]
	Generator gen.Generator[Spec, O]
}

// From implements Runner.
func (r runner[Spec, I, O]) From(ctx context.Context, input I) result.R[Spec] {
	return r.Converter.From(ctx, input)
}

// Gen implements Runner.
func (r runner[Spec, I, O]) Gen(ctx context.Context, spec Spec, output O) error {
	return r.Generator.Gen(ctx, spec, output)
}

func New[Spec, I, O any](f from.ConverterFunc[I, Spec], g gen.GeneratorFunc[Spec, O]) Runner[Spec, I, O] {
	return runner[Spec, I, O]{
		Converter: from.New(f),
		Generator: gen.New(g),
	}
}
