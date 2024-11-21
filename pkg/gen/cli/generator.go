package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/unmango/go/option"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type cli struct {
	name string
	args []string
	enc  tdl.MediaType
}

type Option func(*cli)

// Execute implements tdl.Generator.
func (c cli) Execute(ctx context.Context, spec *tdlv1alpha1.Spec) (afero.Fs, error) {
	log.Debug("creating temp directory")
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("creating exec context: %w", err)
	}

	stderr := &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, c.name, c.args...)
	cmd.Stdin = mediatype.NewReader(spec, c.enc)
	cmd.Stderr = stderr
	cmd.Dir = tmp

	log.Debug("executing command")
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("executing generator: %w: %s", err, stderr)
	}
	if stderr.Len() > 0 {
		return nil, fmt.Errorf("executing generator: %s", stderr)
	}

	log.Debugf("returning a new BasePathFs at %s", tmp)
	return afero.NewBasePathFs(afero.NewOsFs(), tmp), nil
}

// String implements fmt.Stringer
func (c cli) String() string {
	return c.name
}

func New(name string, options ...Option) tdl.Generator {
	gen := cli{
		name: name,
		enc:  mediatype.ApplicationProtobuf,
	}
	option.ApplyAll(&gen, options)

	return gen
}

func WithArgs(args ...string) Option {
	return func(c *cli) {
		c.args = args
	}
}

func WithEncoding(media tdl.MediaType) Option {
	return func(c *cli) {
		c.enc = media
	}
}
