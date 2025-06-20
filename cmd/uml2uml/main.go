package main

import (
	"io"
	"os"

	"github.com/unmango/go/cli"
)

// This is primarily used for testing

func main() {
	spec, err := io.ReadAll(os.Stdin)
	if err != nil {
		cli.Fail(err)
	}

	if len(os.Args) > 1 {
		err = os.WriteFile(os.Args[1], spec, os.ModePerm)
	} else {
		_, err = os.Stdout.Write(spec)
	}
	if err != nil {
		cli.Fail(err)
	}
}
