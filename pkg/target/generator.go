package target

import (
	"context"
	"fmt"

	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

func Generator(ctx context.Context, target tdl.Target, available iter.Seq[tdl.Plugin]) (tdl.Generator, error) {
	plugin, err := target.Choose(available)
	if err != nil {
		return nil, fmt.Errorf("choosing plugin: %w", err)
	}

	generator, ok := plugin.(tdl.GeneratorPlugin)
	if !ok {
		return nil, fmt.Errorf("not a generator: %s", plugin)
	}

	return generator.Generator(ctx, target.Meta())
}
