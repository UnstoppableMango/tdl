package cli

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

func newGenCmd() *cobra.Command {
	var (
		target string
		out    string
		verify bool
		clean  bool
	)

	cmd := &cobra.Command{
		Use:   "gen <file>",
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
			"directory first, and refuses one tdl did not write.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			file, err := parser.Parse(path, bytes.NewReader(data))
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
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "generate only this target")
	cmd.Flags().StringVarP(&out, "out", "o", "", "write here instead of the target block's out directive")
	cmd.Flags().BoolVar(&verify, "verify", false, "generate and compare against disk without writing")
	cmd.Flags().BoolVar(&clean, "clean", false, "empty the output directory before writing")
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
