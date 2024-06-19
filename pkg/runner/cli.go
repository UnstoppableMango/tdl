package runner

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"

	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
)

type cli struct {
	Path string
	Args []string
	log  *slog.Logger
}

type CliOpt func(*cli) error

func NewCli(path string, opts ...CliOpt) (uml.Runner, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	return uml.Apply(cli{
		Path: path,
		Args: []string{},
		log:  slog.Default(),
	}, opts...)
}

func WithArgs(arg ...string) CliOpt {
	return func(c *cli) error {
		c.Args = append(c.Args, arg...)
		return nil
	}
}

func WithLogger(log *slog.Logger) CliOpt {
	return func(c *cli) error {
		c.log = log
		return nil
	}
}

// From implements uml.Runner.
func (c *cli) From(ctx context.Context, reader io.Reader) (*uml.Spec, error) {
	args := append([]string{"from"}, c.Args...)
	cmd := exec.Command(c.Path, args...)
	c.log.Debug("built command", "path", c.Path, "args", args)

	buf := &bytes.Buffer{}
	cmd.Stdin = reader
	cmd.Stdout = buf

	c.log.Info("executing command")
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	c.log.Debug("unmarshalling proto response")
	spec := &uml.Spec{}
	err := proto.Unmarshal(buf.Bytes(), spec)

	return spec, err
}

// Gen implements uml.Runner.
func (c *cli) Gen(ctx context.Context, spec *uml.Spec, writer io.Writer) error {
	c.log.Debug("marshalling input proto")
	inData, err := proto.Marshal(spec)
	if err != nil {
		return err
	}

	args := append([]string{"gen"}, c.Args...)
	cmd := exec.Command(c.Path, args...)
	c.log.Debug("built command", "path", c.Path, "args", args)

	cmd.Stdin = bytes.NewReader(inData)
	cmd.Stdout = writer

	c.log.Info("executing command")
	return cmd.Run()
}

var _ uml.Runner = &cli{}
