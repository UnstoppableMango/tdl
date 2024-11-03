package pipe

import "github.com/unstoppablemango/tdl/pkg/constraint"

// WTF am I doing

func Map[
	T, V, X, Y any,
	A constraint.Pipeline[T, V],
	B constraint.Pipeline[X, Y],
](p A, f func(X, Y) (T, V)) B {
	return func(x X, y Y) error {
		return p(f(x, y))
	}
}

func MapA[
	A, B, T any,
	PA constraint.Pipeline[A, T],
	PB constraint.Pipeline[B, T],
](p PA, f func(B) A) PB {
	return Map[A, T, B, T, PA, PB](p,
		func(b B, t T) (A, T) {
			return f(b), t
		},
	)
}

func MapB[
	A, B, T any,
	PA constraint.Pipeline[T, A],
	PB constraint.Pipeline[T, B],
](p PA, f func(B) A) PB {
	return Map[T, A, T, B, PA, PB](p,
		func(t T, b B) (T, A) {
			return t, f(b)
		},
	)
}
