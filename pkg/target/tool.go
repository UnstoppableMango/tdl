package target

import (
	"fmt"

	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/meta"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

type tool struct {
	name string
}

// Meta implements tdl.Target.
func (t tool) Meta() tdl.Meta {
	return meta.Map{
		"name": t.name,
	}
}

// Tool implements tdl.Target.
func (t tool) Choose(available iter.Seq[tdl.Plugin]) (tdl.Plugin, error) {
	plugin, ok := plugin.Find(plugin.FilterSupported(available, t),
		func(p tdl.Plugin) bool {
			return p.String() == t.name
		},
	)
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", t.name)
	}

	return plugin, nil
}

// String implements tdl.Target.
func (t tool) String() string {
	return t.name
}

func NewTool(name string) tdl.Target {
	return &tool{name}
}
