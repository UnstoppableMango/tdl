package constraint

import (
	"github.com/unstoppablemango/tdl/pkg/tdl"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Gen interface {
	~func(*tdlv1alpha1.Spec, tdl.Sink) error
}

type Pipeline[T, V any] interface {
	~func(T, V) error
}
