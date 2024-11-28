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
)

type plugin struct {
	client client.APIClient
	image  string
}

// SinkGenerator implements tdl.Plugin.
func (p *plugin) SinkGenerator(tdl.Target) (tdl.SinkGenerator, error) {
	panic("unimplemented")
}

// Generator implements tdl.Plugin.
func (p *plugin) Generator(ctx context.Context, t tdl.Target) (tdl.Generator, error) {
	if err := p.ensure(ctx); err != nil {
		return nil, fmt.Errorf("pulling image: %w", err)
	}

	return docker.New(p.client, p.image), nil
}

// String implements tdl.Plugin.
func (p *plugin) String() string {
	return p.image
}

func (p *plugin) ensure(ctx context.Context) error {
	log.Debug("listing images")
	images, err := p.client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return err
	}

	log.Debug("searching for existing image",
		"images", len(images),
		"image", p.image,
	)
	for _, i := range images {
		if slices.Contains(i.RepoTags, p.image) {
			log.Debug("image exists")
			return nil
		}
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

func New(client client.APIClient, image string) tdl.Plugin {
	return &plugin{client, image}
}
