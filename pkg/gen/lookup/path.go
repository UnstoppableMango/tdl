package lookup

import (
	"fmt"
	"os/exec"

	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/tdl"
)

func FromPath(token tdl.Token, options ...gen.CliOption) (tdl.Generator, error) {
	path, err := exec.LookPath(token.Name)
	if err != nil {
		return nil, fmt.Errorf("looking up bin from path: %w", err)
	}

	return gen.NewCli(path, options...), nil
}
