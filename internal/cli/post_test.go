package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// postModel is a model whose debug target block names post commands, and
// a tdl.toml declaring them; "toml" is the tdl.toml content, or "" for
// none.
func postModel(t *testing.T, post, toml string) (dir, file string) {
	t.Helper()
	dir = t.TempDir()
	files := map[string]string{
		"m.tdl": "package p\n\nprimitive string\n\ntype Order {\n  id: string\n}\n\ntarget debug for p {\n  out(\"./out\")\n  " + post + "\n}\n",
	}
	if toml != "" {
		files["tdl.toml"] = toml
	}
	writeFiles(t, dir, files)
	return dir, filepath.Join(dir, "m.tdl")
}

func readOut(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "out", "model.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// A post command filters every file before it is written, and a verify
// right after compares the filtered bytes, so nothing is stale.
func TestPostFiltersWrittenFiles(t *testing.T) {
	dir, file := postModel(t, `post("up")`, "[post]\nup = \"tr a-z A-Z\"\n")
	if _, errOut, err := run(t, newGenCmd(), file); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}
	got := readOut(t, dir)
	if got != strings.ToUpper(got) || !strings.Contains(got, "ORDER") {
		t.Errorf("model.txt was not upper-cased:\n%s", got)
	}
	if _, errOut, err := run(t, newGenCmd(), file, "--verify"); err != nil {
		t.Errorf("verify: %v\n%s", err, errOut)
	}
}

// Names run in order, each one's output feeding the next, across one
// directive's arguments and repeated directives.
func TestPostRunsInOrder(t *testing.T) {
	toml := "[post]\nup = \"tr a-z A-Z\"\nmark = \"sed s/^/x/\"\n"
	for _, tt := range []struct{ post, want string }{
		{`post("up", "mark")`, "xPACKAGE"},
		{`post("mark", "up")`, "XPACKAGE"},
		{`post("up") post("mark")`, "xPACKAGE"},
	} {
		dir, file := postModel(t, tt.post, toml)
		if _, errOut, err := run(t, newGenCmd(), file); err != nil {
			t.Fatalf("%s: gen: %v\n%s", tt.post, err, errOut)
		}
		if got := readOut(t, dir); !strings.HasPrefix(got, tt.want) {
			t.Errorf("%s: model.txt starts %q, want %q", tt.post, got[:min(len(got), 20)], tt.want)
		}
	}
}

// {file} is the destination relative to the directory holding tdl.toml,
// and the command runs in that directory, though the model is below it.
func TestPostFileAndDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"tdl.toml":        "[post]\nwhere = \"sh -c 'pwd; echo \\\"$0\\\"' {file}\"\n",
		"models/shop.tdl": "package p\n\nprimitive string\n\ntarget debug for p {\n  out(\"../gen/debug\")\n  post(\"where\")\n}\n",
	})
	if _, errOut, err := run(t, newGenCmd(), filepath.Join(dir, "models", "shop.tdl")); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}
	got, err := os.ReadFile(filepath.Join(dir, "gen", "debug", "model.txt"))
	if err != nil {
		t.Fatal(err)
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := real + "\ngen/debug/model.txt\n"; string(got) != want {
		t.Errorf("model.txt = %q, want %q", got, want)
	}
}

// A failing command, an undeclared name, a missing tdl.toml, and a missing
// executable each fail the target, name it, and write nothing.
func TestPostFailures(t *testing.T) {
	for _, tt := range []struct {
		name, post, toml string
		want             []string
	}{
		{"exit", `post("fail")`, "[post]\nfail = \"sh -c 'echo broken >&2; exit 3'\"\n", []string{"target debug", "fail", "broken"}},
		{"undeclared", `post("biome")`, "[post]\nup = \"tr a-z A-Z\"\n", []string{"target debug", `"biome"`, "[post]"}},
		{"no tdl.toml", `post("biome")`, "", []string{"target debug", `"biome"`, "tdl.toml"}},
		{"missing", `post("gone")`, "[post]\ngone = \"tdl-test-no-such-command\"\n", []string{"target debug", "tdl-test-no-such-command"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir, file := postModel(t, tt.post, tt.toml)
			_, errOut, err := run(t, newGenCmd(), file)
			if err == nil {
				t.Fatal("gen succeeded")
			}
			msg := err.Error() + errOut
			for _, w := range tt.want {
				if !strings.Contains(msg, w) {
					t.Errorf("error %q does not mention %q", msg, w)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, "out")); !os.IsNotExist(err) {
				t.Errorf("out was written: %v", err)
			}
		})
	}
}

