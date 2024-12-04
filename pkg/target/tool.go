package target

import (
	"context"
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
		func(p tdl.ToolPlugin) bool {
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

func Tool(ctx context.Context, target tdl.Target, available iter.Seq[tdl.Plugin]) (tdl.Tool, error) {
	plugin, err := target.Choose(available)
	if err != nil {
		return nil, fmt.Errorf("choosing plugin: %w", err)
	}

	tool, ok := plugin.(tdl.ToolPlugin)
	if !ok {
		return nil, fmt.Errorf("not a tool: %s", plugin)
	}

	return tool.Tool(ctx, target.Meta())
}
