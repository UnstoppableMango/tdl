package input

import (
	"errors"
	"fmt"

	aferox "github.com/unmango/go/fs"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

func ParseArgs(os tdl.OS, args []string) (res tdl.ParseResult, err error) {
	switch len(args) {
	case 0:
		err = errors.New("no arguments provided")
	case 1:
		if args[0] == "-" {
			res.Inputs = append(res.Inputs, Stdin(os.Stdin()))
		}

		res.Output = aferox.NewWriter(os.Stdout())
		fallthrough
	case 2:
		if input, err := Open(os.Fs(), args[0]); err != nil {
			return res, err
		} else {
			res.Inputs = append(res.Inputs, input)
		}

		if output, err := Fs(os.Fs(), args[1]); err != nil {
			return res, err
		} else {
			res.Output = output
		}
	default:
		return res, fmt.Errorf("too many arguments: %#v", args)
	}

	return
}
