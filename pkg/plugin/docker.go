package plugin

import (
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen"
)

type Docker interface {
	tdl.Plugin
}

type docker struct {
	image    string
	priority int
}

// Generator implements tdl.Plugin.
func (d docker) Generator(tdl.Target) (tdl.Generator, error) {
	return gen.NewDocker(d.image), nil
}

// String implements tdl.Plugin.
func (d docker) String() string {
	return d.image
}
