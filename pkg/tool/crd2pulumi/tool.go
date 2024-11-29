package crd2pulumi

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"os"
	"os/exec"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
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

	args := []string{}
	for k, v := range maps.All(langs) {
		if v == nil {
			continue
		}

		if v.enabled {
			args = append(args, "--"+k)
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

	log.Debugf("returning a new BasePathFs at %s", tmp)
	return afero.NewBasePathFs(afero.NewOsFs(), tmp), nil
}

func New() tdl.Tool {
	return tool{}
}
