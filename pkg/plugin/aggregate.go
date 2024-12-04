package plugin

import (
	"fmt"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

const UnwrapDepth = 3

type Aggregate []tdl.Plugin

// Meta implements tdl.Plugin.
func (a Aggregate) Meta() tdl.Meta {
	panic("unimplemented")
}

// Supports implements tdl.Plugin
func (a Aggregate) Supports(target tdl.Target) bool {
	for _, p := range a {
		if p.Supports(target) {
			return true
		}
	}

	return false
}

// String implements tdl.Plugin.
func (a Aggregate) String() string {
	return fmt.Sprintf("%v", []tdl.Plugin(a))
}

func NewAggregate(plugins ...tdl.Plugin) Aggregate {
	return Aggregate(plugins)
}

var _ tdl.Plugin = Aggregate{}
