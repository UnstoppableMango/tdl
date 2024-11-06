package plugin

import (
	"errors"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

func FirstAvailable(target tdl.Target) (tdl.Plugin, error) {
	for p := range target.Plugins() {
		return p, nil
	}

	return nil, errors.New("no plugins available")
}
