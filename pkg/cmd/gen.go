package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/cmd/flags"
	"github.com/unstoppablemango/tdl/pkg/gen/io"
)

func NewGen(pipeline io.PipelineFunc) *cobra.Command {
	var conformanceTest bool

	cmd := &cobra.Command{
		Use:  "gen",
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			log.Debug("executing pipeline")
			if err := pipeline(os.Stdin, os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}
		},
	}

	flags.ConformanceTest(cmd.Flags(), &conformanceTest)

	return cmd
}

func NewGenFor(lookup io.LookupFunc) *cobra.Command {
	var conformanceTest bool

	cmd := &cobra.Command{
		Use:   "gen [TARGET]",
		Short: "Run code generation for TARGET",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			log.Debug("lookup up pipeline", "target", args[0])
			pipeline, err := lookup(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}

			log.Debug("executing pipeline", "target", args[0])
			if err := pipeline(os.Stdin, os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}
		},
	}

	flags.ConformanceTest(cmd.Flags(), &conformanceTest)

	return cmd
}
