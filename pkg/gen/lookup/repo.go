package lookup

import (
	"context"
	"os"
	"path/filepath"

	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/tdl"
)

func localRepo(name string) (tdl.Generator, error) {
	gitRoot, err := util.GitRoot(context.Background())
	if err != nil {
		return nil, err
	}

	path := filepath.Join(gitRoot, "bin", name)
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	return gen.NewCli(path), nil
}
