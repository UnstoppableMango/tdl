package tdl

import (
	"fmt"
	"io"
	"iter"

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

type Target interface {
	Dependencies() iter.Seq[Target]
}

type MediaType string

// String implements fmt.Stringer.
func (m MediaType) String() string {
	return string(m)
}

type Token struct {
	Path string
	Url  string
}

// String implements fmt.Stringer.
func (t Token) String() string {
	return t.Path
}

var _ fmt.Stringer = Token{}