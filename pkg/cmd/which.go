package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/gen/lookup"
	"github.com/unstoppablemango/tdl/pkg/tdl"
)

func NewWhich() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "which [TOKENISH]",
		Short: "Print the token for the closest matching generator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			token := tdl.Token{Name: args[0]}

			generator, err := lookup.Name(token)
			if !errors.Is(err, lookup.ErrNotFound) {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}

			generator, err = lookup.FromPath(token)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}

			fmt.Println(generator)
		},
	}

	return cmd
}