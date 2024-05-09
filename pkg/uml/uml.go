package uml

import (
	"context"
	"io"

	tdl "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1"
)

type Type = tdl.Type
type Spec = tdl.Spec

type ConverterOptions struct {
	MimeType *string
}

type GeneratorOptions struct{}

type ConverterOption func(*ConverterOptions)
type GeneratorOption func(*GeneratorOptions)

type Converter interface {
	From(ctx context.Context, reader io.Reader, opts ...ConverterOption) (*Spec, error)
}

type Generator interface {
	Gen(ctx context.Context, spec *Spec, writer io.Writer, opts ...GeneratorOption) error
}

func WithMimeType(t string) ConverterOption {
	return func(opts *ConverterOptions) {
		opts.MimeType = &t
	}
}

type opt[T any] interface {
	func(*T)
}

func Apply[T opt[V], V any](options *V, opts ...T) {
	for _, opt := range opts {
		opt(options)
	}
}
