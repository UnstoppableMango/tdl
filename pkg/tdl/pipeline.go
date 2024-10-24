package tdl

type Pipeline[T, V any] func(T, V) error
