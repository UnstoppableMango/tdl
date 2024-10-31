package pipeline

import "github.com/unstoppablemango/tdl/pkg/tdl"

// WTF am I doing

func Map[
	T, V, X, Y any,
	A tdl.Pipeline[T, V],
	B tdl.Pipeline[X, Y],
](p A, f func(X, Y) (T, V)) B {
	return func(x X, y Y) error {
		return p(f(x, y))
	}
}

func MapA[
	A, B, T any,
	PA tdl.Pipeline[A, T],
	PB tdl.Pipeline[B, T],
](p PA, f func(B) A) PB {
	return Map[A, T, B, T, PA, PB](p,
		func(b B, t T) (A, T) {
			return f(b), t
		},
	)
}

func MapB[
	A, B, T any,
	PA tdl.Pipeline[T, A],
	PB tdl.Pipeline[T, B],
](p PA, f func(B) A) PB {
	return Map[T, A, T, B, PA, PB](p,
		func(t T, b B) (T, A) {
			return t, f(b)
		},
	)
}
