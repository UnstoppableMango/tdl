package plugin

import (
	"fmt"
	"iter"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

const UnwrapDepth = 3

type Aggregate []tdl.Plugin

// Meta implements tdl.Plugin.
func (a Aggregate) Meta() tdl.Meta {
	panic("unimplemented")
}

// String implements tdl.Plugin.
func (a Aggregate) String() string {
	return fmt.Sprintf("%v", []tdl.Plugin(a))
}

func (a Aggregate) sorted() []Ordered {
	return Sorted(a.ordered())
}

func (a Aggregate) ordered() iter.Seq[Ordered] {
	return func(yield func(Ordered) bool) {
		for _, p := range a {
			if !yield(AsOrdered(p)) {
				return
			}
		}
	}
}

func NewAggregate(plugins ...tdl.Plugin) Aggregate {
	return Aggregate(plugins)
}

var _ tdl.Plugin = Aggregate{}
