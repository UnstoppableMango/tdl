package writer

type monoid[T any] interface {
	Empty() T
	Append(T, T) T
}

type Writer[T any, V monoid[T]] interface{}

type sliceMonoid[T any] struct{}

func (m sliceMonoid[T]) Empty() []T {
	return []T{}
}

func (m sliceMonoid[T]) Append(a []T, b []T) []T {
	return append(a, b...)
}

type intMonoid struct{}

func (m intMonoid) Empty() int {
	return 0
}

func (m intMonoid) Append(a int, b int) int {
	return a + b
}

var _ Writer[int, intMonoid] = intMonoid{}
