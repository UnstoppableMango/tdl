package conform

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/unmango/go/vcs/git"
	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
)

func LocalGitPath(ctx context.Context) (string, error) {
	root, err := git.Root(ctx)
	if err != nil {
		return "", fmt.Errorf("looking up local git path to conformance suites: %w", err)
	}

	return filepath.Join(root, "conformance"), nil
}

func ReadLocalGitSuite(ctx context.Context, fs afero.Fs, name string) (e2e.Suite, error) {
	return e2e.ReadLocalGitSuite(ctx, fs,
		filepath.Join("conformance", name),
	)
}

func ReadLocalGitTests(
	ctx context.Context,
	fs afero.Fs,
	name string,
	assertions map[string][]e2e.Assertion,
) (e2e.Suite, error) {
	return e2e.ReadLocalGitTests(ctx, fs,
		filepath.Join("conformance", name),
		assertions,
	)
}

func ReadLocalTypeScriptSuite(ctx context.Context, fs afero.Fs) (e2e.Suite, error) {
	return ReadLocalGitSuite(ctx, fs, "typescript")
}
