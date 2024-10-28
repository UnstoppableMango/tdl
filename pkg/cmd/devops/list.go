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
	"pcl",
	".tdl-old",
	".uml2ts-old",
	"testdata",
	".idea",
	".vscode",
	".git",
}

type ListOptions struct {
	Absolute   bool
	Go         bool
	Proto      bool
	Typescript bool
}

type printer struct {
	Opts    *ListOptions
	Sources []string
	Root    string
}

func (o *ListOptions) sources() []string {
	sources := []string{}
	if o.Go {
		sources = append(sources, ".go")
	}
	if o.Proto {
		sources = append(sources, ".proto")
	}
	if o.Typescript {
		sources = append(sources, ".ts")
	}

	return sources
}

func (o *ListOptions) printer(root string) *printer {
	return &printer{
		Opts:    o,
		Sources: o.sources(),
		Root:    root,
	}
}

func (p *printer) shouldPrint(path string) bool {
	if len(p.Sources) == 0 {
		return true
	}

	return slices.Contains(p.Sources, filepath.Ext(path))
}

func (p *printer) handle(path string) error {
	if !p.shouldPrint(path) {
		return nil
	}

	rel, err := filepath.Rel(p.Root, path)
	if err != nil {
		return err
	}

	_, err = fmt.Println(rel)
	return err
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

			printer := options.printer(root)
			err = filepath.WalkDir(root,
				func(path string, d fs.DirEntry, err error) error {
					if d.IsDir() {
						if blacklisted(path) {
							return filepath.SkipDir
						}

						return nil
					}

					return printer.handle(path)
				},
			)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolVar(&options.Absolute, "absolute", false, "Print fully qualified paths rather than paths relative to the git root")
	cmd.Flags().BoolVar(&options.Go, "go", false, "List Go sources")
	cmd.Flags().BoolVar(&options.Typescript, "ts", false, "List TypeScript sources")
	cmd.Flags().BoolVar(&options.Proto, "proto", false, "List protobuf sources")

	return cmd
}

func blacklisted(path string) bool {
	return slices.ContainsFunc(Blacklist, func(b string) bool {
		return strings.Contains(path, b)
	})
}
