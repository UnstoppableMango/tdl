package gen

import (
	"context"
	"errors"
)

type GeneratorFunc[I, O any] func(context.Context, I, O) error

type Generator[Input, Output any] interface {
	gen(context.Context, Input, Output) error
}

type generator[I, O any] struct {
	run GeneratorFunc[I, O]
}

// gen implements Generator.
func (g generator[I, O]) gen(ctx context.Context, input I, output O) error {
	return g.run(ctx, input, output)
}

var _ Generator[string, string] = generator[string, string]{}

func New[I, O any](gen GeneratorFunc[I, O]) Generator[I, O] {
	return generator[I, O]{run: gen}
}

func MapI[A, B, Output any](
	x Generator[A, Output],
	f func(GeneratorFunc[A, Output]) GeneratorFunc[B, Output],
) Generator[B, Output] {
	return generator[B, Output]{
		run: func(ctx context.Context, b B, o Output) error {
			return nil
		},
	}
}

func MapO[A, B, Input any](x Generator[Input, A], f func(A) B) Generator[Input, B] {
	return generator[Input, B]{
		Input:  x.input(),
		Output: f(x.output()),
	}
}

func Run[I, O any](ctx context.Context, g Generator[I, O]) error {
	return errors.New("TODO")
}
