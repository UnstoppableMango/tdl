package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/internal/config"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// stdoutName names the imported file in positions when it goes to
// standard output, so an import in it resolves from the working directory.
const stdoutName = "<stdout>"

func newImportCmd() *cobra.Command {
	return importCmd(gen.Resolve)
}

// importCmd builds the command over a resolver, so a test can serve a
// backend tdl does not compile in.
func importCmd(resolve func(string) (plugin.Backend, error)) *cobra.Command {
	var (
		from       string
		out        string
		pkg        string
		allowLossy []string
	)

	cmd := &cobra.Command{
		Use:   "import --from <target> <file>...",
		Short: "Turn a target language's source back into TDL",
		Long: "Turn a target language's source back into TDL.\n\n" +
			"The backend named by --from reads the files and returns a model,\n" +
			"which tdl prints as TDL and lowers once more before writing it, so\n" +
			"the result always passes `tdl check`. A target tdl has no backend\n" +
			"for resolves to tdl-gen-<name> on PATH, as in `tdl gen`.\n\n" +
			"What the source cannot say in TDL, or says ambiguously, is a\n" +
			"warning naming a loss code, such as lossy.collection. The [lossy]\n" +
			"table of the nearest tdl.toml above the first file, or\n" +
			"--allow-lossy, silences one.\n\n" +
			"The result goes to standard output unless -o names a file.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if from == "" {
				return fmt.Errorf("--from names the language to import")
			}

			backend, err := resolve(from)
			if err != nil {
				return fmt.Errorf("%w; compiled in: %v", err, gen.BuiltinNames())
			}
			importer, ok := asImporter(backend)
			if !ok {
				return fmt.Errorf("%s does not import; compiled-in backends that do: %v", from, gen.ReverseNames())
			}

			files := make([]*plugin.File, 0, len(args))
			for _, path := range args {
				data, err := readSource(cmd, path)
				if err != nil {
					return err
				}
				files = append(files, &plugin.File{Path: displayName(path), Content: data})
			}

			dir := "."
			if !isStdin(args[0]) {
				dir = filepath.Dir(args[0])
			}
			cfg, err := config.Find(dir)
			if err != nil {
				return err
			}
			allowed := slices.Concat(cfg.Allowed(from), allowLossy)

			resp, err := importer.Import(cmd.Context(), &plugin.ImportRequest{
				Target:     from,
				Files:      files,
				AllowLossy: allowed,
			})
			if err != nil {
				return fmt.Errorf("%s: %w", from, err)
			}
			reportDiagnostics(cmd, gen.Silence(resp.GetDiagnostics(), allowed))
			if gen.Fatal(resp.GetDiagnostics()) {
				return fmt.Errorf("%s reported errors; nothing was written", from)
			}

			model := resp.GetModel()
			if model == nil {
				return fmt.Errorf("%s returned no model", from)
			}
			if pkg != "" {
				model.Package = pkg
			}
			src := ast.Fprint(unlower.File(model))

			name := stdoutName
			if out != "" {
				name = out
			}
			if err := relower(name, src); err != nil {
				return err
			}

			if out == "" {
				_, err := fmt.Fprint(cmd.OutOrStdout(), src)
				return err
			}
			return os.WriteFile(out, []byte(src), 0o644)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "the backend that reads the files")
	cmd.Flags().StringVarP(&out, "out", "o", "", "write the TDL here instead of standard output")
	cmd.Flags().StringVar(&pkg, "package", "", "the package the TDL declares, replacing the one the backend chose")
	cmd.Flags().StringSliceVar(&allowLossy, "allow-lossy", nil, "silence warnings with these loss codes")
	return cmd
}

// asImporter returns a backend that imports. A plugin is asked when it
// runs, since its description costs a process of its own.
func asImporter(b plugin.Backend) (plugin.Importer, bool) {
	if sub, ok := b.(*gen.Subprocess); ok {
		return sub, true
	}
	i, ok := b.(plugin.Importer)
	return i, ok && b.Describe().Reverse
}

// relower checks that printed TDL lowers, so import never writes a file
// `tdl check` rejects. The source follows the diagnostics, since they
// point into it and it was never written.
func relower(name, src string) error {
	file, err := parser.Parse(name, bytes.NewReader([]byte(src)))
	if err == nil {
		if _, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{})); len(diags) > 0 {
			err = diags
		}
	}
	if err != nil {
		return fmt.Errorf("the imported model does not lower, so nothing was written:\n%v\n\n%s", err, numbered(src))
	}
	return nil
}

func numbered(src string) string {
	lines := strings.Split(strings.TrimSuffix(src, "\n"), "\n")
	for i, l := range lines {
		lines[i] = fmt.Sprintf("%4d  %s", i+1, l)
	}
	return strings.Join(lines, "\n")
}
