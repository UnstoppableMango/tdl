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
	"syscall"

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
// output directory as the run would leave it, and fails the target when
// it exits non-zero.
type Checks struct {
	// Dir is the directory holding tdl.toml, which a program named by a
	// relative path is relative to, as it is for post.
	Dir string

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
	return &Checks{Dir: cfg.Dir(), Commands: cmds}, nil
}

// Run stages the output directory as the run would leave it: a copy of
// what out holds now, less the files tdl owns there when clean is set,
// with files written over it. The copy is placed beside out, so a tool run
// there finds the configuration the real output would, or in the system's
// temporary directory when the directory above out cannot be written.
// Each command runs there in order, with {dir} naming that directory. The
// first to fail is an error carrying everything it printed on either
// stream. The directory is removed whatever happens. A nil Checks runs
// nothing.
func (c *Checks) Run(ctx context.Context, out string, files []*plugin.File, clean bool) (err error) {
	if c == nil || len(c.Commands) == 0 {
		return nil
	}
	parent, err := existingParent(out)
	if err != nil {
		return err
	}
	pattern := "." + filepath.Base(filepath.Clean(out)) + ".tdl-check-"
	staged, err := os.MkdirTemp(parent, pattern)
	if errors.Is(err, fs.ErrPermission) || errors.Is(err, syscall.EROFS) {
		staged, err = os.MkdirTemp("", pattern)
	}
	if err != nil {
		return fmt.Errorf("staging the files to check: %w", err)
	}
	defer func() {
		err = errors.Join(err, os.RemoveAll(staged))
	}()

	var skip map[string]bool
	if clean {
		if skip, err = Owned(out); err != nil {
			return err
		}
	}
	if err := copyTree(out, staged, skip); err != nil {
		return fmt.Errorf("staging the files to check: %w", err)
	}
	if _, err := Write(staged, files); err != nil {
		return fmt.Errorf("staging the files to check: %w", err)
	}
	for _, cmd := range c.Commands {
		if err := cmd.check(ctx, c.Dir, staged); err != nil {
			return err
		}
	}
	return nil
}

// copyTree copies everything under src into dst, keeping modes and
// symbolic links, but for the paths skip names, joined to src. A src that
// does not exist copies nothing.
func copyTree(src, dst string, skip map[string]bool) error {
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case rel == ".":
			return nil
		case skip[path]:
			return nil
		case d.IsDir():
			return os.Mkdir(target, 0o755)
		case d.Type()&fs.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case !d.Type().IsRegular():
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, info.Mode().Perm())
	})
	if errors.Is(err, fs.ErrNotExist) {
		if _, statErr := os.Lstat(src); errors.Is(statErr, fs.ErrNotExist) {
			return nil
		}
	}
	return err
}

// check runs one check command in dir. A program named by a relative path
// with a separator, such as ./node_modules/.bin/tsc, is relative to
// manifest, the directory holding tdl.toml.
func (c Command) check(ctx context.Context, manifest, dir string) error {
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		args[i] = strings.ReplaceAll(a, dirPlaceholder, dir)
	}
	if strings.ContainsRune(args[0], '/') && !filepath.IsAbs(args[0]) {
		args[0] = filepath.Join(manifest, args[0])
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
