package result

func Map[A, B any](r R[A], f func(A) B) R[B] {
	if a, err := r.Unwrap(); err != nil {
		return Err[B](err)
	} else {
		return Ok(f(a))
	}
}
