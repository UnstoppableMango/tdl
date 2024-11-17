package main

import (
	"io"
	"os"

	"github.com/unstoppablemango/tdl/internal/util"
)

func main() {
	spec, err := io.ReadAll(os.Stdin)
	if err != nil {
		util.Fail(err)
	}

	err = os.WriteFile("out", spec, os.ModePerm)
	if err != nil {
		util.Fail(err)
	}
}
