package run

import (
	"fmt"

	tdl "github.com/unstoppablemango/tdl/pkg"
	uxv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/ux/v1alpha1"
)

type Config struct {
	proto *uxv1alpha1.RunConfig
	os    tdl.OS
}

func (c *Config) Inputs() ([]tdl.Input, error) {
	return Inputs(c.os, c.proto)
}

func (c *Config) Output() (tdl.Output, error) {
	return Output(c.os, c.proto)
}

func NewConfig(os tdl.OS, config *uxv1alpha1.RunConfig) tdl.RunConfig {
	return &Config{config, os}
}

func Inputs(os tdl.OS, config *uxv1alpha1.RunConfig) ([]tdl.Input, error) {
	inputs := []tdl.Input{}
	for _, input := range config.Inputs {
		if i, err := parseInput(os, input); err != nil {
			return nil, fmt.Errorf("parsing run config: %w", err)
		} else {
			inputs = append(inputs, i)
		}
	}

	return inputs, nil
}

func parseInput(os tdl.OS, input *uxv1alpha1.Input) (tdl.Input, error) {
	switch {
	case input.GetStdin():
		return StdinInput(os.Stdin())
	case input.GetFile() != nil:
		return OpenFile(os.Fs(), input.GetFile().GetPath())
	default:
		return nil, fmt.Errorf("unsupported: %v", input)
	}
}

func Output(os tdl.OS, config *uxv1alpha1.RunConfig) (tdl.Output, error) {
	switch {
	case config.GetPath() != "":
		return FsOutput(os.Fs(), config.GetPath()), nil
	case config.GetStdout():
		fallthrough
	default:
		return WriterOutput(os.Stdout()), nil
	}
}
