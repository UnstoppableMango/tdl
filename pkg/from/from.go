package from

import (
	"context"

	"github.com/unstoppablemango/tdl/pkg/result"
)

type From[T, V any] interface {
	func(context.Context, T) result.R[V]
}
