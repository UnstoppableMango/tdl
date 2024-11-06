package cli

import (
	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/pipe"
	"github.com/unstoppablemango/tdl/pkg/pipe/io"
)

func NewPipeline(name string, args ...string) pipe.IO {
	gen := gen.NewCli(name, gen.WithCliArgs(args...))
	input := io.ReadSpec(gen)

	return io.WriteSink(input)
}
