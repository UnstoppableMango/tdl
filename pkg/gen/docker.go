package gen

import (
	"context"
	"io"
	"slices"

	"github.com/charmbracelet/log"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Docker struct {
	options []client.Opt
	image   string
}

// Execute implements tdl.Generator.
func (d *Docker) Execute(
	ctx context.Context,
	spec *tdlv1alpha1.Spec,
) (afero.Fs, error) {
	client, err := client.NewClientWithOpts(d.options...)
	if err != nil {
		return nil, err
	}

	if err = d.ensure(ctx, client); err != nil {
		return nil, err
	}

	log.Debug("creating container")
	res, err := client.ContainerCreate(ctx,
		&container.Config{
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Image:        d.image,
			Labels: map[string]string{
				"owner": "ux",
			},
		},
		&container.HostConfig{
			AutoRemove: true,
		},
		&network.NetworkingConfig{},
		&v1.Platform{},
		"",
	)
	if err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
	}
	defer ctr.Close()

	log.Debug("starting container")
	err = client.ContainerStart(ctx,
		res.ID,
		container.StartOptions{},
	)
	if err != nil {
		return nil, err
	}

	log.Debug("marshaling spec")
	data, err := mediatype.Marshal(spec, mediatype.ApplicationProtobuf)
	if err != nil {
		return nil, err
	}

	log.Debug("writing spec to container")
	if _, err = ctr.Conn.Write(data); err != nil {
		return nil, err
	}

	log.Debug("reading generator response")
	fs := afero.NewMemMapFs()
	err = afero.WriteReader(fs, "out", ctr.Reader)
	if err != nil {
		return nil, err
	}

	return fs, nil
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
			log.Debug("image exists")
			return nil
		}
	}

	log.Debug("pulling image")
	r, err := client.ImagePull(ctx, d.image, image.PullOptions{})
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

func (d *Docker) String() string {
	return d.image
}

var _ tdl.Generator = &Docker{}

func NewDocker(image string, options ...client.Opt) *Docker {
	return &Docker{
		options: options,
		image:   image,
	}
}
