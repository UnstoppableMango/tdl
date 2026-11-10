package gen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/plugin"
)

// escapingLink makes out/link a symlink to a directory beside out, and
// returns out and that directory.
func escapingLink(t *testing.T) (out, outside string) {
	t.Helper()
	dir := t.TempDir()
	out, outside = filepath.Join(dir, "out"), filepath.Join(dir, "outside")
	for _, d := range []string{out, outside} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join("..", "outside"), filepath.Join(out, "link")); err != nil {
		t.Skipf("cannot make a symlink: %v", err)
	}
	return out, outside
}

func assertEmpty(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("%s holds %v, want nothing", dir, entries)
	}
}

func assertLeaves(t *testing.T, err error, path string) {
	t.Helper()
	switch {
	case err == nil:
		t.Fatalf("%s was followed out of the output directory", path)
	case !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "outside the output directory"):
		t.Errorf("error = %q, want it to name %s and say it leaves the output directory", err, path)
	}
}

func TestWriteRefusesASymlinkLeavingTheTree(t *testing.T) {
	out, outside := escapingLink(t)
	_, err := gen.Write(out, []*plugin.File{
		{Path: "a.txt", Content: []byte("a")},
		{Path: "link/escaped.txt", Content: []byte("escaped")},
	})
	assertLeaves(t, err, "link/escaped.txt")
	assertEmpty(t, outside)
	if _, err := os.Stat(filepath.Join(out, "a.txt")); !os.IsNotExist(err) {
		t.Errorf("a.txt was written before the refusal: %v", err)
	}
}

func TestCleanRefusesASymlinkLeavingTheTree(t *testing.T) {
	out, outside := escapingLink(t)
	victim := filepath.Join(outside, "victim.txt")
	handwrite(t, victim)
	write(t, out, &plugin.File{Path: "mine.txt", Content: []byte("mine")})
	marker := filepath.Join(out, gen.MarkerName)
	content, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, append(content, "link/victim.txt\n"...), 0o644); err != nil {
		t.Fatal(err)
	}

	removed, err := gen.Clean(out)
	assertLeaves(t, err, "link/victim.txt")
	if _, err := os.Stat(victim); err != nil {
		t.Errorf("the file outside was removed: %v", err)
	}
	if want := filepath.Join(out, "mine.txt"); len(removed) != 1 || removed[0] != want {
		t.Errorf("removed %v, want only %s", removed, want)
	}
	// The entry stays listed, so the next clean reports it again.
	_, err = gen.Clean(out)
	assertLeaves(t, err, "link/victim.txt")
}

func TestOrphanedRefusesASymlinkLeavingTheTree(t *testing.T) {
	out, outside := escapingLink(t)
	handwrite(t, filepath.Join(outside, "x.txt"))
	if err := os.WriteFile(filepath.Join(out, gen.MarkerName), []byte("link/x.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := gen.Orphaned(out, nil)
	assertLeaves(t, err, "link/x.txt")
}

func TestVerifyRefusesASymlinkLeavingTheTree(t *testing.T) {
	out, outside := escapingLink(t)
	handwrite(t, filepath.Join(outside, "x.txt"))
	_, _, err := gen.Verify(out, []*plugin.File{{Path: "link/x.txt", Content: []byte("mine")}})
	assertLeaves(t, err, "link/x.txt")
}

func TestKeptCheckRefusesASymlinkLeavingTheTree(t *testing.T) {
	out, outside := escapingLink(t)
	handwrite(t, filepath.Join(outside, "x.txt"))
	marker := "#@ yolo request abc 1\n#@ yolo file 0000 link/x.txt\n"
	if err := os.WriteFile(filepath.Join(out, gen.MarkerName), []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}
	kept, ok, err := gen.Recorded(out, "yolo")
	if err != nil || !ok {
		t.Fatalf("recorded: %v, %v", ok, err)
	}
	_, err = kept.Check(out)
	assertLeaves(t, err, "link/x.txt")
}

// An output directory that is itself a symlink is opened at its target.
func TestASymlinkedOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	real, out := filepath.Join(dir, "real"), filepath.Join(dir, "out")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", out); err != nil {
		t.Skipf("cannot make a symlink: %v", err)
	}

	files := []*plugin.File{{Path: "nested/a.txt", Content: []byte("a")}}
	write(t, out, files...)
	if _, err := os.Stat(filepath.Join(real, "nested", "a.txt")); err != nil {
		t.Fatalf("not written through the link: %v", err)
	}
	stale, _, err := gen.Verify(out, files)
	if err != nil || len(stale) != 0 {
		t.Errorf("verify: %v, %v", stale, err)
	}
	removed, err := gen.Clean(out)
	if err != nil || len(removed) != 1 {
		t.Errorf("clean: %v, %v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(real, "nested")); !os.IsNotExist(err) {
		t.Errorf("the emptied directory survived: %v", err)
	}
}

// A symlink inside the tree that points within it is followed.
func TestASymlinkWithinTheTree(t *testing.T) {
	out := t.TempDir()
	if err := os.Mkdir(filepath.Join(out, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(out, "link")); err != nil {
		t.Skipf("cannot make a symlink: %v", err)
	}

	files := []*plugin.File{{Path: "link/a.txt", Content: []byte("a")}}
	write(t, out, files...)
	if _, err := os.Stat(filepath.Join(out, "real", "a.txt")); err != nil {
		t.Fatalf("not written through the link: %v", err)
	}
	stale, _, err := gen.Verify(out, files)
	if err != nil || len(stale) != 0 {
		t.Errorf("verify: %v, %v", stale, err)
	}
	if removed, err := gen.Clean(out); err != nil || len(removed) != 1 {
		t.Errorf("clean: %v, %v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(out, "real", "a.txt")); !os.IsNotExist(err) {
		t.Errorf("a.txt survived: %v", err)
	}
}
