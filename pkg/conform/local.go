package conform

import (
	"context"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/unmango/go/vcs/git"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

type localSuite struct {
	fs    afero.Fs
	suite string
}

// Describe implements Suite.
func (l *localSuite) Describe(tdl.Generator) {
	ctx := context.Background()
	root, err := git.Root(ctx)
	if err != nil {
		panic(err)
	}

	path := filepath.Join(root, "conformance", l.suite)
	raw, err := testing.Discover(l.fs, path)
	if err != nil {
		panic(err)
	}

	tests := make([]*testing.Test, len(raw))
	for i, test := range raw {
		tests[i] = &testing.Test{
			Name: test.Name,
		}
	}

	panic("unimplemented")
}

func NewLocalSuite(fs afero.Fs, name string) Suite {
	return &localSuite{fs, name}
}
