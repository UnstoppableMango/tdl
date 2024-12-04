package crd2pulumi

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/charmbracelet/log"
	"github.com/dlclark/regexp2"
	"github.com/spf13/afero"
)

var (
	crdRegex    = regexp.MustCompile(".*\\.ya?ml")
	outputRegex = regexp2.MustCompile("(?!.*\\.ya?ml)", 0)
)

type LangOptions struct {
	Enabled bool
	Name    string
	Path    string
}

func (o *LangOptions) args(lang string) (args []string) {
	if o.Enabled {
		args = append(args, "--"+lang)
	}
	if o.Name != "" {
		args = append(args,
			fmt.Sprintf("--%sName", lang),
			o.Name,
		)
	}
	if o.Path != "" {
		args = append(args,
			fmt.Sprintf("--%sPath", lang),
			o.Path,
		)
	}

	return
}

type Tool struct {
	NodeJS  *LangOptions
	Python  *LangOptions
	Dotnet  *LangOptions
	Go      *LangOptions
	Java    *LangOptions
	Force   bool
	Version string
}

func (t Tool) String() string {
	return "crd2pulumi"
}

func (t Tool) Execute(ctx context.Context, src afero.Fs) (afero.Fs, error) {
	log.Debug("creating temp directory")
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("creating exec context: %w", err)
	}

	langs := map[string]*LangOptions{
		"nodejs": t.NodeJS,
		"python": t.Python,
		"dotnet": t.Dotnet,
		"golang": t.Go,
		"java":   t.Java,
	}

	args := []string{}
	for k, v := range maps.All(langs) {
		if v == nil {
			continue
		}

		if v.Enabled {
			args = append(args, "--"+k)
		}
		if v.Name != "" {
			args = append(args,
				fmt.Sprintf("--%sName", k),
				v.Name,
			)
		}
		if v.Path != "" {
			args = append(args,
				fmt.Sprintf("--%sPath", k),
				v.Path,
			)
		} else {
			args = append(args,
				fmt.Sprintf("--%sPath", k),
				filepath.Join(tmp, k),
			)
		}
	}

	if t.Version != "" {
		args = append(args, "--version", t.Version)
	}
	if t.Force {
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
