package plugin

import (
	tdl "github.com/unstoppablemango/tdl/pkg"
)

// This is probably really inefficient, but we'll get there eventually
// TODO: Ordering?

func Unwrap(plugin tdl.Plugin) []tdl.Plugin {
	return unwrapRec(plugin, 0)
}

func UnwrapAll(plugins []tdl.Plugin) []tdl.Plugin {
	return unwrapRec(NewAggregate(plugins...), -1)
}

func unwrapRec(plugin tdl.Plugin, depth int) []tdl.Plugin {
	plugins, isAggregate := plugin.(Aggregate)
	if !isAggregate {
		return []tdl.Plugin{plugin}
	}
	if depth >= UnwrapDepth {
		return plugins
	}

	acc := []tdl.Plugin{}
	for _, p := range plugins {
		acc = append(acc, unwrapRec(p, depth+1)...)
	}

	return acc
}
