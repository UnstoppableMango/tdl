package lookup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/tdl"
)

func localRepo(name string) (tdl.Generator, error) {
	revParse, err := exec.Command(
		"git", "rev-parse", "--show-toplevel",
	).CombinedOutput()
	if err != nil {
		return nil, err
	}

	gitRoot := strings.TrimSpace(string(revParse))
	path := filepath.Join(gitRoot, "bin", name)

	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	return gen.NewCli(path), nil
}
