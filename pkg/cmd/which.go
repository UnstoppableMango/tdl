package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/target"
)

func NewWhich() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "which [TOKENISH]",
		Short: "Print the token for the closest matching generator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			t, err := target.Parse(args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
			}

			fmt.Println(generator)
		},
	}

	return cmd
}
