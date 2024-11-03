package pipe

import (
	"io"

	"github.com/unstoppablemango/tdl/pkg/constraint"
	"github.com/unstoppablemango/tdl/pkg/tdl"
)

type Func[T, V any] func(T, V) error

type (
	FromReader[T any] Func[io.Reader, T]
	ToWriter[T any]   Func[T, io.Writer]
)

type IO Func[io.Reader, io.Writer]

func (f Func[T, V]) Execute(source T, sink V) error {
	return f(source, sink)
}

func Lift[T, V any, P constraint.Pipeline[T, V]](pipeline P) tdl.Pipeline[T, V] {
	return Func[T, V](pipeline)
}
