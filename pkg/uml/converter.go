package uml

import (
	"context"
	"io"
)

type ConverterOptions struct {
	MimeType *string
}

type ConverterOption func(*ConverterOptions) error

type ConvertFrom interface {
	From(ctx context.Context, reader io.Reader) (*Spec, error)
}

type ConvertTo interface {
	To(ctx context.Context, spec *Spec, writer io.Writer) error
}

type Converter interface {
	ConvertFrom
	ConvertTo
}

type (
	NewConvertFrom func(ConverterOptions) ConvertFrom
	NewConvertTo   func(ConverterOptions) ConvertTo
	NewConverter   func(ConverterOptions) Converter
)

func WithMimeType(t string) ConverterOption {
	return func(opts *ConverterOptions) error {
		opts.MimeType = &t
		return nil
	}
}

func ConvFrom(opts []ConverterOption) ConverterOptions {
	return Apply(ConverterOptions{}, opts)
}
