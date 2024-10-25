package devops

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

func NewList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List source files in the repo",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()

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
					if strings.Contains(path, "node_modules") {
						return filepath.SkipDir
					}

					return nil
				},
			)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}

		},
	}
}
