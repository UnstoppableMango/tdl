package target

import (
	"errors"

	"github.com/charmbracelet/log"
	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/meta"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

type typescript string

var TypeScript typescript = "TypeScript"

func (t typescript) Meta() tdl.Meta {
	return meta.Map{
		meta.WellKnown.Name: string(t),
		meta.WellKnown.Lang: meta.Lang.TypeScript,
	}
}

// Generator implements tdl.Target.
func (t typescript) Choose(available iter.Seq[tdl.Plugin]) (tdl.Plugin, error) {
	plugin, ok := plugin.Find(available, func(p tdl.Plugin) bool {
		log.Debugf("considering %s", p)
		return plugin.MatchesName(p, "uml2ts")
	})
	if !ok {
		return nil, errors.New("no suitable plugin")
	}

	return plugin, nil
}

// String implements tdl.Target.
func (t typescript) String() string {
	return string(t)
}
