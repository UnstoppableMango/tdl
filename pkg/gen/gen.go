package gen

import "context"

type Gen[T, V any] interface {
	func(context.Context, T, V) error
}
