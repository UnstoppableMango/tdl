package gen

import (
	"context"
	"fmt"
	"slices"

	"github.com/charmbracelet/log"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Docker struct {
	options []client.Opt
	image   string
}

// Execute implements tdl.Generator.
func (d *Docker) Execute(spec *tdlv1alpha1.Spec, sink tdl.Sink) error {
	client, err := client.NewClientWithOpts(d.options...)
	if err != nil {
		return err
	}

	ctx := context.TODO()
	if err = d.ensure(ctx, client); err != nil {
		return err
	}

	log.Debug("creating container")
	res, err := client.ContainerCreate(ctx,
		&container.Config{
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
		},
		&container.HostConfig{
			AutoRemove: true,
		},
		&network.NetworkingConfig{},
		&v1.Platform{},
		fmt.Sprintf("tdl-gen-%s", "TODO"),
	)
	if err != nil {
		return err
	}
	for _, w := range res.Warnings {
		log.Warn(w)
	}

	log.Debug("attaching to container")
	ctr, err := client.ContainerAttach(ctx,
		res.ID,
		container.AttachOptions{
			Stdin:  true,
			Stdout: true,
			Stderr: true,
		},
	)
	defer ctr.Close()
	if err != nil {
		return err
	}

	log.Debug("starting container")
	err = client.ContainerStart(ctx,
		res.ID,
		container.StartOptions{},
	)
	if err != nil {
		return err
	}

	log.Debug("marshaling spec")
	data, err := mediatype.Marshal(spec, mediatype.ApplicationProtobuf)
	if err != nil {
		return err
	}

	log.Debug("writing spec to container")
	if _, err = ctr.Conn.Write(data); err != nil {
		return err
	}

	log.Debug("reading generator response")
	return sink.WriteUnit("TODO", ctr.Conn)
}

func (d *Docker) ensure(ctx context.Context, client *client.Client) error {
	log.Debug("listing images")
	images, err := client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return err
	}

	log.Debug("searching for existing image",
		"images", len(images),
		"image", d.image,
	)
	for _, i := range images {
		if slices.Contains(i.RepoTags, d.image) {
			return nil
		}
	}

	log.Debug("pulling image")
	r, err := client.ImagePull(ctx, d.image, image.PullOptions{})
	if err != nil {
		return err
	}

	defer r.Close()
	return nil
}

var _ tdl.Generator = &Docker{}

func NewDocker(image string, options ...client.Opt) *Docker {
	return &Docker{
		options: options,
		image:   image,
	}
}
