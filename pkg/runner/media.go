package runner

import (
	"context"
	"io"

	"github.com/unstoppablemango/tdl/pkg/uml"
)

type mediaType struct {
	MediaType string
	Runner    uml.Runner
}

func WithMediaType(runner uml.Runner, media string) uml.Runner {
	return &mediaType{
		MediaType: media,
		Runner:    runner,
	}
}

// From implements uml.Runner.
func (m *mediaType) From(ctx context.Context, reader io.Reader) (*uml.Spec, error) {
	return m.Runner.From(ctx, reader)
}

// Gen implements uml.Runner.
func (m *mediaType) Gen(ctx context.Context, spec *uml.Spec, writer io.Writer) error {
	panic("unimplemented")
}

var _ uml.Runner = &mediaType{}
