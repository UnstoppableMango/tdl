package target

import (
	"iter"
	"slices"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

type typescript string

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

var TypeScript typescript = "TypeScript"
