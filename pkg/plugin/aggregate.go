package plugin

import (
	"errors"
	"fmt"
	"iter"
	"slices"

	"github.com/charmbracelet/log"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

const UnwrapDepth = 3

type Ordered interface {
	tdl.Plugin
	Order() int
}

type Aggregate []tdl.Plugin

// Generator implements tdl.Plugin.
func (a Aggregate) Generator(t tdl.Target) (tdl.Generator, error) {
	if len(a) == 0 {
		return nil, errors.New("empty aggregate plugin")
	}

	errs := []error{}
	for _, p := range a.sorted() {
		g, err := p.Generator(t)
		if err == nil {
			return g, nil
		}

		log.Error(err, "generator", g)
		errs = append(errs, err)
	}

	return nil, errors.Join(errs...)
}

// String implements tdl.Plugin.
func (a Aggregate) String() string {
	return fmt.Sprintf("%#v", []tdl.Plugin(a))
}

func (a Aggregate) sorted() []Ordered {
	return Sorted(a.ordered())
}

func (a Aggregate) ordered() iter.Seq[Ordered] {
	return func(yield func(Ordered) bool) {
		for _, p := range a {
			if o, ok := p.(Ordered); ok {
				if !yield(o) {
					return
				}
			}
		}
	}
}

func NewAggregate(plugins ...tdl.Plugin) Aggregate {
	return Aggregate(plugins)
}

var _ tdl.Plugin = Aggregate{}

func Sorted[O Ordered](seq iter.Seq[O]) []O {
	return slices.SortedFunc(seq, compare)
}

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

func compare[O Ordered](a O, b O) int {
	return a.Order() - b.Order()
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
