package cli

import "github.com/spf13/cobra"

func InputFile(cmd *cobra.Command, file *string, usage string) {
	cmd.PersistentFlags().StringVarP(file, "input", "i", "", usage)
}
