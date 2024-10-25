package main

import (
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/pkg/cmd"
)

func main() {
	if err := cmd.NewUx().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
