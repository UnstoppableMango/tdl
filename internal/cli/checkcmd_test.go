package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A passing check lets the run write as usual, and runs in the staged
// directory, {dir}, beside the output directory, which is gone afterwards.
func TestGenCheckPassesAndWrites(t *testing.T) {
	dir := t.TempDir()
	seen := filepath.Join(dir, "seen")
	toml := "[check]\nwhere = 'sh -c \"pwd >" + seen + "; echo $1 >>" + seen + "; test -f model.txt\" sh {dir}'\n"
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
	lines := strings.Fields(string(got))
	if len(lines) != 2 {
		t.Fatalf("check printed %q", got)
	}
	wd, dirArg := lines[0], lines[1]
	if real, err := filepath.EvalSymlinks(dirArg); err == nil {
		dirArg = real
	}
	if wd != dirArg {
		t.Errorf("check ran in %s, but {dir} was %s", wd, dirArg)
	}
	models, err := filepath.EvalSymlinks(filepath.Join(dir, "models"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(wd) != models {
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

	ran := filepath.Join(dir, "ran")
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
