package runner

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/gogo/protobuf/proto"
	tdl "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

type cli struct {
	Path string
}

// From implements uml.Runner.
func (c *cli) From(ctx context.Context, reader io.Reader) (*tdl.Spec, error) {
	inData, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(c.Path)
	stdin, stdout, err := pipes(cmd)
	if err != nil {
		return nil, err
	}

	if _, err = stdin.Write(inData); err != nil {
		return nil, err
	}

	if err = cmd.Run(); err != nil {
		return nil, err
	}

	outData, err := io.ReadAll(stdout)
	if err != nil {
		return nil, err
	}

	spec := &tdl.Spec{}
	if err = proto.Unmarshal(outData, spec); err != nil {
		return nil, err
	}

	return spec, nil
}

// Gen implements uml.Runner.
func (c *cli) Gen(ctx context.Context, spec *tdl.Spec, writer io.Writer) error {
	inData, err := proto.Marshal(spec)
	if err != nil {
		return err
	}

	cmd := exec.Command(c.Path)
	stdin, stdout, err := pipes(cmd)
	if err != nil {
		return err
	}

	if _, err = stdin.Write(inData); err != nil {
		return err
	}

	if err = cmd.Run(); err != nil {
		return err
	}

	outData, err := io.ReadAll(stdout)
	if err != nil {
		return err
	}

	if _, err = writer.Write(outData); err != nil {
		return err
	}

	return nil
}

var _ uml.Runner = &cli{}

func NewCli(path string) (uml.Runner, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	return &cli{Path: path}, nil
}

func pipes(cmd *exec.Cmd) (io.WriteCloser, io.ReadCloser, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	return stdin, stdout, nil
}
