package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/cmd/flags"
	"github.com/unstoppablemango/tdl/pkg/gen/lookup"
	genio "github.com/unstoppablemango/tdl/pkg/pipe/io"
	"github.com/unstoppablemango/tdl/pkg/tdl/spec"
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

			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
			}

			spec, err := spec.FromProto(data)
			sink := genio.NewSink(os.Stdout)

			log.Debug("executing pipeline")
			if err := gen.Execute(spec, sink); err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}
		},
	}

	flags.ConformanceTest(cmd.Flags(), &conformanceTest)

	return cmd
}
