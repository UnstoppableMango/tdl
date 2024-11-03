package repo

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type Local struct {
	path   string
	target tdl.Target
}

// Available implements tdl.Plugin.
func (l *Local) Available() bool {
	stat, err := os.Stat(l.gitDir())
	if err != nil {
		log.Error(err)
		return false
	}

	return stat.IsDir()
}

// String implements tdl.Plugin.
func (l *Local) String() string {
	return l.path
}

func (l *Local) gitDir() string {
	return filepath.Join(l.path, ".git")
}

func NewLocal(path string, target tdl.Target) *Local {
	return &Local{path: path, target: target}
}
