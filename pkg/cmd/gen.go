package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/cmd/flags"
	"github.com/unstoppablemango/tdl/pkg/gen/lookup"
	pipeio "github.com/unstoppablemango/tdl/pkg/pipe/io"
	iosink "github.com/unstoppablemango/tdl/pkg/sink/io"
)

func NewGen() *cobra.Command {
	var conformanceTest bool

	cmd := &cobra.Command{
		Use:   "gen [TARGET]",
		Short: "Run code generation for TARGET",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			target := args[0]
			log := log.With("target", target)

			log.Debug("lookup up generator")
			gen, err := lookup.Lookup(target)
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
