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
	"github.com/unmango/go/fs/filter"
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

func (t Tool) Execute(ctx context.Context, src afero.Fs, args []string) (afero.Fs, error) {
	base := afero.NewOsFs()
	workdir, workfs, err := tmpfs(base)
	if err != nil {
		return nil, fmt.Errorf("creating working directory: %w", err)
	}

	flags := t.NewFlagSet()
	if err = flags.Parse(args); err != nil {
		return nil, fmt.Errorf("applying extra args: %w", err)
	}

	var inputs []string
	if len(flags.Args()) > 0 {
		inputs = flags.Args()
	} else {
		inputs, err = search(src)
	}
	if err != nil {
		return nil, fmt.Errorf("reading input paths: %w", err)
	}
	if len(inputs) == 0 {
		return nil, errors.New("no input files")
	}

	relInputs := []string{}
	for _, i := range inputs {
		f, err := src.Open(i)
		if err != nil {
			return nil, fmt.Errorf("opening input: %w", err)
		}

		name := filepath.Base(i)
		if err = afero.WriteReader(workfs, name, f); err != nil {
			return nil, fmt.Errorf("copying input: %w", err)
		}

		relInputs = append(relInputs, name)
	}

	outdir, outfs, err := tmpfs(base)
	if err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	paths := t.Paths(outdir)
	args = append(t.Args(paths), relInputs...)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, t.path(), args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = workdir

	log.Info("executing command", "cmd", cmd, "work", workdir)
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("executing tool: %w: %s", err, stderr)
	}
	if stderr.Len() > 0 {
		return nil, fmt.Errorf("executing tool: %s", stderr)
	}
	if stdout.Len() > 0 {
		log.Info("tool output", "stdout", stdout)
	}

	return filter.NewFs(outfs, t.ShouldInclude), nil
}

func (t Tool) path() string {
	if t.Path != "" {
		return t.Path
	} else {
		return "crd2pulumi"
	}
}

func tmpfs(fs afero.Fs) (string, afero.Fs, error) {
	if name, err := afero.TempDir(fs, "", ""); err != nil {
		return "", nil, err
	} else {
		return name, afero.NewBasePathFs(fs, name), nil
	}
}

func search(src afero.Fs) (inputs []string, err error) {
	err = afero.Walk(src, "",
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == "" {
				return nil
			}
			switch info.Name() {
			case ".git":
				fallthrough
			case ".idea":
				return filepath.SkipDir
			}
			if !CrdRegex.MatchString(path) {
				log.Debugf("ignoring %s", path)
				return nil
			}
			name := filepath.Base(path)
			inputs = append(inputs, name)
			return nil
		},
	)
	if err != nil {
		return nil, err
	} else {
		return
	}
}
