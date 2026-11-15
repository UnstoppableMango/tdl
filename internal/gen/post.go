package gen

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/unstoppablemango/tdl/plugin"
)

// Post runs a target's post commands over every file it returned, each
// command's output feeding the next, and returns the files with what the
// last one wrote. {file} in a command line is the file's destination
// relative to t.Dir, where the command runs, so a tool finds its own
// configuration as though the file were in place. Nothing is written.
func Post(ctx context.Context, t Target, files []*plugin.File) ([]*plugin.File, error) {
	if len(t.Post) == 0 {
		return files, nil
	}
	out := make([]*plugin.File, len(files))
	for i, f := range files {
		dest, err := resolve(t.Out, f.GetPath())
		if err != nil {
			return nil, err
		}
		rel, err := relativeTo(t.Dir, dest)
		if err != nil {
			return nil, err
		}

		content := f.GetContent()
		for _, c := range t.Post {
			stdout, stderr, err := c.run(ctx, t.Dir, map[string]string{"{file}": rel}, content)
			if err != nil {
				return nil, withOutput(fmt.Errorf("post %s on %s: %w", c.Name, dest, err), stderr)
			}
			content = stdout
		}
		out[i] = &plugin.File{Path: f.GetPath(), Content: content}
	}
	return out, nil
}

// relativeTo writes path relative to dir with forward slashes, as a
// command line spells it.
func relativeTo(dir, path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absDir, abs)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}
