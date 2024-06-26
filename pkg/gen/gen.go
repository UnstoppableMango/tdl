package gen

import "context"

type Generator[Input, Output any] interface {
	input() Input
	output() Output
}

type generator[I, O any] struct {
	Input  I
	Output O
}

// input implements Generator.
func (g generator[I, O]) input() I {
	return g.Input
}

// output implements Generator.
func (g generator[I, O]) output() O {
	return g.Output
}

var _ Generator[string, string] = generator[string, string]{}

func MapI[A, B, Output any](x Generator[A, Output], f func(A) B) Generator[B, Output] {
	return generator[B, Output]{
		Input:  f(x.input()),
		Output: x.output(),
	}
}

func MapO[A, B, Input any](x Generator[Input, A], f func(A) B) Generator[Input, B] {
	return generator[Input, B]{
		Input:  x.input(),
		Output: f(x.output()),
	}
}

func Run[I, O any](ctx context.Context, g Generator[I, O], input I, output O) error {

}
