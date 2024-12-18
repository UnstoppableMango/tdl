package crd2pulumi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
)

var (
	CrdRegex = regexp.MustCompile(`.*\.ya?ml`)
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
	workdir, workfs, err := t.tmpfs(base)
	if err != nil {
		return nil, fmt.Errorf("creating working directory: %w", err)
	}

	inputs := []string{}
	err = afero.Walk(afero.NewRegexpFs(src, CrdRegex), "",
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == "" || !CrdRegex.MatchString(path) {
				log.Debugf("ignoring %s", path)
				return nil
			}

			s, err := src.Open(path)
			if err != nil {
				return fmt.Errorf("opening %s: %w", path, err)
			}

			name := filepath.Base(path)
			if err = afero.WriteReader(workfs, name, s); err != nil {
				return fmt.Errorf("copying %s: %w", path, err)
			}

			inputs = append(inputs, name)
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("reading input paths: %w", err)
	}
	if len(inputs) == 0 {
		return nil, errors.New("no input files")
	}

	outdir, outfs, err := t.tmpfs(base)
	if err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	paths := t.Paths(outdir)
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
		log.Info("tool output", "stdout", stdout)
	}

	return outfs, nil
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
