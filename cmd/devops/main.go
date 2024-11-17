package main

import (
	"github.com/charmbracelet/log"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/cmd"
	"github.com/unstoppablemango/tdl/pkg/cmd/devops"
)

func main() {
	log.SetLevel(log.ErrorLevel)

	cmd := cmd.NewDevOps()
	cmd.AddCommand(
		devops.NewList(&devops.ListOptions{}),
	)

	if err := cmd.Execute(); err != nil {
		util.Fail(err)
	}
}
