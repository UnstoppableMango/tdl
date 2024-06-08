package cmd

import (
	"context"
	"io"

	cli "github.com/unstoppablemango/tdl/cli/internal"
	tdl "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1"
	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
)

func init() {
	rootCmd.AddCommand(fromCmd)
}

type echoConverter struct {}

// From implements uml.Converter.
func (e echoConverter) From(ctx context.Context, reader io.Reader) (*tdl.Spec, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	spec := tdl.Spec{}
	err = proto.Unmarshal(data, &spec)
	if err != nil {
		return nil, err
	}

	return &spec, nil
}

var fromCmd = cli.NewFromCmd(func(_ uml.ConverterOptions) uml.Converter {
	return echoConverter{}
})
