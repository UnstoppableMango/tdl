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

func MapFromInput[A, B, C any, F From[A, C], R From[B, C]](from F, f func(B) A) R {
	return func(ctx context.Context, input B) result.R[C] {
		return from(ctx, f(input))
	}
}

func MapFromOutput[A, B, C any, F From[A, B], R From[A, C]](from F, f func(B) C) R {
	return func(ctx context.Context, input A) result.R[C] {
		return result.Map(from(ctx, input), f)
	}
}

func FromMediaType[F From[io.Reader, *Spec], R From[io.Reader, io.Reader]](mediaType string, from F) R {
	return func(ctx context.Context, reader io.Reader) result.R[io.Reader] {
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
