package pipe

import c "github.com/unstoppablemango/tdl/pkg/constraint"

// WTF am I doing

func Map[
	A c.Pipeline[T, V],
	T, V, X, Y any,
](p A, f func(X, Y) (T, V)) Func[X, Y] {
	return func(x X, y Y) error {
		return p(f(x, y))
	}
}

func MapA[
	PA c.Pipeline[A, T],
	A, B, T any,
](p PA, f func(B) A) Func[B, T] {
	return Map(p,
		func(b B, t T) (A, T) {
			return f(b), t
		},
	)
}

func MapB[
	PA c.Pipeline[T, A],
	A, B, T any,
](p PA, f func(B) A) Func[T, B] {
	return Map(p,
		func(t T, b B) (T, A) {
			return t, f(b)
		},
	)
}
