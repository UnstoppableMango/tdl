package tdl

import (
	"io"

	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Sink interface {
	WriteUnit(string, io.Reader) error
}

type Pipeline[T, V any] interface {
	Execute(T, V) error
}

type Generator interface {
	Pipeline[*tdlv1alpha1.Spec, Sink]
}

type MediaType string

// String implements fmt.Stringer.
func (m MediaType) String() string {
	return string(m)
}

func WithMediaType(media MediaType) func() MediaType {
	return func() MediaType { return media }
}
