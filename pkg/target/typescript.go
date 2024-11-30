package target

import (
	"context"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

type typescript string

// Generator implements tdl.Target.
func (t typescript) Generator(available iter.Seq[tdl.Plugin]) (tdl.Generator, error) {
	plugin, ok := plugin.Find(available, func(p tdl.Plugin) bool {
		log.Debugf("considering %s", p)
		return p.String() == "uml2ts"
	})
	if !ok {
		return nil, errors.New("no suitable plugin")
	} else {
		return plugin.Generator(context.TODO(), t)
	}
}

var TypeScript typescript = "TypeScript"

// Choose implements tdl.Target.
func (t typescript) Choose(available []tdl.SinkGenerator) (tdl.SinkGenerator, error) {
	if len(available) == 0 {
		return nil, errors.New("no generators to choose from")
	}

	errs := []error{}
	for _, g := range available {
		return g, nil
	}

	return nil, errors.Join(errs...)
}

// String implements tdl.Target.
func (t typescript) String() string {
	return string(t)
}
