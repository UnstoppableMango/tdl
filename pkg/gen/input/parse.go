package input

import (
	"fmt"
	"os"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

func ParseArgs(fsys afero.Fs, args []string) (tdl.Input, error) {
	switch len(args) {
	case 0:
		return Stdin(os.Stdin), nil
	case 1:
		return Open(fsys, args[0])
	default:
		return nil, fmt.Errorf("too many arguments: %#v", args)
	}
}
