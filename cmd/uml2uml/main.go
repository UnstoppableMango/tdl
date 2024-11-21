package main

import (
	"io"
	"os"

	"github.com/unstoppablemango/tdl/internal/util"
)

// This is primarily used for testing

func main() {
	spec, err := io.ReadAll(os.Stdin)
	if err != nil {
		util.Fail(err)
	}

	if len(os.Args) > 0 {
		err = os.WriteFile(os.Args[0], spec, os.ModePerm)
	} else {
		_, err = os.Stdout.Write(spec)
	}
	if err != nil {
		util.Fail(err)
	}
}
