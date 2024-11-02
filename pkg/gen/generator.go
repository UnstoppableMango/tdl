package gen

import (
	"io"
	"iter"

	"github.com/unstoppablemango/tdl/pkg/tdl"
	"github.com/unstoppablemango/tdl/pkg/tdl/constraint"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Func func(*tdlv1alpha1.Spec, tdl.Sink) error

type Source interface {
	Units() iter.Seq[string]
	Reader(string) (io.Reader, error)
}

type Pipe interface {
	tdl.Sink
	Source
}

func (f Func) Execute(spec *tdlv1alpha1.Spec, sink tdl.Sink) error {
	return f(spec, sink)
}

func Lift[G constraint.Gen](fn G) tdl.Generator {
	return Func(fn)
}

func New(gen Func) tdl.Generator {
	return Lift(gen)
}
