package docker

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/charmbracelet/log"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen/docker"
	"github.com/unstoppablemango/tdl/pkg/meta"
)

type plugin struct {
	client client.APIClient
	image  string
}

// Meta implements tdl.GeneratorPlugin.
func (p *plugin) Meta() tdl.Meta {
	return meta.Map{
		"image": p.image,
	}
}

// Generator implements tdl.Plugin.
func (p *plugin) Generator(ctx context.Context, t tdl.Meta) (tdl.Generator, error) {
	if err := p.ensure(ctx); err != nil {
		return nil, fmt.Errorf("pulling image: %w", err)
	}

	return docker.New(p.client, p.image), nil
}

// Supports implements tdl.Target.
func (g *plugin) Supports(target tdl.Target) bool {
	return target.String() == "TypeScript" // TODO
}

// String implements tdl.Plugin.
func (p *plugin) String() string {
	return p.image
}

func (p *plugin) ensure(ctx context.Context) error {
	exists, err := ImageExists(ctx, p.client, p.image)
	if err != nil {
		return err
	}
	if exists {
		log.Debug("image exists")
		return nil
	}

	log.Debug("pulling image")
	r, err := p.client.ImagePull(ctx, p.image, image.PullOptions{})
	if err != nil {
		return err
	}
	defer r.Close()

	log.Debug("reading response")
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	// TODO: This contains a JSON stream
	// we could use for reporting progress
	log.Debug(string(data))
	return nil
}

func New(client client.APIClient, image string) tdl.GeneratorPlugin {
	return &plugin{client, image}
}

func ImageExists(ctx context.Context, c client.APIClient, name string) (bool, error) {
	log.Debug("listing images")
	images, err := c.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return false, err
	}

	log.Debug("searching for existing image", "images", len(images), "name", name)
	for _, i := range images {
		if slices.Contains(i.RepoTags, name) {
			return true, nil
		}
	}

	return false, nil
}
