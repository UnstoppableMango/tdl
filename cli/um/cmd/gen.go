package cmd

import (
	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/cli/um/runner"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

func init() {
	rootCmd.AddCommand(genCmd)
}

var genCmd = cli.NewGenCmd(func(options uml.GeneratorOptions) uml.Generator {
	docker := runner.NewDocker(runner.FromGen(options))
	return &docker
})
