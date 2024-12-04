package plugin

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewCache() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Cache known plugins",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stderr, "Unimplemented!")
		},
	}

	return cmd
}
