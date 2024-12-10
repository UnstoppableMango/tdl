package cmd

import (
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/cmd/plugin"
)

func NewPlugin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Commands for working with plugins",
	}
	cmd.AddCommand(
		plugin.NewPull(),
		plugin.NewList(),
	)

	return cmd
}
