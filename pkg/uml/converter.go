package uml

import (
	"context"
	"io"
)

type ConverterOptions struct {
	MimeType *string
}

type ConverterOption func(*ConverterOptions) error

type Converter interface {
	From(ctx context.Context, reader io.Reader) (*Spec, error)
}

type NewConverter[T any] interface {
	RunnerFactory[T, Converter]
}

func WithMimeType(t string) ConverterOption {
	return func(opts *ConverterOptions) error {
		opts.MimeType = &t
		return nil
	}
}
