package uml

type Opt[T any] interface {
	~func(*T) error
}

func Apply[T Opt[V], V any](options V, opts ...T) (*V, error) {
	for _, opt := range opts {
		if err := opt(&options); err != nil {
			return nil, err
		}
	}

	return &options, nil
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
