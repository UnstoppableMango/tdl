package plugin

import (
	"context"
	"errors"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin/cache"
	"github.com/unstoppablemango/tdl/pkg/plugin/github"
)

var static = []tdl.Plugin{
	github.NewUml2Ts(),
}

func FirstAvailable(target tdl.Target) (tdl.Plugin, error) {
	for _, p := range static {
		if err := tryCache(p); err != nil {
			return nil, err
		}

		return p, nil
	}

	for p := range target.Plugins() {
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
