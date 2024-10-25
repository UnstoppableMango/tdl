package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/pkg/cmd"
	"github.com/unstoppablemango/tdl/pkg/cmd/devops"
)

func main() {
	cmd := cmd.NewDevOps()
	cmd.AddCommand(
		devops.NewList(),
	)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
