// Package cli wires up the tdl command-line interface.
package cli

import "github.com/spf13/cobra"

// Execute builds and runs the tdl root command.
func Execute() error {
	root := &cobra.Command{
		Use:           "tdl",
		Short:         "tdl compiles Type Description Language source into other structured formats",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newAstCmd())
	root.AddCommand(newCheckCmd())
	root.AddCommand(newFmtCmd())
	root.AddCommand(newPlayCmd())
	root.AddCommand(newTokensCmd())
	root.AddCommand(newVersionCmd())
	return root.Execute()
}
