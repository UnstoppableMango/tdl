package cmd

import "github.com/spf13/cobra"

func NewWhich() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "which [TOKENISH]",
		Short: "Print the token for the closest matching generator",
		Args:  cobra.ExactArgs(1),
	}

	return cmd
}
