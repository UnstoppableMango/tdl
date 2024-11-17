package tdl

import (
	"context"
	"fmt"
	"io"
	"iter"

	"github.com/spf13/afero"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Sink interface {
	WriteUnit(string, io.Reader) error
}

type Pipeline[T, V any] interface {
	Execute(context.Context, T, V) error
}

type SinkGenerator interface {
	Pipeline[*tdlv1alpha1.Spec, Sink]
}

type Generator interface {
	fmt.Stringer
	Pipeline[*tdlv1alpha1.Spec, afero.Fs]
}

type Plugin interface {
	fmt.Stringer
	Generator(Target) (SinkGenerator, error)
}

type Target interface {
	fmt.Stringer
	Choose([]SinkGenerator) (SinkGenerator, error)
	Plugins() iter.Seq[Plugin]
}

type MediaType string

// String implements fmt.Stringer.
func (m MediaType) String() string {
	return string(m)
}

type Input interface {
	io.Reader
	MediaType() MediaType
}
