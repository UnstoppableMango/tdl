package cmd

import (
	"os"
	"path"

	"github.com/adrg/xdg"
	"github.com/google/go-github/v63/github"
	"github.com/spf13/cobra"
	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/pkg/logging"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

var (
	AppName   = "um"
	ConfigDir = path.Join(xdg.ConfigHome, AppName)
	PluginDir = path.Join(ConfigDir, "plugins")
	plugins   plugin.PluginCache
)

var rootCmd = &cobra.Command{
	Use:   "um",
	Short: "UnstoppableMango's Type Description Language CLI",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log := logging.NewLogger()
		cli.SetLogger(cmd, log)

		gh := github.NewClient(nil)
		plugins = plugin.NewCache(gh, PluginDir, log)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
