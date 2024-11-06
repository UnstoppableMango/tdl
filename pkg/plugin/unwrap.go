package plugin

import (
	"github.com/charmbracelet/log"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

func Unwrap(plugin tdl.Plugin) []tdl.Plugin {
	return unwrapRec(plugin, 0)
}

func UnwrapAll(plugins []tdl.Plugin) []tdl.Plugin {
	res := []tdl.Plugin{}
	for _, p := range plugins {
		res = append(res, Unwrap(p)...)
	}

	return res
}

// This is probably really inefficient, but we'll get there eventually
// TODO: Ordering?

func unwrapRec(plugin tdl.Plugin, depth int) []tdl.Plugin {
	log := log.With("plugin", plugin, "depth", depth)

	plugins, isAggregate := plugin.(Aggregate)
	if !isAggregate {
		log.Debug("not an aggregate")
		return []tdl.Plugin{plugin}
	}

	if depth >= UnwrapDepth {
		log.Debug("at depth")
		return plugins
	}

	acc := []tdl.Plugin{}
	for _, p := range plugins {
		log.Debug("unwrapping", "inner", p)
		acc = append(acc, unwrapRec(p, depth+1)...)
	}

	log.Debug("returning accumulated plugins", "accumulator", acc)
	return acc
}
