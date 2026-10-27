package gen

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The rule names the target and each prerequisite, then gives every
// prerequisite an empty rule, as gcc -MD -MP writes.
func TestDepFileWritesARuleAndPhonyPrerequisites(t *testing.T) {
	got := string(DepFile("out/.stamp", []string{"b.tdl", "a.tdl", "b.tdl", "tdl.toml"}))
	want := "out/.stamp: a.tdl b.tdl tdl.toml\na.tdl:\nb.tdl:\ntdl.toml:\n"
	if got != want {
		t.Errorf("DepFile =\n%s\nwant\n%s", got, want)
	}
}

// A space, a hash, or a dollar would split or end the rule, so each is
// escaped the way GNU make reads it back.
func TestDepFileEscapesWhatMakeReads(t *testing.T) {
	got := string(DepFile("my out/.stamp", []string{"a b.tdl", "c$d.tdl", "e#f.tdl"}))
	want := "my\\ out/.stamp: a\\ b.tdl c$$d.tdl e\\#f.tdl\na\\ b.tdl:\nc$$d.tdl:\ne\\#f.tdl:\n"
	if got != want {
		t.Errorf("DepFile =\n%s\nwant\n%s", got, want)
	}
}

func TestWriteDepFileCreatesItsDirectory(t *testing.T) {
	target := filepath.Join(t.TempDir(), "nested", "dir", ".stamp")
	if err := WriteDepFile(target, []string{"a.tdl"}); err != nil {
		t.Fatalf("WriteDepFile: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading the dependency file: %v", err)
	}
	if want := string(DepFile(target, []string{"a.tdl"})); string(got) != want {
		t.Errorf("wrote\n%s\nwant\n%s", got, want)
	}
}

// Writing replaces the file whole and leaves no temporary beside it, and
// it touches the file even when the rule is unchanged, since make reads the
// mtime.
func TestWriteDepFileReplacesAndTouches(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".stamp")
	if err := WriteDepFile(target, []string{"a.tdl"}); err != nil {
		t.Fatalf("WriteDepFile: %v", err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(target, old, old); err != nil {
		t.Fatalf("aging the dependency file: %v", err)
	}
	if err := WriteDepFile(target, []string{"a.tdl"}); err != nil {
		t.Fatalf("WriteDepFile: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !info.ModTime().After(old) {
		t.Errorf("mtime %v not after %v", info.ModTime(), old)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the directory: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("directory holds %d entries, want only the dependency file", len(entries))
	}
}
