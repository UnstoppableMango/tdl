package cmd

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/unmango/go/fs/ignore"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/cmd/internal"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/target"
	"github.com/unstoppablemango/tdl/pkg/tool"
)

var DefaultIgnorePatterns = tool.DefaultIgnorePatterns

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

			src := afero.NewBasePathFs(afero.NewOsFs(), work)
			if i, err := internal.OpenGitIgnore(ctx); err != nil {
				log.Info("not a git repo", "err", err)
				src = ignore.NewFsFromGitIgnoreLines(src, DefaultIgnorePatterns...)
			} else if src, err = ignore.NewFsFromGitIgnoreReader(src, i); err != nil {
				util.Fail(err)
			}

			out, err := tool.Execute(ctx, src)
			if err != nil {
				util.Fail(err)
			}

			if err = internal.PrintFs(out); err != nil {
				util.Fail(err)
			}
		},
	}
}
