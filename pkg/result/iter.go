package result

func Iter[T any](r R[T], f func(T)) error {
	x, err := r.Unwrap()
	if err == nil {
		f(x)
	}

	return err
}

// I can't remember what this function is supposed to be called but I need
// this functionality so it's gonna be called IterE until I decide to fix it

func IterE[T any](r R[T], f func(T) error) error {
	x, err := r.Unwrap()
	if err == nil {
		return f(x)
	}

	return err
}
