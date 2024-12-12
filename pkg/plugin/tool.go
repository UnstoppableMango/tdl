package plugin

import (
	"context"
	"errors"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

func Tool(
	ctx context.Context,
	plugin tdl.Plugin,
	target tdl.Target,
) (tdl.Tool, error) {
	return nil, errors.New("TODO: tool")
}
