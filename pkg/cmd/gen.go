package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/cmd/flags"
	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/sink"
	"github.com/unstoppablemango/tdl/pkg/spec"
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
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			log := log.With("target", target)

			log.Debug("searching for a plugin")
			plugin, err := plugin.FirstAvailable(target)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			log.Debug("searching for a generator")
			generator, err := plugin.Generator(target)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			var (
				input io.Reader
				media tdl.MediaType
			)

			fsys := afero.NewOsFs()
			if len(args) == 1 {
				log.Debug("choosing stdin")
				input = os.Stdin
				media = mediatype.ApplicationProtobuf
			} else {
				log.Debug("choosing input file")
				path := args[1]
				media, err = mediatype.Guess(path)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}

				log.Debug("opening input", "path", path)
				input, err = fsys.Open(path)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			}

			log.Debug("creating pipeline")
			pipeline := mediatype.PipeRead[gen.FromReader](
				generator.Execute,
				media,
				spec.Zero,
			)
			sink := sink.WriteTo(os.Stdout)

			log.Debug("executing pipeline")
			if err := pipeline(input, sink); err != nil {
				fmt.Fprintln(os.Stderr, err)
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
