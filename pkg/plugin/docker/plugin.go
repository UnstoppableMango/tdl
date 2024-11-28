package docker

import (
	"context"

	"github.com/docker/docker/client"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen"
)

type plugin struct {
	client client.ContainerAPIClient
	image  string
}

// SinkGenerator implements tdl.Plugin.
func (p *plugin) SinkGenerator(tdl.Target) (tdl.SinkGenerator, error) {
	panic("unimplemented")
}

// Generator implements tdl.Plugin.
func (p *plugin) Generator(context.Context, tdl.Target) (tdl.Generator, error) {
	return gen.NewDocker(p.image), nil
}

// String implements tdl.Plugin.
func (p *plugin) String() string {
	return p.image
}

func New(client client.ContainerAPIClient, image string) tdl.Plugin {
	return &plugin{client, image}
}
