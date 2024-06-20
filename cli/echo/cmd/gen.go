package cmd

import (
	"context"
	"io"

	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
)

func init() {
	rootCmd.AddCommand(genCmd)
}

type echoGenerator struct{}

// Gen implements uml.Generator.
func (e echoGenerator) Gen(ctx context.Context, spec *uml.Spec, writer io.Writer) error {
	data, err := proto.Marshal(spec)
	if err != nil {
		return err
	}

	_, err = writer.Write(data)
	return err
}

var genCmd = cli.NewGenCmd(
	func(ctx context.Context, _ cli.GenCmdOptions, args []string) (uml.Generator, error) {
		return echoGenerator{}, nil
	},
)
