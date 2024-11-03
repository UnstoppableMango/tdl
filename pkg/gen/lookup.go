package gen

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/unstoppablemango/tdl/internal/util"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

var ErrNotFound = errors.New("not found")

func FromPath(token tdl.Token, options ...CliOption) (tdl.Generator, error) {
	path, err := exec.LookPath(token.Name)
	if err != nil {
		return nil, fmt.Errorf("bin from path: %w", err)
	}

	return NewCli(path, options...), nil
}

func Name(token tdl.Token) (tdl.Generator, error) {
	switch token.Name {
	case "ts":
		fallthrough
	case "uml2ts":
		return localRepo("uml2ts")
	}

	return nil, fmt.Errorf("%w: %s", ErrNotFound, token.Name)
}

func Lookup(tokenish string) (tdl.Generator, error) {
	token := tdl.Token{Name: tokenish}

	generator, err := Name(token)
	if err == nil {
		return generator, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("lookup: %w", err)
	}

	generator, err = FromPath(token)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotFound, err)
	}

	return generator, nil
}

func localRepo(name string) (tdl.Generator, error) {
	gitRoot, err := util.GitRoot(context.Background())
	if err != nil {
		return nil, err
	}

	path := filepath.Join(gitRoot, "bin", name)
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	return NewCli(path), nil
}
