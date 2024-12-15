package cmd

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/target"
)

func NewTool() *cobra.Command {
	return &cobra.Command{
		Use:   "tool [NAME]",
		Short: "Execute a tool",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			t, err := target.Parse(args[0])
			if err != nil {
				util.Fail(err)
			}

			ctx := cmd.Context()
			tool, err := target.Tool(ctx, t, plugin.Static())
			if err != nil {
				util.Fail(err)
			}

			work, err := os.Getwd()
			if err != nil {
				util.Fail(err)
			}

			fsys := afero.NewBasePathFs(afero.NewOsFs(), work)
			out, err := tool.Execute(ctx, fsys)
			if err != nil {
				util.Fail(err)
			}

			err = afero.Walk(out, "",
				func(path string, info fs.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if path == "" {
						return nil
					}

					fmt.Println(path)
					return nil
				},
			)
			if err != nil {
				util.Fail(err)
			}
		},
	}
}
