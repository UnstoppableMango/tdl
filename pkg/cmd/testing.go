package cmd

import (
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/cmd/testing"
)

func NewTesting() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "testing",
		Short: "Commands for ux testing",
	}
	cmd.AddCommand(
		testing.NewConform(),
		testing.NewDiscover(),
	)

	return cmd
}
