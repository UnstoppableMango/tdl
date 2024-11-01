package gen

import (
	"github.com/unstoppablemango/tdl/pkg/tdl"
	"github.com/unstoppablemango/tdl/pkg/tdl/constraint"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type funcGenerator[G constraint.Gen] struct {
	generator G
}

func (g funcGenerator[G]) Execute(spec *tdlv1alpha1.Spec, sink tdl.Sink) error {
	return g.generator(spec, sink)
}

func Lift[G constraint.Gen](fn G) tdl.Generator {
	return funcGenerator[G]{fn}
}

func New(gen tdl.Gen) tdl.Generator {
	return Lift(gen)
}
