package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/internal/gen"
)

// twoFiles writes a canonical model and a broken one.
func twoFiles(t *testing.T) (good, bad string) {
	t.Helper()
	dir := t.TempDir()
	good, bad = filepath.Join(dir, "good.tdl"), filepath.Join(dir, "bad.tdl")
	if err := os.WriteFile(good, []byte("primitive string\n"), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
	if err := os.WriteFile(bad, []byte("type E: Entity { id string }\n"), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
	return good, bad
}

// run executes cmd with its streams captured and usage silenced, as the
// root command runs it.
func run(t *testing.T, cmd *cobra.Command, args ...string) (out, errOut string, err error) {
	t.Helper()
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	o, e := captureCmd(cmd)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return o.String(), e.String(), err
}

// A banner is printed only when there is more than one file.
func TestCommandsTakeSeveralFiles(t *testing.T) {
	good, _ := twoFiles(t)

	cases := []struct {
		name string
		cmd  func() *cobra.Command
		want string // a fragment the single-file output must contain
	}{
		{"ast", newAstCmd, "File "},
		{"fmt", newFmtCmd, "primitive string"},
		{"ir", newIrCmd, "primitive string"},
		{"tokens", newTokensCmd, "IDENT"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			one, _, err := run(t, tc.cmd(), good)
			if err != nil {
				t.Fatalf("one file: %v", err)
			}
			if !strings.Contains(one, tc.want) {
				t.Errorf("output does not contain %q:\n%s", tc.want, one)
			}
			if strings.Contains(one, "==>") {
				t.Errorf("a single file should print no banner:\n%s", one)
			}

			two, _, err := run(t, tc.cmd(), good, good)
			if err != nil {
				t.Fatalf("two files: %v", err)
			}
			if got := strings.Count(two, "==> "+good+" <=="); got != 2 {
				t.Errorf("banner appeared %d times, want 2:\n%s", got, two)
			}

			// A file that fails before printing leaves no gap.
			after, _, _ := run(t, tc.cmd(), filepath.Join(t.TempDir(), "missing.tdl"), good)
			if !strings.HasPrefix(after, "==> ") {
				t.Errorf("output after a failed file should start with a banner:\n%q", after)
			}
		})
	}
}

func TestCheckIsSilentOnSuccess(t *testing.T) {
	good, _ := twoFiles(t)

	out, errOut, err := run(t, newCheckCmd(), good, good)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" || errOut != "" {
		t.Errorf("expected no output, got stdout %q stderr %q", out, errOut)
	}
}

// A broken file does not stop the rest, and the error only counts failures.
func TestCommandsReportEveryBadFile(t *testing.T) {
	good, bad := twoFiles(t)

	_, errOut, err := run(t, newCheckCmd(), bad, good, bad)
	if err == nil {
		t.Fatal("expected an error")
	}
	if got, want := err.Error(), "2 files failed"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
	if got := strings.Count(errOut, bad+":"); got != 2 {
		t.Errorf("the bad file was reported %d times, want 2:\n%s", got, errOut)
	}
}

func TestCommandsRejectNoArguments(t *testing.T) {
	for name, newCmd := range map[string]func() *cobra.Command{
		"ast":    newAstCmd,
		"check":  newCheckCmd,
		"fmt":    newFmtCmd,
		"gen":    newGenCmd,
		"ir":     newIrCmd,
		"tokens": newTokensCmd,
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := run(t, newCmd()); err == nil {
				t.Error("expected an error when given no file")
			}
		})
	}
}

func TestGenWatchTakesOneFile(t *testing.T) {
	good, _ := twoFiles(t)

	_, _, err := run(t, newGenCmd(), "--watch", good, good)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "single file") {
		t.Errorf("error = %q, want it to mention a single file", err)
	}
}

