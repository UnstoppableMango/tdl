package cmd

import "github.com/spf13/cobra"

var toCmd = &cobra.Command{
	Use: "to",
}

func init() {
	rootCmd.AddCommand(toCmd)
}
