package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/cmd/flags"
	pipeio "github.com/unstoppablemango/tdl/pkg/pipe"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	iosink "github.com/unstoppablemango/tdl/pkg/sink"
	"github.com/unstoppablemango/tdl/pkg/target"
)

func NewGen() *cobra.Command {
	var conformanceTest bool

	cmd := &cobra.Command{
		Use:   "gen [TARGET] [INPUT]",
		Short: "Run code generation for TARGET",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			target, err := target.Parse(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}
			log := log.With("target", target)

			fsys := afero.NewOsFs()
			files, err := openInput(fsys, args[1:])
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}

			log.Debug("searching for a plugin")
			plugin, err := plugin.FirstAvailable(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}

			log.Debug("searching for a generator")
			gen, err := plugin.Generator(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}

			pipeline := pipeio.ReadSpec(gen)
			sink := iosink.WriteTo(os.Stdout)

			var input io.Reader
			if len(files) == 0 {
				log.Debug("using stdin")
				input = os.Stdin
			} else {
				log.Debug("using input files")
				input = files[0]
			}

			log.Debug("executing pipeline")
			if err := pipeline.Execute(input, sink); err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}
		},
	}

	flags.ConformanceTest(cmd.Flags(), &conformanceTest)

	return cmd
}

func openInput(fsys afero.Fs, paths []string) (files []afero.File, err error) {
	var file afero.File
	for _, path := range paths {
		if file, err = fsys.Open(path); err != nil {
			return nil, err
		} else {
			files = append(files, file)
		}
	}

	return
}
