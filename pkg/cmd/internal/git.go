package internal

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/unmango/go/vcs/git"
)

func OpenGitIgnore(ctx context.Context) (io.ReadCloser, error) {
	if root, err := git.Root(ctx); err != nil {
		return nil, fmt.Errorf("locating git root: %w", err)
	} else {
		return os.Open(filepath.Join(root, ".gitignore"))
	}
}
