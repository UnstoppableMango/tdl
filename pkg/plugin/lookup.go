package plugin

import (
	"context"
	"errors"

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

func tryCache(p tdl.Plugin) error {
	c := cache.XdgBinHome
	r, ok := p.(cache.Cachable)
	if !ok || r.Cached(c) {
		return nil
	}

	return r.Cache(context.Background(), c)
}
