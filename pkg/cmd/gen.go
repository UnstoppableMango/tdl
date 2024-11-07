package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/cmd/flags"
	pipeio "github.com/unstoppablemango/tdl/pkg/pipe"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	iosink "github.com/unstoppablemango/tdl/pkg/sink"
	"github.com/unstoppablemango/tdl/pkg/target"
)

func NewGen() *cobra.Command {
	var conformanceTest bool

	cmd := &cobra.Command{
		Use:   "gen [TARGET]",
		Short: "Run code generation for TARGET",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			target, err := target.Parse(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}

			log := log.With("target", target)
			log.Debug("searching for a plugin")
			plugin, err := plugin.FirstAvailable(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}

			gen, err := plugin.Generator(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}

			pipeline := pipeio.ReadSpec(gen)
			sink := iosink.NewSink(os.Stdout)

			log.Debug("executing pipeline")
			if err := pipeline.Execute(os.Stdin, sink); err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}
		},
	}

	flags.ConformanceTest(cmd.Flags(), &conformanceTest)

	return cmd
}
