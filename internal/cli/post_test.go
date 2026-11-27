package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
)

// postModel is a model under models/ whose debug target writes
// models/out/model.txt, with the given entries in its target block.
func postModel(block string) string {
	return "package p\n\nprimitive string\n\ntype Order: Entity {\n  id: string\n}\n\n" +
		"target debug for p {\n  out(\"./out\")\n" + block + "}\n"
}

// writePostProject writes the model under dir/models and, unless toml is
// empty, a tdl.toml in dir, and returns the model's path.
func writePostProject(t *testing.T, dir, block, toml string) string {
	t.Helper()
	files := map[string]string{"models/m.tdl": postModel(block)}
	if toml != "" {
		files[filepath.Join("tdl.toml")] = toml
	}
	writeFiles(t, dir, files)
	return filepath.Join(dir, "models", "m.tdl")
}

// Post commands filter every file before it is written, and a verify run
// filters again, so fresh output is not stale.
func TestGenPostFiltersWrittenFiles(t *testing.T) {
	dir := t.TempDir()
	path := writePostProject(t, dir, "  post(\"up\")\n", "[post]\nup = \"tr a-z A-Z\"\n")

	if _, errOut, err := run(t, newGenCmd(), path); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}
	got, err := os.ReadFile(filepath.Join(dir, "models", "out", "model.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || !bytes.Equal(got, bytes.ToUpper(got)) {
		t.Errorf("model.txt is not upper-cased:\n%s", got)
	}

	if _, errOut, err := run(t, newGenCmd(), "--verify", path); err != nil {
		t.Errorf("--verify right after gen: %v\n%s", err, errOut)
	}
}

// Two names run in order, each one's output feeding the next, and each runs
// in the tdl.toml directory with {file} the destination relative to it.
func TestGenPostRunsInOrderFromTheManifestDir(t *testing.T) {
	dir := t.TempDir()
	toml := "[post]\n" +
		"where = 'sh -c \"cat >/dev/null; echo $1; cat here\" sh {file}'\n" +
		"up = \"tr a-z A-Z\"\n"
	path := writePostProject(t, dir, "  post(\"where\", \"up\")\n", toml)
	// where reads here from its working directory, which shows where it ran
	// without comparing paths, which Windows spells another way.
	writeFiles(t, dir, map[string]string{"here": "home\n"})

	if _, errOut, err := run(t, newGenCmd(), path); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}
	got, err := os.ReadFile(filepath.Join(dir, "models", "out", "model.txt"))
	if err != nil {
		t.Fatal(err)
	}
	// Upper-cased, so `up` ran after `where`.
	want := strings.ToUpper(filepath.Join("models", "out", "model.txt") + "\nhome\n")
	if string(got) != want {
		t.Errorf("model.txt = %q, want %q", got, want)
	}
}

// Post-processing changes the bytes, never which files a run lists: the
// marker and the dependency file are the same with and without it.
func TestGenPostListsTheSameFiles(t *testing.T) {
	dir := t.TempDir()
	toml := "[post]\nup = \"tr a-z A-Z\"\n"
	stamp := filepath.Join(dir, ".stamp")
	marker := filepath.Join(dir, "models", "out", gen.MarkerName)

	generate := func(block string) (string, string) {
		t.Helper()
		path := writePostProject(t, dir, block, toml)
		if _, errOut, err := run(t, newGenCmd(), "--clean", "--make", stamp, path); err != nil {
			t.Fatalf("gen: %v\n%s", err, errOut)
		}
		m, err := os.ReadFile(marker)
		if err != nil {
			t.Fatal(err)
		}
		d, err := os.ReadFile(stamp)
		if err != nil {
			t.Fatal(err)
		}
		return string(m), string(d)
	}

	plainMarker, plainDeps := generate("")
	postMarker, postDeps := generate("  post(\"up\")\n")
	if plainMarker != postMarker {
		t.Errorf("marker with post =\n%s\nwithout =\n%s", postMarker, plainMarker)
	}
	if plainDeps != postDeps {
		t.Errorf("dependency file with post =\n%s\nwithout =\n%s", postDeps, plainDeps)
	}
}

// Each way a post command can fail is an error naming the target, and
// nothing is written for it.
func TestGenPostErrors(t *testing.T) {
	cases := []struct {
		name, block, toml string
		want              []string
	}{
		{
			"exits non-zero", "  post(\"fail\")\n", "[post]\nfail = 'sh -c \"echo nope >&2; exit 2\"'\n",
			[]string{"target debug", "post command fail failed", "exit status 2", "nope"},
		},
		{
			"undeclared", "  post(\"biome\")\n", "[post]\nup = \"tr a-z A-Z\"\n",
			[]string{"target debug", `post names "biome"`, `add biome = "<command>"`},
		},
		{
			"no tdl.toml", "  post(\"up\")\n", "",
			[]string{"target debug", `post names "up"`, "no tdl.toml"},
		},
		{
			"not found", "  post(\"gone\")\n", "[post]\ngone = \"tdl-no-such-formatter --check\"\n",
			[]string{"target debug", "post command gone", "tdl-no-such-formatter was not found"},
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

// A post directive outside the target block, or with no arguments, is
// refused before anything runs.
func TestGenRefusesAMisusedPost(t *testing.T) {
	cases := map[string]string{
		"on a declaration": "  Order => post(\"up\")\n",
		"no arguments":     "  post()\n",
	}
	for name, block := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := writePostProject(t, dir, block, "[post]\nup = \"tr a-z A-Z\"\n")
			_, errOut, err := run(t, newGenCmd(), path)
			if err == nil || !strings.Contains(errOut, "target debug uses a directive incorrectly") {
				t.Fatalf("err = %v\n%s", err, errOut)
			}
			if !strings.Contains(errOut, "error: post ") {
				t.Errorf("stderr does not report post:\n%s", errOut)
			}
			if _, err := os.Stat(filepath.Join(dir, "models", "out")); err == nil {
				t.Error("gen wrote output")
			}
		})
	}
}

// The smoke model's typescript output, formatted by Biome through post,
// passes `biome check`.
func TestGenPostBiome(t *testing.T) {
	if testing.Short() {
		t.Skip("runs biome")
	}
	biome, err := exec.LookPath("biome")
	if err != nil {
		t.Skip("biome is not on PATH")
	}

	src, err := os.ReadFile(filepath.Join("..", "..", "testdata", "gen", "smoke", "source.tdl"))
	if err != nil {
		t.Fatal(err)
	}
	const block = "target typescript for smoke {\n  out(\"typescript\")\n"
	if !bytes.Contains(src, []byte(block)) {
		t.Fatalf("the smoke model has no %q", block)
	}
	src = bytes.Replace(src, []byte(block), []byte(block+"  post(\"biome\")\n"), 1)

	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"source.tdl": string(src),
		"tdl.toml":   "[post]\nbiome = \"biome format --stdin-file-path={file}\"\n",
	})
	path := filepath.Join(dir, "source.tdl")
	if _, errOut, err := run(t, newGenCmd(), "--target", "typescript", path); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}

	check := exec.Command(biome, "check", "typescript")
	check.Dir = dir
	if out, err := check.CombinedOutput(); err != nil {
		t.Errorf("biome check: %v\n%s", err, out)
	}
}
