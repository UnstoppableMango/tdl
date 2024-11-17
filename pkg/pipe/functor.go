package pipe

import (
	"context"

	c "github.com/unstoppablemango/tdl/pkg/constraint"
)

// WTF am I doing

func Map[
	A c.Pipeline[T, V],
	T, V, X, Y any,
](p A, f func(context.Context, X, Y) (T, V)) Func[X, Y] {
	return func(ctx context.Context, x X, y Y) error {
		t, v := f(ctx, x, y)
		return p(ctx, t, v)
	}
}

func MapA[
	PA c.Pipeline[A, T],
	A, B, T any,
](p PA, f func(context.Context, B) A) Func[B, T] {
	return Map(p,
		func(ctx context.Context, b B, t T) (A, T) {
			return f(ctx, b), t
		},
	)
}

func MapB[
	PA c.Pipeline[T, A],
	A, B, T any,
](p PA, f func(context.Context, B) A) Func[T, B] {
	return Map(p,
		func(ctx context.Context, t T, b B) (T, A) {
			return t, f(ctx, b)
		},
	)
}
