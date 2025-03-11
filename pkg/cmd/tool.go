package cmd

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/unmango/aferox/github"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/cmd/internal"
	"github.com/unstoppablemango/tdl/pkg/logging"
	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/target"
)

func NewTool() *cobra.Command {
	var (
		cwd    string
		output string
	)

	cmd := &cobra.Command{
		Use:   "tool [NAME]",
		Short: "Execute a tool",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			logging.Init()
			args, extraArgs := internal.SplitAt(args, cmd.ArgsLenAtDash())
			if len(args) <= 1 {
				util.Fail(errors.New("no input specified"))
			}

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

			src, err := internal.CwdFs(ctx, cwd)
			if err != nil {
				util.Fail(err)
			}

			paths, err := resolveGh(args[1:])
			if err != nil {
				util.Fail(err)
			}

			paths, err = makeRel(paths, cwd)
			if err != nil {
				util.Fail(err)
			}

			toolArgs := append(paths, extraArgs...)
			log.Debug("executing", "tool", tool, "cwd", cwd, "args", toolArgs)
			out, err := tool.Execute(ctx, src, toolArgs)
			if err != nil {
				util.Fail(err)
			}

			if output != "" {
				err = internal.CopyOutput(out, output)
			} else {
				err = internal.PrintFs(out)
			}
			if err != nil {
				util.Fail(err)
			}
		},
	}

	cmd.Flags().StringVarP(&cwd, "cwd", "C", "", "sets the working directory")
	_ = cmd.MarkFlagDirname("cwd")
	cmd.Flags().StringVarP(&output, "output", "o", "", "the directory to write generated code")
	_ = cmd.MarkFlagDirname("output")

	return cmd
}

func makeRel(ps []string, wd string) (paths []string, err error) {
	for _, p := range ps {
		if filepath.IsAbs(p) {
			if p, err = filepath.Rel(wd, p); err != nil {
				return nil, err
			}
		}

		paths = append(paths, p)
	}

	return
}

func resolveGh(paths []string) (local []string, err error) {
	fs := github.NewFs(github.NewClient(nil))
	for _, p := range paths {
		if !strings.HasPrefix(p, "github.com") {
			local = append(local, p)
			continue
		}

		f, err := fs.Open(p)
		if err != nil {
			return nil, err
		}

		t, err := os.CreateTemp("", "")
		if err != nil {
			return nil, err
		}

		log.Debug("copying", "github", f.Name(), "local", t.Name())
		if _, err = io.Copy(t, f); err != nil {
			return nil, err
		}

		local = append(local, t.Name())
	}

	return
}
