package gen

import (
	"context"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/constraint"
	"github.com/unstoppablemango/tdl/pkg/pipe"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Func func(context.Context, *tdlv1alpha1.Spec, tdl.Sink) error

type (
	FromInput  pipe.FromInput[tdl.Sink]
	FromFs     pipe.FromFs[tdl.Sink]
	FromReader pipe.FromReader[tdl.Sink]
	ToWriter   pipe.ToWriter[*tdlv1alpha1.Spec]
)

func (f Func) Execute(
	ctx context.Context,
	spec *tdlv1alpha1.Spec,
	sink tdl.Sink,
) error {
	return f(ctx, spec, sink)
}

func Lift[G constraint.Gen](fn G) tdl.SinkGenerator {
	return Func(fn)
}

func New(gen Func) tdl.SinkGenerator {
	return Lift(gen)
}
