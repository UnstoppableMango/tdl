package gen

import (
	"context"

	"github.com/unstoppablemango/tdl/pkg/result"
)

func MapI[A, B, Output any](x GeneratorFunc[A, Output], f func(B) result.R[A]) GeneratorFunc[B, Output] {
	return func(ctx context.Context, b B, output Output) error {
		return result.IterE(f(b), func(a A) error {
			return x.Gen(ctx, a, output)
		})
	}
}

func MapO[A, B, Input any](x GeneratorFunc[Input, A], f func(B) result.R[A]) GeneratorFunc[Input, B] {
	return func(ctx context.Context, input Input, b B) error {
		return result.IterE(f(b), func(a A) error {
			return x.Gen(ctx, input, a)
		})
	}
}