// Post-processing changes what files hold, not which files are written, so
// .tdl-output and --make list the same files either way.
func TestPostKeepsTheFileList(t *testing.T) {
	list := func(post, toml string) (marker, dep string) {
		dir, file := postModel(t, post, toml)
		stamp := filepath.Join(dir, "stamp.d")
		if _, errOut, err := run(t, newGenCmd(), file, "--make", stamp); err != nil {
			t.Fatalf("gen: %v\n%s", err, errOut)
		}
		m, err := os.ReadFile(filepath.Join(dir, "out", ".tdl-output"))
		if err != nil {
			t.Fatal(err)
		}
		d, err := os.ReadFile(stamp)
		if err != nil {
			t.Fatal(err)
		}
		return string(m), strings.ReplaceAll(string(d), dir, "DIR")
	}
	toml := "[post]\nup = \"tr a-z A-Z\"\n"
	m1, d1 := list(`post("up")`, toml)
	m2, d2 := list("", toml)
	if m1 != m2 {
		t.Errorf(".tdl-output with post:\n%s\nwithout:\n%s", m1, m2)
	}
	if d1 != d2 {
		t.Errorf("dependency file with post:\n%s\nwithout:\n%s", d1, d2)
	}
}

// The smoke model's typescript output, formatted by Biome as a post
// command, is what a default `biome check` accepts.
func TestPostBiome(t *testing.T) {
	if testing.Short() {
		t.Skip("runs biome")
	}
	biome, err := exec.LookPath("biome")
	if err != nil {
		t.Skip("biome is not on PATH")
	}
	src, err := os.ReadFile("../../testdata/gen/smoke/source.tdl")
	if err != nil {
		t.Fatal(err)
	}
	block := "target typescript for smoke {\n"
	if !strings.Contains(string(src), block) {
		t.Fatal("the smoke model has no typescript target block")
	}
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"source.tdl": strings.Replace(string(src), block, block+"  post(\"biome\")\n", 1),
		"tdl.toml":   "[post]\nbiome = \"biome format --stdin-file-path={file}\"\n",
	})
	if _, errOut, err := run(t, newGenCmd(), filepath.Join(dir, "source.tdl"), "--target", "typescript"); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}
	cmd := exec.Command(biome, "check", "typescript")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("biome check: %v\n%s", err, out)
	}
}

// A program named by a relative path is relative to the directory holding
// tdl.toml, wherever tdl runs.
func TestPostRelativeProgram(t *testing.T) {
	dir, file := postModel(t, `post("local")`, "[post]\nlocal = \"./bin/up\"\n")
	writeFiles(t, dir, map[string]string{"bin/up": "#!/bin/sh\ntr a-z A-Z\n"})
	if err := os.Chmod(filepath.Join(dir, "bin", "up"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if _, errOut, err := run(t, newGenCmd(), file); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}
	if got := readOut(t, dir); !strings.Contains(got, "ORDER") {
		t.Errorf("model.txt was not upper-cased:\n%s", got)
	}
}

// A misused post directive is reported where it is written, before its
// names are looked up.
func TestPostMisuseIsADirectiveError(t *testing.T) {
	_, file := postModel(t, "post(1)", "")
	_, errOut, err := run(t, newGenCmd(), file)
	if err == nil || !strings.Contains(errOut, "post argument 1 is") {
		t.Errorf("gen = %v\n%s", err, errOut)
	}
}
