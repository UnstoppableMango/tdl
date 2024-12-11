package crd2pulumi

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"regexp"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	aferox "github.com/unmango/go/fs"
)

var (
	crdRegex = regexp.MustCompile(`.*\.ya?ml`)
)

type Tool struct {
	Options
	Path string
}

func (t Tool) String() string {
	return "crd2pulumi"
}

func (t Tool) Execute(ctx context.Context, src afero.Fs) (afero.Fs, error) {
	base := afero.NewOsFs()
	crdfs := afero.NewReadOnlyFs(afero.NewRegexpFs(src, crdRegex))
	inputs, err := aferox.Fold(crdfs, "",
		func(path string, info fs.FileInfo, paths []string, err error) ([]string, error) {
			return append(paths, path), err
		},
		[]string{},
	)
	if err != nil {
		return nil, fmt.Errorf("reading input paths: %w", err)
	}

	workdir, workfs, err := t.tmpfs(base)
	if err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	paths := t.Paths(workdir)
	args := append(t.Args(paths), inputs...)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, t.path(), args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = workdir

	log.Debug("executing command")
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("executing tool: %w: %s", err, stderr)
	}
	if stderr.Len() > 0 {
		return nil, fmt.Errorf("executing tool: %s", stderr)
	}
	if stdout.Len() > 0 {
		log.Errorf("executing tool: %s", stdout)
	}

	return workfs, nil
}

func (t Tool) tmpfs(fs afero.Fs) (string, afero.Fs, error) {
	if name, err := afero.TempDir(fs, "", ""); err != nil {
		return "", nil, err
	} else {
		return name, afero.NewBasePathFs(fs, name), nil
	}
}

func (t Tool) path() string {
	if t.Path != "" {
		return t.Path
	} else {
		return "crd2pulumi"
	}
}
