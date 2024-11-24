package conform

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/unmango/go/vcs/git"
	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
)

type Suite interface {
	GeneratorSuite
}

func ReadLocalGitSuite(ctx context.Context, fs afero.Fs, name string) (e2e.Suite, error) {
	path, err := pathTo(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("finding local git path: %w", err)
	}

	return e2e.ReadSuite(fs, path)
}

func ReadLocalGitTests(
	ctx context.Context,
	fs afero.Fs,
	name string,
	assertions map[string][]e2e.Assertion,
) (e2e.Suite, error) {
	path, err := pathTo(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("finding local git path: %w", err)
	}

	return e2e.ReadTests(fs, path, assertions)
}

func pathTo(ctx context.Context, name string) (string, error) {
	root, err := git.Root(ctx)
	if err != nil {
		return "", err
	}

	return filepath.Join(root, "conformance", name), nil
}
