package result

type R[T any] interface {
	GetError() error
	GetValue() T
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

// GetValue implements R
func (r *result[T]) GetValue() T {
	return r.v
}

// IsErr implements R
func (r *result[T]) IsErr() bool {
	return r.err != nil
}

// IsOk implements R
func (r *result[T]) IsOk() bool {
	return r.err == nil
}

// Unwrap implements R
func (r *result[T]) Unwrap() (T, error) {
	return r.v, r.err
}

func Ok[T any](v T) R[T] {
	return &result[T]{v: v}
}

func Err[T any](err error) R[T] {
	return &result[T]{err: err}
}
