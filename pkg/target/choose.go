package target

import (
	"fmt"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

type RejectionErr struct {
	Generator tdl.Generator
	Reason    string
}

func (e RejectionErr) Error() string {
	return fmt.Sprintf("rejected %s: %s", e.Generator, e.Reason)
}

func Reject(generator tdl.Generator, reason string) error {
	return &RejectionErr{generator, reason}
}