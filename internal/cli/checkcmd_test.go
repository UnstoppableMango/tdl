package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
)

// A passing check lets the run write as usual, and runs in the staged
// directory, {dir}, beside the output directory, which is gone afterwards.
func TestGenCheckPassesAndWrites(t *testing.T) {
	dir := t.TempDir()
	seen := filepath.ToSlash(filepath.Join(dir, "seen"))
	toml := "[check]\nwhere = 'sh -c \"echo \\\"$1\\\" >\\\"$2\\\"; test -f model.txt\" sh {dir} " + seen + "'\n"
	path := writePostProject(t, dir, "  check(\"where\")\n", toml)

	if _, errOut, err := run(t, newGenCmd(), path); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}
	if _, err := os.Stat(filepath.Join(dir, "models", "out", "model.txt")); err != nil {
		t.Fatalf("gen wrote nothing: %v", err)
	}

	got, err := os.ReadFile(seen)
	if err != nil {
		t.Fatal(err)
	}
	// The check found model.txt where it ran, so it ran in {dir}.
	wd := strings.TrimSpace(string(got))
	parent, err := filepath.EvalSymlinks(filepath.Dir(wd))
	if err != nil {
		t.Fatal(err)
	}
	models, err := filepath.EvalSymlinks(filepath.Join(dir, "models"))
	if err != nil {
		t.Fatal(err)
	}
	if parent != models {
		t.Errorf("staged in %s, want a directory in %s", wd, models)
	}
	if _, err := os.Stat(wd); err == nil {
		t.Errorf("staged directory %s outlived the run", wd)
	}
}

// A failing check is an error showing what it printed on both streams, and
// leaves the output directory exactly as it was, with or without --clean;
// its staged directory is removed too.
func TestGenCheckFailureWritesNothing(t *testing.T) {
	for _, args := range [][]string{nil, {"--clean"}} {
		t.Run(strings.Join(append([]string{"gen"}, args...), " "), func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.Join(dir, "models", "out")
			stale := filepath.Join(out, "stale.txt")
			writeFiles(t, dir, map[string]string{"models/out/stale.txt": "old\n"})

			toml := "[check]\nfail = 'sh -c \"echo to-stdout; echo to-stderr >&2; exit 3\"'\n"
			path := writePostProject(t, dir, "  check(\"fail\")\n", toml)

			_, errOut, err := run(t, newGenCmd(), append(args, path)...)
			if err == nil {
				t.Fatal("gen succeeded")
			}
			for _, w := range []string{"target debug", "check command fail failed", "exit status 3", "to-stdout", "to-stderr"} {
				if !strings.Contains(errOut, w) {
					t.Errorf("stderr = %q, want it to contain %q", errOut, w)
				}
			}

			entries, err := os.ReadDir(out)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Name() != "stale.txt" {
				t.Errorf("output directory holds %v, want only stale.txt", entries)
			}
			if b, _ := os.ReadFile(stale); string(b) != "old\n" {
				t.Errorf("stale.txt = %q", b)
			}
			models, err := os.ReadDir(filepath.Join(dir, "models"))
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range models {
				if strings.Contains(e.Name(), "tdl-check") {
					t.Errorf("staged directory %s outlived the run", e.Name())
				}
			}
		})
	}
}

// A check sees the bytes post wrote, not the backend's.
func TestGenCheckSeesPostProcessedFiles(t *testing.T) {
	dir := t.TempDir()
	toml := "[post]\nup = \"tr a-z A-Z\"\n" +
		"[check]\nupper = 'sh -c \"! grep -q [a-z] model.txt\"'\n"
	path := writePostProject(t, dir, "  post(\"up\")\n  check(\"upper\")\n", toml)
	if _, errOut, err := run(t, newGenCmd(), path); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}

	// Without post, the same check fails.
	dir = t.TempDir()
	path = writePostProject(t, dir, "  check(\"upper\")\n", toml)
	if _, _, err := run(t, newGenCmd(), path); err == nil {
		t.Error("check passed over lower-case output")
	}
}

// --verify runs no check, so a failing one does not fail it.
func TestGenVerifyRunsNoCheck(t *testing.T) {
	dir := t.TempDir()
	toml := "[check]\nok = \"true\"\n"
	path := writePostProject(t, dir, "  check(\"ok\")\n", toml)
	if _, errOut, err := run(t, newGenCmd(), path); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}

	ran := filepath.ToSlash(filepath.Join(dir, "ran"))
	writeFiles(t, dir, map[string]string{"tdl.toml": "[check]\nok = 'sh -c \"touch " + ran + "; exit 1\"'\n"})
	if _, errOut, err := run(t, newGenCmd(), "--verify", path); err != nil {
		t.Errorf("--verify: %v\n%s", err, errOut)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("--verify ran the check")
	}
}

