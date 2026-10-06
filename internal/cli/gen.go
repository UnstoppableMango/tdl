package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/plugin"
)

func newGenCmd() *cobra.Command {
	r := &genRun{}

	cmd := &cobra.Command{
		Use:   "gen <file>...",
		Short: "Generate code from a TDL file",
		Long: "Generate code from a TDL file.\n\n" +
			"Every target block in the file runs. A target block exists, so it\n" +
			"generates; --target narrows a run to one backend.\n\n" +
			"A target tdl has no backend for resolves to tdl-gen-<name> on\n" +
			"PATH. Both kinds speak the same protocol.\n\n" +
			"Where output goes comes from the block's own `out` directive,\n" +
			"relative to the file declaring the block, and -o, relative to the\n" +
			"working directory, overrides it for one invocation.\n\n" +
			"--verify generates and compares against disk without writing,\n" +
			"exiting non-zero when they differ. --clean first removes the files\n" +
			"an earlier run wrote, which .tdl-output in the directory lists.\n" +
			"A file there that tdl did not write is never overwritten or removed.\n\n" +
			"--watch regenerates when the file changes, holding open any\n" +
			"plugin that declared it can serve more than one request. It\n" +
			"takes a single file, since it does not return.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if r.watch && r.verify {
				return fmt.Errorf("--watch regenerates, so it cannot be combined with --verify")
			}
			if r.watch && len(args) > 1 {
				return fmt.Errorf("--watch takes a single file, got %d", len(args))
			}
			if r.verify && r.clean {
				return fmt.Errorf("--verify writes nothing, so it cannot be combined with --clean")
			}

			// Imports resolve next to the importing file, and stdin has no
			// directory.
			for _, path := range args {
				if isStdin(path) {
					return fmt.Errorf("gen needs a file on disk: an import resolves next to it, and %s has no directory", stdinName)
				}
			}

			r.cmd = cmd
			r.backends = map[string]plugin.Backend{}
			defer r.close()

			if !r.watch {
				r.cleaned = map[string]bool{}
				return eachFile(cmd, args, r.generate)
			}

			// Errors are reported and the watch continues.
			path := args[0]
			save := func() {
				// Each save is its own run for --clean.
				r.cleaned = map[string]bool{}
				if err := r.generate(path); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), err)
				}
			}
			save()
			fmt.Fprintf(cmd.ErrOrStderr(), "watching %s\n", path)
			gen.Watch(cmd.Context().Done(), path, save)
			return nil
		},
	}

	cmd.Flags().StringVar(&r.target, "target", "", "generate only this target")
	cmd.Flags().StringVarP(&r.out, "out", "o", "", "write here instead of the target block's out directive")
	cmd.Flags().BoolVar(&r.verify, "verify", false, "generate and compare against disk without writing")
	cmd.Flags().BoolVar(&r.clean, "clean", false, "remove the files tdl wrote before writing")
	cmd.Flags().BoolVar(&r.watch, "watch", false, "regenerate when the file changes")
	return cmd
}

// genRun is one invocation of gen and the state it keeps across files or
// saves.
type genRun struct {
	cmd    *cobra.Command
	target string
	out    string
	verify bool
	clean  bool
	watch  bool

	// cleaned holds the output directories --clean has emptied in this run.
	// Each is emptied once, since -o applies to every file given.
	cleaned map[string]bool

	// backends caches each resolved backend. Under --watch a plugin that
	// declared reuse is held open here across saves.
	backends map[string]plugin.Backend
}

// backend resolves the backend serving a target, once per name.
func (r *genRun) backend(ctx context.Context, name string) (plugin.Backend, error) {
	if b, ok := r.backends[name]; ok {
		return b, nil
	}

	b, err := gen.Resolve(name)
	if err != nil {
		return nil, fmt.Errorf("%w; compiled in: %v", err, gen.BuiltinNames())
	}
	if sub, ok := b.(*gen.Subprocess); ok && r.watch {
		session, err := gen.Open(ctx, sub)
		if err != nil {
			return nil, err
		}
		b = session
	}
	r.backends[name] = b
	return b, nil
}

func (r *genRun) close() {
	for _, b := range r.backends {
		if s, ok := b.(*gen.Session); ok {
			s.Close()
		}
	}
}

// generate runs every target block in one file.
func (r *genRun) generate(path string) error {
	cmd := r.cmd
	file, err := loadFile(cmd, path)
	if err != nil {
		return err
	}

	model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
	if len(diags) > 0 {
		return diags
	}

	targets, err := gen.Targets(model, r.out)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("%s declares no target blocks", path)
	}

	ran, stale := 0, 0
	for _, t := range targets {
		if r.target != "" && t.Name != r.target {
			continue
		}

		mode := gen.ModeWrite
		switch out := filepath.Clean(t.Out); {
		case r.verify:
			mode = gen.ModeVerify
		case r.clean && !r.cleaned[out]:
			mode = gen.ModeClean
			r.cleaned[out] = true
		}

		backend, err := r.backend(cmd.Context(), t.Name)
		if err != nil {
			return err
		}

		// Checked before running so a mistyped directive fails with a
		// position instead of after some files are written.
		problems := gen.CheckDirectives(t.Name, model, backend.Describe())
		reportDiagnostics(cmd, problems)
		if gen.Fatal(problems) {
			return fmt.Errorf("target %s uses a directive incorrectly", t.Name)
		}

		result, err := gen.Run(cmd.Context(), backend, t, model, mode)
		reportDiagnostics(cmd, result.Diagnostics)
		if err != nil {
			return err
		}
		for _, removed := range result.Removed {
			fmt.Fprintln(cmd.OutOrStdout(), "removed "+removed)
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
		return fmt.Errorf("no target named %s in %s", r.target, path)
	}
	if stale > 0 {
		return fmt.Errorf("%d file(s) would change; run without --verify to update them", stale)
	}
	return nil
}

// reportDiagnostics prints backend or directive diagnostics in the
// compiler's diagnostic format.
func reportDiagnostics(cmd *cobra.Command, diags []*plugin.Diagnostic) {
	for _, d := range diags {
		severity := "error"
		if d.GetSeverity() == plugin.Severity_SEVERITY_WARNING {
			severity = "warning"
		}
		pos := d.GetPosition()
		fmt.Fprintf(cmd.ErrOrStderr(), "%s:%d:%d: %s: %s\n",
			pos.GetFilename(), pos.GetLine(), pos.GetColumn(), severity, d.GetMessage())
	}
}
