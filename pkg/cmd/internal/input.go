package internal

import (
	"context"
	"path/filepath"
	"regexp"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/unmango/go/fs/filter"
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

func FilterInput(fs afero.Fs, cwd string, expressions []string) afero.Fs {
	var matches []string
	for _, e := range expressions {
		if m, err := afero.Glob(fs, e); err != nil {
			log.Debug(err)
			continue
		} else {
			matches = append(matches, m...)
		}
	}

	return filter.NewFs(fs, func(s string) bool {
		for _, m := range matches {
			a := filepath.Clean(filepath.Join(cwd, s))
			b := filepath.Clean(filepath.Join(cwd, m))

			if a == b {
				return true
			}
		}

		for _, e := range expressions {
			if ok, err := filepath.Match(e, s); err != nil {
				log.Debug(err)
			} else if ok {
				log.Debugf("matched expression %s: %s", e, s)
				return true
			}

			if re, err := regexp.Compile(e); err != nil {
				log.Debug(err)
				continue
			} else if re.MatchString(s) {
				log.Debugf("matched regex %s: %s", e, s)
				return true
			}
		}

		log.Debugf("no match: %s", s)
		return false
	})
}
