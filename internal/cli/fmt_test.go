package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// Formatting must not widen a file's permissions. The mode a file is
// stored with is the author's decision, not the formatter's.
func TestWriteFormattedKeepsMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "types.tdl")
	if err := os.WriteFile(path, []byte("primitive string\n"), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	if err := writeFormatted(path, "primitive int\n"); err != nil {
		t.Fatalf("writeFormatted: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Errorf("mode = %o, want %o", got, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if got, want := string(data), "primitive int\n"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
}

func TestWriteFormattedCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.tdl")

	if err := writeFormatted(path, "primitive string\n"); err != nil {
		t.Fatalf("writeFormatted: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o644); got != want {
		t.Errorf("mode = %o, want %o", got, want)
	}
}

// --check lists the files that need formatting and exits non-zero, and
// says nothing about the ones that do not.
func TestFmtCheckListsOnlyStaleFiles(t *testing.T) {
	dir := t.TempDir()
	canonical := filepath.Join(dir, "canonical.tdl")
	messy := filepath.Join(dir, "messy.tdl")
	if err := os.WriteFile(canonical, []byte("primitive string\n"), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
	if err := os.WriteFile(messy, []byte("primitive    string"), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	cmd := newFmtCmd()
	// The root command is what silences these; a command run on its own
	// would otherwise print its usage over the output under test.
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	out, errOut := captureCmd(cmd)
	cmd.SetArgs([]string{"--check", canonical, messy})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected a non-zero exit when a file is not canonical")
	}
	if got, want := out.String(), messy+"\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("expected nothing on stderr, got %q", errOut)
	}

	// The check must write nothing: the messy file is still messy.
	data, err := os.ReadFile(messy)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if got, want := string(data), "primitive    string"; got != want {
		t.Errorf("--check rewrote the file: %q, want %q", got, want)
	}
}

func TestFmtCheckPassesOnCanonicalFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "canonical.tdl")
	if err := os.WriteFile(path, []byte("primitive string\n"), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	cmd := newFmtCmd()
	out, _ := captureCmd(cmd)
	cmd.SetArgs([]string{"--check", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output, got %q", out)
	}
}
