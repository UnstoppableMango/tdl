package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var fromCmd = &cobra.Command{
	Use: "from",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not implemented")
	},
}

func init() {
	rootCmd.AddCommand(fromCmd)
}
