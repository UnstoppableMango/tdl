package result

type R[T any] interface {
	GetError() error
	GetResult() T
	IsErr() bool
	IsOk() bool
	Unwrap() (T, error)
}

type result[T any] struct {
	err error
	v   T
}

// GetError implements R
func (r *result[T]) GetError() error {
	return r.err
}

// GetResult implements R
func (r *result[T]) GetResult() T {
	return r.v
}

func (r *result[T]) IsErr() bool {
	return r.err != nil
}

func (r *result[T]) IsOk() bool {
	return r.err == nil
}

func (r *result[T]) Unwrap() (T, error) {
	return r.v, r.err
}

func Of[T any](v T) R[T] {
	return &result[T]{v: v}
}

func OfErr[T any](err error) R[T] {
	return &result[T]{err: err}
}
