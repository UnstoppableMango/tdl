package tdl

type Pipeline[T, V any] interface {
	~func(T, V) error
}
