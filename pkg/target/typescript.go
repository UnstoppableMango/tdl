package target

import (
	"errors"
	"iter"
	"slices"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

type typescript string

var TypeScript typescript = "TypeScript"

// Choose implements tdl.Target.
func (t typescript) Choose(available []tdl.Generator) (tdl.Generator, error) {
	if len(available) == 0 {
		return nil, errors.New("no generators to choose from")
	}

	for _, g := range available {
		if supported(g) {
			return g, nil
		}
	}

	return nil, errors.New("no supported generators")
}

// Plugins implements tdl.Target.
func (t typescript) Plugins() iter.Seq[tdl.Plugin] {
	return slices.Values([]tdl.Plugin{
		plugin.Uml2Ts,
	})
}

// String implements tdl.Target.
func (t typescript) String() string {
	return string(t)
}

func supported(generator tdl.Generator) bool {
	cli, ok := generator.(*gen.Cli)
	if !ok {
		return false
	}

	return cli.String() == "uml2ts"
}
