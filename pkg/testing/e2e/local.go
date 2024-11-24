package e2e

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/unmango/go/vcs/git"
)

func ReadLocalGitSuite(ctx context.Context, fs afero.Fs, name string) (Suite, error) {
	path, err := pathTo(ctx, name)
	if err != nil {
		return nil, err
	}

	return ReadSuite(fs, path)
}

func ReadLocalGitTests(
	ctx context.Context,
	fs afero.Fs,
	name string,
	assertions map[string][]Assertion,
) (Suite, error) {
	path, err := pathTo(ctx, name)
	if err != nil {
		return nil, err
	}

	return ReadTests(fs, path, assertions)
}

func pathTo(ctx context.Context, path string) (string, error) {
	root, err := git.Root(ctx)
	if err != nil {
		return "", fmt.Errorf("creating local git path: %w", err)
	}

	return filepath.Join(root, path), nil
}
