package runner

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	tdl "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
)

type Docker struct {
	name   string
	plugin string
}

type DockerOption func(*Docker) error

var (
	WithUml2Ts  = pluginOption("uml2ts")
	WithUml2Go  = pluginOption("uml2go")
	WithUml2Pcl = pluginOption("uml2pcl")
)

func WithName(name string) DockerOption {
	return func(d *Docker) error {
		d.name = name
		return nil
	}
}

func WithTarget(target string) DockerOption {
	return func(d *Docker) error {
		p, err := plugin.LookupPath(target)
		d.plugin = p
		return err
	}
}

func FromGen(options uml.GeneratorOptions) DockerOption {
	return uml.Flat(WithTarget(options.Target))
}

func pluginOption(plugin string) DockerOption {
	return func(d *Docker) error {
		d.plugin = plugin
		return nil
	}
}

func NewDocker(opts ...DockerOption) Docker {
	docker := Docker{}
	for _, opt := range opts {
		if err := opt(&docker); err != nil {
			panic(err)
		}
	}
	return docker
}

var (
	genCmd = []string{"gen"}
)

// Gen implements uml.Generator.
func (d *Docker) Gen(ctx context.Context, spec *tdl.Spec, writer io.Writer) error {
	return d.run(ctx, genCmd, spec, writer)
}

// From implements uml.Converter.
func (d *Docker) From(ctx context.Context, reader io.Reader) (*tdl.Spec, error) {
	panic("unimplemented")
}

var _ uml.Runner = &Docker{}

func (d *Docker) run(ctx context.Context, cmd []string, spec *tdl.Spec, writer io.Writer) error {
	docker, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return err
	}
	defer docker.Close()

	img := fmt.Sprintf("ghcr.io/unstoppablemango/%s:%s", d.plugin, "main")
	pull, err := docker.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		return err
	}
	defer pull.Close()

	_, err = io.Copy(os.Stdout, pull)
	if err != nil {
		return err
	}

	ctr, err := docker.ContainerCreate(ctx,
		&container.Config{
			Image:        img,
			Tty:          false,
			OpenStdin:    true,
			StdinOnce:    true,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Cmd:          cmd,
		},
		nil, nil, nil,
		d.name,
	)
	if err != nil {
		return err
	}

	stream, err := docker.ContainerAttach(ctx, ctr.ID, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Logs:   true,
	})
	if err != nil {
		return err
	}
	defer stream.Close()

	if err = docker.ContainerStart(ctx, ctr.ID, container.StartOptions{}); err != nil {
		return err
	}

	dispose := func() {
		// TODO: Handle the errors better
		err := docker.ContainerStop(ctx, ctr.ID, container.StopOptions{})
		if err != nil {
			return
		}

		err = docker.ContainerRemove(ctx, ctr.ID, container.RemoveOptions{})
		if err != nil {
			return
		}
	}
	defer dispose()

	bytes, err := proto.Marshal(spec)
	if err != nil {
		return err
	}

	_, err = stream.Conn.Write(bytes)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, stream.Reader)
	return err
}
