package cmd

import (
	"errors"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/cmd/flags"
	"github.com/unstoppablemango/tdl/pkg/config/run"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/spec"
	"github.com/unstoppablemango/tdl/pkg/target"
)

func NewGen() *cobra.Command {
	var conformanceTest bool

	cmd := &cobra.Command{
		Use:   "gen TARGET [INPUT] [OUTPUT]",
		Short: "Run code generation for TARGET",
		Args:  cobra.RangeArgs(1, 3),
		Run: func(cmd *cobra.Command, args []string) {
			// log.SetLevel(log.DebugLevel)
			ctx := cmd.Context()
			config, err := run.ParseArgs(args)
			if err != nil {
				util.Fail(err)
			}

			log.Debug("parsing target")
			target, err := target.Parse(config.Target)
			if err != nil {
				util.Fail(err)
			}

			log.Debug("searching for a plugin")
			plugin, err := plugin.FirstAvailable(target)
			if err != nil {
				util.Fail(err)
			}

			log.Debug("searching for a generator")
			generator, err := plugin.Generator(ctx, target)
			if err != nil {
				util.Fail(err)
			}

			log.Debug("parsing inputs")
			os := internal.RealOs()
			inputs, err := run.ParseInputs(os, config)
			if err != nil {
				util.Fail(err)
			}
			if len(inputs) != 1 {
				util.Fail(errors.New("only a single input may be provided"))
			}

			log.Debugf("reading spec: %s", inputs[0])
			spec, err := spec.ReadInput(inputs[0])
			if err != nil {
				util.Fail(err)
			}

			log.Debugf("executing generator: %s", generator)
			fs, err := generator.Execute(ctx, spec)
			if err != nil {
				util.Fail(err)
			}

			log.Debug("parsing output")
			output, err := run.ParseOutput(os, config)
			if err != nil {
				util.Fail(err)
			}

			log.Debugf("writing output: %s %s", fs.Name(), output)
			if err = output.Write(fs); err != nil {
				util.Fail(err)
			}
		},
	}

	_ = flags.ConformanceTest(cmd.Flags(), &conformanceTest)

	return cmd
}
