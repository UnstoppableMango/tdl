package target

import (
	"fmt"

	"github.com/unmango/go/iter"
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

func Choose[T tdl.Plugin](target tdl.Target, available iter.Seq[tdl.Plugin]) (res T, err error) {
	plugin, err := target.Choose(available)
	if err != nil {
		return
	}

	var ok bool
	if res, ok = plugin.(T); ok {
		return
	} else {
		return res, fmt.Errorf("invalid type for: %s", plugin)
	}
}
