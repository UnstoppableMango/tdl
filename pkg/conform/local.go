package conform

import (
	"context"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/unmango/go/vcs/git"
	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
)

// TODO: Currently this is executing as the CLI runs, meaning it can execute outside of the repo and thus fail
// var (
// 	TypeScriptSuite = RequireLocalSuite("typescript")
// )

func ReadLocalSuite(ctx context.Context, fs afero.Fs, name string) (Suite, error) {
	root, err := git.Root(ctx)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(root, "conformance", name)
	suite, err := e2e.ReadSuite(fs, path)
	if err != nil {
		return nil, err
	}

	return IncludeTests(suite), nil
}

func RequireLocalSuite(name string) Suite {
	ctx := context.Background()
	fs := afero.NewOsFs()
	suite, err := ReadLocalSuite(ctx, fs, name)
	if err != nil {
		panic(err)
	}

	return suite
}
