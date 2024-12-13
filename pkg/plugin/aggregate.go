package plugin

import (
	"context"
	"fmt"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/meta"
)

const UnwrapDepth = 3

type Aggregate []tdl.Plugin

// Prepare implements tdl.Plugin.
func (a Aggregate) Prepare(ctx context.Context) error {
	for _, p := range a {
		if err := p.Prepare(ctx); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
	}

	return nil
}

// Meta implements tdl.Plugin.
func (a Aggregate) Meta() tdl.Meta {
	return meta.Map{}
}

// Supports implements tdl.Plugin
func (a Aggregate) Supports(target tdl.Target) bool {
	for _, p := range a {
		if p.Supports(target) {
			return true
		}
	}

	return false
}

// String implements tdl.Plugin.
func (a Aggregate) String() string {
	return fmt.Sprintf("%v", []tdl.Plugin(a))
}

func NewAggregate(plugins ...tdl.Plugin) Aggregate {
	return Aggregate(plugins)
}

var _ tdl.Plugin = Aggregate{}
