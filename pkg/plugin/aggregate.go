package plugin

import (
	"errors"
	"fmt"
	"iter"
	"slices"

	"github.com/charmbracelet/log"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

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

func compare[O Ordered](a O, b O) int {
	return a.Order() - b.Order()
}
