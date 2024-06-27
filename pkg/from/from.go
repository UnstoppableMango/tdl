package from

import (
	"context"

	"github.com/unstoppablemango/tdl/pkg/result"
)

type ConverterFunc[I, O any] func(context.Context, I) result.R[O]

type Converter[I, O any] struct {
	run ConverterFunc[I, O]
}

func New[I, O any](c ConverterFunc[I, O]) Converter[I, O] {
	return Converter[I, O]{run: c}
}

func (c Converter[I, O]) Convert(ctx context.Context, input I) result.R[O] {
	return c.run(ctx, input)
}
