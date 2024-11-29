package docker

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type docker struct {
	client client.APIClient
	image  string
}

// Execute implements tdl.Generator.
func (d *docker) Execute(
	ctx context.Context,
	spec *tdlv1alpha1.Spec,
) (afero.Fs, error) {
	log.Debug("creating container")
	res, err := d.client.ContainerCreate(ctx,
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
	ctr, err := d.client.ContainerAttach(ctx,
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
	err = d.client.ContainerStart(ctx,
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
	err = afero.WriteReader(fs, "stdout", ctr.Reader)
	if err != nil {
		return nil, err
	}

	return fs, nil
}

func (d *docker) String() string {
	return d.image
}

var _ tdl.Generator = &docker{}

func New(client client.APIClient, image string) tdl.Generator {
	return &docker{
		client: client,
		image:  image,
	}
}
