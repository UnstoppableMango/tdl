package crd2pulumi

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"regexp"

	"github.com/charmbracelet/log"
	"github.com/dlclark/regexp2"
	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

var (
	crdRegex    = regexp.MustCompile(".*\\.ya?ml")
	outputRegex = regexp2.MustCompile("(?!.*\\.ya?ml)", 0)
)

type LangOptions struct {
	enabled bool
	name    string
	path    string
}

type tool struct {
	nodejs  *LangOptions
	python  *LangOptions
	dotnet  *LangOptions
	golang  *LangOptions
	java    *LangOptions
	force   bool
	version string
}

func (t tool) String() string {
	return "crd2pulumi"
}

func (t tool) Execute(ctx context.Context, src afero.Fs) (afero.Fs, error) {
	log.Debug("creating temp directory")
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("creating exec context: %w", err)
	}

	langs := map[string]*LangOptions{
		"nodejs": t.nodejs,
		"python": t.python,
		"dotnet": t.dotnet,
		"golang": t.golang,
		"java":   t.java,
	}

	output := []afero.Fs{}
	args := []string{}
	for k, v := range maps.All(langs) {
		if v == nil {
			continue
		}

		if v.enabled {
			args = append(args, "--"+k)
			output = append(output, outputFs(v.path))
		}
		if v.name != "" {
			args = append(args,
				fmt.Sprintf("--%sName", k),
				v.name,
			)
		}
		if v.path != "" {
			args = append(args,
				fmt.Sprintf("--%sPath", k),
				v.path,
			)
		}
	}

	if t.version != "" {
		args = append(args, "--version", t.version)
	}
	if t.force {
		args = append(args, "--force")
	}

	tmpfs := afero.NewBasePathFs(afero.NewOsFs(), tmp)
	crdfs := afero.NewReadOnlyFs(afero.NewRegexpFs(src, crdRegex))
	err = afero.Walk(crdfs, "",
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == "" {
				return nil // skip root
			}
			if info.IsDir() {
				return tmpfs.Mkdir(path, os.ModeDir)
			}

			f, err := crdfs.Open(path)
			if err != nil {
				return fmt.Errorf("copying to context: %w", err)
			}

			if err = afero.WriteReader(tmpfs, path, f); err != nil {
				return fmt.Errorf("copying to context: %w", err)
			}

			args = append(args, path)
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("building args: %w", err)
	}

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, "crd2pulumi", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = tmp

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

	log.Debugf("returning a new BasePathFs at %s", tmp)
	return afero.NewBasePathFs(afero.NewOsFs(), tmp), nil
}

func New() tdl.Tool {
	return tool{}
}

func outputFs(path string) afero.Fs {
	return afero.NewBasePathFs(afero.NewOsFs(), path)
}
