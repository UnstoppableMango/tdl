package plugin

import (
	"context"
	"errors"

	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin/cache"
)

func FirstAvailable(target tdl.Target) (tdl.Plugin, error) {
	if len(static) > 0 {
		p := static[0]
		if err := tryCache(p); err != nil {
			return nil, err
		}

		return p, nil
	}

	for _, p := range static {
		return p, nil
	}

	return nil, errors.New("no plugins available")
}

func Find(plugins iter.Seq[tdl.Plugin], pred func(tdl.Plugin) bool) (tdl.Plugin, bool) {
	for plugin := range plugins {
		for _, nested := range Unwrap(plugin) {
			if pred(nested) {
				return nested, true
			}
		}
	}

	return nil, false
}

func tryCache(p tdl.Plugin) error {
	c := cache.XdgBinHome
	r, ok := p.(cache.Cachable)
	if !ok || r.Cached(c) {
		return nil
	}

	return r.Cache(context.Background(), c)
}
