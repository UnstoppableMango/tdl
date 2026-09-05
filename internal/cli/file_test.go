package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestCmd returns a command whose output and error streams are captured.
func newTestCmd() (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	cmd := &cobra.Command{}
	out, errOut := captureCmd(cmd)
	return cmd, out, errOut
}

// captureCmd redirects a command's streams into buffers the test can read.
func captureCmd(cmd *cobra.Command) (out, errOut *bytes.Buffer) {
	out, errOut = &bytes.Buffer{}, &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	return out, errOut
}

// A bad file must not stop the walk: a run over several files says which of
// them failed, not which one failed first.
func TestEachFileReportsEveryFailure(t *testing.T) {
	cmd, _, errOut := newTestCmd()

	var seen []string
	err := eachFile(cmd, []string{"a.tdl", "b.tdl", "c.tdl"}, func(_ int, path string) error {
		seen = append(seen, path)
		if path == "a.tdl" || path == "c.tdl" {
			return errors.New(path + ": boom")
		}
		return nil
	})

	if err == nil {
		t.Fatal("expected an error when a file failed")
	}
	if got, want := err.Error(), "2 files failed"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
	if got, want := strings.Join(seen, " "), "a.tdl b.tdl c.tdl"; got != want {
		t.Errorf("visited %q, want %q", got, want)
	}
	for _, want := range []string{"a.tdl: boom", "c.tdl: boom"} {
		if !strings.Contains(errOut.String(), want) {
			t.Errorf("stderr missing %q:\n%s", want, errOut)
		}
	}
}

func TestEachFileSucceedsSilently(t *testing.T) {
	cmd, out, errOut := newTestCmd()

	if err := eachFile(cmd, []string{"a.tdl"}, func(int, string) error { return nil }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Len() != 0 || errOut.Len() != 0 {
		t.Errorf("expected no output, got stdout %q stderr %q", out, errOut)
	}
}

func TestEachFileCountsOneFailure(t *testing.T) {
	cmd, _, _ := newTestCmd()

	err := eachFile(cmd, []string{"a.tdl"}, func(int, string) error { return errors.New("boom") })
	if err == nil {
		t.Fatal("expected an error")
	}
	if got, want := err.Error(), "1 file failed"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

// One file prints no banner, so output stays pipeable in the common case.
func TestWriteHeaderSingleFileIsSilent(t *testing.T) {
	cmd, out, _ := newTestCmd()

	paths := []string{"a.tdl"}
	writeHeader(cmd, paths, 0, "a.tdl")

	if out.Len() != 0 {
		t.Errorf("expected no header for a single file, got %q", out)
	}
}

// Two or more files are separated the way head(1) separates them, with a
// blank line before every banner but the first.
func TestWriteHeaderSeparatesFiles(t *testing.T) {
	cmd, out, _ := newTestCmd()

	paths := []string{"a.tdl", "b.tdl"}
	for i, path := range paths {
		writeHeader(cmd, paths, i, path)
	}

	want := "==> a.tdl <==\n\n==> b.tdl <==\n"
	if got := out.String(); got != want {
		t.Errorf("header output = %q, want %q", got, want)
	}
}
