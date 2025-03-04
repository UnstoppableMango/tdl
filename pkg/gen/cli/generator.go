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

type generator struct {
	name   string
	args   []string
	enc    tdl.MediaType
	stdout bool
}

type Option func(*generator)

// Execute implements tdl.Generator.
func (c generator) Execute(ctx context.Context, spec *tdlv1alpha1.Spec) (afero.Fs, error) {
	log.Debug("creating temp directory")
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("creating exec context: %w", err)
	}

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, c.name, c.args...)
	cmd.Stdin = mediatype.ProtoReader(spec, c.enc)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = tmp

	log.Debug("executing command")
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("executing generator: %w: %s", err, stderr)
	}
	if stderr.Len() > 0 {
		return nil, fmt.Errorf("executing generator: %s", stderr)
	}

	if c.stdout {
		log.Debug("reading stdout")
		fs := afero.NewMemMapFs()
		err = afero.WriteFile(fs, "stdout", stdout.Bytes(), os.ModePerm)
		if err != nil {
			return nil, err
		}

		return fs, nil
	} else {
		log.Debugf("returning a new BasePathFs at %s", tmp)
		return afero.NewBasePathFs(afero.NewOsFs(), tmp), nil
	}
}

// String implements fmt.Stringer
func (c generator) String() string {
	return c.name
}

func New(name string, options ...Option) tdl.Generator {
	gen := generator{
		name: name,
		enc:  mediatype.ApplicationProtobuf,
	}
	option.ApplyAll(&gen, options)

	return gen
}

func WithArgs(args ...string) Option {
	return func(c *generator) {
		c.args = args
	}
}

func WithEncoding(media tdl.MediaType) Option {
	return func(c *generator) {
		c.enc = media
	}
}

func ExpectStdout(cli *generator) {
	cli.stdout = true
}

func WithExpectStdout(stdout bool) Option {
	return func(c *generator) {
		c.stdout = stdout
	}
}
