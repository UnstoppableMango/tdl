package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen"
)

func NewWhich() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "which [TOKENISH]",
		Short: "Print the token for the closest matching generator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			token := tdl.Token{Name: args[0]}

			generator, err := gen.Name(token)
			if !errors.Is(err, gen.ErrNotFound) {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}

			generator, err = gen.FromPath(token)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}

			fmt.Println(generator)
		},
	}

	return cmd
}
