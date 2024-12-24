package work

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	aferox "github.com/unmango/go/fs"
	"github.com/unmango/go/fs/github"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type PrepFunc func(context.Context, afero.Fs) error

type workspace struct {
	prepFuncs []PrepFunc
}

// Prepare implements tdl.Workspace.
func (d *workspace) Prepare(ctx context.Context) (afero.Fs, error) {
	osfs := afero.NewOsFs()
	dir, err := afero.TempDir(osfs, "", "")
	if err != nil {
		return nil, fmt.Errorf("creating temp directory: %w", err)
	}
	fs := afero.NewBasePathFs(osfs, dir)

	for _, prepare := range d.prepFuncs {
		if err := prepare(ctx, fs); err != nil {
			return nil, fmt.Errorf("preparing workspace: %w", err)
		}
	}

	return fs, nil
}

func NewLocal(options ...PrepFunc) tdl.Workspace {
	return &workspace{options}
}

func WithDir(path string) PrepFunc {
	return withDir(path).prepare
}

func WithStdin(stdin io.Reader) PrepFunc {
	return withStdin{stdin}.prepare
}

func WithGitHub(urls []string) PrepFunc {
	return withGitHub(urls).prepare
}

type withDir string

func (p withDir) prepare(ctx context.Context, fs afero.Fs) error {
	return aferox.Copy(afero.NewBasePathFs(afero.NewOsFs(), string(p)), fs)
}

type withStdin struct{ io.Reader }

func (r withStdin) prepare(ctx context.Context, fs afero.Fs) error {
	f, err := afero.TempFile(fs, "", "")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	if _, err = io.Copy(f, r); err != nil {
		return fmt.Errorf("copying stdin to %s: %w", f.Name(), err)
	}

	return nil
}

type withGitHub []string

func (urls withGitHub) prepare(ctx context.Context, fs afero.Fs) error {
	ghfs := github.NewFs(github.NewClient(nil))
	for _, p := range urls {
		if !strings.HasPrefix(p, "github.com") {
			log.Debugf("not a github url: %s", p)
			continue
		}

		f, err := ghfs.Open(p)
		if err != nil {
			return fmt.Errorf("opening %s: %w", p, err)
		}

		t, err := afero.TempFile(fs, "", "")
		if err != nil {
			return fmt.Errorf("creating temp file: %w", err)
		}

		log.Debug("copying", "github", f.Name(), "local", t.Name())
		if _, err = io.Copy(t, f); err != nil {
			return err
		}
	}

	return nil
}

func noop(context.Context, afero.Fs) error { return nil }
