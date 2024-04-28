package cmd

import (
	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/pkg/pcl"
)

func init() {
	rootCmd.AddCommand(fromCmd)
}

var fromCmd = cli.NewFromCmd(pcl.Converter)
