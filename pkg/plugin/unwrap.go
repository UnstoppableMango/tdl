package plugin

import tdl "github.com/unstoppablemango/tdl/pkg"

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
	if depth >= UnwrapDepth {
		return []tdl.Plugin{}
	}

	plugins, ok := plugin.(Aggregate)
	if !ok {
		return []tdl.Plugin{plugin}
	}

	acc := []tdl.Plugin{}
	for _, p := range plugins {
		acc = append(acc, unwrapRec(p, depth+1)...)
	}

	return acc
}
