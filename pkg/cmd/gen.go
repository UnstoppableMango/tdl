package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/cmd/flags"
	"github.com/unstoppablemango/tdl/pkg/gen/input"
	"github.com/unstoppablemango/tdl/pkg/gen/output"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/spec"
	"github.com/unstoppablemango/tdl/pkg/target"
)

func NewGen() *cobra.Command {
	var conformanceTest bool

	cmd := &cobra.Command{
		Use:   "gen TARGET [INPUT] [OUTPUT]",
		Short: "Run code generation for TARGET",
		Args:  cobra.RangeArgs(1, 3),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			target, err := target.Parse(args[0])
			if err != nil {
				util.Fail(err)
			}
			log := log.With("target", target)

			log.Debug("searching for a plugin")
			plugin, err := plugin.FirstAvailable(target)
			if err != nil {
				util.Fail(err)
			}

			log.Debug("searching for a generator")
			generator, err := plugin.Generator(target)
			if err != nil {
				util.Fail(err)
			}

			fsys := afero.NewOsFs()
			input, err := input.ParseArgs(fsys, args[1:])
			if err != nil {
				util.Fail(err)
			}

			output, err := output.ParseArgs(fsys, args[1:])
			if err != nil {
				util.Fail(err)
			}

			log.Debug("creating pipeline")
			pipeline := spec.PipeInput(generator.Execute)

			log.Debug("executing pipeline")
			if err := pipeline(ctx, input, output); err != nil {
				util.Fail(err)
			}
		},
	}

	_ = flags.ConformanceTest(cmd.Flags(), &conformanceTest)

	return cmd
}
