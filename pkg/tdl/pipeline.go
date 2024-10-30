package tdl

type Pipeline[T, V any] interface {
	~func(T, V) error
}

type Lookup[T, V any, P Pipeline[T, V]] func(string) (P, error)

// WTF

func Map[
	T, V, X, Y any,
	A Pipeline[T, V],
	B Pipeline[X, Y],
](p A, f func(X, Y) (T, V)) B {
	return func(x X, y Y) error {
		return p(f(x, y))
	}
}

func MapA[
	A, B, T any,
	PA Pipeline[A, T],
	PB Pipeline[B, T],
](p PA, f func(B) A) PB {
	return Map[A, T, B, T, PA, PB](p,
		func(b B, t T) (A, T) {
			return f(b), t
		},
	)
}

func MapB[
	A, B, T any,
	PA Pipeline[T, A],
	PB Pipeline[T, B],
](p PA, f func(B) A) PB {
	return Map[T, A, T, B, PA, PB](p,
		func(t T, b B) (T, A) {
			return t, f(b)
		},
	)
}
