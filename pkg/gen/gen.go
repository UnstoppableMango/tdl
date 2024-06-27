package gen

import (
	"context"
)

type GeneratorFunc[I, O any] func(context.Context, I, O) error
