package plugin

import (
	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

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
