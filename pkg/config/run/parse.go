package run

import (
	"errors"

	uxv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/ux/v1alpha1"
)

func ParseArgs(args []string) (*uxv1alpha1.RunConfig, error) {
	if len(args) == 0 {
		return nil, errors.New("not enough arguments")
	}

	var inputs []*uxv1alpha1.Input
	if input, err := parseInput(args); err != nil {
		return nil, err
	} else {
		inputs = []*uxv1alpha1.Input{input}
	}

	config := &uxv1alpha1.RunConfig{
		Inputs: inputs,
	}

	if len(args) > 1 {
		config.Output = &uxv1alpha1.RunConfig_Path{
			Path: args[1],
		}
	} else {
		config.Output = &uxv1alpha1.RunConfig_Stdout{
			Stdout: true,
		}
	}

	return config, nil
}

func parseInput(args []string) (*uxv1alpha1.Input, error) {
	if len(args) < 1 {
		return nil, errors.New("no input")
	}

	path := args[0]
	input := &uxv1alpha1.Input{}
	if path == "-" {
		input.Value = &uxv1alpha1.Input_Stdin{
			Stdin: true,
		}
	} else {
		input.Value = &uxv1alpha1.Input_File{
			File: &uxv1alpha1.FileInput{
				Path: path,
			},
		}
	}

	return input, nil
}
