package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/config"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

func newRequestCmd() *cobra.Command {
	return requestCmd(gen.Resolve)
}

// requestCmd builds the command over a resolver, as genCmd does.
func requestCmd(resolve func(string) (plugin.Backend, error)) *cobra.Command {
	r := &requestRun{resolve: resolve}

	cmd := &cobra.Command{
		Use:   "request --target <name> [--overlay] [file]...",
		Short: "Write the requests gen would send a backend",
		Long: "Write the requests gen would send a backend, without running it.\n\n" +
			"It writes two framed messages to standard output, as the plugin\n" +
			"protocol frames them: a Handshake, then one RequestSet holding a\n" +
			"request per target block for each file that lowers, and every\n" +
			"diagnostic. A file with an error yields no request. A host that runs\n" +
			"the backend itself, such as the C# source generator, reads them.\n\n" +
			"With --overlay, standard input holds one framed Overlay: file texts\n" +
			"read in place of the disk, such as an editor's unsaved buffers. Each\n" +
			"file it holds is requested as if it were given.\n\n" +
			"A warning naming a loss code is dropped as gen drops it: by the\n" +
			"[lossy] table of the nearest tdl.toml, or by --allow-lossy.\n\n" +
			"It writes nothing to disk.",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, path := range args {
				if isStdin(path) {
					return fmt.Errorf("request needs a file on disk or in the overlay: an import resolves next to it, and %s has no directory", stdinName)
				}
			}

			r.texts = map[string]string{}
			paths := slices.Clone(args)
			if r.overlay {
				var overlay plugin.Overlay
				if err := plugin.NewConn(cmd.InOrStdin(), nil).Recv(&overlay); err != nil {
					return fmt.Errorf("reading the overlay: %w", err)
				}
				for _, f := range overlay.GetFiles() {
					name := filepath.Clean(f.GetPath())
					if _, ok := r.texts[name]; !ok && !slices.ContainsFunc(args, func(a string) bool { return filepath.Clean(a) == name }) {
						paths = append(paths, f.GetPath())
					}
					r.texts[name] = f.GetText()
				}
			}
			if len(paths) == 0 {
				return errors.New("no files given and no overlay read")
			}

			backend, err := r.resolve(r.target)
			if err != nil {
				return fmt.Errorf("%w; compiled in: %v", err, gen.BuiltinNames())
			}
			r.desc = backend.Describe()
			if sub, ok := backend.(*gen.Subprocess); ok {
				if r.desc, err = sub.Probe(); err != nil {
					return fmt.Errorf("%s: %w", sub.Path, err)
				}
			}

			r.set = &plugin.RequestSet{}
			for _, path := range paths {
				r.request(path)
			}

			conn := plugin.NewConn(nil, cmd.OutOrStdout())
			if err := conn.Send(&plugin.Handshake{FramingVersion: plugin.FramingVersion, IrVersion: plugin.IRVersion}); err != nil {
				return err
			}
			return conn.Send(r.set)
		},
	}

	cmd.Flags().StringVar(&r.target, "target", "", "the backend the requests are for")
	cmd.Flags().BoolVar(&r.overlay, "overlay", false, "read file texts from standard input before the disk")
	cmd.Flags().StringSliceVar(&r.allowLossy, "allow-lossy", nil, "drop warnings with these loss codes")
	_ = cmd.MarkFlagRequired("target")
	return cmd
}

// requestRun is one invocation of request.
type requestRun struct {
	target     string
	overlay    bool
	allowLossy []string

	resolve func(string) (plugin.Backend, error)
	desc    plugin.Description

	// texts holds the overlay's files by cleaned path.
	texts map[string]string

	set *plugin.RequestSet
}

// text serves the overlay to sema.Overlay.
func (r *requestRun) text(name string) (string, bool) {
	src, ok := r.texts[filepath.Clean(name)]
	return src, ok
}

// request adds one file's requests, or what stopped them, to the set.
func (r *requestRun) request(path string) {
	src, ok := r.text(path)
	if !ok {
		data, err := os.ReadFile(path)
		if err != nil {
			r.fail(&ir.Position{Filename: path}, err.Error())
			return
		}
		src = string(data)
	}

	file, err := parser.Parse(path, strings.NewReader(src))
	if err != nil {
		var errs parser.ErrorList
		if !errors.As(err, &errs) {
			r.fail(&ir.Position{Filename: path}, err.Error())
			return
		}
		for _, e := range errs {
			r.fail(&ir.Position{Filename: e.Pos.Filename, Line: int32(e.Pos.Line), Column: int32(e.Pos.Col)}, e.Msg)
		}
		return
	}

	model, diags := sema.Lower(file, sema.WithLoader(sema.Overlay{Text: r.text}))
	for _, d := range diags {
		r.fail(&ir.Position{
			Filename:  d.Pos.Filename,
			Line:      int32(d.Pos.Line),
			Column:    int32(d.Pos.Col),
			EndLine:   int32(d.End.Line),
			EndColumn: int32(d.End.Col),
		}, d.Msg)
	}
	if len(diags) > 0 {
		return
	}

	blocks := slices.DeleteFunc(slices.Clone(model.GetTargets()), func(b *ir.TargetBlock) bool {
		return b.GetMeta().GetName() != r.target
	})
	if len(blocks) == 0 {
		return
	}

	cfg, err := config.Find(filepath.Dir(path))
	if err != nil {
		r.fail(&ir.Position{Filename: path}, err.Error())
		return
	}
	allowed := slices.Concat(cfg.Allowed(r.target), r.allowLossy)

	// Checked once per model, as gen checks before running each block.
	problems := host.CheckDirectives(r.target, model, r.desc)
	r.set.Diagnostics = append(r.set.Diagnostics, host.Silence(problems, allowed)...)
	if host.Fatal(problems) {
		return
	}
	for _, block := range blocks {
		req, problems := analysis.Request(model, r.desc, r.target, gen.Out(block))
		r.set.Diagnostics = append(r.set.Diagnostics, host.Silence(problems, allowed)...)
		if !host.Fatal(problems) {
			r.set.Requests = append(r.set.Requests, req)
		}
	}
}

// fail adds an error diagnostic.
func (r *requestRun) fail(pos *ir.Position, msg string) {
	r.set.Diagnostics = append(r.set.Diagnostics, &plugin.Diagnostic{
		Severity: plugin.Severity_SEVERITY_ERROR,
		Message:  msg,
		Position: pos,
	})
}
