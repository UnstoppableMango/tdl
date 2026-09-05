package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/plugin"
)

func newGenCmd() *cobra.Command {
	var (
		target string
		out    string
		verify bool
		clean  bool
		watch  bool
	)

	cmd := &cobra.Command{
		Use:   "gen <file>...",
		Short: "Generate code from a TDL file",
		Long: "Generate code from a TDL file.\n\n" +
			"Every target block in the file runs. A target block exists, so it\n" +
			"generates; --target narrows a run to one backend.\n\n" +
			"A target tdl has no backend for resolves to tdl-gen-<name> on\n" +
			"PATH. Both kinds speak the same protocol.\n\n" +
			"Where output goes comes from the block's own `out` directive, and\n" +
			"-o overrides it for one invocation.\n\n" +
			"--verify generates and compares against disk without writing,\n" +
			"exiting non-zero when they differ. --clean empties the output\n" +
			"directory first, and refuses one tdl did not write.\n\n" +
			"--watch regenerates when the file changes, holding open any\n" +
			"plugin that declared it can serve more than one request. It\n" +
			"takes a single file, since it does not return.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if watch && verify {
				return fmt.Errorf("--watch regenerates, so it cannot be combined with --verify")
			}
			if watch && len(args) > 1 {
				return fmt.Errorf("--watch takes a single file, got %d", len(args))
			}
			// An import resolves next to the file that wrote it, and
			// standard input has no directory: every import would quietly
			// resolve against the working directory instead. `ir` reads a
			// model and lives with that; gen writes files from it.
			for _, path := range args {
				if isStdin(path) {
					return fmt.Errorf("gen needs a file on disk: an import resolves next to it, and %s has no directory", stdinName)
				}
			}
			generate := func(path string) error {
				file, err := loadFile(cmd, path)
				if err != nil {
					return err
				}

				model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
				if len(diags) > 0 {
					return diags
				}

				targets, err := gen.Targets(model, out)
				if err != nil {
					return err
				}
				if len(targets) == 0 {
					return fmt.Errorf("%s declares no target blocks", path)
				}

				if verify && clean {
					return fmt.Errorf("--verify writes nothing, so it cannot be combined with --clean")
				}
				mode := gen.ModeWrite
				switch {
				case verify:
					mode = gen.ModeVerify
				case clean:
					mode = gen.ModeClean
				}

				ran, stale := 0, 0
				for _, t := range targets {
					if target != "" && t.Name != target {
						continue
					}

					backend, err := gen.Resolve(t.Name)
					if err != nil {
						return fmt.Errorf("%w; compiled in: %v", err, gen.BuiltinNames())
					}

					// What a backend understands is checked against the target
					// block before anything runs, so a mistyped directive fails
					// with a position rather than half way through writing files.
					failed := false
					for _, p := range gen.CheckDirectives(t.Name, model, backend.Describe()) {
						fmt.Fprintln(cmd.ErrOrStderr(), p)
						failed = failed || !p.Warning
					}
					if failed {
						return fmt.Errorf("target %s uses a directive incorrectly", t.Name)
					}

					result, err := gen.Run(cmd.Context(), backend, t, model, mode)
					reportDiagnostics(cmd, result)
					if err != nil {
						return err
					}
					for _, r := range result.Removed {
						fmt.Fprintln(cmd.OutOrStdout(), "removed "+r)
					}
					for _, w := range result.Written {
						fmt.Fprintln(cmd.OutOrStdout(), w)
					}
					for _, s := range result.Stale {
						fmt.Fprintf(cmd.ErrOrStderr(), "%s: %s\n", s.Path, s.Reason)
					}
					stale += len(result.Stale)
					ran++
				}

				if ran == 0 {
					return fmt.Errorf("no target named %s in %s", target, path)
				}
				if stale > 0 {
					return fmt.Errorf("%d file(s) would change; run without --verify to update them", stale)
				}
				return nil
			}

			if !watch {
				return eachFile(cmd, args, func(_ int, path string) error {
					return generate(path)
				})
			}

			path := args[0]

			// The first run reports its errors and the watch continues: a
			// file being edited is expected to be broken between saves.
			if err := generate(path); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "watching %s\n", path)

			gen.Watch(cmd.Context().Done(), path, func() {
				if err := generate(path); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), err)
				}
			})
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "generate only this target")
	cmd.Flags().StringVarP(&out, "out", "o", "", "write here instead of the target block's out directive")
	cmd.Flags().BoolVar(&verify, "verify", false, "generate and compare against disk without writing")
	cmd.Flags().BoolVar(&clean, "clean", false, "empty the output directory before writing")
	cmd.Flags().BoolVar(&watch, "watch", false, "regenerate when the file changes")
	return cmd
}

// severity names a diagnostic's level for output.
func severity(d *plugin.Diagnostic) string {
	if d.GetSeverity() == plugin.Severity_SEVERITY_WARNING {
		return "warning"
	}
	return "error"
}

// reportDiagnostics prints what a backend said, in the same shape as the
// compiler's own diagnostics.
func reportDiagnostics(cmd *cobra.Command, result gen.Result) {
	for _, d := range result.Diagnostics {
		pos := d.GetPosition()
		fmt.Fprintf(cmd.ErrOrStderr(), "%s:%d:%d: %s: %s\n",
			pos.GetFilename(), pos.GetLine(), pos.GetColumn(),
			severity(d), d.GetMessage())
	}
}
