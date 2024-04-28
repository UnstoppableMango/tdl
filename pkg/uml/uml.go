package uml

import (
	"context"
	"io"

	tdl "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1"
)

type Type = tdl.Type
type Spec = tdl.Spec

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

type Generator interface {
	Gen(ctx context.Context, spec *Spec, writer io.Writer) error
}
