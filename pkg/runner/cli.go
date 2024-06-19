package runner

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
)

type cli struct {
	Path string
	Args []string
}

type CliOpt = uml.Opt[cli]

func NewCli[T CliOpt](path string, opts ...T) (uml.Runner, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	return uml.Apply(cli{Path: path}, opts...)
}

func WithArgs[T CliOpt](arg ...string) T {
	return func(opts *cli) error {
		opts.Args = append(opts.Args, arg...)
		return nil
	}
}

// From implements uml.Runner.
func (c *cli) From(ctx context.Context, reader io.Reader) (*uml.Spec, error) {
	args := append([]string{"from"}, c.Args...)
	cmd := exec.Command(c.Path, args...)

	cmd.Stdin = reader
	buf := &bytes.Buffer{}
	cmd.Stdout = buf

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	spec := &uml.Spec{}
	err := proto.Unmarshal(buf.Bytes(), spec)

	return spec, err
}

// Gen implements uml.Runner.
func (c *cli) Gen(ctx context.Context, spec *uml.Spec, writer io.Writer) error {
	inData, err := proto.Marshal(spec)
	if err != nil {
		return err
	}

	args := append([]string{"gen"}, c.Args...)
	cmd := exec.Command(c.Path, args...)

	cmd.Stdin = bytes.NewReader(inData)
	cmd.Stdout = writer

	return cmd.Run()
}

var _ uml.Runner = &cli{}
