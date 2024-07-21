package runner

import (
	"context"
	"io"

	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
)

var (
	echoGenerator = gen.New(func(ctx context.Context, spec *uml.Spec, writer io.Writer) error {
		data, err := proto.Marshal(spec)
		if err != nil {
			return err
		}

		_, err = writer.Write(data)
		return err
	})
)

type echoRunner struct{}

func NewEcho() uml.Runner {
	return &echoRunner{}
}

// From implements uml.Runner.
func (e echoRunner) From(ctx context.Context, reader io.Reader) (*uml.Spec, error) {
	panic("unimplemented")
}

// Gen implements uml.Runner.
func (e echoRunner) Gen(ctx context.Context, spec *uml.Spec, writer io.Writer) error {
	return echoGenerator.Gen(ctx, spec, writer)
}

var _ uml.Runner = echoRunner{}
