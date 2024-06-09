package uml

import "context"

type Runner interface {
	Converter
	Generator
}

type RunnerFactory[T any, V any] interface {
	func(context.Context, T, []string) (V, error)
}
