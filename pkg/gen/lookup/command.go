package lookup

import (
	"fmt"
	"io"

	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/tdl"
)

func Execute(tokenish string, stdout io.Writer) error {
	generator, err := FromPath(tdl.Token{Name: tokenish})
	if err != nil {
		return fmt.Errorf("PATH lookup: %w", err)
	}

	cli, ok := generator.(*gen.Cli)
	if !ok {
		return fmt.Errorf("unsupported generator: %v", generator)
	}

	_, err = fmt.Fprintln(stdout, cli.Name())
	return err
}
