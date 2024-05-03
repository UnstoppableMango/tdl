package uml

import (
	"context"
	"io"
)

type Generator interface {
	Gen(ctx context.Context, spec *Spec, writer io.Writer) error
}

type GeneratorOptions struct {
	Target string
}

type (
	GeneratorOption func(*GeneratorOptions) error
	NewGenerator    func(GeneratorOptions) Generator
)

func WithTarget(t string) GeneratorOption {
	return func(opts *GeneratorOptions) error {
		opts.Target = t
		return nil
	}
}

func GenFrom(opts []GeneratorOption) GeneratorOptions {
	return Apply(GeneratorOptions{}, opts)
}
