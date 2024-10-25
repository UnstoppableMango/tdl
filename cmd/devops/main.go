package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
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
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
