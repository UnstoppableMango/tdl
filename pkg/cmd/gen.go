package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
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
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			log := log.With("target", target)

			log.Debug("searching for a plugin")
			plugin, err := plugin.FirstAvailable(target)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			log.Debug("searching for a generator")
			generator, err := plugin.Generator(target)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			fsys := afero.NewOsFs()
			input, err := input.ParseArgs(fsys, args[1:])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			output, err := output.ParseArgs(fsys, args[1:])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			log.Debug("creating pipeline")
			pipeline := spec.PipeInput(generator.Execute)

			log.Debug("executing pipeline")
			if err := pipeline(ctx, input, output); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}

	_ = flags.ConformanceTest(cmd.Flags(), &conformanceTest)

	return cmd
}
