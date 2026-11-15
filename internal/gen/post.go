package gen

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/unstoppablemango/tdl/plugin"
)

// Post runs a target's post commands over every file it returned, each
// command's output feeding the next, and returns the files with what the
// last one wrote. {file} in a command line is the file's destination
// relative to t.Root, where the command runs, so a tool finds its own
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
		rel, err := relativeTo(t.Root, dest)
		if err != nil {
			return nil, err
		}

		content := f.GetContent()
		for _, c := range t.Post {
			var stdout, stderr bytes.Buffer
			if err := c.run(ctx, t.Root, t.Root, map[string]string{"{file}": rel}, content, &stdout, &stderr); err != nil {
				return nil, withOutput(fmt.Errorf("post %s on %s: %w", c.Name, dest, err), stderr.Bytes())
			}
			content = stdout.Bytes()
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

// withPost folds a target's post command lines into its request hash, so
// output a nondeterministic backend wrote is regenerated, and processed
// again, when they change. A target with none keeps the request's hash.
func withPost(hash string, post []Command) string {
	if len(post) == 0 {
		return hash
	}
	var b strings.Builder
	b.WriteString(hash)
	for _, c := range post {
		b.WriteString("\x00" + c.Line)
	}
	return digest([]byte(b.String()))
}
