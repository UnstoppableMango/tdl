package crd2pulumi

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	aferox "github.com/unmango/go/fs"
)

var (
	crdRegex = regexp.MustCompile(`.*\.ya?ml`)
)

type LangOptions struct {
	Enabled bool
	Name    string
	Path    string
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
	base := afero.NewOsFs()
	work, workfs, err := t.tmpfs(base)
	if err != nil {
		return nil, fmt.Errorf("creating working directory: %w", err)
	}

	crdfs := afero.NewReadOnlyFs(afero.NewRegexpFs(src, crdRegex))
	if err = aferox.Copy(crdfs, workfs); err != nil {
		return nil, fmt.Errorf("copying source files: %w", err)
	}

	out, outfs, err := t.tmpfs(base)
	if err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	paths := t.Paths(out)
	args := t.Args(paths)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, "crd2pulumi", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = work

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

	return outfs, nil
}

func (t Tool) tmpfs(fs afero.Fs) (string, afero.Fs, error) {
	if name, err := afero.TempDir(fs, "", ""); err != nil {
		return "", nil, err
	} else {
		return name, afero.NewBasePathFs(fs, name), nil
	}
}

func (t Tool) langs() map[string]*LangOptions {
	return map[string]*LangOptions{
		"nodejs": t.NodeJS,
		"python": t.Python,
		"dotnet": t.Dotnet,
		"golang": t.Go,
		"java":   t.Java,
	}
}

func (t Tool) Paths(root string) map[string]string {
	paths := map[string]string{}
	for k, v := range t.langs() {
		if v == nil {
			continue
		}

		if v.Path != "" {
			paths[k] = v.Path
		} else {
			paths[k] = filepath.Join(root, k)
		}
	}

	return paths
}

func (t Tool) Args(paths map[string]string) []string {
	args := ArgBuilder{}
	for k, v := range t.langs() {
		if v == nil {
			continue
		}

		if v.Enabled {
			args = args.LangOpt(k)
		}
		if v.Name != "" {
			args = args.NameOpt(k, v.Name)
		}
		if v.Enabled || v.Path != "" {
			args = args.PathOpt(k, paths[k])
		}
	}

	if t.Version != "" {
		args = args.VersionOpt(t.Version)
	}
	if t.Force {
		args = args.ForceOpt()
	}

	return args
}
