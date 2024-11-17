package util

import (
	"fmt"
	"os"
)

func Fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
