package tdl

import (
	"io"

	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Gen func(*tdlv1alpha1.Spec, io.Writer) error

type MediaType string

type Pipeline[T, V any] interface {
	~func(T, V) error
}