// Each way a check can be misnamed is an error naming the target, and
// nothing is written for it.
func TestGenCheckErrors(t *testing.T) {
	cases := []struct {
		name, block, toml string
		want              []string
	}{
		{
			"undeclared", "  check(\"tsc\")\n", "[check]\nok = \"true\"\n",
			[]string{"target debug", `check names "tsc"`, `add tsc = "<command>"`},
		},
		{
			"declared under post", "  check(\"up\")\n", "[post]\nup = \"tr a-z A-Z\"\n",
			[]string{"target debug", `check names "up"`, "[check]"},
		},
		{
			"no tdl.toml", "  check(\"ok\")\n", "",
			[]string{"target debug", `check names "ok"`, "no tdl.toml"},
		},
		{
			"not found", "  check(\"gone\")\n", "[check]\ngone = \"tdl-no-such-checker {dir}\"\n",
			[]string{"target debug", "check command gone", "tdl-no-such-checker was not found"},
		},
		{
			"on a declaration", "  Order => check(\"ok\")\n", "[check]\nok = \"true\"\n",
			[]string{"target debug uses a directive incorrectly", "error: check "},
		},
		{
			"no arguments", "  check()\n", "[check]\nok = \"true\"\n",
			[]string{"target debug uses a directive incorrectly", "error: check "},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writePostProject(t, dir, tc.block, tc.toml)
			_, errOut, err := run(t, newGenCmd(), path)
			if err == nil {
				t.Fatal("gen succeeded")
			}
			for _, w := range tc.want {
				if !strings.Contains(errOut, w) {
					t.Errorf("stderr = %q, want it to contain %q", errOut, w)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, "models", "out")); err == nil {
				t.Error("gen wrote output")
			}
		})
	}
}

// The staged copy holds what the output directory already holds, so a
// check sees a hand-written file beside the generated ones, but not a file
// tdl owns that --clean removes.
func TestGenCheckSeesTheWholeOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "models", "out")
	writeFiles(t, dir, map[string]string{"models/out/sub/hand.txt": "kept\n"})

	toml := "[check]\nhand = 'sh -c \"test -f sub/hand.txt && test -f model.txt\"'\n"
	path := writePostProject(t, dir, "  check(\"hand\")\n", toml)
	if _, errOut, err := run(t, newGenCmd(), path); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}

	// A file tdl owns but no longer generates is staged by a plain run and
	// left out under --clean, as each run would leave the directory.
	owned := filepath.Join(out, "old.txt")
	if err := os.WriteFile(owned, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(out, gen.MarkerName)
	m, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, append(m, "old.txt\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, dir, map[string]string{"tdl.toml": "[check]\nold = 'sh -c \"test -f old.txt\"'\n"})
	path = writePostProject(t, dir, "  check(\"old\")\n", "")
	if _, errOut, err := run(t, newGenCmd(), path); err != nil {
		t.Errorf("gen without --clean did not stage old.txt: %v\n%s", err, errOut)
	}
	if _, _, err := run(t, newGenCmd(), "--clean", path); err == nil {
		t.Error("gen --clean staged a file it removes")
	}
}

// A program named by a relative path is relative to tdl.toml, as in post.
func TestGenCheckProgramIsRelativeToTheManifest(t *testing.T) {
	dir := t.TempDir()
	prog, body := "ok", "#!/bin/sh\nexit 0\n"
	if runtime.GOOS == "windows" {
		prog, body = "ok.bat", "@exit /b 0\r\n"
	}
	writeFiles(t, dir, map[string]string{"bin/" + prog: body})
	if err := os.Chmod(filepath.Join(dir, "bin", prog), 0o755); err != nil {
		t.Fatal(err)
	}
	path := writePostProject(t, dir, "  check(\"ok\")\n", "[check]\nok = \"./bin/"+prog+" {dir}\"\n")
	if _, errOut, err := run(t, newGenCmd(), path); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}
}

// When the directory above the output cannot be written, the copy is
// staged in the system's temporary directory instead.
func TestGenCheckStagesElsewhereBesideAReadOnlyParent(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root writes to read-only directories")
	}
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"models/out/.keep": ""})
	path := writePostProject(t, dir, "  check(\"ok\")\n", "[check]\nok = \"true\"\n")
	models := filepath.Join(dir, "models")
	if err := os.Chmod(models, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(models, 0o755) })

	if _, errOut, err := run(t, newGenCmd(), path); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}
	if _, err := os.Stat(filepath.Join(models, "out", "model.txt")); err != nil {
		t.Error(err)
	}
}

// A check failing before --clean empties a directory leaves it to be
// cleaned by the next file writing there in the same run.
func TestGenCleanAfterAFailedCheck(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"tdl.toml":                     "[check]\nfail = 'sh -c \"exit 1\"'\n",
		"models/a.tdl":                 postModel("  check(\"fail\")\n"),
		"models/b.tdl":                 postModel(""),
		"models/out/old.txt":           "old\n",
		"models/out/" + gen.MarkerName: "old.txt\n",
	})

	_, errOut, err := run(t, newGenCmd(), "--clean", filepath.Join(dir, "models", "a.tdl"), filepath.Join(dir, "models", "b.tdl"))
	if err == nil || !strings.Contains(errOut, "check command fail failed") {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}
	if _, err := os.Stat(filepath.Join(dir, "models", "out", "old.txt")); err == nil {
		t.Error("old.txt survived --clean: the second file did not clean")
	}
	if _, err := os.Stat(filepath.Join(dir, "models", "out", "model.txt")); err != nil {
		t.Errorf("the second file wrote nothing: %v", err)
	}
}
