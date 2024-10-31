package tdl

import (
	"io"
	"iter"

	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Sink interface {
	WriteUnit(string, io.Reader) error
}

type Source interface {
	Units() iter.Seq[string]
	Reader(string) (io.Reader, error)
}

type Pipe interface {
	Sink
	Source
}

type Gen func(*tdlv1alpha1.Spec, Sink) error

type Generator interface {
	Execute(*tdlv1alpha1.Spec, Sink) error
}

type MediaType string
