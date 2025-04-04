package run

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/target"
	uxv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/ux/v1alpha1"
)

var (
	targetIndex = 0
	inputIndex  = 1
	outputIndex = 2
)

func ParseArgs(args []string) (*uxv1alpha1.RunConfig, error) {
	if len(args) == 0 {
		return nil, errors.New("not enough arguments")
	}
	if len(args) == 1 {
		return nil, errors.New("no input specified")
	}

	log := log.With("args", args)
	input := &uxv1alpha1.Input{}
	if len(args) == 1 || args[inputIndex] == "-" {
		log.Debug("choosing stdin")
		input.Value = &uxv1alpha1.Input_Stdin{
			Stdin: true,
		}
	} else {
		log.Debugf("choosing input file: %s", args[inputIndex])
		input.Value = &uxv1alpha1.Input_File{
			File: &uxv1alpha1.FileInput{
				Path: args[inputIndex],
			},
		}
	}

	config := &uxv1alpha1.RunConfig{
		Target: args[targetIndex],
		Inputs: []*uxv1alpha1.Input{input},
	}

	if len(args) > 2 {
		log.Debugf("choosing output file: %s", args[outputIndex])
		config.Output = &uxv1alpha1.RunConfig_Path{
			Path: args[outputIndex],
		}
	} else {
		log.Debug("choosing stdout")
		config.Output = &uxv1alpha1.RunConfig_Stdout{
			Stdout: true,
		}
	}

	return config, nil
}

func ParseTarget(config *uxv1alpha1.RunConfig) (tdl.Target, error) {
	return target.Parse(config.Target)
}

func ParseInputs(os tdl.OS, config *uxv1alpha1.RunConfig) ([]tdl.Input, error) {
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

func ParseOutput(os tdl.OS, config *uxv1alpha1.RunConfig) (tdl.Output, error) {
	switch {
	case config.GetPath() != "":
		return FsOutput(os.Fs(), config.GetPath()), nil
	case config.GetStdout():
		fallthrough
	default:
		return WriterOutput(os.Stdout()), nil
	}
}
