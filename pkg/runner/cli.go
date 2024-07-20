package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

type (
	CliOpt func(*cli) error
	CliErr struct {
		Stdout string
		Stderr string
	}
)

// Error implements error.
func (c *CliErr) Error() string {
	// Kinda hate all of this
	stdout := fmt.Sprintf("stdout: %s\n", c.Stdout)
	stderr := fmt.Sprintf("stderr: %s\n", c.Stderr)
	return fmt.Sprintf("%s, %s", stdout, stderr)
}

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

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdin = reader
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	c.log.Info("executing command")
	if err := cmd.Run(); err != nil {
		return nil, errors.Join(err, &CliErr{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		})
	}

	c.log.Debug("unmarshalling proto response")
	spec := &uml.Spec{}
	err := proto.Unmarshal(stdout.Bytes(), spec)

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

	stderr := &bytes.Buffer{}
	cmd.Stdin = bytes.NewReader(inData)
	cmd.Stdout = writer
	cmd.Stderr = stderr

	c.log.Info("executing command")
	if err := cmd.Run(); err != nil {
		return errors.Join(err, &CliErr{
			Stderr: stderr.String(),
		})
	}

	return nil
}

var _ uml.Runner = &cli{}
var _ error = &CliErr{}
