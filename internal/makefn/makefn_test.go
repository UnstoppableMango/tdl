package makefn_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/makefn"
)

// tree writes files under a temporary directory and returns it. A value is
// the file's source.
func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, src := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const pkg = "package shop\n"

func TestFiles(t *testing.T) {
	root := tree(t, map[string]string{
		"model/b.tdl":        pkg,
		"model/a.tdl":        pkg,
		"model/nested/c.tdl": pkg,
		"model/README.md":    "",
		"other/d.tdl":        pkg,
	})
	tests := []struct {
		name string
		dirs []string
		want []string
	}{
		{"one dir", []string{"model"}, []string{"model/a.tdl", "model/b.tdl", "model/nested/c.tdl"}},
		{"several", []string{"other", "model/nested"}, []string{"model/nested/c.tdl", "other/d.tdl"}},
		{"overlapping", []string{"model", "model/nested"}, []string{"model/a.tdl", "model/b.tdl", "model/nested/c.tdl"}},
		{"missing", []string{"nope"}, []string{}},
		{"absolute", []string{filepath.Join(root, "other")}, []string{"other/d.tdl"}},
		{"none", nil, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makefn.Files(root, tt.dirs)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Files(%q) = %q, want %q", tt.dirs, got, tt.want)
			}
		})
	}
}

func TestFilesReportsAWalkError(t *testing.T) {
	root := tree(t, map[string]string{"model/a.tdl": pkg})
	// A path through a file is not missing, so it is an error rather than
	// an empty list.
	if _, err := makefn.Files(root, []string{"model/a.tdl/x"}); err == nil {
		t.Error("Files listed a path through a file")
	}
}

func TestImports(t *testing.T) {
	root := tree(t, map[string]string{
		"model/order.tdl":  pkg + `import "money.tdl" as money` + "\n" + `import "../shim/type.tdl" as _` + "\n",
		"model/money.tdl":  pkg + `import "../shim/type.tdl" as _` + "\n",
		"model/unused.tdl": pkg,
		"shim/type.tdl":    pkg,
	})
	got, err := makefn.Imports(root, []string{"model/order.tdl"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"model/money.tdl", "model/order.tdl", "shim/type.tdl"}
	if !slices.Equal(got, want) {
		t.Errorf("Imports = %q, want %q", got, want)
	}
}

func TestImportsReportsAMissingImport(t *testing.T) {
	root := tree(t, map[string]string{"a.tdl": pkg + `import "gone.tdl" as gone` + "\n"})
	_, err := makefn.Imports(root, []string{"a.tdl"})
	if err == nil || !strings.Contains(err.Error(), "gone.tdl") {
		t.Errorf("Imports error = %v, want one naming gone.tdl", err)
	}
}

func TestImportsReportsASyntaxError(t *testing.T) {
	root := tree(t, map[string]string{"a.tdl": "package"})
	if _, err := makefn.Imports(root, []string{"a.tdl"}); err == nil {
		t.Error("Imports read a file that does not parse")
	}
}
