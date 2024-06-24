package uml

import (
	"context"
	"io"
	"log/slog"
)

type GeneratorOptions struct {
	Target string
	Log    *slog.Logger
}

type GeneratorOption func(*GeneratorOptions) error

type Gen[T, V any] interface {
	func(context.Context, *T, V) error
}

type Generator interface {
	Gen(ctx context.Context, spec *Spec, writer io.Writer) error
}

type NewGenerator[T any] interface {
	RunnerFactory[T, Generator]
}

func WithTarget(t string) GeneratorOption {
	return func(opts *GeneratorOptions) error {
		opts.Target = t
		return nil
	}
}
