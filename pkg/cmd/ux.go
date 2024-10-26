package cmd

import (
	"github.com/spf13/cobra"
)

func NewUx() *cobra.Command {
	return &cobra.Command{
		Use:   "ux",
		Short: "Generate types for things in different languages",
	}
}
