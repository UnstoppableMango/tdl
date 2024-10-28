package devops

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var Blacklist = []string{
	"node_modules",
	"bin", "obj",
	"pcl", "uml",
}

type ListOptions struct {
	Absolute bool
	Go       bool
	Proto    bool
}

func NewList(options *ListOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List source files in the repo",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			log.Debug("running with options", "options", options)

			log.Debug("looking up repo root")
			revParse, err := exec.CommandContext(ctx,
				"git", "rev-parse", "--show-toplevel",
			).CombinedOutput()
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}

			root := strings.TrimSpace(string(revParse))
			log.Debugf("walking root: %s", root)

			err = filepath.WalkDir(root,
				func(path string, d fs.DirEntry, err error) error {
					if blacklisted(path) {
						return filepath.SkipDir
					}
					if options.Go && !strings.HasSuffix(path, ".go") {
						return nil
					}
					if options.Proto && !strings.HasSuffix(path, ".proto") {
						return nil
					}

					rel, err := filepath.Rel(root, path)
					if err != nil {
						return err
					}

					_, err = fmt.Println(rel)
					return err
				},
			)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolVar(&options.Absolute, "absolute", false, "Print fully qualified paths rather than paths relative to the git root")
	cmd.Flags().BoolVar(&options.Go, "go", false, "List go sources")
	cmd.Flags().BoolVar(&options.Proto, "proto", false, "List protobuf sources")

	return cmd
}

func blacklisted(path string) bool {
	return slices.ContainsFunc(Blacklist, func(b string) bool {
		return strings.Contains(path, b)
	})
}
