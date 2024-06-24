package uml

import (
	"context"
	"io"
)

type ConverterOptions struct {
	MediaType *string
}

type ConverterOption func(*ConverterOptions) error

type Converter interface {
	From(ctx context.Context, reader io.Reader) (*Spec, error)
}

type NewConverter[T any] interface {
	RunnerFactory[T, Converter]
}

func WithMediaType(t string) ConverterOption {
	return func(opts *ConverterOptions) error {
		opts.MediaType = &t
		return nil
	}
}
