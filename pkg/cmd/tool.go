package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/unmango/go/fs/ignore"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/cmd/internal"
	"github.com/unstoppablemango/tdl/pkg/logging"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/target"
	"github.com/unstoppablemango/tdl/pkg/tool"
)

var DefaultIgnorePatterns = tool.DefaultIgnorePatterns

func NewTool() *cobra.Command {
	var cwd string

	cmd := &cobra.Command{
		Use:   "tool [NAME]",
		Short: "Execute a tool",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			logging.Init()
			t, err := target.Parse(args[0])
			if err != nil {
				util.Fail(err)
			}

			ctx := cmd.Context()
			tool, err := target.Tool(ctx, t, plugin.Static())
			if err != nil {
				util.Fail(err)
			}

			if cwd, err = internal.Cwd(cwd); err != nil {
				util.Fail(err)
			}

			src := afero.NewBasePathFs(afero.NewOsFs(), cwd)
			if i, err := internal.OpenGitIgnore(ctx); err != nil {
				log.Info("not a git repo", "err", err)
				src = ignore.NewFsFromGitIgnoreLines(src, DefaultIgnorePatterns...)
			} else if src, err = ignore.NewFsFromGitIgnoreReader(src, i); err != nil {
				util.Fail(err)
			}

			extraArgs := []string{}
			if l := cmd.Flags().ArgsLenAtDash(); l > 0 {
				extraArgs = args[l:]
			}

			log.Debug("executing", "tool", tool, "cwd", cwd, "args", extraArgs)
			out, err := tool.Execute(ctx, src, extraArgs)
			if err != nil {
				util.Fail(err)
			}

			fmt.Println("successfully executed")
			if err = internal.PrintFs(out); err != nil {
				util.Fail(err)
			}
		},
	}

	cmd.Flags().StringVarP(&cwd, "cwd", "C", "", "sets the working directory")
	_ = cmd.MarkFlagDirname("cwd")

	return cmd
}
