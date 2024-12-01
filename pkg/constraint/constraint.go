package constraint

import (
	"context"
	"io"

	"github.com/spf13/afero"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type (
	FromReader[T any] Pipeline[io.Reader, T]
	ToWriter[T any]   Pipeline[T, io.Writer]
	SpecReader[T any] Pipeline[*tdlv1alpha1.Spec, T]
)

type Gen SpecReader[afero.Fs]

type Pipeline[T, V any] interface {
	~func(context.Context, T) (V, error)
}
