package gen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/unstoppablemango/tdl/internal/config"
	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// dirPlaceholder is replaced, in each word of a check command line, with
// the absolute path of the directory the files are staged in.
const dirPlaceholder = "{dir}"

// Checks is the commands a target block's check directive names, read
// through tdl.toml's [check] table. Each runs over a staged copy of the
// target's files and fails the target when it exits non-zero.
type Checks struct {
	Commands []Command
}

// ResolveChecks reads a target block's check directive through cfg, the
// tdl.toml nearest the model. It returns nil when the block has none, and
// an error naming the target for a name cfg does not declare, including
// when there is no tdl.toml.
func ResolveChecks(block *ir.TargetBlock, cfg *config.Config) (*Checks, error) {
	cmds, err := commands(block, cfg, host.CheckDirective, cfg.Check)
	if err != nil || cmds == nil {
		return nil, err
	}
	return &Checks{Commands: cmds}, nil
}

// Run writes files into a temporary directory laid out as out would be and
// placed beside it, so a tool run there finds the configuration the real
// output would, and runs each command there in order, with {dir} naming
// that directory. The first to fail is an error carrying everything it
// printed on either stream. The directory is removed whatever happens. A
// nil Checks runs nothing.
func (c *Checks) Run(ctx context.Context, out string, files []*plugin.File) (err error) {
	if c == nil || len(c.Commands) == 0 {
		return nil
	}
	parent, err := existingParent(out)
	if err != nil {
		return err
	}
	staged, err := os.MkdirTemp(parent, "."+filepath.Base(filepath.Clean(out))+".tdl-check-")
	if err != nil {
		return fmt.Errorf("staging the files to check: %w", err)
	}
	defer func() {
		err = errors.Join(err, os.RemoveAll(staged))
	}()

	if _, err := Write(staged, files); err != nil {
		return fmt.Errorf("staging the files to check: %w", err)
	}
	for _, cmd := range c.Commands {
		if err := cmd.check(ctx, staged); err != nil {
			return err
		}
	}
	return nil
}

// check runs one check command in dir.
func (c Command) check(ctx context.Context, dir string) error {
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		args[i] = strings.ReplaceAll(a, dirPlaceholder, dir)
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output

	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return nil
	case errors.Is(err, exec.ErrNotFound), errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("check command %s: %s was not found", c.Name, args[0])
	case errors.As(err, &exit):
		msg := fmt.Sprintf("check command %s failed: %v", c.Name, err)
		if s := strings.TrimSpace(output.String()); s != "" {
			msg += "\n" + s
		}
		return errors.New(msg)
	}
	return fmt.Errorf("check command %s: %w", c.Name, err)
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
