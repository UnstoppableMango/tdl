package uml

import (
	"bytes"
	"context"
	"io"
	"log/slog"

	"github.com/unstoppablemango/tdl/pkg/result"
)

type ConverterOptions struct {
	MediaType *string
	Target    *string
	Log       *slog.Logger
}

type ConverterOption func(*ConverterOptions) error

type From[T, V any] interface {
	func(context.Context, T) result.R[V]
}

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

func MapFromInput[A, B, C any, F From[A, C], R From[B, C]](f func(B) A, from F) R {
	return func(ctx context.Context, input B) (C, error) {
		return from(ctx, f(input))
	}
}

func MapFromOutput[A, B, C any, F From[A, B], R From[A, C]](f func(B) C, from F) R {
	return func(ctx context.Context, input A) (C, error) {
		result, err := from(ctx, input)
		if err != nil {
			return nil, err
		}

		return f(result), nil
	}
}

func FromMediaType[F From[io.Reader, *Spec], R From[io.Reader, io.Reader]](mediaType string, from F) R {
	return func(ctx context.Context, reader io.Reader) (io.Reader, error) {
		spec, err := from(ctx, reader)
		if err != nil {
			return nil, err
		}

		data, err := Marshal(mediaType, spec)
		if err != nil {
			return nil, err
		}

		return bytes.NewBuffer(data), nil
	}
}
