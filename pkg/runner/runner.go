package runner

import (
	"github.com/unstoppablemango/tdl/pkg/from"
	"github.com/unstoppablemango/tdl/pkg/gen"
)

type Runner[Spec, I, O any] interface {
	gen.Generator[Spec, I]
	from.Converter[O, Spec]
}
