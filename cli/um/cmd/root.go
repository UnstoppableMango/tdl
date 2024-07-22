package cmd

import (
	"os"
	"path"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/pkg/logging"
)

var (
	AppName   = "um"
	ConfigDir = path.Join(xdg.ConfigHome, AppName)
	PluginDir = path.Join(ConfigDir, "plugins")
)

var rootCmd = &cobra.Command{
	Use:   "um",
	Short: "UnstoppableMango's Type Description Language CLI",
	PreRun: func(cmd *cobra.Command, args []string) {
		cli.SetLogger(cmd, logging.NewLogger())
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
