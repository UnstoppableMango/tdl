package gen

import (
	"os/exec"

	"github.com/unmango/go/option"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/sink"
	"github.com/unstoppablemango/tdl/pkg/spec"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Cli struct {
	name string
	args []string
}

type CliOption func(*Cli)

// Execute implements tdl.Generator.
func (c *Cli) Execute(s *tdlv1alpha1.Spec, si tdl.Sink) error {
	cmd := exec.Command(c.name, c.args...)
	cmd.Stdin = spec.NewReader(s)
	cmd.Stdout = sink.NewWriter(si)

	return cmd.Run()
}

func (c *Cli) String() string {
	return c.name
}

var _ tdl.SinkGenerator = &Cli{}

func NewCli(name string, options ...CliOption) *Cli {
	gen := &Cli{name: name}
	option.ApplyAll(gen, options)

	return gen
}

func WithCliArgs(args ...string) CliOption {
	return func(c *Cli) {
		c.args = args
	}
}
