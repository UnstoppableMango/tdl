package cmd

import "github.com/spf13/cobra"

var fromCmd = &cobra.Command{
	Use: "from",
}

func init() {
	rootCmd.AddCommand(fromCmd)
}
