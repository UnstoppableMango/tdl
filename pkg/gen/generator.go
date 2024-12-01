package gen

import (
	"context"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/constraint"
	"github.com/unstoppablemango/tdl/pkg/pipe"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Func func(context.Context, *tdlv1alpha1.Spec) (afero.Fs, error)

type (
	FromInput  pipe.FromInput[afero.Fs]
	FromFs     pipe.FromFs[afero.Fs]
	FromReader pipe.FromReader[afero.Fs]
	ToWriter   pipe.ToWriter[*tdlv1alpha1.Spec]
)

func (f Func) Execute(ctx context.Context, spec *tdlv1alpha1.Spec) (afero.Fs, error) {
	return f(ctx, spec)
}

func (f Func) String() string {
	return "anonymous"
}

func Lift[G constraint.Gen](fn G) tdl.Generator {
	return Func(fn)
}

func New(gen Func) tdl.Generator {
	return Lift(gen)
}