func TestFmtWriteRewritesEveryFile(t *testing.T) {
	dir := t.TempDir()
	var paths []string
	for _, name := range []string{"a.tdl", "b.tdl"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("primitive    string"), 0o644); err != nil {
			t.Fatalf("writing the fixture: %v", err)
		}
		paths = append(paths, path)
	}

	out, _, err := run(t, newFmtCmd(), append([]string{"-w"}, paths...)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("-w should print nothing, got %q", out)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading back: %v", err)
		}
		if got, want := string(data), "primitive string\n"; got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
}

func TestGenWritesAndVerifies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.tdl")
	src := "package p\n\nprimitive string\n\ntype E: Entity {\n  id: string\n}\n\ntarget debug for p {\n  out(\"./out\")\n}\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	// The block's out directive is relative to the working directory.
	out := filepath.Join(dir, "out")
	if _, _, err := run(t, newGenCmd(), "-o", out, path); err != nil {
		t.Fatalf("gen: %v", err)
	}
	written := filepath.Join(out, "model.txt")
	if _, err := os.Stat(written); err != nil {
		t.Fatalf("gen wrote nothing: %v", err)
	}

	if _, _, err := run(t, newGenCmd(), "-o", out, "--verify", path); err != nil {
		t.Errorf("--verify over fresh output: %v", err)
	}

	if err := os.WriteFile(written, []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("staling the output: %v", err)
	}
	if _, _, err := run(t, newGenCmd(), "-o", out, "--verify", path); err == nil {
		t.Error("--verify accepted stale output")
	}
}

// --clean empties a shared output directory once per run, not once per file.
func TestGenCleansASharedOutputDirectoryOnce(t *testing.T) {
	dir := t.TempDir()
	src := "package p\n\nprimitive string\n\ntarget debug for p {\n  out(\"./out\")\n}\n"
	var paths []string
	for _, name := range []string{"a.tdl", "b.tdl"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatalf("writing the fixture: %v", err)
		}
		paths = append(paths, path)
	}

	out := filepath.Join(dir, "out")
	if _, _, err := run(t, newGenCmd(), "-o", out, paths[0]); err != nil {
		t.Fatalf("gen: %v", err)
	}
	orphan := filepath.Join(out, "orphan.txt")
	if err := os.WriteFile(orphan, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("writing the orphan: %v", err)
	}
	if err := gen.Mark(out, []string{orphan}); err != nil {
		t.Fatalf("listing the orphan: %v", err)
	}

	stdout, _, err := run(t, newGenCmd(), append([]string{"-o", out, "--clean"}, paths...)...)
	if err != nil {
		t.Fatalf("gen --clean: %v", err)
	}
	if !strings.Contains(stdout, "removed "+orphan) {
		t.Errorf("--clean did not remove the orphan:\n%s", stdout)
	}
	// The orphan and the earlier model.txt; a second clean would make three.
	if got := strings.Count(stdout, "removed "); got != 2 {
		t.Errorf("%d removals, want 2 (one clean pass):\n%s", got, stdout)
	}
	if _, err := os.Stat(orphan); err == nil {
		t.Error("the orphan is still on disk")
	}
	if _, err := os.Stat(filepath.Join(out, "model.txt")); err != nil {
		t.Errorf("model.txt missing after the run: %v", err)
	}
}

func TestGenRejectsAFileWithNoTarget(t *testing.T) {
	good, _ := twoFiles(t)

	if _, _, err := run(t, newGenCmd(), good); err == nil {
		t.Error("expected an error for a file declaring no target block")
	}
}

func TestGenVerifyAndCleanAreExclusive(t *testing.T) {
	good, _ := twoFiles(t)

	if _, _, err := run(t, newGenCmd(), "--verify", "--clean", good); err == nil {
		t.Error("expected an error when --verify is combined with --clean")
	}
}

func TestGenWatchAndVerifyAreExclusive(t *testing.T) {
	good, _ := twoFiles(t)

	if _, _, err := run(t, newGenCmd(), "--watch", "--verify", good); err == nil {
		t.Error("expected an error when --watch is combined with --verify")
	}
}
