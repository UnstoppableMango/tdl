package cmd

import "github.com/spf13/cobra"

func NewDevOps() *cobra.Command {
	return &cobra.Command{
		Use:   "devops",
		Short: "Workflow commands for developing in TDL",
	}
}
