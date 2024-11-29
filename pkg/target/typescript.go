package target

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/unmango/go/iter"
	"github.com/unmango/go/slices"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

type typescript string

// Generator implements tdl.Target.
func (t typescript) Generator(available iter.Seq[tdl.Plugin]) (tdl.Generator, error) {
	for p := range available {
		if p.String() == "uml2ts" {
			return p.Generator(context.TODO(), t)
		}
	}

	return nil, errors.New("no suitable plugin")
}

var TypeScript typescript = "TypeScript"

// Choose implements tdl.Target.
func (t typescript) Choose(available []tdl.SinkGenerator) (tdl.SinkGenerator, error) {
	if len(available) == 0 {
		return nil, errors.New("no generators to choose from")
	}

	errs := []error{}
	for _, g := range available {
		if err := supported(g); err != nil {
			errs = append(errs, err)
		} else {
			return g, nil
		}
	}

	return nil, errors.Join(errs...)
}

// Plugins implements tdl.Target.
func (t typescript) Plugins() iter.Seq[tdl.Plugin] {
	return slices.Values([]tdl.Plugin{
		plugin.Uml2Ts,
	})
}

// String implements tdl.Target.
func (t typescript) String() string {
	return string(t)
}

func supported(g tdl.SinkGenerator) error {
	stringer, ok := g.(fmt.Stringer)
	if !ok {
		return Reject(g, "unable to inspec generator")
	}

	name := filepath.Base(stringer.String())
	if name != "uml2ts" {
		return Reject(g, "only uml2ts is supported")
	}

	return nil
}
