package plugin

import (
	"context"
	"errors"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

func Generator(
	ctx context.Context,
	plugin tdl.Plugin,
	target tdl.Target,
) (tdl.Generator, error) {
	return nil, errors.New("TODO: generator")
}
