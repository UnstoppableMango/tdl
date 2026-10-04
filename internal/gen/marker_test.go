package gen_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/plugin"
)

func TestCleanRefusesADirectoryItDoesNotOwn(t *testing.T) {
	out := t.TempDir()
	handwritten := filepath.Join(out, "notes.md")
	if err := os.WriteFile(handwritten, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := gen.Clean(out); !errors.Is(err, gen.ErrNotOurs) {
		t.Fatalf("err = %v, want ErrNotOurs", err)
	}
	if _, err := os.Stat(handwritten); err != nil {
		t.Error("the file was removed anyway")
	}
}

func TestCleanRemovesWhatItOwns(t *testing.T) {
	out := t.TempDir()
	if err := gen.Mark(out); err != nil {
		t.Fatal(err)
	}
	if _, err := gen.Write(out, []*plugin.File{
		{Path: "a.txt"},
		{Path: "nested/b.txt"},
	}); err != nil {
		t.Fatal(err)
	}

	removed, err := gen.Clean(out)
	if err != nil {
		t.Fatalf("clean: %v", err)
	}
	if len(removed) != 2 {
		t.Errorf("removed %v, want two entries", removed)
	}

	if !gen.Owned(out) {
		t.Error("cleaning removed the marker")
	}
}

func TestCleanAdoptsAnEmptyDirectory(t *testing.T) {
	if _, err := gen.Clean(t.TempDir()); err != nil {
		t.Errorf("clean: %v", err)
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
	stale, err := gen.Verify(out, files)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(stale) != 1 || stale[0].Reason != "missing" {
		t.Fatalf("stale = %+v", stale)
	}

	// Written, so nothing to report.
	if err := gen.Mark(out); err != nil {
		t.Fatal(err)
	}
	if _, err := gen.Write(out, files); err != nil {
		t.Fatal(err)
	}
	if stale, err := gen.Verify(out, files); err != nil || len(stale) != 0 {
		t.Fatalf("stale = %+v, err = %v", stale, err)
	}

	// Changed underneath.
	if err := os.WriteFile(filepath.Join(out, "a.txt"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, err = gen.Verify(out, files)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(stale) != 1 || stale[0].Reason != "differs" {
		t.Errorf("stale = %+v", stale)
	}
}

// A file tdl wrote and would no longer write is stale.
func TestVerifyReportsOrphans(t *testing.T) {
	out := t.TempDir()
	if err := gen.Mark(out); err != nil {
		t.Fatal(err)
	}
	if _, err := gen.Write(out, []*plugin.File{{Path: "gone.txt"}}); err != nil {
		t.Fatal(err)
	}

	stale, err := gen.Verify(out, []*plugin.File{{Path: "kept.txt"}})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(stale) != 2 {
		t.Fatalf("stale = %+v, want the missing one and the orphan", stale)
	}
}

// Without the marker, orphans are not reported.
func TestVerifyDoesNotClaimUnownedFiles(t *testing.T) {
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "theirs.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	stale, err := gen.Verify(out, nil)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(stale) != 0 {
		t.Errorf("a file tdl never wrote was called stale: %+v", stale)
	}
}
