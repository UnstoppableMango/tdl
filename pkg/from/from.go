package from

import (
	"context"

	"github.com/unstoppablemango/tdl/pkg/result"
)

type ConverterFunc[I, O any] func(context.Context, I) result.R[O]

type Converter[I, O any] interface {
	From(context.Context, I) result.R[O]
}

type converter[I, O any] struct {
	run ConverterFunc[I, O]
}

// From implements Converter.
func (c converter[I, O]) From(ctx context.Context, input I) result.R[O] {
	return c.run(ctx, input)
}

func New[I, O any](c ConverterFunc[I, O]) Converter[I, O] {
	return converter[I, O]{run: c}
}
