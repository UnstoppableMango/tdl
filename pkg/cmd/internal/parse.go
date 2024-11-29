package internal

import (
	"errors"
	"fmt"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

func ParseArgs(os tdl.OS, args []string) (res tdl.ParseResult, err error) {
	if res.Inputs, err = ParseInputArgs(os, args); err != nil {
		return
	}
	if res.Output, err = ParseOutputArgs(os, args); err != nil {
		return
	}

	return res, nil
}

func ParseOutputArgs(os tdl.OS, args []string) (tdl.Output, error) {
	switch len(args) {
	case 0:
		return nil, errors.New("no input file provided")
	case 1:
		return WriterOutput(os.Stdout()), nil
	case 2:
		return FsOutput(os.Fs(), args[1]), nil
	default:
		return nil, fmt.Errorf("too many arguments: %#v", args)
	}
}

func ParseInputArgs(os tdl.OS, args []string) ([]tdl.Input, error) {
	switch len(args) {
	case 0:
		return nil, errors.New("no input file provided")
	case 1:
		if args[0] == "-" {
			return collect(StdinInput(os.Stdin()))
		}
		fallthrough
	case 2:
		return collect(OpenFile(os.Fs(), args[0]))
	default:
		return nil, fmt.Errorf("too many arguments: %#v", args)
	}
}

func collect(i tdl.Input, err error) ([]tdl.Input, error) {
	if err != nil {
		return nil, err
	} else {
		return []tdl.Input{i}, nil
	}
}
