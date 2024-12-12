package plugin

import (
	"context"
	"fmt"

	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/tool/crd2pulumi"
)

// The goal is to remove these two interfaces, but that will
// take a decent bit more refactoring to make happen

type GeneratorPlugin interface {
	tdl.Plugin
	Generator(context.Context, tdl.Meta) (tdl.Generator, error)
}

type ToolPlugin interface {
	tdl.Plugin
	Tool(context.Context, tdl.Meta) (tdl.Tool, error)
}

type Predicate[T tdl.Plugin] func(T) bool

func Find[T tdl.Plugin](plugins iter.Seq[tdl.Plugin], pred Predicate[T]) (res T, found bool) {
	for plugin := range plugins {
		for _, nested := range Unwrap(plugin) {
			if n, ok := nested.(T); ok && pred(n) {
				return n, true
			}
		}
	}

	return res, false
}

func OfType[T tdl.Plugin](seq iter.Seq[tdl.Plugin]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for p := range UnwrapEach(seq) {
			if t, ok := p.(T); ok && !yield(t) {
				break
			}
		}
	}
}

func FilterSupported(seq iter.Seq[tdl.Plugin], target tdl.Target) iter.Seq[tdl.Plugin] {
	return iter.Filter(seq, func(plugin tdl.Plugin) bool {
		return plugin.Supports(target)
	})
}

func Generator(
	ctx context.Context,
	plugin tdl.Plugin,
	target tdl.Target,
) (tdl.Generator, error) {
	if g, ok := plugin.(GeneratorPlugin); ok {
		return g.Generator(ctx, target.Meta())
	}

	return nil, fmt.Errorf("no generator for plugin: %s", plugin)
}

func Tool(
	ctx context.Context,
	plugin tdl.Plugin,
	target tdl.Target,
) (tdl.Tool, error) {
	if plugin.String() == "crd2pulumi" {
		return &crd2pulumi.Tool{}, nil // TODO
	}
	if t, ok := plugin.(ToolPlugin); ok {
		return t.Tool(ctx, target.Meta())
	}

	return nil, fmt.Errorf("no tool for plugin: %s", plugin)
}
