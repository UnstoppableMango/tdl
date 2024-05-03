package uml

type Opt[T any] interface {
	~func(*T) error
}

func Apply[T Opt[V], V any](options V, opts []T) V {
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

func Flat[T Opt[V], V any](opts ...T) T {
	return func(o *V) error {
		for _, opt := range opts {
			if err := opt(o); err != nil {
				return err
			}
		}

		return nil
	}
}
