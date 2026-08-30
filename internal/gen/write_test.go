package gen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/plugin"
)

func TestWrite(t *testing.T) {
	out := t.TempDir()
	written, err := gen.Write(out, []*plugin.File{
		{Path: "a.txt", Content: []byte("a")},
		{Path: "nested/b.txt", Content: []byte("b")},
	})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if len(written) != 2 {
		t.Fatalf("wrote %d files, want 2", len(written))
	}

	for path, want := range map[string]string{"a.txt": "a", "nested/b.txt": "b"} {
		got, err := os.ReadFile(filepath.Join(out, path))
		if err != nil {
			t.Errorf("reading %s: %v", path, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
}

// A backend cannot reach outside the directory the project pointed it at,
// whatever it intends.
func TestWriteRefusesEscapingPaths(t *testing.T) {
	tests := []struct{ name, path string }{
		{"absolute", "/etc/passwd"},
		{"climbing", "../escaped.txt"},
		{"climbing through a subdirectory", "sub/../../escaped.txt"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := t.TempDir()
			if _, err := gen.Write(out, []*plugin.File{{Path: tt.path}}); err == nil {
				t.Fatal("the path was accepted")
			}

			entries, err := os.ReadDir(out)
			if err != nil {
				t.Fatalf("reading %s: %v", out, err)
			}
			if len(entries) != 0 {
				t.Errorf("something was written anyway: %v", entries)
			}
		})
	}
}

// Every path is checked before any file is written, so a response with one
// bad path writes nothing rather than half of itself.
func TestWriteIsAllOrNothing(t *testing.T) {
	out := t.TempDir()
	_, err := gen.Write(out, []*plugin.File{
		{Path: "good.txt", Content: []byte("fine")},
		{Path: "../bad.txt", Content: []byte("not fine")},
	})
	if err == nil {
		t.Fatal("the escaping path was accepted")
	}
	if !strings.Contains(err.Error(), "outside the output directory") {
		t.Errorf("err = %v", err)
	}

	if _, err := os.Stat(filepath.Join(out, "good.txt")); err == nil {
		t.Error("the good file was written despite the bad one")
	}
}

// A path that only looks like it climbs is fine.
func TestWriteAllowsInnocentDots(t *testing.T) {
	out := t.TempDir()
	if _, err := gen.Write(out, []*plugin.File{
		{Path: "sub/../back.txt", Content: []byte("x")},
		{Path: "..hidden.txt", Content: []byte("y")},
	}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "back.txt")); err != nil {
		t.Errorf("back.txt: %v", err)
	}
}
