package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unstoppablemango/tdl/internal/gen"
)

// writeFiles writes each file under dir, creating its directory.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, src := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
}

// makeModel is a model importing b.tdl, which imports c.tdl, beside a
// _ import standing in for a replacement prelude and a tdl.toml.
var makeModel = map[string]string{
	"a.tdl": "package a\n\nimport \"b.tdl\" as b\nimport \"std/prelude.tdl\" as _\n\ntype A {\n  b: b.B\n}\n\ntarget debug for a {\n  out(\"./out\")\n}\n",
	"b.tdl": "package b\n\nimport \"c.tdl\" as c\n\ntype B {\n  c: c.C\n}\n",
	"c.tdl": "package c\n\nprimitive string\n\ntype C {\n  s: string\n}\n",

	"std/prelude.tdl": "package std\n\nprimitive Extra\n",
	"tdl.toml":        "",
}

// The dependency file names every file the run read: the input, its
// transitive imports, a _ import, and the tdl.toml found above it.
func TestGenMakeWritesEveryFileRead(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, makeModel)
	stamp := filepath.Join(dir, "out", ".stamp")

	a := filepath.Join(dir, "a.tdl")
	if _, errOut, err := run(t, newGenCmd(), a, "--make", stamp); err != nil {
		t.Fatalf("gen --make: %v\n%s", err, errOut)
	}

	got, err := os.ReadFile(stamp)
	if err != nil {
		t.Fatalf("reading the dependency file: %v", err)
	}
	var want []string
	for _, name := range []string{"a.tdl", "b.tdl", "c.tdl", "std/prelude.tdl", "tdl.toml"} {
		want = append(want, filepath.Join(dir, name))
	}
	if string(got) != string(gen.DepFile(stamp, want)) {
		t.Errorf("dependency file =\n%s\nwant\n%s", got, gen.DepFile(stamp, want))
	}
}

// A successful verify writes the dependency file too, so a check-only
// target can use it as its stamp.
func TestGenVerifyMakeWritesOnSuccess(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, makeModel)
	a := filepath.Join(dir, "a.tdl")
	if _, errOut, err := run(t, newGenCmd(), a); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}

	stamp := filepath.Join(dir, ".verified")
	if _, errOut, err := run(t, newGenCmd(), a, "--verify", "--make", stamp); err != nil {
		t.Fatalf("gen --verify --make: %v\n%s", err, errOut)
	}
	if _, err := os.Stat(stamp); err != nil {
		t.Errorf("--verify --make wrote nothing: %v", err)
	}
}

// A run that fails leaves the dependency file's bytes and mtime alone, so
// make retries.
func TestGenMakeLeavesTheFileAloneOnFailure(t *testing.T) {
	cases := map[string]struct {
		args  []string
		spoil func(t *testing.T, dir string)
	}{
		"lowering": {
			spoil: func(t *testing.T, dir string) {
				writeFiles(t, dir, map[string]string{"b.tdl": "package b\n\ntype {\n"})
			},
		},
		"verify": {
			args: []string{"--verify"},
			spoil: func(t *testing.T, dir string) {
				writeFiles(t, dir, map[string]string{"out/model.txt": "stale\n"})
			},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, makeModel)
			a := filepath.Join(dir, "a.tdl")
			stamp := filepath.Join(dir, ".stamp")
			if _, errOut, err := run(t, newGenCmd(), a, "--make", stamp); err != nil {
				t.Fatalf("gen --make: %v\n%s", err, errOut)
			}
			before, err := os.ReadFile(stamp)
			if err != nil {
				t.Fatal(err)
			}
			old := time.Now().Add(-time.Hour).Truncate(time.Second)
			if err := os.Chtimes(stamp, old, old); err != nil {
				t.Fatal(err)
			}

			c.spoil(t, dir)
			if _, _, err := run(t, newGenCmd(), append([]string{a, "--make", stamp}, c.args...)...); err == nil {
				t.Fatal("expected the run to fail")
			}

			after, err := os.ReadFile(stamp)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Errorf("dependency file changed:\n%s\nwas\n%s", after, before)
			}
			if info, err := os.Stat(stamp); err != nil || !info.ModTime().Equal(old) {
				t.Errorf("mtime moved: %v, %v", info.ModTime(), err)
			}
		})
	}
}

func TestGenMakeNeedsATarget(t *testing.T) {
	good, _ := twoFiles(t)

	if _, _, err := run(t, newGenCmd(), good, "--make"); err == nil {
		t.Error("expected an error for --make with no target")
	}
}

func TestGenWatchAndMakeAreExclusive(t *testing.T) {
	good, _ := twoFiles(t)

	_, _, err := run(t, newGenCmd(), "--watch", "--make", filepath.Join(t.TempDir(), ".stamp"), good)
	if err == nil || !strings.Contains(err.Error(), "--make") {
		t.Errorf("expected --watch with --make to be rejected, got %v", err)
	}
}

// Under make, a second run does nothing, touching an import regenerates,
// and deleting an import with its reference does not break the build.
func TestGenMakeDrivesMake(t *testing.T) {
	if testing.Short() {
		t.Skip("runs make and tdl")
	}
	makeBin, err := exec.LookPath("make")
	if err != nil {
		t.Skip("make is not on PATH")
	}
	tdl := filepath.Join(t.TempDir(), "tdl")
	if out, err := exec.Command("go", "build", "-o", tdl, "github.com/unstoppablemango/tdl/cmd/tdl").CombinedOutput(); err != nil {
		t.Fatalf("building tdl: %v\n%s", err, out)
	}

	dir := t.TempDir()
	writeFiles(t, dir, makeModel)
	writeFiles(t, dir, map[string]string{"Makefile": "out/.stamp: a.tdl\n\t$(TDL) gen a.tdl --make $@\n\n-include out/.stamp\n"})

	runMake := func() string {
		t.Helper()
		cmd := exec.Command(makeBin, "--no-print-directory", "TDL="+tdl)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("make: %v\n%s", err, out)
		}
		return string(out)
	}
	ran := func(out string) bool { return strings.Contains(out, " gen a.tdl") }

	if out := runMake(); !ran(out) {
		t.Fatalf("the first make did not generate:\n%s", out)
	}
	if out := runMake(); ran(out) {
		t.Errorf("a second make regenerated:\n%s", out)
	}

	// An mtime may share a second with the stamp's, so age the stamp and
	// touch the import. An import in the future would remake the included
	// stamp forever.
	stamp := filepath.Join(dir, "out", ".stamp")
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(stamp, old, old); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := os.Chtimes(filepath.Join(dir, "c.tdl"), now, now); err != nil {
		t.Fatal(err)
	}
	if out := runMake(); !ran(out) {
		t.Errorf("touching a transitive import did not regenerate:\n%s", out)
	}

	if err := os.Chtimes(stamp, old, old); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, dir, map[string]string{"b.tdl": "package b\n\nprimitive string\n\ntype B {\n  s: string\n}\n"})
	if err := os.Remove(filepath.Join(dir, "c.tdl")); err != nil {
		t.Fatal(err)
	}
	if out := runMake(); !ran(out) {
		t.Errorf("removing an import did not regenerate:\n%s", out)
	}
}
