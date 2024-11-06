package plugin

import (
	"iter"
	"math"
	"slices"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

type Ordered interface {
	tdl.Plugin
	Order() int
}

type ordered struct {
	tdl.Plugin
	order int
}

// Order implements Ordered.
func (p *ordered) Order() int {
	return p.order
}

func Sorted[O Ordered](seq iter.Seq[O]) []O {
	return slices.SortedFunc(seq, compare)
}

func AsOrdered(plugin tdl.Plugin) Ordered {
	if o, ok := plugin.(Ordered); ok {
		return o
	}

	return WithPriority(plugin, math.MaxInt)
}

func WithPriority(plugin tdl.Plugin, priority int) Ordered {
	if o, ok := plugin.(*ordered); ok {
		// TODO: Don't mutate
		o.order = priority
		return o
	} else {
		return &ordered{plugin, priority}
	}
}

func compare[O Ordered](a O, b O) int {
	return a.Order() - b.Order()
}
