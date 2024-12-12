package plugin

import (
	"context"
	"fmt"
	"os/exec"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen/cli"
	"github.com/unstoppablemango/tdl/pkg/meta"
)

type fromPath struct {
	name   string
	meta   meta.Map
	stdout bool
	order  int
}

// Meta implements tdl.GeneratorPlugin.
func (f fromPath) Meta() tdl.Meta {
	return meta.Map{
		"name":  f.name,
		"stout": fmt.Sprint(f.stdout),
		"order": fmt.Sprint(f.order),
	}
}

// Generator implements tdl.Plugin.
func (f fromPath) Generator(context.Context, tdl.Meta) (tdl.Generator, error) {
	path, err := exec.LookPath(f.name)
	if err != nil {
		return nil, fmt.Errorf("from path: %w", err)
	}

	return cli.New(path,
		cli.WithExpectStdout(f.stdout),
	), nil
}

func (f fromPath) Supports(target tdl.Target) bool {
	return target.String() == "TypeScript" // TODO
}

// String implements tdl.Plugin.
func (f fromPath) String() string {
	if path, err := exec.LookPath(f.name); err != nil {
		return f.name
	} else {
		return path
	}
}

func (f fromPath) Order() int {
	return f.order
}

func FromPath(name string) *fromPath {
	return &fromPath{name, meta.Map{}, false, 69}
}
