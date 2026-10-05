package gen_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/plugin"
)

// write writes files under out and lists them in its marker, as a run does.
func write(t *testing.T, out string, files ...*plugin.File) {
	t.Helper()
	written, err := gen.Write(out, files)
	if err != nil {
		t.Fatal(err)
	}
	if err := gen.Mark(out, written); err != nil {
		t.Fatal(err)
	}
}

func handwrite(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A directory can hold tdl's files beside someone else's, and cleaning it
// removes only the ones the marker lists, and the directories that leaves
// empty.
func TestCleanRemovesOnlyWhatItWrote(t *testing.T) {
	out := t.TempDir()
	handwritten := filepath.Join(out, "keep.go")
	handwrite(t, handwritten)
	write(t, out, &plugin.File{Path: "a.txt"}, &plugin.File{Path: "nested/b.txt"})

	removed, err := gen.Clean(out)
	if err != nil {
		t.Fatalf("clean: %v", err)
	}
	if len(removed) != 2 {
		t.Errorf("removed %v, want two entries", removed)
	}
	if _, err := os.Stat(handwritten); err != nil {
		t.Error("a file tdl did not write was removed")
	}
	if _, err := os.Stat(filepath.Join(out, "nested")); err == nil {
		t.Error("a directory cleaning emptied was left behind")
	}
	if owned, err := gen.Owned(out); err != nil || len(owned) != 0 {
		t.Errorf("owned after clean = %v, %v", owned, err)
	}
}

// A file already in the output directory that tdl did not write is not
// overwritten, and nothing in that response is written.
func TestWriteRefusesAFileItDidNotWrite(t *testing.T) {
	out := t.TempDir()
	handwritten := filepath.Join(out, "health.go")
	handwrite(t, handwritten)

	if _, err := gen.Write(out, []*plugin.File{{Path: "node.go"}, {Path: "health.go", Content: []byte("generated")}}); err == nil {
		t.Fatal("overwrote a file tdl did not write")
	}
	if got, _ := os.ReadFile(handwritten); string(got) != "mine" {
		t.Errorf("health.go = %q", got)
	}
	if _, err := os.Stat(filepath.Join(out, "node.go")); err == nil {
		t.Error("part of a refused response was written")
	}
}

// A marker written before markers listed files claimed the whole
// directory, so every file in it is still tdl's.
func TestALegacyMarkerOwnsEverything(t *testing.T) {
	out := t.TempDir()
	handwrite(t, filepath.Join(out, "a.go"))
	if err := os.WriteFile(filepath.Join(out, gen.MarkerName), []byte("This directory is written by `tdl gen`.\n`tdl gen --clean` will delete its contents.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	owned, err := gen.Owned(out)
	if err != nil || !owned[filepath.Join(out, "a.go")] || len(owned) != 1 {
		t.Errorf("owned = %v, %v", owned, err)
	}
}

func TestCleanWithoutAMarker(t *testing.T) {
	out := t.TempDir()
	handwrite(t, filepath.Join(out, "notes.md"))
	if removed, err := gen.Clean(out); err != nil || len(removed) != 0 {
		t.Errorf("removed %v, err = %v", removed, err)
	}
}

func TestCleanOnAMissingDirectory(t *testing.T) {
	if _, err := gen.Clean(filepath.Join(t.TempDir(), "absent")); err != nil {
		t.Errorf("clean: %v", err)
	}
}

func TestVerify(t *testing.T) {
	out := t.TempDir()
	files := []*plugin.File{{Path: "a.txt", Content: []byte("current")}}

	// Nothing there yet.
	stale, paths, err := gen.Verify(out, files)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(stale) != 1 || stale[0].Reason != "missing" {
		t.Fatalf("stale = %+v", stale)
	}
	if len(paths) != 1 || paths[0] != filepath.Join(out, "a.txt") {
		t.Errorf("paths = %v", paths)
	}

	// Written, so nothing to report.
	write(t, out, files...)
	if stale, _, err := gen.Verify(out, files); err != nil || len(stale) != 0 {
		t.Fatalf("stale = %+v, err = %v", stale, err)
	}

	// Changed underneath.
	if err := os.WriteFile(filepath.Join(out, "a.txt"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, _, err = gen.Verify(out, files)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(stale) != 1 || stale[0].Reason != "differs" {
		t.Errorf("stale = %+v", stale)
	}
}

// A file tdl wrote and nothing would write any more is stale too, which is
// what catches a declaration someone deleted. A file the marker does not
// list was never tdl's, so it is not an orphan.
func TestOrphaned(t *testing.T) {
	out := t.TempDir()
	write(t, out, &plugin.File{Path: "gone.txt"}, &plugin.File{Path: "kept.txt"})
	handwrite(t, filepath.Join(out, "theirs.txt"))

	stale, err := gen.Orphaned(out, map[string]bool{filepath.Join(out, "kept.txt"): true})
	if err != nil {
		t.Fatalf("orphaned: %v", err)
	}
	if len(stale) != 1 || stale[0].Path != filepath.Join(out, "gone.txt") {
		t.Errorf("stale = %+v, want only gone.txt", stale)
	}
}
