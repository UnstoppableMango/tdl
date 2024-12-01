package pipe

import (
	"context"
	"io"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	c "github.com/unstoppablemango/tdl/pkg/constraint"
)

type Func[T, V any] func(context.Context, T) (V, error)

type (
	FromInput[T any]  Func[tdl.Input, T]
	FromFs[T any]     Func[afero.Fs, T]
	FromReader[T any] Func[io.Reader, T]
	ToWriter[T any]   Func[T, io.Writer]
)

type IO Func[io.Reader, io.Writer]

func (f Func[T, V]) Execute(ctx context.Context, source T) (V, error) {
	return f(ctx, source)
}

func Lift[
	P c.Pipeline[T, V],
	T, V any,
](pipeline P) tdl.Pipeline[T, V] {
	return Func[T, V](pipeline)
}
