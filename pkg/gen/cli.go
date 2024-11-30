package gen

import (
	"context"
	"os/exec"

	"github.com/unmango/go/option"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	"github.com/unstoppablemango/tdl/pkg/sink"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Cli struct {
	name string
	args []string
	enc  tdl.MediaType
}

type CliOption func(*Cli)

// Execute implements tdl.Generator.
func (c *Cli) Execute(ctx context.Context, s *tdlv1alpha1.Spec, si tdl.Sink) error {
	var args []string
	if len(c.args) > 0 {
		args = append(args, c.args...)
	} else {
		args = []string{"generate"}
	}

	cmd := exec.CommandContext(ctx, c.name, args...)

	cmd.Stdin = mediatype.ProtoReader(s, c.enc)
	cmd.Stdout = sink.NewWriter(si)

	return cmd.Run()
}

func (c *Cli) String() string {
	return c.name
}

var _ tdl.SinkGenerator = &Cli{}

func NewCli(name string, options ...CliOption) *Cli {
	gen := &Cli{name: name, enc: mediatype.ApplicationProtobuf}
	option.ApplyAll(gen, options)

	return gen
}

func WithCliArgs(args ...string) CliOption {
	return func(c *Cli) {
		c.args = args
	}
}

func WithCliEncoding(media tdl.MediaType) CliOption {
	return func(c *Cli) {
		c.enc = media
	}
}
