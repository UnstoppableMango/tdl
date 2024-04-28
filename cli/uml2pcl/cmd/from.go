package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/pcl"
)

func init() {
	rootCmd.AddCommand(fromCmd)
}

var fromCmd = &cobra.Command{
	Use:  "from",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pcl, err := pcl.From("")
		fmt.Printf("output:\n%s\n", pcl)
		return err
	},
}
