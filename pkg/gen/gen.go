package gen

import (
	"context"
)

type GeneratorFunc[I, O any] func(context.Context, I, O) error

type Generator[I, O any] struct {
	run GeneratorFunc[I, O]
}

func New[I, O any](g GeneratorFunc[I, O]) Generator[I, O] {
	return Generator[I, O]{run: g}
}

func (g Generator[I, O]) Generate(ctx context.Context, input I, output O) error {
	return g.run(ctx, input, output)
}
