package cli

import (
	"strings"
	"testing"
)

// A file named `-` is read from standard input, and positions name it
// <stdin> rather than `-`, which reads as a flag.
func TestLoadFileReadsStdin(t *testing.T) {
	cmd, _, _ := newTestCmd()
	cmd.SetIn(strings.NewReader("primitive string\n"))

	file, err := loadFile(cmd, "-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := file.Filename, "<stdin>"; got != want {
		t.Errorf("filename = %q, want %q", got, want)
	}
	if len(file.Decls) != 1 {
		t.Fatalf("got %d declarations, want 1", len(file.Decls))
	}
}

func TestLoadFileReportsStdinPositions(t *testing.T) {
	cmd, _, _ := newTestCmd()
	cmd.SetIn(strings.NewReader("entity E { id string }\n"))

	_, err := loadFile(cmd, "-")
	if err == nil {
		t.Fatal("expected a parse error")
	}
	if !strings.Contains(err.Error(), "<stdin>:1:") {
		t.Errorf("error %q does not name <stdin> with a position", err)
	}
}

// -w has nowhere to write standard input back to, and says so rather than
// creating a file called `-`.
func TestFmtWriteRejectsStdin(t *testing.T) {
	cmd := newFmtCmd()
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	captureCmd(cmd)
	cmd.SetIn(strings.NewReader("primitive string\n"))
	cmd.SetArgs([]string{"-w", "-"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "<stdin>") {
		t.Errorf("error %q does not mention <stdin>", err)
	}
}

func TestFmtFormatsStdin(t *testing.T) {
	cmd := newFmtCmd()
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	out, _ := captureCmd(cmd)
	cmd.SetIn(strings.NewReader("primitive    string"))
	cmd.SetArgs([]string{"-"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := out.String(), "primitive string\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestDisplayName(t *testing.T) {
	if got, want := displayName("-"), "<stdin>"; got != want {
		t.Errorf("displayName(%q) = %q, want %q", "-", got, want)
	}
	if got, want := displayName("a.tdl"), "a.tdl"; got != want {
		t.Errorf("displayName(%q) = %q, want %q", "a.tdl", got, want)
	}
}

// gen writes files from a model, and an import resolves next to the file
// that wrote it, so it needs one.
func TestGenRejectsStdin(t *testing.T) {
	cmd := newGenCmd()
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	captureCmd(cmd)
	cmd.SetIn(strings.NewReader("primitive string\n"))
	cmd.SetArgs([]string{"-"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "import") {
		t.Errorf("error %q does not say why", err)
	}
}

// ir does accept it: it reads a model rather than writing files from one.
func TestIrAcceptsStdin(t *testing.T) {
	cmd := newIrCmd()
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	out, _ := captureCmd(cmd)
	cmd.SetIn(strings.NewReader("package p\n\nprimitive string\n"))
	cmd.SetArgs([]string{"-"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Model p") {
		t.Errorf("output does not look like a model:\n%s", out)
	}
}
