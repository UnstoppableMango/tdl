package internal

import (
	"context"

	"github.com/spf13/afero"
	"github.com/unmango/go/fs/ignore"
	"github.com/unstoppablemango/tdl/pkg/tool"
)

var IgnorePatterns = tool.DefaultIgnorePatterns

func CwdFs(ctx context.Context, cwd string) (afero.Fs, error) {
	src := afero.NewBasePathFs(afero.NewOsFs(), cwd)
	if i, err := OpenGitIgnore(ctx); err == nil {
		return ignore.NewFsFromGitIgnoreReader(src, i)
	}

	return ignore.NewFsFromGitIgnoreLines(src, IgnorePatterns...), nil
}
