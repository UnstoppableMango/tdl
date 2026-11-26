package gen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/unstoppablemango/tdl/internal/config"
	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// filePlaceholder is replaced, in each word of a post command line, with
// the destination path of the file being processed.
const filePlaceholder = "{file}"

// Post is the commands a target block's post directive names, read through
// tdl.toml's [post] table. Each is a filter over one file: tdl writes the
// content to its standard input and takes its standard output in its
// place.
type Post struct {
	// Dir is the directory holding tdl.toml. Every command runs there, and
	// {file} is relative to it.
	Dir string

	// Commands run in the order the directive names them, each one's output
	// feeding the next.
	Commands []host.Command
}

// ResolvePost reads a target block's post directive through cfg, the
// tdl.toml nearest the model. It returns nil when the block has none, and
// an error naming the target for a name cfg does not declare, including
// when there is no tdl.toml.
func ResolvePost(block *ir.TargetBlock, cfg *config.Config) (*Post, error) {
	cmds, err := host.Commands(block, cfg, host.PostDirective, cfg.Post)
	if err != nil || cmds == nil {
		return nil, err
	}
	return &Post{Dir: cfg.Dir(), Commands: cmds}, nil
}

// Apply runs every command over every file, returning the files with
// their processed content. out is the target's output directory, which a
// file's path is relative to. A nil Post returns the files unchanged.
func (p *Post) Apply(ctx context.Context, out string, files []*plugin.File) ([]*plugin.File, error) {
	if p == nil || len(p.Commands) == 0 {
		return files, nil
	}
	processed := make([]*plugin.File, len(files))
	for i, f := range files {
		// A path the write would refuse is refused before any command
		// sees it.
		dest, err := resolve(out, f.GetPath())
		if err != nil {
			return nil, err
		}
		if dest, err = filepath.Abs(dest); err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(p.Dir, dest)
		if err != nil {
			return nil, err
		}

		content := f.GetContent()
		for _, c := range p.Commands {
			if content, err = filter(ctx, c, p.Dir, rel, content); err != nil {
				return nil, err
			}
		}
		processed[i] = &plugin.File{Path: f.GetPath(), Content: content}
	}
	return processed, nil
}

// filter runs one post command over one file's content.
func filter(ctx context.Context, c host.Command, dir, file string, content []byte) ([]byte, error) {
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		args[i] = strings.ReplaceAll(a, filePlaceholder, file)
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(content)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return stdout.Bytes(), nil
	case errors.Is(err, exec.ErrNotFound), errors.Is(err, os.ErrNotExist):
		return nil, fmt.Errorf("post command %s on %s: %s was not found", c.Name, file, args[0])
	case errors.As(err, &exit):
		msg := fmt.Sprintf("post command %s failed on %s: %v", c.Name, file, err)
		if s := strings.TrimSpace(stderr.String()); s != "" {
			msg += "\n" + s
		}
		return nil, errors.New(msg)
	}
	return nil, fmt.Errorf("post command %s on %s: %w", c.Name, file, err)
}
