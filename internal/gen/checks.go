package gen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/unstoppablemango/tdl/plugin"
)

// Check runs a target's check commands over its files, staged in a
// temporary directory laid out as the output directory would be and
// placed beside it, so a tool run there finds the configuration the real
// output would. Each command runs in that directory, which {dir} in its
// command line also names, and the first to fail is an error carrying
// everything it printed. The directory is removed whatever happens.
func Check(ctx context.Context, t Target, files []*plugin.File) (err error) {
	if len(t.Check) == 0 {
		return nil
	}
	parent, err := existingParent(t.Out)
	if err != nil {
		return err
	}
	staged, err := os.MkdirTemp(parent, "."+filepath.Base(filepath.Clean(t.Out))+".tdl-check-")
	if err != nil {
		return fmt.Errorf("staging the files to check: %w", err)
	}
	defer func() {
		err = errors.Join(err, os.RemoveAll(staged))
	}()

	if _, err := Write(staged, files); err != nil {
		return fmt.Errorf("staging the files to check: %w", err)
	}
	for _, c := range t.Check {
		var output bytes.Buffer
		if err := c.run(ctx, t.Root, staged, map[string]string{"{dir}": staged}, nil, &output, &output); err != nil {
			return withOutput(fmt.Errorf("check %s: %w", c.Name, err), output.Bytes())
		}
	}
	return nil
}

// existingParent returns the nearest directory above out that exists,
// absolute, so staging creates nothing that outlives it.
func existingParent(out string) (string, error) {
	dir, err := filepath.Abs(out)
	if err != nil {
		return "", err
	}
	for {
		parent := filepath.Dir(dir)
		info, err := os.Stat(parent)
		switch {
		case err == nil && info.IsDir():
			return parent, nil
		case err != nil && !errors.Is(err, fs.ErrNotExist):
			return "", err
		case parent == dir:
			return "", fmt.Errorf("no directory above %s exists", out)
		}
		dir = parent
	}
}
