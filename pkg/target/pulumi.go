package target

import (
	"context"
	"errors"
	"strings"

	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin"
)

type pulumi string

// Choose implements tdl.Target.
func (pulumi) Choose([]tdl.SinkGenerator) (tdl.SinkGenerator, error) {
	panic("unimplemented")
}

// Generator implements tdl.Target.
func (t pulumi) Generator(available iter.Seq[tdl.Plugin]) (tdl.Generator, error) {
	for p := range available {
		for _, pi := range plugin.Unwrap(p) {
			if strings.Contains(p.String(), "crd2pulumi") {
				return pi.Generator(context.TODO(), t)
			}
		}
	}

	return nil, errors.New("no supported plugins")
}

// String implements tdl.Target.
func (t pulumi) String() string {
	return string(t)
}

var Pulumi pulumi = "Pulumi"

var _ tdl.Target = Pulumi
