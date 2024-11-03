package lookup

import (
	"errors"
	"fmt"
	"io"
	"os/exec"

	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/tdl"
)

func Execute(tokenish string, stdout io.Writer) error {
	token := tdl.Token{Name: tokenish}

	generator, err := gen.Name(token)
	if err != nil && !errors.Is(err, gen.ErrNotFound) {
		return fmt.Errorf("lookup: %w", err)
	}

	generator, err = gen.FromPath(token)
	if errors.Is(err, exec.ErrNotFound) {
		fmt.Fprintf(stdout, "Not Found: %s\n", tokenish)
	}
	if err != nil {
		return err
	}

	fmt.Fprintln(stdout, generator)
	return nil
}
