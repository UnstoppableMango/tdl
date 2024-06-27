package result

func Iter[T any](r R[T], f func(T)) error {
	x, err := r.Unwrap()
	if err == nil {
		f(x)
	}

	return err
}
