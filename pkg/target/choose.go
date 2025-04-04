package target

import (
	"context"
	"fmt"

	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

type RejectionErr struct {
	Generator tdl.Generator
	Reason    string
}

func (e RejectionErr) Error() string {
	return fmt.Sprintf("rejected %s: %s", e.Generator, e.Reason)
}

func Reject(generator tdl.Generator, reason string) error {
	return &RejectionErr{generator, reason}
}

func Choose[T tdl.Plugin](target tdl.Target, available iter.Seq[tdl.Plugin]) (res T, err error) {
	plugin, err := target.Choose(available)
	if err != nil {
		return res, err
	}

	var ok bool
	if res, ok = plugin.(T); !ok {
		return res, fmt.Errorf("invalid type for: %s", plugin)
	}

	return
}

func Generator(
	ctx context.Context,
	target tdl.Target,
	available iter.Seq[tdl.Plugin],
) (tdl.Generator, error) {
	if p, err := target.Choose(available); err != nil {
		return nil, fmt.Errorf("choosing plugin: %w", err)
	} else {
		return plugin.Generator(ctx, p, target)
	}
}

func Tool(
	ctx context.Context,
	target tdl.Target,
	available iter.Seq[tdl.Plugin],
) (tdl.Tool, error) {
	if p, err := target.Choose(available); err != nil {
		return nil, fmt.Errorf("choosing plugin: %w", err)
	} else if err = p.Prepare(ctx); err != nil {
		return nil, fmt.Errorf("preparing plugin: %w", err)
	} else {
		return plugin.Tool(ctx, p, target)
	}
}
