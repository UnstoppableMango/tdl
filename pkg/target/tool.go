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
	plugin, ok := plugin.Find(available,
		func(p tdl.Plugin) bool {
			return meta.Supports(p.Meta(), t.Meta())
		},
	)
	if !ok {
		return nil, fmt.Errorf("no match for target: %s", t.name)
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
