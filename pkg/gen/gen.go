package gen

import (
	"context"
)

type GeneratorFunc[I, O any] func(context.Context, I, O) error

func (f GeneratorFunc[I, O]) Gen(ctx context.Context, input I, output O) error {
	return f(ctx, input, output)
}

type Generator[I, O any] interface {
	Gen(context.Context, I, O) error
}

type generator[I, O any] struct {
	run GeneratorFunc[I, O]
}

// Gen implements Generator.
func (g generator[I, O]) Gen(ctx context.Context, input I, output O) error {
	return g.run(ctx, input, output)
}

func New[I, O any](g GeneratorFunc[I, O]) Generator[I, O] {
	return generator[I, O]{run: g}
}
