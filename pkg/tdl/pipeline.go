package tdl

import "github.com/unmango/go/fp"

type Pipeline[T, V any] interface {
	~func(T, V) error
}

// WTF
func Map[
	T, V, X, Y any,
	A Pipeline[T, V],
	B Pipeline[X, Y],
	F fp.Functor[A, B],
](a A, f F) B {
	return f(a)
}
