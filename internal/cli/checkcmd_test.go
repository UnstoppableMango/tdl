package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A passing check lets the files be written.
func TestCheckPassWrites(t *testing.T) {
	dir, file := postModel(t, `check("ok")`, "[check]\nok = \"sh -c 'test -f model.txt'\"\n")
	if _, errOut, err := run(t, newGenCmd(), file); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}
	if got := readOut(t, dir); !strings.Contains(got, "Order") {
		t.Errorf("model.txt = %q", got)
	}
}

// A failing check writes nothing, cleans nothing, and reports the target,
// the command, and what the command printed on either stream.
func TestCheckFailWritesNothing(t *testing.T) {
	toml := "[check]\nok = \"true\"\nbad = \"sh -c 'echo to-stdout; echo to-stderr >&2; exit 2'\"\n"
	dir, file := postModel(t, `check("ok")`, toml)
	if _, errOut, err := run(t, newGenCmd(), file); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}
	before := readOut(t, dir)

	// The model changes, so a write would change model.txt, and --clean
	// would remove it first.
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(src), "id: string", "id: string\n  sku: string", 1)
	changed = strings.Replace(changed, `check("ok")`, `check("ok", "bad")`, 1)
	writeFiles(t, dir, map[string]string{"m.tdl": changed})

	for _, args := range [][]string{{file}, {file, "--clean"}} {
		_, errOut, err := run(t, newGenCmd(), args...)
		if err == nil {
			t.Fatalf("%v: gen succeeded", args)
		}
		msg := err.Error() + errOut
		for _, w := range []string{"target debug", "bad", "to-stdout", "to-stderr"} {
			if !strings.Contains(msg, w) {
				t.Errorf("%v: error %q does not mention %q", args, msg, w)
			}
		}
		if got := readOut(t, dir); got != before {
			t.Errorf("%v: model.txt changed to %q", args, got)
		}
	}
}

// An undeclared name, a missing tdl.toml, and a missing command are each
// an error, never a skip.
func TestCheckUndeclaredAndMissing(t *testing.T) {
	for _, tt := range []struct {
		name, check, toml string
		want              []string
	}{
		{"undeclared", `check("tsc")`, "[check]\nok = \"true\"\n", []string{"target debug", `"tsc"`, "[check]"}},
		{"no tdl.toml", `check("tsc")`, "", []string{"target debug", `"tsc"`, "tdl.toml"}},
		{"missing", `check("gone")`, "[check]\ngone = \"tdl-test-no-such-command\"\n", []string{"target debug", "tdl-test-no-such-command"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir, file := postModel(t, tt.check, tt.toml)
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

// --verify compares bytes that were checked when written, so it runs no
// check, even one that would now fail.
func TestVerifyRunsNoCheck(t *testing.T) {
	dir, file := postModel(t, `check("gate")`, "[check]\ngate = \"true\"\n")
	if _, errOut, err := run(t, newGenCmd(), file); err != nil {
		t.Fatalf("gen: %v\n%s", err, errOut)
	}
	writeFiles(t, dir, map[string]string{"tdl.toml": "[check]\ngate = \"false\"\n"})
	if _, errOut, err := run(t, newGenCmd(), file, "--verify"); err != nil {
		t.Errorf("verify: %v\n%s", err, errOut)
	}
	if _, _, err := run(t, newGenCmd(), file); err == nil {
		t.Error("a write ran no check")
	}
}

// A check sees the bytes post wrote, laid out as in the output directory,
// in a directory beside it that is gone afterwards, pass or fail; {dir}
// names that directory, which is also the working directory.
func TestCheckStagesPostProcessedFiles(t *testing.T) {
	record := filepath.Join(t.TempDir(), "where")
	for _, status := range []string{"0", "1"} {
		toml := "[post]\nup = \"tr a-z A-Z\"\n\n[check]\nsee = \"sh -c 'pwd > \\\"$0\\\"; echo \\\"$1\\\" >> \\\"$0\\\"; grep -q ORDER model.txt && exit " + status + "' " + record + " {dir}\"\n"
		dir, file := postModel(t, `post("up") check("see")`, toml)
		_, errOut, err := run(t, newGenCmd(), file)
		if (err == nil) != (status == "0") {
			t.Fatalf("exit %s: gen returned %v\n%s", status, err, errOut)
		}

		got, err := os.ReadFile(record)
		if err != nil {
			t.Fatalf("exit %s: the check never ran: %v", status, err)
		}
		lines := strings.Split(strings.TrimSpace(string(got)), "\n")
		if len(lines) != 2 {
			t.Fatalf("exit %s: record = %q", status, got)
		}
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			t.Fatal(err)
		}
		staged, _ := filepath.EvalSymlinks(lines[1])
		if staged == "" {
			staged = lines[1]
		}
		if lines[0] != lines[1] && lines[0] != staged {
			t.Errorf("exit %s: ran in %s, {dir} is %s", status, lines[0], lines[1])
		}
		if parent := filepath.Dir(lines[0]); parent != real {
			t.Errorf("exit %s: staged in %s, want beside the output in %s", status, parent, real)
		}
		if _, err := os.Stat(lines[1]); !os.IsNotExist(err) {
			t.Errorf("exit %s: %s is still there: %v", status, lines[1], err)
		}
	}
}
