package conform

import (
	"fmt"
	"os"
)

type T struct{}

func (t *T) Fail() {
	fmt.Fprintln(os.Stderr, "fail")
	os.Exit(1)
}
