package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/gen/lookup"
)

func NewWhich() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "which [TOKENISH]",
		Short: "Print the token for the closest matching generator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := lookup.Execute(args[0], os.Stdout); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		},
	}

	return cmd
}
