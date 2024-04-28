package cmd

import (
	cli "github.com/unstoppablemango/tdl/cli/internal"
	"github.com/unstoppablemango/tdl/pkg/pcl"
)

func init() {
	rootCmd.AddCommand(toCmd)
}

var toCmd = cli.NewToCmd(pcl.Converter)
