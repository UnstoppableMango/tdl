package plugin

import (
	"fmt"
	"os/exec"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen"
)

type fromPath struct {
	name  string
	order int
}

// Generator implements tdl.Plugin.
func (f fromPath) Generator(tdl.Target) (tdl.Generator, error) {
	path, err := exec.LookPath(f.name)
	if err != nil {
		return nil, fmt.Errorf("from path: %w", err)
	}

	return gen.NewCli(path), nil
}

// String implements tdl.Plugin.
func (f fromPath) String() string {
	return fmt.Sprintf("PATH: %s", f.name)
}

func (f fromPath) Order() int {
	return f.order
}

func FromPath(name string) tdl.Plugin {
	return &fromPath{name, 69}
}
